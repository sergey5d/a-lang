package typed

import (
	"fmt"

	"a-lang/parser"
	"a-lang/typecheck"
)

type Builder struct {
	exprTypes  map[parser.Expr]*typecheck.Type
	functions  map[string]*parser.FunctionDecl
	classes    map[string]*parser.ClassDecl
	interfaces map[string]*parser.InterfaceDecl
}

func Build(program *parser.Program, info typecheck.Result) (*Program, error) {
	b := &Builder{
		exprTypes:  info.ExprTypes,
		functions:  map[string]*parser.FunctionDecl{},
		classes:    map[string]*parser.ClassDecl{},
		interfaces: map[string]*parser.InterfaceDecl{},
	}
	for _, fn := range program.Functions {
		b.functions[fn.Name] = fn
	}
	for _, class := range program.Classes {
		b.classes[class.Name] = class
	}
	for _, iface := range program.Interfaces {
		b.interfaces[iface.Name] = iface
	}
	return b.buildProgram(program)
}

func (b *Builder) buildProgram(program *parser.Program) (*Program, error) {
	out := &Program{Span: program.Span}

	for _, stmt := range program.Statements {
		built, err := b.buildStmt(stmt)
		if err != nil {
			return nil, err
		}
		out.Globals = append(out.Globals, built)
	}

	for _, fn := range program.Functions {
		built, err := b.buildFunction(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, built)
	}

	for _, iface := range program.Interfaces {
		built, err := b.buildInterface(iface)
		if err != nil {
			return nil, err
		}
		out.Interfaces = append(out.Interfaces, built)
	}

	for _, class := range program.Classes {
		built, err := b.buildClass(class)
		if err != nil {
			return nil, err
		}
		out.Classes = append(out.Classes, built)
	}

	return out, nil
}

func (b *Builder) buildFunction(fn *parser.FunctionDecl) (*FunctionDecl, error) {
	body, err := b.buildBlock(fn.Body)
	if err != nil {
		return nil, err
	}
	return &FunctionDecl{
		Name:       fn.Name,
		Parameters: b.buildParameters(fn.Parameters),
		ReturnType: b.typeFromRef(fn.ReturnType),
		Body:       body,
		Span:       fn.Span,
	}, nil
}

func (b *Builder) buildInterface(iface *parser.InterfaceDecl) (*InterfaceDecl, error) {
	methods := make([]InterfaceMethod, len(iface.Methods))
	for i, method := range iface.Methods {
		methods[i] = InterfaceMethod{
			Name:       method.Name,
			Parameters: b.buildParameters(method.Parameters),
			ReturnType: b.typeFromRef(method.ReturnType),
			Span:       method.Span,
		}
	}

	return &InterfaceDecl{
		Name:           iface.Name,
		TypeParameters: b.buildTypeParameters(iface.TypeParameters),
		Methods:        methods,
		Span:           iface.Span,
	}, nil
}

func (b *Builder) buildClass(class *parser.ClassDecl) (*ClassDecl, error) {
	fields := make([]FieldDecl, len(class.Fields))
	for i, field := range class.Fields {
		var init Expr
		var err error
		if field.Initializer != nil {
			init, err = b.buildExpr(field.Initializer)
			if err != nil {
				return nil, err
			}
		}
		fields[i] = FieldDecl{
			Name:     field.Name,
			Type:     b.typeFromRef(field.Type),
			Mode:     modeFromMutable(field.Mutable),
			InitMode: initMode(field.Deferred, init),
			Init:     init,
			Private:  field.Private,
			Span:     field.Span,
		}
	}

	methods := make([]*MethodDecl, len(class.Methods))
	for i, method := range class.Methods {
		body, err := b.buildBlock(method.Body)
		if err != nil {
			return nil, err
		}
		methods[i] = &MethodDecl{
			Name:        method.Name,
			Parameters:  b.buildParameters(method.Parameters),
			ReturnType:  b.typeFromRef(method.ReturnType),
			Body:        body,
			Private:     method.Private,
			Constructor: method.Constructor,
			Span:        method.Span,
		}
	}

	implements := make([]*typecheck.Type, len(class.Implements))
	for i, impl := range class.Implements {
		implements[i] = b.typeFromRef(impl)
	}

	return &ClassDecl{
		Name:           class.Name,
		TypeParameters: b.buildTypeParameters(class.TypeParameters),
		Interfaces:     implements,
		Fields:         fields,
		Methods:        methods,
		Span:           class.Span,
	}, nil
}

func (b *Builder) buildBlock(block *parser.BlockStmt) (*BlockStmt, error) {
	if block == nil {
		return nil, nil
	}
	statements := make([]Stmt, len(block.Statements))
	for i, stmt := range block.Statements {
		built, err := b.buildStmt(stmt)
		if err != nil {
			return nil, err
		}
		statements[i] = built
	}
	return &BlockStmt{Statements: statements, Span: block.Span}, nil
}

