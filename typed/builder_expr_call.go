package typed

import "a-lang/parser"

// callExprBuilder classifies parser call expressions into typed call forms.
type callExprBuilder struct {
	ctx   *buildContext
	exprs Builder[parser.Expr, Expr]
	types *typeRefBuilder
}

// Build converts a parser call expression into a typed call expression.
func (b *callExprBuilder) Build(call *parser.CallExpr) (Expr, error) {
	args := make([]Expr, len(call.Args))
	for i, arg := range call.Args {
		built, err := b.exprs.Build(arg)
		if err != nil {
			return nil, err
		}
		args[i] = built
	}

	switch callee := call.Callee.(type) {
	case *parser.Identifier:
		if _, ok := b.ctx.classes[callee.Name]; ok {
			expr := &ConstructorCallExpr{
				baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
				Class:    callee.Name,
				Args:     args,
				Dispatch: DispatchConstruct,
			}
			if symbol, ok := b.ctx.classSymbols[callee.Name]; ok {
				expr.ClassSymbol = &symbol
			}
			expr.Constructor = b.types.resolveConstructorSymbol(callee.Name, args)
			return expr, nil
		}
		if _, ok := b.ctx.functions[callee.Name]; ok {
			expr := &FunctionCallExpr{
				baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
				Name:     callee.Name,
				Args:     args,
			}
			if symbol, ok := b.ctx.functionSymbols[callee.Name]; ok {
				expr.Function = &symbol
			}
			return expr, nil
		}
	case *parser.MemberExpr:
		receiver, err := b.exprs.Build(callee.Receiver)
		if err != nil {
			return nil, err
		}
		method := &MethodCallExpr{
			baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
			Receiver: receiver,
			Method:   callee.Name,
			Args:     args,
		}
		method.Target, method.Dispatch = b.types.resolveMethodTarget(receiver.GetType(), callee.Name, args)
		return method, nil
	}

	calleeExpr, err := b.exprs.Build(call.Callee)
	if err != nil {
		return nil, err
	}
	return &InvokeExpr{
		baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
		Callee:   calleeExpr,
		Args:     args,
	}, nil
}
