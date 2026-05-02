package parser

import "fmt"

func (p *Parser) parseClass() (*ClassDecl, error) {
	return p.parseClassLike(TokenClass, false, false, "class")
}

func (p *Parser) parseObject() (*ClassDecl, error) {
	return p.parseClassLike(TokenObject, false, false, "object")
}

func (p *Parser) parseRecord() (*ClassDecl, error) {
	return p.parseClassLike(TokenRecord, true, false, "record")
}

func (p *Parser) parsePrivateClass() (*ClassDecl, error) {
	return p.parseClassLike(TokenClass, false, true, "class")
}

func (p *Parser) parsePrivateObject() (*ClassDecl, error) {
	return p.parseClassLike(TokenObject, false, true, "object")
}

func (p *Parser) parsePrivateRecord() (*ClassDecl, error) {
	return p.parseClassLike(TokenRecord, true, true, "record")
}

func (p *Parser) parseEnum() (*ClassDecl, error) {
	return p.parseEnumLike(false)
}

func (p *Parser) parsePrivateEnum() (*ClassDecl, error) {
	return p.parseEnumLike(true)
}

func (p *Parser) parseEnumLike(private bool) (*ClassDecl, error) {
	start, err := p.consume(TokenEnum, "expected 'enum'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected enum name")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParameters()
	if err != nil {
		return nil, err
	}
	decl := &ClassDecl{Name: name.Lexeme, Private: private, Enum: true, TypeParameters: typeParams}
	if _, err := p.consume(TokenLBrace, "expected '{' after enum name"); err != nil {
		return nil, err
	}
	sawNonField := false
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		switch p.peek().Type {
		case TokenIdentifier:
			if sawNonField {
				return nil, fmt.Errorf("enum fields must appear before method or case declarations")
			}
			field, err := p.parseField(false)
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, field)
		case TokenDef, TokenImpl:
			sawNonField = true
			method, err := p.parseMethodLike(false, false)
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, method)
		case TokenOperator:
			return nil, fmt.Errorf("use 'def %s' instead of 'operator %s' in enum declarations", p.peekNextOperatorExample(), p.peekNextOperatorExample())
		case TokenCase:
			sawNonField = true
			enumCase, err := p.parseEnumCase()
			if err != nil {
				return nil, err
			}
			decl.Cases = append(decl.Cases, enumCase)
		default:
			return nil, fmt.Errorf("expected enum member, got %s", p.peek().String())
		}
	}
	end, err := p.consume(TokenRBrace, "expected '}' after enum body")
	if err != nil {
		return nil, err
	}
	decl.Span = mergeSpans(tokenSpan(start), tokenSpan(end))
	return decl, nil
}

