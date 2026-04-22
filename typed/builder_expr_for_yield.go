package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

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
		parts := make([]BindingDecl, len(binding.Bindings))
		for j, part := range binding.Bindings {
			symbol := b.ctx.newSymbol(SymbolBinding, part.Name, "", part.Span)
			typ := elemType
			if len(binding.Bindings) > 1 {
				typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
			}
			if part.Type != nil {
				typ = b.types.BuildType(part.Type)
			}
			parts[j] = BindingDecl{Name: part.Name, Type: typ, Mode: BindingImmutable, Symbol: symbol, Span: part.Span}
			if part.Name != "_" {
				b.ctx.defineSymbol(symbol)
			}
		}
		bindings[i] = ForBinding{Bindings: parts, Iterable: iterable, Span: binding.Span}
	}
	yieldBody, err := b.blocks.Build(forYield.YieldBody)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}
	return &ForYieldExpr{baseExpr: b.base(expr), Bindings: bindings, YieldBody: yieldBody}, nil
}
