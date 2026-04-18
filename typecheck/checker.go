package typecheck

import (
	"fmt"

	"a-lang/parser"
	"a-lang/semantic"
)

type Signature struct {
	Parameters []*Type
	ReturnType *Type
	Variadic   bool
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
	decl         *parser.ClassDecl
	fields       map[string]fieldInfo
	methods      map[string][]methodInfo
	constructors []*parser.MethodDecl
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
	lambdaScopes  []int
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
	c.installBuiltinInterfaces()
	c.collectDecls(program)
	c.checkProgram(program)
	return Result{Diagnostics: c.diagnostics, ExprTypes: c.exprTypes}
}

func (c *Checker) installBuiltinInterfaces() {
	for name, info := range builtinInterfaceInfos() {
		c.interfaces[name] = info
	}
}

func builtinInterfaceInfos() map[string]interfaceInfo {
	out := map[string]interfaceInfo{}

	listDecl := &parser.InterfaceDecl{
		Name:           "List",
		TypeParameters: []parser.TypeParameter{{Name: "T"}},
		Methods: []parser.InterfaceMethod{
			{Name: "append", Parameters: []parser.Parameter{{Name: "value", Type: namedType("T")}}, ReturnType: genericType("List", "T")},
			{Name: "get", Parameters: []parser.Parameter{{Name: "index", Type: namedType("Int")}}, ReturnType: genericType("Option", "T")},
			{Name: "size", Parameters: nil, ReturnType: namedType("Int")},
		},
	}
	out["List"] = interfaceInfo{decl: listDecl, methods: map[string]interfaceMethodInfo{
		"append": {decl: listDecl.Methods[0]},
		"get":    {decl: listDecl.Methods[1]},
		"size":   {decl: listDecl.Methods[2]},
	}}

	setDecl := &parser.InterfaceDecl{
		Name:           "Set",
		TypeParameters: []parser.TypeParameter{{Name: "T"}},
		Methods: []parser.InterfaceMethod{
			{Name: "add", Parameters: []parser.Parameter{{Name: "value", Type: namedType("T")}}, ReturnType: genericType("Set", "T")},
			{Name: "contains", Parameters: []parser.Parameter{{Name: "value", Type: namedType("T")}}, ReturnType: namedType("Bool")},
			{Name: "size", Parameters: nil, ReturnType: namedType("Int")},
		},
	}
	out["Set"] = interfaceInfo{decl: setDecl, methods: map[string]interfaceMethodInfo{
		"add":      {decl: setDecl.Methods[0]},
		"contains": {decl: setDecl.Methods[1]},
		"size":     {decl: setDecl.Methods[2]},
	}}

	mapDecl := &parser.InterfaceDecl{
		Name:           "Map",
		TypeParameters: []parser.TypeParameter{{Name: "K"}, {Name: "V"}},
		Methods: []parser.InterfaceMethod{
			{Name: "set", Parameters: []parser.Parameter{{Name: "key", Type: namedType("K")}, {Name: "value", Type: namedType("V")}}, ReturnType: genericType("Map", "K", "V")},
			{Name: "get", Parameters: []parser.Parameter{{Name: "key", Type: namedType("K")}}, ReturnType: genericType("Option", "V")},
			{Name: "contains", Parameters: []parser.Parameter{{Name: "key", Type: namedType("K")}}, ReturnType: namedType("Bool")},
			{Name: "size", Parameters: nil, ReturnType: namedType("Int")},
		},
	}
	out["Map"] = interfaceInfo{decl: mapDecl, methods: map[string]interfaceMethodInfo{
		"set":      {decl: mapDecl.Methods[0]},
		"get":      {decl: mapDecl.Methods[1]},
		"contains": {decl: mapDecl.Methods[2]},
		"size":     {decl: mapDecl.Methods[3]},
	}}

	termDecl := &parser.InterfaceDecl{
		Name: "Term",
		Methods: []parser.InterfaceMethod{
			{Name: "print", Parameters: []parser.Parameter{{Name: "value", Type: namedType("String")}}, ReturnType: namedType("Term")},
			{Name: "println", Parameters: []parser.Parameter{{Name: "value", Type: namedType("String"), Variadic: true}}, ReturnType: namedType("Term")},
		},
	}
	out["Term"] = interfaceInfo{decl: termDecl, methods: map[string]interfaceMethodInfo{
		"print":   {decl: termDecl.Methods[0]},
		"println": {decl: termDecl.Methods[1]},
	}}

	optionDecl := &parser.InterfaceDecl{
		Name:           "Option",
		TypeParameters: []parser.TypeParameter{{Name: "T"}},
		Methods: []parser.InterfaceMethod{
			{Name: "isSet", Parameters: nil, ReturnType: namedType("Bool")},
			{Name: "get", Parameters: nil, ReturnType: namedType("T")},
			{Name: "getOr", Parameters: []parser.Parameter{{Name: "defaultValue", Type: namedType("T")}}, ReturnType: namedType("T")},
		},
	}
	out["Option"] = interfaceInfo{decl: optionDecl, methods: map[string]interfaceMethodInfo{
		"isSet": {decl: optionDecl.Methods[0]},
		"get":   {decl: optionDecl.Methods[1]},
		"getOr": {decl: optionDecl.Methods[2]},
	}}

	return out
}

func namedType(name string) *parser.TypeRef {
	return &parser.TypeRef{Name: name}
}

