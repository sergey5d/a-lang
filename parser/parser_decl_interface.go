package parser

func (p *Parser) parseInterface() (*InterfaceDecl, error) {
	start, err := p.consume(TokenInterface, "expected 'interface'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected interface name")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParameters()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenLBrace, "expected '{' after interface name"); err != nil {
		return nil, err
	}
	decl := &InterfaceDecl{Name: name.Lexeme, TypeParameters: typeParams}
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		method, err := p.parseInterfaceMethod()
		if err != nil {
			return nil, err
		}
		decl.Methods = append(decl.Methods, method)
	}
	end, err := p.consume(TokenRBrace, "expected '}' after interface body")
	if err != nil {
		return nil, err
	}
	decl.Span = mergeSpans(tokenSpan(start), tokenSpan(end))
	return decl, nil
}

func (p *Parser) parseInterfaceMethod() (InterfaceMethod, error) {
	start, err := p.consume(TokenDef, "expected 'def' in interface")
	if err != nil {
		return InterfaceMethod{}, err
	}
	name, err := p.consume(TokenIdentifier, "expected method name")
	if err != nil {
		return InterfaceMethod{}, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return InterfaceMethod{}, err
	}
	returnType, err := p.parseTypeRef()
	if err != nil {
		return InterfaceMethod{}, err
	}
	return InterfaceMethod{
		Name:       name.Lexeme,
		Parameters: params,
		ReturnType: returnType,
		Span:       mergeSpans(tokenSpan(start), typeSpan(returnType)),
	}, nil
}
