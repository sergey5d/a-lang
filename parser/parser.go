package parser

import "fmt"

func Parse(input string) (*Program, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}

	parser := &Parser{tokens: tokens}
	return parser.parseProgram()
}

type Parser struct {
	tokens []Token
	pos    int
}

func tokenSpan(token Token) Span {
	return Span{
		Start: Position{Line: token.Line, Column: token.Column},
		End:   Position{Line: token.EndLine, Column: token.EndColumn},
	}
}

func mergeSpans(start, end Span) Span {
	return Span{Start: start.Start, End: end.End}
}

func exprSpan(expr Expr) Span {
	switch e := expr.(type) {
	case *Identifier:
		return e.Span
	case *PlaceholderExpr:
		return e.Span
	case *IntegerLiteral:
		return e.Span
	case *StringLiteral:
		return e.Span
	case *ListLiteral:
		return e.Span
	case *MapLiteral:
		return e.Span
	case *CallExpr:
		return e.Span
	case *MemberExpr:
		return e.Span
	case *LambdaExpr:
		return e.Span
	case *BinaryExpr:
		return e.Span
	case *UnaryExpr:
		return e.Span
	case *GroupExpr:
		return e.Span
	default:
		return Span{}
	}
}

func stmtSpan(stmt Statement) Span {
	switch s := stmt.(type) {
	case *ValStmt:
		return s.Span
	case *IfStmt:
		return s.Span
	case *ForStmt:
		return s.Span
	case *DoYieldStmt:
		return s.Span
	case *MatchStmt:
		return s.Span
	case *ReturnStmt:
		return s.Span
	case *BreakStmt:
		return s.Span
	case *ExprStmt:
		return s.Span
	default:
		return Span{}
	}
}

func (p *Parser) parseProgram() (*Program, error) {
	program := &Program{}
	for !p.isAtEnd() {
		fn, err := p.parseFunction()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}
	if len(program.Functions) > 0 {
		program.Span = mergeSpans(program.Functions[0].Span, program.Functions[len(program.Functions)-1].Span)
	}
	return program, nil
}

func (p *Parser) parseFunction() (*FunctionDecl, error) {
	defToken, err := p.consume(TokenDef, "expected 'def'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected function name")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenLParen, "expected '(' after function name"); err != nil {
		return nil, err
	}

	var params []Parameter
	if !p.check(TokenRParen) {
		for {
			paramName, err := p.consume(TokenIdentifier, "expected parameter name")
			if err != nil {
				return nil, err
			}
			paramType, err := p.consume(TokenIdentifier, "expected parameter type")
			if err != nil {
				return nil, err
			}
			params = append(params, Parameter{
				Name: paramName.Lexeme,
				Type: paramType.Lexeme,
				Span: mergeSpans(tokenSpan(paramName), tokenSpan(paramType)),
			})
			if !p.match(TokenComma) {
				break
			}
		}
	}

	if _, err := p.consume(TokenRParen, "expected ')' after parameters"); err != nil {
		return nil, err
	}

	returnType, err := p.consume(TokenIdentifier, "expected return type")
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &FunctionDecl{
		Name:       name.Lexeme,
		Parameters: params,
		ReturnType: returnType.Lexeme,
		Body:       body,
		Span:       mergeSpans(tokenSpan(defToken), body.Span),
	}, nil
}

func (p *Parser) parseBlock() (*BlockStmt, error) {
	start, err := p.consume(TokenLBrace, "expected '{'")
	if err != nil {
		return nil, err
	}
	block := &BlockStmt{}
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		block.Statements = append(block.Statements, stmt)
	}
	end, err := p.consume(TokenRBrace, "expected '}'")
	if err != nil {
		return nil, err
	}
	block.Span = mergeSpans(tokenSpan(start), tokenSpan(end))
	return block, nil
}

func (p *Parser) parseStatement() (Statement, error) {
	switch p.peek().Type {
	case TokenLet:
		return p.parseBindingStmt(false)
	case TokenMut:
		return p.parseBindingStmt(true)
	case TokenIf:
		return p.parseIfStmt()
	case TokenFor:
		return p.parseForStmt()
	case TokenDo:
		return p.parseDoYieldStmt()
	case TokenRet:
		return p.parseReturnStmt()
	case TokenBreak:
		p.advance()
		return &BreakStmt{}, nil
	default:
		return p.parseExprOrMatchStmt()
	}
}

