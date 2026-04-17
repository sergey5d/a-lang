package typed

import "a-lang/parser"

// symbolCollector preallocates stable top-level and member symbols.
type symbolCollector struct {
	ctx *buildContext
}

// collect populates the shared symbol tables before typed construction starts.
func (s *symbolCollector) collect(program *parser.Program) {
	for _, fn := range program.Functions {
		s.ctx.functionSymbols[fn.Name] = s.ctx.newSymbol(SymbolFunction, fn.Name, "", fn.Span)
	}
	for _, iface := range program.Interfaces {
		s.ctx.interfaceSymbols[iface.Name] = s.ctx.newSymbol(SymbolInterface, iface.Name, "", iface.Span)
	}
	for _, class := range program.Classes {
		s.ctx.classSymbols[class.Name] = s.ctx.newSymbol(SymbolClass, class.Name, "", class.Span)
		fields := map[string]SymbolRef{}
		methods := map[string][]methodSymbol{}
		for _, field := range class.Fields {
			fields[field.Name] = s.ctx.newSymbol(SymbolField, field.Name, class.Name, field.Span)
		}
		for _, method := range class.Methods {
			sym := s.ctx.newSymbol(SymbolMethod, method.Name, class.Name, method.Span)
			methods[method.Name] = append(methods[method.Name], methodSymbol{decl: method, symbol: sym})
		}
		s.ctx.fieldSymbols[class.Name] = fields
		s.ctx.methodSymbols[class.Name] = methods
	}
}
