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
	TokenImpl       TokenType = "IMPL"
	TokenPackage    TokenType = "PACKAGE"
	TokenImport     TokenType = "IMPORT"
	TokenAs         TokenType = "AS"
	TokenInterface  TokenType = "INTERFACE"
	TokenClass      TokenType = "CLASS"
	TokenObject     TokenType = "OBJECT"
	TokenRecord     TokenType = "RECORD"
	TokenEnum       TokenType = "ENUM"
	TokenCase       TokenType = "CASE"
	TokenWith       TokenType = "WITH"
	TokenPrivate    TokenType = "PRIVATE"
	TokenIf         TokenType = "IF"
	TokenMatch      TokenType = "MATCH"
	TokenIs         TokenType = "IS"
	TokenElse       TokenType = "ELSE"
	TokenLoop       TokenType = "LOOP"
	TokenFor        TokenType = "FOR"
	TokenYield      TokenType = "YIELD"
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
	TokenQuestion TokenType = "?"
	TokenEllipsis TokenType = "..."
	TokenAssign TokenType = "="
	TokenFatArrow TokenType = "=>"
	TokenColonAssign TokenType = ":="

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

	TokenUnder TokenType = "_"
)

var keywords = map[string]TokenType{
	"def":        TokenDef,
	"impl":       TokenImpl,
	"package":    TokenPackage,
	"import":     TokenImport,
	"as":         TokenAs,
	"interface":  TokenInterface,
	"class":      TokenClass,
	"object":     TokenObject,
	"record":     TokenRecord,
	"enum":       TokenEnum,
	"case":       TokenCase,
	"with":       TokenWith,
	"private":    TokenPrivate,
	"if":         TokenIf,
	"match":      TokenMatch,
	"is":         TokenIs,
	"else":       TokenElse,
	"loop":       TokenLoop,
	"for":        TokenFor,
	"yield":      TokenYield,
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
