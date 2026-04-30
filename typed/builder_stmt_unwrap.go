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

// guardStmtBuilder builds typed guarded unwrap statements.
type guardStmtBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser unwrap statement into a typed unwrap statement.
func (b *unwrapStmtBuilder) Build(stmt *parser.UnwrapStmt) (Stmt, error) {
	value, err := b.exprs.Build(stmt.Value)
	if err != nil {
		return nil, err
	}
	bindings := buildBindingDecls(b.ctx, b.types, stmt.Bindings)
	return &UnwrapStmt{Bindings: bindings, Value: value, Span: stmt.Span}, nil
}

// Build converts a parser guard statement into a typed guarded unwrap statement.
func (b *guardStmtBuilder) Build(stmt *parser.GuardStmt) (Stmt, error) {
	value, err := b.exprs.Build(stmt.Value)
	if err != nil {
		return nil, err
	}
	fallback, err := b.blocks.Build(stmt.Fallback)
	if err != nil {
		return nil, err
	}
	bindings := buildBindingDecls(b.ctx, b.types, stmt.Bindings)
	return &GuardStmt{Bindings: bindings, Value: value, Fallback: fallback, Span: stmt.Span}, nil
}

func buildBindingDecls(ctx *buildContext, types *typeRefBuilder, bindings []parser.Binding) []BindingDecl {
	out := make([]BindingDecl, len(bindings))
	for i, binding := range bindings {
		symbol := ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
		var typ *typecheck.Type
		if binding.Type != nil {
			typ = types.BuildType(binding.Type)
		}
		if typ == nil {
			typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
		}
		out[i] = BindingDecl{
			Name:   binding.Name,
			Type:   typ,
			Mode:   BindingImmutable,
			Symbol: symbol,
			Span:   binding.Span,
		}
		if binding.Name != "_" {
			ctx.defineSymbol(symbol)
		}
	}
	return out
}
