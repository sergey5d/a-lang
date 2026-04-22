package parser

import "fmt"

func (p *Parser) parseBlock() (*BlockStmt, error) {
	start, err := p.consume(TokenLBrace, "expected '{'")
	if err != nil {
		return nil, err
	}
	p.beginScope()
	defer p.endScope()
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
	case TokenIf:
		return p.parseIfStmt()
	case TokenLoop:
		return p.parseLoopStmt()
	case TokenFor:
		return p.parseForStmt()
	case TokenDef:
		return p.parseLocalFunctionStmt()
	case TokenReturn:
		return p.parseReturnStmt()
	case TokenBreak:
		token := p.advance()
		return &BreakStmt{Span: tokenSpan(token)}, nil
	default:
		if p.isBareBindingStart() {
			return p.parseBareBindingStmt()
		}
		return p.parseExprStmt()
	}
}

func (p *Parser) parseLocalFunctionStmt() (Statement, error) {
	fn, err := p.parseFunction()
	if err != nil {
		return nil, err
	}
	return &LocalFunctionStmt{Function: fn, Span: fn.Span}, nil
}

func (p *Parser) parseBareBindingStmt() (Statement, error) {
	var start Token
	var err error
	if p.check(TokenIdentifier) {
		start, err = p.consume(TokenIdentifier, "expected binding name")
	} else {
		start, err = p.consume(TokenUnder, "expected binding name")
	}
	if err != nil {
		return nil, err
	}
	return p.parseBindingStmtWithStart(start, true)
}

func (p *Parser) parseBindingStmtWithStart(start Token, firstIsName bool) (Statement, error) {
	bindings, err := p.parseBindingsWithStart(start, firstIsName)
	if err != nil {
		return nil, err
	}

	operator := p.peek().Type
	if operator != TokenAssign && operator != TokenColonAssign {
		return nil, fmt.Errorf("expected '=' or ':=' after bindings, got %s", p.peek().String())
	}
	p.advance()
	mutable := operator == TokenColonAssign
	for i := range bindings {
		bindings[i].Mutable = mutable
	}

	values, err := p.parseBindingInitializers(len(bindings))
	if err != nil {
		return nil, err
	}
	for i := range bindings {
		if i < len(values) && values[i] != nil {
			values[i] = wrapThunkExpr(bindings[i].Type, values[i])
		}
	}
	stmt := &ValStmt{Bindings: bindings, Values: values}
	stmt.Span = tokenSpan(start)
	for i := range bindings {
		if i < len(values) && values[i] != nil {
			stmt.Span = mergeSpans(stmt.Span, exprSpan(values[i]))
			continue
		}
		if i >= len(values) {
			continue
		}
		bindings[i].Deferred = true
		stmt.Bindings[i].Deferred = true
		if i == len(bindings)-1 {
			stmt.Span = mergeSpans(stmt.Span, bindings[i].Span)
		}
	}
	for _, binding := range bindings {
		if binding.Name != "_" {
			p.declare(binding.Name)
		}
	}
	return stmt, nil
}

func (p *Parser) parseBindingsWithStart(start Token, firstIsName bool) ([]Binding, error) {
	var bindings []Binding
	useStart := start
	first := true
	for {
		binding := Binding{}

		name := useStart
		if !first || !firstIsName {
			var err error
			if p.check(TokenIdentifier) {
				name, err = p.consume(TokenIdentifier, "expected binding name")
			} else {
				name, err = p.consume(TokenUnder, "expected binding name")
			}
			if err != nil {
				return nil, err
			}
		}
		binding.Name = name.Lexeme
		binding.Span = tokenSpan(name)
		if p.check(TokenIdentifier) || p.check(TokenLParen) {
			typeRef, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			binding.Type = typeRef
			binding.Span = mergeSpans(binding.Span, typeSpan(typeRef))
		}
		bindings = append(bindings, binding)
		if !p.match(TokenComma) {
			break
		}
		first = false
		useStart = Token{}
	}
	return bindings, nil
}

func (p *Parser) isBareBindingStart() bool {
	if !p.check(TokenIdentifier) && !p.check(TokenUnder) {
		return false
	}

	if p.checkNext(TokenAssign) {
		return true
	}

	if p.checkNext(TokenColonAssign) {
		return !p.isDeclared(p.peek().Lexeme)
	}

	if p.checkNext(TokenIdentifier) || p.checkNext(TokenLParen) || p.checkNext(TokenComma) {
		return p.bindingListFollowedByAssign(p.pos)
	}

	return false
}

