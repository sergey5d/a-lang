package semantic

import "a-lang/parser"

type Resolver struct {
	diagnostics []Diagnostic
	scopes      []scope
	typeScopes  []typeScope
	globals     map[string]symbol
	functions   map[string]parser.Span
	classes     map[string]parser.Span
	interfaces  map[string]parser.Span
	classTypes  map[string]typeDecl
	ifaceTypes  map[string]typeDecl
	loopDepth   int
}

type symbol struct {
	span    parser.Span
	mutable bool
}

type scope map[string]symbol
type typeScope map[string]parser.Span

type typeDecl struct {
	span  parser.Span
	arity int
}

func Analyze(program *parser.Program) []Diagnostic {
	resolver := &Resolver{
		globals:    map[string]symbol{},
		functions:  map[string]parser.Span{},
		classes:    map[string]parser.Span{},
		interfaces: map[string]parser.Span{},
		classTypes: map[string]typeDecl{},
		ifaceTypes: map[string]typeDecl{},
	}
	resolver.resolveProgram(program)
	return resolver.diagnostics
}

func (r *Resolver) resolveProgram(program *parser.Program) {
	for _, fn := range program.Functions {
		if previous, exists := r.functions[fn.Name]; exists {
			r.addDiagnostic("duplicate_function", "duplicate function '"+fn.Name+"'", fn.Span)
			r.addDiagnostic("duplicate_function", "previous declaration of function '"+fn.Name+"'", previous)
			continue
		}
		r.functions[fn.Name] = fn.Span
	}
	for _, decl := range program.Interfaces {
		if previous, exists := r.interfaces[decl.Name]; exists {
			r.addDiagnostic("duplicate_interface", "duplicate interface '"+decl.Name+"'", decl.Span)
			r.addDiagnostic("duplicate_interface", "previous declaration of interface '"+decl.Name+"'", previous)
			continue
		}
		r.interfaces[decl.Name] = decl.Span
		r.ifaceTypes[decl.Name] = typeDecl{span: decl.Span, arity: len(decl.TypeParameters)}
	}
	for _, decl := range program.Classes {
		if previous, exists := r.classes[decl.Name]; exists {
			r.addDiagnostic("duplicate_class", "duplicate class '"+decl.Name+"'", decl.Span)
			r.addDiagnostic("duplicate_class", "previous declaration of class '"+decl.Name+"'", previous)
			continue
		}
		r.classes[decl.Name] = decl.Span
		r.classTypes[decl.Name] = typeDecl{span: decl.Span, arity: len(decl.TypeParameters)}
	}
	r.resolveGlobals(program.Statements)

	for _, fn := range program.Functions {
		r.resolveFunction(fn)
	}
	for _, decl := range program.Interfaces {
		r.resolveInterface(decl)
	}
	for _, decl := range program.Classes {
		r.resolveClass(decl)
	}
	r.pushScope()
	for _, stmt := range program.Statements {
		if _, ok := stmt.(*parser.ValStmt); ok {
			continue
		}
		r.resolveStatement(stmt)
	}
	r.popScope()
}

func (r *Resolver) resolveGlobals(statements []parser.Statement) {
	r.pushScope()
	defer r.popScope()
	for _, stmt := range statements {
		valStmt, ok := stmt.(*parser.ValStmt)
		if !ok {
			continue
		}
		for _, value := range valStmt.Values {
			if value != nil {
				r.resolveExpr(value)
			}
		}
		for _, binding := range valStmt.Bindings {
			r.resolveTypeRef(binding.Type)
			if previous, exists := r.globals[binding.Name]; exists {
				r.addDiagnostic("duplicate_binding", "duplicate binding '"+binding.Name+"'", binding.Span)
				r.addDiagnostic("duplicate_binding", "previous declaration of '"+binding.Name+"'", previous.span)
				continue
			}
			r.globals[binding.Name] = symbol{span: binding.Span, mutable: binding.Mutable}
			r.defineMutable(binding.Name, binding.Span, binding.Mutable, "duplicate_binding", "duplicate binding '"+binding.Name+"'")
		}
	}
}

