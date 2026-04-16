package typecheck

import (
	"fmt"

	"a-lang/parser"
	"a-lang/semantic"
)

type Signature struct {
	Parameters []*Type
	ReturnType *Type
}

type binding struct {
	typ     *Type
	mutable bool
}

type scope map[string]binding
type typeScope map[string]TypeKind

type fieldInfo struct {
	decl parser.FieldDecl
}

type methodInfo struct {
	decl *parser.MethodDecl
}

type interfaceMethodInfo struct {
	decl parser.InterfaceMethod
}

type classInfo struct {
	decl        *parser.ClassDecl
	fields      map[string]fieldInfo
	methods     map[string]methodInfo
	constructor *parser.MethodDecl
}

type interfaceInfo struct {
	decl    *parser.InterfaceDecl
	methods map[string]interfaceMethodInfo
}

type Result struct {
	Diagnostics []semantic.Diagnostic
	ExprTypes   map[parser.Expr]*Type
}

type Checker struct {
	diagnostics   []semantic.Diagnostic
	scopes        []scope
	typeScopes    []typeScope
	globals       map[string]binding
	functions     map[string]Signature
	classes       map[string]classInfo
	interfaces    map[string]interfaceInfo
	returnTypes   []*Type
	exprTypes     map[parser.Expr]*Type
	currentClass  *parser.ClassDecl
	currentMethod *parser.MethodDecl
}

type typeLookup interface {
	kindOf(name string) TypeKind
}

func Analyze(program *parser.Program) Result {
	c := &Checker{
		globals:    map[string]binding{},
		functions:  map[string]Signature{},
		classes:    map[string]classInfo{},
		interfaces: map[string]interfaceInfo{},
		exprTypes:  map[parser.Expr]*Type{},
	}
	c.collectDecls(program)
	c.checkProgram(program)
	return Result{Diagnostics: c.diagnostics, ExprTypes: c.exprTypes}
}

func (c *Checker) collectDecls(program *parser.Program) {
	for _, decl := range program.Interfaces {
		info := interfaceInfo{
			decl:    decl,
			methods: map[string]interfaceMethodInfo{},
		}
		for _, method := range decl.Methods {
			info.methods[method.Name] = interfaceMethodInfo{decl: method}
		}
		c.interfaces[decl.Name] = info
	}
	for _, decl := range program.Classes {
		info := classInfo{
			decl:    decl,
			fields:  map[string]fieldInfo{},
			methods: map[string]methodInfo{},
		}
		for _, field := range decl.Fields {
			info.fields[field.Name] = fieldInfo{decl: field}
		}
		for _, method := range decl.Methods {
			info.methods[method.Name] = methodInfo{decl: method}
			if method.Constructor {
				info.constructor = method
			}
		}
		c.classes[decl.Name] = info
	}
	for _, fn := range program.Functions {
		params := make([]*Type, len(fn.Parameters))
		for i, param := range fn.Parameters {
			params[i] = fromTypeRef(param.Type, c)
		}
		c.functions[fn.Name] = Signature{
			Parameters: params,
			ReturnType: fromTypeRef(fn.ReturnType, c),
		}
	}
}

func (c *Checker) checkProgram(program *parser.Program) {
	c.checkGlobals(program.Statements)
	for _, fn := range program.Functions {
		c.checkFunction(fn)
	}
	for _, decl := range program.Classes {
		c.checkClass(decl)
	}
}

func (c *Checker) checkGlobals(statements []parser.Statement) {
	c.pushScope()
	defer c.popScope()
	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *parser.ValStmt:
			for i, bindingDecl := range s.Bindings {
				valueType := unknownType
				if i < len(s.Values) {
					valueType = c.checkExpr(s.Values[i])
				}
				declType := valueType
				if bindingDecl.Type != nil {
					declType = c.resolveDeclaredType(bindingDecl.Type)
					c.requireAssignable(valueType, declType, bindingDecl.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+declType.String())
				}
				c.globals[bindingDecl.Name] = binding{typ: declType, mutable: bindingDecl.Mutable}
				c.define(bindingDecl.Name, declType, bindingDecl.Mutable)
			}
		default:
			c.addDiagnostic("unsupported_top_level", "unsupported top-level statement for type checking", stmtSpan(stmt))
		}
	}
}

