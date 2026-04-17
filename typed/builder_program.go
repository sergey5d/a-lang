package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

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

// functionBuilder builds typed top-level function declarations.
type functionBuilder struct {
	ctx    *buildContext
	params *parameterBuilder
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser function declaration into a typed function declaration.
func (b *functionBuilder) Build(fn *parser.FunctionDecl) (*FunctionDecl, error) {
	b.ctx.pushScope()
	defer b.ctx.popScope()

	params := b.params.buildParameters(fn.Parameters)
	for _, param := range params {
		b.ctx.defineSymbol(param.Symbol)
	}
	body, err := b.blocks.Build(fn.Body)
	if err != nil {
		return nil, err
	}
	return &FunctionDecl{
		Name:       fn.Name,
		Parameters: params,
		ReturnType: b.types.BuildType(fn.ReturnType),
		Body:       body,
		Symbol:     b.ctx.functionSymbols[fn.Name],
		Span:       fn.Span,
	}, nil
}

// interfaceBuilder builds typed interface declarations.
type interfaceBuilder struct {
	ctx    *buildContext
	params *parameterBuilder
	types  *typeRefBuilder
}

// Build converts a parser interface declaration into a typed interface declaration.
func (b *interfaceBuilder) Build(iface *parser.InterfaceDecl) (*InterfaceDecl, error) {
	methods := make([]InterfaceMethod, len(iface.Methods))
	for i, method := range iface.Methods {
		methods[i] = InterfaceMethod{
			Name:       method.Name,
			Parameters: b.params.buildParameters(method.Parameters),
			ReturnType: b.types.BuildType(method.ReturnType),
			Span:       method.Span,
		}
	}
	return &InterfaceDecl{
		Name:           iface.Name,
		TypeParameters: b.params.buildTypeParameters(iface.TypeParameters),
		Methods:        methods,
		Symbol:         b.ctx.interfaceSymbols[iface.Name],
		Span:           iface.Span,
	}, nil
}

// classBuilder builds typed class declarations and their members.
type classBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	params *parameterBuilder
	types  *typeRefBuilder
}

// Build converts a parser class declaration into a typed class declaration.
func (b *classBuilder) Build(class *parser.ClassDecl) (*ClassDecl, error) {
	b.ctx.pushScope()
	defer b.ctx.popScope()

	thisSymbol := b.ctx.newSymbol(SymbolThis, "this", class.Name, class.Span)
	b.ctx.thisStack = append(b.ctx.thisStack, thisSymbol)
	defer func() { b.ctx.thisStack = b.ctx.thisStack[:len(b.ctx.thisStack)-1] }()

	b.ctx.defineSymbol(thisSymbol)
	for _, field := range class.Fields {
		b.ctx.defineSymbol(b.ctx.fieldSymbols[class.Name][field.Name])
	}

	fields := make([]FieldDecl, len(class.Fields))
	for i, field := range class.Fields {
		var init Expr
		var err error
		if field.Initializer != nil {
			init, err = b.exprs.Build(field.Initializer)
			if err != nil {
				return nil, err
			}
		}
		fields[i] = FieldDecl{
			Name:     field.Name,
			Type:     b.types.BuildType(field.Type),
			Mode:     modeFromMutable(field.Mutable),
			InitMode: initMode(field.Deferred, init),
			Init:     init,
			Private:  field.Private,
			Symbol:   b.ctx.fieldSymbols[class.Name][field.Name],
			Span:     field.Span,
		}
	}

	methods := make([]*MethodDecl, len(class.Methods))
	for i, method := range class.Methods {
		b.ctx.pushScope()
		b.ctx.defineSymbol(thisSymbol)
		params := b.params.buildParameters(method.Parameters)
		for _, param := range params {
			b.ctx.defineSymbol(param.Symbol)
		}
		body, err := b.blocks.Build(method.Body)
		b.ctx.popScope()
		if err != nil {
			return nil, err
		}
		methods[i] = &MethodDecl{
			Name:        method.Name,
			Parameters:  params,
			ReturnType:  b.types.BuildType(method.ReturnType),
			Body:        body,
			Private:     method.Private,
			Constructor: method.Constructor,
			Symbol:      b.types.lookupMethodSymbol(class.Name, method),
			Span:        method.Span,
		}
	}

	implements := make([]*typecheck.Type, len(class.Implements))
	for i, impl := range class.Implements {
		implements[i] = b.types.BuildType(impl)
	}

	return &ClassDecl{
		Name:           class.Name,
		TypeParameters: b.params.buildTypeParameters(class.TypeParameters),
		Interfaces:     implements,
		Fields:         fields,
		Methods:        methods,
		Symbol:         b.ctx.classSymbols[class.Name],
		Span:           class.Span,
	}, nil
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
