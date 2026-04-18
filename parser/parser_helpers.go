package parser

import "fmt"

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

func (p *Parser) beginScope() {
	p.scopes = append(p.scopes, map[string]struct{}{})
}

func (p *Parser) endScope() {
	if len(p.scopes) == 0 {
		return
	}
	p.scopes = p.scopes[:len(p.scopes)-1]
}

func (p *Parser) declare(name string) {
	if len(p.scopes) == 0 {
		p.scopes = []map[string]struct{}{{}}
	}
	p.scopes[len(p.scopes)-1][name] = struct{}{}
}

func (p *Parser) isDeclared(name string) bool {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if _, ok := p.scopes[i][name]; ok {
			return true
		}
	}
	return false
}
