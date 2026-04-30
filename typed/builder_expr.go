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
	case *parser.UnitLiteral:
		return &UnitLiteral{baseExpr: b.base(expr)}, nil
	case *parser.ListLiteral:
		return b.buildListLiteral(expr, e)
	case *parser.TupleLiteral:
		return b.buildTupleLiteral(expr, e)
	case *parser.AnonymousRecordExpr:
		return b.buildAnonymousRecordExpr(expr, e)
	case *parser.GroupExpr:
		inner, err := b.Build(e.Inner)
		if err != nil {
			return nil, err
		}
		return &GroupExpr{baseExpr: b.base(expr), Inner: inner}, nil
	case *parser.BlockExpr:
		body, err := b.blocks.Build(e.Body)
		if err != nil {
			return nil, err
		}
		return &BlockExpr{baseExpr: b.base(expr), Body: body}, nil
	case *parser.UnaryExpr:
		return b.buildUnaryExpr(expr, e)
	case *parser.BinaryExpr:
		return b.buildBinaryExpr(expr, e)
	case *parser.IsExpr:
		return b.buildIsExpr(expr, e)
	case *parser.MemberExpr:
		return b.buildMemberExpr(expr, e)
	case *parser.IndexExpr:
		return b.buildIndexExpr(expr, e)
	case *parser.RecordUpdateExpr:
		return b.buildRecordUpdateExpr(expr, e)
	case *parser.AnonymousInterfaceExpr:
		return b.buildAnonymousInterfaceExpr(expr, e)
	case *parser.IfExpr:
		return b.buildIfExpr(expr, e)
	case *parser.MatchExpr:
		return b.buildMatchExpr(expr, e)
	case *parser.ForYieldExpr:
		return b.buildForYieldExpr(expr, e)
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
