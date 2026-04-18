package parser

func (p *Parser) parseMethod(private bool) (*MethodDecl, error) {
	start, err := p.consume(TokenDef, "expected 'def'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected method name")
	if err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	constructor := name.Lexeme == "init"
	var returnType *TypeRef
	if !constructor && !p.check(TokenLBrace) && !p.check(TokenAssign) {
		typ, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		returnType = typ
	}
	body, err := p.parseCallableBody()
	if err != nil {
		return nil, err
	}
	if !constructor && returnType == nil {
		returnType = implicitUnitType(body.Span)
	}
	return &MethodDecl{
		Name:        name.Lexeme,
		Parameters:  params,
		ReturnType:  returnType,
		Body:        body,
		Private:     private,
		Constructor: constructor,
		Span:        mergeSpans(tokenSpan(start), body.Span),
	}, nil
}
