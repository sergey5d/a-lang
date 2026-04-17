package typed

import "a-lang/parser"

// ifStmtBuilder builds typed if statements.
type ifStmtBuilder struct {
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
}

// Build converts a parser if statement into a typed if statement.
func (b *ifStmtBuilder) Build(stmt *parser.IfStmt) (Stmt, error) {
	cond, err := b.exprs.Build(stmt.Condition)
	if err != nil {
		return nil, err
	}
	thenBlock, err := b.blocks.Build(stmt.Then)
	if err != nil {
		return nil, err
	}
	var elseIf *IfStmt
	if stmt.ElseIf != nil {
		built, err := b.Build(stmt.ElseIf)
		if err != nil {
			return nil, err
		}
		elseIf = built.(*IfStmt)
	}
	elseBlock, err := b.blocks.Build(stmt.Else)
	if err != nil {
		return nil, err
	}
	return &IfStmt{Condition: cond, Then: thenBlock, ElseIf: elseIf, Else: elseBlock, Span: stmt.Span}, nil
}
