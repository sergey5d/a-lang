package lower

import (
	"fmt"
	"strconv"

	"a-lang/parser"
	"a-lang/typecheck"
)

type Lowerer struct {
	typeInfo  map[parser.Expr]*typecheck.Type
	functions map[string]*parser.FunctionDecl
	classes   map[string]*parser.ClassDecl
}

func ProgramFromAST(program *parser.Program, types typecheck.Result) (*Program, error) {
	l := &Lowerer{
		typeInfo:  types.ExprTypes,
		functions: map[string]*parser.FunctionDecl{},
		classes:   map[string]*parser.ClassDecl{},
	}
	for _, fn := range program.Functions {
		l.functions[fn.Name] = fn
	}
	for _, class := range program.Classes {
		l.classes[class.Name] = class
	}
	return l.lowerProgram(program)
}

func (l *Lowerer) lowerProgram(program *parser.Program) (*Program, error) {
	out := &Program{}
	for _, stmt := range program.Statements {
		global, err := l.lowerGlobal(stmt)
		if err != nil {
			return nil, err
		}
		out.Globals = append(out.Globals, global...)
	}
	for _, fn := range program.Functions {
		lowered, err := l.lowerFunction(fn, "", false, false)
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

func (l *Lowerer) lowerGlobal(stmt parser.Statement) ([]*Global, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		var globals []*Global
		for i, binding := range s.Bindings {
			var init Expr
			if i < len(s.Values) {
				var err error
				init, err = l.lowerExpr(s.Values[i])
				if err != nil {
					return nil, err
				}
			}
			globals = append(globals, &Global{
				Name:    binding.Name,
				Mutable: binding.Mutable,
				Type:    l.lowerType(binding.Type),
				Init:    init,
			})
		}
		return globals, nil
	default:
		return nil, fmt.Errorf("unsupported top-level statement %T during lowering", stmt)
	}
}

func (l *Lowerer) lowerClass(class *parser.ClassDecl) (*Class, error) {
	out := &Class{Name: class.Name}
	for _, field := range class.Fields {
		out.Fields = append(out.Fields, &Field{
			Name:    field.Name,
			Mutable: field.Mutable,
			Private: field.Private,
			Type:    l.lowerType(field.Type),
		})
	}
	for _, method := range class.Methods {
		lowered, err := l.lowerFunction(method, class.Name, method.Private, method.Constructor)
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

func (l *Lowerer) lowerFunction(fn any, receiver string, private, constructor bool) (*Function, error) {
	switch decl := fn.(type) {
	case *parser.FunctionDecl:
		body, err := l.lowerBlock(decl.Body)
		if err != nil {
			return nil, err
		}
		return &Function{
			Name:       decl.Name,
			Parameters: l.lowerParams(decl.Parameters),
			ReturnType: l.lowerType(decl.ReturnType),
			Body:       body,
		}, nil
	case *parser.MethodDecl:
		body, err := l.lowerBlock(decl.Body)
		if err != nil {
			return nil, err
		}
		return &Function{
			Name:        decl.Name,
			Parameters:  l.lowerParams(decl.Parameters),
			ReturnType:  l.lowerType(decl.ReturnType),
			Body:        body,
			Receiver:    receiver,
			Private:     private,
			Constructor: constructor,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported function declaration %T", fn)
	}
}

func (l *Lowerer) lowerParams(params []parser.Parameter) []Parameter {
	out := make([]Parameter, len(params))
	for i, param := range params {
		out[i] = Parameter{Name: param.Name, Type: l.lowerType(param.Type)}
	}
	return out
}

func (l *Lowerer) lowerBlock(block *parser.BlockStmt) ([]Stmt, error) {
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

func (l *Lowerer) lowerStmt(stmt parser.Statement) ([]Stmt, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		var out []Stmt
		for i, binding := range s.Bindings {
			var init Expr
			if i < len(s.Values) {
				var err error
				init, err = l.lowerExpr(s.Values[i])
				if err != nil {
					return nil, err
				}
			}
			out = append(out, &VarDecl{
				Name:    binding.Name,
				Mutable: binding.Mutable,
				Type:    l.lowerType(binding.Type),
				Init:    init,
			})
		}
		return out, nil
	case *parser.AssignmentStmt:
		target, err := l.lowerExpr(s.Target)
		if err != nil {
			return nil, err
		}
		value, err := l.lowerExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return []Stmt{&Assign{Target: target, Operator: s.Operator, Value: value}}, nil
	case *parser.IfStmt:
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
	case *parser.ForStmt:
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
	case *parser.ReturnStmt:
		value, err := l.lowerExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return []Stmt{&Return{Value: value}}, nil
	case *parser.BreakStmt:
		return []Stmt{&Break{}}, nil
	case *parser.ExprStmt:
		expr, err := l.lowerExpr(s.Expr)
		if err != nil {
			return nil, err
		}
		return []Stmt{&ExprStmt{Expr: expr}}, nil
	default:
		return nil, fmt.Errorf("unsupported statement %T during lowering", stmt)
	}
}

func (l *Lowerer) lowerExpr(expr parser.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		if e.Name == "this" {
			return &ThisRef{Type: l.typeOf(expr)}, nil
		}
		return &VarRef{Name: e.Name, Type: l.typeOf(expr)}, nil
	case *parser.IntegerLiteral:
		v, err := strconv.ParseInt(e.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return &IntLiteral{Value: v, Type: l.typeOf(expr)}, nil
	case *parser.FloatLiteral:
		v, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return nil, err
		}
		return &FloatLiteral{Value: v, Type: l.typeOf(expr)}, nil
	case *parser.BoolLiteral:
		return &BoolLiteral{Value: e.Value, Type: l.typeOf(expr)}, nil
	case *parser.StringLiteral:
		return &StringLiteral{Value: e.Value, Type: l.typeOf(expr)}, nil
	case *parser.RuneLiteral:
		runes := []rune(e.Value)
		var value rune
		if len(runes) > 0 {
			value = runes[0]
		}
		return &RuneLiteral{Value: value, Type: l.typeOf(expr)}, nil
	case *parser.ListLiteral:
		items := make([]Expr, len(e.Elements))
		for i, item := range e.Elements {
			lowered, err := l.lowerExpr(item)
			if err != nil {
				return nil, err
			}
			items[i] = lowered
		}
		return &ListLiteral{Elements: items, Type: l.typeOf(expr)}, nil
	case *parser.GroupExpr:
		return l.lowerExpr(e.Inner)
	case *parser.UnaryExpr:
		right, err := l.lowerExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &Unary{Operator: e.Operator, Right: right, Type: l.typeOf(expr)}, nil
	case *parser.BinaryExpr:
		left, err := l.lowerExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &Binary{Left: left, Operator: e.Operator, Right: right, Type: l.typeOf(expr)}, nil
	case *parser.CallExpr:
		return l.lowerCall(e)
	case *parser.MemberExpr:
		receiver, err := l.lowerExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		return &FieldGet{Receiver: receiver, Name: e.Name, Type: l.typeOf(expr)}, nil
	case *parser.LambdaExpr:
		return nil, fmt.Errorf("lambdas are not supported by lowering yet")
	case *parser.PlaceholderExpr:
		return nil, fmt.Errorf("placeholder expressions are not supported by lowering")
	default:
		return nil, fmt.Errorf("unsupported expression %T during lowering", expr)
	}
}

func (l *Lowerer) lowerCall(call *parser.CallExpr) (Expr, error) {
	args := make([]Expr, len(call.Args))
	for i, arg := range call.Args {
		lowered, err := l.lowerExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = lowered
	}
	switch callee := call.Callee.(type) {
	case *parser.Identifier:
		if _, ok := l.classes[callee.Name]; ok {
			return &Construct{Class: callee.Name, Args: args, Type: l.typeOf(call)}, nil
		}
		return &FunctionCall{Name: callee.Name, Args: args, Type: l.typeOf(call)}, nil
	case *parser.MemberExpr:
		receiver, err := l.lowerExpr(callee.Receiver)
		if err != nil {
			return nil, err
		}
		return &MethodCall{Receiver: receiver, Method: callee.Name, Args: args, Type: l.typeOf(call)}, nil
	default:
		return nil, fmt.Errorf("unsupported callee %T during lowering", call.Callee)
	}
}

func (l *Lowerer) typeOf(expr parser.Expr) *typecheck.Type {
	if l.typeInfo == nil {
		return nil
	}
	return l.typeInfo[expr]
}

func (l *Lowerer) lowerType(ref *parser.TypeRef) *typecheck.Type {
	if ref == nil {
		return nil
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = l.lowerType(param)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: l.lowerType(ref.ReturnType),
			},
		}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = l.lowerType(arg)
	}
	kind := typecheck.TypeBuiltin
	switch ref.Name {
	case "Int", "Int64", "Bool", "String", "Rune", "Float", "Float64", "List", "Set", "Array", "Map":
		kind = typecheck.TypeBuiltin
	default:
		if _, ok := l.classes[ref.Name]; ok {
			kind = typecheck.TypeClass
		} else {
			kind = typecheck.TypeInterface
		}
	}
	return &typecheck.Type{Kind: kind, Name: ref.Name, Args: args}
}
