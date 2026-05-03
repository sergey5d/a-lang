package typed

import (
	"fmt"

	"a-lang/parser"
)

// stmtBuilder dispatches parser statements to their dedicated typed builders.
type stmtBuilder struct {
	bindings         Builder[*parser.ValStmt, Stmt]
	unwraps          Builder[*parser.UnwrapStmt, Stmt]
	unwrapBlocks     Builder[*parser.UnwrapBlockStmt, Stmt]
	guards           Builder[*parser.GuardStmt, Stmt]
	guardBlocks      Builder[*parser.GuardBlockStmt, Stmt]
	assignments      Builder[*parser.AssignmentStmt, Stmt]
	multiAssignments Builder[*parser.MultiAssignmentStmt, Stmt]
	ifs              Builder[*parser.IfStmt, Stmt]
	loops            Builder[*parser.LoopStmt, Stmt]
	fors             Builder[*parser.ForStmt, Stmt]
	returns          Builder[*parser.ReturnStmt, Stmt]
	breaks           Builder[*parser.BreakStmt, Stmt]
	exprs            Builder[*parser.ExprStmt, Stmt]
}

// Build converts a parser statement into its typed equivalent.
func (b *stmtBuilder) Build(stmt parser.Statement) (Stmt, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		return b.bindings.Build(s)
	case *parser.UnwrapStmt:
		return b.unwraps.Build(s)
	case *parser.UnwrapBlockStmt:
		return b.unwrapBlocks.Build(s)
	case *parser.GuardStmt:
		return b.guards.Build(s)
	case *parser.GuardBlockStmt:
		return b.guardBlocks.Build(s)
	case *parser.AssignmentStmt:
		return b.assignments.Build(s)
	case *parser.MultiAssignmentStmt:
		return b.multiAssignments.Build(s)
	case *parser.IfStmt:
		return b.ifs.Build(s)
	case *parser.LoopStmt:
		return b.loops.Build(s)
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
