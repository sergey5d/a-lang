package parser

func (p *Parser) parseProgram() (*Program, error) {
	program := &Program{}
	for !p.isAtEnd() {
		switch p.peek().Type {
		case TokenDef:
			fn, err := p.parseFunction()
			if err != nil {
				return nil, err
			}
			program.Functions = append(program.Functions, fn)
		case TokenInterface:
			decl, err := p.parseInterface()
			if err != nil {
				return nil, err
			}
			program.Interfaces = append(program.Interfaces, decl)
		case TokenClass:
			decl, err := p.parseClass()
			if err != nil {
				return nil, err
			}
			program.Classes = append(program.Classes, decl)
		default:
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			program.Statements = append(program.Statements, stmt)
		}
	}
	if span, ok := p.programSpan(program); ok {
		program.Span = span
	}
	return program, nil
}

func (p *Parser) programSpan(program *Program) (Span, bool) {
	var spans []Span
	for _, fn := range program.Functions {
		spans = append(spans, fn.Span)
	}
	for _, decl := range program.Interfaces {
		spans = append(spans, decl.Span)
	}
	for _, decl := range program.Classes {
		spans = append(spans, decl.Span)
	}
	for _, stmt := range program.Statements {
		spans = append(spans, stmtSpan(stmt))
	}
	if len(spans) == 0 {
		return Span{}, false
	}
	return mergeSpans(spans[0], spans[len(spans)-1]), true
}
