package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// bindingStmtBuilder builds typed binding statements.
type bindingStmtBuilder struct {
	ctx   *buildContext
	exprs Builder[parser.Expr, Expr]
	types *typeRefBuilder
}

// Build converts a parser binding statement into a typed binding statement.
func (b *bindingStmtBuilder) Build(stmt *parser.ValStmt) (Stmt, error) {
	bindings := make([]BindingDecl, len(stmt.Bindings))
	for i, binding := range stmt.Bindings {
		var init Expr
		var err error
		if i < len(stmt.Values) && stmt.Values[i] != nil {
			init, err = b.exprs.Build(stmt.Values[i])
			if err != nil {
				return nil, err
			}
		}
		symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
		var typ *typecheck.Type
		if binding.Type != nil {
			typ = b.types.BuildType(binding.Type)
		} else if init != nil {
			typ = init.GetType()
		}
		if typ == nil {
			typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
		}
		bindings[i] = BindingDecl{
			Name:     binding.Name,
			Type:     typ,
			Mode:     modeFromMutable(binding.Mutable),
			InitMode: initMode(binding.Deferred, init),
			Init:     init,
			Symbol:   symbol,
			Span:     binding.Span,
		}
		if binding.Name != "_" {
			b.ctx.defineSymbol(symbol)
		}
	}
	return &BindingStmt{Bindings: bindings, Span: stmt.Span}, nil
}
