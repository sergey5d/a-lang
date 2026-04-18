package typed

import "a-lang/parser"

// buildIfExpr converts parser if expressions into typed if expressions.
func (b *exprBuilder) buildIfExpr(expr parser.Expr, ifExpr *parser.IfExpr) (Expr, error) {
	condition, err := b.Build(ifExpr.Condition)
	if err != nil {
		return nil, err
	}
	thenBlock, err := b.blocks.Build(ifExpr.Then)
	if err != nil {
		return nil, err
	}
	elseBlock, err := b.blocks.Build(ifExpr.Else)
	if err != nil {
		return nil, err
	}
	return &IfExpr{baseExpr: b.base(expr), Condition: condition, Then: thenBlock, Else: elseBlock}, nil
}
