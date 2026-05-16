package lower

import (
	"fmt"
	"strconv"

	"a-lang/parser"
	"a-lang/typecheck"
	"a-lang/typed"
)

func unsupportedExprErr(expr typed.Expr) error {
	return fmt.Errorf("unsupported expression %T during lowering", expr)
}

func (l *Lowerer) lowerExpr(expr typed.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *typed.IdentifierExpr:
		if e.Name == "this" {
			return &ThisRef{Type: e.GetType()}, nil
		}
		return &VarRef{Name: e.Name, Type: e.GetType()}, nil
	case *typed.IntegerLiteral:
		v, err := strconv.ParseInt(e.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return &IntLiteral{Value: v, Type: e.GetType()}, nil
	case *typed.FloatLiteral:
		v, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return nil, err
		}
		return &FloatLiteral{Value: v, Type: e.GetType()}, nil
	case *typed.BoolLiteral:
		return &BoolLiteral{Value: e.Value, Type: e.GetType()}, nil
	case *typed.StringLiteral:
		return &StringLiteral{Value: e.Value, Type: e.GetType()}, nil
	case *typed.UnitLiteral:
		return &UnitLiteral{Type: e.GetType()}, nil
	case *typed.RuneLiteral:
		runes := []rune(e.Value)
		var value rune
		if len(runes) > 0 {
			value = runes[0]
		}
		return &RuneLiteral{Value: value, Type: e.GetType()}, nil
	case *typed.ListLiteral:
		items := make([]Expr, len(e.Elements))
		for i, item := range e.Elements {
			lowered, err := l.lowerExpr(item)
			if err != nil {
				return nil, err
			}
			items[i] = lowered
		}
		return &ListLiteral{Elements: items, Type: e.GetType()}, nil
	case *typed.TupleLiteral:
		items := make([]Expr, len(e.Elements))
		for i, item := range e.Elements {
			lowered, err := l.lowerExpr(item)
			if err != nil {
				return nil, err
			}
			items[i] = lowered
		}
		return &TupleLiteral{Elements: items, Type: e.GetType()}, nil
	case *typed.GroupExpr:
		return l.lowerExpr(e.Inner)
	case *typed.BlockExpr:
		return nil, unsupportedExprErr(expr)
	case *typed.AnonymousRecordExpr:
		fields := make([]RecordFieldValue, len(e.Fields))
		for i, field := range e.Fields {
			value, err := l.lowerExpr(field.Value)
			if err != nil {
				return nil, err
			}
			fields[i] = RecordFieldValue{Name: field.Name, Value: value}
		}
		return &RecordLiteral{Fields: fields, Type: e.GetType()}, nil
	case *typed.AnonymousInterfaceExpr:
		return nil, unsupportedExprErr(expr)
	case *typed.IfExpr:
		condition, err := l.lowerExpr(e.Condition)
		if err != nil {
			return nil, err
		}
		thenPrefix, thenValue, err := l.lowerExprBlock(e.Then)
		if err != nil {
			return nil, err
		}
		elsePrefix, elseValue, err := l.lowerExprBlock(e.Else)
		if err != nil {
			return nil, err
		}
		return &IfExpr{
			Condition:  condition,
			ThenPrefix: thenPrefix,
			ThenValue:  thenValue,
			ElsePrefix: elsePrefix,
			ElseValue:  elseValue,
			Type:       e.GetType(),
		}, nil
	case *typed.MatchExpr:
		return l.lowerMatchExpr(e)
	case *typed.UnaryExpr:
		right, err := l.lowerExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &Unary{Operator: e.Operator, Right: right, Type: e.GetType()}, nil
	case *typed.BinaryExpr:
		left, err := l.lowerExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &Binary{Left: left, Operator: e.Operator, Right: right, Type: e.GetType()}, nil
	case *typed.FunctionCallExpr:
		args, err := l.lowerArgs(e.Args)
		if err != nil {
			return nil, err
		}
		return &FunctionCall{Name: e.Name, Args: args, Type: e.GetType()}, nil
	case *typed.ConstructorCallExpr:
		args, err := l.lowerArgs(e.Args)
		if err != nil {
			return nil, err
		}
		return &Construct{Class: e.Class, Args: args, Type: e.GetType()}, nil
	case *typed.MethodCallExpr:
		receiver, err := l.lowerExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		args, err := l.lowerArgs(e.Args)
		if err != nil {
			return nil, err
		}
		return &MethodCall{Receiver: receiver, Method: e.Method, Args: args, Type: e.GetType()}, nil
	case *typed.FieldExpr:
		receiver, err := l.lowerExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		return &FieldGet{Receiver: receiver, Name: e.Name, Type: e.GetType()}, nil
	case *typed.IndexExpr:
		receiver, err := l.lowerExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		index, err := l.lowerExpr(e.Index)
		if err != nil {
			return nil, err
		}
		return &IndexGet{Receiver: receiver, Index: index, Type: e.GetType()}, nil
	case *typed.LambdaExpr:
		body, err := l.lowerLambdaBody(e)
		if err != nil {
			return nil, err
		}
		return &Lambda{
			Parameters: l.lowerLambdaParams(e),
			ReturnType: lambdaReturnType(e.GetType()),
			Body:       body,
			Type:       e.GetType(),
		}, nil
	case *typed.PlaceholderExpr:
		return nil, fmt.Errorf("placeholder expressions are not supported by lowering")
	case *typed.InvokeExpr:
		callee, err := l.lowerExpr(e.Callee)
		if err != nil {
			return nil, err
		}
		args, err := l.lowerArgs(e.Args)
		if err != nil {
			return nil, err
		}
		return &Invoke{Callee: callee, Args: args, Type: e.GetType()}, nil
	default:
		return nil, unsupportedExprErr(expr)
	}
}

