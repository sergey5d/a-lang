package parser

import "fmt"

func (p *Parser) parseFunction() (*FunctionDecl, error) {
	return p.parseFunctionWithPrivate(false)
}

func (p *Parser) parsePrivateFunction() (*FunctionDecl, error) {
	return p.parseFunctionWithPrivate(true)
}

func (p *Parser) parseFunctionWithPrivate(private bool) (*FunctionDecl, error) {
	defToken, err := p.consume(TokenDef, "expected 'def'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected function name")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParameters()
	if err != nil {
		return nil, err
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	var returnType *TypeRef
	if !p.check(TokenAssign) && !p.check(TokenLBrace) {
		returnType, err = p.parseTypeRef()
		if err != nil {
			return nil, err
		}
	}
	p.beginScope()
	for _, param := range params {
		p.declare(param.Name)
	}
	body, err := p.parseCallableBody()
	p.endScope()
	if err != nil {
		return nil, err
	}
	if returnType == nil {
		returnType = implicitUnitType(body.Span)
	}

	return &FunctionDecl{
		Name:           name.Lexeme,
		TypeParameters: typeParams,
		Parameters:     params,
		ReturnType:     returnType,
		Body:           body,
		Private:        private,
		Span:           mergeSpans(tokenSpan(defToken), body.Span),
	}, nil
}

func (p *Parser) parseCallableBody() (*BlockStmt, error) {
	if p.match(TokenAssign) {
		if p.check(TokenLBrace) {
			return p.parseBlock()
		}
		assign := p.previous()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		stmt := &ExprStmt{Expr: expr, Span: exprSpan(expr)}
		return &BlockStmt{
			Statements: []Statement{stmt},
			Span:       mergeSpans(tokenSpan(assign), exprSpan(expr)),
		}, nil
	}
	return p.parseBlock()
}

func (p *Parser) parseParameters() ([]Parameter, error) {
	if _, err := p.consume(TokenLParen, "expected '('"); err != nil {
		return nil, err
	}
	var params []Parameter
	if !p.check(TokenRParen) {
		for {
			paramName, err := p.consume(TokenIdentifier, "expected parameter name")
			if err != nil {
				return nil, err
			}
			paramType, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			variadic := p.match(TokenEllipsis)
			if variadic && !p.check(TokenRParen) {
				return nil, fmt.Errorf("variadic parameter must be the last parameter at %d:%d", p.previous().Line, p.previous().Column)
			}
			params = append(params, Parameter{
				Name:     paramName.Lexeme,
				Type:     paramType,
				Variadic: variadic,
				Span:     mergeSpans(tokenSpan(paramName), typeSpan(paramType)),
			})
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')' after parameters"); err != nil {
		return nil, err
	}
	return params, nil
}

func (p *Parser) parseTypeParameters() ([]TypeParameter, error) {
	if !p.match(TokenLBracket) {
		return nil, nil
	}
	var params []TypeParameter
	if !p.check(TokenRBracket) {
		for {
			name, err := p.consume(TokenIdentifier, "expected type parameter name")
			if err != nil {
				return nil, err
			}
			param := TypeParameter{Name: name.Lexeme, Span: tokenSpan(name)}
			if p.match(TokenWith) {
				bound, err := p.parseTypeRef()
				if err != nil {
					return nil, err
				}
				param.Bounds = append(param.Bounds, bound)
				param.Span = mergeSpans(tokenSpan(name), typeSpan(bound))
			}
			params = append(params, param)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRBracket, "expected ']' after type parameters"); err != nil {
		return nil, err
	}
	return params, nil
}
