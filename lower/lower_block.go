package lower

import "a-lang/typed"

func (l *Lowerer) lowerBlock(block *typed.BlockStmt) ([]Stmt, error) {
	if block == nil {
		return nil, nil
	}
	return l.lowerStmtBlock(block.Statements)
}

func (l *Lowerer) lowerStmtBlock(stmts []typed.Stmt) ([]Stmt, error) {
	var out []Stmt
	for _, stmt := range stmts {
		lowered, err := l.lowerStmt(stmt)
		if err != nil {
			return nil, err
		}
		out = append(out, lowered...)
	}
	return out, nil
}