func (p *Parser) bindingListFollowedByAssign(start int) bool {
	i := start
	sawType := false
	sawUndeclared := false
	for {
		if i >= len(p.tokens) || (p.tokens[i].Type != TokenIdentifier && p.tokens[i].Type != TokenUnder) {
			return false
		}
		if p.tokens[i].Type == TokenIdentifier && !p.isDeclared(p.tokens[i].Lexeme) {
			sawUndeclared = true
		}
		i++
		if i < len(p.tokens) && (p.tokens[i].Type == TokenIdentifier || p.tokens[i].Type == TokenLParen) {
			end, ok := p.scanTypeRef(i)
			if !ok {
				return false
			}
			sawType = true
			i = end
		}
		if i >= len(p.tokens) {
			return false
		}
		if p.tokens[i].Type == TokenAssign {
			return true
		}
		if p.tokens[i].Type == TokenColonAssign {
			return sawType || sawUndeclared
		}
		if p.tokens[i].Type != TokenComma {
			return false
		}
		i++
	}
}

func (p *Parser) parseBindingInitializers(count int) ([]Expr, error) {
	if count <= 0 {
		return nil, nil
	}
	if p.match(TokenQuestion) {
		values := []Expr{nil}
		for len(values) < count && p.match(TokenComma) {
			if !p.match(TokenQuestion) {
				return nil, fmt.Errorf("expected '?' after ',' in initializer list, got %s", p.peek().String())
			}
			values = append(values, nil)
		}
		return values, nil
	}
	first, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	values := []Expr{first}
	if count > 1 && !p.check(TokenComma) {
		return values, nil
	}
	for len(values) < count && p.match(TokenComma) {
		if p.match(TokenQuestion) {
			values = append(values, nil)
			continue
		}
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		values = append(values, expr)
	}
	return values, nil
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
	return p.parseIfStmtAfterStart(start)
}

func (p *Parser) parseIfStmtAfterStart(start Token) (Statement, error) {
	stmt := &IfStmt{}
	if (p.check(TokenIdentifier) || p.check(TokenUnder)) && p.bindingListFollowedByArrow(p.pos) {
		var name Token
		var err error
		if p.check(TokenIdentifier) {
			name, err = p.consume(TokenIdentifier, "expected binding name after 'if'")
		} else {
			name, err = p.consume(TokenUnder, "expected binding name after 'if'")
		}
		if err != nil {
			return nil, err
		}
		bindings, err := p.parseBindingsWithStart(name, true)
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenLeftArrow, "expected '<-' after if binding"); err != nil {
			return nil, err
		}
		value, err := p.parseExpressionWithOptions(0, false)
		if err != nil {
			return nil, err
		}
		stmt.Bindings = bindings
		stmt.BindingValue = value
	} else {
		condition, err := p.parseExpressionWithOptions(0, false)
		if err != nil {
			return nil, err
		}
		stmt.Condition = condition
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt.Then = thenBlock
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

func (p *Parser) bindingListFollowedByArrow(start int) bool {
	if start >= len(p.tokens) || (p.tokens[start].Type != TokenIdentifier && p.tokens[start].Type != TokenUnder) {
		return false
	}

	i := start + 1
	for {
		if i < len(p.tokens) && (p.tokens[i].Type == TokenIdentifier || p.tokens[i].Type == TokenLParen) {
			end, ok := p.scanTypeRef(i)
			if !ok {
				return false
			}
			i = end
		}
		if i < len(p.tokens) && p.tokens[i].Type == TokenLeftArrow {
			return true
		}
		if i >= len(p.tokens) || p.tokens[i].Type != TokenComma {
			return false
		}
		i++
		if i >= len(p.tokens) || (p.tokens[i].Type != TokenIdentifier && p.tokens[i].Type != TokenUnder) {
			return false
		}
		i++
	}
}

func (p *Parser) parseForStmt() (Statement, error) {
	start := p.advance()
	return p.parseForStmtAfterStart(start)
}

func (p *Parser) parseLoopStmt() (Statement, error) {
	start := p.advance()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &LoopStmt{Body: body, Span: mergeSpans(tokenSpan(start), body.Span)}, nil
}

func (p *Parser) parseForStmtAfterStart(start Token) (Statement, error) {
	p.beginScope()
	defer p.endScope()
	if (p.check(TokenIdentifier) || p.check(TokenUnder)) && p.bindingListFollowedByArrow(p.pos) {
		binding, err := p.parseForClause()
		if err != nil {
			return nil, err
		}
		p.declareBindings(binding.Bindings)
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &ForStmt{
			Bindings: []ForBinding{binding},
			Body:     body,
			Span:     mergeSpans(tokenSpan(start), body.Span),
		}, nil
	}
	if p.check(TokenLBrace) && p.isForYieldStart() {
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
		return &ForStmt{
			Bindings:  bindings,
			YieldBody: yieldBody,
			Span:      mergeSpans(tokenSpan(start), yieldBody.Span),
		}, nil
	}
	return nil, fmt.Errorf("for loop requires bindings like 'for item <- items { ... }' or a yield form")
}

func (p *Parser) parseForClause() (ForBinding, error) {
	var name Token
	var err error
	if p.check(TokenIdentifier) {
		name, err = p.consume(TokenIdentifier, "expected loop variable")
	} else {
		name, err = p.consume(TokenUnder, "expected loop variable")
	}
	if err != nil {
		return ForBinding{}, err
	}
	bindings, err := p.parseBindingsWithStart(name, true)
	if err != nil {
		return ForBinding{}, err
	}
	switch p.peek().Type {
	case TokenLeftArrow:
		p.advance()
		iterable, err := p.parseExpressionWithOptions(0, false)
		if err != nil {
			return ForBinding{}, err
		}
		return ForBinding{
			Bindings: bindings,
			Iterable: iterable,
			Span:     mergeSpans(tokenSpan(name), exprSpan(iterable)),
		}, nil
	case TokenAssign, TokenColonAssign:
		operator := p.advance().Type
		mutable := operator == TokenColonAssign
		for i := range bindings {
			bindings[i].Mutable = mutable
		}
		values, err := p.parseBindingInitializers(len(bindings))
		if err != nil {
			return ForBinding{}, err
		}
		for i := range bindings {
			if i < len(values) && values[i] != nil {
				values[i] = wrapThunkExpr(bindings[i].Type, values[i])
			}
		}
		end := tokenSpan(name)
		for i, value := range values {
			if value != nil {
				end = mergeSpans(end, exprSpan(value))
				continue
			}
			if i < len(bindings) {
				end = mergeSpans(end, bindings[i].Span)
			}
		}
		return ForBinding{
			Bindings: bindings,
			Values:   values,
			Span:     end,
		}, nil
	default:
		return ForBinding{}, fmt.Errorf("expected '<-' or '=' after for bindings, got %s", p.peek().String())
	}
}

func (p *Parser) parseForBindingsBlock() ([]ForBinding, error) {
	if _, err := p.consume(TokenLBrace, "expected '{' after 'for'"); err != nil {
		return nil, err
	}
	var bindings []ForBinding
	if !p.check(TokenRBrace) {
		for {
			binding, err := p.parseForClause()
			if err != nil {
				return nil, err
			}
			bindings = append(bindings, binding)
			p.declareBindings(binding.Bindings)
			if p.match(TokenComma) {
				continue
			}
			if (p.check(TokenIdentifier) || p.check(TokenUnder)) &&
				(p.bindingListFollowedByArrow(p.pos) || p.bindingListFollowedByAssign(p.pos)) {
				continue
			}
			break
		}
	}
	if _, err := p.consume(TokenRBrace, "expected '}' after for bindings"); err != nil {
		return nil, err
	}
	return bindings, nil
}

func (p *Parser) declareBindings(bindings []Binding) {
	for _, binding := range bindings {
		if binding.Name != "_" {
			p.declare(binding.Name)
		}
	}
}

func (p *Parser) isForYieldStart() bool {
	if !p.check(TokenLBrace) {
		return false
	}
	depthBrace := 0
	for i := p.pos; i < len(p.tokens); i++ {
		switch p.tokens[i].Type {
		case TokenLBrace:
			depthBrace++
		case TokenRBrace:
			depthBrace--
			if depthBrace == 0 {
				return i+1 < len(p.tokens) && p.tokens[i+1].Type == TokenYield
			}
		}
	}
	return false
}

func (p *Parser) parseReturnStmt() (Statement, error) {
	start := p.advance()
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: value, Span: mergeSpans(tokenSpan(start), exprSpan(value))}, nil
}

