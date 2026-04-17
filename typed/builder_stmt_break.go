package typed

import "a-lang/parser"

// breakStmtBuilder builds typed break statements.
type breakStmtBuilder struct{}

// Build converts a parser break statement into a typed break statement.
func (b *breakStmtBuilder) Build(stmt *parser.BreakStmt) (Stmt, error) {
	return &BreakStmt{Span: stmt.Span}, nil
}
