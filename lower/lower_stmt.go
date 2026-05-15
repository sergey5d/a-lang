package lower

import (
	"fmt"

	"a-lang/typecheck"
	"a-lang/typed"
)

func unsupportedTopLevelErr(stmt typed.Stmt) error {
	return fmt.Errorf("unsupported top-level statement %T during lowering", stmt)
}

func unsupportedStmtErr(stmt typed.Stmt) error {
	return fmt.Errorf("unsupported statement %T during lowering", stmt)
}

func (l *Lowerer) lowerStmt(stmt typed.Stmt) ([]Stmt, error) {
	switch s := stmt.(type) {
	case *typed.BindingStmt:
		return l.lowerBindingStmt(s)
	case *typed.UnwrapStmt:
		return nil, unsupportedStmtErr(stmt)
	case *typed.GuardStmt:
		return l.lowerGuardStmt(s)
	case *typed.GuardBlockStmt:
		return l.lowerGuardBlockStmt(s)
	case *typed.AssignmentStmt:
		return l.lowerAssignmentStmt(s)
	case *typed.IfStmt:
		return l.lowerIfStmt(s)
	case *typed.WhileStmt:
		return l.lowerWhileStmt(s)
	case *typed.ForStmt:
		return l.lowerForStmt(s)
	case *typed.ReturnStmt:
		return l.lowerReturnStmt(s)
	case *typed.BreakStmt:
		return l.lowerBreakStmt(s)
	case *typed.ExprStmt:
		return l.lowerExprStmt(s)
	default:
		return nil, unsupportedStmtErr(stmt)
	}
}

func (l *Lowerer) lowerBindingStmt(stmt *typed.BindingStmt) ([]Stmt, error) {
	var out []Stmt
	for _, binding := range stmt.Bindings {
		var init Expr
		var err error
		if binding.Init != nil {
			init, err = l.lowerExpr(binding.Init)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, &VarDecl{
			Name:    binding.Name,
			Mutable: binding.Mode == typed.BindingMutable,
			Type:    binding.Type,
			Init:    init,
		})
	}
	return out, nil
}

func (l *Lowerer) lowerAssignmentStmt(stmt *typed.AssignmentStmt) ([]Stmt, error) {
	target, err := l.lowerExpr(stmt.Target)
	if err != nil {
		return nil, err
	}
	value, err := l.lowerExpr(stmt.Value)
	if err != nil {
		return nil, err
	}
	return []Stmt{&Assign{Target: target, Operator: stmt.Operator, Value: value}}, nil
}

func (l *Lowerer) lowerGuardStmt(stmt *typed.GuardStmt) ([]Stmt, error) {
	return l.lowerGuardClauses([]*typed.UnwrapStmt{{
		Bindings: stmt.Bindings,
		Value:    stmt.Value,
		Span:     stmt.Span,
	}}, stmt.Fallback)
}

func (l *Lowerer) lowerGuardBlockStmt(stmt *typed.GuardBlockStmt) ([]Stmt, error) {
	return l.lowerGuardClauses(stmt.Clauses, stmt.Fallback)
}

func (l *Lowerer) lowerGuardClauses(clauses []*typed.UnwrapStmt, fallback *typed.BlockStmt) ([]Stmt, error) {
	fallbackPrefix, fallbackValue, err := l.lowerValueBlock(fallback, "unwrap else block must end with a value-producing statement")
	if err != nil {
		return nil, err
	}

	var out []Stmt
	for _, clause := range clauses {
		clauseStmts, err := l.lowerGuardClause(clause, fallbackPrefix, fallbackValue)
		if err != nil {
			return nil, err
		}
		out = append(out, clauseStmts...)
	}
	return out, nil
}

func (l *Lowerer) lowerGuardClause(clause *typed.UnwrapStmt, fallbackPrefix []Stmt, fallbackValue Expr) ([]Stmt, error) {
	source, err := l.lowerExpr(clause.Value)
	if err != nil {
		return nil, err
	}
	sourceType := clause.Value.GetType()
	if sourceType == nil {
		return nil, fmt.Errorf("unwrap binding requires resolved source type")
	}

	sourceName := l.nextTemp("unwrap")
	sourceRef := &VarRef{Name: sourceName, Type: sourceType}
	stmts := []Stmt{&VarDecl{
		Name:    sourceName,
		Mutable: false,
		Type:    sourceType,
		Init:    source,
	}}

	condition, unwrapped, unwrappedType, err := l.lowerUnwrapAccess(sourceRef, sourceType)
	if err != nil {
		return nil, err
	}

	bindPrefix, bindThen, err := l.lowerGuardBindings(clause.Bindings, unwrapped, unwrappedType)
	if err != nil {
		return nil, err
	}
	stmts = append(stmts, bindPrefix...)

	elseBranch := append([]Stmt{}, fallbackPrefix...)
	elseBranch = append(elseBranch, &Return{Value: fallbackValue})

	stmts = append(stmts, &If{
		Condition: condition,
		Then:      bindThen,
		Else:      elseBranch,
	})
	return stmts, nil
}

