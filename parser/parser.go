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

func typeSpan(ref *TypeRef) Span {
	if ref == nil {
		return Span{}
	}
	return ref.Span
}

func exprSpan(expr Expr) Span {
	switch e := expr.(type) {
	case *Identifier:
		return e.Span
	case *PlaceholderExpr:
		return e.Span
	case *IntegerLiteral:
		return e.Span
	case *FloatLiteral:
		return e.Span
	case *RuneLiteral:
		return e.Span
	case *BoolLiteral:
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
	case *AssignmentStmt:
		return s.Span
	case *IfStmt:
		return s.Span
	case *ForStmt:
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
		switch p.peek().Type {
		case TokenDef:
			fn, err := p.parseFunction()
			if err != nil {
				return nil, err
			}
			program.Functions = append(program.Functions, fn)
		case TokenInterface:
			decl, err := p.parseInterface()
			if err != nil {
				return nil, err
			}
			program.Interfaces = append(program.Interfaces, decl)
		case TokenClass:
			decl, err := p.parseClass()
			if err != nil {
				return nil, err
			}
			program.Classes = append(program.Classes, decl)
		default:
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			program.Statements = append(program.Statements, stmt)
		}
	}
	if span, ok := p.programSpan(program); ok {
		program.Span = span
	}
	return program, nil
}

