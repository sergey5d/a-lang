package semantic

import "a-lang/parser"

type Resolver struct {
	diagnostics []Diagnostic
	scopes      []scope
	functions   map[string]parser.Span
	loopDepth   int
}

type symbol struct {
	span    parser.Span
	mutable bool
}

type scope map[string]symbol

func Analyze(program *parser.Program) []Diagnostic {
	resolver := &Resolver{
		functions: map[string]parser.Span{},
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

	for _, fn := range program.Functions {
		r.resolveFunction(fn)
	}
}

func (r *Resolver) resolveFunction(fn *parser.FunctionDecl) {
	r.pushScope()
	defer r.popScope()

	for _, param := range fn.Parameters {
		r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
	}
	r.resolveBlock(fn.Body)
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
			r.resolveExpr(value)
		}
		for _, binding := range s.Bindings {
			r.defineMutable(binding.Name, binding.Span, binding.Mutable, "duplicate_binding", "duplicate binding '"+binding.Name+"'")
		}
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
	case *parser.MatchStmt:
		r.resolveExpr(s.Target)
		for _, arm := range s.Arms {
			r.resolveExpr(arm.Pattern)
			r.resolveExpr(arm.Result)
		}
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
		if r.isDefined(e.Name) || r.functions[e.Name] != (parser.Span{}) || isBuiltin(e.Name) {
			return
		}
		r.addDiagnostic("undefined_name", "undefined name '"+e.Name+"'", e.Span)
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
	case *parser.LambdaExpr:
		r.pushScope()
		for _, param := range e.Parameters {
			r.defineMutable(param.Name, param.Span, false, "duplicate_parameter", "duplicate parameter '"+param.Name+"'")
		}
		r.resolveExpr(e.Body)
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
		}
	case *parser.MemberExpr:
		r.resolveExpr(target.Receiver)
	default:
		r.addDiagnostic("invalid_assignment_target", "invalid assignment target", stmt.Span)
	}
	r.resolveExpr(stmt.Value)
}

func (r *Resolver) define(name string, span parser.Span, code, message string) {
	current := r.currentScope()
	if previous, exists := current[name]; exists {
		r.addDiagnostic(code, message, span)
		r.addDiagnostic(code, "previous declaration of '"+name+"'", previous.span)
		return
	}
	current[name] = symbol{span: span, mutable: true}
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
	return r.scopes[len(r.scopes)-1]
}

func isBuiltin(name string) bool {
	switch name {
	case "Map", "Set", "Array", "range":
		return true
	default:
		return false
	}
}
