package parser

import "fmt"

func (p *Parser) parseExpression(minPrec int) (Expr, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	for {
		if p.check(TokenLParen) {
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			call := &CallExpr{Callee: left, Args: args}
			endSpan := exprSpan(left)
			if len(args) > 0 {
				endSpan = exprSpan(args[len(args)-1])
			}
			call.Span = mergeSpans(exprSpan(left), endSpan)
			left = call
			continue
		}
		if p.match(TokenDot) {
			name, err := p.consume(TokenIdentifier, "expected member name after '.'")
			if err != nil {
				return nil, err
			}
			left = &MemberExpr{Receiver: left, Name: name.Lexeme, Span: mergeSpans(exprSpan(left), tokenSpan(name))}
			continue
		}
		if p.match(TokenLBracket) {
			index, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			end, err := p.consume(TokenRBracket, "expected ']' after index expression")
			if err != nil {
				return nil, err
			}
			left = &IndexExpr{Receiver: left, Index: index, Span: mergeSpans(exprSpan(left), tokenSpan(end))}
			continue
		}

		op := p.peek().Type
		prec := precedence(op)
		if prec < minPrec {
			break
		}

		token := p.advance()
		right, err := p.parseExpression(prec + 1)
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:     left,
			Operator: token.Lexeme,
			Right:    right,
			Span:     mergeSpans(exprSpan(left), exprSpan(right)),
		}
	}

	return left, nil
}

func (p *Parser) parsePrefix() (Expr, error) {
	if p.isLambdaIdentifierStart() {
		return p.parseLambdaIdentifier()
	}
	if p.check(TokenLParen) && p.isLambdaParenStart() {
		return p.parseLambdaParen()
	}

	token := p.advance()
	switch token.Type {
	case TokenBang, TokenMinus:
		right, err := p.parseExpression(unaryPrecedence())
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: token.Lexeme, Right: right, Span: mergeSpans(tokenSpan(token), exprSpan(right))}, nil
	case TokenIdentifier:
		return &Identifier{Name: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenInteger:
		return &IntegerLiteral{Value: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenFloat:
		return &FloatLiteral{Value: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenRune:
		return &RuneLiteral{Value: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenBool:
		return &BoolLiteral{Value: token.Lexeme == "true", Span: tokenSpan(token)}, nil
	case TokenString:
		return &StringLiteral{Value: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenUnder:
		return &PlaceholderExpr{Span: tokenSpan(token)}, nil
	case TokenLParen:
		return p.parseParenExpr(token)
	case TokenLBracket:
		if p.match(TokenRBracket) {
			return &ListLiteral{Span: mergeSpans(tokenSpan(token), tokenSpan(p.previous()))}, nil
		}
		var items []Expr
		for {
			expr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			items = append(items, expr)
			if !p.match(TokenComma) {
				break
			}
		}
		if _, err := p.consume(TokenRBracket, "expected ']'"); err != nil {
			return nil, err
		}
		return &ListLiteral{Elements: items, Span: mergeSpans(tokenSpan(token), tokenSpan(p.previous()))}, nil
	case TokenIf:
		return p.parseIfExprAfterStart(token)
	case TokenFor:
		return p.parseForYieldExprAfterStart(token)
	default:
		return nil, fmt.Errorf("unexpected token %s", token.String())
	}
}

func (p *Parser) parseIfExprAfterStart(start Token) (Expr, error) {
	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenElse, "expected 'else' in if expression"); err != nil {
		return nil, err
	}
	elseBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &IfExpr{
		Condition: condition,
		Then:      thenBlock,
		Else:      elseBlock,
		Span:      mergeSpans(tokenSpan(start), elseBlock.Span),
	}, nil
}

func (p *Parser) parseForYieldExprAfterStart(start Token) (Expr, error) {
	if !p.check(TokenLBrace) || !p.isForYieldStart() {
		return nil, fmt.Errorf("for expression requires 'for { ... } yield { ... }'")
	}
	bindings, err := p.parseForBindingsBlock()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenYield, "expected 'yield' after for bindings"); err != nil {
		return nil, err
	}
	yieldBody, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ForYieldExpr{
		Bindings:  bindings,
		YieldBody: yieldBody,
		Span:      mergeSpans(tokenSpan(start), yieldBody.Span),
	}, nil
}

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

func (p *Parser) parseCallArgs() ([]Expr, error) {
	if _, err := p.consume(TokenLParen, "expected '('"); err != nil {
		return nil, err
	}
	var args []Expr
	if !p.check(TokenRParen) {
		for {
			expr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')' after arguments"); err != nil {
		return nil, err
	}
	return args, nil
}

func precedence(t TokenType) int {
	switch t {
	case TokenOrOr:
		return 1
	case TokenAndAnd:
		return 2
	case TokenEqEq, TokenBangEq:
		return 3
	case TokenLT, TokenLTE, TokenGT, TokenGTE:
		return 4
	case TokenPlus, TokenMinus:
		return 5
	case TokenStar, TokenSlash, TokenPercent:
		return 6
	case TokenColon:
		return 7
	case TokenRange:
		return 8
	default:
		return -1
	}
}

func (p *Parser) parseParenExpr(start Token) (Expr, error) {
	if p.check(TokenRParen) {
		end, err := p.consume(TokenRParen, "expected ')'")
		if err != nil {
			return nil, err
		}
		return &UnitLiteral{Span: mergeSpans(tokenSpan(start), tokenSpan(end))}, nil
	}
	first, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if !p.match(TokenComma) {
		if _, err := p.consume(TokenRParen, "expected ')'"); err != nil {
			return nil, err
		}
		return &GroupExpr{Inner: first, Span: mergeSpans(tokenSpan(start), tokenSpan(p.previous()))}, nil
	}
	elements := []Expr{first}
	for {
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		elements = append(elements, expr)
		if !p.match(TokenComma) {
			break
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')'"); err != nil {
		return nil, err
	}
	return &TupleLiteral{Elements: elements, Span: mergeSpans(tokenSpan(start), tokenSpan(p.previous()))}, nil
}

func unaryPrecedence() int {
	return 9
}
