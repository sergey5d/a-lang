package typed

import "a-lang/parser"

// returnStmtBuilder builds typed return statements.
type returnStmtBuilder struct {
	exprs Builder[parser.Expr, Expr]
}

// Build converts a parser return statement into a typed return statement.
func (b *returnStmtBuilder) Build(stmt *parser.ReturnStmt) (Stmt, error) {
	value, err := b.exprs.Build(stmt.Value)
	if err != nil {
		return nil, err
	}
	return &ReturnStmt{Value: value, Span: stmt.Span}, nil
}
