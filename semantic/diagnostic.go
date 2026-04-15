package semantic

import (
	"fmt"

	"a-lang/parser"
)

type Diagnostic struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Span    parser.Span `json:"span"`
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", d.Code, d.Span.Start.Line, d.Span.Start.Column, d.Message)
}
