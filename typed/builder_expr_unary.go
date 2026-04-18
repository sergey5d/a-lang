package typed

import "a-lang/parser"

// buildUnaryExpr converts parser unary expressions into typed unary expressions.
func (b *exprBuilder) buildUnaryExpr(expr parser.Expr, unary *parser.UnaryExpr) (Expr, error) {
	right, err := b.Build(unary.Right)
	if err != nil {
		return nil, err
	}
	return &UnaryExpr{baseExpr: b.base(expr), Operator: unary.Operator, Right: right}, nil
}