func (l *Lowerer) lowerMatchExpr(expr *typed.MatchExpr) (Expr, error) {
	value, err := l.lowerExpr(expr.Value)
	if err != nil {
		return nil, err
	}
	return l.lowerMatchCases(value, expr.Value.GetType(), expr.Cases, expr.Partial, expr.GetType())
}

func (l *Lowerer) lowerMatchCases(value Expr, valueType *typecheck.Type, cases []typed.MatchCase, partial bool, resultType *typecheck.Type) (Expr, error) {
	if len(cases) == 0 {
		if partial {
			return &FunctionCall{Name: "None", Args: nil, Type: resultType}, nil
		}
		return zeroValueExpr(resultType), nil
	}

	current := cases[0]
	elseExpr, err := l.lowerMatchCases(value, valueType, cases[1:], partial, resultType)
	if err != nil {
		return nil, err
	}

	condition, bindPrefix, err := l.lowerMatchPattern(value, valueType, current.Pattern)
	if err != nil {
		return nil, err
	}

	thenPrefix := append([]Stmt{}, bindPrefix...)
	thenValue, err := l.lowerMatchCaseValue(current, partial, resultType)
	if err != nil {
		return nil, err
	}
	if current.Body != nil {
		casePrefix, caseValue, err := l.lowerExprBlock(current.Body)
		if err != nil {
			return nil, err
		}
		thenPrefix = append(thenPrefix, casePrefix...)
		thenValue = caseValue
		if partial {
			thenValue = &FunctionCall{Name: "Some", Args: []Expr{thenValue}, Type: resultType}
		}
	}
	if current.Guard != nil {
		guardExpr, err := l.lowerExpr(current.Guard)
		if err != nil {
			return nil, err
		}
		thenValue = &IfExpr{
			Condition:  guardExpr,
			ThenPrefix: nil,
			ThenValue:  thenValue,
			ElsePrefix: nil,
			ElseValue:  elseExpr,
			Type:       resultType,
		}
	}

	return &IfExpr{
		Condition:  condition,
		ThenPrefix: thenPrefix,
		ThenValue:  thenValue,
		ElsePrefix: nil,
		ElseValue:  elseExpr,
		Type:       resultType,
	}, nil
}

