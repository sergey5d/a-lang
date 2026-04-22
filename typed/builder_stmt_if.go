package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// ifStmtBuilder builds typed if statements.
type ifStmtBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser if statement into a typed if statement.
func (b *ifStmtBuilder) Build(stmt *parser.IfStmt) (Stmt, error) {
	var cond Expr
	var bindingValue Expr
	var bindings []BindingDecl
	var err error
	if stmt.BindingValue != nil {
		bindingValue, err = b.exprs.Build(stmt.BindingValue)
		if err != nil {
			return nil, err
		}
		elemType := &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
		if optionType := b.ctx.exprTypes[stmt.BindingValue]; optionType != nil && optionType.Name == "Option" && len(optionType.Args) == 1 {
			elemType = optionType.Args[0]
		}
		bindings = make([]BindingDecl, len(stmt.Bindings))
		for i, binding := range stmt.Bindings {
			symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			typ := elemType
			if len(stmt.Bindings) > 1 {
				typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
			}
			if binding.Type != nil {
				typ = b.types.BuildType(binding.Type)
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
	} else {
		cond, err = b.exprs.Build(stmt.Condition)
		if err != nil {
			return nil, err
		}
	}
	thenBlock, err := b.blocks.Build(stmt.Then)
	if err != nil {
		return nil, err
	}
	var elseIf *IfStmt
	if stmt.ElseIf != nil {
		built, err := b.Build(stmt.ElseIf)
		if err != nil {
			return nil, err
		}
		elseIf = built.(*IfStmt)
	}
	elseBlock, err := b.blocks.Build(stmt.Else)
	if err != nil {
		return nil, err
	}
	return &IfStmt{Condition: cond, Bindings: bindings, BindingValue: bindingValue, Then: thenBlock, ElseIf: elseIf, Else: elseBlock, Span: stmt.Span}, nil
}
