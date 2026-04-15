package parser

import "fmt"

type TokenType string

const (
	TokenEOF        TokenType = "EOF"
	TokenIdentifier TokenType = "IDENT"
	TokenInteger    TokenType = "INT"
	TokenString     TokenType = "STRING"

	TokenDef   TokenType = "DEF"
	TokenVal   TokenType = "VAL"
	TokenIf    TokenType = "IF"
	TokenElse  TokenType = "ELSE"
	TokenFor   TokenType = "FOR"
	TokenFrom  TokenType = "FROM"
	TokenDo    TokenType = "DO"
	TokenYield TokenType = "YIELD"
	TokenMatch TokenType = "MATCH"
	TokenRet   TokenType = "RET"
	TokenBreak TokenType = "BREAK"

	TokenLParen   TokenType = "("
	TokenRParen   TokenType = ")"
	TokenLBrace   TokenType = "{"
	TokenRBrace   TokenType = "}"
	TokenLBracket TokenType = "["
	TokenRBracket TokenType = "]"
	TokenComma    TokenType = ","
	TokenColon    TokenType = ":"
	TokenDot      TokenType = "."
	TokenAssign   TokenType = "="

	TokenPlus    TokenType = "+"
	TokenMinus   TokenType = "-"
	TokenStar    TokenType = "*"
	TokenSlash   TokenType = "/"
	TokenPercent TokenType = "%"
	TokenArrow   TokenType = "->"
	TokenEqEq    TokenType = "=="
	TokenBang    TokenType = "!"
	TokenBangEq  TokenType = "!="
	TokenLT      TokenType = "<"
	TokenLTE     TokenType = "<="
	TokenGT      TokenType = ">"
	TokenGTE     TokenType = ">="
	TokenAndAnd  TokenType = "&&"
	TokenOrOr    TokenType = "||"
	TokenRange   TokenType = ".."
	TokenUnder   TokenType = "_"
)

var keywords = map[string]TokenType{
	"def":   TokenDef,
	"val":   TokenVal,
	"if":    TokenIf,
	"else":  TokenElse,
	"for":   TokenFor,
	"from":  TokenFrom,
	"do":    TokenDo,
	"yield": TokenYield,
	"match": TokenMatch,
	"ret":   TokenRet,
	"break": TokenBreak,
}

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q @ %d:%d)", t.Type, t.Lexeme, t.Line, t.Column)
}