func (b *Builder) buildStmt(stmt parser.Statement) (Stmt, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		bindings := make([]BindingDecl, len(s.Bindings))
		for i, binding := range s.Bindings {
			var init Expr
			var err error
			if i < len(s.Values) && s.Values[i] != nil {
				init, err = b.buildExpr(s.Values[i])
				if err != nil {
					return nil, err
				}
			}
			bindings[i] = BindingDecl{
				Name:     binding.Name,
				Type:     b.bindingType(binding, init),
				Mode:     modeFromMutable(binding.Mutable),
				InitMode: initMode(binding.Deferred, init),
				Init:     init,
				Span:     binding.Span,
			}
		}
		return &BindingStmt{Bindings: bindings, Span: s.Span}, nil
	case *parser.AssignmentStmt:
		target, err := b.buildExpr(s.Target)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return &AssignmentStmt{Target: target, Operator: s.Operator, Value: value, Span: s.Span}, nil
	case *parser.IfStmt:
		cond, err := b.buildExpr(s.Condition)
		if err != nil {
			return nil, err
		}
		thenBlock, err := b.buildBlock(s.Then)
		if err != nil {
			return nil, err
		}
		var elseIf *IfStmt
		if s.ElseIf != nil {
			built, err := b.buildStmt(s.ElseIf)
			if err != nil {
				return nil, err
			}
			elseIf = built.(*IfStmt)
		}
		elseBlock, err := b.buildBlock(s.Else)
		if err != nil {
			return nil, err
		}
		return &IfStmt{Condition: cond, Then: thenBlock, ElseIf: elseIf, Else: elseBlock, Span: s.Span}, nil
	case *parser.ForStmt:
		bindings := make([]ForBinding, len(s.Bindings))
		for i, binding := range s.Bindings {
			iterable, err := b.buildExpr(binding.Iterable)
			if err != nil {
				return nil, err
			}
			bindings[i] = ForBinding{
				Name:     binding.Name,
				Type:     elementType(iterable.GetType()),
				Iterable: iterable,
				Span:     binding.Span,
			}
		}
		body, err := b.buildBlock(s.Body)
		if err != nil {
			return nil, err
		}
		yieldBody, err := b.buildBlock(s.YieldBody)
		if err != nil {
			return nil, err
		}
		return &ForStmt{Bindings: bindings, Body: body, YieldBody: yieldBody, Span: s.Span}, nil
	case *parser.ReturnStmt:
		value, err := b.buildExpr(s.Value)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{Value: value, Span: s.Span}, nil
	case *parser.BreakStmt:
		return &BreakStmt{Span: s.Span}, nil
	case *parser.ExprStmt:
		expr, err := b.buildExpr(s.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: expr, Span: s.Span}, nil
	default:
		return nil, fmt.Errorf("unsupported statement type %T", stmt)
	}
}