func (p *Parser) parseBindingStmt(mutable bool) (Statement, error) {
	start := p.advance()

	var bindings []Binding
	for {
		binding := Binding{Mutable: mutable}

		name, err := p.consume(TokenIdentifier, "expected binding name")
		if err != nil {
			return nil, err
		}
		binding.Name = name.Lexeme
		binding.Span = tokenSpan(name)
		if p.check(TokenIdentifier) {
			typeToken := p.advance()
			binding.Type = typeToken.Lexeme
			binding.Span = mergeSpans(binding.Span, tokenSpan(typeToken))
		}
		bindings = append(bindings, binding)
		if !p.match(TokenComma) {
			break
		}
	}

	if _, err := p.consume(TokenAssign, "expected '=' after bindings"); err != nil {
		return nil, err
	}

	values, err := p.parseExprList(TokenRBrace)
	if err != nil {
		return nil, err
	}
	stmt := &ValStmt{Bindings: bindings, Values: values}
	if len(values) > 0 {
		stmt.Span = mergeSpans(tokenSpan(start), exprSpan(values[len(values)-1]))
	} else {
		stmt.Span = tokenSpan(start)
	}
	return stmt, nil
}

func (p *Parser) parseExprList(until TokenType) ([]Expr, error) {
	var values []Expr
	for {
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		values = append(values, expr)
		if !p.match(TokenComma) {
			break
		}
		if p.check(until) {
			break
		}
	}
	return values, nil
}

func (p *Parser) parseIfStmt() (Statement, error) {
	start := p.advance()
	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt := &IfStmt{Condition: condition, Then: thenBlock}
	if p.match(TokenElse) {
		if p.check(TokenIf) {
			elseIfStmt, err := p.parseIfStmt()
			if err != nil {
				return nil, err
			}
			stmt.ElseIf = elseIfStmt.(*IfStmt)
			stmt.Span = mergeSpans(tokenSpan(start), stmt.ElseIf.Span)
			return stmt, nil
		}
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Else = elseBlock
		stmt.Span = mergeSpans(tokenSpan(start), elseBlock.Span)
		return stmt, nil
	}
	stmt.Span = mergeSpans(tokenSpan(start), thenBlock.Span)
	return stmt, nil
}

func (p *Parser) parseForStmt() (Statement, error) {
	start := p.advance()
	name, err := p.consume(TokenIdentifier, "expected loop variable")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenLeftArrow, "expected '<-'"); err != nil {
		return nil, err
	}
	iterable, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ForStmt{
		Name:     name.Lexeme,
		Iterable: iterable,
		Body:     body,
		Span:     mergeSpans(tokenSpan(start), body.Span),
	}, nil
}

