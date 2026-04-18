package typed

import "a-lang/parser"

// buildIndexExpr converts parser indexing into typed index expressions.
func (b *exprBuilder) buildIndexExpr(expr parser.Expr, indexExpr *parser.IndexExpr) (Expr, error) {
	receiver, err := b.Build(indexExpr.Receiver)
	if err != nil {
		return nil, err
	}
	index, err := b.Build(indexExpr.Index)
	if err != nil {
		return nil, err
	}
	return &IndexExpr{baseExpr: b.base(expr), Receiver: receiver, Index: index}, nil
}
