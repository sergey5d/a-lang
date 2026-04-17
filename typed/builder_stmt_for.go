package typed

import "a-lang/parser"

// forStmtBuilder builds typed for statements.
type forStmtBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
}

// Build converts a parser for statement into a typed for statement.
func (b *forStmtBuilder) Build(stmt *parser.ForStmt) (Stmt, error) {
	bindings := make([]ForBinding, len(stmt.Bindings))
	for i, binding := range stmt.Bindings {
		iterable, err := b.exprs.Build(binding.Iterable)
		if err != nil {
			return nil, err
		}
		symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
		bindings[i] = ForBinding{
			Name:     binding.Name,
			Type:     elementType(iterable.GetType()),
			Iterable: iterable,
			Symbol:   symbol,
			Span:     binding.Span,
		}
	}

	b.ctx.pushScope()
	for _, binding := range bindings {
		b.ctx.defineSymbol(binding.Symbol)
	}
	body, err := b.blocks.Build(stmt.Body)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}

	b.ctx.pushScope()
	for _, binding := range bindings {
		b.ctx.defineSymbol(binding.Symbol)
	}
	yieldBody, err := b.blocks.Build(stmt.YieldBody)
	b.ctx.popScope()
	if err != nil {
		return nil, err
	}

	return &ForStmt{Bindings: bindings, Body: body, YieldBody: yieldBody, Span: stmt.Span}, nil
}
