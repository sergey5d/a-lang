package parser

import "fmt"

type Lexer struct {
	input  []rune
	pos    int
	line   int
	column int
}

func Lex(input string) ([]Token, error) {
	lexer := &Lexer{
		input:  []rune(input),
		line:   1,
		column: 1,
	}

	var tokens []Token
	for {
		token, err := lexer.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			return tokens, nil
		}
	}
}

func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	startLine, startColumn := l.line, l.column
	if l.isAtEnd() {
		return Token{Type: TokenEOF, Line: startLine, Column: startColumn}, nil
	}

	ch := l.peek()
	switch ch {
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Lexeme: "(", Line: startLine, Column: startColumn}, nil
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Lexeme: ")", Line: startLine, Column: startColumn}, nil
	case '{':
		l.advance()
		return Token{Type: TokenLBrace, Lexeme: "{", Line: startLine, Column: startColumn}, nil
	case '}':
		l.advance()
		return Token{Type: TokenRBrace, Lexeme: "}", Line: startLine, Column: startColumn}, nil
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Lexeme: "[", Line: startLine, Column: startColumn}, nil
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Lexeme: "]", Line: startLine, Column: startColumn}, nil
	case ',':
		l.advance()
		return Token{Type: TokenComma, Lexeme: ",", Line: startLine, Column: startColumn}, nil
	case ':':
		l.advance()
		return Token{Type: TokenColon, Lexeme: ":", Line: startLine, Column: startColumn}, nil
	case '.':
		l.advance()
		if l.match('.') {
			return Token{Type: TokenRange, Lexeme: "..", Line: startLine, Column: startColumn}, nil
		}
		return Token{Type: TokenDot, Lexeme: ".", Line: startLine, Column: startColumn}, nil
	case '+':
		l.advance()
		return Token{Type: TokenPlus, Lexeme: "+", Line: startLine, Column: startColumn}, nil
	case '-':
		l.advance()
		return Token{Type: TokenMinus, Lexeme: "-", Line: startLine, Column: startColumn}, nil
	case '*':
		l.advance()
		return Token{Type: TokenStar, Lexeme: "*", Line: startLine, Column: startColumn}, nil
	case '/':
		l.advance()
		return Token{Type: TokenSlash, Lexeme: "/", Line: startLine, Column: startColumn}, nil
	case '%':
		l.advance()
		return Token{Type: TokenPercent, Lexeme: "%", Line: startLine, Column: startColumn}, nil
	case '!':
		l.advance()
		if l.match('=') {
			return Token{Type: TokenBangEq, Lexeme: "!=", Line: startLine, Column: startColumn}, nil
		}
		return Token{Type: TokenBang, Lexeme: "!", Line: startLine, Column: startColumn}, nil
	case '_':
		l.advance()
		return Token{Type: TokenUnder, Lexeme: "_", Line: startLine, Column: startColumn}, nil
	case '=':
		l.advance()
		if l.match('=') {
			return Token{Type: TokenEqEq, Lexeme: "==", Line: startLine, Column: startColumn}, nil
		}
		return Token{Type: TokenAssign, Lexeme: "=", Line: startLine, Column: startColumn}, nil
	case '<':
		l.advance()
		if l.match('-') {
			return Token{Type: TokenLeftArrow, Lexeme: "<-", Line: startLine, Column: startColumn}, nil
		}
		if l.match('=') {
			return Token{Type: TokenLTE, Lexeme: "<=", Line: startLine, Column: startColumn}, nil
		}
		return Token{Type: TokenLT, Lexeme: "<", Line: startLine, Column: startColumn}, nil
	case '>':
		l.advance()
		if l.match('=') {
			return Token{Type: TokenGTE, Lexeme: ">=", Line: startLine, Column: startColumn}, nil
		}
		return Token{Type: TokenGT, Lexeme: ">", Line: startLine, Column: startColumn}, nil
	case '&':
		l.advance()
		if l.match('&') {
			return Token{Type: TokenAndAnd, Lexeme: "&&", Line: startLine, Column: startColumn}, nil
		}
		return Token{}, fmt.Errorf("unexpected '&' at %d:%d", startLine, startColumn)
	case '|':
		l.advance()
		if l.match('|') {
			return Token{Type: TokenOrOr, Lexeme: "||", Line: startLine, Column: startColumn}, nil
		}
		return Token{}, fmt.Errorf("unexpected '|' at %d:%d", startLine, startColumn)
	case '"':
		return l.lexString(startLine, startColumn)
	}

	if isDigit(ch) {
		return l.lexNumber(startLine, startColumn), nil
	}
	if isAlpha(ch) {
		return l.lexIdentifier(startLine, startColumn), nil
	}

	return Token{}, fmt.Errorf("unexpected %q at %d:%d", ch, startLine, startColumn)
}

func (l *Lexer) lexString(line, column int) (Token, error) {
	l.advance()
	start := l.pos
	for !l.isAtEnd() && l.peek() != '"' {
		if l.peek() == '\n' {
			return Token{}, fmt.Errorf("unterminated string at %d:%d", line, column)
		}
		l.advance()
	}
	if l.isAtEnd() {
		return Token{}, fmt.Errorf("unterminated string at %d:%d", line, column)
	}

	value := string(l.input[start:l.pos])
	l.advance()
	return Token{Type: TokenString, Lexeme: value, Line: line, Column: column}, nil
}

func (l *Lexer) lexNumber(line, column int) Token {
	start := l.pos
	for !l.isAtEnd() && isDigit(l.peek()) {
		l.advance()
	}
	return Token{Type: TokenInteger, Lexeme: string(l.input[start:l.pos]), Line: line, Column: column}
}

func (l *Lexer) lexIdentifier(line, column int) Token {
	start := l.pos
	for !l.isAtEnd() && isAlphaNumeric(l.peek()) {
		l.advance()
	}
	lexeme := string(l.input[start:l.pos])
	if keyword, ok := keywords[lexeme]; ok {
		return Token{Type: keyword, Lexeme: lexeme, Line: line, Column: column}
	}
	return Token{Type: TokenIdentifier, Lexeme: lexeme, Line: line, Column: column}
}

func (l *Lexer) skipWhitespace() {
	for !l.isAtEnd() {
		switch l.peek() {
		case ' ', '\r', '\t':
			l.advance()
		case '\n':
			l.advance()
		default:
			return
		}
	}
}

func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() || l.input[l.pos] != expected {
		return false
	}
	l.advance()
	return true
}

func (l *Lexer) peek() rune {
	return l.input[l.pos]
}

func (l *Lexer) advance() rune {
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func (l *Lexer) isAtEnd() bool {
	return l.pos >= len(l.input)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isAlphaNumeric(ch rune) bool {
	return isAlpha(ch) || isDigit(ch)
}