func genericType(name string, args ...string) *parser.TypeRef {
	ref := &parser.TypeRef{Name: name}
	for _, arg := range args {
		ref.Arguments = append(ref.Arguments, namedType(arg))
	}
	return ref
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
			methods: map[string][]methodInfo{},
		}
		for _, field := range decl.Fields {
			info.fields[field.Name] = fieldInfo{decl: field}
		}
		for _, method := range decl.Methods {
			info.methods[method.Name] = append(info.methods[method.Name], methodInfo{decl: method})
			if method.Constructor {
				info.constructors = append(info.constructors, method)
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
			Variadic:   len(fn.Parameters) > 0 && fn.Parameters[len(fn.Parameters)-1].Variadic,
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
				hasValue := i < len(s.Values) && s.Values[i] != nil
				if hasValue {
					expected := unknownType
					if bindingDecl.Type != nil {
						expected = c.resolveDeclaredType(bindingDecl.Type)
					}
					valueType = c.checkExprWithExpected(s.Values[i], expected)
				}
				declType := valueType
				if bindingDecl.Type != nil {
					declType = c.resolveDeclaredType(bindingDecl.Type)
					if hasValue {
						c.requireAssignable(valueType, declType, bindingDecl.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+declType.String())
					}
				} else if !hasValue {
					c.addDiagnostic("invalid_deferred", "deferred binding '"+bindingDecl.Name+"' requires an explicit type", bindingDecl.Span)
					declType = unknownType
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
		paramType := fromTypeRef(param.Type, c)
		if param.Variadic {
			paramType = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{paramType}}
		}
		c.define(param.Name, paramType, false)
	}
	implicitReturn := c.checkBlock(fn.Body)
	if fn.ReturnType != nil && !isUnknown(implicitReturn) {
		c.requireAssignable(implicitReturn, expectedReturn, fn.Body.Span, "invalid_return_type", "cannot implicitly return "+implicitReturn.String()+" from function returning "+expectedReturn.String())
	}
}

func (c *Checker) checkClass(decl *parser.ClassDecl) {
	info := c.classes[decl.Name]

	c.pushTypeScope()
	defer c.popTypeScope()
	for _, param := range decl.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}

	for _, field := range decl.Fields {
		fieldType := c.resolveDeclaredType(field.Type)
		if field.Initializer != nil {
			valueType := c.checkExprWithExpected(field.Initializer, fieldType)
			c.requireAssignable(valueType, fieldType, exprSpan(field.Initializer), "type_mismatch", "cannot assign "+valueType.String()+" to "+fieldType.String())
		}
	}
	c.checkConstructorRules(info)
	for _, method := range decl.Methods {
		c.checkMethod(method, decl)
	}
	for _, impl := range decl.Implements {
		if impl.Name == "Eq" {
			c.checkEqImplementation(info, impl)
			continue
		}
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
		paramType := c.resolveDeclaredType(param.Type)
		if param.Variadic {
			paramType = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{paramType}}
		}
		c.define(param.Name, paramType, false)
	}
	implicitReturn := c.checkBlock(method.Body)
	if !method.Constructor && method.ReturnType != nil && !isUnknown(implicitReturn) {
		c.requireAssignable(implicitReturn, expectedReturn, method.Body.Span, "invalid_return_type", "cannot implicitly return "+implicitReturn.String()+" from method returning "+expectedReturn.String())
	}
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
		classMethods, ok := class.methods[method.Name]
		if !ok || len(classMethods) == 0 {
			c.addDiagnostic("interface_not_implemented", "class '"+class.decl.Name+"' does not implement method '"+method.Name+"'", class.decl.Span)
			continue
		}
		expected := c.instantiateInterfaceMethodSignature(method, subst)
		classMethod, ok := c.findMatchingMethodOverload(class, method.Name, expected.Parameters)
		if !ok {
			c.addDiagnostic("interface_not_implemented", "class '"+class.decl.Name+"' does not implement method '"+method.Name+"' with matching signature", class.decl.Span)
			continue
		}
		actual := c.instantiateMethodSignature(classMethod.decl, class.decl, nil)
		c.compareSignatures(actual, expected, classMethod.decl.Span, method.Name)
	}
}

