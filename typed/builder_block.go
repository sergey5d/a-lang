package typed

import "a-lang/parser"

// blockBuilder builds typed blocks by delegating each nested statement.
type blockBuilder struct {
	ctx   *buildContext
	stmts Builder[parser.Statement, Stmt]
}

// Build converts a parser block into a typed block.
func (b *blockBuilder) Build(block *parser.BlockStmt) (*BlockStmt, error) {
	if block == nil {
		return nil, nil
	}
	b.ctx.pushScope()
	defer b.ctx.popScope()

	statements := make([]Stmt, len(block.Statements))
	for i, stmt := range block.Statements {
		built, err := b.stmts.Build(stmt)
		if err != nil {
			return nil, err
		}
		statements[i] = built
	}
	return &BlockStmt{Statements: statements, Span: block.Span}, nil
}