func (c *Checker) checkFunction(fn *parser.FunctionDecl) {
	c.pushScope()
	defer c.popScope()
	expectedReturn := fromTypeRef(fn.ReturnType, c)
	c.returnTypes = append(c.returnTypes, expectedReturn)
	defer func() { c.returnTypes = c.returnTypes[:len(c.returnTypes)-1] }()

	for _, param := range fn.Parameters {
		c.define(param.Name, fromTypeRef(param.Type, c), false)
	}
	c.checkBlock(fn.Body)
}

func (c *Checker) checkClass(decl *parser.ClassDecl) {
	info := c.classes[decl.Name]

	c.pushTypeScope()
	defer c.popTypeScope()
	for _, param := range decl.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}

	for _, field := range decl.Fields {
		c.resolveDeclaredType(field.Type)
	}
	for _, method := range decl.Methods {
		c.checkMethod(method, decl)
	}
	for _, impl := range decl.Implements {
		c.checkInterfaceImplementation(info, impl)
	}
}

func (c *Checker) checkMethod(method *parser.MethodDecl, owner *parser.ClassDecl) {
	c.pushScope()
	defer c.popScope()

	prevClass := c.currentClass
	prevMethod := c.currentMethod
	c.currentClass = owner
	c.currentMethod = method
	defer func() {
		c.currentClass = prevClass
		c.currentMethod = prevMethod
	}()

	classArgs := make([]*Type, len(owner.TypeParameters))
	for i, param := range owner.TypeParameters {
		classArgs[i] = &Type{Kind: TypeParam, Name: param.Name}
	}
	c.define("this", &Type{Kind: TypeClass, Name: owner.Name, Args: classArgs}, false)

	expectedReturn := unknownType
	if !method.Constructor {
		expectedReturn = c.resolveDeclaredType(method.ReturnType)
	}
	c.returnTypes = append(c.returnTypes, expectedReturn)
	defer func() { c.returnTypes = c.returnTypes[:len(c.returnTypes)-1] }()

	for _, param := range method.Parameters {
		c.define(param.Name, c.resolveDeclaredType(param.Type), false)
	}
	c.checkBlock(method.Body)
}

func (c *Checker) checkInterfaceImplementation(class classInfo, impl *parser.TypeRef) {
	if impl == nil {
		return
	}
	iface, ok := c.interfaces[impl.Name]
	if !ok {
		return
	}
	subst := map[string]*Type{}
	for i, param := range iface.decl.TypeParameters {
		if i < len(impl.Arguments) {
			subst[param.Name] = c.instantiateTypeRef(impl.Arguments[i], nil)
		}
	}

	for _, method := range iface.decl.Methods {
		classMethod, ok := class.methods[method.Name]
		if !ok {
			c.addDiagnostic("interface_not_implemented", "class '"+class.decl.Name+"' does not implement method '"+method.Name+"'", class.decl.Span)
			continue
		}
		expected := c.instantiateInterfaceMethodSignature(method, subst)
		actual := c.instantiateMethodSignature(classMethod.decl, class.decl, nil)
		c.compareSignatures(actual, expected, classMethod.decl.Span, method.Name)
	}
}

func (c *Checker) compareSignatures(actual, expected Signature, span parser.Span, name string) {
	if len(actual.Parameters) != len(expected.Parameters) {
		c.addDiagnostic("interface_not_implemented", "method '"+name+"' has wrong parameter count", span)
		return
	}
	for i := range actual.Parameters {
		if !sameType(actual.Parameters[i], expected.Parameters[i]) {
			c.addDiagnostic("interface_not_implemented", "method '"+name+"' parameter types do not match interface", span)
			return
		}
	}
	if !sameType(actual.ReturnType, expected.ReturnType) {
		c.addDiagnostic("interface_not_implemented", "method '"+name+"' return type does not match interface", span)
	}
}

func (c *Checker) checkBlock(block *parser.BlockStmt) {
	c.pushScope()
	defer c.popScope()
	for _, stmt := range block.Statements {
		c.checkStmt(stmt)
	}
}

