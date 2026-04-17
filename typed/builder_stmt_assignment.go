package typed

import "a-lang/parser"

// assignmentStmtBuilder builds typed assignment statements.
type assignmentStmtBuilder struct {
	exprs Builder[parser.Expr, Expr]
}

// Build converts a parser assignment statement into a typed assignment statement.
func (b *assignmentStmtBuilder) Build(stmt *parser.AssignmentStmt) (Stmt, error) {
	target, err := b.exprs.Build(stmt.Target)
	if err != nil {
		return nil, err
	}
	value, err := b.exprs.Build(stmt.Value)
	if err != nil {
		return nil, err
	}
	return &AssignmentStmt{Target: target, Operator: stmt.Operator, Value: value, Span: stmt.Span}, nil
}
