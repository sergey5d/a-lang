package typed

import "a-lang/parser"

// buildTupleLiteral converts parser tuple literals into typed tuple literals.
func (b *exprBuilder) buildTupleLiteral(expr parser.Expr, tuple *parser.TupleLiteral) (Expr, error) {
	elements := make([]Expr, len(tuple.Elements))
	for i, element := range tuple.Elements {
		built, err := b.Build(element)
		if err != nil {
			return nil, err
		}
		elements[i] = built
	}
	return &TupleLiteral{baseExpr: b.base(expr), Elements: elements}, nil
}
