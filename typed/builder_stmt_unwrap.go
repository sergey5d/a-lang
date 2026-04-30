package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// unwrapStmtBuilder builds typed short-circuit unwrap statements.
type unwrapStmtBuilder struct {
	ctx   *buildContext
	exprs Builder[parser.Expr, Expr]
	types *typeRefBuilder
}

// Build converts a parser unwrap statement into a typed unwrap statement.
func (b *unwrapStmtBuilder) Build(stmt *parser.UnwrapStmt) (Stmt, error) {
	value, err := b.exprs.Build(stmt.Value)
	if err != nil {
		return nil, err
	}
	var guard Expr
	if stmt.Guard != nil {
		guard, err = b.exprs.Build(stmt.Guard)
		if err != nil {
			return nil, err
		}
	}
	bindings := make([]BindingDecl, len(stmt.Bindings))
	for i, binding := range stmt.Bindings {
		symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
		var typ *typecheck.Type
		if binding.Type != nil {
			typ = b.types.BuildType(binding.Type)
		}
		if typ == nil {
			typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
		}
		bindings[i] = BindingDecl{
			Name:   binding.Name,
			Type:   typ,
			Mode:   BindingImmutable,
			Symbol: symbol,
			Span:   binding.Span,
		}
		if binding.Name != "_" {
			b.ctx.defineSymbol(symbol)
		}
	}
	return &UnwrapStmt{Bindings: bindings, Value: value, Guard: guard, Span: stmt.Span}, nil
}