func (p *Parser) parseDoYieldStmt() (Statement, error) {
	start := p.advance()
	if _, err := p.consume(TokenLParen, "expected '(' after 'do'"); err != nil {
		return nil, err
	}
	var bindings []ForBinding
	if !p.check(TokenRParen) {
		for {
			name, err := p.consume(TokenIdentifier, "expected generator name")
			if err != nil {
				return nil, err
			}
			if _, err := p.consume(TokenLeftArrow, "expected '<-'"); err != nil {
				return nil, err
			}
			iterable, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			bindings = append(bindings, ForBinding{
				Name:     name.Lexeme,
				Iterable: iterable,
				Span:     mergeSpans(tokenSpan(name), exprSpan(iterable)),
			})
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')' after generators"); err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenYield, "expected 'yield'"); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &DoYieldStmt{Bindings: bindings, Body: body, Span: mergeSpans(tokenSpan(start), body.Span)}, nil
}

func (p *Parser) parseReturnStmt() (Statement, error) {
	start := p.advance()
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: value, Span: mergeSpans(tokenSpan(start), exprSpan(value))}, nil
}

func (p *Parser) parseExprOrMatchStmt() (Statement, error) {
	target, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.match(TokenMatch) {
		if _, err := p.consume(TokenLBrace, "expected '{' after 'match'"); err != nil {
			return nil, err
		}
		var arms []MatchArm
		for !p.check(TokenRBrace) && !p.isAtEnd() {
			arm, err := p.parseMatchArm()
			if err != nil {
				return nil, err
			}
			arms = append(arms, arm)
		}
		if _, err := p.consume(TokenRBrace, "expected '}' after match arms"); err != nil {
			return nil, err
		}
		stmt := &MatchStmt{Target: target, Arms: arms}
		if len(arms) > 0 {
			stmt.Span = mergeSpans(exprSpan(target), arms[len(arms)-1].Span)
		} else {
			stmt.Span = exprSpan(target)
		}
		return stmt, nil
	}
	return &ExprStmt{Expr: target, Span: exprSpan(target)}, nil
}

func (p *Parser) parseMatchArm() (MatchArm, error) {
	// Match arms use ':' as a separator, so the pattern must stop before it.
	pattern, err := p.parseExpression(precedence(TokenColon) + 1)
	if err != nil {
		return MatchArm{}, err
	}

	var patternType string
	if p.check(TokenIdentifier) && p.checkNext(TokenColon) {
		patternType = p.advance().Lexeme
	}

	if _, err := p.consume(TokenColon, "expected ':' after match pattern"); err != nil {
		return MatchArm{}, err
	}
	result, err := p.parseExpression(0)
	if err != nil {
		return MatchArm{}, err
	}
	return MatchArm{
		Pattern:     pattern,
		PatternType: patternType,
		Result:      result,
		Span:        mergeSpans(exprSpan(pattern), exprSpan(result)),
	}, nil
}

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
	case TokenString:
		return &StringLiteral{Value: token.Lexeme, Span: tokenSpan(token)}, nil
	case TokenUnder:
		return &PlaceholderExpr{Span: tokenSpan(token)}, nil
	case TokenLParen:
		inner, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')'"); err != nil {
			return nil, err
		}
		return &GroupExpr{Inner: inner, Span: mergeSpans(tokenSpan(token), tokenSpan(p.previous()))}, nil
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
	case TokenLBrace:
		end, err := p.consume(TokenRBrace, "expected '}' for map literal")
		if err != nil {
			return nil, err
		}
		return &MapLiteral{Span: mergeSpans(tokenSpan(token), tokenSpan(end))}, nil
	default:
		return nil, fmt.Errorf("unexpected token %s", token.String())
	}
}

func (p *Parser) parseLambdaIdentifier() (Expr, error) {
	name, err := p.consume(TokenIdentifier, "expected lambda parameter")
	if err != nil {
		return nil, err
	}
	param := LambdaParameter{Name: name.Lexeme}
	param.Span = tokenSpan(name)
	if p.check(TokenIdentifier) && p.checkNext(TokenArrow) {
		typeToken := p.advance()
		param.Type = typeToken.Lexeme
		param.Span = mergeSpans(param.Span, tokenSpan(typeToken))
	}
	if _, err := p.consume(TokenArrow, "expected '->' after lambda parameter"); err != nil {
		return nil, err
	}
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &LambdaExpr{
		Parameters: []LambdaParameter{param},
		Body:       body,
		Span:       mergeSpans(param.Span, exprSpan(body)),
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
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	startSpan := Span{}
	if len(params) > 0 {
		startSpan = params[0].Span
	}
	return &LambdaExpr{Parameters: params, Body: body, Span: mergeSpans(startSpan, exprSpan(body))}, nil
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
			if p.check(TokenIdentifier) {
				typeToken := p.advance()
				lambdaParam.Type = typeToken.Lexeme
				lambdaParam.Span = mergeSpans(lambdaParam.Span, tokenSpan(typeToken))
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
	return p.checkNext(TokenIdentifier) && p.checkNth(2, TokenArrow)
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
			i++
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

func unaryPrecedence() int {
	return 9
}

func (p *Parser) consume(tt TokenType, message string) (Token, error) {
	if p.check(tt) {
		return p.advance(), nil
	}
	return Token{}, fmt.Errorf("%s, got %s", message, p.peek().String())
}

func (p *Parser) match(tt TokenType) bool {
	if !p.check(tt) {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) check(tt TokenType) bool {
	if p.isAtEnd() {
		return tt == TokenEOF
	}
	return p.peek().Type == tt
}

func (p *Parser) checkNext(tt TokenType) bool {
	if p.pos+1 >= len(p.tokens) {
		return false
	}
	return p.tokens[p.pos+1].Type == tt
}

func (p *Parser) checkNth(offset int, tt TokenType) bool {
	if p.pos+offset >= len(p.tokens) {
		return false
	}
	return p.tokens[p.pos+offset].Type == tt
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) previous() Token {
	return p.tokens[p.pos-1]
}

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == TokenEOF
}