func (c *Checker) checkEqImplementation(class classInfo, impl *parser.TypeRef) {
	if len(impl.Arguments) != 1 {
		c.addDiagnostic("interface_not_implemented", "Eq requires exactly one type argument", impl.Span)
		return
	}
	expectedSelf := c.instantiateTypeRef(impl.Arguments[0], c.substForDecl(class.decl.TypeParameters, nil))
	classMethods, ok := class.methods["equals"]
	if !ok || len(classMethods) == 0 {
		c.addDiagnostic("interface_not_implemented", "class '"+class.decl.Name+"' does not implement method 'equals' required by Eq", class.decl.Span)
		return
	}
	method, ok := c.findMatchingMethodOverload(class, "equals", []*Type{expectedSelf})
	if !ok {
		c.addDiagnostic("interface_not_implemented", "class '"+class.decl.Name+"' does not implement method 'equals' with signature required by Eq", class.decl.Span)
		return
	}
	actual := c.instantiateMethodSignature(method.decl, class.decl, nil)
	expected := Signature{Parameters: []*Type{expectedSelf}, ReturnType: builtin("Bool")}
	c.compareSignatures(actual, expected, method.decl.Span, "equals")
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

func (c *Checker) checkBlock(block *parser.BlockStmt) *Type {
	c.pushScope()
	defer c.popScope()
	if block == nil || len(block.Statements) == 0 {
		return unknownType
	}
	for i := 0; i < len(block.Statements)-1; i++ {
		c.checkStmt(block.Statements[i])
	}
	last := block.Statements[len(block.Statements)-1]
	if exprStmt, ok := last.(*parser.ExprStmt); ok {
		return c.checkExpr(exprStmt.Expr)
	}
	c.checkStmt(last)
	return unknownType
}

func (c *Checker) checkStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		for i, bindingDecl := range s.Bindings {
			valueType := unknownType
			hasValue := i < len(s.Values) && s.Values[i] != nil
			if hasValue {
				expected := unknownType
				if bindingDecl.Type != nil {
					expected = c.resolveDeclaredType(bindingDecl.Type)
				}
				valueType = c.checkExprWithExpected(s.Values[i], expected)
			}
			declType := valueType
			if bindingDecl.Type != nil {
				declType = c.resolveDeclaredType(bindingDecl.Type)
				if hasValue {
					c.requireAssignable(valueType, declType, bindingDecl.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+declType.String())
				}
			} else if !hasValue {
				c.addDiagnostic("invalid_deferred", "deferred binding '"+bindingDecl.Name+"' requires an explicit type", bindingDecl.Span)
				declType = unknownType
			}
			c.define(bindingDecl.Name, declType, bindingDecl.Mutable)
		}
	case *parser.LocalFunctionStmt:
		sig := Signature{Parameters: make([]*Type, len(s.Function.Parameters)), ReturnType: fromTypeRef(s.Function.ReturnType, c), Variadic: len(s.Function.Parameters) > 0 && s.Function.Parameters[len(s.Function.Parameters)-1].Variadic}
		for i, param := range s.Function.Parameters {
			sig.Parameters[i] = fromTypeRef(param.Type, c)
		}
		c.define(s.Function.Name, functionType(s.Function.Name, sig), false)
		c.pushScope()
		defer c.popScope()
		expectedReturn := fromTypeRef(s.Function.ReturnType, c)
		c.returnTypes = append(c.returnTypes, expectedReturn)
		defer func() { c.returnTypes = c.returnTypes[:len(c.returnTypes)-1] }()
		for _, param := range s.Function.Parameters {
			paramType := fromTypeRef(param.Type, c)
			if param.Variadic {
				paramType = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{paramType}}
			}
			c.define(param.Name, paramType, false)
		}
		implicitReturn := c.checkBlock(s.Function.Body)
		if s.Function.ReturnType != nil && !isUnknown(implicitReturn) {
			c.requireAssignable(implicitReturn, expectedReturn, s.Function.Body.Span, "invalid_return_type", "cannot implicitly return "+implicitReturn.String()+" from function returning "+expectedReturn.String())
		}
	case *parser.AssignmentStmt:
		targetType, mutable := c.checkAssignmentTarget(s.Target, s.Span)
		valueType := c.checkExpr(s.Value)
		if !mutable {
			return
		}
		if s.Operator == "=" && !c.allowEqualsAssignment(s.Target) {
			c.addDiagnostic("invalid_assignment_operator", "use ':=' for mutable reassignment", s.Span)
			return
		}
		if s.Operator != "=" && s.Operator != ":=" {
			op := s.Operator[:len(s.Operator)-1]
			c.checkBinaryOperation(targetType, valueType, op, s.Span)
		}
		c.requireAssignable(valueType, targetType, s.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+targetType.String())
	case *parser.MultiAssignmentStmt:
		if len(s.Targets) != len(s.Values) {
			c.addDiagnostic("invalid_assignment_count", fmt.Sprintf("assignment expects %d values, got %d", len(s.Targets), len(s.Values)), s.Span)
		}
		count := len(s.Targets)
		if len(s.Values) < count {
			count = len(s.Values)
		}
		for i := 0; i < count; i++ {
			targetType, mutable := c.checkAssignmentTarget(s.Targets[i], s.Span)
			valueType := c.checkExpr(s.Values[i])
			if !mutable {
				continue
			}
			if s.Operator == "=" && !c.allowEqualsAssignment(s.Targets[i]) {
				c.addDiagnostic("invalid_assignment_operator", "use ':=' for mutable reassignment", s.Span)
				continue
			}
			if s.Operator != "=" && s.Operator != ":=" {
				c.addDiagnostic("invalid_assignment_operator", "multi-assignment supports only '=' and ':='", s.Span)
				continue
			}
			c.requireAssignable(valueType, targetType, s.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+targetType.String())
		}
		for i := count; i < len(s.Targets); i++ {
			c.checkAssignmentTarget(s.Targets[i], s.Span)
		}
		for i := count; i < len(s.Values); i++ {
			c.checkExpr(s.Values[i])
		}
	case *parser.IfStmt:
		condType := c.checkExpr(s.Condition)
		c.requireAssignable(condType, builtin("Bool"), exprSpan(s.Condition), "invalid_condition_type", "if condition must be Bool")
		_ = c.checkBlock(s.Then)
		if s.ElseIf != nil {
			c.checkStmt(s.ElseIf)
		}
		if s.Else != nil {
			_ = c.checkBlock(s.Else)
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
		expectedReturn := unknownType
		if len(c.returnTypes) > 0 {
			expectedReturn = c.returnTypes[len(c.returnTypes)-1]
		}
		valueType := c.checkExprWithExpected(s.Value, expectedReturn)
		if len(c.returnTypes) == 0 {
			c.addDiagnostic("invalid_return", "return used outside callable body", s.Span)
			return
		}
		expected := c.returnTypes[len(c.returnTypes)-1]
		if isUnknown(expected) {
			c.returnTypes[len(c.returnTypes)-1] = valueType
			return
		}
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

func (c *Checker) checkBlockResult(block *parser.BlockStmt, code, message string) *Type {
	if block == nil || len(block.Statements) == 0 {
		c.addDiagnostic(code, message, blockSpan(block))
		return unknownType
	}
	c.pushScope()
	defer c.popScope()
	for i := 0; i < len(block.Statements)-1; i++ {
		c.checkStmt(block.Statements[i])
	}
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*parser.ExprStmt)
	if !ok {
		c.checkStmt(last)
		c.addDiagnostic(code, message, stmtSpan(last))
		return unknownType
	}
	return c.checkExpr(exprStmt.Expr)
}

