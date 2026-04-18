package typed

import "a-lang/parser"

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
