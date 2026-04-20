package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

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
		Record:         class.Record,
		TypeParameters: b.params.buildTypeParameters(class.TypeParameters),
		Interfaces:     implements,
		Fields:         fields,
		Methods:        methods,
		Symbol:         b.ctx.classSymbols[class.Name],
		Span:           class.Span,
	}, nil
}
