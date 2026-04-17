package typed

import (
	"fmt"

	"a-lang/parser"
)

// exprBuilder builds typed expressions and depends only on type conversion and blocks.
type exprBuilder struct {
	ctx    *buildContext
	types  *typeRefBuilder
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	calls  Builder[*parser.CallExpr, Expr]
}

// Build converts a parser expression into a typed expression node.
func (b *exprBuilder) Build(expr parser.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		ident := &IdentifierExpr{baseExpr: b.base(expr), Name: e.Name}
		if symbol, ok := b.ctx.lookupSymbol(e.Name); ok {
			ident.Symbol = symbol
		}
		return ident, nil
	case *parser.PlaceholderExpr:
		return &PlaceholderExpr{baseExpr: b.base(expr)}, nil
	case *parser.IntegerLiteral:
		return &IntegerLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.FloatLiteral:
		return &FloatLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.RuneLiteral:
		return &RuneLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.BoolLiteral:
		return &BoolLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.StringLiteral:
		return &StringLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.ListLiteral:
		elements := make([]Expr, len(e.Elements))
		for i, element := range e.Elements {
			built, err := b.Build(element)
			if err != nil {
				return nil, err
			}
			elements[i] = built
		}
		return &ListLiteral{baseExpr: b.base(expr), Elements: elements}, nil
	case *parser.GroupExpr:
		inner, err := b.Build(e.Inner)
		if err != nil {
			return nil, err
		}
		return &GroupExpr{baseExpr: b.base(expr), Inner: inner}, nil
	case *parser.UnaryExpr:
		right, err := b.Build(e.Right)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{baseExpr: b.base(expr), Operator: e.Operator, Right: right}, nil
	case *parser.BinaryExpr:
		left, err := b.Build(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := b.Build(e.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{baseExpr: b.base(expr), Left: left, Operator: e.Operator, Right: right}, nil
	case *parser.MemberExpr:
		receiver, err := b.Build(e.Receiver)
		if err != nil {
			return nil, err
		}
		field := &FieldExpr{baseExpr: b.base(expr), Receiver: receiver, Name: e.Name}
		field.Field = b.types.resolveFieldSymbol(receiver.GetType(), e.Name)
		return field, nil
	case *parser.IndexExpr:
		receiver, err := b.Build(e.Receiver)
		if err != nil {
			return nil, err
		}
		index, err := b.Build(e.Index)
		if err != nil {
			return nil, err
		}
		return &IndexExpr{baseExpr: b.base(expr), Receiver: receiver, Index: index}, nil
	case *parser.CallExpr:
		return b.calls.Build(e)
	case *parser.LambdaExpr:
		b.ctx.pushScope()
		defer b.ctx.popScope()

		params := make([]LambdaParameter, len(e.Parameters))
		for i, param := range e.Parameters {
			symbol := b.ctx.newSymbol(SymbolParameter, param.Name, "", param.Span)
			params[i] = LambdaParameter{
				Name:   param.Name,
				Type:   b.types.BuildType(param.Type),
				Symbol: symbol,
				Span:   param.Span,
			}
			b.ctx.defineSymbol(symbol)
		}

		var body Expr
		var err error
		if e.Body != nil {
			body, err = b.Build(e.Body)
			if err != nil {
				return nil, err
			}
		}
		blockBody, err := b.blocks.Build(e.BlockBody)
		if err != nil {
			return nil, err
		}

		return &LambdaExpr{baseExpr: b.base(expr), Parameters: params, Body: body, BlockBody: blockBody}, nil
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

// base constructs the common typed expression metadata for a parser expression.
func (b *exprBuilder) base(expr parser.Expr) baseExpr {
	return baseExpr{Type: b.ctx.exprTypes[expr], Span: exprSpan(expr)}
}