func (c *Checker) checkStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		for i, bindingDecl := range s.Bindings {
			valueType := unknownType
			if i < len(s.Values) {
				valueType = c.checkExpr(s.Values[i])
			}
			declType := valueType
			if bindingDecl.Type != nil {
				declType = c.resolveDeclaredType(bindingDecl.Type)
				c.requireAssignable(valueType, declType, bindingDecl.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+declType.String())
			}
			c.define(bindingDecl.Name, declType, bindingDecl.Mutable)
		}
	case *parser.AssignmentStmt:
		targetType, mutable := c.checkAssignmentTarget(s.Target, s.Span)
		valueType := c.checkExpr(s.Value)
		if !mutable {
			return
		}
		if s.Operator != "=" {
			op := s.Operator[:len(s.Operator)-1]
			c.checkBinaryOperation(targetType, valueType, op, s.Span)
		}
		c.requireAssignable(valueType, targetType, s.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+targetType.String())
	case *parser.IfStmt:
		condType := c.checkExpr(s.Condition)
		c.requireAssignable(condType, builtin("Bool"), exprSpan(s.Condition), "invalid_condition_type", "if condition must be Bool")
		c.checkBlock(s.Then)
		if s.ElseIf != nil {
			c.checkStmt(s.ElseIf)
		}
		if s.Else != nil {
			c.checkBlock(s.Else)
		}
	case *parser.ForStmt:
		c.pushScope()
		for _, binding := range s.Bindings {
			iterType := c.checkExpr(binding.Iterable)
			elemType := c.iterableElementType(iterType)
			c.define(binding.Name, elemType, false)
		}
		if s.Body != nil {
			c.checkBlockStatements(s.Body.Statements)
		}
		if s.YieldBody != nil {
			c.checkBlockStatements(s.YieldBody.Statements)
		}
		c.popScope()
	case *parser.ReturnStmt:
		valueType := c.checkExpr(s.Value)
		if len(c.returnTypes) == 0 {
			c.addDiagnostic("invalid_return", "return used outside callable body", s.Span)
			return
		}
		expected := c.returnTypes[len(c.returnTypes)-1]
		if !isUnknown(expected) {
			c.requireAssignable(valueType, expected, s.Span, "invalid_return_type", "cannot return "+valueType.String()+" from function returning "+expected.String())
		}
	case *parser.ExprStmt:
		c.checkExpr(s.Expr)
	}
}

func (c *Checker) checkBlockStatements(statements []parser.Statement) {
	c.pushScope()
	defer c.popScope()
	for _, stmt := range statements {
		c.checkStmt(stmt)
	}
}

func (c *Checker) checkExpr(expr parser.Expr) *Type {
	var result *Type
	switch e := expr.(type) {
	case *parser.Identifier:
		if binding, ok := c.lookup(e.Name); ok {
			result = binding.typ
			break
		}
		if sig, ok := c.functions[e.Name]; ok {
			result = functionType(e.Name, sig)
			break
		}
		if _, ok := c.classes[e.Name]; ok {
			result = &Type{Kind: TypeClass, Name: e.Name}
			break
		}
		if isBuiltinValue(e.Name) {
			result = unknownType
			break
		}
		c.addDiagnostic("undefined_name", "undefined name '"+e.Name+"'", e.Span)
		result = unknownType
	case *parser.IntegerLiteral:
		result = builtin("Int")
	case *parser.FloatLiteral:
		result = builtin("Float")
	case *parser.RuneLiteral:
		result = builtin("Rune")
	case *parser.BoolLiteral:
		result = builtin("Bool")
	case *parser.StringLiteral:
		result = builtin("String")
	case *parser.ListLiteral:
		if len(e.Elements) == 0 {
			result = &Type{Kind: TypeBuiltin, Name: "List", Args: []*Type{unknownType}}
			break
		}
		elemType := c.checkExpr(e.Elements[0])
		for _, elem := range e.Elements[1:] {
			nextType := c.checkExpr(elem)
			if !sameType(elemType, nextType) {
				c.addDiagnostic("type_mismatch", "list literal elements must have the same type", exprSpan(elem))
			}
		}
		result = &Type{Kind: TypeBuiltin, Name: "List", Args: []*Type{elemType}}
	case *parser.MapLiteral:
		result = &Type{Kind: TypeBuiltin, Name: "Map", Args: []*Type{unknownType, unknownType}}
	case *parser.GroupExpr:
		result = c.checkExpr(e.Inner)
	case *parser.UnaryExpr:
		right := c.checkExpr(e.Right)
		switch e.Operator {
		case "!":
			c.requireAssignable(right, builtin("Bool"), e.Span, "invalid_unary_operand", "operator ! requires Bool")
			result = builtin("Bool")
		case "-":
			if !isNumeric(right) {
				c.addDiagnostic("invalid_unary_operand", "operator - requires numeric operand", e.Span)
			}
			result = right
		default:
			result = unknownType
		}
	case *parser.BinaryExpr:
		left := c.checkExpr(e.Left)
		right := c.checkExpr(e.Right)
		result = c.checkBinaryOperation(left, right, e.Operator, e.Span)
	case *parser.CallExpr:
		result = c.checkCall(e)
	case *parser.MemberExpr:
		result = c.checkMemberExpr(e)
	case *parser.LambdaExpr:
		result = unknownType
	case *parser.PlaceholderExpr:
		result = unknownType
	default:
		result = unknownType
	}
	if expr != nil {
		c.exprTypes[expr] = result
	}
	return result
}