func (c *Checker) checkExpr(expr parser.Expr) *Type {
	return c.checkExprWithExpected(expr, nil)
}

func (c *Checker) checkExprWithExpected(expr parser.Expr, expected *Type) *Type {
	var result *Type
	switch e := expr.(type) {
	case *parser.Identifier:
		if binding, depth, ok := c.lookupWithDepth(e.Name); ok {
			if c.capturesMutableOuterBinding(binding, depth) {
				c.addDiagnostic("invalid_lambda_capture", "lambdas cannot capture mutable binding '"+e.Name+"'", e.Span)
			}
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
		if e.Name == "Term" {
			result = &Type{Kind: TypeInterface, Name: "Term"}
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
			result = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{unknownType}}
			break
		}
		elemType := c.checkExpr(e.Elements[0])
		for _, elem := range e.Elements[1:] {
			nextType := c.checkExpr(elem)
			if !sameType(elemType, nextType) {
				c.addDiagnostic("type_mismatch", "list literal elements must have the same type", exprSpan(elem))
			}
		}
		result = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{elemType}}
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
	case *parser.IndexExpr:
		result = c.checkIndexExpr(e)
	case *parser.IfExpr:
		condType := c.checkExpr(e.Condition)
		c.requireAssignable(condType, builtin("Bool"), exprSpan(e.Condition), "invalid_condition_type", "if condition must be Bool")
		thenType := c.checkBlockResult(e.Then, "invalid_if_expression", "if expression branches must end with an expression")
		elseType := c.checkBlockResult(e.Else, "invalid_if_expression", "if expression branches must end with an expression")
		if !sameType(thenType, elseType) {
			c.addDiagnostic("type_mismatch", "if expression branches must have the same type", e.Span)
			result = unknownType
			break
		}
		result = thenType
	case *parser.ForYieldExpr:
		c.pushScope()
		for _, binding := range e.Bindings {
			iterType := c.checkExpr(binding.Iterable)
			elemType := c.iterableElementType(iterType)
			c.define(binding.Name, elemType, false)
		}
		yieldType := c.checkBlockResult(e.YieldBody, "invalid_yield_expression", "yield body must end with an expression")
		c.popScope()
		result = &Type{Kind: TypeInterface, Name: "List", Args: []*Type{yieldType}}
	case *parser.LambdaExpr:
		result = c.checkLambdaExpr(e, expected)
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
		if isBuiltinValue(ident.Name) {
			return c.checkBuiltinConstructorCall(ident.Name, call)
		}
		if class, ok := c.classes[ident.Name]; ok {
			return c.checkConstructorCall(class, call)
		}
	}
	if member, ok := call.Callee.(*parser.MemberExpr); ok {
		return c.checkMethodCall(member, call.Args)
	}

	calleeType := c.checkExpr(call.Callee)
	if calleeType.Kind != TypeFunction || calleeType.Signature == nil {
		for _, arg := range call.Args {
			c.checkExpr(arg)
		}
		return unknownType
	}

	sig := *calleeType.Signature
	if !validArgCount(sig, len(call.Args)) {
		c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %s arguments, got %d", expectedArgCount(sig), len(call.Args)), call.Span)
	}
	for i, arg := range call.Args {
		var argType *Type
		if expected, ok := paramTypeForArg(sig, i); ok {
			argType = c.checkExprWithExpected(arg, expected)
			c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			continue
		}
		argType = c.checkExpr(arg)
	}
	return sig.ReturnType
}

