package typed

import "a-lang/parser"

// multiAssignmentStmtBuilder builds typed multi-assignment statements.
type multiAssignmentStmtBuilder struct {
	exprs Builder[parser.Expr, Expr]
}

// Build converts a parser multi-assignment statement into a typed multi-assignment statement.
func (b *multiAssignmentStmtBuilder) Build(stmt *parser.MultiAssignmentStmt) (Stmt, error) {
	targets := make([]Expr, len(stmt.Targets))
	for i, target := range stmt.Targets {
		built, err := b.exprs.Build(target)
		if err != nil {
			return nil, err
		}
		targets[i] = built
	}
	values := make([]Expr, len(stmt.Values))
	for i, value := range stmt.Values {
		built, err := b.exprs.Build(value)
		if err != nil {
			return nil, err
		}
		values[i] = built
	}
	return &MultiAssignmentStmt{Targets: targets, Operator: stmt.Operator, Values: values, Span: stmt.Span}, nil
}