func (c *Checker) checkCall(call *parser.CallExpr) *Type {
	if ident, ok := call.Callee.(*parser.Identifier); ok {
		if class, ok := c.classes[ident.Name]; ok {
			return c.checkConstructorCall(class, call)
		}
	}

	calleeType := c.checkExpr(call.Callee)
	if calleeType.Kind != TypeFunction || calleeType.Signature == nil {
		for _, arg := range call.Args {
			c.checkExpr(arg)
		}
		return unknownType
	}

	sig := *calleeType.Signature
	if len(call.Args) != len(sig.Parameters) {
		c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %d arguments, got %d", len(sig.Parameters), len(call.Args)), call.Span)
	}
	for i, arg := range call.Args {
		argType := c.checkExpr(arg)
		if i < len(sig.Parameters) {
			c.requireAssignable(argType, sig.Parameters[i], exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+sig.Parameters[i].String())
		}
	}
	return sig.ReturnType
}

func (c *Checker) checkConstructorCall(class classInfo, call *parser.CallExpr) *Type {
	classType := &Type{Kind: TypeClass, Name: class.decl.Name}
	if ctor := class.constructor; ctor != nil {
		sig := c.instantiateMethodSignature(ctor, class.decl, constructorTypeArgs(class.decl, call.Callee))
		if len(call.Args) != len(sig.Parameters) {
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("constructor '%s' expects %d arguments, got %d", class.decl.Name, len(sig.Parameters), len(call.Args)), call.Span)
		}
		for i, arg := range call.Args {
			argType := c.checkExpr(arg)
			if i < len(sig.Parameters) {
				c.requireAssignable(argType, sig.Parameters[i], exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+sig.Parameters[i].String())
			}
		}
	}
	if ident, ok := call.Callee.(*parser.Identifier); ok {
		if refType, ok := c.lookupTypeInstance(ident.Name); ok {
			classType = refType
		}
	}
	return classType
}

func (c *Checker) checkMemberExpr(expr *parser.MemberExpr) *Type {
	receiverType := c.checkExpr(expr.Receiver)
	if memberType, ok := c.lookupMember(receiverType, expr.Name, expr.Span); ok {
		return memberType
	}
	if expr.Name == "toString" {
		return functionType("toString", Signature{ReturnType: builtin("String")})
	}
	return unknownType
}

func (c *Checker) lookupMember(receiver *Type, name string, span parser.Span) (*Type, bool) {
	if isUnknown(receiver) {
		return unknownType, true
	}
	switch receiver.Kind {
	case TypeClass:
		info, ok := c.classes[receiver.Name]
		if !ok {
			return unknownType, false
		}
		subst := c.substForDecl(info.decl.TypeParameters, receiver.Args)
		if field, ok := info.fields[name]; ok {
			if field.decl.Private && !c.canAccessPrivate(info.decl) {
				c.addDiagnostic("private_access", "cannot access private field '"+name+"' outside class '"+info.decl.Name+"'", span)
				return unknownType, true
			}
			return c.instantiateTypeRef(field.decl.Type, subst), true
		}
		if method, ok := info.methods[name]; ok {
			if method.decl.Private && !c.canAccessPrivate(info.decl) {
				c.addDiagnostic("private_access", "cannot access private method '"+name+"' outside class '"+info.decl.Name+"'", span)
				return unknownType, true
			}
			sig := c.instantiateMethodSignature(method.decl, info.decl, subst)
			return functionType(name, sig), true
		}
	case TypeInterface:
		info, ok := c.interfaces[receiver.Name]
		if !ok {
			return unknownType, false
		}
		subst := c.substForDecl(info.decl.TypeParameters, receiver.Args)
		if method, ok := info.methods[name]; ok {
			sig := c.instantiateInterfaceMethodSignature(method.decl, subst)
			return functionType(name, sig), true
		}
	}
	return unknownType, false
}

