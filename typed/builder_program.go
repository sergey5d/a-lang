package typed

import "a-lang/parser"

// programBuilder builds the typed root by composing statement and declaration builders.
type programBuilder struct {
	ctx        *buildContext
	stmts      Builder[parser.Statement, Stmt]
	functions  Builder[*parser.FunctionDecl, *FunctionDecl]
	interfaces Builder[*parser.InterfaceDecl, *InterfaceDecl]
	classes    Builder[*parser.ClassDecl, *ClassDecl]
}

// Build converts a parser program into a typed program.
func (b *programBuilder) Build(program *parser.Program) (*Program, error) {
	out := &Program{Span: program.Span}
	b.ctx.pushScope()
	defer b.ctx.popScope()

	for _, stmt := range program.Statements {
		built, err := b.stmts.Build(stmt)
		if err != nil {
			return nil, err
		}
		out.Globals = append(out.Globals, built)
	}

	for _, fn := range program.Functions {
		built, err := b.functions.Build(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, built)
	}

	for _, iface := range program.Interfaces {
		built, err := b.interfaces.Build(iface)
		if err != nil {
			return nil, err
		}
		out.Interfaces = append(out.Interfaces, built)
	}

	for _, class := range program.Classes {
		built, err := b.classes.Build(class)
		if err != nil {
			return nil, err
		}
		out.Classes = append(out.Classes, built)
	}

	return out, nil
}

// blockBuilder builds typed blocks by delegating each nested statement.
type blockBuilder struct {
	ctx   *buildContext
	stmts Builder[parser.Statement, Stmt]
}

// Build converts a parser block into a typed block.
func (b *blockBuilder) Build(block *parser.BlockStmt) (*BlockStmt, error) {
	if block == nil {
		return nil, nil
	}
	b.ctx.pushScope()
	defer b.ctx.popScope()

	statements := make([]Stmt, len(block.Statements))
	for i, stmt := range block.Statements {
		built, err := b.stmts.Build(stmt)
		if err != nil {
			return nil, err
		}
		statements[i] = built
	}
	return &BlockStmt{Statements: statements, Span: block.Span}, nil
}
