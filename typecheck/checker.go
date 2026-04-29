package typecheck

import (
	"fmt"

	"a-lang/module"
	"a-lang/parser"
	"a-lang/predef"
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
	name         string
	decl         *parser.ClassDecl
	fields       map[string]fieldInfo
	methods      map[string][]methodInfo
	constructors []*parser.MethodDecl
	enumCases    map[string]parser.EnumCaseDecl
}

type interfaceInfo struct {
	decl    *parser.InterfaceDecl
	methods map[string]interfaceMethodInfo
}

type moduleInfo struct {
	functions     map[string]Signature
	functionDecls map[string]*parser.FunctionDecl
	classes       map[string]classInfo
	interfaces    map[string]interfaceInfo
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
	functionDecls map[string]*parser.FunctionDecl
	classes       map[string]classInfo
	interfaces    map[string]interfaceInfo
	imports       map[string]moduleInfo
	importedClasses    map[string]classInfo
	importedInterfaces map[string]interfaceInfo
	importedInterfaceNames map[string]string
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
		globals:       map[string]binding{},
		functions:     map[string]Signature{},
		functionDecls: map[string]*parser.FunctionDecl{},
		classes:       map[string]classInfo{},
		interfaces:    map[string]interfaceInfo{},
		imports:       map[string]moduleInfo{},
		importedClasses:    map[string]classInfo{},
		importedInterfaces: map[string]interfaceInfo{},
		importedInterfaceNames: map[string]string{},
		exprTypes:     map[parser.Expr]*Type{},
	}
	c.installBuiltinInterfaces()
	c.collectDecls(program)
	c.checkProgram(program)
	return Result{Diagnostics: c.diagnostics, ExprTypes: c.exprTypes}
}

func AnalyzeModule(mod *module.LoadedModule) Result {
	seen := map[string]Result{}
	var analyzeOne func(*module.LoadedModule) Result
	analyzeOne = func(current *module.LoadedModule) Result {
		if result, ok := seen[current.Path]; ok {
			return result
		}
		c := &Checker{
			globals:       map[string]binding{},
			functions:     map[string]Signature{},
			functionDecls: map[string]*parser.FunctionDecl{},
			classes:       map[string]classInfo{},
			interfaces:    map[string]interfaceInfo{},
				imports:       map[string]moduleInfo{},
				importedClasses:    map[string]classInfo{},
				importedInterfaces: map[string]interfaceInfo{},
				importedInterfaceNames: map[string]string{},
				exprTypes:     map[parser.Expr]*Type{},
			}
		c.installBuiltinInterfaces()
		c.installModuleImports(current)
		c.collectDecls(current.Program)
		c.checkProgram(current.Program)
		result := Result{Diagnostics: append([]semantic.Diagnostic(nil), c.diagnostics...), ExprTypes: c.exprTypes}
		seen[current.Path] = result
			for _, imported := range current.Dependencies {
				child := analyzeOne(imported)
				result.Diagnostics = append(result.Diagnostics, child.Diagnostics...)
			}
		seen[current.Path] = result
		return result
	}
	return analyzeOne(mod)
}

func (c *Checker) installBuiltinInterfaces() {
	registry, err := predef.Load()
	if err != nil {
		panic(err)
	}
	for _, decl := range registry.Program.Interfaces {
		if isBuiltinType(decl.Name) {
			continue
		}
		info := interfaceInfo{
			decl:    decl,
			methods: map[string]interfaceMethodInfo{},
		}
		for _, method := range decl.Methods {
			info.methods[method.Name] = interfaceMethodInfo{decl: method}
		}
		c.interfaces[decl.Name] = info
	}
	for _, decl := range registry.Program.Classes {
		if isBuiltinType(decl.Name) {
			continue
		}
		info := classInfo{
			name:      decl.Name,
			decl:      decl,
			fields:    map[string]fieldInfo{},
			methods:   map[string][]methodInfo{},
			enumCases: map[string]parser.EnumCaseDecl{},
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
		for _, enumCase := range decl.Cases {
			info.enumCases[enumCase.Name] = enumCase
		}
		c.classes[decl.Name] = info
	}
}

func (c *Checker) installModuleImports(current *module.LoadedModule) {
	currentPackage := current.Program.PackageName
	for alias, imported := range current.Imports {
		samePackage := currentPackage != "" && imported.Program.PackageName == currentPackage
		info := moduleInfo{
			functions:     map[string]Signature{},
			functionDecls: map[string]*parser.FunctionDecl{},
			classes:       map[string]classInfo{},
			interfaces:    map[string]interfaceInfo{},
		}
		for _, fn := range imported.Program.Functions {
			if fn.Private && !samePackage {
				continue
			}
			params := make([]*Type, len(fn.Parameters))
			for i, param := range fn.Parameters {
				params[i] = fromTypeRef(param.Type, c)
			}
			info.functions[fn.Name] = Signature{
				Parameters: params,
				ReturnType: fromTypeRef(fn.ReturnType, c),
				Variadic:   len(fn.Parameters) > 0 && fn.Parameters[len(fn.Parameters)-1].Variadic,
			}
			info.functionDecls[fn.Name] = fn
		}
		for _, decl := range imported.Program.Interfaces {
			if decl.Private && !samePackage {
				continue
			}
			qualified := imported.Path + "::" + decl.Name
			iface := interfaceInfo{
				decl:    decl,
				methods: map[string]interfaceMethodInfo{},
			}
			for _, method := range decl.Methods {
				iface.methods[method.Name] = interfaceMethodInfo{decl: method}
			}
			info.interfaces[decl.Name] = iface
			c.interfaces[qualified] = iface
		}
		for _, decl := range imported.Program.Classes {
			if decl.Private && !samePackage {
				continue
			}
			qualified := imported.Path + "::" + decl.Name
			class := classInfo{
				name:      qualified,
				decl:      decl,
				fields:    map[string]fieldInfo{},
				methods:   map[string][]methodInfo{},
				enumCases: map[string]parser.EnumCaseDecl{},
			}
			for _, field := range decl.Fields {
				class.fields[field.Name] = fieldInfo{decl: field}
			}
			for _, method := range decl.Methods {
				class.methods[method.Name] = append(class.methods[method.Name], methodInfo{decl: method})
				if method.Constructor {
					class.constructors = append(class.constructors, method)
				}
			}
			for _, enumCase := range decl.Cases {
				class.enumCases[enumCase.Name] = enumCase
			}
			info.classes[decl.Name] = class
			c.classes[qualified] = class
		}
			c.imports[alias] = info
		}
	for localName, symbol := range current.SymbolImports {
		samePackage := currentPackage != "" && symbol.Module.SourceProgram.PackageName == currentPackage
		if symbol.IsInterface {
			for _, decl := range symbol.Module.SourceProgram.Interfaces {
				if decl.Name != symbol.OriginalName {
					continue
				}
				if decl.Private && !samePackage {
					break
				}
				info := interfaceInfo{
					decl:    decl,
					methods: map[string]interfaceMethodInfo{},
				}
				for _, method := range decl.Methods {
					info.methods[method.Name] = interfaceMethodInfo{decl: method}
				}
				qualified := symbol.Module.Path + "::" + decl.Name
				c.importedInterfaces[localName] = info
				c.importedInterfaceNames[localName] = qualified
				c.interfaces[qualified] = info
				break
			}
			continue
		}
		for _, decl := range symbol.Module.SourceProgram.Classes {
			if decl.Name != symbol.OriginalName {
				continue
			}
			if decl.Private && !samePackage {
				break
			}
			class := classInfo{
				name:      symbol.Module.Path + "::" + decl.Name,
				decl:      decl,
				fields:    map[string]fieldInfo{},
				methods:   map[string][]methodInfo{},
				enumCases: map[string]parser.EnumCaseDecl{},
			}
			for _, field := range decl.Fields {
				class.fields[field.Name] = fieldInfo{decl: field}
			}
			for _, method := range decl.Methods {
				class.methods[method.Name] = append(class.methods[method.Name], methodInfo{decl: method})
				if method.Constructor {
					class.constructors = append(class.constructors, method)
				}
			}
			for _, enumCase := range decl.Cases {
				class.enumCases[enumCase.Name] = enumCase
			}
			c.importedClasses[localName] = class
			c.classes[class.name] = class
			break
		}
	}
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
			name:      decl.Name,
			decl:      decl,
			fields:    map[string]fieldInfo{},
			methods:   map[string][]methodInfo{},
			enumCases: map[string]parser.EnumCaseDecl{},
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
		for _, enumCase := range decl.Cases {
			info.enumCases[enumCase.Name] = enumCase
		}
		c.classes[decl.Name] = info
		if decl.Object {
			c.globals[decl.Name] = binding{typ: &Type{Kind: TypeClass, Name: decl.Name}, mutable: false}
		}
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
		c.functionDecls[fn.Name] = fn
	}
}

func (c *Checker) checkProgram(program *parser.Program) {
	c.checkGlobals(program.Statements)
	for _, fn := range program.Functions {
		c.checkFunction(fn)
	}
	for _, decl := range program.Interfaces {
		c.checkInterface(decl)
	}
	for _, decl := range program.Classes {
		c.checkClass(decl)
	}
}

func (c *Checker) checkInterface(decl *parser.InterfaceDecl) {
	c.pushTypeScope()
	defer c.popTypeScope()
	for _, param := range decl.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}
	c.validateTypeParameterBounds(decl.TypeParameters)
	for _, method := range decl.Methods {
		if method.Name == "this" {
			c.addDiagnostic("invalid_interface_method", "interface '"+decl.Name+"': interfaces cannot declare constructors", method.Span)
		}
	}
	for _, parent := range decl.Extends {
		parentType := c.resolveDeclaredType(parent)
		if parentType.Kind != TypeInterface {
			c.addDiagnostic("invalid_interface_inheritance", "interface '"+decl.Name+"' can only inherit from interfaces", parent.Span)
		}
	}
}

func (c *Checker) checkGlobals(statements []parser.Statement) {
	c.pushScope()
	defer c.popScope()
	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *parser.ValStmt:
			for i, bindingDecl := range s.Bindings {
				if bindingDecl.Deferred {
					c.addDiagnostic("invalid_deferred", "binding '"+bindingDecl.Name+"' cannot be initialized with '?' outside class fields", bindingDecl.Span)
				}
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
					c.addDiagnostic("invalid_deferred", "binding '"+bindingDecl.Name+"' cannot be initialized with '?' outside class fields", bindingDecl.Span)
					declType = unknownType
				}
				if bindingDecl.Name != "_" {
					c.globals[bindingDecl.Name] = binding{typ: declType, mutable: bindingDecl.Mutable}
					c.define(bindingDecl.Name, declType, bindingDecl.Mutable)
				}
			}
		case *parser.ExprStmt, *parser.AssignmentStmt, *parser.MultiAssignmentStmt, *parser.IfStmt, *parser.LoopStmt, *parser.ForStmt:
			c.checkStmt(stmt)
		default:
			c.addDiagnostic("unsupported_top_level", "unsupported top-level statement for type checking", stmtSpan(stmt))
		}
	}
}

