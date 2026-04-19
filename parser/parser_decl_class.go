package parser

import "fmt"

func (p *Parser) parseClass() (*ClassDecl, error) {
	start, err := p.consume(TokenClass, "expected 'class'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected class name")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParameters()
	if err != nil {
		return nil, err
	}
	decl := &ClassDecl{Name: name.Lexeme, TypeParameters: typeParams}
	if p.match(TokenWith) {
		for {
			target, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			decl.Implements = append(decl.Implements, target)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenLBrace, "expected '{' after class name"); err != nil {
		return nil, err
	}
	sawMethod := false
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		private := p.match(TokenPrivate)
		switch p.peek().Type {
		case TokenIdentifier:
			if sawMethod {
				return nil, fmt.Errorf("class fields must appear before method declarations")
			}
			field, err := p.parseField(private)
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, field)
		case TokenDef:
			sawMethod = true
			method, err := p.parseMethod(private)
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, method)
		default:
			return nil, fmt.Errorf("expected class member, got %s", p.peek().String())
		}
	}
	end, err := p.consume(TokenRBrace, "expected '}' after class body")
	if err != nil {
		return nil, err
	}
	decl.Span = mergeSpans(tokenSpan(start), tokenSpan(end))
	return decl, nil
}

func (p *Parser) parseField(private bool) (FieldDecl, error) {
	start, err := p.consume(TokenIdentifier, "expected field name")
	if err != nil {
		return FieldDecl{}, err
	}
	name := start
	typ, err := p.parseTypeRef()
	if err != nil {
		return FieldDecl{}, err
	}
	field := FieldDecl{
		Name:     name.Lexeme,
		Type:     typ,
		Deferred: true,
		Private:  private,
		Span:     mergeSpans(tokenSpan(start), typeSpan(typ)),
	}
	switch p.peek().Type {
	case TokenAssign, TokenColonAssign:
		operator := p.advance()
		field.Mutable = operator.Type == TokenColonAssign
		if p.match(TokenDeferred) {
			field.Deferred = true
			field.Span = mergeSpans(field.Span, tokenSpan(p.previous()))
			return field, nil
		}
		expr, err := p.parseExpression(0)
		if err != nil {
			return FieldDecl{}, err
		}
		field.Initializer = expr
		field.Deferred = false
		field.Span = mergeSpans(field.Span, exprSpan(expr))
	}
	return field, nil
}

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
	constructor := name.Lexeme == "init" || name.Lexeme == "this"
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
