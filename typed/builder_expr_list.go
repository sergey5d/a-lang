package typed

import "a-lang/parser"

// buildListLiteral converts parser list literals into typed list literals.
func (b *exprBuilder) buildListLiteral(expr parser.Expr, list *parser.ListLiteral) (Expr, error) {
	elements := make([]Expr, len(list.Elements))
	for i, element := range list.Elements {
		built, err := b.Build(element)
		if err != nil {
			return nil, err
		}
		elements[i] = built
	}
	return &ListLiteral{baseExpr: b.base(expr), Elements: elements}, nil
}