func (c *Checker) checkBuiltinConstructorCall(name string, call *parser.CallExpr) *Type {
	switch name {
	case "List":
		if len(call.Args) == 0 {
			return &Type{Kind: TypeInterface, Name: "List", Args: []*Type{unknownType}}
		}
		elemType := c.checkExpr(call.Args[0])
		for _, arg := range call.Args[1:] {
			argType := c.checkExpr(arg)
			if !sameType(elemType, argType) {
				c.addDiagnostic("type_mismatch", "List constructor arguments must have the same type", exprSpan(arg))
			}
		}
		return &Type{Kind: TypeInterface, Name: "List", Args: []*Type{elemType}}
	case "Set":
		if len(call.Args) == 0 {
			return &Type{Kind: TypeInterface, Name: "Set", Args: []*Type{unknownType}}
		}
		elemType := c.checkExpr(call.Args[0])
		for _, arg := range call.Args[1:] {
			argType := c.checkExpr(arg)
			if !sameType(elemType, argType) {
				c.addDiagnostic("type_mismatch", "Set constructor arguments must have the same type", exprSpan(arg))
			}
		}
		return &Type{Kind: TypeInterface, Name: "Set", Args: []*Type{elemType}}
	case "Map":
		if len(call.Args) == 0 {
			return &Type{Kind: TypeInterface, Name: "Map", Args: []*Type{unknownType, unknownType}}
		}
		keyType := unknownType
		valType := unknownType
		for i, arg := range call.Args {
			pair, ok := arg.(*parser.BinaryExpr)
			if !ok || pair.Operator != ":" {
				c.addDiagnostic("invalid_argument_type", "Map constructor expects key : value pairs", exprSpan(arg))
				c.checkExpr(arg)
				continue
			}
			leftType := c.checkExpr(pair.Left)
			rightType := c.checkExpr(pair.Right)
			if i == 0 {
				keyType, valType = leftType, rightType
				continue
			}
			if !sameType(keyType, leftType) {
				c.addDiagnostic("type_mismatch", "Map constructor keys must have the same type", exprSpan(pair.Left))
			}
			if !sameType(valType, rightType) {
				c.addDiagnostic("type_mismatch", "Map constructor values must have the same type", exprSpan(pair.Right))
			}
		}
		return &Type{Kind: TypeInterface, Name: "Map", Args: []*Type{keyType, valType}}
	case "Array":
		if len(call.Args) != 1 {
			for _, arg := range call.Args {
				c.checkExpr(arg)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Array constructor expects 1 argument, got %d", len(call.Args)), call.Span)
			return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
		}
		lengthType := c.checkExpr(call.Args[0])
		c.requireAssignable(lengthType, builtin("Int"), exprSpan(call.Args[0]), "invalid_argument_type", "Array constructor length must be Int")
		return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
	case "Some":
		if len(call.Args) != 1 {
			for _, arg := range call.Args {
				c.checkExpr(arg)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Some constructor expects 1 argument, got %d", len(call.Args)), call.Span)
			return &Type{Kind: TypeInterface, Name: "Option", Args: []*Type{unknownType}}
		}
		valueType := c.checkExpr(call.Args[0])
		return &Type{Kind: TypeInterface, Name: "Option", Args: []*Type{valueType}}
	case "None":
		if len(call.Args) != 0 {
			for _, arg := range call.Args {
				c.checkExpr(arg)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("None constructor expects 0 arguments, got %d", len(call.Args)), call.Span)
		}
		return &Type{Kind: TypeInterface, Name: "Option", Args: []*Type{unknownType}}
	default:
		for _, arg := range call.Args {
			c.checkExpr(arg)
		}
		return unknownType
	}
}

func (c *Checker) checkIndexExpr(expr *parser.IndexExpr) *Type {
	receiverType := c.checkExpr(expr.Receiver)
	indexType := c.checkExpr(expr.Index)
	c.requireAssignable(indexType, builtin("Int"), exprSpan(expr.Index), "invalid_index_type", "index expression must be Int")
	if isUnknown(receiverType) {
		return unknownType
	}
	if receiverType.Kind == TypeBuiltin && receiverType.Name == "Array" && len(receiverType.Args) == 1 {
		return receiverType.Args[0]
	}
	c.addDiagnostic("invalid_index_target", "indexing requires Array[T]", expr.Span)
	return unknownType
}

func (c *Checker) checkConstructorCall(class classInfo, call *parser.CallExpr) *Type {
	classType := &Type{Kind: TypeClass, Name: class.decl.Name}
	if len(class.constructors) != 0 {
		argTypes := c.checkArgTypes(call.Args)
		if ctor, ok := c.resolveConstructorOverload(class, argTypes, call.Span); ok {
			sig := c.instantiateMethodSignature(ctor, class.decl, constructorTypeArgs(class.decl, call.Callee))
			for i := range argTypes {
				if expected, ok := paramTypeForArg(sig, i); ok {
					argType := c.checkExprWithExpected(call.Args[i], expected)
					c.requireAssignable(argType, expected, exprSpan(call.Args[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
				} else {
					c.checkExpr(call.Args[i])
				}
			}
		}
	} else if len(call.Args) != 0 {
		c.checkArgTypes(call.Args)
		c.addDiagnostic("invalid_argument_count", fmt.Sprintf("constructor '%s' expects 0 arguments, got %d", class.decl.Name, len(call.Args)), call.Span)
	}
	if ident, ok := call.Callee.(*parser.Identifier); ok {
		if refType, ok := c.lookupTypeInstance(ident.Name); ok {
			classType = refType
		}
	}
	return classType
}

func (c *Checker) checkMethodCall(member *parser.MemberExpr, args []parser.Expr) *Type {
	receiverType := c.checkExpr(member.Receiver)
	argTypes := c.checkArgTypes(args)
	if isUnknown(receiverType) {
		return unknownType
	}
	if receiverType.Kind == TypeBuiltin && receiverType.Name == "Array" {
		switch member.Name {
		case "size":
			if len(argTypes) != 0 {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %d arguments, got %d", member.Name, 0, len(argTypes)), member.Span)
			}
			return builtin("Int")
		default:
			c.addDiagnostic("unknown_member", "unknown member '"+member.Name+"'", member.Span)
			return unknownType
		}
	}
	switch receiverType.Kind {
	case TypeClass:
		info, ok := c.classes[receiverType.Name]
		if !ok {
			return unknownType
		}
		method, ok := c.resolveMethodOverload(info, receiverType, member.Name, argTypes, member.Span)
		if !ok {
			return unknownType
		}
		sig := c.instantiateMethodSignature(method.decl, info.decl, c.substForDecl(info.decl.TypeParameters, receiverType.Args))
		for i := range argTypes {
			if expected, ok := paramTypeForArg(sig, i); ok {
				argType := c.checkExprWithExpected(args[i], expected)
				c.requireAssignable(argType, expected, exprSpan(args[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			} else {
				c.checkExpr(args[i])
			}
		}
		return sig.ReturnType
	case TypeInterface:
		if receiverType.Name == "Term" && (member.Name == "println" || member.Name == "print") {
			for _, arg := range args {
				c.checkExpr(arg)
			}
			return receiverType
		}
		info, ok := c.interfaces[receiverType.Name]
		if !ok {
			return unknownType
		}
		method, ok := info.methods[member.Name]
		if !ok {
			c.addDiagnostic("unknown_member", "unknown member '"+member.Name+"'", member.Span)
			return unknownType
		}
		sig := c.instantiateInterfaceMethodSignature(method.decl, c.substForDecl(info.decl.TypeParameters, receiverType.Args))
		if !validArgCount(sig, len(argTypes)) {
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %s arguments, got %d", member.Name, expectedArgCount(sig), len(argTypes)), member.Span)
		}
		for i := range argTypes {
			if expected, ok := paramTypeForArg(sig, i); ok {
				argType := c.checkExprWithExpected(args[i], expected)
				c.requireAssignable(argType, expected, exprSpan(args[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			} else {
				c.checkExpr(args[i])
			}
		}
		return sig.ReturnType
	default:
		c.addDiagnostic("invalid_call_target", "member call requires class or interface receiver", member.Span)
		return unknownType
	}
}

func (c *Checker) checkLambdaExpr(expr *parser.LambdaExpr, expected *Type) *Type {
	c.pushScope()
	defer c.popScope()

	boundary := len(c.scopes) - 1
	c.lambdaScopes = append(c.lambdaScopes, boundary)
	defer func() { c.lambdaScopes = c.lambdaScopes[:len(c.lambdaScopes)-1] }()

	params := make([]*Type, len(expr.Parameters))
	expectedSig := (*Signature)(nil)
	if expected != nil && expected.Kind == TypeFunction && expected.Signature != nil {
		expectedSig = expected.Signature
		if len(expectedSig.Parameters) != len(expr.Parameters) {
			c.addDiagnostic("invalid_lambda_type", "lambda parameter count does not match expected function type", expr.Span)
			expectedSig = nil
		}
	}
	for i, param := range expr.Parameters {
		paramType := unknownType
		if param.Type != nil {
			paramType = c.resolveDeclaredType(param.Type)
		} else if expectedSig != nil {
			paramType = expectedSig.Parameters[i]
		} else {
			c.addDiagnostic("invalid_lambda_type", "untyped lambda parameters require a contextual function type", param.Span)
		}
		params[i] = paramType
		c.define(param.Name, paramType, false)
	}

	returnType := unknownType
	if expr.Body != nil {
		returnType = c.checkExpr(expr.Body)
		if expectedSig != nil && !isUnknown(expectedSig.ReturnType) {
			c.requireAssignable(returnType, expectedSig.ReturnType, exprSpan(expr.Body), "invalid_lambda_type", "lambda body does not match expected return type")
			returnType = expectedSig.ReturnType
		}
	}
	if expr.BlockBody != nil {
		expectedReturn := unknownType
		if expectedSig != nil {
			expectedReturn = expectedSig.ReturnType
		}
		c.returnTypes = append(c.returnTypes, expectedReturn)
		implicitReturn := c.checkBlock(expr.BlockBody)
		returnType = c.returnTypes[len(c.returnTypes)-1]
		c.returnTypes = c.returnTypes[:len(c.returnTypes)-1]
		if !isUnknown(implicitReturn) {
			if isUnknown(returnType) {
				returnType = implicitReturn
			} else {
				c.requireAssignable(implicitReturn, returnType, expr.BlockBody.Span, "invalid_lambda_type", "lambda body does not match expected return type")
			}
		}
	}
	return functionType("<lambda>", Signature{Parameters: params, ReturnType: returnType})
}

func (c *Checker) checkMemberExpr(expr *parser.MemberExpr) *Type {
	receiverType := c.checkExpr(expr.Receiver)
	if memberType, ok := c.lookupMember(receiverType, expr.Name, expr.Span); ok {
		return memberType
	}
	return unknownType
}

func (c *Checker) lookupMember(receiver *Type, name string, span parser.Span) (*Type, bool) {
	if isUnknown(receiver) {
		return unknownType, true
	}
	if receiver.Kind == TypeBuiltin && receiver.Name == "Array" {
		if name == "size" {
			c.addDiagnostic("invalid_member_access", "method '"+name+"' must be called with ()", span)
			return unknownType, true
		}
		return unknownType, false
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
		if methods, ok := info.methods[name]; ok && len(methods) > 0 {
			if hasPrivateOnlyMatch(methods, info.decl, c) {
				c.addDiagnostic("private_access", "cannot access private method '"+name+"' outside class '"+info.decl.Name+"'", span)
				return unknownType, true
			}
			c.addDiagnostic("invalid_member_access", "method '"+name+"' must be called with ()", span)
			return unknownType, true
		}
	case TypeInterface:
		info, ok := c.interfaces[receiver.Name]
		if !ok {
			return unknownType, false
		}
		subst := c.substForDecl(info.decl.TypeParameters, receiver.Args)
		if method, ok := info.methods[name]; ok {
			_ = c.instantiateInterfaceMethodSignature(method.decl, subst)
			c.addDiagnostic("invalid_member_access", "method '"+name+"' must be called with ()", span)
			return unknownType, true
		}
	}
	return unknownType, false
}

func (c *Checker) checkBinaryOperation(left, right *Type, op string, span parser.Span) *Type {
	switch op {
	case "+":
		if sameType(left, builtin("String")) || sameType(right, builtin("String")) {
			return builtin("String")
		}
		if !isNumeric(left) || !isNumeric(right) {
			c.addDiagnostic("invalid_binary_operand", "operator + requires numeric operands unless one side is String", span)
			return unknownType
		}
		if !sameType(left, right) {
			c.addDiagnostic("type_mismatch", "arithmetic operands must have the same type", span)
		}
		return left
	case "-", "*", "/", "%":
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
		if !c.supportsEquality(left) || !c.supportsEquality(right) {
			c.addDiagnostic("invalid_binary_operand", "equality requires Eq support for this type", span)
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
	case *parser.IndexExpr:
		elemType := c.checkIndexExpr(t)
		if isUnknown(elemType) {
			return unknownType, false
		}
		return elemType, true
	default:
		c.addDiagnostic("invalid_assignment_target", "invalid assignment target", span)
		return unknownType, false
	}
}

func (c *Checker) allowEqualsAssignment(target parser.Expr) bool {
	member, ok := target.(*parser.MemberExpr)
	if !ok {
		return false
	}
	if c.currentMethod == nil || !c.currentMethod.Constructor {
		return false
	}
	ident, ok := member.Receiver.(*parser.Identifier)
	return ok && ident.Name == "this"
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
		if methods, hasMethod := info.methods[expr.Name]; hasMethod && len(methods) > 0 {
			if len(methods) == 1 && methods[0].decl.Private && !c.canAccessPrivate(info.decl) {
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
	if ref.ReturnType != nil {
		params := make([]*Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = c.instantiateTypeRef(param, subst)
		}
		return &Type{
			Kind: TypeFunction,
			Name: "func",
			Signature: &Signature{
				Parameters: params,
				ReturnType: c.instantiateTypeRef(ref.ReturnType, subst),
			},
		}
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
	return Signature{Parameters: params, ReturnType: result, Variadic: len(method.Parameters) > 0 && method.Parameters[len(method.Parameters)-1].Variadic}
}

func (c *Checker) checkConstructorRules(class classInfo) {
	if len(class.constructors) == 0 {
		if missing := c.uninitializedLetFields(class.decl, nil); len(missing) > 0 {
			c.addDiagnostic("constructor_required", "class '"+class.decl.Name+"' requires init to initialize immutable fields: "+joinNames(missing), class.decl.Span)
		}
		return
	}
	seen := map[string]*parser.MethodDecl{}
	for _, ctor := range class.constructors {
		key := methodSignatureKey(ctor)
		if prev, ok := seen[key]; ok {
			c.addDiagnostic("duplicate_constructor", "duplicate constructor overload for class '"+class.decl.Name+"'", ctor.Span)
			c.addDiagnostic("duplicate_constructor", "duplicate constructor overload for class '"+class.decl.Name+"'", prev.Span)
			continue
		}
		seen[key] = ctor
		if missing := c.uninitializedLetFields(class.decl, ctor); len(missing) > 0 {
			c.addDiagnostic("uninitialized_field", "constructor 'init' must initialize immutable fields: "+joinNames(missing), ctor.Span)
		}
	}
}

func (c *Checker) uninitializedLetFields(owner *parser.ClassDecl, ctor *parser.MethodDecl) []string {
	initialized := map[string]bool{}
	if ctor != nil {
		c.collectInitializedFields(ctor.Body, initialized)
	}
	var missing []string
	for _, field := range owner.Fields {
		if field.Mutable {
			continue
		}
		if field.Initializer != nil {
			continue
		}
		if !initialized[field.Name] {
			missing = append(missing, field.Name)
		}
	}
	return missing
}

func (c *Checker) collectInitializedFields(block *parser.BlockStmt, initialized map[string]bool) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *parser.AssignmentStmt:
			if s.Operator != "=" {
				continue
			}
			member, ok := s.Target.(*parser.MemberExpr)
			if !ok {
				continue
			}
			ident, ok := member.Receiver.(*parser.Identifier)
			if ok && ident.Name == "this" {
				initialized[member.Name] = true
			}
		case *parser.IfStmt:
			// Keep constructor rules simple: writes must happen unconditionally in the constructor body.
		}
	}
}

func (c *Checker) checkArgTypes(args []parser.Expr) []*Type {
	result := make([]*Type, len(args))
	for i, arg := range args {
		result[i] = c.checkExpr(arg)
	}
	return result
}

func (c *Checker) resolveConstructorOverload(class classInfo, argTypes []*Type, span parser.Span) (*parser.MethodDecl, bool) {
	candidates := make([]*parser.MethodDecl, 0, len(class.constructors))
	for _, ctor := range class.constructors {
		sig := c.instantiateMethodSignature(ctor, class.decl, nil)
		if signatureMatches(sig, argTypes) {
			candidates = append(candidates, ctor)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	if len(candidates) > 1 {
		c.addDiagnostic("ambiguous_overload", "constructor call for class '"+class.decl.Name+"' is ambiguous", span)
		return nil, false
	}
	c.addDiagnostic("no_matching_overload", fmt.Sprintf("no constructor overload for class '%s' matches %d arguments", class.decl.Name, len(argTypes)), span)
	return nil, false
}

func (c *Checker) resolveMethodOverload(class classInfo, receiver *Type, name string, argTypes []*Type, span parser.Span) (methodInfo, bool) {
	methods, ok := class.methods[name]
	if !ok || len(methods) == 0 {
		c.addDiagnostic("unknown_member", "unknown member '"+name+"'", span)
		return methodInfo{}, false
	}
	subst := c.substForDecl(class.decl.TypeParameters, receiver.Args)
	var matches []methodInfo
	for _, method := range methods {
		if method.decl.Private && !c.canAccessPrivate(class.decl) {
			continue
		}
		sig := c.instantiateMethodSignature(method.decl, class.decl, subst)
		if signatureMatches(sig, argTypes) {
			matches = append(matches, method)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) > 1 {
		c.addDiagnostic("ambiguous_overload", "method call '"+name+"' is ambiguous", span)
		return methodInfo{}, false
	}
	if hasPrivateOnlyMatch(methods, class.decl, c) {
		c.addDiagnostic("private_access", "cannot access private method '"+name+"' outside class '"+class.decl.Name+"'", span)
		return methodInfo{}, false
	}
	c.addDiagnostic("no_matching_overload", fmt.Sprintf("no overload of method '%s' matches %d arguments", name, len(argTypes)), span)
	return methodInfo{}, false
}

func (c *Checker) findMatchingMethodOverload(class classInfo, name string, paramTypes []*Type) (methodInfo, bool) {
	methods, ok := class.methods[name]
	if !ok {
		return methodInfo{}, false
	}
	for _, method := range methods {
		sig := c.instantiateMethodSignature(method.decl, class.decl, nil)
		if signatureMatches(sig, paramTypes) {
			return method, true
		}
	}
	return methodInfo{}, false
}

func signatureMatches(sig Signature, argTypes []*Type) bool {
	if !validArgCount(sig, len(argTypes)) {
		return false
	}
	for i := range argTypes {
		expected, ok := paramTypeForArg(sig, i)
		if !ok || !sameType(expected, argTypes[i]) {
			return false
		}
	}
	return true
}

func methodSignatureKey(method *parser.MethodDecl) string {
	sig := method.Name + "("
	for i, param := range method.Parameters {
		if i > 0 {
			sig += ","
		}
		sig += param.Type.Name
		if param.Variadic {
			sig += "..."
		}
		for _, arg := range param.Type.Arguments {
			sig += "[" + arg.Name + "]"
		}
	}
	return sig + ")"
}

func validArgCount(sig Signature, count int) bool {
	if sig.Variadic {
		return count >= len(sig.Parameters)-1
	}
	return count == len(sig.Parameters)
}

func expectedArgCount(sig Signature) string {
	if sig.Variadic {
		return fmt.Sprintf("at least %d", len(sig.Parameters)-1)
	}
	return fmt.Sprintf("%d", len(sig.Parameters))
}

func paramTypeForArg(sig Signature, index int) (*Type, bool) {
	if !sig.Variadic {
		if index < len(sig.Parameters) {
			return sig.Parameters[index], true
		}
		return nil, false
	}
	if len(sig.Parameters) == 0 {
		return nil, false
	}
	last := len(sig.Parameters) - 1
	if index < last {
		return sig.Parameters[index], true
	}
	return sig.Parameters[last], true
}

func joinNames(names []string) string {
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}

func hasPrivateOnlyMatch(methods []methodInfo, owner *parser.ClassDecl, c *Checker) bool {
	if c.canAccessPrivate(owner) {
		return false
	}
	for _, method := range methods {
		if method.decl.Private {
			return true
		}
	}
	return false
}

func (c *Checker) instantiateInterfaceMethodSignature(method parser.InterfaceMethod, subst map[string]*Type) Signature {
	params := make([]*Type, len(method.Parameters))
	for i, param := range method.Parameters {
		params[i] = c.instantiateTypeRef(param.Type, subst)
	}
	return Signature{
		Parameters: params,
		ReturnType: c.instantiateTypeRef(method.ReturnType, subst),
		Variadic:   len(method.Parameters) > 0 && method.Parameters[len(method.Parameters)-1].Variadic,
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

func (c *Checker) supportsEquality(t *Type) bool {
	if isUnknown(t) {
		return true
	}
	switch t.Kind {
	case TypeBuiltin:
		switch t.Name {
		case "Int", "Int64", "Bool", "String", "Rune", "Float", "Float64":
			return true
		default:
			return false
		}
	case TypeClass:
		return c.classImplementsEq(t)
	default:
		return false
	}
}

func (c *Checker) classImplementsEq(t *Type) bool {
	info, ok := c.classes[t.Name]
	if !ok {
		return false
	}
	for _, impl := range info.decl.Implements {
		if impl.Name != "Eq" || len(impl.Arguments) != 1 {
			continue
		}
		expected := c.instantiateTypeRef(impl.Arguments[0], c.substForDecl(info.decl.TypeParameters, t.Args))
		if sameType(expected, t) {
			return true
		}
	}
	return false
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
	b, _, ok := c.lookupWithDepth(name)
	return b, ok
}

func (c *Checker) lookupWithDepth(name string) (binding, int, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if value, ok := c.scopes[i][name]; ok {
			return value, i, true
		}
	}
	if value, ok := c.globals[name]; ok {
		return value, -1, true
	}
	return binding{}, -1, false
}

func (c *Checker) capturesMutableOuterBinding(b binding, depth int) bool {
	if len(c.lambdaScopes) == 0 || !b.mutable {
		return false
	}
	boundary := c.lambdaScopes[len(c.lambdaScopes)-1]
	if depth == -1 {
		return true
	}
	return depth < boundary
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
	if isBuiltinInterfaceType(name) {
		return TypeInterface
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
	case "Int", "Int64", "Bool", "String", "Rune", "Float", "Float64", "Array":
		return true
	default:
		return false
	}
}

func isBuiltinInterfaceType(name string) bool {
	switch name {
	case "Eq", "List", "Set", "Map", "Term", "Option":
		return true
	default:
		return false
	}
}

func isBuiltinValue(name string) bool {
	switch name {
	case "List", "Map", "Set", "Array", "Some", "None":
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
	case *parser.CallExpr:
		return e.Span
	case *parser.MemberExpr:
		return e.Span
	case *parser.IndexExpr:
		return e.Span
	case *parser.IfExpr:
		return e.Span
	case *parser.ForYieldExpr:
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

func blockSpan(block *parser.BlockStmt) parser.Span {
	if block == nil {
		return parser.Span{}
	}
	return block.Span
}

func stmtSpan(stmt parser.Statement) parser.Span {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		return s.Span
	case *parser.LocalFunctionStmt:
		return s.Span
	case *parser.AssignmentStmt:
		return s.Span
	case *parser.MultiAssignmentStmt:
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