func (l *Lowerer) lowerMatchCaseValue(matchCase typed.MatchCase, partial bool, resultType *typecheck.Type) (Expr, error) {
	if matchCase.Expr != nil {
		value, err := l.lowerExpr(matchCase.Expr)
		if err != nil {
			return nil, err
		}
		if partial {
			return &FunctionCall{Name: "Some", Args: []Expr{value}, Type: resultType}, nil
		}
		return value, nil
	}
	if matchCase.Body != nil {
		casePrefix, caseValue, err := l.lowerExprBlock(matchCase.Body)
		if err != nil {
			return nil, err
		}
		if len(casePrefix) == 0 {
			if partial {
				return &FunctionCall{Name: "Some", Args: []Expr{caseValue}, Type: resultType}, nil
			}
			return caseValue, nil
		}
		return &IfExpr{
			Condition:  &BoolLiteral{Value: true, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}},
			ThenPrefix: casePrefix,
			ThenValue: func() Expr {
				if partial {
					return &FunctionCall{Name: "Some", Args: []Expr{caseValue}, Type: resultType}
				}
				return caseValue
			}(),
			ElsePrefix: nil,
			ElseValue:  zeroValueExpr(resultType),
			Type:       resultType,
		}, nil
	}
	if partial {
		return &FunctionCall{Name: "None", Args: nil, Type: resultType}, nil
	}
	return zeroValueExpr(resultType), nil
}

func (l *Lowerer) lowerMatchPattern(value Expr, valueType *typecheck.Type, pattern parser.Pattern) (Expr, []Stmt, error) {
	switch p := pattern.(type) {
	case *parser.WildcardPattern:
		return &BoolLiteral{Value: true, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}, nil, nil
	case *parser.BindingPattern:
		if p.Name == "_" {
			return &BoolLiteral{Value: true, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}, nil, nil
		}
		return &BoolLiteral{Value: true, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}, []Stmt{
			&VarDecl{Name: p.Name, Mutable: false, Type: valueType},
			&Assign{Target: &VarRef{Name: p.Name, Type: valueType}, Operator: ":=", Value: value},
		}, nil
	case *parser.LiteralPattern:
		lit, err := l.lowerParserLiteral(p.Value)
		if err != nil {
			return nil, nil, err
		}
		return equalityExpr(value, lit, valueType), nil, nil
	case *parser.ConstructorPattern:
		return l.lowerConstructorPattern(value, valueType, p)
	default:
		return nil, nil, fmt.Errorf("unsupported match pattern %T during lowering", pattern)
	}
}

func (l *Lowerer) lowerConstructorPattern(value Expr, valueType *typecheck.Type, pattern *parser.ConstructorPattern) (Expr, []Stmt, error) {
	caseName := pattern.Path[len(pattern.Path)-1]
	if valueType != nil && valueType.Name == "Option" {
		switch caseName {
		case "Some":
			condition := &MethodCall{Receiver: value, Method: "isSet", Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}
			if len(pattern.Args) == 0 {
				return condition, nil, nil
			}
			if len(pattern.Args) != 1 {
				return nil, nil, fmt.Errorf("Option.Some pattern expects 1 argument")
			}
			unwrapped := &MethodCall{Receiver: value, Method: "expect", Type: unwrapElemType(valueType)}
			innerCond, innerPrefix, err := l.lowerMatchPattern(unwrapped, unwrapElemType(valueType), pattern.Args[0])
			if err != nil {
				return nil, nil, err
			}
			return andExpr(condition, innerCond), innerPrefix, nil
		case "None":
			return &MethodCall{Receiver: value, Method: "isEmpty", Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}, nil, nil
		}
	}
	if valueType == nil {
		return nil, nil, fmt.Errorf("constructor pattern requires resolved match type")
	}
	class, ok := l.classes[valueType.Name]
	if !ok || !class.Enum {
		return nil, nil, fmt.Errorf("constructor pattern requires enum-like match type, got %s", valueType.String())
	}
	var enumCase *parser.EnumCaseDecl
	for i := range class.Cases {
		if class.Cases[i].Name == caseName {
			enumCase = &class.Cases[i]
			break
		}
	}
	if enumCase == nil {
		return nil, nil, fmt.Errorf("unknown enum case %q for %s", caseName, valueType.Name)
	}
	caseTypeName := valueType.Name + "_" + caseName
	condition := Expr(&TypeIs{
		Value:  value,
		Target: caseTypeName,
		Type:   &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
	})
	castValue := Expr(&Cast{
		Value:  value,
		Target: caseTypeName,
		Type:   &typecheck.Type{Kind: typecheck.TypeClass, Name: caseTypeName, Args: append([]*typecheck.Type(nil), valueType.Args...)},
	})
	var prefix []Stmt
	fullCondition := condition
	for i, arg := range pattern.Args {
		if i >= len(enumCase.Fields) {
			return nil, nil, fmt.Errorf("enum case %q expects %d pattern args, got %d", caseName, len(enumCase.Fields), len(pattern.Args))
		}
		field := enumCase.Fields[i]
		fieldExpr := &FieldGet{Receiver: castValue, Name: field.Name, Type: l.resolveFieldType(field.Type)}
		argCondition, bindPrefix, err := l.lowerMatchPattern(fieldExpr, l.resolveFieldType(field.Type), arg)
		if err != nil {
			return nil, nil, err
		}
		fullCondition = andExpr(fullCondition, argCondition)
		prefix = append(prefix, bindPrefix...)
	}
	return fullCondition, prefix, nil
}