func (c *Checker) checkFunction(fn *parser.FunctionDecl) {
	c.pushTypeScope()
	defer c.popTypeScope()
	for _, param := range fn.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}
	c.validateTypeParameterBounds(fn.TypeParameters)

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
	if fn.ReturnType != nil && !isUnknown(implicitReturn) && !isUnitType(expectedReturn) {
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
	c.validateTypeParameterBounds(decl.TypeParameters)

	for _, field := range decl.Fields {
		if decl.Object && !field.Mutable && field.Initializer == nil {
			c.addDiagnostic("invalid_object_field", "object '"+decl.Name+"' must initialize immutable field '"+field.Name+"'", field.Span)
		}
		if decl.Record && field.Mutable {
			c.addDiagnostic("invalid_record_field", "record '"+decl.Name+"' cannot declare mutable field '"+field.Name+"'", field.Span)
		}
		if decl.Enum && field.Mutable {
			c.addDiagnostic("invalid_enum_field", "enum '"+decl.Name+"' cannot declare mutable field '"+field.Name+"'", field.Span)
		}
		fieldType := c.resolveDeclaredType(field.Type)
		if field.Initializer != nil {
			valueType := c.checkExprWithExpected(field.Initializer, fieldType)
			c.requireAssignable(valueType, fieldType, exprSpan(field.Initializer), "type_mismatch", "cannot assign "+valueType.String()+" to "+fieldType.String())
		}
	}
	if !decl.Enum && !decl.Object {
		c.checkConstructorRules(info)
	}
	for _, method := range decl.Methods {
		c.checkMethod(method, decl)
	}
	c.checkOperatorMethods(info)
	c.checkImplMethodMarkers(info)
	if decl.Enum {
		c.checkEnumCases(info)
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
	c.pushTypeScope()
	defer c.popTypeScope()
	for _, param := range owner.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}
	for _, param := range method.TypeParameters {
		c.currentTypeScope()[param.Name] = TypeParam
	}
	c.validateTypeParameterBounds(method.TypeParameters)

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
	if owner.Enum {
		classArgs := make([]*Type, len(owner.TypeParameters))
		for i, param := range owner.TypeParameters {
			classArgs[i] = &Type{Kind: TypeParam, Name: param.Name}
		}
		for _, enumCase := range owner.Cases {
			if len(enumCase.Fields) == 0 {
				c.define(enumCase.Name, &Type{Kind: TypeClass, Name: owner.Name, Args: classArgs}, false)
			}
		}
		if method.Constructor {
			c.addDiagnostic("invalid_enum_method", "enum '"+owner.Name+"' cannot declare constructors", method.Span)
		}
	}
	if owner.Object && method.Constructor {
		c.addDiagnostic("invalid_object_method", "object '"+owner.Name+"' cannot declare constructors", method.Span)
	}
	if owner.Record && method.Constructor {
		c.addDiagnostic("invalid_record_method", "record '"+owner.Name+"': records cannot declare constructors", method.Span)
	}
	implicitReturn := c.checkBlock(method.Body)
	if !method.Constructor && method.ReturnType != nil && !isUnknown(implicitReturn) && !isUnitType(expectedReturn) {
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

	for _, method := range c.interfaceMethods(iface.decl, map[string]bool{}) {
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
		if !classMethod.decl.Impl {
			c.addDiagnostic("interface_method_requires_impl", "method '"+class.decl.Name+"."+classMethod.decl.Name+"' must use 'impl def' because it implements interface '"+iface.decl.Name+"'", classMethod.decl.Span)
		}
		actual := c.instantiateMethodSignature(classMethod.decl, class.decl, nil)
		c.compareSignatures(actual, expected, classMethod.decl.Span, method.Name)
	}
}

func (c *Checker) checkImplMethodMarkers(class classInfo) {
	if len(class.decl.Implements) == 0 {
		for _, methods := range class.methods {
			for _, method := range methods {
				if method.decl.Impl {
					c.addDiagnostic("invalid_impl_method", "method '"+class.decl.Name+"."+method.decl.Name+"' uses 'impl def' but class '"+class.decl.Name+"' does not implement any interfaces", method.decl.Span)
				}
			}
		}
		return
	}
	for _, methods := range class.methods {
		for _, method := range methods {
			if !method.decl.Impl {
				continue
			}
			if !c.methodMatchesDeclaredInterface(class, method.decl) && !c.methodMatchesEqInterface(class, method.decl) {
				c.addDiagnostic("invalid_impl_method", "method '"+class.decl.Name+"."+method.decl.Name+"' uses 'impl def' but does not match any declared interface method", method.decl.Span)
			}
		}
	}
}

func (c *Checker) checkOperatorMethods(class classInfo) {
	for _, methods := range class.methods {
		for _, method := range methods {
			if !method.decl.Operator {
				continue
			}
			if class.decl.Object {
				c.addDiagnostic("invalid_operator_method", "objects cannot declare operators", method.decl.Span)
				continue
			}
			if !isAllowedOperatorName(method.decl.Name) {
				c.addDiagnostic("invalid_operator_method", "operator '"+method.decl.Name+"' cannot be overloaded", method.decl.Span)
				continue
			}
			switch method.decl.Name {
			case "[]", ":+", ":-", "++", "--", "+", "*", "/", "%", "|", "&", ">>", "<<", "::":
				if len(method.decl.Parameters) != 1 {
					c.addDiagnostic("invalid_operator_method", "operator '"+method.decl.Name+"' must declare exactly 1 parameter", method.decl.Span)
				}
			case "~":
				if len(method.decl.Parameters) != 0 {
					c.addDiagnostic("invalid_operator_method", "operator '~' must declare 0 parameters", method.decl.Span)
				}
			case "-":
				if len(method.decl.Parameters) != 0 && len(method.decl.Parameters) != 1 {
					c.addDiagnostic("invalid_operator_method", "operator '-' must declare 0 or 1 parameters", method.decl.Span)
				}
			}
		}
	}
}

func isAllowedOperatorName(name string) bool {
	switch name {
	case "+", "-", "*", "/", "%", "[]", ":+", ":-", "++", "--", "|", "&", ">>", "<<", "~", "::":
		return true
	default:
		return false
	}
}

func (c *Checker) methodMatchesDeclaredInterface(class classInfo, method *parser.MethodDecl) bool {
	for _, impl := range class.decl.Implements {
		if impl == nil {
			continue
		}
		iface, ok := c.interfaces[impl.Name]
		if !ok {
			continue
		}
		subst := map[string]*Type{}
		for i, param := range iface.decl.TypeParameters {
			if i < len(impl.Arguments) {
				subst[param.Name] = c.instantiateTypeRef(impl.Arguments[i], nil)
			}
		}
		for _, ifaceMethod := range c.interfaceMethods(iface.decl, map[string]bool{}) {
			if ifaceMethod.Name != method.Name {
				continue
			}
			expected := c.instantiateInterfaceMethodSignature(ifaceMethod, subst)
			actual := c.instantiateMethodSignature(method, class.decl, nil)
			if signaturesCompatible(actual, expected) {
				return true
			}
		}
	}
	return false
}

func (c *Checker) methodMatchesEqInterface(class classInfo, method *parser.MethodDecl) bool {
	if method.Name != "equals" {
		return false
	}
	for _, impl := range class.decl.Implements {
		if impl == nil || impl.Name != "Eq" || len(impl.Arguments) != 1 {
			continue
		}
		expectedSelf := c.instantiateTypeRef(impl.Arguments[0], c.substForDecl(class.decl.TypeParameters, nil))
		expected := Signature{Parameters: []*Type{expectedSelf}, ReturnType: builtin("Bool")}
		actual := c.instantiateMethodSignature(method, class.decl, nil)
		if signaturesCompatible(actual, expected) {
			return true
		}
	}
	return false
}

func signaturesCompatible(actual Signature, expected Signature) bool {
	if actual.Variadic != expected.Variadic {
		return false
	}
	if len(actual.Parameters) != len(expected.Parameters) {
		return false
	}
	for i := range actual.Parameters {
		if !sameType(actual.Parameters[i], expected.Parameters[i]) {
			return false
		}
	}
	return sameType(actual.ReturnType, expected.ReturnType)
}

func (c *Checker) checkEnumCases(info classInfo) {
	sharedFields := map[string]parser.FieldDecl{}
	for _, field := range info.decl.Fields {
		sharedFields[field.Name] = field
	}
	seenCases := map[string]parser.Span{}
	for _, enumCase := range info.decl.Cases {
		if prev, ok := seenCases[enumCase.Name]; ok {
			c.addDiagnostic("duplicate_enum_case", "duplicate enum case '"+enumCase.Name+"'", enumCase.Span)
			c.addDiagnostic("duplicate_enum_case", "previous declaration of enum case '"+enumCase.Name+"'", prev)
			continue
		}
		seenCases[enumCase.Name] = enumCase.Span
		assigned := map[string]bool{}
		for _, field := range enumCase.Fields {
			if _, ok := sharedFields[field.Name]; ok {
				c.addDiagnostic("invalid_enum_case_field", "enum case '"+enumCase.Name+"' must assign shared field '"+field.Name+"' instead of redeclaring it", field.Span)
			}
			if field.Mutable {
				c.addDiagnostic("invalid_enum_case_field", "enum case '"+enumCase.Name+"' cannot declare mutable field '"+field.Name+"'", field.Span)
			}
			fieldType := c.resolveDeclaredType(field.Type)
			if field.Initializer != nil {
				valueType := c.checkExprWithExpected(field.Initializer, fieldType)
				c.requireAssignable(valueType, fieldType, exprSpan(field.Initializer), "type_mismatch", "cannot assign "+valueType.String()+" to "+fieldType.String())
			}
		}
		for _, assignment := range enumCase.Assignments {
			field, ok := sharedFields[assignment.Name]
			if !ok {
				c.addDiagnostic("unknown_member", "unknown shared enum field '"+assignment.Name+"' in case '"+enumCase.Name+"'", assignment.Span)
				c.checkExpr(assignment.Value)
				continue
			}
			if assigned[assignment.Name] {
				c.addDiagnostic("duplicate_enum_case_assignment", "duplicate assignment to shared enum field '"+assignment.Name+"' in case '"+enumCase.Name+"'", assignment.Span)
				c.checkExpr(assignment.Value)
				continue
			}
			assigned[assignment.Name] = true
			expected := c.resolveDeclaredType(field.Type)
			valueType := c.checkExprWithExpected(assignment.Value, expected)
			c.requireAssignable(valueType, expected, exprSpan(assignment.Value), "type_mismatch", "cannot assign "+valueType.String()+" to "+expected.String())
		}
		for _, field := range info.decl.Fields {
			if field.Initializer == nil && !assigned[field.Name] {
				c.addDiagnostic("invalid_enum_case", "enum case '"+enumCase.Name+"' must initialize shared field '"+field.Name+"'", enumCase.Span)
			}
		}
	}
}

func (c *Checker) interfaceMethods(decl *parser.InterfaceDecl, seen map[string]bool) []parser.InterfaceMethod {
	if decl == nil {
		return nil
	}
	key := decl.Name
	if seen[key] {
		return nil
	}
	seen[key] = true
	var methods []parser.InterfaceMethod
	added := map[string]bool{}
	for _, parent := range decl.Extends {
		info, ok := c.interfaces[parent.Name]
		if !ok {
			continue
		}
		for _, method := range c.interfaceMethods(info.decl, seen) {
			sigKey := interfaceMethodKey(method)
			if added[sigKey] {
				continue
			}
			added[sigKey] = true
			methods = append(methods, method)
		}
	}
	for _, method := range decl.Methods {
		sigKey := interfaceMethodKey(method)
		if added[sigKey] {
			continue
		}
		added[sigKey] = true
		methods = append(methods, method)
	}
	return methods
}

func interfaceMethodKey(method parser.InterfaceMethod) string {
	key := method.Name + "("
	for i, param := range method.Parameters {
		if i > 0 {
			key += ","
		}
		key += param.Type.Name
		for _, arg := range param.Type.Arguments {
			key += "[" + arg.Name + "]"
		}
	}
	key += "):"
	if method.ReturnType != nil {
		key += method.ReturnType.Name
		for _, arg := range method.ReturnType.Arguments {
			key += "[" + arg.Name + "]"
		}
	}
	return key
}

func (c *Checker) lookupInterfaceMethodInfo(decl *parser.InterfaceDecl, name string, seen map[string]bool) (interfaceMethodInfo, bool) {
	if decl == nil {
		return interfaceMethodInfo{}, false
	}
	key := decl.Name
	if seen[key] {
		return interfaceMethodInfo{}, false
	}
	seen[key] = true
	for _, method := range decl.Methods {
		if method.Name == name {
			return interfaceMethodInfo{decl: method}, true
		}
	}
	for _, parent := range decl.Extends {
		info, ok := c.interfaces[parent.Name]
		if !ok {
			continue
		}
		if method, ok := c.lookupInterfaceMethodInfo(info.decl, name, seen); ok {
			return method, true
		}
	}
	return interfaceMethodInfo{}, false
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
	if !method.decl.Impl {
		c.addDiagnostic("interface_method_requires_impl", "method '"+class.decl.Name+"."+method.decl.Name+"' must use 'impl def' because it implements interface 'Eq'", method.decl.Span)
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

func (c *Checker) bindingValueTypes(bindings []parser.Binding, values []parser.Expr, span parser.Span) []*Type {
	if len(bindings) == 0 || len(values) == 0 {
		return nil
	}
	if len(bindings) == len(values) {
		out := make([]*Type, len(values))
		for i, value := range values {
			if value == nil {
				out[i] = nil
				continue
			}
			expected := unknownType
			if bindings[i].Type != nil {
				expected = c.resolveDeclaredType(bindings[i].Type)
			}
			out[i] = c.checkExprWithExpected(value, expected)
		}
		return out
	}
	if len(values) == 1 {
		valueType := c.checkExpr(values[0])
		return c.destructureValueTypes(len(bindings), valueType, span, "invalid_binding_count", "binding")
	}
	for _, value := range values {
		c.checkExpr(value)
	}
	c.addDiagnostic("invalid_binding_count", fmt.Sprintf("binding expects %d values, got %d", len(bindings), len(values)), span)
	return nil
}

func (c *Checker) assignmentValueTypes(targetCount int, values []parser.Expr, span parser.Span) []*Type {
	if targetCount == len(values) {
		out := make([]*Type, len(values))
		for i, value := range values {
			out[i] = c.checkExpr(value)
		}
		return out
	}
	if len(values) == 1 {
		valueType := c.checkExpr(values[0])
		return c.destructureValueTypes(targetCount, valueType, span, "invalid_assignment_count", "assignment")
	}
	for _, value := range values {
		c.checkExpr(value)
	}
	c.addDiagnostic("invalid_assignment_count", fmt.Sprintf("assignment expects %d values, got %d", targetCount, len(values)), span)
	return nil
}

func (c *Checker) destructurableValueTypes(valueType *Type) ([]*Type, string, bool) {
	if valueType == nil || isUnknown(valueType) {
		return nil, "", false
	}
	if valueType.Kind == TypeTuple {
		return valueType.Args, "tuple", true
	}
	if valueType.Kind != TypeClass {
		return nil, "", false
	}
	info, ok := c.classes[valueType.Name]
	if !ok || info.decl.Enum || info.decl.Object {
		return nil, "", false
	}
	for _, field := range info.decl.Fields {
		if field.Private {
			return nil, "", false
		}
	}
	subst := c.substForDecl(info.decl.TypeParameters, valueType.Args)
	out := make([]*Type, len(info.decl.Fields))
	for i, field := range info.decl.Fields {
		out[i] = c.instantiateTypeRef(field.Type, subst)
	}
	return out, "destructured", true
}

func (c *Checker) destructureValueTypes(count int, valueType *Type, span parser.Span, code string, context string) []*Type {
	parts, kind, ok := c.destructurableValueTypes(valueType)
	if !ok {
		c.addDiagnostic(code, fmt.Sprintf("%s expects %d values, got 1", context, count), span)
		return []*Type{valueType}
	}
	if len(parts) != count {
		c.addDiagnostic(code, fmt.Sprintf("%s expects %d %s values, got %d", context, count, kind, len(parts)), span)
	}
	return parts
}

func (c *Checker) exprHasEffect(expr parser.Expr) bool {
	switch e := expr.(type) {
	case *parser.CallExpr:
		return true
	case *parser.GroupExpr:
		return c.exprHasEffect(e.Inner)
	case *parser.BlockExpr:
		return c.blockHasEffect(e.Body)
	case *parser.IfExpr:
		return c.blockHasEffect(e.Then) || c.blockHasEffect(e.Else)
	case *parser.ForYieldExpr:
		return c.blockHasEffect(e.YieldBody)
	default:
		return false
	}
}

func (c *Checker) blockHasEffect(block *parser.BlockStmt) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *parser.ExprStmt:
			if c.exprHasEffect(s.Expr) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func (c *Checker) defineForBindingParts(bindings []parser.Binding, bindingTypes []*Type) {
	for i, part := range bindings {
		if part.Name == "_" {
			continue
		}
		bindingType := unknownType
		if i < len(bindingTypes) && bindingTypes[i] != nil {
			bindingType = bindingTypes[i]
		}
		if part.Type != nil {
			declType := c.resolveDeclaredType(part.Type)
			c.requireAssignable(bindingType, declType, part.Span, "type_mismatch", "cannot assign "+bindingType.String()+" to "+declType.String())
			bindingType = declType
		}
		c.define(part.Name, bindingType, part.Mutable)
	}
}

func (c *Checker) checkForClause(binding parser.ForBinding) {
	if binding.Iterable != nil {
		iterType := c.checkExpr(binding.Iterable)
		elemType := c.iterableElementType(iterType)
		bindingTypes := []*Type{elemType}
		if len(binding.Bindings) > 1 {
			bindingTypes = c.destructureValueTypes(len(binding.Bindings), elemType, binding.Span, "invalid_binding_count", "for binding")
		}
		c.defineForBindingParts(binding.Bindings, bindingTypes)
		return
	}
	valueTypes := c.bindingValueTypes(binding.Bindings, binding.Values, binding.Span)
	c.defineForBindingParts(binding.Bindings, valueTypes)
}

func (c *Checker) checkStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		valueTypes := c.bindingValueTypes(s.Bindings, s.Values, s.Span)
		for i, bindingDecl := range s.Bindings {
			if bindingDecl.Deferred {
				c.addDiagnostic("invalid_deferred", "binding '"+bindingDecl.Name+"' cannot be initialized with '?' outside class fields", bindingDecl.Span)
			}
			valueType := unknownType
			hasValue := i < len(valueTypes) && valueTypes[i] != nil
			if hasValue {
				expected := unknownType
				if bindingDecl.Type != nil {
					expected = c.resolveDeclaredType(bindingDecl.Type)
				}
				valueType = valueTypes[i]
				if expected != nil && !isUnknown(expected) {
					c.requireAssignable(valueType, expected, bindingDecl.Span, "type_mismatch", "cannot assign "+valueType.String()+" to "+expected.String())
				}
			}
			declType := valueType
			if bindingDecl.Type != nil {
				declType = c.resolveDeclaredType(bindingDecl.Type)
			} else if !hasValue {
				c.addDiagnostic("invalid_deferred", "binding '"+bindingDecl.Name+"' cannot be initialized with '?' outside class fields", bindingDecl.Span)
				declType = unknownType
			}
			if bindingDecl.Name != "_" {
				c.define(bindingDecl.Name, declType, bindingDecl.Mutable)
			}
		}
	case *parser.UnwrapStmt:
		if len(c.returnTypes) == 0 {
			c.addDiagnostic("invalid_unwrap", "unwrap binding used outside callable body", s.Span)
			return
		}
		sourceType := c.checkExpr(s.Value)
		successType, ok := c.unwrappableSuccessType(sourceType)
		if !ok {
			c.addDiagnostic("invalid_unwrap", "unwrap binding requires Unwrappable[T]", exprSpan(s.Value))
			successType = unknownType
		}
		if !c.shortCircuitCompatible(sourceType, c.returnTypes[len(c.returnTypes)-1]) {
			c.addDiagnostic("invalid_unwrap", "unwrap binding requires function return type compatible with "+sourceType.String(), s.Span)
		}
		bindingTypes := []*Type{successType}
		if len(s.Bindings) > 1 {
			bindingTypes = c.destructureValueTypes(len(s.Bindings), successType, s.Span, "invalid_binding_count", "unwrap binding")
		}
		for i, bindingDecl := range s.Bindings {
			if bindingDecl.Name == "_" {
				continue
			}
			bindingType := unknownType
			if i < len(bindingTypes) && bindingTypes[i] != nil {
				bindingType = bindingTypes[i]
			}
			if bindingDecl.Type != nil {
				declType := c.resolveDeclaredType(bindingDecl.Type)
				c.requireAssignable(bindingType, declType, bindingDecl.Span, "type_mismatch", "cannot assign "+bindingType.String()+" to "+declType.String())
				bindingType = declType
			}
			c.define(bindingDecl.Name, bindingType, false)
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
		if s.Function.ReturnType != nil && !isUnknown(implicitReturn) && !isUnitType(expectedReturn) {
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
		valueTypes := c.assignmentValueTypes(len(s.Targets), s.Values, s.Span)
		count := len(s.Targets)
		if len(valueTypes) < count {
			count = len(valueTypes)
		}
		for i := 0; i < count; i++ {
			targetType, mutable := c.checkAssignmentTarget(s.Targets[i], s.Span)
			valueType := valueTypes[i]
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
	case *parser.IfStmt:
		if s.BindingValue != nil {
			optionType := c.checkExpr(s.BindingValue)
			elemType := c.optionElementType(optionType)
			if isUnknown(elemType) {
				c.addDiagnostic("invalid_condition_type", "if binding requires Option[T]", exprSpan(s.BindingValue))
				elemType = unknownType
			}
			bindingTypes := []*Type{elemType}
			if len(s.Bindings) > 1 {
				bindingTypes = c.destructureValueTypes(len(s.Bindings), elemType, s.Span, "invalid_binding_count", "if binding")
			}
			c.pushScope()
			for i, binding := range s.Bindings {
				if binding.Name == "_" {
					continue
				}
				bindingType := unknownType
				if i < len(bindingTypes) && bindingTypes[i] != nil {
					bindingType = bindingTypes[i]
				}
				if binding.Type != nil {
					declType := c.resolveDeclaredType(binding.Type)
					c.requireAssignable(bindingType, declType, binding.Span, "type_mismatch", "cannot assign "+bindingType.String()+" to "+declType.String())
					bindingType = declType
				}
				c.define(binding.Name, bindingType, false)
			}
			c.checkBlockStatements(s.Then.Statements, false)
			c.popScope()
		} else {
			condType := c.checkExpr(s.Condition)
			c.requireAssignable(condType, builtin("Bool"), exprSpan(s.Condition), "invalid_condition_type", "if condition must be Bool")
			c.checkBlockStatements(s.Then.Statements, false)
		}
		if s.ElseIf != nil {
			c.checkStmt(s.ElseIf)
		}
		if s.Else != nil {
			c.checkBlockStatements(s.Else.Statements, false)
		}
	case *parser.MatchStmt:
		valueType := c.checkExpr(s.Value)
		for _, matchCase := range s.Cases {
			c.pushScope()
			c.checkMatchPattern(matchCase.Pattern, valueType)
			if matchCase.Body != nil {
				c.checkBlockStatements(matchCase.Body.Statements, false)
			}
			if matchCase.Expr != nil {
				c.checkExpr(matchCase.Expr)
			}
			c.popScope()
		}
		c.checkMatchExhaustiveness(valueType, s.Cases, s.Span)
	case *parser.LoopStmt:
		c.pushScope()
		if s.Body != nil {
			c.checkBlockStatements(s.Body.Statements, false)
		}
		c.popScope()
	case *parser.ForStmt:
		c.pushScope()
		if s.Condition != nil {
			condType := c.checkExpr(s.Condition)
			c.requireAssignable(condType, builtin("Bool"), exprSpan(s.Condition), "invalid_condition_type", "for condition must be Bool")
		}
		for _, binding := range s.Bindings {
			c.checkForClause(binding)
		}
		if s.Body != nil {
			c.checkBlockStatements(s.Body.Statements, false)
		}
		if s.YieldBody != nil {
			c.checkBlockStatements(s.YieldBody.Statements, true)
		}
		c.popScope()
	case *parser.ReturnStmt:
		if len(c.returnTypes) == 0 {
			c.addDiagnostic("invalid_return", "return used outside callable body", s.Span)
			return
		}
		expected := c.returnTypes[len(c.returnTypes)-1]
		if isUnitType(expected) {
			valueType := c.checkExpr(s.Value)
			c.addDiagnostic("invalid_return_type", "cannot explicitly return "+valueType.String()+" from function returning Unit", s.Span)
			return
		}
		valueType := c.checkExprWithExpected(s.Value, expected)
		if isUnknown(expected) {
			c.returnTypes[len(c.returnTypes)-1] = valueType
			return
		}
		if !isUnknown(expected) {
			c.requireAssignable(valueType, expected, s.Span, "invalid_return_type", "cannot return "+valueType.String()+" from function returning "+expected.String())
		}
	case *parser.ExprStmt:
		before := len(c.diagnostics)
		c.checkExpr(s.Expr)
		if len(c.diagnostics) == before && !c.exprHasEffect(s.Expr) {
			c.addDiagnostic("useless_expression", "expression statement has no effect", s.Span)
		}
	}
}

func (c *Checker) checkMatchPattern(pattern parser.Pattern, valueType *Type) {
	switch p := pattern.(type) {
	case *parser.WildcardPattern:
		return
	case *parser.BindingPattern:
		c.define(p.Name, valueType, false)
	case *parser.TypePattern:
		targetType := c.resolveDeclaredType(p.Target)
		if !isUnknown(valueType) && !c.patternTypeCouldMatch(valueType, targetType) {
			c.addDiagnostic("invalid_match_pattern", "type pattern does not match value type", p.Span)
		}
		if p.Name != "" && p.Name != "_" {
			c.define(p.Name, targetType, false)
		}
	case *parser.LiteralPattern:
		patternType := c.checkExpr(p.Value)
		if !sameType(valueType, patternType) && !isUnknown(valueType) && !isUnknown(patternType) {
			c.addDiagnostic("invalid_match_pattern", "pattern does not match value type", p.Span)
		}
	case *parser.TuplePattern:
		partTypes := c.destructureValueTypes(len(p.Elements), valueType, p.Span, "invalid_match_pattern", "match pattern")
		for i, elem := range p.Elements {
			elemType := unknownType
			if i < len(partTypes) && partTypes[i] != nil {
				elemType = partTypes[i]
			}
			c.checkMatchPattern(elem, elemType)
		}
	case *parser.ConstructorPattern:
		c.checkConstructorPattern(p, valueType)
	}
}

func (c *Checker) checkMatchExhaustiveness(valueType *Type, cases []parser.MatchCase, span parser.Span) {
	if valueType == nil || isUnknown(valueType) || valueType.Kind != TypeClass {
		return
	}
	info, ok := c.classes[valueType.Name]
	if !ok || !info.decl.Enum {
		return
	}
	if len(info.decl.Cases) == 0 {
		return
	}
	covered := map[string]bool{}
	for _, matchCase := range cases {
		if c.patternIsCatchAll(matchCase.Pattern, valueType) {
			return
		}
		if caseName, ok := c.enumCaseNameForPattern(matchCase.Pattern, valueType); ok {
			covered[caseName] = true
		}
	}
	missing := make([]string, 0, len(info.decl.Cases))
	for _, enumCase := range info.decl.Cases {
		if !covered[enumCase.Name] {
			missing = append(missing, enumCase.Name)
		}
	}
	if len(missing) == 0 {
		return
	}
	c.addDiagnostic("non_exhaustive_match", "match does not cover enum cases: "+joinNames(missing), span)
}

func (c *Checker) patternIsCatchAll(pattern parser.Pattern, valueType *Type) bool {
	switch p := pattern.(type) {
	case *parser.WildcardPattern:
		return true
	case *parser.BindingPattern:
		return true
	case *parser.TypePattern:
		targetType := c.resolveDeclaredType(p.Target)
		return c.isAssignable(valueType, targetType)
	default:
		return false
	}
}

func (c *Checker) enumCaseNameForPattern(pattern parser.Pattern, valueType *Type) (string, bool) {
	constructor, ok := pattern.(*parser.ConstructorPattern)
	if !ok {
		return "", false
	}
	info, ok := c.classes[valueType.Name]
	if !ok || !info.decl.Enum {
		return "", false
	}
	switch len(constructor.Path) {
	case 1:
		if _, ok := info.enumCases[constructor.Path[0]]; ok {
			return constructor.Path[0], true
		}
	case 2:
		if constructor.Path[0] != valueType.Name {
			return "", false
		}
		if _, ok := info.enumCases[constructor.Path[1]]; ok {
			return constructor.Path[1], true
		}
	}
	return "", false
}

func (c *Checker) patternTypeCouldMatch(valueType, targetType *Type) bool {
	if isUnknown(valueType) || isUnknown(targetType) {
		return true
	}
	if sameType(valueType, targetType) {
		return true
	}
	if valueType.Kind == TypeClass && targetType.Kind == TypeClass {
		if c.isAssignable(valueType, targetType) || c.isAssignable(targetType, valueType) {
			return true
		}
	}
	if valueType.Kind == TypeClass && targetType.Kind == TypeInterface {
		if c.isAssignable(valueType, targetType) {
			return true
		}
	}
	if valueType.Kind == TypeInterface && targetType.Kind == TypeClass {
		if c.isAssignable(targetType, valueType) {
			return true
		}
	}
	return false
}

func (c *Checker) checkConstructorPattern(pattern *parser.ConstructorPattern, valueType *Type) {
	if valueType == nil || isUnknown(valueType) || valueType.Kind != TypeClass {
		c.addDiagnostic("invalid_match_pattern", "constructor pattern requires class, record, or enum value", pattern.Span)
		return
	}
	info, ok := c.classes[valueType.Name]
	if !ok {
		c.addDiagnostic("invalid_match_pattern", "constructor pattern requires class, record, or enum value", pattern.Span)
		return
	}
	if info.decl.Enum {
		caseName := ""
		switch len(pattern.Path) {
		case 1:
			caseName = pattern.Path[0]
		case 2:
			if pattern.Path[0] != valueType.Name {
				c.addDiagnostic("invalid_match_pattern", "constructor pattern does not match value type", pattern.Span)
				return
			}
			caseName = pattern.Path[1]
		default:
			c.addDiagnostic("invalid_match_pattern", "unsupported constructor pattern", pattern.Span)
			return
		}
		var enumCase *parser.EnumCaseDecl
		for i := range info.decl.Cases {
			if info.decl.Cases[i].Name == caseName {
				enumCase = &info.decl.Cases[i]
				break
			}
		}
		if enumCase == nil {
			c.addDiagnostic("invalid_match_pattern", "unknown enum case '"+caseName+"'", pattern.Span)
			return
		}
		if len(pattern.Args) != len(enumCase.Fields) {
			c.addDiagnostic("invalid_match_pattern", fmt.Sprintf("enum case '%s' expects %d pattern arguments, got %d", caseName, len(enumCase.Fields), len(pattern.Args)), pattern.Span)
			return
		}
		subst := c.substForDecl(info.decl.TypeParameters, valueType.Args)
		for i, arg := range pattern.Args {
			fieldType := c.instantiateTypeRef(enumCase.Fields[i].Type, subst)
			c.checkMatchPattern(arg, fieldType)
		}
		return
	}
	if len(pattern.Path) != 1 || pattern.Path[0] != valueType.Name {
		c.addDiagnostic("invalid_match_pattern", "constructor pattern does not match value type", pattern.Span)
		return
	}
	fieldTypes, _, ok := c.destructurableValueTypes(valueType)
	if !ok {
		c.addDiagnostic("invalid_match_pattern", "constructor pattern requires destructurable class or record", pattern.Span)
		return
	}
	if len(pattern.Args) != len(fieldTypes) {
		c.addDiagnostic("invalid_match_pattern", fmt.Sprintf("pattern '%s' expects %d arguments, got %d", valueType.Name, len(fieldTypes), len(pattern.Args)), pattern.Span)
		return
	}
	for i, arg := range pattern.Args {
		c.checkMatchPattern(arg, fieldTypes[i])
	}
}

func (c *Checker) checkBlockStatements(statements []parser.Statement, allowTailExpr bool) {
	c.pushScope()
	defer c.popScope()
	for i, stmt := range statements {
		if allowTailExpr && i == len(statements)-1 {
			_ = c.checkStmtResult(stmt, "invalid_tail_expression", "block must end with a value-producing statement")
			return
		}
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
	return c.checkStmtResult(last, code, message)
}

func (c *Checker) checkStmtResult(stmt parser.Statement, code, message string) *Type {
	switch s := stmt.(type) {
	case *parser.ExprStmt:
		return c.checkExpr(s.Expr)
	case *parser.IfStmt:
		return c.checkIfStmtResult(s, code, message)
	case *parser.MatchStmt:
		return c.checkMatchStmtResult(s, code, message)
	case *parser.ForStmt:
		if s.YieldBody != nil {
			return c.checkForStmtResult(s, code, message)
		}
		c.checkStmt(stmt)
		c.addDiagnostic(code, message, stmtSpan(stmt))
		return unknownType
	default:
		c.checkStmt(stmt)
		c.addDiagnostic(code, message, stmtSpan(stmt))
		return unknownType
	}
}

func (c *Checker) checkIfStmtResult(s *parser.IfStmt, code, message string) *Type {
	var thenType *Type
	if s.BindingValue != nil {
		optionType := c.checkExpr(s.BindingValue)
		elemType := c.optionElementType(optionType)
		if isUnknown(elemType) {
			c.addDiagnostic("invalid_condition_type", "if binding requires Option[T]", exprSpan(s.BindingValue))
			elemType = unknownType
		}
		bindingTypes := []*Type{elemType}
		if len(s.Bindings) > 1 {
			bindingTypes = c.destructureValueTypes(len(s.Bindings), elemType, s.Span, "invalid_binding_count", "if binding")
		}
		c.pushScope()
		for i, binding := range s.Bindings {
			if binding.Name == "_" {
				continue
			}
			bindingType := unknownType
			if i < len(bindingTypes) && bindingTypes[i] != nil {
				bindingType = bindingTypes[i]
			}
			if binding.Type != nil {
				declType := c.resolveDeclaredType(binding.Type)
				c.requireAssignable(bindingType, declType, binding.Span, "type_mismatch", "cannot assign "+bindingType.String()+" to "+declType.String())
				bindingType = declType
			}
			c.define(binding.Name, bindingType, false)
		}
		thenType = c.checkBlockResult(s.Then, code, message)
		c.popScope()
	} else {
		condType := c.checkExpr(s.Condition)
		c.requireAssignable(condType, builtin("Bool"), exprSpan(s.Condition), "invalid_condition_type", "if condition must be Bool")
		thenType = c.checkBlockResult(s.Then, code, message)
	}

	var elseType *Type
	switch {
	case s.ElseIf != nil:
		elseType = c.checkIfStmtResult(s.ElseIf, code, message)
	case s.Else != nil:
		elseType = c.checkBlockResult(s.Else, code, message)
	default:
		c.addDiagnostic(code, message, s.Span)
		return unknownType
	}
	if !sameType(thenType, elseType) {
		c.addDiagnostic("type_mismatch", "if branches must have the same type", s.Span)
		return unknownType
	}
	return thenType
}

func (c *Checker) checkMatchStmtResult(s *parser.MatchStmt, code, message string) *Type {
	valueType := c.checkExpr(s.Value)
	var resultType *Type
	for _, matchCase := range s.Cases {
		c.pushScope()
		c.checkMatchPattern(matchCase.Pattern, valueType)
		caseType := unknownType
		if matchCase.Body != nil {
			caseType = c.checkBlockResult(matchCase.Body, code, message)
		} else if matchCase.Expr != nil {
			caseType = c.checkExpr(matchCase.Expr)
		} else {
			c.addDiagnostic(code, message, matchCase.Span)
		}
		c.popScope()
		if resultType == nil {
			resultType = caseType
			continue
		}
		if !sameType(resultType, caseType) {
			c.addDiagnostic("type_mismatch", "match cases must have the same type", s.Span)
			resultType = unknownType
		}
	}
	c.checkMatchExhaustiveness(valueType, s.Cases, s.Span)
	if resultType == nil {
		c.addDiagnostic(code, message, s.Span)
		return unknownType
	}
	return resultType
}

func (c *Checker) checkForStmtResult(s *parser.ForStmt, code, message string) *Type {
	if s.YieldBody == nil {
		c.addDiagnostic(code, message, s.Span)
		return unknownType
	}
	c.pushScope()
	for _, binding := range s.Bindings {
		c.checkForClause(binding)
	}
	yieldType := c.checkBlockResult(s.YieldBody, code, message)
	c.popScope()
	return &Type{Kind: TypeInterface, Name: "List", Args: []*Type{yieldType}}
}

func (c *Checker) checkExpr(expr parser.Expr) *Type {
	return c.checkExprWithExpected(expr, nil)
}

func (c *Checker) checkExprWithExpected(expr parser.Expr, expected *Type) *Type {
	originalExpr := expr
	if expected != nil && expected.Kind == TypeFunction && expected.Signature != nil &&
		len(expected.Signature.Parameters) == 1 && parser.HasPlaceholderExpr(expr) {
		expr = parser.WrapPlaceholderLambdaExpr(expr)
	}
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
		if fieldType, ok := c.currentFieldType(e.Name); ok {
			result = fieldType
			break
		}
			if _, ok := c.imports[e.Name]; ok {
				result = &Type{Kind: TypeModule, Name: e.Name}
				break
			}
			if class, ok := c.importedClasses[e.Name]; ok {
				result = &Type{Kind: TypeClass, Name: class.name}
				break
			}
			if _, ok := c.importedInterfaces[e.Name]; ok {
				result = &Type{Kind: TypeInterface, Name: c.importedInterfaceNames[e.Name]}
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
		result = builtin("Str")
	case *parser.UnitLiteral:
		result = builtin("Unit")
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
	case *parser.TupleLiteral:
		elements := make([]*Type, len(e.Elements))
		for i, elem := range e.Elements {
			elements[i] = c.checkExpr(elem)
		}
		result = &Type{Kind: TypeTuple, Name: "Tuple", Args: elements}
	case *parser.GroupExpr:
		result = c.checkExpr(e.Inner)
	case *parser.BlockExpr:
		result = c.checkBlockResult(e.Body, "invalid_block_expression", "block expression must end with an expression")
	case *parser.UnaryExpr:
		right := c.checkExpr(e.Right)
		switch e.Operator {
		case "!":
			c.requireAssignable(right, builtin("Bool"), e.Span, "invalid_unary_operand", "operator ! requires Bool")
			result = builtin("Bool")
		case "-":
			if overloaded, ok := c.resolveOperatorExprType(right, "-", nil, e.Span); ok {
				result = overloaded
				break
			}
			if !isNumeric(right) {
				c.addDiagnostic("invalid_unary_operand", "operator - requires numeric operand", e.Span)
			}
			result = right
		case "~":
			if overloaded, ok := c.resolveOperatorExprType(right, "~", nil, e.Span); ok {
				result = overloaded
				break
			}
			c.addDiagnostic("invalid_unary_operand", "operator ~ requires an overloaded operand", e.Span)
			result = unknownType
		default:
			result = unknownType
		}
	case *parser.BinaryExpr:
		left := c.checkExpr(e.Left)
		right := c.checkExpr(e.Right)
		result = c.checkBinaryOperation(left, right, e.Operator, e.Span)
	case *parser.IsExpr:
		c.checkExpr(e.Left)
		c.resolveDeclaredType(e.Target)
		result = builtin("Bool")
	case *parser.CallExpr:
		result = c.checkCall(e)
	case *parser.MemberExpr:
		result = c.checkMemberExpr(e)
	case *parser.IndexExpr:
		result = c.checkIndexExpr(e)
	case *parser.RecordUpdateExpr:
		result = c.checkRecordUpdateExpr(e)
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
	case *parser.MatchExpr:
		valueType := c.checkExpr(e.Value)
		var resultType *Type
		for _, matchCase := range e.Cases {
			c.pushScope()
			c.checkMatchPattern(matchCase.Pattern, valueType)
			caseType := unknownType
			if matchCase.Body != nil {
				caseType = c.checkBlockResult(matchCase.Body, "invalid_match_expression", "match case must end with an expression")
			} else if matchCase.Expr != nil {
				caseType = c.checkExpr(matchCase.Expr)
			}
			c.popScope()
			if resultType == nil {
				resultType = caseType
				continue
			}
			if !sameType(resultType, caseType) {
				c.addDiagnostic("type_mismatch", "match expression cases must have the same type", e.Span)
				resultType = unknownType
			}
		}
		c.checkMatchExhaustiveness(valueType, e.Cases, e.Span)
		if resultType == nil {
			result = unknownType
			break
		}
		result = resultType
	case *parser.ForYieldExpr:
		c.pushScope()
		for _, binding := range e.Bindings {
			c.checkForClause(binding)
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
	if originalExpr != nil {
		c.exprTypes[originalExpr] = result
	}
	if expr != nil && expr != originalExpr {
		c.exprTypes[expr] = result
	}
	return result
}

func (c *Checker) checkCall(call *parser.CallExpr) *Type {
	if ident, ok := call.Callee.(*parser.Identifier); ok {
		if ident.Name == "this" && c.currentClass != nil && c.currentMethod != nil && c.currentMethod.Constructor {
			info := c.classes[c.currentClass.Name]
			classType := &Type{Kind: TypeClass, Name: c.currentClass.Name}
			if hasNamedCallArgs(call.Args) {
				if ctor, reordered, ok := c.resolveNamedConstructorOverload(info, call.Args, call.Span); ok {
					sig := c.primaryConstructorSignature(info.decl)
					if ctor != nil {
						sig = c.instantiateMethodSignature(ctor, info.decl, nil)
					}
					for i := range reordered {
						if expected, ok := paramTypeForArg(sig, i); ok {
							argType := c.checkExprWithExpected(reordered[i], expected)
							c.requireAssignable(argType, expected, exprSpan(reordered[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
						}
					}
				}
			} else {
				argTypes := c.checkArgTypes(callArgValues(call.Args))
				if ctor, ok := c.resolveConstructorOverload(info, argTypes, call.Span); ok {
					sig := c.primaryConstructorSignature(info.decl)
					if ctor != nil {
						sig = c.instantiateMethodSignature(ctor, info.decl, nil)
					}
					for i, arg := range callArgValues(call.Args) {
						if expected, ok := paramTypeForArg(sig, i); ok {
							argType := c.checkExprWithExpected(arg, expected)
							c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
						}
					}
				}
			}
			return classType
		}
		if isBuiltinValue(ident.Name) {
			return c.checkBuiltinConstructorCall(ident.Name, call)
		}
			if class, ok := c.classes[ident.Name]; ok {
				if class.decl.Object {
					return c.checkApplyCall(class, &Type{Kind: TypeClass, Name: class.name}, call.Args, call.Span, "object '"+class.decl.Name+"' is not callable")
				}
				return c.checkConstructorCall(class, call)
			}
			if class, ok := c.importedClasses[ident.Name]; ok {
				if class.decl.Object {
					return c.checkApplyCall(class, &Type{Kind: TypeClass, Name: class.name}, call.Args, call.Span, "object '"+ident.Name+"' is not callable")
				}
				return c.checkConstructorCall(class, call)
			}
			if fnDecl, ok := c.functionDecls[ident.Name]; ok {
			orderedArgs := callArgValues(call.Args)
			if hasNamedCallArgs(call.Args) {
				reordered, ok := c.reorderCallArgs(fnDecl.Parameters, call.Args, call.Span, "function '"+ident.Name+"'")
				if !ok {
					c.checkArgTypes(callArgValues(call.Args))
					return c.instantiateFunctionSignature(fnDecl, nil).ReturnType
				}
				orderedArgs = reordered
			}
			if len(fnDecl.TypeParameters) == 0 {
				sig := c.functions[ident.Name]
				if !validArgCount(sig, len(orderedArgs)) {
					c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %s arguments, got %d", expectedArgCount(sig), len(orderedArgs)), call.Span)
				}
				for i, arg := range orderedArgs {
					if expected, ok := paramTypeForArg(sig, i); ok {
						argType := c.checkExprWithExpected(arg, expected)
						c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
						continue
					}
					c.checkExpr(arg)
				}
				return sig.ReturnType
			}
			sig, ok := c.resolveFunctionCallSignature(fnDecl, orderedArgs, call.Span)
			if !ok {
				return unknownType
			}
			if !validArgCount(sig, len(orderedArgs)) {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %s arguments, got %d", expectedArgCount(sig), len(orderedArgs)), call.Span)
			}
			for i, arg := range orderedArgs {
				if expected, ok := paramTypeForArg(sig, i); ok {
					argType := c.checkExprWithExpected(arg, expected)
					c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
					continue
				}
				c.checkExpr(arg)
			}
			return sig.ReturnType
		}
	}
	if member, ok := call.Callee.(*parser.MemberExpr); ok {
		return c.checkMethodCall(member, call.Args)
	}

	calleeType := c.checkExpr(call.Callee)
	if calleeType.Kind == TypeClass {
		if info, ok := c.classes[calleeType.Name]; ok {
			return c.checkInstanceApplyCall(info, calleeType, call.Args, call.Span)
		}
	}
	if hasNamedCallArgs(call.Args) {
		for _, arg := range call.Args {
			c.checkExpr(arg.Value)
		}
		c.addDiagnostic("invalid_named_argument", "named arguments require a direct function, method, constructor, or callable object", call.Span)
		return unknownType
	}
	if calleeType.Kind != TypeFunction || calleeType.Signature == nil {
		for _, arg := range call.Args {
			c.checkExpr(arg.Value)
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
			argType = c.checkExprWithExpected(arg.Value, expected)
			c.requireAssignable(argType, expected, exprSpan(arg.Value), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			continue
		}
		argType = c.checkExpr(arg.Value)
	}
	return sig.ReturnType
}

func (c *Checker) checkApplyCall(class classInfo, receiverType *Type, args []parser.CallArg, span parser.Span, missingMessage string) *Type {
	applyMethods, ok := class.methods["apply"]
	if !ok || len(applyMethods) == 0 {
		for _, arg := range args {
			c.checkExpr(arg.Value)
		}
		c.addDiagnostic("invalid_call_target", missingMessage, span)
		return unknownType
	}
	var (
		method      methodInfo
		okMethod    bool
		orderedArgs []parser.Expr
	)
	if hasNamedCallArgs(args) {
		method, orderedArgs, okMethod = c.resolveNamedMethodOverload(class, receiverType, "apply", args, span)
	} else {
		orderedArgs = callArgValues(args)
		argTypes := c.checkArgTypes(orderedArgs)
		method, okMethod = c.resolveMethodOverload(class, receiverType, "apply", argTypes, span)
	}
	if !okMethod {
		return unknownType
	}
	sig, ok := c.resolveMethodCallSignature(class, receiverType, method.decl, orderedArgs, span)
	if !ok {
		return unknownType
	}
	for i := range orderedArgs {
		if expected, ok := paramTypeForArg(sig, i); ok {
			argType := c.checkExprWithExpected(orderedArgs[i], expected)
			c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
		} else {
			c.checkExpr(orderedArgs[i])
		}
	}
	return sig.ReturnType
}

func (c *Checker) resolveFunctionCallSignature(fn *parser.FunctionDecl, args []parser.Expr, span parser.Span) (Signature, bool) {
	sig := c.instantiateFunctionSignature(fn, nil)
	argCount := len(args)
	if len(fn.TypeParameters) == 0 {
		argTypes := c.checkArgTypes(args)
		if !signatureMatches(sig, argTypes) {
			c.addDiagnostic("no_matching_overload", fmt.Sprintf("function '%s' does not match %d arguments", fn.Name, len(argTypes)), span)
			return Signature{}, false
		}
		return sig, true
	}
	inferred, ok := c.inferCallableTypeArgsFromExprs(fn.TypeParameters, fn.Parameters, args, nil)
	if !ok {
		c.addDiagnostic("cannot_infer_type_args", "cannot infer type arguments for function '"+fn.Name+"'", span)
		return Signature{}, false
	}
	if !c.checkTypeArgBounds(c.typeArgsInOrder(fn.TypeParameters, inferred), fn.TypeParameters, span) {
		return Signature{}, false
	}
	sig = c.instantiateFunctionSignature(fn, inferred)
	argTypes := c.checkArgTypes(args)
	if !signatureMatches(sig, argTypes) {
		c.addDiagnostic("no_matching_overload", fmt.Sprintf("function '%s' does not match %d arguments", fn.Name, argCount), span)
		return Signature{}, false
	}
	return sig, true
}

func (c *Checker) resolveMethodCallSignature(class classInfo, receiver *Type, method *parser.MethodDecl, args []parser.Expr, span parser.Span) (Signature, bool) {
	baseSubst := c.substForDecl(class.decl.TypeParameters, receiver.Args)
	sig := c.instantiateMethodSignature(method, class.decl, baseSubst)
	if len(method.TypeParameters) == 0 {
		argTypes := c.checkArgTypes(args)
		if !signatureMatches(sig, argTypes) {
			c.addDiagnostic("no_matching_overload", fmt.Sprintf("no overload of method '%s' matches %d arguments", method.Name, len(argTypes)), span)
			return Signature{}, false
		}
		return sig, true
	}
	inferred, ok := c.inferCallableTypeArgsFromExprs(method.TypeParameters, method.Parameters, args, baseSubst)
	if !ok {
		c.addDiagnostic("cannot_infer_type_args", "cannot infer type arguments for method '"+method.Name+"'", span)
		return Signature{}, false
	}
	if !c.checkTypeArgBounds(c.typeArgsInOrder(method.TypeParameters, inferred), method.TypeParameters, span) {
		return Signature{}, false
	}
	sig = c.instantiateMethodSignature(method, class.decl, mergeSubst(inferred, baseSubst))
	argTypes := c.checkArgTypes(args)
	if !signatureMatches(sig, argTypes) {
		c.addDiagnostic("no_matching_overload", fmt.Sprintf("no overload of method '%s' matches %d arguments", method.Name, len(argTypes)), span)
		return Signature{}, false
	}
	return sig, true
}

func (c *Checker) inferCallableTypeArgsFromExprs(typeParams []parser.TypeParameter, params []parser.Parameter, args []parser.Expr, baseSubst map[string]*Type) (map[string]*Type, bool) {
	if len(typeParams) == 0 {
		return nil, true
	}
	if len(params) != len(args) {
		return nil, false
	}
	typeParamNames := map[string]bool{}
	for _, param := range typeParams {
		typeParamNames[param.Name] = true
	}
	templateSubst := mergeSubst(c.substForDecl(typeParams, nil), baseSubst)
	inferred := map[string]*Type{}
	for i, param := range params {
		template := c.instantiateTypeRef(param.Type, templateSubst)
		contextual := replaceTypeParamsWithUnknown(template, typeParamNames)
		argType := c.checkExprWithExpected(args[i], contextual)
		if !inferTypeArgsFromTypes(argType, template, inferred, typeParamNames) {
			return nil, false
		}
	}
	return inferred, true
}

func (c *Checker) inferCallableTypeArgs(typeParams []parser.TypeParameter, params []parser.Parameter, argTypes []*Type, baseSubst map[string]*Type) (map[string]*Type, bool) {
	if len(typeParams) == 0 {
		return nil, true
	}
	if len(params) != len(argTypes) {
		return nil, false
	}
	typeParamNames := map[string]bool{}
	for _, param := range typeParams {
		typeParamNames[param.Name] = true
	}
	templateSubst := mergeSubst(c.substForDecl(typeParams, nil), baseSubst)
	inferred := map[string]*Type{}
	for i, param := range params {
		template := c.instantiateTypeRef(param.Type, templateSubst)
		if !inferTypeArgsFromTypes(argTypes[i], template, inferred, typeParamNames) {
			return nil, false
		}
	}
	for _, param := range typeParams {
		if _, ok := inferred[param.Name]; !ok {
			return nil, false
		}
	}
	return inferred, true
}

func replaceTypeParamsWithUnknown(t *Type, names map[string]bool) *Type {
	if t == nil {
		return nil
	}
	if t.Kind == TypeParam && names[t.Name] {
		return unknownType
	}
	out := *t
	if len(t.Args) > 0 {
		out.Args = make([]*Type, len(t.Args))
		for i, arg := range t.Args {
			out.Args[i] = replaceTypeParamsWithUnknown(arg, names)
		}
	}
	if t.Signature != nil {
		params := make([]*Type, len(t.Signature.Parameters))
		for i, param := range t.Signature.Parameters {
			params[i] = replaceTypeParamsWithUnknown(param, names)
		}
		sig := *t.Signature
		sig.Parameters = params
		sig.ReturnType = replaceTypeParamsWithUnknown(t.Signature.ReturnType, names)
		out.Signature = &sig
	}
	return &out
}

func inferTypeArgsFromTypes(actual, template *Type, inferred map[string]*Type, typeParams map[string]bool) bool {
	if isUnknown(actual) || isUnknown(template) {
		return true
	}
	if template.Kind == TypeParam && typeParams[template.Name] {
		if existing, ok := inferred[template.Name]; ok {
			return sameType(existing, actual)
		}
		inferred[template.Name] = actual
		return true
	}
	if template.Kind == TypeFunction && actual.Kind == TypeFunction && template.Signature != nil && actual.Signature != nil {
		if len(template.Signature.Parameters) != len(actual.Signature.Parameters) {
			return false
		}
		for i := range template.Signature.Parameters {
			if !inferTypeArgsFromTypes(actual.Signature.Parameters[i], template.Signature.Parameters[i], inferred, typeParams) {
				return false
			}
		}
		return inferTypeArgsFromTypes(actual.Signature.ReturnType, template.Signature.ReturnType, inferred, typeParams)
	}
	if template.Kind == TypeTuple && actual.Kind == TypeTuple && len(template.Args) == len(actual.Args) {
		for i := range template.Args {
			if !inferTypeArgsFromTypes(actual.Args[i], template.Args[i], inferred, typeParams) {
				return false
			}
		}
		return true
	}
	if template.Kind == actual.Kind && template.Name == actual.Name && len(template.Args) == len(actual.Args) {
		for i := range template.Args {
			if !inferTypeArgsFromTypes(actual.Args[i], template.Args[i], inferred, typeParams) {
				return false
			}
		}
	}
	return true
}

func (c *Checker) checkInstanceApplyCall(class classInfo, receiverType *Type, args []parser.CallArg, span parser.Span) *Type {
	return c.checkApplyCall(class, receiverType, args, span, "value of type '"+receiverType.String()+"' is not callable")
}

func (c *Checker) checkBuiltinConstructorCall(name string, call *parser.CallExpr) *Type {
	if hasNamedCallArgs(call.Args) {
		for _, arg := range call.Args {
			c.checkExpr(arg.Value)
		}
		c.addDiagnostic("invalid_named_argument", "named arguments are not supported for builtin constructors", call.Span)
		return unknownType
	}
	switch name {
	case "List":
		if len(call.Args) == 0 {
			return &Type{Kind: TypeInterface, Name: "List", Args: []*Type{unknownType}}
		}
		elemType := c.checkExpr(call.Args[0].Value)
		for _, arg := range call.Args[1:] {
			argType := c.checkExpr(arg.Value)
			if !sameType(elemType, argType) {
				c.addDiagnostic("type_mismatch", "List constructor arguments must have the same type", exprSpan(arg.Value))
			}
		}
		return &Type{Kind: TypeInterface, Name: "List", Args: []*Type{elemType}}
	case "Set":
		if len(call.Args) == 0 {
			return &Type{Kind: TypeInterface, Name: "Set", Args: []*Type{unknownType}}
		}
		elemType := c.checkExpr(call.Args[0].Value)
		for _, arg := range call.Args[1:] {
			argType := c.checkExpr(arg.Value)
			if !sameType(elemType, argType) {
				c.addDiagnostic("type_mismatch", "Set constructor arguments must have the same type", exprSpan(arg.Value))
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
			pair, ok := arg.Value.(*parser.BinaryExpr)
			if !ok || pair.Operator != ":" {
				c.addDiagnostic("invalid_argument_type", "Map constructor expects key : value pairs", exprSpan(arg.Value))
				c.checkExpr(arg.Value)
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
				c.checkExpr(arg.Value)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Array constructor expects 1 argument, got %d", len(call.Args)), call.Span)
			return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
		}
		lengthType := c.checkExpr(call.Args[0].Value)
		c.requireAssignable(lengthType, builtin("Int"), exprSpan(call.Args[0].Value), "invalid_argument_type", "Array constructor length must be Int")
		return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
		case "Some":
			optionType := &Type{Kind: TypeInterface, Name: "Option", Args: []*Type{unknownType}}
			if _, ok := c.classes["Option"]; ok {
				optionType = &Type{Kind: TypeClass, Name: "Option", Args: []*Type{unknownType}}
		}
		if len(call.Args) != 1 {
			for _, arg := range call.Args {
				c.checkExpr(arg.Value)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Some constructor expects 1 argument, got %d", len(call.Args)), call.Span)
			return optionType
		}
		valueType := c.checkExpr(call.Args[0].Value)
		optionType.Args = []*Type{valueType}
		return optionType
	case "None":
		optionType := &Type{Kind: TypeInterface, Name: "Option", Args: []*Type{unknownType}}
		if _, ok := c.classes["Option"]; ok {
			optionType = &Type{Kind: TypeClass, Name: "Option", Args: []*Type{unknownType}}
		}
		if len(call.Args) != 0 {
			for _, arg := range call.Args {
				c.checkExpr(arg.Value)
			}
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("None constructor expects 0 arguments, got %d", len(call.Args)), call.Span)
			}
			return optionType
		case "Ok":
			resultType := &Type{Kind: TypeInterface, Name: "Result", Args: []*Type{unknownType, unknownType}}
			if _, ok := c.classes["Result"]; ok {
				resultType = &Type{Kind: TypeClass, Name: "Result", Args: []*Type{unknownType, unknownType}}
			}
			if len(call.Args) != 1 {
				for _, arg := range call.Args {
					c.checkExpr(arg.Value)
				}
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Ok constructor expects 1 argument, got %d", len(call.Args)), call.Span)
				return resultType
			}
			valueType := c.checkExpr(call.Args[0].Value)
			resultType.Args = []*Type{valueType, unknownType}
			return resultType
		case "Err":
			resultType := &Type{Kind: TypeInterface, Name: "Result", Args: []*Type{unknownType, unknownType}}
			if _, ok := c.classes["Result"]; ok {
				resultType = &Type{Kind: TypeClass, Name: "Result", Args: []*Type{unknownType, unknownType}}
			}
			if len(call.Args) != 1 {
				for _, arg := range call.Args {
					c.checkExpr(arg.Value)
				}
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Err constructor expects 1 argument, got %d", len(call.Args)), call.Span)
				return resultType
			}
			errorType := c.checkExpr(call.Args[0].Value)
			resultType.Args = []*Type{unknownType, errorType}
			return resultType
		case "Left":
			eitherType := &Type{Kind: TypeInterface, Name: "Either", Args: []*Type{unknownType, unknownType}}
			if _, ok := c.classes["Either"]; ok {
				eitherType = &Type{Kind: TypeClass, Name: "Either", Args: []*Type{unknownType, unknownType}}
			}
			if len(call.Args) != 1 {
				for _, arg := range call.Args {
					c.checkExpr(arg.Value)
				}
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Left constructor expects 1 argument, got %d", len(call.Args)), call.Span)
				return eitherType
			}
			leftType := c.checkExpr(call.Args[0].Value)
			eitherType.Args = []*Type{leftType, unknownType}
			return eitherType
		case "Right":
			eitherType := &Type{Kind: TypeInterface, Name: "Either", Args: []*Type{unknownType, unknownType}}
			if _, ok := c.classes["Either"]; ok {
				eitherType = &Type{Kind: TypeClass, Name: "Either", Args: []*Type{unknownType, unknownType}}
			}
			if len(call.Args) != 1 {
				for _, arg := range call.Args {
					c.checkExpr(arg.Value)
				}
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("Right constructor expects 1 argument, got %d", len(call.Args)), call.Span)
				return eitherType
			}
			rightType := c.checkExpr(call.Args[0].Value)
			eitherType.Args = []*Type{unknownType, rightType}
			return eitherType
		default:
			for _, arg := range call.Args {
				c.checkExpr(arg.Value)
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
	if receiverType.Kind == TypeInterface && receiverType.Name == "List" && len(receiverType.Args) == 1 {
		return receiverType.Args[0]
	}
	if receiverType.Kind == TypeInterface && receiverType.Name == "Map" && len(receiverType.Args) == 2 {
		expectedKey := receiverType.Args[0]
		if !sameType(indexType, expectedKey) {
			c.addDiagnostic("type_mismatch", "map index must have key type "+expectedKey.String(), expr.Span)
		}
		return receiverType.Args[1]
	}
	if result, ok := c.resolveOperatorExprType(receiverType, "[]", []*Type{indexType}, expr.Span); ok {
		return result
	}
	c.addDiagnostic("invalid_index_target", "indexing requires Array[T], List[T], Map[K, V], or operator []", expr.Span)
	return unknownType
}

func (c *Checker) checkRecordUpdateExpr(expr *parser.RecordUpdateExpr) *Type {
	receiverType := c.checkExpr(expr.Receiver)
	if isUnknown(receiverType) {
		for _, update := range expr.Updates {
			c.checkExpr(update.Value)
		}
		return unknownType
	}
	if receiverType.Kind != TypeClass {
		for _, update := range expr.Updates {
			c.checkExpr(update.Value)
		}
		c.addDiagnostic("invalid_record_update", "record update requires a record value", expr.Span)
		return unknownType
	}
	info, ok := c.classes[receiverType.Name]
	if !ok || !info.decl.Record {
		for _, update := range expr.Updates {
			c.checkExpr(update.Value)
		}
		c.addDiagnostic("invalid_record_update", "record update requires a record value", expr.Span)
		return unknownType
	}
	subst := c.substForDecl(info.decl.TypeParameters, receiverType.Args)
	seen := map[string]bool{}
	for _, update := range expr.Updates {
		if seen[update.Name] {
			c.addDiagnostic("invalid_record_update", "duplicate record field '"+update.Name+"'", expr.Span)
			c.checkExpr(update.Value)
			continue
		}
		seen[update.Name] = true
		field, ok := info.fields[update.Name]
		if !ok {
			c.addDiagnostic("unknown_member", "unknown record field '"+update.Name+"'", expr.Span)
			c.checkExpr(update.Value)
			continue
		}
		if field.decl.Private && !c.canAccessPrivate(info.decl) {
			c.addDiagnostic("private_access", "cannot access private field '"+update.Name+"' outside class '"+info.decl.Name+"'", expr.Span)
			c.checkExpr(update.Value)
			continue
		}
		expected := c.instantiateTypeRef(field.decl.Type, subst)
		valueType := c.checkExprWithExpected(update.Value, expected)
		c.requireAssignable(valueType, expected, exprSpan(update.Value), "type_mismatch", "cannot assign "+valueType.String()+" to "+expected.String())
	}
	return receiverType
}

func (c *Checker) checkConstructorCall(class classInfo, call *parser.CallExpr) *Type {
	classType := &Type{Kind: TypeClass, Name: class.name}
	if class.decl.Object {
		c.addDiagnostic("invalid_call_target", "object '"+class.decl.Name+"' is a singleton and cannot be called as a constructor", call.Span)
		for _, arg := range call.Args {
			c.checkExpr(arg.Value)
		}
		return classType
	}
	orderedArgs := callArgValues(call.Args)
	if hasNamedCallArgs(call.Args) {
		if ctor, reordered, ok := c.resolveNamedConstructorOverload(class, call.Args, call.Span); ok {
			orderedArgs = reordered
			sig := c.primaryConstructorSignature(class.decl)
			if ctor != nil {
				sig = c.instantiateMethodSignature(ctor, class.decl, constructorTypeArgs(class.decl, call.Callee))
			}
			for i := range orderedArgs {
				if expected, ok := paramTypeForArg(sig, i); ok {
					argType := c.checkExprWithExpected(orderedArgs[i], expected)
					c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
				} else {
					c.checkExpr(orderedArgs[i])
				}
			}
		}
	} else {
		argTypes := c.checkArgTypes(orderedArgs)
		if ctor, ok := c.resolveConstructorOverload(class, argTypes, call.Span); ok {
			sig := c.primaryConstructorSignature(class.decl)
			if ctor != nil {
				sig = c.instantiateMethodSignature(ctor, class.decl, constructorTypeArgs(class.decl, call.Callee))
			}
			for i := range orderedArgs {
				if expected, ok := paramTypeForArg(sig, i); ok {
					argType := c.checkExprWithExpected(orderedArgs[i], expected)
					c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
				} else {
					c.checkExpr(orderedArgs[i])
				}
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

func (c *Checker) checkMethodCall(member *parser.MemberExpr, args []parser.CallArg) *Type {
	receiverType := c.checkExpr(member.Receiver)
	if isUnknown(receiverType) {
		c.checkArgTypes(callArgValues(args))
		return unknownType
	}
	if receiverType.Kind == TypeModule {
		info, ok := c.imports[receiverType.Name]
		if !ok {
			c.checkArgTypes(callArgValues(args))
			return unknownType
		}
		if fnDecl, ok := info.functionDecls[member.Name]; ok {
			orderedArgs := callArgValues(args)
			if hasNamedCallArgs(args) {
				reordered, ok := c.reorderCallArgs(fnDecl.Parameters, args, member.Span, "function '"+member.Name+"'")
				if !ok {
					c.checkArgTypes(callArgValues(args))
					return c.instantiateFunctionSignature(fnDecl, nil).ReturnType
				}
				orderedArgs = reordered
			}
			if len(fnDecl.TypeParameters) == 0 {
				fn := info.functions[member.Name]
				if !validArgCount(fn, len(orderedArgs)) {
					c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %s arguments, got %d", expectedArgCount(fn), len(orderedArgs)), member.Span)
				}
				for i, arg := range orderedArgs {
					if expected, ok := paramTypeForArg(fn, i); ok {
						argType := c.checkExprWithExpected(arg, expected)
						c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
					} else {
						c.checkExpr(arg)
					}
				}
				return fn.ReturnType
			}
			sig, ok := c.resolveFunctionCallSignature(fnDecl, orderedArgs, member.Span)
			if !ok {
				return unknownType
			}
			if !validArgCount(sig, len(orderedArgs)) {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("call expects %s arguments, got %d", expectedArgCount(sig), len(orderedArgs)), member.Span)
			}
			for i, arg := range orderedArgs {
				if expected, ok := paramTypeForArg(sig, i); ok {
					argType := c.checkExprWithExpected(arg, expected)
					c.requireAssignable(argType, expected, exprSpan(arg), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
				} else {
					c.checkExpr(arg)
				}
			}
			return sig.ReturnType
		}
			if class, ok := info.classes[member.Name]; ok {
				if class.decl.Object {
					return c.checkApplyCall(class, &Type{Kind: TypeClass, Name: class.name}, args, member.Span, "object '"+class.decl.Name+"' is not callable")
				}
				call := &parser.CallExpr{Callee: member, Args: args, Span: member.Span}
				return c.checkConstructorCall(class, call)
			}
		c.checkArgTypes(callArgValues(args))
		c.addDiagnostic("unknown_member", "unknown imported member '"+member.Name+"' on module '"+receiverType.Name+"'", member.Span)
		return unknownType
	}
	if receiverType.Kind == TypeBuiltin && receiverType.Name == "Array" {
		if hasNamedCallArgs(args) {
			c.checkArgTypes(callArgValues(args))
			c.addDiagnostic("invalid_named_argument", "named arguments are not supported for Array methods", member.Span)
			return unknownType
		}
		argTypes := c.checkArgTypes(callArgValues(args))
		switch member.Name {
		case "size":
			if len(argTypes) != 0 {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %d arguments, got %d", member.Name, 0, len(argTypes)), member.Span)
			}
			return builtin("Int")
		case "zip":
			if len(argTypes) != 1 {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %d arguments, got %d", member.Name, 1, len(argTypes)), member.Span)
				return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
			}
			if argTypes[0].Kind != TypeBuiltin || argTypes[0].Name != "Array" || len(argTypes[0].Args) != 1 {
				c.addDiagnostic("invalid_argument_type", "zip expects parameter of type Array[T]", member.Span)
				return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{unknownType}}
			}
			elemType := unknownType
			if len(receiverType.Args) == 1 {
				elemType = receiverType.Args[0]
			}
			return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{{Kind: TypeTuple, Name: "Tuple", Args: []*Type{elemType, argTypes[0].Args[0]}}}}
		case "zipWithIndex":
			if len(argTypes) != 0 {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %d arguments, got %d", member.Name, 0, len(argTypes)), member.Span)
			}
			elemType := unknownType
			if len(receiverType.Args) == 1 {
				elemType = receiverType.Args[0]
			}
			return &Type{Kind: TypeBuiltin, Name: "Array", Args: []*Type{{Kind: TypeTuple, Name: "Tuple", Args: []*Type{elemType, builtin("Int")}}}}
		default:
			c.addDiagnostic("unknown_member", "unknown member '"+member.Name+"'", member.Span)
			return unknownType
		}
	}
	if receiverType.Kind == TypeBuiltin && receiverType.Name == "Str" {
		if hasNamedCallArgs(args) {
			c.checkArgTypes(callArgValues(args))
			c.addDiagnostic("invalid_named_argument", "named arguments are not supported for Str methods", member.Span)
			return unknownType
		}
		argTypes := c.checkArgTypes(callArgValues(args))
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
			c.checkArgTypes(callArgValues(args))
			return unknownType
		}
		if info.decl.Enum {
			if enumCase, ok := info.enumCases[member.Name]; ok {
				params := make([]parser.Parameter, len(enumCase.Fields))
				sig := Signature{Parameters: make([]*Type, len(enumCase.Fields)), ReturnType: receiverType}
				for i, field := range enumCase.Fields {
					params[i] = parser.Parameter{Name: field.Name, Type: field.Type, Span: field.Span}
					sig.Parameters[i] = c.resolveDeclaredType(field.Type)
				}
				orderedArgs := callArgValues(args)
				if hasNamedCallArgs(args) {
					reordered, ok := c.reorderCallArgs(params, args, member.Span, "enum case '"+member.Name+"'")
					if !ok {
						c.checkArgTypes(callArgValues(args))
						return receiverType
					}
					orderedArgs = reordered
				}
				if !validArgCount(sig, len(orderedArgs)) {
					c.addDiagnostic("invalid_argument_count", fmt.Sprintf("enum case '%s' expects %s arguments, got %d", member.Name, expectedArgCount(sig), len(orderedArgs)), member.Span)
				}
				for i := range orderedArgs {
					if expected, ok := paramTypeForArg(sig, i); ok {
						argType := c.checkExprWithExpected(orderedArgs[i], expected)
						c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
					} else {
						c.checkExpr(orderedArgs[i])
					}
				}
				return receiverType
			}
		}
		var (
			method      methodInfo
			okMethod    bool
			orderedArgs []parser.Expr
		)
		if hasNamedCallArgs(args) {
			method, orderedArgs, okMethod = c.resolveNamedMethodOverload(info, receiverType, member.Name, args, member.Span)
		} else {
			orderedArgs = callArgValues(args)
			argTypes := c.checkArgTypes(orderedArgs)
			method, okMethod = c.resolveMethodOverload(info, receiverType, member.Name, argTypes, member.Span)
		}
		if !okMethod {
			return unknownType
		}
		sig, ok := c.resolveMethodCallSignature(info, receiverType, method.decl, orderedArgs, member.Span)
		if !ok {
			return unknownType
		}
		for i := range orderedArgs {
			if expected, ok := paramTypeForArg(sig, i); ok {
				argType := c.checkExprWithExpected(orderedArgs[i], expected)
				c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			} else {
				c.checkExpr(orderedArgs[i])
			}
		}
		return sig.ReturnType
	case TypeInterface:
		info, ok := c.interfaces[receiverType.Name]
		if !ok {
			c.checkArgTypes(callArgValues(args))
			return unknownType
		}
		method, ok := c.lookupInterfaceMethodInfo(info.decl, member.Name, map[string]bool{})
		if !ok {
			c.addDiagnostic("unknown_member", "unknown member '"+member.Name+"'", member.Span)
			return unknownType
		}
		orderedArgs := callArgValues(args)
		if hasNamedCallArgs(args) {
			if method.decl.Parameters[len(method.decl.Parameters)-1].Variadic {
				c.checkArgTypes(callArgValues(args))
				c.addDiagnostic("invalid_named_argument", "named arguments are not supported for variadic methods", member.Span)
				return unknownType
			}
			reordered, ok := c.reorderCallArgs(method.decl.Parameters, args, member.Span, "method '"+member.Name+"'")
			if !ok {
				c.checkArgTypes(callArgValues(args))
				return unknownType
			}
			orderedArgs = reordered
		}
		if receiverType.Name == "Term" && (member.Name == "println" || member.Name == "print") {
			for _, arg := range orderedArgs {
				c.checkExpr(arg)
			}
			return receiverType
		}
		sig, ok := c.resolveInterfaceMethodCallSignature(info, receiverType, method.decl, orderedArgs, member.Span)
		if !ok {
			return unknownType
		}
		if !validArgCount(sig, len(orderedArgs)) {
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %s arguments, got %d", member.Name, expectedArgCount(sig), len(orderedArgs)), member.Span)
		}
		for i := range orderedArgs {
			if expected, ok := paramTypeForArg(sig, i); ok {
				argType := c.checkExprWithExpected(orderedArgs[i], expected)
				c.requireAssignable(argType, expected, exprSpan(orderedArgs[i]), "invalid_argument_type", "cannot pass "+argType.String()+" to parameter of type "+expected.String())
			} else {
				c.checkExpr(orderedArgs[i])
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
	destructuredTupleArg := false
	if expected != nil && expected.Kind == TypeFunction && expected.Signature != nil {
		expectedSig = expected.Signature
		if len(expectedSig.Parameters) != len(expr.Parameters) {
			if len(expectedSig.Parameters) == 1 && len(expr.Parameters) > 1 {
				parts, _, ok := c.destructurableValueTypes(expectedSig.Parameters[0])
				if ok && len(parts) == len(expr.Parameters) {
					destructuredTupleArg = true
					params = append([]*Type(nil), parts...)
				} else {
					c.addDiagnostic("invalid_lambda_type", "lambda parameter count does not match expected function type", expr.Span)
					expectedSig = nil
				}
			} else {
				c.addDiagnostic("invalid_lambda_type", "lambda parameter count does not match expected function type", expr.Span)
				expectedSig = nil
			}
		}
	}
	for i, param := range expr.Parameters {
		paramType := unknownType
		if param.Type != nil {
			paramType = c.resolveDeclaredType(param.Type)
		} else if expectedSig != nil {
			if destructuredTupleArg {
				paramType = params[i]
			} else {
			paramType = expectedSig.Parameters[i]
			}
		} else {
			c.addDiagnostic("invalid_lambda_type", "untyped lambda parameters require a contextual function type", param.Span)
		}
		if destructuredTupleArg {
			if param.Type != nil {
				c.requireAssignable(paramType, params[i], param.Span, "invalid_lambda_type", "lambda parameter does not match expected tuple element type")
			}
			params[i] = paramType
		} else {
			params[i] = paramType
		}
		if param.Name != "_" {
			c.define(param.Name, paramType, false)
		}
	}

	returnType := unknownType
	if expr.Body != nil {
		returnType = c.checkExpr(expr.Body)
		if expectedSig != nil && !containsUnknownType(expectedSig.ReturnType) {
			if !isUnitType(expectedSig.ReturnType) {
				c.requireAssignable(returnType, expectedSig.ReturnType, exprSpan(expr.Body), "invalid_lambda_type", "lambda body does not match expected return type")
			}
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
			} else if !isUnitType(returnType) {
				c.requireAssignable(implicitReturn, returnType, expr.BlockBody.Span, "invalid_lambda_type", "lambda body does not match expected return type")
			}
		}
	}
	if destructuredTupleArg && expectedSig != nil {
		return functionType("<lambda>", Signature{Parameters: expectedSig.Parameters, ReturnType: returnType})
	}
	return functionType("<lambda>", Signature{Parameters: params, ReturnType: returnType})
}

func containsUnknownType(t *Type) bool {
	if t == nil {
		return false
	}
	if isUnknown(t) {
		return true
	}
	for _, arg := range t.Args {
		if containsUnknownType(arg) {
			return true
		}
	}
	if t.Signature != nil {
		for _, param := range t.Signature.Parameters {
			if containsUnknownType(param) {
				return true
			}
		}
		if containsUnknownType(t.Signature.ReturnType) {
			return true
		}
	}
	return false
}

func (c *Checker) checkMemberExpr(expr *parser.MemberExpr) *Type {
	receiverType := c.checkExpr(expr.Receiver)
	if receiverType.Kind == TypeModule {
		info, ok := c.imports[receiverType.Name]
		if !ok {
			return unknownType
		}
		if fn, ok := info.functions[expr.Name]; ok {
			return functionType(expr.Name, fn)
		}
		if class, ok := info.classes[expr.Name]; ok {
			return &Type{Kind: TypeClass, Name: class.name}
		}
		c.addDiagnostic("unknown_member", "unknown imported member '"+expr.Name+"' on module '"+receiverType.Name+"'", expr.Span)
		return unknownType
	}
	if receiverType.Kind == TypeClass {
		if info, ok := c.classes[receiverType.Name]; ok && info.decl.Enum {
			if enumCase, ok := info.enumCases[expr.Name]; ok {
				if len(enumCase.Fields) == 0 {
					return receiverType
				}
				params := make([]*Type, len(enumCase.Fields))
				for i, field := range enumCase.Fields {
					params[i] = c.resolveDeclaredType(field.Type)
				}
				return functionType(expr.Name, Signature{Parameters: params, ReturnType: receiverType})
			}
		}
	}
	if memberType, ok := c.lookupMember(receiverType, expr.Name, expr.Span); ok {
		return memberType
	}
	c.addDiagnostic("unknown_member", "unknown member '"+expr.Name+"'", expr.Span)
	return unknownType
}

func (c *Checker) lookupMember(receiver *Type, name string, span parser.Span) (*Type, bool) {
	if isUnknown(receiver) {
		return unknownType, true
	}
	if receiver.Kind == TypeTuple {
		for i, tupleName := range receiver.TupleNames {
			if tupleName == name {
				if i < len(receiver.Args) {
					return receiver.Args[i], true
				}
				return unknownType, true
			}
		}
		return unknownType, false
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
		if method, ok := c.lookupInterfaceMethodInfo(info.decl, name, map[string]bool{}); ok {
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
		if sameType(left, builtin("Str")) || sameType(right, builtin("Str")) {
			return builtin("Str")
		}
		if result, ok := c.resolveOperatorExprType(left, op, []*Type{right}, span); ok {
			return result
		}
		if !isNumeric(left) || !isNumeric(right) {
			c.addDiagnostic("invalid_binary_operand", "operator + requires numeric operands unless one side is Str", span)
			return unknownType
		}
		if !sameType(left, right) {
			c.addDiagnostic("type_mismatch", "arithmetic operands must have the same type", span)
		}
		return left
	case "-", "*", "/", "%":
		if result, ok := c.resolveOperatorExprType(left, op, []*Type{right}, span); ok {
			return result
		}
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
	case ":+":
		if result, ok := c.checkCollectionAppendType(left, right, span); ok {
			return result
		}
		if result, ok := c.resolveOperatorExprType(left, op, []*Type{right}, span); ok {
			return result
		}
		c.addDiagnostic("invalid_binary_operand", "operator :+ requires a collection receiver or matching operator overload", span)
		return unknownType
	case "++":
		if result, ok := c.checkCollectionConcatType(left, right, span); ok {
			return result
		}
		if result, ok := c.resolveOperatorExprType(left, op, []*Type{right}, span); ok {
			return result
		}
		c.addDiagnostic("invalid_binary_operand", "operator ++ requires matching collections or a matching operator overload", span)
		return unknownType
	case ":-", "--", "|", "&", ">>", "<<", "::":
		if result, ok := c.resolveOperatorExprType(left, op, []*Type{right}, span); ok {
			return result
		}
		c.addDiagnostic("invalid_binary_operand", "operator "+op+" requires a matching operator overload", span)
		return unknownType
	default:
		return unknownType
	}
}

func (c *Checker) resolveOperatorExprType(receiver *Type, name string, argTypes []*Type, span parser.Span) (*Type, bool) {
	if isUnknown(receiver) || receiver.Kind != TypeClass {
		return unknownType, false
	}
	class, ok := c.classes[receiver.Name]
	if !ok {
		return unknownType, false
	}
	method, ok := c.findOperatorOverload(class, receiver, name, argTypes, span)
	if !ok {
		return unknownType, false
	}
	subst := c.substForDecl(class.decl.TypeParameters, receiver.Args)
	sig := c.instantiateMethodSignature(method.decl, class.decl, subst)
	return sig.ReturnType, true
}

func (c *Checker) findOperatorOverload(class classInfo, receiver *Type, name string, argTypes []*Type, span parser.Span) (methodInfo, bool) {
	methods, ok := class.methods[name]
	if !ok || len(methods) == 0 {
		return methodInfo{}, false
	}
	subst := c.substForDecl(class.decl.TypeParameters, receiver.Args)
	var matches []methodInfo
	for _, method := range methods {
		if !method.decl.Operator {
			continue
		}
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
		c.addDiagnostic("ambiguous_overload", "operator '"+name+"' is ambiguous", span)
	}
	return methodInfo{}, false
}

func (c *Checker) checkCollectionAppendType(left, right *Type, span parser.Span) (*Type, bool) {
	if isUnknown(left) || isUnknown(right) {
		return unknownType, false
	}
	if left.Kind == TypeInterface && left.Name == "List" && len(left.Args) == 1 {
		c.requireAssignable(right, left.Args[0], span, "type_mismatch", "cannot append "+right.String()+" to List["+left.Args[0].String()+"]")
		return left, true
	}
	if left.Kind == TypeInterface && left.Name == "Set" && len(left.Args) == 1 {
		c.requireAssignable(right, left.Args[0], span, "type_mismatch", "cannot add "+right.String()+" to Set["+left.Args[0].String()+"]")
		return left, true
	}
	return unknownType, false
}

func (c *Checker) checkCollectionConcatType(left, right *Type, span parser.Span) (*Type, bool) {
	if isUnknown(left) || isUnknown(right) {
		return unknownType, false
	}
	if left.Kind == TypeInterface && right.Kind == TypeInterface && left.Name == right.Name && len(left.Args) == len(right.Args) {
		switch left.Name {
		case "List", "Set", "Map":
			for i := range left.Args {
				if !sameType(left.Args[i], right.Args[i]) {
					c.addDiagnostic("type_mismatch", "operator ++ requires matching collection element types", span)
					return unknownType, true
				}
			}
			return left, true
		}
	}
	return unknownType, false
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
	c.addDiagnostic("assign_immutable", "cannot assign to immutable field '"+expr.Name+"' outside constructor", expr.Span)
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
	typ := c.instantiateTypeRef(ref, nil)
	c.validateTypeRefBounds(ref, typ)
	return typ
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
	if len(ref.TupleElements) > 0 {
		args := make([]*Type, len(ref.TupleElements))
		for i, arg := range ref.TupleElements {
			args[i] = c.instantiateTypeRef(arg, subst)
		}
		return &Type{Kind: TypeTuple, Name: "Tuple", Args: args, TupleNames: append([]string(nil), ref.TupleNames...)}
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
	name := ref.Name
	if class, ok := c.importedClasses[ref.Name]; ok {
		name = class.name
	}
	if qualified, ok := c.importedInterfaceNames[ref.Name]; ok {
		name = qualified
	}
	return &Type{Kind: kind, Name: name, Args: args}
}

func (c *Checker) validateTypeParameterBounds(params []parser.TypeParameter) {
	for _, param := range params {
		for _, bound := range param.Bounds {
			boundType := c.resolveDeclaredType(bound)
			if boundType.Kind != TypeInterface {
				c.addDiagnostic("invalid_type_bound", "type parameter '"+param.Name+"' bound must be an interface", bound.Span)
			}
		}
	}
}

func (c *Checker) validateTypeRefBounds(ref *parser.TypeRef, typ *Type) {
	if ref == nil || typ == nil || len(ref.Arguments) == 0 {
		return
	}
	params, ok := c.typeParametersForName(ref.Name)
	if !ok || len(params) == 0 || len(params) != len(typ.Args) {
		return
	}
	c.checkTypeArgBounds(typ.Args, params, ref.Span)
}

func (c *Checker) instantiateMethodSignature(method *parser.MethodDecl, owner *parser.ClassDecl, subst map[string]*Type) Signature {
	effective := mergeSubst(subst, c.substForDecl(owner.TypeParameters, nil))
	effective = mergeSubst(effective, c.substForDecl(method.TypeParameters, nil))
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

func (c *Checker) instantiateFunctionSignature(fn *parser.FunctionDecl, subst map[string]*Type) Signature {
	effective := mergeSubst(subst, c.substForDecl(fn.TypeParameters, nil))
	params := make([]*Type, len(fn.Parameters))
	for i, param := range fn.Parameters {
		params[i] = c.instantiateTypeRef(param.Type, effective)
	}
	return Signature{
		Parameters: params,
		ReturnType: c.instantiateTypeRef(fn.ReturnType, effective),
		Variadic:   len(fn.Parameters) > 0 && fn.Parameters[len(fn.Parameters)-1].Variadic,
	}
}

func (c *Checker) checkConstructorRules(class classInfo) {
	if len(class.constructors) == 0 {
		if missing := c.uninitializedLetFields(class.decl, nil); len(missing) > 0 {
			c.addDiagnostic("constructor_required", "class '"+class.decl.Name+"' requires a constructor to initialize immutable fields: "+joinNames(missing), class.decl.Span)
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
			c.addDiagnostic("uninitialized_field", "constructor 'this' must initialize immutable fields: "+joinNames(missing), ctor.Span)
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
		if constructorVisibleField(field) {
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

func constructorVisibleField(field parser.FieldDecl) bool {
	return !field.Private && field.Initializer == nil
}

func (c *Checker) currentFieldType(name string) (*Type, bool) {
	if c.currentClass == nil {
		return nil, false
	}
	for _, field := range c.currentClass.Fields {
		if field.Name == name {
			return c.resolveDeclaredType(field.Type), true
		}
	}
	return nil, false
}

func primaryConstructorParams(class *parser.ClassDecl) []parser.Parameter {
	params := make([]parser.Parameter, 0, len(class.Fields))
	for _, field := range class.Fields {
		if constructorVisibleField(field) {
			params = append(params, parser.Parameter{Name: field.Name, Type: field.Type, Span: field.Span})
		}
	}
	return params
}

func (c *Checker) primaryConstructorSignature(class *parser.ClassDecl) Signature {
	params := primaryConstructorParams(class)
	out := make([]*Type, len(params))
	for i, param := range params {
		out[i] = c.resolveDeclaredType(param.Type)
	}
	return Signature{Parameters: out, ReturnType: &Type{Kind: TypeClass, Name: class.Name}}
}

func callArgValues(args []parser.CallArg) []parser.Expr {
	values := make([]parser.Expr, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return values
}

func hasNamedCallArgs(args []parser.CallArg) bool {
	for _, arg := range args {
		if arg.Name != "" {
			return true
		}
	}
	return false
}

func tryReorderCallArgs(params []parser.Parameter, args []parser.CallArg) ([]parser.Expr, bool) {
	if len(args) == 0 {
		if len(params) == 0 {
			return []parser.Expr{}, true
		}
		return nil, false
	}
	if len(params) > 0 && params[len(params)-1].Variadic {
		return nil, false
	}
	ordered := make([]parser.Expr, len(params))
	filled := make([]bool, len(params))
	seenNamed := false
	pos := 0
	for _, arg := range args {
		if arg.Name == "" {
			if seenNamed || pos >= len(params) {
				return nil, false
			}
			ordered[pos] = arg.Value
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
		ordered[paramIndex] = arg.Value
		filled[paramIndex] = true
	}
	for _, ok := range filled {
		if !ok {
			return nil, false
		}
	}
	return ordered, true
}

func (c *Checker) reorderCallArgs(params []parser.Parameter, args []parser.CallArg, span parser.Span, callable string) ([]parser.Expr, bool) {
	if len(params) > 0 && params[len(params)-1].Variadic {
		c.addDiagnostic("invalid_named_argument", "named arguments are not supported for variadic "+callable, span)
		return nil, false
	}
	ordered := make([]parser.Expr, len(params))
	filled := make([]bool, len(params))
	seenNamed := false
	pos := 0
	for _, arg := range args {
		if arg.Name == "" {
			if seenNamed {
				c.addDiagnostic("invalid_named_argument", "positional arguments cannot follow named arguments", arg.Span)
				return nil, false
			}
			if pos >= len(params) {
				c.addDiagnostic("invalid_argument_count", fmt.Sprintf("%s expects %d arguments, got %d", callable, len(params), len(args)), span)
				return nil, false
			}
			ordered[pos] = arg.Value
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
		if paramIndex < 0 {
			c.addDiagnostic("unknown_argument", "unknown named argument '"+arg.Name+"'", arg.Span)
			return nil, false
		}
		if filled[paramIndex] {
			c.addDiagnostic("duplicate_argument", "argument '"+arg.Name+"' was provided more than once", arg.Span)
			return nil, false
		}
		ordered[paramIndex] = arg.Value
		filled[paramIndex] = true
	}
	for i, ok := range filled {
		if !ok {
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("missing argument '%s' in %s", params[i].Name, callable), span)
			return nil, false
		}
	}
	return ordered, true
}

func (c *Checker) resolveConstructorOverload(class classInfo, argTypes []*Type, span parser.Span) (*parser.MethodDecl, bool) {
	candidates := make([]*parser.MethodDecl, 0, len(class.constructors))
	for _, ctor := range class.constructors {
		sig := c.instantiateMethodSignature(ctor, class.decl, nil)
		if signatureMatches(sig, argTypes) {
			candidates = append(candidates, ctor)
		}
	}
	primarySig := c.primaryConstructorSignature(class.decl)
	primaryMatches := signatureMatches(primarySig, argTypes)
	if primaryMatches && len(candidates) == 0 {
		return nil, true
	}
	if primaryMatches && len(candidates) > 0 {
		if len(candidates) == 1 {
			return candidates[0], true
		}
		c.addDiagnostic("ambiguous_overload", "constructor call for class '"+class.decl.Name+"' is ambiguous", span)
		return nil, false
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
		if len(method.decl.TypeParameters) > 0 {
			inferred, ok := c.inferCallableTypeArgs(method.decl.TypeParameters, method.decl.Parameters, argTypes, subst)
			if ok {
				sig = c.instantiateMethodSignature(method.decl, class.decl, mergeSubst(inferred, subst))
			}
		}
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

func (c *Checker) resolveNamedConstructorOverload(class classInfo, args []parser.CallArg, span parser.Span) (*parser.MethodDecl, []parser.Expr, bool) {
	var (
		matchCtor  *parser.MethodDecl
		matchArgs  []parser.Expr
		matchCount int
	)
	for _, ctor := range class.constructors {
		ordered, ok := tryReorderCallArgs(ctor.Parameters, args)
		if !ok {
			continue
		}
		argTypes := c.checkArgTypes(ordered)
		sig := c.instantiateMethodSignature(ctor, class.decl, nil)
		if !signatureMatches(sig, argTypes) {
			continue
		}
		matchCtor = ctor
		matchArgs = ordered
		matchCount++
	}
	primaryParams := primaryConstructorParams(class.decl)
	if ordered, ok := tryReorderCallArgs(primaryParams, args); ok {
		argTypes := c.checkArgTypes(ordered)
		if signatureMatches(c.primaryConstructorSignature(class.decl), argTypes) {
			matchArgs = ordered
			matchCount++
		}
	}
	if matchCount == 1 {
		return matchCtor, matchArgs, true
	}
	if matchCount == 2 && matchCtor != nil {
		return matchCtor, matchArgs, true
	}
	if matchCount > 1 {
		c.addDiagnostic("ambiguous_overload", "constructor call for class '"+class.decl.Name+"' is ambiguous", span)
		return nil, nil, false
	}
	if len(class.constructors) == 1 {
		c.reorderCallArgs(class.constructors[0].Parameters, args, span, "constructor '"+class.decl.Name+"'")
		return nil, nil, false
	}
	if len(class.constructors) == 0 {
		c.reorderCallArgs(primaryParams, args, span, "constructor '"+class.decl.Name+"'")
		return nil, nil, false
	}
	c.addDiagnostic("no_matching_overload", fmt.Sprintf("no constructor overload for class '%s' matches %d arguments", class.decl.Name, len(args)), span)
	return nil, nil, false
}

func (c *Checker) resolveNamedMethodOverload(class classInfo, receiver *Type, name string, args []parser.CallArg, span parser.Span) (methodInfo, []parser.Expr, bool) {
	methods, ok := class.methods[name]
	if !ok || len(methods) == 0 {
		c.addDiagnostic("unknown_member", "unknown member '"+name+"'", span)
		return methodInfo{}, nil, false
	}
	subst := c.substForDecl(class.decl.TypeParameters, receiver.Args)
	type candidate struct {
		method methodInfo
		args   []parser.Expr
	}
	var matches []candidate
	for _, method := range methods {
		if method.decl.Private && !c.canAccessPrivate(class.decl) {
			continue
		}
		ordered, ok := tryReorderCallArgs(method.decl.Parameters, args)
		if !ok {
			continue
		}
		argTypes := c.checkArgTypes(ordered)
		sig := c.instantiateMethodSignature(method.decl, class.decl, subst)
		if len(method.decl.TypeParameters) > 0 {
			inferred, ok := c.inferCallableTypeArgs(method.decl.TypeParameters, method.decl.Parameters, argTypes, subst)
			if ok {
				sig = c.instantiateMethodSignature(method.decl, class.decl, mergeSubst(inferred, subst))
			}
		}
		if signatureMatches(sig, argTypes) {
			matches = append(matches, candidate{method: method, args: ordered})
		}
	}
	if len(matches) == 1 {
		return matches[0].method, matches[0].args, true
	}
	if len(matches) > 1 {
		c.addDiagnostic("ambiguous_overload", "method call '"+name+"' is ambiguous", span)
		return methodInfo{}, nil, false
	}
	if hasPrivateOnlyMatch(methods, class.decl, c) {
		c.addDiagnostic("private_access", "cannot access private method '"+name+"' outside class '"+class.decl.Name+"'", span)
		return methodInfo{}, nil, false
	}
	if len(methods) == 1 {
		c.reorderCallArgs(methods[0].decl.Parameters, args, span, "method '"+name+"'")
		return methodInfo{}, nil, false
	}
	c.addDiagnostic("no_matching_overload", fmt.Sprintf("no overload of method '%s' matches %d arguments", name, len(args)), span)
	return methodInfo{}, nil, false
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
	effective := mergeSubst(subst, c.substForDecl(method.TypeParameters, nil))
	params := make([]*Type, len(method.Parameters))
	for i, param := range method.Parameters {
		params[i] = c.instantiateTypeRef(param.Type, effective)
	}
	return Signature{
		Parameters: params,
		ReturnType: c.instantiateTypeRef(method.ReturnType, effective),
		Variadic:   len(method.Parameters) > 0 && method.Parameters[len(method.Parameters)-1].Variadic,
	}
}

func (c *Checker) resolveInterfaceMethodCallSignature(info interfaceInfo, receiver *Type, method parser.InterfaceMethod, args []parser.Expr, span parser.Span) (Signature, bool) {
	baseSubst := c.substForDecl(info.decl.TypeParameters, receiver.Args)
	sig := c.instantiateInterfaceMethodSignature(method, baseSubst)
	if len(method.TypeParameters) == 0 {
		if !validArgCount(sig, len(args)) {
			c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %s arguments, got %d", method.Name, expectedArgCount(sig), len(args)), span)
			return Signature{}, false
		}
		return sig, true
	}
	inferred, ok := c.inferCallableTypeArgsFromExprs(method.TypeParameters, method.Parameters, args, baseSubst)
	if !ok {
		c.addDiagnostic("cannot_infer_type_args", "cannot infer type arguments for method '"+method.Name+"'", span)
		return Signature{}, false
	}
	if !c.checkTypeArgBounds(c.typeArgsInOrder(method.TypeParameters, inferred), method.TypeParameters, span) {
		return Signature{}, false
	}
	sig = c.instantiateInterfaceMethodSignature(method, mergeSubst(inferred, baseSubst))
	if !validArgCount(sig, len(args)) {
		c.addDiagnostic("invalid_argument_count", fmt.Sprintf("method '%s' expects %s arguments, got %d", method.Name, expectedArgCount(sig), len(args)), span)
		return Signature{}, false
	}
	return sig, true
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
	if class, ok := c.importedClasses[name]; ok {
		return &Type{Kind: TypeClass, Name: class.name}, true
	}
	if _, ok := c.interfaces[name]; ok {
		return &Type{Kind: TypeInterface, Name: name}, true
	}
	if _, ok := c.importedInterfaces[name]; ok {
		return &Type{Kind: TypeInterface, Name: c.importedInterfaceNames[name]}, true
	}
	return nil, false
}

func (c *Checker) iterableElementType(t *Type) *Type {
	if isUnknown(t) {
		return unknownType
	}
	if t.Name == "Option" && len(t.Args) == 1 {
		return t.Args[0]
	}
	if t.Name == "Array" && len(t.Args) == 1 {
		return t.Args[0]
	}
	if t.Kind == TypeInterface {
		if t.Name == "Iterable" && len(t.Args) == 1 {
			return t.Args[0]
		}
		if info, ok := c.interfaces[t.Name]; ok {
			subst := c.substForDecl(info.decl.TypeParameters, t.Args)
			if elem := c.iterableTypeFromRefs(info.decl.Extends, subst); !isUnknown(elem) {
				return elem
			}
		}
	}
	if t.Kind == TypeClass {
		if info, ok := c.classes[t.Name]; ok {
			subst := c.substForDecl(info.decl.TypeParameters, t.Args)
			if elem := c.iterableTypeFromRefs(info.decl.Implements, subst); !isUnknown(elem) {
				return elem
			}
		}
	}
	return unknownType
}

func (c *Checker) optionElementType(t *Type) *Type {
	if isUnknown(t) {
		return unknownType
	}
	if t.Name == "Option" && len(t.Args) == 1 {
		return t.Args[0]
	}
	return unknownType
}

func (c *Checker) unwrappableSuccessType(t *Type) (*Type, bool) {
	if isUnknown(t) {
		return unknownType, true
	}
	if args, ok := c.interfaceArgsForType(t, "Unwrappable"); ok && len(args) == 1 {
		return args[0], true
	}
	switch t.Name {
	case "Option":
		if len(t.Args) == 1 {
			return t.Args[0], true
		}
	case "Result":
		if len(t.Args) == 2 {
			return t.Args[0], true
		}
	case "Either":
		if len(t.Args) == 2 {
			return t.Args[1], true
		}
	}
	return unknownType, false
}

func (c *Checker) shortCircuitCompatible(source, target *Type) bool {
	if isUnknown(source) || isUnknown(target) {
		return true
	}
	if source.Name != target.Name {
		return false
	}
	switch source.Name {
	case "Option":
		return len(source.Args) == 1 && len(target.Args) == 1
	case "Result":
		return len(source.Args) == 2 && len(target.Args) == 2 && sameType(source.Args[1], target.Args[1])
	case "Either":
		return len(source.Args) == 2 && len(target.Args) == 2 && sameType(source.Args[0], target.Args[0])
	default:
		return false
	}
}

func (c *Checker) interfaceArgsForType(t *Type, target string) ([]*Type, bool) {
	switch t.Kind {
	case TypeClass:
		info, ok := c.classes[t.Name]
		if !ok {
			return nil, false
		}
		subst := c.substForDecl(info.decl.TypeParameters, t.Args)
		return c.interfaceArgsFromRefs(info.decl.Implements, subst, target)
	case TypeInterface:
		if t.Name == target {
			return t.Args, true
		}
		info, ok := c.interfaces[t.Name]
		if !ok {
			return nil, false
		}
		subst := c.substForDecl(info.decl.TypeParameters, t.Args)
		return c.interfaceArgsFromRefs(info.decl.Extends, subst, target)
	default:
		return nil, false
	}
}

func (c *Checker) interfaceArgsFromRefs(refs []*parser.TypeRef, subst map[string]*Type, target string) ([]*Type, bool) {
	for _, ref := range refs {
		inst := c.instantiateTypeRef(ref, subst)
		if inst.Name == target {
			return inst.Args, true
		}
		if inst.Kind == TypeInterface {
			if args, ok := c.interfaceArgsForType(inst, target); ok {
				return args, true
			}
		}
	}
	return nil, false
}

func (c *Checker) iterableTypeFromRefs(refs []*parser.TypeRef, subst map[string]*Type) *Type {
	for _, ref := range refs {
		inst := c.instantiateTypeRef(ref, subst)
		if inst.Name == "Iterable" && len(inst.Args) == 1 {
			return inst.Args[0]
		}
		if inst.Kind == TypeInterface {
			if info, ok := c.interfaces[inst.Name]; ok {
				nextSubst := c.substForDecl(info.decl.TypeParameters, inst.Args)
				if elem := c.iterableTypeFromRefs(info.decl.Extends, nextSubst); !isUnknown(elem) {
					return elem
				}
			}
		}
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
		case "Int", "Int64", "Bool", "Str", "Rune", "Float", "Float64":
			return true
		default:
			return false
		}
	case TypeClass:
		if info, ok := c.classes[t.Name]; ok && info.decl.Enum {
			return true
		}
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
	if !c.isAssignable(actual, expected) {
		c.addDiagnostic(code, message, span)
	}
}

func (c *Checker) isAssignable(actual, expected *Type) bool {
	if sameType(actual, expected) {
		return true
	}
	if expected.Kind != TypeInterface {
		return false
	}
	switch actual.Kind {
	case TypeClass:
		info, ok := c.classes[actual.Name]
		if !ok {
			return false
		}
		subst := c.substForDecl(info.decl.TypeParameters, actual.Args)
		for _, impl := range info.decl.Implements {
			inst := c.instantiateTypeRef(impl, subst)
			if sameType(inst, expected) || c.interfaceAssignable(inst, expected, map[string]bool{}) {
				return true
			}
		}
	case TypeInterface:
		return c.interfaceAssignable(actual, expected, map[string]bool{})
	}
	return false
}

func (c *Checker) interfaceAssignable(actual, expected *Type, seen map[string]bool) bool {
	if sameType(actual, expected) {
		return true
	}
	if actual == nil || actual.Kind != TypeInterface {
		return false
	}
	key := actual.Name + actual.String()
	if seen[key] {
		return false
	}
	seen[key] = true
	info, ok := c.interfaces[actual.Name]
	if !ok {
		return false
	}
	subst := c.substForDecl(info.decl.TypeParameters, actual.Args)
	for _, parent := range info.decl.Extends {
		inst := c.instantiateTypeRef(parent, subst)
		if sameType(inst, expected) || c.interfaceAssignable(inst, expected, seen) {
			return true
		}
	}
	return false
}

func (c *Checker) typeParametersForName(name string) ([]parser.TypeParameter, bool) {
	if info, ok := c.classes[name]; ok {
		return info.decl.TypeParameters, true
	}
	if info, ok := c.importedClasses[name]; ok {
		return info.decl.TypeParameters, true
	}
	if info, ok := c.interfaces[name]; ok {
		return info.decl.TypeParameters, true
	}
	if info, ok := c.importedInterfaces[name]; ok {
		return info.decl.TypeParameters, true
	}
	return nil, false
}

func (c *Checker) typeArgsInOrder(params []parser.TypeParameter, subst map[string]*Type) []*Type {
	args := make([]*Type, len(params))
	for i, param := range params {
		if subst != nil {
			args[i] = subst[param.Name]
		}
		if args[i] == nil {
			args[i] = unknownType
		}
	}
	return args
}

func (c *Checker) checkTypeArgBounds(args []*Type, params []parser.TypeParameter, span parser.Span) bool {
	if len(args) != len(params) {
		return false
	}
	subst := c.substForDecl(params, args)
	ok := true
	for i, param := range params {
		for _, bound := range param.Bounds {
			expected := c.instantiateTypeRef(bound, subst)
			if !c.typeSatisfiesBound(args[i], expected) {
				c.addDiagnostic("type_argument_bound", "type argument "+args[i].String()+" does not satisfy bound "+expected.String()+" for '"+param.Name+"'", span)
				ok = false
			}
		}
	}
	return ok
}

func (c *Checker) typeSatisfiesBound(actual, bound *Type) bool {
	if isUnknown(actual) || isUnknown(bound) {
		return true
	}
	if c.isAssignable(actual, bound) {
		return true
	}
	if bound.Kind == TypeInterface && c.hasBoundWitness(bound) {
		return true
	}
	return false
}

func (c *Checker) hasBoundWitness(expected *Type) bool {
	for _, info := range c.classes {
		if !info.decl.Object {
			continue
		}
		if c.isAssignable(&Type{Kind: TypeClass, Name: info.name, Args: nil}, expected) {
			return true
		}
	}
	return false
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
	if _, ok := c.classes[name]; ok {
		return TypeClass
	}
	if _, ok := c.importedClasses[name]; ok {
		return TypeClass
	}
	if _, ok := c.interfaces[name]; ok {
		return TypeInterface
	}
	if _, ok := c.importedInterfaces[name]; ok {
		return TypeInterface
	}
	if isBuiltinInterfaceType(name) {
		return TypeInterface
	}
	if isBuiltinType(name) {
		return TypeBuiltin
	}
	return ""
}

func functionType(name string, sig Signature) *Type {
	return &Type{Kind: TypeFunction, Name: name, Signature: &sig}
}

func builtin(name string) *Type { return &Type{Kind: TypeBuiltin, Name: name} }

func isUnitType(t *Type) bool {
	return t != nil && t.Kind == TypeBuiltin && t.Name == "Unit"
}

func isBuiltinType(name string) bool {
	switch name {
	case "Int", "Int64", "Bool", "Str", "Rune", "Float", "Float64", "Array", "Unit":
		return true
	default:
		return false
	}
}

func isBuiltinInterfaceType(name string) bool {
	switch name {
	case "Eq", "Ordering", "List", "Set", "Map", "Term", "Option", "Result", "Either", "Unwrappable":
		return true
	default:
		return false
	}
}

func isBuiltinValue(name string) bool {
	switch name {
	case "List", "Map", "Set", "Array", "Some", "None", "Ok", "Err", "Left", "Right":
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
	case "Int", "Int64", "Float", "Float64", "Str", "Rune":
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
	case *parser.UnitLiteral:
		return e.Span
	case *parser.ListLiteral:
		return e.Span
	case *parser.CallExpr:
		return e.Span
	case *parser.MemberExpr:
		return e.Span
	case *parser.IndexExpr:
		return e.Span
	case *parser.RecordUpdateExpr:
		return e.Span
	case *parser.IfExpr:
		return e.Span
	case *parser.ForYieldExpr:
		return e.Span
	case *parser.LambdaExpr:
		return e.Span
	case *parser.BinaryExpr:
		return e.Span
	case *parser.IsExpr:
		return e.Span
	case *parser.UnaryExpr:
		return e.Span
	case *parser.GroupExpr:
		return e.Span
	case *parser.BlockExpr:
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
	case *parser.LoopStmt:
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
