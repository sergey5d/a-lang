package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// forStmtBuilder builds typed for statements.
type forStmtBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser for statement into a typed for statement.
func (b *forStmtBuilder) Build(stmt *parser.ForStmt) (Stmt, error) {
	bindings := make([]ForBinding, len(stmt.Bindings))
	b.ctx.pushScope()
	for i, binding := range stmt.Bindings {
		parts := make([]BindingDecl, len(binding.Bindings))
		var iterable Expr
		var values []Expr
		var valueTypes []*typecheck.Type
		if binding.Iterable != nil {
			var err error
			iterable, err = b.exprs.Build(binding.Iterable)
			if err != nil {
				b.ctx.popScope()
				return nil, err
			}
			elemType := elementType(iterable.GetType())
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
				built, err := b.exprs.Build(value)
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
			if binding.Iterable != nil && len(binding.Bindings) == 1 && iterable != nil {
				typ = elementType(iterable.GetType())
			} else if binding.Iterable == nil && len(binding.Values) == len(binding.Bindings) && j < len(values) && values[j] != nil {
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
	body, err := b.blocks.Build(stmt.Body)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}

	b.ctx.pushScope()
	for _, binding := range bindings {
		for _, part := range binding.Bindings {
			if part.Name != "_" {
				b.ctx.defineSymbol(part.Symbol)
			}
		}
	}
	yieldBody, err := b.blocks.Build(stmt.YieldBody)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}

	return &ForStmt{Bindings: bindings, Body: body, YieldBody: yieldBody, Span: stmt.Span}, nil
}
