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
	for i, binding := range stmt.Bindings {
		iterable, err := b.exprs.Build(binding.Iterable)
		if err != nil {
			return nil, err
		}
		elemType := elementType(iterable.GetType())
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
		}
		bindings[i] = ForBinding{Bindings: parts, Iterable: iterable, Span: binding.Span}
	}

	b.ctx.pushScope()
	for _, binding := range bindings {
		for _, part := range binding.Bindings {
			if part.Name != "_" {
				b.ctx.defineSymbol(part.Symbol)
			}
		}
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
