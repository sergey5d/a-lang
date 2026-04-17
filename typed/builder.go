package typed

import (
	"fmt"

	"a-lang/parser"
	"a-lang/typecheck"
)

type Builder struct {
	exprTypes        map[parser.Expr]*typecheck.Type
	functions        map[string]*parser.FunctionDecl
	classes          map[string]*parser.ClassDecl
	interfaces       map[string]*parser.InterfaceDecl
	nextID           int
	scopes           []map[string]SymbolRef
	thisStack        []SymbolRef
	functionSymbols  map[string]SymbolRef
	classSymbols     map[string]SymbolRef
	interfaceSymbols map[string]SymbolRef
	fieldSymbols     map[string]map[string]SymbolRef
	methodSymbols    map[string]map[string][]methodSymbol
}

type methodSymbol struct {
	decl   *parser.MethodDecl
	symbol SymbolRef
}

func Build(program *parser.Program, info typecheck.Result) (*Program, error) {
	b := &Builder{
		exprTypes:        info.ExprTypes,
		functions:        map[string]*parser.FunctionDecl{},
		classes:          map[string]*parser.ClassDecl{},
		interfaces:       map[string]*parser.InterfaceDecl{},
		functionSymbols:  map[string]SymbolRef{},
		classSymbols:     map[string]SymbolRef{},
		interfaceSymbols: map[string]SymbolRef{},
		fieldSymbols:     map[string]map[string]SymbolRef{},
		methodSymbols:    map[string]map[string][]methodSymbol{},
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
	b.collectSymbols(program)
	return b.buildProgram(program)
}

func (b *Builder) collectSymbols(program *parser.Program) {
	for _, fn := range program.Functions {
		b.functionSymbols[fn.Name] = b.newSymbol(SymbolFunction, fn.Name, "", fn.Span)
	}
	for _, iface := range program.Interfaces {
		b.interfaceSymbols[iface.Name] = b.newSymbol(SymbolInterface, iface.Name, "", iface.Span)
	}
	for _, class := range program.Classes {
		b.classSymbols[class.Name] = b.newSymbol(SymbolClass, class.Name, "", class.Span)
		fields := map[string]SymbolRef{}
		methods := map[string][]methodSymbol{}
		for _, field := range class.Fields {
			fields[field.Name] = b.newSymbol(SymbolField, field.Name, class.Name, field.Span)
		}
		for _, method := range class.Methods {
			sym := b.newSymbol(SymbolMethod, method.Name, class.Name, method.Span)
			methods[method.Name] = append(methods[method.Name], methodSymbol{decl: method, symbol: sym})
		}
		b.fieldSymbols[class.Name] = fields
		b.methodSymbols[class.Name] = methods
	}
}

func (b *Builder) newSymbol(kind SymbolKind, name, owner string, span parser.Span) SymbolRef {
	b.nextID++
	return SymbolRef{ID: b.nextID, Kind: kind, Name: name, Owner: owner, Span: span}
}

func (b *Builder) pushScope() {
	b.scopes = append(b.scopes, map[string]SymbolRef{})
}

func (b *Builder) popScope() {
	b.scopes = b.scopes[:len(b.scopes)-1]
}

func (b *Builder) defineSymbol(symbol SymbolRef) {
	if len(b.scopes) == 0 {
		b.pushScope()
	}
	b.scopes[len(b.scopes)-1][symbol.Name] = symbol
}

func (b *Builder) lookupSymbol(name string) (*SymbolRef, bool) {
	for i := len(b.scopes) - 1; i >= 0; i-- {
		if symbol, ok := b.scopes[i][name]; ok {
			return &symbol, true
		}
	}
	if symbol, ok := b.functionSymbols[name]; ok {
		return &symbol, true
	}
	if symbol, ok := b.classSymbols[name]; ok {
		return &symbol, true
	}
	if symbol, ok := b.interfaceSymbols[name]; ok {
		return &symbol, true
	}
	return nil, false
}

func (b *Builder) buildProgram(program *parser.Program) (*Program, error) {
	out := &Program{Span: program.Span}
	b.pushScope()
	defer b.popScope()

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
	b.pushScope()
	defer b.popScope()
	params := b.buildParameters(fn.Parameters)
	for _, param := range params {
		b.defineSymbol(param.Symbol)
	}
	body, err := b.buildBlock(fn.Body)
	if err != nil {
		return nil, err
	}
	return &FunctionDecl{
		Name:       fn.Name,
		Parameters: params,
		ReturnType: b.typeFromRef(fn.ReturnType),
		Body:       body,
		Symbol:     b.functionSymbols[fn.Name],
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
		Symbol:         b.interfaceSymbols[iface.Name],
		Span:           iface.Span,
	}, nil
}

func (b *Builder) buildClass(class *parser.ClassDecl) (*ClassDecl, error) {
	b.pushScope()
	defer b.popScope()
	thisSymbol := b.newSymbol(SymbolThis, "this", class.Name, class.Span)
	b.thisStack = append(b.thisStack, thisSymbol)
	defer func() { b.thisStack = b.thisStack[:len(b.thisStack)-1] }()
	b.defineSymbol(thisSymbol)
	for _, field := range class.Fields {
		b.defineSymbol(b.fieldSymbols[class.Name][field.Name])
	}
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
			Symbol:   b.fieldSymbols[class.Name][field.Name],
			Span:     field.Span,
		}
	}

	methods := make([]*MethodDecl, len(class.Methods))
	for i, method := range class.Methods {
		b.pushScope()
		b.defineSymbol(thisSymbol)
		params := b.buildParameters(method.Parameters)
		for _, param := range params {
			b.defineSymbol(param.Symbol)
		}
		body, err := b.buildBlock(method.Body)
		b.popScope()
		if err != nil {
			return nil, err
		}
		methods[i] = &MethodDecl{
			Name:        method.Name,
			Parameters:  params,
			ReturnType:  b.typeFromRef(method.ReturnType),
			Body:        body,
			Private:     method.Private,
			Constructor: method.Constructor,
			Symbol:      b.lookupMethodSymbol(class.Name, method),
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
		Symbol:         b.classSymbols[class.Name],
		Span:           class.Span,
	}, nil
}

