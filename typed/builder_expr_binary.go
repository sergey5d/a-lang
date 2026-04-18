package typed

import "a-lang/parser"

// buildBinaryExpr converts parser binary expressions into typed binary expressions.
func (b *exprBuilder) buildBinaryExpr(expr parser.Expr, binary *parser.BinaryExpr) (Expr, error) {
	left, err := b.Build(binary.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.Build(binary.Right)
	if err != nil {
		return nil, err
	}
	return &BinaryExpr{baseExpr: b.base(expr), Left: left, Operator: binary.Operator, Right: right}, nil
}
