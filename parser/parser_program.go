package parser

import "fmt"

func (p *Parser) parseProgram() (*Program, error) {
	program := &Program{}
	if p.match(TokenPackage) {
		keyword := p.previous()
		name, span, err := p.parseModulePath("expected package name")
		if err != nil {
			return nil, err
		}
		program.PackageName = name
		program.PackageSpan = mergeSpans(tokenSpan(keyword), span)
	}
	for p.match(TokenImport) {
		keyword := p.previous()
		path, span, err := p.parseModulePath("expected import path")
		if err != nil {
			return nil, err
		}
		program.Imports = append(program.Imports, ImportDecl{
			Path: path,
			Span: mergeSpans(tokenSpan(keyword), span),
		})
	}
	for !p.isAtEnd() {
		switch p.peek().Type {
		case TokenPackage:
			return nil, fmt.Errorf("'package' must appear before declarations")
		case TokenImport:
			return nil, fmt.Errorf("'import' must appear before declarations")
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
		case TokenObject:
			decl, err := p.parseObject()
			if err != nil {
				return nil, err
			}
			program.Classes = append(program.Classes, decl)
		case TokenRecord:
			decl, err := p.parseRecord()
			if err != nil {
				return nil, err
			}
			program.Classes = append(program.Classes, decl)
		case TokenEnum:
			decl, err := p.parseEnum()
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
	if program.PackageName != "" {
		spans = append(spans, program.PackageSpan)
	}
	for _, imp := range program.Imports {
		spans = append(spans, imp.Span)
	}
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

func (p *Parser) parseModulePath(message string) (string, Span, error) {
	start, err := p.consume(TokenIdentifier, message)
	if err != nil {
		return "", Span{}, err
	}
	path := start.Lexeme
	span := tokenSpan(start)
	for p.match(TokenSlash) {
		next, err := p.consume(TokenIdentifier, "expected path segment after '/'")
		if err != nil {
			return "", Span{}, err
		}
		path += "/" + next.Lexeme
		span = mergeSpans(span, tokenSpan(next))
	}
	return path, span, nil
}
