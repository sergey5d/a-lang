package parser

import (
	"fmt"
)

type TokenType string

const (
	TokenEOF        TokenType = "EOF"
	TokenIdentifier TokenType = "IDENT"
	TokenInteger    TokenType = "INT"
	TokenFloat      TokenType = "FLOAT"
	TokenRune       TokenType = "RUNE"
	TokenString     TokenType = "STRING"
	TokenBool       TokenType = "BOOL"

	TokenDef        TokenType = "DEF"
	TokenInterface  TokenType = "INTERFACE"
	TokenClass      TokenType = "CLASS"
	TokenImplements TokenType = "IMPLEMENTS"
	TokenPrivate    TokenType = "PRIVATE"
	TokenLet        TokenType = "LET"
	TokenVar        TokenType = "VAR"
	TokenIf         TokenType = "IF"
	TokenElse       TokenType = "ELSE"
	TokenFor        TokenType = "FOR"
	TokenYield      TokenType = "YIELD"
	TokenMatch      TokenType = "MATCH"
	TokenReturn     TokenType = "RETURN"
	TokenBreak      TokenType = "BREAK"

	TokenLParen   TokenType = "("
	TokenRParen   TokenType = ")"
	TokenLBrace   TokenType = "{"
	TokenRBrace   TokenType = "}"
	TokenLBracket TokenType = "["
	TokenRBracket TokenType = "]"

	TokenComma  TokenType = ","
	TokenColon  TokenType = ":"
	TokenDot    TokenType = "."
	TokenAssign TokenType = "="

	TokenPlus    TokenType = "+"
	TokenMinus   TokenType = "-"
	TokenStar    TokenType = "*"
	TokenSlash   TokenType = "/"
	TokenPercent TokenType = "%"

	TokenPlusEq    TokenType = "+="
	TokenMinusEq   TokenType = "-="
	TokenStarEq    TokenType = "*="
	TokenSlashEq   TokenType = "/="
	TokenPercentEq TokenType = "%="

	TokenArrow     TokenType = "->"
	TokenLeftArrow TokenType = "<-"
	TokenEqEq      TokenType = "=="
	TokenBang      TokenType = "!"
	TokenBangEq    TokenType = "!="
	TokenLT        TokenType = "<"
	TokenLTE       TokenType = "<="
	TokenGT        TokenType = ">"
	TokenGTE       TokenType = ">="

	TokenAndAnd TokenType = "&&"
	TokenOrOr   TokenType = "||"

	TokenRange TokenType = ".."
	TokenUnder TokenType = "_"
)

var keywords = map[string]TokenType{
	"def":        TokenDef,
	"interface":  TokenInterface,
	"class":      TokenClass,
	"implements": TokenImplements,
	"private":    TokenPrivate,
	"let":        TokenLet,
	"var":        TokenVar,
	"if":         TokenIf,
	"else":       TokenElse,
	"for":        TokenFor,
	"yield":      TokenYield,
	"match":      TokenMatch,
	"return":     TokenReturn,
	"break":      TokenBreak,
	"true":       TokenBool,
	"false":      TokenBool,
}

type Token struct {
	Type      TokenType
	Lexeme    string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q @ %d:%d)", t.Type, t.Lexeme, t.Line, t.Column)
}