func (b *Builder) buildBlock(block *parser.BlockStmt) (*BlockStmt, error) {
	if block == nil {
		return nil, nil
	}
	b.pushScope()
	defer b.popScope()
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
			symbol := b.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			bindings[i] = BindingDecl{
				Name:     binding.Name,
				Type:     b.bindingType(binding, init),
				Mode:     modeFromMutable(binding.Mutable),
				InitMode: initMode(binding.Deferred, init),
				Init:     init,
				Symbol:   symbol,
				Span:     binding.Span,
			}
			b.defineSymbol(symbol)
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
			symbol := b.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			bindings[i] = ForBinding{
				Name:     binding.Name,
				Type:     elementType(iterable.GetType()),
				Iterable: iterable,
				Symbol:   symbol,
				Span:     binding.Span,
			}
		}
		b.pushScope()
		for _, binding := range bindings {
			b.defineSymbol(binding.Symbol)
		}
		body, err := b.buildBlock(s.Body)
		b.popScope()
		if err != nil {
			return nil, err
		}
		b.pushScope()
		for _, binding := range bindings {
			b.defineSymbol(binding.Symbol)
		}
		yieldBody, err := b.buildBlock(s.YieldBody)
		b.popScope()
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
		ident := &IdentifierExpr{baseExpr: b.base(expr), Name: e.Name}
		if symbol, ok := b.lookupSymbol(e.Name); ok {
			ident.Symbol = symbol
		}
		return ident, nil
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
		field := &FieldExpr{baseExpr: b.base(expr), Receiver: receiver, Name: e.Name}
		field.Field = b.resolveFieldSymbol(receiver.GetType(), e.Name)
		return field, nil
	case *parser.IndexExpr:
		receiver, err := b.buildExpr(e.Receiver)
		if err != nil {
			return nil, err
		}
		index, err := b.buildExpr(e.Index)
		if err != nil {
			return nil, err
		}
		return &IndexExpr{baseExpr: b.base(expr), Receiver: receiver, Index: index}, nil
	case *parser.CallExpr:
		return b.buildCallExpr(e)
	case *parser.LambdaExpr:
		b.pushScope()
		defer b.popScope()
		params := make([]LambdaParameter, len(e.Parameters))
		for i, param := range e.Parameters {
			symbol := b.newSymbol(SymbolParameter, param.Name, "", param.Span)
			params[i] = LambdaParameter{Name: param.Name, Type: b.typeFromRef(param.Type), Symbol: symbol, Span: param.Span}
			b.defineSymbol(symbol)
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
			expr := &ConstructorCallExpr{
				baseExpr: b.base(call),
				Class:    callee.Name,
				Args:     args,
				Dispatch: DispatchConstruct,
			}
			if symbol, ok := b.classSymbols[callee.Name]; ok {
				expr.ClassSymbol = &symbol
			}
			expr.Constructor = b.resolveConstructorSymbol(callee.Name, args)
			return expr, nil
		}
		if _, ok := b.functions[callee.Name]; ok {
			expr := &FunctionCallExpr{baseExpr: b.base(call), Name: callee.Name, Args: args}
			if symbol, ok := b.functionSymbols[callee.Name]; ok {
				expr.Function = &symbol
			}
			return expr, nil
		}
	case *parser.MemberExpr:
		receiver, err := b.buildExpr(callee.Receiver)
		if err != nil {
			return nil, err
		}
		method := &MethodCallExpr{baseExpr: b.base(call), Receiver: receiver, Method: callee.Name, Args: args}
		method.Target, method.Dispatch = b.resolveMethodTarget(receiver.GetType(), callee.Name, args)
		return method, nil
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
		out[i] = Parameter{
			Name:   param.Name,
			Type:   b.typeFromRef(param.Type),
			Symbol: b.newSymbol(SymbolParameter, param.Name, "", param.Span),
			Span:   param.Span,
		}
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
	case *parser.IndexExpr:
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

