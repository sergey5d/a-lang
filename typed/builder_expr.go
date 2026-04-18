package typed

import (
	"fmt"

	"a-lang/parser"
)

// exprBuilder builds typed expressions and depends only on type conversion and blocks.
type exprBuilder struct {
	ctx     *buildContext
	types   *typeRefBuilder
	blocks  Builder[*parser.BlockStmt, *BlockStmt]
	calls   Builder[*parser.CallExpr, Expr]
	lambdas Builder[*parser.LambdaExpr, Expr]
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
	case *parser.IfExpr:
		condition, err := b.Build(e.Condition)
		if err != nil {
			return nil, err
		}
		thenBlock, err := b.blocks.Build(e.Then)
		if err != nil {
			return nil, err
		}
		elseBlock, err := b.blocks.Build(e.Else)
		if err != nil {
			return nil, err
		}
		return &IfExpr{baseExpr: b.base(expr), Condition: condition, Then: thenBlock, Else: elseBlock}, nil
	case *parser.ForYieldExpr:
		bindings := make([]ForBinding, len(e.Bindings))
		b.ctx.pushScope()
		for i, binding := range e.Bindings {
			iterable, err := b.Build(binding.Iterable)
			if err != nil {
				b.ctx.popScope()
				return nil, err
			}
			boundType := b.ctx.exprTypes[binding.Iterable]
			elemType := b.types.iterableElementType(boundType)
			boundSymbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			b.ctx.defineSymbol(boundSymbol)
			bindings[i] = ForBinding{
				Name:     binding.Name,
				Type:     elemType,
				Iterable: iterable,
				Symbol:   boundSymbol,
				Span:     binding.Span,
			}
		}
		yieldBody, err := b.blocks.Build(e.YieldBody)
		b.ctx.popScope()
		if err != nil {
			return nil, err
		}
		return &ForYieldExpr{baseExpr: b.base(expr), Bindings: bindings, YieldBody: yieldBody}, nil
	case *parser.CallExpr:
		return b.calls.Build(e)
	case *parser.LambdaExpr:
		return b.lambdas.Build(e)
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

// base constructs the common typed expression metadata for a parser expression.
func (b *exprBuilder) base(expr parser.Expr) baseExpr {
	return baseExpr{Type: b.ctx.exprTypes[expr], Span: exprSpan(expr)}
}
