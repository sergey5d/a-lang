package typed

import "a-lang/parser"

// buildForYieldExpr converts parser yield loops into typed yield expressions.
func (b *exprBuilder) buildForYieldExpr(expr parser.Expr, forYield *parser.ForYieldExpr) (Expr, error) {
	bindings := make([]ForBinding, len(forYield.Bindings))
	b.ctx.pushScope()
	for i, binding := range forYield.Bindings {
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
	yieldBody, err := b.blocks.Build(forYield.YieldBody)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}
	return &ForYieldExpr{baseExpr: b.base(expr), Bindings: bindings, YieldBody: yieldBody}, nil
}
