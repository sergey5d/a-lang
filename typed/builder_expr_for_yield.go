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
		parts := make([]BindingDecl, len(binding.Bindings))
		var iterable Expr
		var values []Expr
		var valueTypes []*typecheck.Type
		if binding.Iterable != nil {
			var err error
			iterable, err = b.Build(binding.Iterable)
			if err != nil {
				b.ctx.popScope()
				return nil, err
			}
			boundType := b.ctx.exprTypes[binding.Iterable]
			elemType := b.types.iterableElementType(boundType)
			valueTypes = []*typecheck.Type{elemType}
			if len(binding.Bindings) > 1 {
				valueTypes = make([]*typecheck.Type, len(binding.Bindings))
			}
		} else {
			values = make([]Expr, len(binding.Values))
			valueTypes = make([]*typecheck.Type, len(binding.Bindings))
			for j, value := range binding.Values {
				if value == nil {
					continue
				}
				built, err := b.Build(value)
				if err != nil {
					b.ctx.popScope()
					return nil, err
				}
				values[j] = built
			}
		}
		for j, part := range binding.Bindings {
			symbol := b.ctx.newSymbol(SymbolBinding, part.Name, "", part.Span)
			typ := &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
			if j < len(valueTypes) && valueTypes[j] != nil {
				typ = valueTypes[j]
			}
			if binding.Iterable == nil && len(binding.Values) == len(binding.Bindings) && j < len(values) && values[j] != nil {
				typ = values[j].GetType()
			}
			if part.Type != nil {
				typ = b.types.BuildType(part.Type)
			}
			mode := BindingImmutable
			if part.Mutable {
				mode = BindingMutable
			}
			parts[j] = BindingDecl{Name: part.Name, Type: typ, Mode: mode, Symbol: symbol, Span: part.Span}
			if part.Name != "_" {
				b.ctx.defineSymbol(symbol)
			}
		}
		bindings[i] = ForBinding{Bindings: parts, Iterable: iterable, Values: values, Span: binding.Span}
	}
	yieldBody, err := b.blocks.Build(forYield.YieldBody)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}
	return &ForYieldExpr{baseExpr: b.base(expr), Bindings: bindings, YieldBody: yieldBody}, nil
}
