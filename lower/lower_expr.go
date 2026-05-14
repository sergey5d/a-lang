package lower

import (
	"fmt"
	"strconv"

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
		return nil, unsupportedExprErr(expr)
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
	return l.lowerBlock(expr.BlockBody)
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