func (c *Checker) checkBinaryOperation(left, right *Type, op string, span parser.Span) *Type {
	switch op {
	case "+", "-", "*", "/", "%":
		if !isNumeric(left) || !isNumeric(right) {
			c.addDiagnostic("invalid_binary_operand", "arithmetic operators require numeric operands", span)
			return unknownType
		}
		if !sameType(left, right) {
			c.addDiagnostic("type_mismatch", "arithmetic operands must have the same type", span)
		}
		return left
	case "&&", "||":
		c.requireAssignable(left, builtin("Bool"), span, "invalid_binary_operand", "logical operators require Bool operands")
		c.requireAssignable(right, builtin("Bool"), span, "invalid_binary_operand", "logical operators require Bool operands")
		return builtin("Bool")
	case "==", "!=":
		if !sameType(left, right) {
			c.addDiagnostic("type_mismatch", "comparison operands must have the same type", span)
		}
		return builtin("Bool")
	case "<", "<=", ">", ">=":
		if !isOrdered(left) || !isOrdered(right) {
			c.addDiagnostic("invalid_binary_operand", "comparison operators require ordered operands", span)
			return builtin("Bool")
		}
		if !sameType(left, right) {
			c.addDiagnostic("type_mismatch", "comparison operands must have the same type", span)
		}
		return builtin("Bool")
	case ":":
		return unknownType
	case "..":
		if !sameType(left, builtin("Int")) || !sameType(right, builtin("Int")) {
			c.addDiagnostic("invalid_binary_operand", "range operands must be Int", span)
		}
		return &Type{Kind: TypeBuiltin, Name: "List", Args: []*Type{builtin("Int")}}
	default:
		return unknownType
	}
}

func (c *Checker) checkAssignmentTarget(target parser.Expr, span parser.Span) (*Type, bool) {
	switch t := target.(type) {
	case *parser.Identifier:
		b, ok := c.lookup(t.Name)
		if !ok {
			c.addDiagnostic("undefined_name", "undefined name '"+t.Name+"'", t.Span)
			return unknownType, false
		}
		if !b.mutable {
			c.addDiagnostic("assign_immutable", "cannot assign to immutable binding '"+t.Name+"'", t.Span)
		}
		return b.typ, b.mutable
	case *parser.MemberExpr:
		receiverType := c.checkExpr(t.Receiver)
		memberType, mutable, ok := c.checkFieldAssignment(t, receiverType)
		if ok {
			return memberType, mutable
		}
		return unknownType, false
	default:
		c.addDiagnostic("invalid_assignment_target", "invalid assignment target", span)
		return unknownType, false
	}
}

func (c *Checker) checkFieldAssignment(expr *parser.MemberExpr, receiverType *Type) (*Type, bool, bool) {
	if isUnknown(receiverType) {
		return unknownType, false, true
	}
	if receiverType.Kind != TypeClass {
		c.addDiagnostic("invalid_assignment_target", "member assignment expects class instance", expr.Span)
		return unknownType, false, true
	}
	info, ok := c.classes[receiverType.Name]
	if !ok {
		c.addDiagnostic("invalid_assignment_target", "member assignment expects class instance", expr.Span)
		return unknownType, false, true
	}
	field, ok := info.fields[expr.Name]
	if !ok {
		if _, hasMethod := info.methods[expr.Name]; hasMethod {
			if info.methods[expr.Name].decl.Private && !c.canAccessPrivate(info.decl) {
				c.addDiagnostic("private_access", "cannot access private method '"+expr.Name+"' outside class '"+info.decl.Name+"'", expr.Span)
				return unknownType, false, true
			}
			c.addDiagnostic("invalid_assignment_target", "cannot assign to method '"+expr.Name+"'", expr.Span)
			return unknownType, false, true
		}
		c.addDiagnostic("unknown_member", "unknown member '"+expr.Name+"'", expr.Span)
		return unknownType, false, true
	}
	if field.decl.Private && !c.canAccessPrivate(info.decl) {
		c.addDiagnostic("private_access", "cannot access private field '"+expr.Name+"' outside class '"+info.decl.Name+"'", expr.Span)
		return unknownType, false, true
	}
	fieldType := c.instantiateTypeRef(field.decl.Type, c.substForDecl(info.decl.TypeParameters, receiverType.Args))
	if field.decl.Mutable {
		return fieldType, true, true
	}
	if c.canAssignImmutableField(expr, info.decl) {
		return fieldType, true, true
	}
	c.addDiagnostic("assign_immutable", "cannot assign to immutable field '"+expr.Name+"' outside init", expr.Span)
	return fieldType, false, true
}