func (p *Parser) programSpan(program *Program) (Span, bool) {
	var spans []Span
	for _, fn := range program.Functions {
		spans = append(spans, fn.Span)
	}
	for _, decl := range program.Interfaces {
		spans = append(spans, decl.Span)
	}
	for _, decl := range program.Classes {
		spans = append(spans, decl.Span)
	}
	for _, stmt := range program.Statements {
		spans = append(spans, stmtSpan(stmt))
	}
	if len(spans) == 0 {
		return Span{}, false
	}
	return mergeSpans(spans[0], spans[len(spans)-1]), true
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
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	returnType, err := p.parseTypeRef()
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
		ReturnType: returnType,
		Body:       body,
		Span:       mergeSpans(tokenSpan(defToken), body.Span),
	}, nil
}

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
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		private := p.match(TokenPrivate)
		switch p.peek().Type {
		case TokenLet, TokenVar:
			field, err := p.parseField(private)
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, field)
		case TokenDef:
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
	start := p.advance()
	mutable := start.Type == TokenVar
	name, err := p.consume(TokenIdentifier, "expected field name")
	if err != nil {
		return FieldDecl{}, err
	}
	typ, err := p.parseTypeRef()
	if err != nil {
		return FieldDecl{}, err
	}
	return FieldDecl{
		Name:    name.Lexeme,
		Type:    typ,
		Mutable: mutable,
		Private: private,
		Span:    mergeSpans(tokenSpan(start), typeSpan(typ)),
	}, nil
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
	constructor := name.Lexeme == "init"
	var returnType *TypeRef
	if !constructor {
		typ, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		returnType = typ
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
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
			params = append(params, Parameter{
				Name: paramName.Lexeme,
				Type: paramType,
				Span: mergeSpans(tokenSpan(paramName), typeSpan(paramType)),
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
	case TokenVar:
		return p.parseBindingStmt(true)
	case TokenIf:
		return p.parseIfStmt()
	case TokenFor:
		return p.parseForStmt()
	case TokenReturn:
		return p.parseReturnStmt()
	case TokenBreak:
		token := p.advance()
		return &BreakStmt{Span: tokenSpan(token)}, nil
	default:
		return p.parseExprStmt()
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
	if p.check(TokenIdentifier) && p.checkNext(TokenLeftArrow) {
		binding, err := p.parseForBinding()
		if err != nil {
			return nil, err
		}
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
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ForStmt{Body: body, Span: mergeSpans(tokenSpan(start), body.Span)}, nil
}

func (p *Parser) parseForBinding() (ForBinding, error) {
	name, err := p.consume(TokenIdentifier, "expected loop variable")
	if err != nil {
		return ForBinding{}, err
	}
	if _, err := p.consume(TokenLeftArrow, "expected '<-'"); err != nil {
		return ForBinding{}, err
	}
	iterable, err := p.parseExpression(0)
	if err != nil {
		return ForBinding{}, err
	}
	return ForBinding{
		Name:     name.Lexeme,
		Iterable: iterable,
		Span:     mergeSpans(tokenSpan(name), exprSpan(iterable)),
	}, nil
}

func (p *Parser) parseForBindingsBlock() ([]ForBinding, error) {
	if _, err := p.consume(TokenLBrace, "expected '{' after 'for'"); err != nil {
		return nil, err
	}
	var bindings []ForBinding
	if !p.check(TokenRBrace) {
		for {
			binding, err := p.parseForBinding()
			if err != nil {
				return nil, err
			}
			bindings = append(bindings, binding)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRBrace, "expected '}' after for bindings"); err != nil {
		return nil, err
	}
	return bindings, nil
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

func isAssignmentOperator(tt TokenType) bool {
	switch tt {
	case TokenAssign, TokenPlusEq, TokenMinusEq, TokenStarEq, TokenSlashEq, TokenPercentEq:
		return true
	default:
		return false
	}
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
			params = append(params, TypeParameter{Name: name.Lexeme, Span: tokenSpan(name)})
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

func (p *Parser) parseTypeRef() (*TypeRef, error) {
	if p.check(TokenLParen) {
		return p.parseParenFunctionTypeRef()
	}
	return p.parseArrowTypeRef()
}

func (p *Parser) parseArrowTypeRef() (*TypeRef, error) {
	left, err := p.parseNamedTypeRef()
	if err != nil {
		return nil, err
	}
	if p.match(TokenArrow) {
		returnType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		return &TypeRef{
			ParameterTypes: []*TypeRef{left},
			ReturnType:     returnType,
			Span:           mergeSpans(left.Span, typeSpan(returnType)),
		}, nil
	}
	return left, nil
}

func (p *Parser) parseNamedTypeRef() (*TypeRef, error) {
	name, err := p.consume(TokenIdentifier, "expected type name")
	if err != nil {
		return nil, err
	}
	ref := &TypeRef{Name: name.Lexeme, Span: tokenSpan(name)}
	if p.match(TokenLBracket) {
		for {
			arg, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			ref.Arguments = append(ref.Arguments, arg)
			if !p.match(TokenComma) {
				break
			}
		}
		end, err := p.consume(TokenRBracket, "expected ']' after type arguments")
		if err != nil {
			return nil, err
		}
		ref.Span = mergeSpans(ref.Span, tokenSpan(end))
	}
	return ref, nil
}

func (p *Parser) parseParenFunctionTypeRef() (*TypeRef, error) {
	start, err := p.consume(TokenLParen, "expected '('")
	if err != nil {
		return nil, err
	}
	var params []*TypeRef
	if !p.check(TokenRParen) {
		for {
			param, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRParen, "expected ')' after function type parameters"); err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenArrow, "expected '->' after function type parameters"); err != nil {
		return nil, err
	}
	returnType, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	return &TypeRef{
		ParameterTypes: params,
		ReturnType:     returnType,
		Span:           mergeSpans(tokenSpan(start), typeSpan(returnType)),
	}, nil
}

func (p *Parser) typeRefFollowedBy(tt TokenType) bool {
	return p.typeRefFollowedByAt(p.pos, tt)
}

func (p *Parser) simpleTypeRefFollowedBy(tt TokenType) bool {
	return p.simpleTypeRefFollowedByAt(p.pos, tt)
}

func (p *Parser) typeRefFollowedByAt(start int, tt TokenType) bool {
	end, ok := p.scanTypeRef(start)
	if !ok || end >= len(p.tokens) {
		return false
	}
	return p.tokens[end].Type == tt
}

func (p *Parser) simpleTypeRefFollowedByAt(start int, tt TokenType) bool {
	end, ok := p.scanSimpleTypeRef(start)
	if !ok || end >= len(p.tokens) {
		return false
	}
	return p.tokens[end].Type == tt
}

func (p *Parser) scanTypeRef(start int) (int, bool) {
	if start >= len(p.tokens) {
		return start, false
	}
	if p.tokens[start].Type == TokenLParen {
		i := start + 1
		if i < len(p.tokens) && p.tokens[i].Type != TokenRParen {
			for {
				var ok bool
				i, ok = p.scanTypeRef(i)
				if !ok || i >= len(p.tokens) {
					return start, false
				}
				if p.tokens[i].Type == TokenComma {
					i++
					continue
				}
				if p.tokens[i].Type == TokenRParen {
					i++
					break
				}
				return start, false
			}
		} else if i < len(p.tokens) && p.tokens[i].Type == TokenRParen {
			i++
		} else {
			return start, false
		}
		if i >= len(p.tokens) || p.tokens[i].Type != TokenArrow {
			return start, false
		}
		return p.scanTypeRef(i + 1)
	}
	if p.tokens[start].Type != TokenIdentifier {
		return start, false
	}
	i := start + 1
	if i < len(p.tokens) && p.tokens[i].Type == TokenLBracket {
		i++
		for {
			var ok bool
			i, ok = p.scanTypeRef(i)
			if !ok {
				return start, false
			}
			if i >= len(p.tokens) {
				return start, false
			}
			if p.tokens[i].Type == TokenComma {
				i++
				continue
			}
			if p.tokens[i].Type == TokenRBracket {
				i++
				break
			}
			return start, false
		}
	}
	if i < len(p.tokens) && p.tokens[i].Type == TokenArrow {
		return p.scanTypeRef(i + 1)
	}
	return i, true
}

func (p *Parser) scanSimpleTypeRef(start int) (int, bool) {
	if start >= len(p.tokens) || p.tokens[start].Type != TokenIdentifier {
		return start, false
	}
	i := start + 1
	if i < len(p.tokens) && p.tokens[i].Type == TokenLBracket {
		i++
		for {
			var ok bool
			i, ok = p.scanTypeRef(i)
			if !ok || i >= len(p.tokens) {
				return start, false
			}
			if p.tokens[i].Type == TokenComma {
				i++
				continue
			}
			if p.tokens[i].Type == TokenRBracket {
				i++
				break
			}
			return start, false
		}
	}
	return i, true
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
