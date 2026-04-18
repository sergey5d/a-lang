package parser

func (p *Parser) parseLambdaIdentifier() (Expr, error) {
	name, err := p.consume(TokenIdentifier, "expected lambda parameter")
	if err != nil {
		return nil, err
	}
	param := LambdaParameter{Name: name.Lexeme}
	param.Span = tokenSpan(name)
	if p.check(TokenIdentifier) && p.simpleTypeRefFollowedBy(TokenArrow) {
		typeRef, err := p.parseNamedTypeRef()
		if err != nil {
			return nil, err
		}
		param.Type = typeRef
		param.Span = mergeSpans(param.Span, typeSpan(typeRef))
	}
	if _, err := p.consume(TokenArrow, "expected '->' after lambda parameter"); err != nil {
		return nil, err
	}
	body, blockBody, endSpan, err := p.parseLambdaBody()
	if err != nil {
		return nil, err
	}
	return &LambdaExpr{
		Parameters: []LambdaParameter{param},
		Body:       body,
		BlockBody:  blockBody,
		Span:       mergeSpans(param.Span, endSpan),
	}, nil
}

func (p *Parser) parseLambdaParen() (Expr, error) {
	params, err := p.parseLambdaParams()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenArrow, "expected '->' after lambda parameters"); err != nil {
		return nil, err
	}
	body, blockBody, endSpan, err := p.parseLambdaBody()
	if err != nil {
		return nil, err
	}
	startSpan := Span{}
	if len(params) > 0 {
		startSpan = params[0].Span
	}
	return &LambdaExpr{Parameters: params, Body: body, BlockBody: blockBody, Span: mergeSpans(startSpan, endSpan)}, nil
}

func (p *Parser) parseLambdaBody() (Expr, *BlockStmt, Span, error) {
	if p.check(TokenLBrace) {
		block, err := p.parseBlock()
		if err != nil {
			return nil, nil, Span{}, err
		}
		return nil, block, block.Span, nil
	}
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, nil, Span{}, err
	}
	return body, nil, exprSpan(body), nil
}

func (p *Parser) parseLambdaParams() ([]LambdaParameter, error) {
	if _, err := p.consume(TokenLParen, "expected '('"); err != nil {
		return nil, err
	}
	var params []LambdaParameter
	if !p.check(TokenRParen) {
		for {
			param, err := p.consume(TokenIdentifier, "expected lambda parameter")
			if err != nil {
				return nil, err
			}
			lambdaParam := LambdaParameter{Name: param.Lexeme}
			lambdaParam.Span = tokenSpan(param)
			if (p.check(TokenIdentifier) || p.check(TokenLParen)) && (p.typeRefFollowedBy(TokenComma) || p.typeRefFollowedBy(TokenRParen)) {
				typeRef, err := p.parseTypeRef()
				if err != nil {
					return nil, err
				}
				lambdaParam.Type = typeRef
				lambdaParam.Span = mergeSpans(lambdaParam.Span, typeSpan(typeRef))
			}
			params = append(params, lambdaParam)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')' after lambda parameters"); err != nil {
		return nil, err
	}
	return params, nil
}

func (p *Parser) isLambdaIdentifierStart() bool {
	if !p.check(TokenIdentifier) {
		return false
	}
	if p.checkNext(TokenArrow) {
		return true
	}
	return p.checkNext(TokenIdentifier) && p.simpleTypeRefFollowedByAt(p.pos+1, TokenArrow)
}

func (p *Parser) isLambdaParenStart() bool {
	if !p.check(TokenLParen) {
		return false
	}
	i := p.pos + 1
	if p.tokens[p.pos].Type != TokenLParen {
		return false
	}
	if i >= len(p.tokens) {
		return false
	}
	if p.tokens[i].Type == TokenRParen {
		return i+1 < len(p.tokens) && p.tokens[i+1].Type == TokenArrow
	}
	for {
		if i >= len(p.tokens) || p.tokens[i].Type != TokenIdentifier {
			return false
		}
		i++
		if i < len(p.tokens) && p.tokens[i].Type == TokenIdentifier {
			end, ok := p.scanTypeRef(i)
			if !ok {
				return false
			}
			i = end
		}
		if i >= len(p.tokens) {
			return false
		}
		if p.tokens[i].Type == TokenComma {
			i++
			continue
		}
		if p.tokens[i].Type == TokenRParen {
			return i+1 < len(p.tokens) && p.tokens[i+1].Type == TokenArrow
		}
		return false
	}
}