func (c *Checker) canAssignImmutableField(expr *parser.MemberExpr, owner *parser.ClassDecl) bool {
	if c.currentClass == nil || c.currentMethod == nil || !c.currentMethod.Constructor {
		return false
	}
	if c.currentClass.Name != owner.Name {
		return false
	}
	ident, ok := expr.Receiver.(*parser.Identifier)
	return ok && ident.Name == "this"
}

func (c *Checker) canAccessPrivate(owner *parser.ClassDecl) bool {
	return c.currentClass != nil && c.currentClass.Name == owner.Name
}

func (c *Checker) resolveDeclaredType(ref *parser.TypeRef) *Type {
	return c.instantiateTypeRef(ref, nil)
}

func (c *Checker) instantiateTypeRef(ref *parser.TypeRef, subst map[string]*Type) *Type {
	if ref == nil {
		return unknownType
	}
	if subst != nil {
		if resolved, ok := subst[ref.Name]; ok && len(ref.Arguments) == 0 {
			return resolved
		}
	}
	args := make([]*Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = c.instantiateTypeRef(arg, subst)
	}
	kind := c.kindOf(ref.Name)
	if kind == "" {
		kind = TypeUnknown
	}
	return &Type{Kind: kind, Name: ref.Name, Args: args}
}

func (c *Checker) instantiateMethodSignature(method *parser.MethodDecl, owner *parser.ClassDecl, subst map[string]*Type) Signature {
	effective := mergeSubst(subst, c.substForDecl(owner.TypeParameters, nil))
	params := make([]*Type, len(method.Parameters))
	for i, param := range method.Parameters {
		params[i] = c.instantiateTypeRef(param.Type, effective)
	}
	result := unknownType
	if !method.Constructor {
		result = c.instantiateTypeRef(method.ReturnType, effective)
	}
	return Signature{Parameters: params, ReturnType: result}
}

func (c *Checker) instantiateInterfaceMethodSignature(method parser.InterfaceMethod, subst map[string]*Type) Signature {
	params := make([]*Type, len(method.Parameters))
	for i, param := range method.Parameters {
		params[i] = c.instantiateTypeRef(param.Type, subst)
	}
	return Signature{
		Parameters: params,
		ReturnType: c.instantiateTypeRef(method.ReturnType, subst),
	}
}

func (c *Checker) substForDecl(params []parser.TypeParameter, args []*Type) map[string]*Type {
	if len(params) == 0 {
		return nil
	}
	result := map[string]*Type{}
	for i, param := range params {
		if i < len(args) && args[i] != nil {
			result[param.Name] = args[i]
		} else {
			result[param.Name] = &Type{Kind: TypeParam, Name: param.Name}
		}
	}
	return result
}

func mergeSubst(primary, fallback map[string]*Type) map[string]*Type {
	if primary == nil && fallback == nil {
		return nil
	}
	result := map[string]*Type{}
	for k, v := range fallback {
		result[k] = v
	}
	for k, v := range primary {
		result[k] = v
	}
	return result
}

func constructorTypeArgs(owner *parser.ClassDecl, callee parser.Expr) map[string]*Type {
	_ = callee
	if len(owner.TypeParameters) == 0 {
		return nil
	}
	result := map[string]*Type{}
	for _, param := range owner.TypeParameters {
		result[param.Name] = &Type{Kind: TypeParam, Name: param.Name}
	}
	return result
}