func (r *Resolver) resolveFunction(fn *parser.FunctionDecl) {
	r.resolveTypeRef(fn.ReturnType)
	r.pushScope()
	defer r.popScope()

	for _, param := range fn.Parameters {
		r.resolveTypeRef(param.Type)
		r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
	}
	r.resolveBlock(fn.Body)
}

func (r *Resolver) resolveInterface(decl *parser.InterfaceDecl) {
	r.pushTypeScope()
	defer r.popTypeScope()
	for _, param := range decl.TypeParameters {
		r.defineType(param.Name, param.Span, "duplicate_type_parameter", "duplicate type parameter '"+param.Name+"'")
	}
	for _, method := range decl.Methods {
		r.resolveTypeRef(method.ReturnType)
		for _, param := range method.Parameters {
			r.resolveTypeRef(param.Type)
		}
	}
}

func (r *Resolver) resolveClass(decl *parser.ClassDecl) {
	r.pushTypeScope()
	defer r.popTypeScope()
	for _, param := range decl.TypeParameters {
		r.defineType(param.Name, param.Span, "duplicate_type_parameter", "duplicate type parameter '"+param.Name+"'")
	}
	for _, target := range decl.Implements {
		r.resolveTypeRef(target)
	}
	for _, field := range decl.Fields {
		r.resolveTypeRef(field.Type)
		if field.Initializer != nil {
			r.resolveExpr(field.Initializer)
		}
	}

	r.pushScope()
	defer r.popScope()
	r.defineMutable("this", decl.Span, false, "duplicate_binding", "duplicate binding 'this'")
	for _, field := range decl.Fields {
		r.defineMutable(field.Name, field.Span, field.Mutable, "duplicate_binding", "duplicate binding '"+field.Name+"'")
	}
	for _, method := range decl.Methods {
		r.resolveMethod(method)
	}
}

func (r *Resolver) resolveMethod(method *parser.MethodDecl) {
	r.resolveTypeRef(method.ReturnType)
	r.pushScope()
	defer r.popScope()
	r.defineMutable("this", method.Span, false, "duplicate_binding", "duplicate binding 'this'")
	for _, param := range method.Parameters {
		r.resolveTypeRef(param.Type)
		r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
	}
	r.resolveBlockStatements(method.Body.Statements)
}

func (r *Resolver) resolveBlock(block *parser.BlockStmt) {
	r.pushScope()
	defer r.popScope()

	for _, stmt := range block.Statements {
		r.resolveStatement(stmt)
	}
}

func (r *Resolver) resolveStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		for _, value := range s.Values {
			if value != nil {
				r.resolveExpr(value)
			}
		}
		for _, binding := range s.Bindings {
			r.resolveTypeRef(binding.Type)
			r.defineMutable(binding.Name, binding.Span, binding.Mutable, "duplicate_binding", "duplicate binding '"+binding.Name+"'")
		}
	case *parser.LocalFunctionStmt:
		r.defineMutable(s.Function.Name, s.Span, false, "duplicate_binding", "duplicate binding '"+s.Function.Name+"'")
		r.resolveTypeRef(s.Function.ReturnType)
		r.pushScope()
		for _, param := range s.Function.Parameters {
			r.resolveTypeRef(param.Type)
			r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
		}
		r.resolveBlockStatements(s.Function.Body.Statements)
		r.popScope()
	case *parser.AssignmentStmt:
		r.resolveAssignment(s)
	case *parser.IfStmt:
		r.resolveExpr(s.Condition)
		r.resolveBlock(s.Then)
		if s.ElseIf != nil {
			r.resolveStatement(s.ElseIf)
		}
		if s.Else != nil {
			r.resolveBlock(s.Else)
		}
	case *parser.ForStmt:
		r.pushScope()
		for _, binding := range s.Bindings {
			r.resolveExpr(binding.Iterable)
			r.defineMutable(binding.Name, binding.Span, false, "duplicate_binding", "duplicate binding '"+binding.Name+"'")
		}
		if s.Body != nil {
			r.loopDepth++
			r.resolveBlockStatements(s.Body.Statements)
			r.loopDepth--
		}
		if s.YieldBody != nil {
			r.resolveBlockStatements(s.YieldBody.Statements)
		}
		r.popScope()
	case *parser.ReturnStmt:
		r.resolveExpr(s.Value)
	case *parser.BreakStmt:
		if r.loopDepth == 0 {
			r.addDiagnostic("invalid_break", "break used outside of a loop", s.Span)
		}
	case *parser.ExprStmt:
		r.resolveExpr(s.Expr)
	}
}