func (l *Lowerer) lowerUnwrapAccess(sourceRef *VarRef, sourceType *typecheck.Type) (Expr, Expr, *typecheck.Type, error) {
	switch sourceType.Name {
	case "Option":
		return &MethodCall{
				Receiver: sourceRef,
				Method:   "isSet",
				Type:     &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
			}, &MethodCall{
				Receiver: sourceRef,
				Method:   "expect",
				Type:     unwrapElemType(sourceType),
			}, unwrapElemType(sourceType), nil
	case "Result", "Either":
		return &Unary{
				Operator: "!",
				Right: &MethodCall{
					Receiver: sourceRef,
					Method:   "isFailure",
					Type:     &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
				},
				Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
			}, &MethodCall{
				Receiver: sourceRef,
				Method:   "unwrap",
				Type:     unwrapElemType(sourceType),
			}, unwrapElemType(sourceType), nil
	default:
		return nil, nil, nil, fmt.Errorf("unwrap binding requires Option[T], Result[T, E], or Either[L, R], got %s", sourceType.String())
	}
}

func (l *Lowerer) lowerGuardBindings(bindings []typed.BindingDecl, unwrapped Expr, unwrappedType *typecheck.Type) ([]Stmt, []Stmt, error) {
	if len(bindings) == 0 {
		return nil, nil, nil
	}
	if len(bindings) == 1 {
		binding := bindings[0]
		if binding.Name == "_" {
			return nil, nil, nil
		}
		bindingType := binding.Type
		if bindingType == nil || bindingType.Kind == typecheck.TypeUnknown {
			bindingType = unwrappedType
		}
		return []Stmt{&VarDecl{
				Name:    binding.Name,
				Mutable: false,
				Type:    bindingType,
			}}, []Stmt{&Assign{
				Target:   &VarRef{Name: binding.Name, Type: bindingType},
				Operator: ":=",
				Value:    unwrapped,
			}}, nil
	}

	if unwrappedType == nil || unwrappedType.Kind != typecheck.TypeTuple {
		typeLabel := "<unknown>"
		if unwrappedType != nil {
			typeLabel = unwrappedType.String()
		}
		return nil, nil, fmt.Errorf("lowering only supports tuple destructuring for unwrap bindings, got %s", typeLabel)
	}
	if len(unwrappedType.Args) != len(bindings) {
		return nil, nil, fmt.Errorf("unwrap binding expects %d values, got %d", len(bindings), len(unwrappedType.Args))
	}

	tempName := l.nextTemp("bound")
	tempRef := &VarRef{Name: tempName, Type: unwrappedType}
	prefix := []Stmt{&VarDecl{Name: tempName, Mutable: false, Type: unwrappedType}}
	then := []Stmt{&Assign{Target: tempRef, Operator: ":=", Value: unwrapped}}
	for i, binding := range bindings {
		if binding.Name == "_" {
			continue
		}
		fieldName := fmt.Sprintf("_%d", i+1)
		bindingType := binding.Type
		if bindingType == nil || bindingType.Kind == typecheck.TypeUnknown {
			bindingType = unwrappedType.Args[i]
		}
		prefix = append(prefix, &VarDecl{Name: binding.Name, Mutable: false, Type: bindingType})
		then = append(then, &Assign{
			Target:   &VarRef{Name: binding.Name, Type: bindingType},
			Operator: ":=",
			Value: &FieldGet{
				Receiver: tempRef,
				Name:     fieldName,
				Type:     bindingType,
			},
		})
	}
	return prefix, then, nil
}

func (l *Lowerer) lowerIfStmt(stmt *typed.IfStmt) ([]Stmt, error) {
	cond, err := l.lowerExpr(stmt.Condition)
	if err != nil {
		return nil, err
	}
	thenBlock, err := l.lowerBlock(stmt.Then)
	if err != nil {
		return nil, err
	}
	var elseBlock []Stmt
	if stmt.ElseIf != nil {
		branch, err := l.lowerStmt(stmt.ElseIf)
		if err != nil {
			return nil, err
		}
		elseBlock = branch
	} else if stmt.Else != nil {
		elseBlock, err = l.lowerBlock(stmt.Else)
		if err != nil {
			return nil, err
		}
	}
	return []Stmt{&If{Condition: cond, Then: thenBlock, Else: elseBlock}}, nil
}

func (l *Lowerer) lowerReturnStmt(stmt *typed.ReturnStmt) ([]Stmt, error) {
	value, err := l.lowerExpr(stmt.Value)
	if err != nil {
		return nil, err
	}
	return []Stmt{&Return{Value: value}}, nil
}

func (l *Lowerer) lowerBreakStmt(_ *typed.BreakStmt) ([]Stmt, error) {
	return []Stmt{&Break{}}, nil
}

func (l *Lowerer) lowerExprStmt(stmt *typed.ExprStmt) ([]Stmt, error) {
	expr, err := l.lowerExpr(stmt.Expr)
	if err != nil {
		return nil, err
	}
	return []Stmt{&ExprStmt{Expr: expr}}, nil
}

