package typed

import (
	"fmt"

	"a-lang/parser"
)

// stmtBuilder dispatches parser statements to their dedicated typed builders.
type stmtBuilder struct {
	bindings    Builder[*parser.ValStmt, Stmt]
	assignments Builder[*parser.AssignmentStmt, Stmt]
	ifs         Builder[*parser.IfStmt, Stmt]
	fors        Builder[*parser.ForStmt, Stmt]
	returns     Builder[*parser.ReturnStmt, Stmt]
	breaks      Builder[*parser.BreakStmt, Stmt]
	exprs       Builder[*parser.ExprStmt, Stmt]
}

// Build converts a parser statement into its typed equivalent.
func (b *stmtBuilder) Build(stmt parser.Statement) (Stmt, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		return b.bindings.Build(s)
	case *parser.AssignmentStmt:
		return b.assignments.Build(s)
	case *parser.IfStmt:
		return b.ifs.Build(s)
	case *parser.ForStmt:
		return b.fors.Build(s)
	case *parser.ReturnStmt:
		return b.returns.Build(s)
	case *parser.BreakStmt:
		return b.breaks.Build(s)
	case *parser.ExprStmt:
		return b.exprs.Build(s)
	default:
		return nil, fmt.Errorf("unsupported statement type %T", stmt)
	}
}