func (l *Lowerer) lowerParserLiteral(expr parser.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		v, err := strconv.ParseInt(e.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return &IntLiteral{Value: v, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Int"}}, nil
	case *parser.FloatLiteral:
		v, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return nil, err
		}
		return &FloatLiteral{Value: v, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Float"}}, nil
	case *parser.BoolLiteral:
		return &BoolLiteral{Value: e.Value, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}, nil
	case *parser.StringLiteral:
		return &StringLiteral{Value: e.Value, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Str"}}, nil
	case *parser.RuneLiteral:
		runes := []rune(e.Value)
		var value rune
		if len(runes) > 0 {
			value = runes[0]
		}
		return &RuneLiteral{Value: value, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Rune"}}, nil
	case *parser.UnitLiteral:
		return &UnitLiteral{Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Unit"}}, nil
	default:
		return nil, fmt.Errorf("unsupported literal pattern %T during lowering", expr)
	}
}

func (l *Lowerer) lowerExprFromParser(expr parser.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		return &VarRef{Name: e.Name}, nil
	case *parser.IntegerLiteral, *parser.FloatLiteral, *parser.BoolLiteral, *parser.StringLiteral, *parser.RuneLiteral, *parser.UnitLiteral:
		return l.lowerParserLiteral(expr)
	case *parser.GroupExpr:
		return l.lowerExprFromParser(e.Inner)
	case *parser.UnaryExpr:
		right, err := l.lowerExprFromParser(e.Right)
		if err != nil {
			return nil, err
		}
		return &Unary{Operator: e.Operator, Right: right}, nil
	case *parser.BinaryExpr:
		left, err := l.lowerExprFromParser(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExprFromParser(e.Right)
		if err != nil {
			return nil, err
		}
		return &Binary{Left: left, Operator: e.Operator, Right: right}, nil
	case *parser.CallExpr:
		callee, err := l.lowerExprFromParser(e.Callee)
		if err != nil {
			return nil, err
		}
		args := make([]Expr, len(e.Args))
		for i, arg := range e.Args {
			value, err := l.lowerExprFromParser(arg.Value)
			if err != nil {
				return nil, err
			}
			args[i] = value
		}
		return &Invoke{Callee: callee, Args: args}, nil
	case *parser.MemberExpr:
		receiver, err := l.lowerExprFromParser(e.Receiver)
		if err != nil {
			return nil, err
		}
		return &FieldGet{Receiver: receiver, Name: e.Name}, nil
	default:
		return nil, fmt.Errorf("unsupported parser expression %T during lowering", expr)
	}
}

func equalityExpr(left, right Expr, leftType *typecheck.Type) Expr {
	if leftType != nil && leftType.Kind == typecheck.TypeBuiltin && leftType.Name == "Str" {
		return &MethodCall{
			Receiver: left,
			Method:   "equals",
			Args:     []Expr{right},
			Type:     &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
		}
	}
	return &Binary{
		Left:     left,
		Operator: "==",
		Right:    right,
		Type:     &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
	}
}

func andExpr(left, right Expr) Expr {
	if isBoolLiteral(left, true) {
		return right
	}
	if isBoolLiteral(right, true) {
		return left
	}
	if isBoolLiteral(left, false) || isBoolLiteral(right, false) {
		return &BoolLiteral{Value: false, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}
	}
	return &Binary{
		Left:     left,
		Operator: "&&",
		Right:    right,
		Type:     &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"},
	}
}

func isBoolLiteral(expr Expr, value bool) bool {
	lit, ok := expr.(*BoolLiteral)
	return ok && lit.Value == value
}

func zeroValueExpr(t *typecheck.Type) Expr {
	if t == nil {
		return &BoolLiteral{Value: false, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Bool"}}
	}
	if t.Name == "Option" {
		return &FunctionCall{Name: "None", Args: nil, Type: t}
	}
	switch t.Kind {
	case typecheck.TypeBuiltin:
		switch t.Name {
		case "Bool":
			return &BoolLiteral{Value: false, Type: t}
		case "Float", "Float64", "Decimal":
			return &FloatLiteral{Value: 0, Type: t}
		case "Str":
			return &StringLiteral{Value: "", Type: t}
		case "Rune":
			return &RuneLiteral{Value: 0, Type: t}
		case "Unit":
			return &UnitLiteral{Type: t}
		default:
			return &IntLiteral{Value: 0, Type: t}
		}
	default:
		return &IntLiteral{Value: 0, Type: &typecheck.Type{Kind: typecheck.TypeBuiltin, Name: "Int"}}
	}
}

func (l *Lowerer) lowerExprBlock(block *typed.BlockStmt) ([]Stmt, Expr, error) {
	if block == nil || len(block.Statements) == 0 {
		return nil, nil, fmt.Errorf("expression block must end with an expression")
	}
	prefix := block.Statements[:len(block.Statements)-1]
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*typed.ExprStmt)
	if !ok {
		return nil, nil, fmt.Errorf("expression block must end with an expression statement")
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

func (l *Lowerer) lowerArgs(args []typed.Expr) ([]Expr, error) {
	out := make([]Expr, len(args))
	for i, arg := range args {
		lowered, err := l.lowerExpr(arg)
		if err != nil {
			return nil, err
		}
		out[i] = lowered
	}
	return out, nil
}

func (l *Lowerer) lowerLambdaBody(expr *typed.LambdaExpr) ([]Stmt, error) {
	if expr.Body != nil {
		body, err := l.lowerExpr(expr.Body)
		if err != nil {
			return nil, err
		}
		return []Stmt{&Return{Value: body}}, nil
	}
	if expr.BlockBody == nil {
		return nil, nil
	}
	if isUnitType(lambdaReturnType(expr.GetType())) {
		return l.lowerBlock(expr.BlockBody)
	}
	prefix, value, err := l.lowerExprBlock(expr.BlockBody)
	if err != nil {
		return nil, err
	}
	return append(prefix, &Return{Value: value}), nil
}

func (l *Lowerer) lowerLambdaParams(expr *typed.LambdaExpr) []Parameter {
	params := expr.Parameters
	out := make([]Parameter, len(params))
	var sig *typecheck.Signature
	if expr.GetType() != nil && expr.GetType().Kind == typecheck.TypeFunction {
		sig = expr.GetType().Signature
	}
	for i, param := range params {
		paramType := param.Type
		if (paramType == nil || paramType.Kind == typecheck.TypeUnknown) && sig != nil && i < len(sig.Parameters) {
			paramType = sig.Parameters[i]
		}
		if paramType == nil {
			paramType = unknownType()
		}
		out[i] = Parameter{Name: param.Name, Type: paramType}
	}
	return out
}

func lambdaReturnType(t *typecheck.Type) *typecheck.Type {
	if t != nil && t.Kind == typecheck.TypeFunction && t.Signature != nil {
		return t.Signature.ReturnType
	}
	return unknownType()
}

func isUnitType(t *typecheck.Type) bool {
	return t != nil && t.Kind == typecheck.TypeBuiltin && t.Name == "Unit"
}