func (l *Lowerer) lowerWhileStmt(stmt *typed.WhileStmt) ([]Stmt, error) {
	cond, err := l.lowerExpr(stmt.Condition)
	if err != nil {
		return nil, err
	}
	body, err := l.lowerBlock(stmt.Body)
	if err != nil {
		return nil, err
	}
	return []Stmt{&Loop{Body: []Stmt{
		&If{
			Condition: cond,
			Then:      body,
			Else:      []Stmt{&Break{}},
		},
	}}}, nil
}

func (l *Lowerer) lowerForStmt(stmt *typed.ForStmt) ([]Stmt, error) {
	body, err := l.lowerBlock(stmt.Body)
	if err != nil {
		return nil, err
	}
	if stmt.YieldBody == nil {
		return l.lowerForBindings(stmt.Bindings, body)
	}

	yieldPrefix, yieldExpr, yieldType, err := l.lowerYieldBody(stmt.YieldBody)
	if err != nil {
		return nil, err
	}
	resultType := &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "List", Args: []*typecheck.Type{yieldType}}
	resultName := l.nextTemp("yield")
	resultRef := &VarRef{Name: resultName, Type: resultType}

	body = append(body, yieldPrefix...)
	body = append(body, &Assign{
		Target:   resultRef,
		Operator: ":=",
		Value: &BuiltinCall{
			Name: "append",
			Args: []Expr{resultRef, yieldExpr},
			Type: resultType,
		},
	})

	stmts := []Stmt{&VarDecl{
		Name:    resultName,
		Mutable: true,
		Type:    resultType,
		Init:    &ListLiteral{Elements: nil, Type: resultType},
	}}
	loops, err := l.lowerForBindings(stmt.Bindings, body)
	if err != nil {
		return nil, err
	}
	stmts = append(stmts, loops...)
	return stmts, nil
}

func (l *Lowerer) lowerForBindings(bindings []typed.ForBinding, body []Stmt) ([]Stmt, error) {
	out := body
	for i := len(bindings) - 1; i >= 0; i-- {
		if len(bindings[i].Values) > 0 {
			if len(bindings[i].Bindings) != 1 {
				return nil, fmt.Errorf("lowering only supports single local for bindings, got %d", len(bindings[i].Bindings))
			}
			binding := bindings[i].Bindings[0]
			var init Expr
			var err error
			if len(bindings[i].Values) > 0 && bindings[i].Values[0] != nil {
				init, err = l.lowerExpr(bindings[i].Values[0])
				if err != nil {
					return nil, err
				}
			}
			out = append([]Stmt{&VarDecl{
				Name:    binding.Name,
				Mutable: binding.Mode == typed.BindingMutable,
				Type:    binding.Type,
				Init:    init,
			}}, out...)
			continue
		}
		if len(bindings[i].Bindings) != 1 {
			return nil, fmt.Errorf("lowering only supports single for binding, got %d", len(bindings[i].Bindings))
		}
		iterable, err := l.lowerExpr(bindings[i].Iterable)
		if err != nil {
			return nil, err
		}
		out = []Stmt{&ForEach{
			Name:     bindings[i].Bindings[0].Name,
			Iterable: iterable,
			Body:     out,
		}}
	}
	return out, nil
}

func (l *Lowerer) lowerYieldBody(block *typed.BlockStmt) ([]Stmt, Expr, *typecheck.Type, error) {
	if block == nil || len(block.Statements) == 0 {
		return nil, nil, unknownType(), fmt.Errorf("yield body must end with an expression")
	}
	prefix := block.Statements[:len(block.Statements)-1]
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*typed.ExprStmt)
	if !ok {
		return nil, nil, unknownType(), fmt.Errorf("yield body must end with an expression statement")
	}
	loweredPrefix, err := l.lowerStmtBlock(prefix)
	if err != nil {
		return nil, nil, unknownType(), err
	}
	yieldExpr, err := l.lowerExpr(exprStmt.Expr)
	if err != nil {
		return nil, nil, unknownType(), err
	}
	yieldType := exprStmt.Expr.GetType()
	if yieldType == nil {
		yieldType = unknownType()
	}
	return loweredPrefix, yieldExpr, yieldType, nil
}

func (l *Lowerer) lowerValueBlock(block *typed.BlockStmt, emptyMessage string) ([]Stmt, Expr, error) {
	if block == nil || len(block.Statements) == 0 {
		return nil, nil, fmt.Errorf("%s", emptyMessage)
	}
	prefix := block.Statements[:len(block.Statements)-1]
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*typed.ExprStmt)
	if !ok {
		return nil, nil, fmt.Errorf("%s", emptyMessage)
	}
	loweredPrefix, err := l.lowerStmtBlock(prefix)
	if err != nil {
		return nil, nil, err
	}
	value, err := l.lowerExpr(exprStmt.Expr)
	if err != nil {
		return nil, nil, err
	}
	return loweredPrefix, value, nil
}

func unwrapElemType(t *typecheck.Type) *typecheck.Type {
	if t == nil || len(t.Args) == 0 {
		return unknownType()
	}
	switch t.Name {
	case "Option", "Result":
		return t.Args[0]
	case "Either":
		if len(t.Args) > 1 {
			return t.Args[1]
		}
	}
	return unknownType()
}
