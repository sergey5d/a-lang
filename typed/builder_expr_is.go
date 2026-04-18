package typed

import "a-lang/parser"

// buildIsExpr converts parser is-expressions into typed is-expressions.
func (b *exprBuilder) buildIsExpr(expr parser.Expr, isExpr *parser.IsExpr) (Expr, error) {
	left, err := b.Build(isExpr.Left)
	if err != nil {
		return nil, err
	}
	return &IsExpr{
		baseExpr: b.base(expr),
		Left:     left,
		Target:   b.types.BuildType(isExpr.Target),
	}, nil
}
