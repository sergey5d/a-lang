package typed

import "a-lang/parser"

type whileStmtBuilder struct {
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
}

func (b *whileStmtBuilder) Build(stmt *parser.WhileStmt) (Stmt, error) {
	condition, err := b.exprs.Build(stmt.Condition)
	if err != nil {
		return nil, err
	}
	body, err := b.blocks.Build(stmt.Body)
	if err != nil {
		return nil, err
	}
	return &WhileStmt{Condition: condition, Body: body, Span: stmt.Span}, nil
}
