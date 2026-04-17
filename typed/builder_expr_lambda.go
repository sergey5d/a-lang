package typed

import "a-lang/parser"

// lambdaExprBuilder builds typed lambda expressions.
type lambdaExprBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser lambda expression into a typed lambda expression.
func (b *lambdaExprBuilder) Build(expr *parser.LambdaExpr) (Expr, error) {
	b.ctx.pushScope()
	defer b.ctx.popScope()

	params := make([]LambdaParameter, len(expr.Parameters))
	for i, param := range expr.Parameters {
		symbol := b.ctx.newSymbol(SymbolParameter, param.Name, "", param.Span)
		params[i] = LambdaParameter{
			Name:   param.Name,
			Type:   b.types.BuildType(param.Type),
			Symbol: symbol,
			Span:   param.Span,
		}
		b.ctx.defineSymbol(symbol)
	}

	var body Expr
	var err error
	if expr.Body != nil {
		body, err = b.exprs.Build(expr.Body)
		if err != nil {
			return nil, err
		}
	}
	blockBody, err := b.blocks.Build(expr.BlockBody)
	if err != nil {
		return nil, err
	}

	return &LambdaExpr{
		baseExpr:   baseExpr{Type: b.ctx.exprTypes[expr], Span: expr.Span},
		Parameters: params,
		Body:       body,
		BlockBody:  blockBody,
	}, nil
}
