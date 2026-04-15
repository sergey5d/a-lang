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

func (p *Parser) parseProgram() (*Program, error) {
	program := &Program{}
	for !p.isAtEnd() {
		fn, err := p.parseFunction()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}
	return program, nil
}

func (p *Parser) parseFunction() (*FunctionDecl, error) {
	if _, err := p.consume(TokenDef, "expected 'def'"); err != nil {
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
			params = append(params, Parameter{Name: paramName.Lexeme, Type: paramType.Lexeme})
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
	}, nil
}

func (p *Parser) parseBlock() (*BlockStmt, error) {
	if _, err := p.consume(TokenLBrace, "expected '{'"); err != nil {
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
	if _, err := p.consume(TokenRBrace, "expected '}'"); err != nil {
		return nil, err
	}
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
	p.advance()

	var bindings []Binding
	for {
		binding := Binding{Mutable: mutable}

		name, err := p.consume(TokenIdentifier, "expected binding name")
		if err != nil {
			return nil, err
		}
		binding.Name = name.Lexeme
		if p.check(TokenIdentifier) {
			binding.Type = p.advance().Lexeme
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
	return &ValStmt{Bindings: bindings, Values: values}, nil
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
	p.advance()
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
			return stmt, nil
		}
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		stmt.Else = elseBlock
	}
	return stmt, nil
}

func (p *Parser) parseForStmt() (Statement, error) {
	p.advance()
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
	return &ForStmt{Name: name.Lexeme, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseDoYieldStmt() (Statement, error) {
	p.advance()
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
			bindings = append(bindings, ForBinding{Name: name.Lexeme, Iterable: iterable})
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
	return &DoYieldStmt{Bindings: bindings, Body: body}, nil
}

func (p *Parser) parseReturnStmt() (Statement, error) {
	p.advance()
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: value}, nil
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
		return &MatchStmt{Target: target, Arms: arms}, nil
	}
	return &ExprStmt{Expr: target}, nil
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
	return MatchArm{Pattern: pattern, PatternType: patternType, Result: result}, nil
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
			left = &CallExpr{Callee: left, Args: args}
			continue
		}
		if p.match(TokenDot) {
			name, err := p.consume(TokenIdentifier, "expected member name after '.'")
			if err != nil {
				return nil, err
			}
			left = &MemberExpr{Receiver: left, Name: name.Lexeme}
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
		}
	}

	return left, nil
}

func (p *Parser) parsePrefix() (Expr, error) {
	token := p.advance()
	switch token.Type {
	case TokenBang, TokenMinus:
		right, err := p.parseExpression(unaryPrecedence())
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: token.Lexeme, Right: right}, nil
	case TokenIdentifier:
		return &Identifier{Name: token.Lexeme}, nil
	case TokenInteger:
		return &IntegerLiteral{Value: token.Lexeme}, nil
	case TokenString:
		return &StringLiteral{Value: token.Lexeme}, nil
	case TokenUnder:
		return &PlaceholderExpr{}, nil
	case TokenLParen:
		inner, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')'"); err != nil {
			return nil, err
		}
		return &GroupExpr{Inner: inner}, nil
	case TokenLBracket:
		if p.match(TokenRBracket) {
			return &ListLiteral{}, nil
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
		return &ListLiteral{Elements: items}, nil
	case TokenLBrace:
		if _, err := p.consume(TokenRBrace, "expected '}' for map literal"); err != nil {
			return nil, err
		}
		return &MapLiteral{}, nil
	default:
		return nil, fmt.Errorf("unexpected token %s", token.String())
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

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == TokenEOF
}
