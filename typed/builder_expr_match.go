package typed

import "a-lang/parser"

func (b *exprBuilder) buildMatchExpr(expr parser.Expr, match *parser.MatchExpr) (Expr, error) {
	value, err := b.Build(match.Value)
	if err != nil {
		return nil, err
	}
	cases := make([]MatchCase, len(match.Cases))
	for i, matchCase := range match.Cases {
		var body *BlockStmt
		var guard Expr
		var caseExpr Expr
		if matchCase.Guard != nil {
			guard, err = b.Build(matchCase.Guard)
			if err != nil {
				return nil, err
			}
		}
		if matchCase.Body != nil {
			body, err = b.blocks.Build(matchCase.Body)
			if err != nil {
				return nil, err
			}
		}
		if matchCase.Expr != nil {
			caseExpr, err = b.Build(matchCase.Expr)
			if err != nil {
				return nil, err
			}
		}
		cases[i] = MatchCase{
			Pattern: matchCase.Pattern,
			Guard:   guard,
			Body:    body,
			Expr:    caseExpr,
		}
	}
	return &MatchExpr{baseExpr: b.base(expr), Value: value, Cases: cases}, nil
}