func (r *Resolver) resolveBlockStatements(statements []parser.Statement) {
	for _, stmt := range statements {
		r.resolveStatement(stmt)
	}
}

func (r *Resolver) resolveExpr(expr parser.Expr) {
	switch e := expr.(type) {
	case *parser.Identifier:
		if r.isDefined(e.Name) || r.functions[e.Name] != (parser.Span{}) || r.classes[e.Name] != (parser.Span{}) || r.interfaces[e.Name] != (parser.Span{}) || isBuiltin(e.Name) {
			return
		}
		r.addDiagnostic("undefined_name", "undefined name '"+e.Name+"'", e.Span)
	case *parser.FloatLiteral:
	case *parser.RuneLiteral:
	case *parser.BoolLiteral:
	case *parser.ListLiteral:
		for _, item := range e.Elements {
			r.resolveExpr(item)
		}
	case *parser.CallExpr:
		r.resolveExpr(e.Callee)
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}
	case *parser.MemberExpr:
		r.resolveExpr(e.Receiver)
	case *parser.IndexExpr:
		r.resolveExpr(e.Receiver)
		r.resolveExpr(e.Index)
	case *parser.IfExpr:
		r.resolveExpr(e.Condition)
		r.pushScope()
		r.resolveBlockStatements(e.Then.Statements)
		r.popScope()
		r.pushScope()
		r.resolveBlockStatements(e.Else.Statements)
		r.popScope()
	case *parser.ForYieldExpr:
		r.pushScope()
		for _, binding := range e.Bindings {
			r.resolveExpr(binding.Iterable)
			r.defineMutable(binding.Name, binding.Span, false, "duplicate_binding", "duplicate binding '"+binding.Name+"'")
		}
		r.resolveBlockStatements(e.YieldBody.Statements)
		r.popScope()
	case *parser.LambdaExpr:
		r.pushScope()
		for _, param := range e.Parameters {
			r.resolveTypeRef(param.Type)
			r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
		}
		if e.Body != nil {
			r.resolveExpr(e.Body)
		}
		if e.BlockBody != nil {
			r.resolveBlockStatements(e.BlockBody.Statements)
		}
		r.popScope()
	case *parser.BinaryExpr:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)
	case *parser.UnaryExpr:
		r.resolveExpr(e.Right)
	case *parser.GroupExpr:
		r.resolveExpr(e.Inner)
	}
}

func (r *Resolver) resolveAssignment(stmt *parser.AssignmentStmt) {
	switch target := stmt.Target.(type) {
	case *parser.Identifier:
		symbol, ok := r.lookup(target.Name)
		if !ok {
			r.addDiagnostic("undefined_name", "undefined name '"+target.Name+"'", target.Span)
		} else if !symbol.mutable {
			r.addDiagnostic("assign_immutable", "cannot assign to immutable binding '"+target.Name+"'", target.Span)
		} else if stmt.Operator == "=" {
			r.addDiagnostic("invalid_assignment_operator", "use ':=' for mutable reassignment of '"+target.Name+"'", target.Span)
		}
	case *parser.MemberExpr:
		r.resolveExpr(target.Receiver)
	case *parser.IndexExpr:
		r.resolveExpr(target.Receiver)
		r.resolveExpr(target.Index)
	default:
		r.addDiagnostic("invalid_assignment_target", "invalid assignment target", stmt.Span)
	}
	r.resolveExpr(stmt.Value)
}

func (r *Resolver) defineMutable(name string, span parser.Span, mutable bool, code, message string) {
	current := r.currentScope()
	if previous, exists := current[name]; exists {
		r.addDiagnostic(code, message, span)
		r.addDiagnostic(code, "previous declaration of '"+name+"'", previous.span)
		return
	}
	current[name] = symbol{span: span, mutable: mutable}
}

