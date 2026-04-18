package typed

import "a-lang/parser"

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