func (b *Builder) buildExpr(expr parser.Expr) (Expr, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		return &IdentifierExpr{baseExpr: b.base(expr), Name: e.Name}, nil
	case *parser.PlaceholderExpr:
		return &PlaceholderExpr{baseExpr: b.base(expr)}, nil
	case *parser.IntegerLiteral:
		return &IntegerLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.FloatLiteral:
		return &FloatLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.RuneLiteral:
		return &RuneLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.BoolLiteral:
		return &BoolLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.StringLiteral:
		return &StringLiteral{baseExpr: b.base(expr), Value: e.Value}, nil
	case *parser.ListLiteral:
		elements := make([]Expr, len(e.Elements))
		for i, element := range e.Elements {
			built, err := b.buildExpr(element)
			if err != nil {
				return nil, err
			}
			elements[i] = built
		}
		return &ListLiteral{baseExpr: b.base(expr), Elements: elements}, nil
	case *parser.GroupExpr:
		inner, err := b.buildExpr(e.Inner)
		if err != nil {
			return nil, err
		}
		return &GroupExpr{baseExpr: b.base(expr), Inner: inner}, nil
	case *parser.UnaryExpr:
		right, err := b.buildExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{baseExpr: b.base(expr), Operator: e.Operator, Right: right}, nil
	case *parser.BinaryExpr:
		left, err := b.buildExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := b.buildExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{baseExpr: b.base(expr), Left: left, Operator: e.Operator, Right: right}, nil
	case *parser.MemberExpr:
		receiver, err := b.buildExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		return &FieldExpr{baseExpr: b.base(expr), Receiver: receiver, Name: e.Name}, nil
	case *parser.CallExpr:
		return b.buildCallExpr(e)
	case *parser.LambdaExpr:
		params := make([]LambdaParameter, len(e.Parameters))
		for i, param := range e.Parameters {
			params[i] = LambdaParameter{Name: param.Name, Type: b.typeFromRef(param.Type), Span: param.Span}
		}
		var body Expr
		var err error
		if e.Body != nil {
			body, err = b.buildExpr(e.Body)
			if err != nil {
				return nil, err
			}
		}
		blockBody, err := b.buildBlock(e.BlockBody)
		if err != nil {
			return nil, err
		}
		return &LambdaExpr{baseExpr: b.base(expr), Parameters: params, Body: body, BlockBody: blockBody}, nil
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

func (b *Builder) buildCallExpr(call *parser.CallExpr) (Expr, error) {
	args := make([]Expr, len(call.Args))
	for i, arg := range call.Args {
		built, err := b.buildExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = built
	}

	switch callee := call.Callee.(type) {
	case *parser.Identifier:
		if _, ok := b.classes[callee.Name]; ok {
			return &ConstructorCallExpr{baseExpr: b.base(call), Class: callee.Name, Args: args}, nil
		}
		if _, ok := b.functions[callee.Name]; ok {
			return &FunctionCallExpr{baseExpr: b.base(call), Name: callee.Name, Args: args}, nil
		}
	case *parser.MemberExpr:
		receiver, err := b.buildExpr(callee.Receiver)
		if err != nil {
			return nil, err
		}
		return &MethodCallExpr{baseExpr: b.base(call), Receiver: receiver, Method: callee.Name, Args: args}, nil
	}

	calleeExpr, err := b.buildExpr(call.Callee)
	if err != nil {
		return nil, err
	}
	return &InvokeExpr{baseExpr: b.base(call), Callee: calleeExpr, Args: args}, nil
}

func (b *Builder) buildParameters(params []parser.Parameter) []Parameter {
	out := make([]Parameter, len(params))
	for i, param := range params {
		out[i] = Parameter{Name: param.Name, Type: b.typeFromRef(param.Type), Span: param.Span}
	}
	return out
}

func (b *Builder) buildTypeParameters(params []parser.TypeParameter) []TypeParameter {
	out := make([]TypeParameter, len(params))
	for i, param := range params {
		out[i] = TypeParameter{Name: param.Name, Span: param.Span}
	}
	return out
}

func (b *Builder) base(expr parser.Expr) baseExpr {
	return baseExpr{Type: b.exprTypes[expr], Span: exprSpan(expr)}
}

func (b *Builder) bindingType(binding parser.Binding, init Expr) *typecheck.Type {
	if binding.Type != nil {
		return b.typeFromRef(binding.Type)
	}
	if init != nil {
		return init.GetType()
	}
	return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
}

func modeFromMutable(mutable bool) BindingMode {
	if mutable {
		return BindingMutable
	}
	return BindingImmutable
}

func initMode(deferred bool, init Expr) InitMode {
	if deferred || init == nil {
		return InitDeferred
	}
	return InitImmediate
}

func elementType(typ *typecheck.Type) *typecheck.Type {
	if typ == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if len(typ.Args) > 0 {
		return typ.Args[0]
	}
	return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
}

func exprSpan(expr parser.Expr) parser.Span {
	switch e := expr.(type) {
	case *parser.Identifier:
		return e.Span
	case *parser.PlaceholderExpr:
		return e.Span
	case *parser.IntegerLiteral:
		return e.Span
	case *parser.FloatLiteral:
		return e.Span
	case *parser.RuneLiteral:
		return e.Span
	case *parser.BoolLiteral:
		return e.Span
	case *parser.StringLiteral:
		return e.Span
	case *parser.ListLiteral:
		return e.Span
	case *parser.CallExpr:
		return e.Span
	case *parser.MemberExpr:
		return e.Span
	case *parser.LambdaExpr:
		return e.Span
	case *parser.BinaryExpr:
		return e.Span
	case *parser.UnaryExpr:
		return e.Span
	case *parser.GroupExpr:
		return e.Span
	default:
		return parser.Span{}
	}
}

func (b *Builder) typeFromRef(ref *parser.TypeRef) *typecheck.Type {
	if ref == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = b.typeFromRef(param)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: b.typeFromRef(ref.ReturnType),
			},
		}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = b.typeFromRef(arg)
	}
	return &typecheck.Type{Kind: b.kindOf(ref.Name), Name: ref.Name, Args: args}
}

func (b *Builder) kindOf(name string) typecheck.TypeKind {
	switch name {
	case "Int", "Float", "Bool", "String", "Rune", "Decimal", "List", "Array", "Map", "Set":
		return typecheck.TypeBuiltin
	}
	if _, ok := b.classes[name]; ok {
		return typecheck.TypeClass
	}
	if _, ok := b.interfaces[name]; ok {
		return typecheck.TypeInterface
	}
	return typecheck.TypeUnknown
}