func (r *Resolver) defineType(name string, span parser.Span, code, message string) {
	current := r.currentTypeScope()
	if previous, exists := current[name]; exists {
		r.addDiagnostic(code, message, span)
		r.addDiagnostic(code, "previous declaration of '"+name+"'", previous)
		return
	}
	current[name] = span
}

func (r *Resolver) resolveTypeRef(ref *parser.TypeRef) {
	if ref == nil {
		return
	}
	if ref.ReturnType != nil {
		for _, param := range ref.ParameterTypes {
			r.resolveTypeRef(param)
		}
		r.resolveTypeRef(ref.ReturnType)
		return
	}
	for _, arg := range ref.Arguments {
		r.resolveTypeRef(arg)
	}
	if r.isTypeParameter(ref.Name) {
		if len(ref.Arguments) != 0 {
			r.addDiagnostic("invalid_type_arity", "type parameter '"+ref.Name+"' cannot have type arguments", ref.Span)
		}
		return
	}
	if arity, ok := builtinTypeArity(ref.Name); ok {
		if len(ref.Arguments) != arity {
			r.addDiagnostic("invalid_type_arity", "type '"+ref.Name+"' expects "+arityLabel(arity)+" type arguments", ref.Span)
		}
		return
	}
	if decl, ok := r.classTypes[ref.Name]; ok {
		if len(ref.Arguments) != decl.arity {
			r.addDiagnostic("invalid_type_arity", "type '"+ref.Name+"' expects "+arityLabel(decl.arity)+" type arguments", ref.Span)
		}
		return
	}
	if decl, ok := r.ifaceTypes[ref.Name]; ok {
		if len(ref.Arguments) != decl.arity {
			r.addDiagnostic("invalid_type_arity", "type '"+ref.Name+"' expects "+arityLabel(decl.arity)+" type arguments", ref.Span)
		}
		return
	}
	r.addDiagnostic("undefined_type", "undefined type '"+ref.Name+"'", ref.Span)
}

func (r *Resolver) isTypeParameter(name string) bool {
	for i := len(r.typeScopes) - 1; i >= 0; i-- {
		if _, ok := r.typeScopes[i][name]; ok {
			return true
		}
	}
	return false
}

func builtinTypeArity(name string) (int, bool) {
	switch name {
	case "Int", "Int64", "Bool", "String", "Rune", "Float", "Float64", "Term":
		return 0, true
	case "List", "Set", "Array", "Option":
		return 1, true
	case "Map":
		return 2, true
	case "Eq":
		return 1, true
	default:
		return 0, false
	}
}

func arityLabel(arity int) string {
	switch arity {
	case 0:
		return "0"
	case 1:
		return "1"
	case 2:
		return "2"
	default:
		return "multiple"
	}
}

func (r *Resolver) isDefined(name string) bool {
	_, ok := r.lookup(name)
	return ok
}

func (r *Resolver) lookup(name string) (symbol, bool) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if sym, ok := r.scopes[i][name]; ok {
			return sym, true
		}
	}
	if sym, ok := r.globals[name]; ok {
		return sym, true
	}
	return symbol{}, false
}

func (r *Resolver) addDiagnostic(code, message string, span parser.Span) {
	r.diagnostics = append(r.diagnostics, Diagnostic{
		Code:    code,
		Message: message,
		Span:    span,
	})
}

func (r *Resolver) pushScope() {
	r.scopes = append(r.scopes, scope{})
}

func (r *Resolver) popScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

func (r *Resolver) currentScope() scope {
	if len(r.scopes) == 0 {
		r.pushScope()
	}
	return r.scopes[len(r.scopes)-1]
}

func (r *Resolver) pushTypeScope() {
	r.typeScopes = append(r.typeScopes, typeScope{})
}

func (r *Resolver) popTypeScope() {
	r.typeScopes = r.typeScopes[:len(r.typeScopes)-1]
}

func (r *Resolver) currentTypeScope() typeScope {
	if len(r.typeScopes) == 0 {
		r.pushTypeScope()
	}
	return r.typeScopes[len(r.typeScopes)-1]
}

func isBuiltin(name string) bool {
	switch name {
	case "List", "Map", "Set", "Array", "Some", "None", "Term":
		return true
	default:
		return false
	}
}
