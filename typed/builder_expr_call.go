package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// callExprBuilder classifies parser call expressions into typed call forms.
type callExprBuilder struct {
	ctx   *buildContext
	exprs Builder[parser.Expr, Expr]
	types *typeRefBuilder
}

// Build converts a parser call expression into a typed call expression.
func (b *callExprBuilder) Build(call *parser.CallExpr) (Expr, error) {
	buildArgs := func(args []parser.CallArg) ([]Expr, error) {
		out := make([]Expr, len(args))
		for i, arg := range args {
			built, err := b.exprs.Build(arg.Value)
			if err != nil {
				return nil, err
			}
			out[i] = built
		}
		return out, nil
	}

	args := call.Args

	switch callee := call.Callee.(type) {
	case *parser.Identifier:
		if fn, ok := b.ctx.functions[callee.Name]; ok {
			if ordered, ok := reorderParserCallArgs(fn.Parameters, call.Args); ok {
				args = ordered
			}
		}
		if class, ok := b.ctx.classes[callee.Name]; ok {
			for _, method := range class.Methods {
				if !method.Constructor {
					continue
				}
				if ordered, ok := reorderParserCallArgs(method.Parameters, call.Args); ok {
					args = ordered
					break
				}
			}
		}
	case *parser.MemberExpr:
		receiverType := b.ctx.exprTypes[callee.Receiver]
		if receiverType != nil && receiverType.Kind == typecheck.TypeClass {
			if class, ok := b.ctx.classes[receiverType.Name]; ok {
				for _, method := range class.Methods {
					if method.Name != callee.Name {
						continue
					}
					if ordered, ok := reorderParserCallArgs(method.Parameters, call.Args); ok {
						args = ordered
						break
					}
				}
			}
		}
	}

	builtArgs, err := buildArgs(args)
	if err != nil {
		return nil, err
	}

	switch callee := call.Callee.(type) {
	case *parser.Identifier:
		if _, ok := b.ctx.classes[callee.Name]; ok {
			expr := &ConstructorCallExpr{
				baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
				Class:    callee.Name,
				Args:     builtArgs,
				Dispatch: DispatchConstruct,
			}
			if symbol, ok := b.ctx.classSymbols[callee.Name]; ok {
				expr.ClassSymbol = &symbol
			}
			expr.Constructor = b.types.resolveConstructorSymbol(callee.Name, builtArgs)
			return expr, nil
		}
		if _, ok := b.ctx.functions[callee.Name]; ok {
			expr := &FunctionCallExpr{
				baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
				Name:     callee.Name,
				Args:     builtArgs,
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
			Args:     builtArgs,
		}
		method.Target, method.Dispatch = b.types.resolveMethodTarget(receiver.GetType(), callee.Name, builtArgs)
		return method, nil
	}

	calleeExpr, err := b.exprs.Build(call.Callee)
	if err != nil {
		return nil, err
	}
	return &InvokeExpr{
		baseExpr: baseExpr{Type: b.ctx.exprTypes[call], Span: call.Span},
		Callee:   calleeExpr,
		Args:     builtArgs,
	}, nil
}

func reorderParserCallArgs(params []parser.Parameter, args []parser.CallArg) ([]parser.CallArg, bool) {
	hasNamed := false
	for _, arg := range args {
		if arg.Name != "" {
			hasNamed = true
			break
		}
	}
	if !hasNamed {
		return args, true
	}
	if len(params) > 0 && params[len(params)-1].Variadic {
		return nil, false
	}
	ordered := make([]parser.CallArg, len(params))
	filled := make([]bool, len(params))
	pos := 0
	seenNamed := false
	for _, arg := range args {
		if arg.Name == "" {
			if seenNamed || pos >= len(params) {
				return nil, false
			}
			ordered[pos] = parser.CallArg{Value: arg.Value, Span: arg.Span}
			filled[pos] = true
			pos++
			continue
		}
		seenNamed = true
		paramIndex := -1
		for i, param := range params {
			if param.Name == arg.Name {
				paramIndex = i
				break
			}
		}
		if paramIndex < 0 || filled[paramIndex] {
			return nil, false
		}
		ordered[paramIndex] = parser.CallArg{Value: arg.Value, Span: arg.Span}
		filled[paramIndex] = true
	}
	for _, ok := range filled {
		if !ok {
			return nil, false
		}
	}
	return ordered, true
}