func (c *Checker) lookupTypeInstance(name string) (*Type, bool) {
	if _, ok := c.classes[name]; ok {
		return &Type{Kind: TypeClass, Name: name}, true
	}
	if _, ok := c.interfaces[name]; ok {
		return &Type{Kind: TypeInterface, Name: name}, true
	}
	return nil, false
}

func (c *Checker) iterableElementType(t *Type) *Type {
	if isUnknown(t) {
		return unknownType
	}
	if (t.Name == "List" || t.Name == "Set" || t.Name == "Array") && len(t.Args) == 1 {
		return t.Args[0]
	}
	return unknownType
}

func (c *Checker) requireAssignable(actual, expected *Type, span parser.Span, code, message string) {
	if isUnknown(actual) || isUnknown(expected) {
		return
	}
	if !sameType(actual, expected) {
		c.addDiagnostic(code, message, span)
	}
}

func (c *Checker) addDiagnostic(code, message string, span parser.Span) {
	c.diagnostics = append(c.diagnostics, semantic.Diagnostic{Code: code, Message: message, Span: span})
}

func (c *Checker) define(name string, typ *Type, mutable bool) {
	c.currentScope()[name] = binding{typ: typ, mutable: mutable}
}

func (c *Checker) lookup(name string) (binding, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if value, ok := c.scopes[i][name]; ok {
			return value, true
		}
	}
	if value, ok := c.globals[name]; ok {
		return value, true
	}
	return binding{}, false
}

func (c *Checker) pushScope() { c.scopes = append(c.scopes, scope{}) }
func (c *Checker) popScope()  { c.scopes = c.scopes[:len(c.scopes)-1] }

func (c *Checker) currentScope() scope {
	if len(c.scopes) == 0 {
		c.pushScope()
	}
	return c.scopes[len(c.scopes)-1]
}

func (c *Checker) pushTypeScope() { c.typeScopes = append(c.typeScopes, typeScope{}) }
func (c *Checker) popTypeScope()  { c.typeScopes = c.typeScopes[:len(c.typeScopes)-1] }

func (c *Checker) currentTypeScope() typeScope {
	if len(c.typeScopes) == 0 {
		c.pushTypeScope()
	}
	return c.typeScopes[len(c.typeScopes)-1]
}

func (c *Checker) kindOf(name string) TypeKind {
	for i := len(c.typeScopes) - 1; i >= 0; i-- {
		if kind, ok := c.typeScopes[i][name]; ok {
			return kind
		}
	}
	if isBuiltinType(name) {
		return TypeBuiltin
	}
	if _, ok := c.classes[name]; ok {
		return TypeClass
	}
	if _, ok := c.interfaces[name]; ok {
		return TypeInterface
	}
	return ""
}

func functionType(name string, sig Signature) *Type {
	return &Type{Kind: TypeFunction, Name: name, Signature: &sig}
}

func builtin(name string) *Type { return &Type{Kind: TypeBuiltin, Name: name} }

func isBuiltinType(name string) bool {
	switch name {
	case "Int", "Int64", "Bool", "String", "Rune", "Float", "Float64", "List", "Set", "Array", "Map":
		return true
	default:
		return false
	}
}

func isBuiltinValue(name string) bool {
	switch name {
	case "Map", "Set", "Array", "range":
		return true
	default:
		return false
	}
}

func isNumeric(t *Type) bool {
	if isUnknown(t) {
		return true
	}
	switch t.Name {
	case "Int", "Int64", "Float", "Float64":
		return true
	default:
		return false
	}
}

func isOrdered(t *Type) bool {
	if isUnknown(t) {
		return true
	}
	switch t.Name {
	case "Int", "Int64", "Float", "Float64", "String", "Rune":
		return true
	default:
		return false
	}
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
	case *parser.MapLiteral:
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

func stmtSpan(stmt parser.Statement) parser.Span {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		return s.Span
	case *parser.AssignmentStmt:
		return s.Span
	case *parser.IfStmt:
		return s.Span
	case *parser.ForStmt:
		return s.Span
	case *parser.ReturnStmt:
		return s.Span
	case *parser.BreakStmt:
		return s.Span
	case *parser.ExprStmt:
		return s.Span
	default:
		return parser.Span{}
	}
}
