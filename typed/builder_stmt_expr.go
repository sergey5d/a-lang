package typed

import "a-lang/parser"

// exprStmtBuilder builds typed expression statements.
type exprStmtBuilder struct {
	exprs Builder[parser.Expr, Expr]
}

// Build converts a parser expression statement into a typed expression statement.
func (b *exprStmtBuilder) Build(stmt *parser.ExprStmt) (Stmt, error) {
	expr, err := b.exprs.Build(stmt.Expr)
	if err != nil {
		return nil, err
	}
	return &ExprStmt{Expr: expr, Span: stmt.Span}, nil
}
