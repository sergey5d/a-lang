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
				endSpan = args[len(args)-1].Span
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
		if op == TokenIs {
			prec := precedence(op)
			if prec < minPrec {
				break
			}
			p.advance()
			target, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			left = &IsExpr{
				Left:   left,
				Target: target,
				Span:   mergeSpans(exprSpan(left), typeSpan(target)),
			}
			continue
		}

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
	if p.check(TokenIdentifier) && p.checkNext(TokenLeftArrow) {
		binding, err := p.parseForBinding()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenYield, "expected 'yield' after for binding"); err != nil {
			return nil, err
		}
		yieldBody, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &ForYieldExpr{
			Bindings:  []ForBinding{binding},
			YieldBody: yieldBody,
			Span:      mergeSpans(tokenSpan(start), yieldBody.Span),
		}, nil
	}
	if !p.check(TokenLBrace) || !p.isForYieldStart() {
		return nil, fmt.Errorf("for expression requires 'for item <- items yield { ... }' or 'for { ... } yield { ... }'")
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

func (p *Parser) parseCallArgs() ([]CallArg, error) {
	if _, err := p.consume(TokenLParen, "expected '('"); err != nil {
		return nil, err
	}
	var args []CallArg
	seenNamed := false
	if !p.check(TokenRParen) {
		for {
			if p.check(TokenIdentifier) && p.checkNext(TokenAssign) {
				nameToken := p.advance()
				p.advance()
				value, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				args = append(args, CallArg{
					Name:  nameToken.Lexeme,
					Value: value,
					Span:  mergeSpans(tokenSpan(nameToken), exprSpan(value)),
				})
				seenNamed = true
			} else {
				if seenNamed {
					return nil, fmt.Errorf("positional arguments cannot follow named arguments")
				}
				expr, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				args = append(args, CallArg{
					Value: expr,
					Span:  exprSpan(expr),
				})
			}
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
	case TokenEqEq, TokenBangEq, TokenIs:
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