func (p *Parser) parseClassLike(kind TokenType, record bool, private bool, noun string) (*ClassDecl, error) {
	start, err := p.consume(kind, "expected '"+noun+"'")
	if err != nil {
		return nil, err
	}
	name, err := p.consume(TokenIdentifier, "expected "+noun+" name")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParameters()
	if err != nil {
		return nil, err
	}
	decl := &ClassDecl{Name: name.Lexeme, Private: private, Object: kind == TokenObject, Record: record, TypeParameters: typeParams}
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
	if _, err := p.consume(TokenLBrace, "expected '{' after "+noun+" name"); err != nil {
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
		case TokenDef, TokenImpl:
			sawMethod = true
			method, err := p.parseMethodLike(private, false)
			if err != nil {
				return nil, err
			}
			decl.Methods = append(decl.Methods, method)
		case TokenOperator:
			return nil, fmt.Errorf("use symbolic 'def' declarations instead of the 'operator' keyword")
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
		if err := p.requireSameLineExpressionStart(operator); err != nil {
			return FieldDecl{}, err
		}
		field.Mutable = operator.Type == TokenColonAssign
		if p.match(TokenQuestion) {
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

func (p *Parser) parseMethodLike(private bool, allowShortApply bool) (*MethodDecl, error) {
	return p.parseMethod(private, allowShortApply)
}

func (p *Parser) parseMethod(private bool, allowShortApply bool) (*MethodDecl, error) {
	impl := p.match(TokenImpl)
	start := p.peek()
	if _, err := p.consume(TokenDef, "expected 'def'"); err != nil {
		if impl {
			return nil, err
		}
		return nil, err
	}
	if impl {
		start = p.tokens[p.pos-2]
	} else {
		start = p.previous()
	}
	nameLexeme, isOperator, err := p.parseDeclaredMethodName(allowShortApply, "expected method name")
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
	constructor := nameLexeme == "this"
	var returnType *TypeRef
	if !constructor && !p.check(TokenAssign) && !(p.check(TokenLBrace) && !p.typeRefFollowedBy(TokenAssign)) {
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
		Name:           nameLexeme,
		TypeParameters: typeParams,
		Parameters:     params,
		ReturnType:     returnType,
		Body:           body,
		Impl:           impl,
		Operator:       isOperator,
		Private:        private,
		Constructor:    constructor,
		Span:           mergeSpans(tokenSpan(start), body.Span),
	}, nil
}

func (p *Parser) parseOperatorName() (string, Span, error) {
	token := p.advance()
	switch token.Type {
	case TokenPlus, TokenMinus, TokenStar, TokenSlash, TokenPercent:
		return token.Lexeme, tokenSpan(token), nil
	case TokenColonPlus:
		return ":+", tokenSpan(token), nil
	case TokenColonMinus:
		return ":-", tokenSpan(token), nil
	case TokenPlusPlus:
		return "++", tokenSpan(token), nil
	case TokenMinusMinus:
		return "--", tokenSpan(token), nil
	case TokenPipe:
		return "|", tokenSpan(token), nil
	case TokenAmp:
		return "&", tokenSpan(token), nil
	case TokenGTGT:
		return ">>", tokenSpan(token), nil
	case TokenLTLT:
		return "<<", tokenSpan(token), nil
	case TokenTilde:
		return "~", tokenSpan(token), nil
	case TokenColonColon:
		return "::", tokenSpan(token), nil
	case TokenLBracket:
		end, err := p.consume(TokenRBracket, "expected ']' after operator '['")
		if err != nil {
			return "", Span{}, err
		}
		return "[]", mergeSpans(tokenSpan(token), tokenSpan(end)), nil
	default:
		return "", Span{}, fmt.Errorf("expected operator symbol, got %s", token.String())
	}
}

func (p *Parser) parseDeclaredMethodName(allowShortApply bool, identifierMessage string) (string, bool, error) {
	if allowShortApply && p.check(TokenLParen) {
		return "apply", false, nil
	}
	if p.check(TokenIdentifier) {
		name, err := p.consume(TokenIdentifier, identifierMessage)
		if err != nil {
			return "", false, err
		}
		return name.Lexeme, false, nil
	}
	if isOperatorNameToken(p.peek().Type) {
		name, _, err := p.parseOperatorName()
		if err != nil {
			return "", false, err
		}
		return name, true, nil
	}
	name, err := p.consume(TokenIdentifier, identifierMessage)
	if err != nil {
		return "", false, err
	}
	return name.Lexeme, false, nil
}

func isOperatorNameToken(tokenType TokenType) bool {
	switch tokenType {
	case TokenPlus, TokenMinus, TokenStar, TokenSlash, TokenPercent, TokenColonPlus, TokenColonMinus, TokenPlusPlus, TokenMinusMinus, TokenPipe, TokenAmp, TokenGTGT, TokenLTLT, TokenTilde, TokenColonColon, TokenLBracket:
		return true
	default:
		return false
	}
}

func (p *Parser) peekNextOperatorExample() string {
	if p.pos+1 >= len(p.tokens) {
		return "<op>"
	}
	next := p.tokens[p.pos+1]
	if next.Type == TokenLBracket && p.pos+2 < len(p.tokens) && p.tokens[p.pos+2].Type == TokenRBracket {
		return "[]"
	}
	if next.Lexeme != "" {
		return next.Lexeme
	}
	return "<op>"
}

func (p *Parser) parseEnumCase() (EnumCaseDecl, error) {
	start, err := p.consume(TokenCase, "expected 'case'")
	if err != nil {
		return EnumCaseDecl{}, err
	}
	name, err := p.consume(TokenIdentifier, "expected case name")
	if err != nil {
		return EnumCaseDecl{}, err
	}
	decl := EnumCaseDecl{Name: name.Lexeme, Span: mergeSpans(tokenSpan(start), tokenSpan(name))}
	if !p.match(TokenLBrace) {
		return decl, nil
	}
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenIdentifier) && (p.checkNext(TokenAssign) || p.checkNext(TokenColonAssign)) {
			assignStart := p.advance()
			op := p.advance()
			if op.Type != TokenAssign {
				return EnumCaseDecl{}, fmt.Errorf("enum case field assignments must use '='")
			}
			if err := p.requireSameLineExpressionStart(op); err != nil {
				return EnumCaseDecl{}, err
			}
			value, err := p.parseExpression(0)
			if err != nil {
				return EnumCaseDecl{}, err
			}
			decl.Assignments = append(decl.Assignments, EnumCaseAssignment{
				Name:  assignStart.Lexeme,
				Value: value,
				Span:  mergeSpans(tokenSpan(assignStart), exprSpan(value)),
			})
			continue
		}
		field, err := p.parseField(false)
		if err != nil {
			return EnumCaseDecl{}, err
		}
		decl.Fields = append(decl.Fields, field)
	}
	end, err := p.consume(TokenRBrace, "expected '}' after case body")
	if err != nil {
		return EnumCaseDecl{}, err
	}
	decl.Span = mergeSpans(tokenSpan(start), tokenSpan(end))
	return decl, nil
}

func (p *Parser) parseTopLevelImpl(program *Program) error {
	start, err := p.consume(TokenImpl, "expected 'impl'")
	if err != nil {
		return err
	}
	target, err := p.parseTypeRef()
	if err != nil {
		return err
	}
	var targetClass *ClassDecl
	for i := len(program.Classes) - 1; i >= 0; i-- {
		if program.Classes[i].Name == target.Name {
			targetClass = program.Classes[i]
			break
		}
	}
	if targetClass == nil {
		return fmt.Errorf("unknown impl target '%s'", target.Name)
	}
	var targetCase *EnumCaseDecl
	if p.match(TokenDot) {
		caseName, err := p.consume(TokenIdentifier, "expected enum case name after '.'")
		if err != nil {
			return err
		}
		if !targetClass.Enum {
			return fmt.Errorf("impl target '%s.%s' requires enum '%s'", target.Name, caseName.Lexeme, target.Name)
		}
		for i := range targetClass.Cases {
			if targetClass.Cases[i].Name == caseName.Lexeme {
				targetCase = &targetClass.Cases[i]
				break
			}
		}
		if targetCase == nil {
			return fmt.Errorf("unknown enum case '%s.%s'", target.Name, caseName.Lexeme)
		}
	}
	if _, err := p.consume(TokenLBrace, "expected '{' after impl target"); err != nil {
		return err
	}
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		private := p.match(TokenPrivate)
		if !p.check(TokenDef) && !p.check(TokenImpl) {
			return fmt.Errorf("expected method declaration in impl block, got %s", p.peek().String())
		}
		method, err := p.parseMethodLike(private, false)
		if err != nil {
			return err
		}
		if targetCase != nil {
			targetCase.Methods = append(targetCase.Methods, method)
		} else {
			targetClass.Methods = append(targetClass.Methods, method)
		}
	}
	end, err := p.consume(TokenRBrace, "expected '}' after impl body")
	if err != nil {
		return err
	}
	if targetCase != nil {
		targetCase.Span = mergeSpans(targetCase.Span, tokenSpan(end))
	} else {
		targetClass.Span = mergeSpans(targetClass.Span, tokenSpan(end))
	}
	_ = start
	return nil
}