func (p *Parser) parseExprStmt() (Statement, error) {
	target, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.match(TokenComma) {
		targets := []Expr{target}
		for {
			nextTarget, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			targets = append(targets, nextTarget)
			if !p.match(TokenComma) {
				break
			}
		}
		if !isAssignmentOperator(p.peek().Type) {
			return nil, fmt.Errorf("expected assignment operator after assignment targets at %d:%d", p.peek().Line, p.peek().Column)
		}
		operator := p.advance()
		values, err := p.parseAssignmentValues()
		if err != nil {
			return nil, err
		}
		return &MultiAssignmentStmt{
			Targets:  targets,
			Operator: operator.Lexeme,
			Values:   values,
			Span:     mergeSpans(exprSpan(targets[0]), exprSpan(values[len(values)-1])),
		}, nil
	}
	if isAssignmentOperator(p.peek().Type) {
		operator := p.advance()
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return &AssignmentStmt{
			Target:   target,
			Operator: operator.Lexeme,
			Value:    value,
			Span:     mergeSpans(exprSpan(target), exprSpan(value)),
		}, nil
	}
	return &ExprStmt{Expr: target, Span: exprSpan(target)}, nil
}

func (p *Parser) parseAssignmentValues() ([]Expr, error) {
	values := []Expr{}
	for {
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		if !p.match(TokenComma) {
			break
		}
	}
	return values, nil
}

func isAssignmentOperator(tt TokenType) bool {
	switch tt {
	case TokenAssign, TokenColonAssign, TokenPlusEq, TokenMinusEq, TokenStarEq, TokenSlashEq, TokenPercentEq:
		return true
	default:
		return false
	}
}
