package lower

import (
	"fmt"
	"strconv"

	"a-lang/parser"
	"a-lang/typecheck"
	"a-lang/typed"
)

type Lowerer struct{}

func ProgramFromTyped(program *typed.Program) (*Program, error) {
	l := &Lowerer{}
	return l.lowerProgram(program)
}

func ProgramFromAST(program *parser.Program, types typecheck.Result) (*Program, error) {
	typedProgram, err := typed.Build(program, types)
	if err != nil {
		return nil, err
	}
	return ProgramFromTyped(typedProgram)
}

func (l *Lowerer) lowerProgram(program *typed.Program) (*Program, error) {
	out := &Program{}
	for _, stmt := range program.Globals {
		globals, err := l.lowerGlobal(stmt)
		if err != nil {
			return nil, err
		}
		out.Globals = append(out.Globals, globals...)
	}
	for _, fn := range program.Functions {
		lowered, err := l.lowerFunction(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, lowered)
	}
	for _, class := range program.Classes {
		lowered, err := l.lowerClass(class)
		if err != nil {
			return nil, err
		}
		out.Classes = append(out.Classes, lowered)
	}
	return out, nil
}

func (l *Lowerer) lowerGlobal(stmt typed.Stmt) ([]*Global, error) {
	switch s := stmt.(type) {
	case *typed.BindingStmt:
		var globals []*Global
		for _, binding := range s.Bindings {
			var init Expr
			var err error
			if binding.Init != nil {
				init, err = l.lowerExpr(binding.Init)
				if err != nil {
					return nil, err
				}
			}
			globals = append(globals, &Global{
				Name:    binding.Name,
				Mutable: binding.Mode == typed.BindingMutable,
				Type:    binding.Type,
				Init:    init,
			})
		}
		return globals, nil
	default:
		return nil, fmt.Errorf("unsupported top-level statement %T during lowering", stmt)
	}
}

func (l *Lowerer) lowerClass(class *typed.ClassDecl) (*Class, error) {
	out := &Class{Name: class.Name}
	for _, field := range class.Fields {
		out.Fields = append(out.Fields, &Field{
			Name:    field.Name,
			Mutable: field.Mode == typed.BindingMutable,
			Private: field.Private,
			Type:    field.Type,
		})
	}
	for _, method := range class.Methods {
		lowered, err := l.lowerMethod(class.Name, method)
		if err != nil {
			return nil, err
		}
		if method.Constructor {
			out.Constructor = lowered
		} else {
			out.Methods = append(out.Methods, lowered)
		}
	}
	return out, nil
}

func (l *Lowerer) lowerFunction(fn *typed.FunctionDecl) (*Function, error) {
	body, err := l.lowerBlock(fn.Body)
	if err != nil {
		return nil, err
	}
	return &Function{
		Name:       fn.Name,
		Parameters: l.lowerParams(fn.Parameters),
		ReturnType: fn.ReturnType,
		Body:       body,
	}, nil
}

func (l *Lowerer) lowerMethod(receiver string, method *typed.MethodDecl) (*Function, error) {
	body, err := l.lowerBlock(method.Body)
	if err != nil {
		return nil, err
	}
	return &Function{
		Name:        method.Name,
		Parameters:  l.lowerParams(method.Parameters),
		ReturnType:  method.ReturnType,
		Body:        body,
		Receiver:    receiver,
		Private:     method.Private,
		Constructor: method.Constructor,
	}, nil
}

func (l *Lowerer) lowerParams(params []typed.Parameter) []Parameter {
	out := make([]Parameter, len(params))
	for i, param := range params {
		out[i] = Parameter{Name: param.Name, Type: param.Type}
	}
	return out
}

func (l *Lowerer) lowerBlock(block *typed.BlockStmt) ([]Stmt, error) {
	if block == nil {
		return nil, nil
	}
	var out []Stmt
	for _, stmt := range block.Statements {
		lowered, err := l.lowerStmt(stmt)
		if err != nil {
			return nil, err
		}
		out = append(out, lowered...)
	}
	return out, nil
}

func (l *Lowerer) lowerStmt(stmt typed.Stmt) ([]Stmt, error) {
	switch s := stmt.(type) {
	case *typed.BindingStmt:
		var out []Stmt
		for _, binding := range s.Bindings {
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
	case *typed.AssignmentStmt:
		target, err := l.lowerExpr(s.Target)
		if err != nil {
			return nil, err
		}
		value, err := l.lowerExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return []Stmt{&Assign{Target: target, Operator: s.Operator, Value: value}}, nil
	case *typed.IfStmt:
		cond, err := l.lowerExpr(s.Condition)
		if err != nil {
			return nil, err
		}
		thenBlock, err := l.lowerBlock(s.Then)
		if err != nil {
			return nil, err
		}
		var elseBlock []Stmt
		if s.ElseIf != nil {
			branch, err := l.lowerStmt(s.ElseIf)
			if err != nil {
				return nil, err
			}
			elseBlock = branch
		} else if s.Else != nil {
			elseBlock, err = l.lowerBlock(s.Else)
			if err != nil {
				return nil, err
			}
		}
		return []Stmt{&If{Condition: cond, Then: thenBlock, Else: elseBlock}}, nil
	case *typed.ForStmt:
		if s.YieldBody != nil {
			return nil, fmt.Errorf("yield loops are not supported by lowering yet")
		}
		body, err := l.lowerBlock(s.Body)
		if err != nil {
			return nil, err
		}
		if len(s.Bindings) == 0 {
			return []Stmt{&Loop{Body: body}}, nil
		}
		if len(s.Bindings) != 1 {
			return nil, fmt.Errorf("multi-binding for loops are not supported by lowering yet")
		}
		iterable, err := l.lowerExpr(s.Bindings[0].Iterable)
		if err != nil {
			return nil, err
		}
		return []Stmt{&ForEach{Name: s.Bindings[0].Name, Iterable: iterable, Body: body}}, nil
	case *typed.ReturnStmt:
		value, err := l.lowerExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return []Stmt{&Return{Value: value}}, nil
	case *typed.BreakStmt:
		return []Stmt{&Break{}}, nil
	case *typed.ExprStmt:
		expr, err := l.lowerExpr(s.Expr)
		if err != nil {
			return nil, err
		}
		return []Stmt{&ExprStmt{Expr: expr}}, nil
	default:
		return nil, fmt.Errorf("unsupported statement %T during lowering", stmt)
	}
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
	case *typed.GroupExpr:
		return l.lowerExpr(e.Inner)
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
		return nil, fmt.Errorf("lambdas are not supported by lowering yet")
	case *typed.PlaceholderExpr:
		return nil, fmt.Errorf("placeholder expressions are not supported by lowering")
	case *typed.InvokeExpr:
		return nil, fmt.Errorf("function value invocation is not supported by lowering yet")
	default:
		return nil, fmt.Errorf("unsupported expression %T during lowering", expr)
	}
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
