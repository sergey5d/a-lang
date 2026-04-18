package lower

import (
	"fmt"

	"a-lang/typecheck"
	"a-lang/typed"
)

// Lowerer converts typed AST nodes into the lowered IR.
type Lowerer struct {
	tempID int
}

// ProgramFromTyped lowers a typed program into backend-friendly IR.
func ProgramFromTyped(program *typed.Program) (*Program, error) {
	l := &Lowerer{}
	return l.lowerProgram(program)
}

func (l *Lowerer) nextTemp(prefix string) string {
	l.tempID++
	return fmt.Sprintf("__%s%d", prefix, l.tempID)
}

func unknownType() *typecheck.Type {
	return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
}