func (b *Builder) lookupMethodSymbol(className string, method *parser.MethodDecl) SymbolRef {
	for _, candidate := range b.methodSymbols[className][method.Name] {
		if candidate.decl == method {
			return candidate.symbol
		}
	}
	return SymbolRef{}
}

func (b *Builder) resolveFieldSymbol(receiverType *typecheck.Type, name string) *SymbolRef {
	if receiverType == nil || receiverType.Kind != typecheck.TypeClass {
		return nil
	}
	if fields, ok := b.fieldSymbols[receiverType.Name]; ok {
		if symbol, ok := fields[name]; ok {
			return &symbol
		}
	}
	return nil
}

func (b *Builder) resolveConstructorSymbol(className string, args []Expr) *SymbolRef {
	class, ok := b.classes[className]
	if !ok {
		return nil
	}
	subst := b.substForClass(class, nil)
	for _, candidate := range b.methodSymbols[className]["init"] {
		if b.methodMatches(candidate.decl, class, subst, args) {
			symbol := candidate.symbol
			return &symbol
		}
	}
	return nil
}

func (b *Builder) resolveMethodTarget(receiverType *typecheck.Type, name string, args []Expr) (*SymbolRef, CallDispatch) {
	if receiverType == nil {
		return nil, DispatchStatic
	}
	switch receiverType.Kind {
	case typecheck.TypeClass:
		class, ok := b.classes[receiverType.Name]
		if !ok {
			return nil, DispatchStatic
		}
		subst := b.substForClass(class, receiverType.Args)
		for _, candidate := range b.methodSymbols[receiverType.Name][name] {
			if b.methodMatches(candidate.decl, class, subst, args) {
				symbol := candidate.symbol
				return &symbol, DispatchStatic
			}
		}
	case typecheck.TypeInterface:
		if iface, ok := b.interfaces[receiverType.Name]; ok {
			for _, method := range iface.Methods {
				if method.Name == name {
					symbol := b.newSymbol(SymbolMethod, method.Name, iface.Name, method.Span)
					return &symbol, DispatchVirtual
				}
			}
		}
	}
	return nil, DispatchStatic
}

func (b *Builder) methodMatches(method *parser.MethodDecl, owner *parser.ClassDecl, subst map[string]*typecheck.Type, args []Expr) bool {
	if len(method.Parameters) != len(args) {
		return false
	}
	for i, param := range method.Parameters {
		paramType := b.instantiateTypeRef(param.Type, subst)
		if !sameType(paramType, args[i].GetType()) {
			return false
		}
	}
	return true
}

func (b *Builder) substForClass(class *parser.ClassDecl, args []*typecheck.Type) map[string]*typecheck.Type {
	if len(class.TypeParameters) == 0 {
		return nil
	}
	subst := map[string]*typecheck.Type{}
	for i, param := range class.TypeParameters {
		if i < len(args) && args[i] != nil {
			subst[param.Name] = args[i]
			continue
		}
		subst[param.Name] = &typecheck.Type{Kind: typecheck.TypeParam, Name: param.Name}
	}
	return subst
}

func (b *Builder) instantiateTypeRef(ref *parser.TypeRef, subst map[string]*typecheck.Type) *typecheck.Type {
	if ref == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = b.instantiateTypeRef(param, subst)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: b.instantiateTypeRef(ref.ReturnType, subst),
			},
		}
	}
	if subst != nil {
		if resolved, ok := subst[ref.Name]; ok && len(ref.Arguments) == 0 {
			return resolved
		}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = b.instantiateTypeRef(arg, subst)
	}
	return &typecheck.Type{Kind: b.kindOf(ref.Name), Name: ref.Name, Args: args}
}

func sameType(left, right *typecheck.Type) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Kind == typecheck.TypeUnknown || right.Kind == typecheck.TypeUnknown {
		return true
	}
	if left.Kind != right.Kind || left.Name != right.Name || len(left.Args) != len(right.Args) {
		if left.Kind == typecheck.TypeFunction && right.Kind == typecheck.TypeFunction {
			if left.Signature == nil || right.Signature == nil {
				return left.Signature == right.Signature
			}
			if len(left.Signature.Parameters) != len(right.Signature.Parameters) {
				return false
			}
			for i := range left.Signature.Parameters {
				if !sameType(left.Signature.Parameters[i], right.Signature.Parameters[i]) {
					return false
				}
			}
			return sameType(left.Signature.ReturnType, right.Signature.ReturnType)
		}
		return false
	}
	for i := range left.Args {
		if !sameType(left.Args[i], right.Args[i]) {
			return false
		}
	}
	return true
}
