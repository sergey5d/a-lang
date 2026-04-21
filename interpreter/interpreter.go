package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"a-lang/module"
	"a-lang/parser"
)

type Value any

type deferredValue struct{}

type RuntimeError struct {
	Message string
	Span    parser.Span
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%s at %d:%d", e.Message, e.Span.Start.Line, e.Span.Start.Column)
}

type slot struct {
	value   Value
	mutable bool
}

type env struct {
	parent *env
	values map[string]slot
}

type Interpreter struct {
	program   *parser.Program
	functions map[string]*parser.FunctionDecl
	classes   map[string]*parser.ClassDecl
	interfaces map[string]*parser.InterfaceDecl
	imports   map[string]*Interpreter
	globals   *env
	ready     bool
}

type instance struct {
	class  *parser.ClassDecl
	caseName string
	fields map[string]Value
}

type closure struct {
	params     []string
	variadic   bool
	body       parser.Expr
	blockBody  *parser.BlockStmt
	returnType *parser.TypeRef
	env        *env
}

type builtinRef struct{ name string }

type nativeList struct{ items []Value }
type nativeListIterator struct {
	items []Value
	index int
}
type nativeArray struct{ items []Value }
type nativeTuple struct {
	items []Value
	names []string
}
type nativeOption struct {
	value Value
	set   bool
}
type nativeSet struct {
	keys  map[string]Value
	order []string
}
type nativeMap struct {
	items map[string]Value
	keys  map[string]Value
	order []string
}
type nativeTerm struct{}

type returnSignal struct {
	value Value
}

type breakSignal struct{}

func (t *nativeTuple) String() string {
	parts := make([]string, len(t.items))
	for i, item := range t.items {
		if i < len(t.names) && t.names[i] != "" {
			parts[i] = t.names[i] + "=" + fmt.Sprint(item)
		} else {
			parts[i] = fmt.Sprint(item)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func New(program *parser.Program) *Interpreter {
	in := &Interpreter{
		program:    program,
		functions:  map[string]*parser.FunctionDecl{},
		classes:    map[string]*parser.ClassDecl{},
		interfaces: map[string]*parser.InterfaceDecl{},
		imports:    map[string]*Interpreter{},
	}
	for _, fn := range program.Functions {
		in.functions[fn.Name] = fn
	}
	for _, iface := range program.Interfaces {
		in.interfaces[iface.Name] = iface
	}
	for _, class := range program.Classes {
		in.classes[class.Name] = class
	}
	return in
}

func NewModule(mod *module.LoadedModule) *Interpreter {
	in := New(mod.Program)
	for alias, imported := range mod.Imports {
		in.imports[alias] = NewModule(imported)
	}
	return in
}

func (in *Interpreter) Call(function string, args ...Value) (Value, error) {
	global, err := in.ensureGlobals()
	if err != nil {
		return nil, err
	}
	return in.callFunctionByName(function, args, global)
}

func (in *Interpreter) ensureGlobals() (*env, error) {
	if in.ready {
		return in.globals, nil
	}
	global := newEnv(nil)
	if err := in.execTopLevel(global); err != nil {
		return nil, err
	}
	in.globals = global
	in.ready = true
	return global, nil
}

func (in *Interpreter) execTopLevel(global *env) error {
	for alias, imported := range in.imports {
		global.define(alias, moduleRef{name: alias}, false)
		if _, err := imported.ensureGlobals(); err != nil {
			return err
		}
	}
	for _, stmt := range in.program.Statements {
		if _, signal, err := in.execStmt(stmt, global, nil); err != nil {
			return err
		} else if signal != nil {
			return RuntimeError{Message: "unexpected control flow at top level", Span: stmtSpan(stmt)}
		}
	}
	return nil
}

func (in *Interpreter) callFunctionByName(name string, args []Value, parent *env) (Value, error) {
	fn, ok := in.functions[name]
	if !ok {
		return nil, RuntimeError{Message: "undefined function '" + name + "'", Span: parser.Span{}}
	}
	return in.callFunction(fn, args, parent)
}

func (in *Interpreter) callFunction(fn *parser.FunctionDecl, args []Value, parent *env) (Value, error) {
	if !acceptsArgCount(fn.Parameters, len(args)) {
		return nil, RuntimeError{Message: fmt.Sprintf("function '%s' expects %s args, got %d", fn.Name, expectedCallableArgs(fn.Parameters), len(args)), Span: fn.Span}
	}
	local := newEnv(parent)
	for i, param := range fn.Parameters {
		if param.Variadic {
			local.define(param.Name, &nativeList{items: append([]Value{}, args[i:]...)}, false)
			break
		}
		local.define(param.Name, args[i], false)
	}
	value, signal, err := in.execBlock(fn.Body, local, nil)
	if err != nil {
		return nil, err
	}
	if ret, ok := signal.(returnSignal); ok {
		return in.coerceValueForTypeRef(fn.ReturnType, ret.value), nil
	}
	if fn.ReturnType == nil || isUnitTypeRef(fn.ReturnType) {
		return nil, nil
	}
	return in.coerceValueForTypeRef(fn.ReturnType, value), nil
}

func (in *Interpreter) callMethod(receiver *instance, method *parser.MethodDecl, args []Value, parent *env) (Value, error) {
	if !acceptsArgCount(method.Parameters, len(args)) {
		return nil, RuntimeError{Message: fmt.Sprintf("method '%s' expects %s args, got %d", method.Name, expectedCallableArgs(method.Parameters), len(args)), Span: method.Span}
	}
	local := newEnv(parent)
	local.define("this", receiver, false)
	if receiver.class.Enum {
		for _, enumCase := range receiver.class.Cases {
			if len(enumCase.Fields) == 0 {
				value, err := in.constructEnumCase(receiver.class, enumCase, nil, parent, enumCase.Span)
				if err != nil {
					return nil, err
				}
				local.define(enumCase.Name, value, false)
			}
		}
	}
	if method.Constructor {
		local.define("__constructor__", true, false)
	}
	for i, param := range method.Parameters {
		if param.Variadic {
			local.define(param.Name, &nativeList{items: append([]Value{}, args[i:]...)}, false)
			break
		}
		local.define(param.Name, args[i], false)
	}
	value, signal, err := in.execBlock(method.Body, local, receiver)
	if err != nil {
		return nil, err
	}
	if ret, ok := signal.(returnSignal); ok {
		return in.coerceValueForTypeRef(method.ReturnType, ret.value), nil
	}
	if method.ReturnType == nil || isUnitTypeRef(method.ReturnType) {
		return nil, nil
	}
	return in.coerceValueForTypeRef(method.ReturnType, value), nil
}

func implicitConstructorFields(class *parser.ClassDecl) []parser.FieldDecl {
	fields := make([]parser.FieldDecl, 0, len(class.Fields))
	for _, field := range class.Fields {
		if !field.Private && field.Initializer == nil {
			fields = append(fields, field)
		}
	}
	return fields
}

func (in *Interpreter) applyImplicitConstructor(receiver *instance, args []Value, span parser.Span) error {
	fields := implicitConstructorFields(receiver.class)
	if len(args) != len(fields) {
		return RuntimeError{Message: fmt.Sprintf("constructor '%s' expects %d args, got %d", receiver.class.Name, len(fields), len(args)), Span: span}
	}
	for i, field := range fields {
		receiver.fields[field.Name] = args[i]
	}
	return nil
}

func constructorParamOptions(class *parser.ClassDecl) [][]parser.Parameter {
	var options [][]parser.Parameter
	for _, method := range class.Methods {
		if method.Constructor {
			options = append(options, method.Parameters)
		}
	}
	fields := implicitConstructorFields(class)
	params := make([]parser.Parameter, len(fields))
	for i, field := range fields {
		params[i] = parser.Parameter{Name: field.Name, Type: field.Type, Span: field.Span}
	}
	options = append(options, params)
	return options
}

func reorderConstructorValueArgs(class *parser.ClassDecl, args []namedValueArg, span parser.Span) ([]Value, error) {
	var firstErr error
	for _, params := range constructorParamOptions(class) {
		reordered, err := reorderNamedValueArgs(params, args, span, "constructor '"+class.Name+"'")
		if err == nil {
			return reordered, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return namedArgValues(args), nil
}

func (in *Interpreter) constructInto(receiver *instance, class *parser.ClassDecl, args []Value, parent *env, span parser.Span) error {
	for _, method := range class.Methods {
		if method.Constructor && in.runtimeMethodMatches(method, args) {
			_, err := in.callMethod(receiver, method, args, parent)
			return err
		}
	}
	if len(implicitConstructorFields(class)) > 0 || len(args) == 0 {
		return in.applyImplicitConstructor(receiver, args, span)
	}
	return RuntimeError{Message: fmt.Sprintf("constructor '%s' expects 0 args, got %d", class.Name, len(args)), Span: span}
}

func (in *Interpreter) construct(class *parser.ClassDecl, args []Value, parent *env) (Value, error) {
	obj := &instance{class: class, fields: map[string]Value{}}
	fieldEnv := newEnv(parent)
	fieldEnv.define("this", obj, false)
	for _, field := range class.Fields {
		switch {
		case field.Initializer != nil:
			value, err := in.evalExpr(field.Initializer, fieldEnv)
			if err != nil {
				return nil, err
			}
			obj.fields[field.Name] = value
		case field.Deferred:
			obj.fields[field.Name] = deferredValue{}
		default:
			obj.fields[field.Name] = zeroValue(field.Type)
		}
	}
	if err := in.constructInto(obj, class, args, parent, class.Span); err != nil {
		return nil, err
	}
	return obj, nil
}

func (in *Interpreter) constructEnumCase(class *parser.ClassDecl, enumCase parser.EnumCaseDecl, args []Value, parent *env, span parser.Span) (Value, error) {
	obj := &instance{class: class, caseName: enumCase.Name, fields: map[string]Value{}}
	fieldEnv := newEnv(parent)
	fieldEnv.define("this", obj, false)
	for _, field := range class.Fields {
		switch {
		case field.Initializer != nil:
			value, err := in.evalExpr(field.Initializer, fieldEnv)
			if err != nil {
				return nil, err
			}
			obj.fields[field.Name] = value
		case field.Deferred:
			obj.fields[field.Name] = deferredValue{}
		default:
			obj.fields[field.Name] = zeroValue(field.Type)
		}
	}
	for _, assignment := range enumCase.Assignments {
		value, err := in.evalExpr(assignment.Value, fieldEnv)
		if err != nil {
			return nil, err
		}
		obj.fields[assignment.Name] = value
	}
	if len(args) != len(enumCase.Fields) {
		return nil, RuntimeError{Message: fmt.Sprintf("enum case '%s' expects %d args, got %d", enumCase.Name, len(enumCase.Fields), len(args)), Span: span}
	}
	for i, field := range enumCase.Fields {
		obj.fields[field.Name] = args[i]
	}
	return obj, nil
}

func (in *Interpreter) execBlock(block *parser.BlockStmt, parent *env, self *instance) (Value, any, error) {
	local := newEnv(parent)
	if self != nil {
		local.define("this", self, false)
	}
	var lastValue Value
	for _, stmt := range block.Statements {
		value, signal, err := in.execStmt(stmt, local, self)
		if err != nil || signal != nil {
			return value, signal, err
		}
		lastValue = value
	}
	return lastValue, nil, nil
}

func (in *Interpreter) execStmt(stmt parser.Statement, local *env, self *instance) (Value, any, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		values, err := in.bindingValues(s.Bindings, s.Values, local, s.Span)
		if err != nil {
			return nil, nil, err
		}
		for i, binding := range s.Bindings {
			var value Value
			if i < len(values) {
				value = values[i]
			}
			local.define(binding.Name, value, binding.Mutable)
		}
		return nil, nil, nil
	case *parser.LocalFunctionStmt:
		params := make([]string, len(s.Function.Parameters))
		for i, param := range s.Function.Parameters {
			params[i] = param.Name
		}
		local.define(s.Function.Name, &closure{params: params, variadic: len(s.Function.Parameters) > 0 && s.Function.Parameters[len(s.Function.Parameters)-1].Variadic, blockBody: s.Function.Body, returnType: s.Function.ReturnType, env: local}, false)
		return nil, nil, nil
	case *parser.AssignmentStmt:
		value, err := in.evalExpr(s.Value, local)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, in.assign(s.Target, s.Operator, value, local)
	case *parser.MultiAssignmentStmt:
		values, err := in.assignmentValues(len(s.Targets), s.Values, local, s.Span)
		if err != nil {
			return nil, nil, err
		}
		for i, target := range s.Targets {
			if err := in.assign(target, s.Operator, values[i], local); err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, nil
	case *parser.IfStmt:
		if s.BindingValue != nil {
			optionValue, err := in.evalExpr(s.BindingValue, local)
			if err != nil {
				return nil, nil, err
			}
			set, value, err := in.optionBindingValue(optionValue, local, exprSpan(s.BindingValue))
			if err != nil {
				return nil, nil, err
			}
			if set {
				thenEnv := newEnv(local)
				thenEnv.define(s.BindingName, value, false)
				return in.execBlock(s.Then, thenEnv, self)
			}
		} else {
			cond, err := in.evalExpr(s.Condition, local)
			if err != nil {
				return nil, nil, err
			}
			truthy, err := asBool(cond, exprSpan(s.Condition))
			if err != nil {
				return nil, nil, err
			}
			if truthy {
				return in.execBlock(s.Then, local, self)
			}
		}
		if s.ElseIf != nil {
			return in.execStmt(s.ElseIf, local, self)
		}
		if s.Else != nil {
			return in.execBlock(s.Else, local, self)
		}
		return nil, nil, nil
	case *parser.LoopStmt:
		return in.execLoop(s, local, self)
	case *parser.ForStmt:
		return in.execFor(s, local, self)
	case *parser.ReturnStmt:
		value, err := in.evalExpr(s.Value, local)
		if err != nil {
			return nil, nil, err
		}
		return nil, returnSignal{value: value}, nil
	case *parser.BreakStmt:
		return nil, breakSignal{}, nil
	case *parser.ExprStmt:
		value, err := in.evalExpr(s.Expr, local)
		return value, nil, err
	default:
		return nil, nil, RuntimeError{Message: "unsupported statement", Span: stmtSpan(stmt)}
	}
}

func (in *Interpreter) optionBindingValue(optionValue Value, local *env, span parser.Span) (bool, Value, error) {
	switch value := optionValue.(type) {
	case *nativeOption:
		if !value.set {
			return false, nil, nil
		}
		return true, value.value, nil
	case *instance:
		if value.class.Name != "Option" {
			return false, nil, RuntimeError{Message: "if binding requires Option[T]", Span: span}
		}
		setValue, err := in.invokeMethod(value, "isSet", nil, local, span)
		if err != nil {
			return false, nil, err
		}
		set, ok := setValue.(bool)
		if !ok || !set {
			return false, nil, nil
		}
		unwrapped, err := in.invokeMethod(value, "get", nil, local, span)
		if err != nil {
			return false, nil, err
		}
		return true, unwrapped, nil
	default:
		return false, nil, RuntimeError{Message: "if binding requires Option[T]", Span: span}
	}
}

func (in *Interpreter) execFor(stmt *parser.ForStmt, local *env, self *instance) (Value, any, error) {
	if stmt.YieldBody != nil {
		var yielded []Value
		signal, err := in.execForBindings(stmt.Bindings, 0, local, self, func(loopEnv *env) (any, error) {
			value, signal, err := in.evalYieldBody(stmt.YieldBody, loopEnv, self)
			if err != nil {
				return nil, err
			}
			if signal == nil {
				yielded = append(yielded, value)
			}
			return signal, nil
		})
		if err != nil {
			return nil, nil, err
		}
		switch signal.(type) {
		case nil:
			return yielded, nil, nil
		case breakSignal:
			return yielded, nil, nil
		default:
			return nil, signal, nil
		}
	}
	signal, err := in.execForBindings(stmt.Bindings, 0, local, self, func(loopEnv *env) (any, error) {
		_, signal, err := in.execBlock(stmt.Body, loopEnv, self)
		return signal, err
	})
	if err != nil {
		return nil, nil, err
	}
	switch signal.(type) {
	case nil, breakSignal:
		return nil, nil, nil
	default:
		return nil, signal, nil
	}
}

func (in *Interpreter) execLoop(stmt *parser.LoopStmt, local *env, self *instance) (Value, any, error) {
	for {
		_, signal, err := in.execBlock(stmt.Body, local, self)
		if err != nil {
			return nil, nil, err
		}
		switch signal.(type) {
		case nil:
		case breakSignal:
			return nil, nil, nil
		default:
			return nil, signal, nil
		}
	}
}

func (in *Interpreter) execForBindings(bindings []parser.ForBinding, index int, local *env, self *instance, body func(*env) (any, error)) (any, error) {
	if index == len(bindings) {
		return body(local)
	}
	iterable, err := in.evalExpr(bindings[index].Iterable, local)
	if err != nil {
		return nil, err
	}
	items, ok := in.iterableToSlice(iterable, local, bindings[index].Span)
	if !ok {
		return nil, RuntimeError{Message: "for loop expects iterable list value", Span: bindings[index].Span}
	}
	for _, item := range items {
		loopEnv := newEnv(local)
		loopEnv.define(bindings[index].Name, item, false)
		signal, err := in.execForBindings(bindings, index+1, loopEnv, self, body)
		if err != nil {
			return nil, err
		}
		switch signal.(type) {
		case nil:
		case breakSignal:
			return breakSignal{}, nil
		default:
			return signal, nil
		}
	}
	return nil, nil
}

func (in *Interpreter) evalYieldBody(block *parser.BlockStmt, local *env, self *instance) (Value, any, error) {
	if block == nil || len(block.Statements) == 0 {
		return nil, nil, RuntimeError{Message: "yield body must end with an expression", Span: parser.Span{}}
	}
	for i := 0; i < len(block.Statements)-1; i++ {
		_, signal, err := in.execStmt(block.Statements[i], local, self)
		if err != nil || signal != nil {
			return nil, signal, err
		}
	}
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*parser.ExprStmt)
	if !ok {
		return nil, nil, RuntimeError{Message: "yield body must end with an expression", Span: stmtSpan(last)}
	}
	value, err := in.evalExpr(exprStmt.Expr, local)
	return value, nil, err
}

func (in *Interpreter) evalBlockValue(block *parser.BlockStmt, local *env, self *instance, message string) (Value, any, error) {
	if block == nil || len(block.Statements) == 0 {
		return nil, nil, RuntimeError{Message: message, Span: parser.Span{}}
	}
	blockEnv := newEnv(local)
	for i := 0; i < len(block.Statements)-1; i++ {
		_, signal, err := in.execStmt(block.Statements[i], blockEnv, self)
		if err != nil || signal != nil {
			return nil, signal, err
		}
	}
	last := block.Statements[len(block.Statements)-1]
	exprStmt, ok := last.(*parser.ExprStmt)
	if !ok {
		return nil, nil, RuntimeError{Message: message, Span: stmtSpan(last)}
	}
	value, err := in.evalExpr(exprStmt.Expr, blockEnv)
	return value, nil, err
}

func (in *Interpreter) assign(target parser.Expr, operator string, value Value, local *env) error {
	switch t := target.(type) {
	case *parser.Identifier:
		current, ok := local.resolve(t.Name)
		if !ok {
			return RuntimeError{Message: "undefined name '" + t.Name + "'", Span: t.Span}
		}
		if !current.mutable() {
			return RuntimeError{Message: "cannot assign to immutable binding '" + t.Name + "'", Span: t.Span}
		}
		if operator == "=" {
			return RuntimeError{Message: "use ':=' for mutable reassignment", Span: t.Span}
		}
		if operator != ":=" {
			updated, err := applyBinary(operator[:len(operator)-1], current.value(), value, t.Span)
			if err != nil {
				return err
			}
			value = updated
		}
		value = preserveTupleNames(current.value(), value)
		return current.set(value)
	case *parser.MemberExpr:
		receiver, err := in.evalExpr(t.Receiver, local)
		if err != nil {
			return err
		}
		obj, ok := receiver.(*instance)
		if !ok {
			return RuntimeError{Message: "member assignment expects class instance", Span: t.Span}
		}
		current := obj.fields[t.Name]
		if operator != "=" && operator != ":=" {
			updated, err := applyBinary(operator[:len(operator)-1], current, value, t.Span)
			if err != nil {
				return err
			}
			value = updated
		}
		obj.fields[t.Name] = value
		return nil
	case *parser.IndexExpr:
		receiver, err := in.evalExpr(t.Receiver, local)
		if err != nil {
			return err
		}
		items, ok := indexedItems(receiver)
		if !ok {
			return RuntimeError{Message: "index assignment requires array-like value", Span: t.Span}
		}
		indexValue, err := in.evalExpr(t.Index, local)
		if err != nil {
			return err
		}
		index, ok := indexValue.(int64)
		if !ok {
			return RuntimeError{Message: "index expression must be Int", Span: exprSpan(t.Index)}
		}
		if index < 0 || index >= int64(len(items)) {
			return RuntimeError{Message: "index out of bounds", Span: t.Span}
		}
		if operator == "=" {
			return RuntimeError{Message: "use ':=' for mutable reassignment", Span: t.Span}
		}
		if operator != "=" && operator != ":=" {
			updated, err := applyBinary(operator[:len(operator)-1], items[index], value, t.Span)
			if err != nil {
				return err
			}
			value = updated
		}
		items[index] = value
		return nil
	default:
		return RuntimeError{Message: "invalid assignment target", Span: exprSpan(target)}
	}
}

func (in *Interpreter) evalExpr(expr parser.Expr, local *env) (Value, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		if binding, ok := local.get(e.Name); ok {
			if _, deferred := binding.value.(deferredValue); deferred {
				return nil, RuntimeError{Message: "binding '" + e.Name + "' is deferred and has not been assigned", Span: e.Span}
			}
			return binding.value, nil
		}
		if thisSlot, ok := local.get("this"); ok {
			if obj, ok := thisSlot.value.(*instance); ok {
				if value, exists := obj.fields[e.Name]; exists {
					if _, deferred := value.(deferredValue); deferred {
						return nil, RuntimeError{Message: "field '" + e.Name + "' is deferred and has not been assigned", Span: e.Span}
					}
					return value, nil
				}
			}
		}
		if _, ok := in.functions[e.Name]; ok {
			return functionRef{module: in, name: e.Name}, nil
		}
		switch e.Name {
		case "List", "Set", "Map", "Array", "Some", "None":
			return builtinRef{name: e.Name}, nil
		case "Term":
			return &nativeTerm{}, nil
		}
		if _, ok := in.imports[e.Name]; ok {
			return moduleRef{name: e.Name}, nil
		}
		if _, ok := in.classes[e.Name]; ok {
			return classRef{module: in, name: e.Name}, nil
		}
		return nil, RuntimeError{Message: "undefined name '" + e.Name + "'", Span: e.Span}
	case *parser.IntegerLiteral:
		n, err := strconv.ParseInt(e.Value, 10, 64)
		if err != nil {
			return nil, RuntimeError{Message: "invalid integer literal", Span: e.Span}
		}
		return n, nil
	case *parser.FloatLiteral:
		n, err := strconv.ParseFloat(e.Value, 64)
		if err != nil {
			return nil, RuntimeError{Message: "invalid float literal", Span: e.Span}
		}
		return n, nil
	case *parser.RuneLiteral:
		return []rune(e.Value)[0], nil
	case *parser.BoolLiteral:
		return e.Value, nil
	case *parser.StringLiteral:
		return e.Value, nil
	case *parser.UnitLiteral:
		return nil, nil
	case *parser.ListLiteral:
		items := make([]Value, len(e.Elements))
		for i, item := range e.Elements {
			value, err := in.evalExpr(item, local)
			if err != nil {
				return nil, err
			}
			items[i] = value
		}
		return &nativeList{items: items}, nil
	case *parser.TupleLiteral:
		items := make([]Value, len(e.Elements))
		for i, item := range e.Elements {
			value, err := in.evalExpr(item, local)
			if err != nil {
				return nil, err
			}
			items[i] = value
		}
		return &nativeTuple{items: items}, nil
	case *parser.IfExpr:
		cond, err := in.evalExpr(e.Condition, local)
		if err != nil {
			return nil, err
		}
		truthy, err := asBool(cond, exprSpan(e.Condition))
		if err != nil {
			return nil, err
		}
		if truthy {
			value, signal, err := in.evalBlockValue(e.Then, local, nil, "if expression branches must end with an expression")
			if err != nil {
				return nil, err
			}
			if signal != nil {
				return nil, RuntimeError{Message: "unexpected control flow in if expression", Span: e.Span}
			}
			return value, nil
		}
		value, signal, err := in.evalBlockValue(e.Else, local, nil, "if expression branches must end with an expression")
		if err != nil {
			return nil, err
		}
		if signal != nil {
			return nil, RuntimeError{Message: "unexpected control flow in if expression", Span: e.Span}
		}
		return value, nil
	case *parser.ForYieldExpr:
		var yielded []Value
		signal, err := in.execForBindings(e.Bindings, 0, local, nil, func(loopEnv *env) (any, error) {
			value, signal, err := in.evalYieldBody(e.YieldBody, loopEnv, nil)
			if err != nil {
				return nil, err
			}
			if signal == nil {
				yielded = append(yielded, value)
			}
			return signal, nil
		})
		if err != nil {
			return nil, err
		}
		switch signal.(type) {
		case nil, breakSignal:
			return &nativeList{items: yielded}, nil
		default:
			return nil, RuntimeError{Message: "unexpected control flow in yield expression", Span: e.Span}
		}
	case *parser.GroupExpr:
		return in.evalExpr(e.Inner, local)
	case *parser.UnaryExpr:
		right, err := in.evalExpr(e.Right, local)
		if err != nil {
			return nil, err
		}
		switch e.Operator {
		case "!":
			b, err := asBool(right, e.Span)
			if err != nil {
				return nil, err
			}
			return !b, nil
		case "-":
			switch n := right.(type) {
			case int64:
				return -n, nil
			case float64:
				return -n, nil
			default:
				return nil, RuntimeError{Message: "operator - requires numeric operand", Span: e.Span}
			}
		default:
			return nil, RuntimeError{Message: "unsupported unary operator", Span: e.Span}
		}
	case *parser.BinaryExpr:
		left, err := in.evalExpr(e.Left, local)
		if err != nil {
			return nil, err
		}
		if e.Operator == "&&" {
			lb, err := asBool(left, e.Span)
			if err != nil {
				return nil, err
			}
			if !lb {
				return false, nil
			}
		}
		if e.Operator == "||" {
			lb, err := asBool(left, e.Span)
			if err != nil {
				return nil, err
			}
			if lb {
				return true, nil
			}
		}
		right, err := in.evalExpr(e.Right, local)
		if err != nil {
			return nil, err
		}
		if e.Operator == "==" || e.Operator == "!=" {
			equal, err := in.valuesEqual(left, right, e.Span, local)
			if err != nil {
				return nil, err
			}
			if e.Operator == "!=" {
				return !equal, nil
			}
			return equal, nil
		}
		return applyBinary(e.Operator, left, right, e.Span)
	case *parser.IsExpr:
		left, err := in.evalExpr(e.Left, local)
		if err != nil {
			return nil, err
		}
		return in.runtimeValueMatchesType(left, e.Target), nil
	case *parser.CallExpr:
		return in.evalCall(e, local)
	case *parser.MemberExpr:
		receiver, err := in.evalExpr(e.Receiver, local)
		if err != nil {
			return nil, err
		}
		return in.evalMember(receiver, e)
	case *parser.IndexExpr:
		receiver, err := in.evalExpr(e.Receiver, local)
		if err != nil {
			return nil, err
		}
		items, ok := indexedItems(receiver)
		if !ok {
			return nil, RuntimeError{Message: "indexing requires array-like value", Span: e.Span}
		}
		indexValue, err := in.evalExpr(e.Index, local)
		if err != nil {
			return nil, err
		}
		index, ok := indexValue.(int64)
		if !ok {
			return nil, RuntimeError{Message: "index expression must be Int", Span: exprSpan(e.Index)}
		}
		if index < 0 || index >= int64(len(items)) {
			return nil, RuntimeError{Message: "index out of bounds", Span: e.Span}
		}
		return items[index], nil
	case *parser.RecordUpdateExpr:
		receiver, err := in.evalExpr(e.Receiver, local)
		if err != nil {
			return nil, err
		}
		record, ok := receiver.(*instance)
		if !ok || !record.class.Record {
			return nil, RuntimeError{Message: "record update requires a record value", Span: e.Span}
		}
		copyFields := make(map[string]Value, len(record.fields))
		for name, value := range record.fields {
			copyFields[name] = value
		}
		for _, update := range e.Updates {
			value, err := in.evalExpr(update.Value, local)
			if err != nil {
				return nil, err
			}
			copyFields[update.Name] = value
		}
		return &instance{class: record.class, fields: copyFields}, nil
	case *parser.LambdaExpr:
		params := make([]string, len(e.Parameters))
		for i, param := range e.Parameters {
			params[i] = param.Name
		}
		return &closure{params: params, body: e.Body, blockBody: e.BlockBody, env: local}, nil
	case *parser.PlaceholderExpr:
		return nil, RuntimeError{Message: "placeholder is not supported here", Span: e.Span}
	default:
		return nil, RuntimeError{Message: "unsupported expression", Span: exprSpan(expr)}
	}
}

func (in *Interpreter) evalCall(call *parser.CallExpr, local *env) (Value, error) {
	if ident, ok := call.Callee.(*parser.Identifier); ok && ident.Name == "this" {
		ctorFlag, ok := local.get("__constructor__")
		if !ok || ctorFlag.value != true {
			return nil, RuntimeError{Message: "'this(...)' is only valid inside constructors", Span: call.Span}
		}
		thisSlot, ok := local.get("this")
		if !ok {
			return nil, RuntimeError{Message: "missing this receiver", Span: call.Span}
		}
		obj, ok := thisSlot.value.(*instance)
		if !ok {
			return nil, RuntimeError{Message: "invalid this receiver", Span: call.Span}
		}
		args := make([]namedValueArg, len(call.Args))
		for i, arg := range call.Args {
			value, err := in.evalExpr(arg.Value, local)
			if err != nil {
				return nil, err
			}
			args[i] = namedValueArg{Name: arg.Name, Value: value, Span: arg.Span}
		}
		ordered := namedArgValues(args)
		if hasNamedParserArgs(call.Args) {
			reordered, err := reorderConstructorValueArgs(obj.class, args, call.Span)
			if err != nil {
				return nil, err
			}
			ordered = reordered
		}
		return nil, in.constructInto(obj, obj.class, ordered, local, call.Span)
	}
	if member, ok := call.Callee.(*parser.MemberExpr); ok {
		return in.evalMethodCall(member, call.Args, local)
	}
	callee, err := in.evalExpr(call.Callee, local)
	if err != nil {
		return nil, err
	}
	if fn, ok := callee.(builtinRef); ok {
		return in.callBuiltin(fn.name, call.Args, nil, local, call.Span)
	}
	args := make([]namedValueArg, len(call.Args))
	for i, arg := range call.Args {
		value, err := in.evalExpr(arg.Value, local)
		if err != nil {
			return nil, err
		}
		args[i] = namedValueArg{Name: arg.Name, Value: value, Span: arg.Span}
	}
	switch fn := callee.(type) {
	case functionRef:
		ordered := namedArgValues(args)
		if hasNamedParserArgs(call.Args) {
			decl, ok := fn.module.functions[fn.name]
			if !ok {
				return nil, RuntimeError{Message: "undefined function '" + fn.name + "'", Span: call.Span}
			}
			reordered, err := reorderNamedValueArgs(decl.Parameters, args, call.Span, "function '"+fn.name+"'")
			if err != nil {
				return nil, err
			}
			ordered = reordered
		}
		return fn.module.callFunctionByName(fn.name, ordered, local)
	case classRef:
		class, ok := fn.module.classes[fn.name]
		if !ok {
			return nil, RuntimeError{Message: "undefined class '" + fn.name + "'", Span: call.Span}
		}
			ordered := namedArgValues(args)
			if hasNamedParserArgs(call.Args) {
				reordered, err := reorderConstructorValueArgs(class, args, call.Span)
				if err != nil {
					return nil, err
				}
				ordered = reordered
			}
		return fn.module.construct(class, ordered, local)
	case enumCaseRef:
		class, ok := fn.module.classes[fn.enumName]
		if !ok {
			return nil, RuntimeError{Message: "undefined enum '" + fn.enumName + "'", Span: call.Span}
		}
		for _, enumCase := range class.Cases {
			if enumCase.Name != fn.caseName {
				continue
			}
			ordered := namedArgValues(args)
			if hasNamedParserArgs(call.Args) {
				params := make([]parser.Parameter, len(enumCase.Fields))
				for i, field := range enumCase.Fields {
					params[i] = parser.Parameter{Name: field.Name, Type: field.Type, Span: field.Span}
				}
				reordered, err := reorderNamedValueArgs(params, args, call.Span, "enum case '"+fn.caseName+"'")
				if err != nil {
					return nil, err
				}
				ordered = reordered
			}
			return fn.module.constructEnumCase(class, enumCase, ordered, local, call.Span)
		}
		return nil, RuntimeError{Message: "unknown enum case '" + fn.caseName + "'", Span: call.Span}
	case *closure:
		if hasNamedParserArgs(call.Args) {
			return nil, RuntimeError{Message: "named arguments require a direct function, method, or constructor call", Span: call.Span}
		}
		return in.callClosure(fn, namedArgValues(args))
	default:
		return nil, RuntimeError{Message: "value is not callable", Span: call.Span}
	}
}

func (in *Interpreter) callClosure(fn *closure, args []Value) (Value, error) {
	if !acceptsClosureArgCount(fn, len(args)) {
		return nil, RuntimeError{Message: fmt.Sprintf("lambda expects %s args, got %d", expectedClosureArgs(fn), len(args)), Span: parser.Span{}}
	}
	local := newEnv(fn.env)
	for i, param := range fn.params {
		if fn.variadic && i == len(fn.params)-1 {
			local.define(param, &nativeList{items: append([]Value{}, args[i:]...)}, false)
			break
		}
		local.define(param, args[i], false)
	}
	if fn.body != nil {
		value, err := in.evalExpr(fn.body, local)
		if err != nil {
			return nil, err
		}
		if isUnitTypeRef(fn.returnType) {
			return nil, nil
		}
		return in.coerceValueForTypeRef(fn.returnType, value), nil
	}
	if fn.blockBody != nil {
		value, signal, err := in.execBlock(fn.blockBody, local, nil)
		if err != nil {
			return nil, err
		}
		if ret, ok := signal.(returnSignal); ok {
			return in.coerceValueForTypeRef(fn.returnType, ret.value), nil
		}
		if isUnitTypeRef(fn.returnType) {
			return nil, nil
		}
		return in.coerceValueForTypeRef(fn.returnType, value), nil
	}
	return nil, nil
}

func (in *Interpreter) evalMethodCall(member *parser.MemberExpr, argExprs []parser.CallArg, local *env) (Value, error) {
	receiver, err := in.evalExpr(member.Receiver, local)
	if err != nil {
		return nil, err
	}
	args := make([]namedValueArg, len(argExprs))
	for i, arg := range argExprs {
		value, err := in.evalExpr(arg.Value, local)
		if err != nil {
			return nil, err
		}
		args[i] = namedValueArg{Name: arg.Name, Value: value, Span: arg.Span}
	}
	if modRef, ok := receiver.(moduleRef); ok {
		mod, ok := in.imports[modRef.name]
		if !ok {
			return nil, RuntimeError{Message: "unknown module '" + modRef.name + "'", Span: member.Span}
		}
		ordered := namedArgValues(args)
		if hasNamedParserArgs(argExprs) {
			if decl, ok := mod.functions[member.Name]; ok {
				reordered, err := reorderNamedValueArgs(decl.Parameters, args, member.Span, "function '"+member.Name+"'")
				if err != nil {
					return nil, err
				}
				ordered = reordered
			}
		}
		if _, ok := mod.functions[member.Name]; ok {
			return mod.callFunctionByName(member.Name, ordered, mod.globals)
		}
		if _, ok := mod.classes[member.Name]; ok {
			class := mod.classes[member.Name]
			if hasNamedParserArgs(argExprs) {
				reordered, err := reorderConstructorValueArgs(class, args, member.Span)
				if err != nil {
					return nil, err
				}
				ordered = reordered
			}
			return mod.construct(class, ordered, mod.globals)
		}
		return nil, RuntimeError{Message: "unknown member '" + member.Name + "'", Span: member.Span}
	}
	if class, ok := receiver.(classRef); ok {
		decl, found := class.module.classes[class.name]
		if !found {
			return nil, RuntimeError{Message: "unknown class '" + class.name + "'", Span: member.Span}
		}
		if decl.Enum {
			for _, enumCase := range decl.Cases {
				if enumCase.Name != member.Name {
					continue
				}
				ordered := namedArgValues(args)
				if hasNamedParserArgs(argExprs) {
					params := make([]parser.Parameter, len(enumCase.Fields))
					for i, field := range enumCase.Fields {
						params[i] = parser.Parameter{Name: field.Name, Type: field.Type, Span: field.Span}
					}
					reordered, err := reorderNamedValueArgs(params, args, member.Span, "enum case '"+member.Name+"'")
					if err != nil {
						return nil, err
					}
					ordered = reordered
				}
				return class.module.constructEnumCase(decl, enumCase, ordered, local, member.Span)
			}
		}
	}
	if native, ok := in.callNativeMethod(receiver, member.Name, args, local, member.Span); ok {
		return native.value, native.err
	}
	obj, ok := receiver.(*instance)
	if !ok {
		return nil, RuntimeError{Message: "member call requires class instance", Span: member.Span}
	}
	if hasNamedParserArgs(argExprs) {
		var candidates []*parser.MethodDecl
		for _, method := range obj.class.Methods {
			if method.Name == member.Name {
				candidates = append(candidates, method)
			}
		}
		if len(candidates) == 1 {
			ordered, err := reorderNamedValueArgs(candidates[0].Parameters, args, member.Span, "method '"+member.Name+"'")
			if err != nil {
				return nil, err
			}
			return in.callMethod(obj, candidates[0], ordered, local)
		}
	}
	var firstErr error
	var fallbackMethod *parser.MethodDecl
	var fallbackArgs []Value
	for _, method := range obj.class.Methods {
		if method.Name != member.Name {
			continue
		}
		ordered := namedArgValues(args)
		if hasNamedParserArgs(argExprs) {
			reordered, err := reorderNamedValueArgs(method.Parameters, args, member.Span, "method '"+member.Name+"'")
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			ordered = reordered
			if fallbackMethod == nil {
				fallbackMethod = method
				fallbackArgs = ordered
			}
		}
		if in.runtimeMethodMatches(method, ordered) {
			return in.callMethod(obj, method, ordered, local)
		}
	}
	if fallbackMethod != nil {
		return in.callMethod(obj, fallbackMethod, fallbackArgs, local)
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, RuntimeError{Message: "unknown method '" + member.Name + "'", Span: member.Span}
}

func (in *Interpreter) evalMember(receiver Value, expr *parser.MemberExpr) (Value, error) {
	switch value := receiver.(type) {
	case moduleRef:
		mod, ok := in.imports[value.name]
		if !ok {
			return nil, RuntimeError{Message: "unknown module '" + value.name + "'", Span: expr.Span}
		}
		if _, ok := mod.functions[expr.Name]; ok {
			return functionRef{module: mod, name: expr.Name}, nil
		}
		if _, ok := mod.classes[expr.Name]; ok {
			return classRef{module: mod, name: expr.Name}, nil
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	case *instance:
		if field, ok := value.fields[expr.Name]; ok {
			if _, deferred := field.(deferredValue); deferred {
				return nil, RuntimeError{Message: "field '" + expr.Name + "' is deferred and has not been assigned", Span: expr.Span}
			}
			return field, nil
		}
		for _, method := range value.class.Methods {
			if method.Name == expr.Name {
				return nil, RuntimeError{Message: "method '" + expr.Name + "' must be called with ()", Span: expr.Span}
			}
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	case classRef:
		class, ok := value.module.classes[value.name]
		if !ok {
			return nil, RuntimeError{Message: "unknown class '" + value.name + "'", Span: expr.Span}
		}
		if class.Enum {
			for _, enumCase := range class.Cases {
				if enumCase.Name != expr.Name {
					continue
				}
				if len(enumCase.Fields) == 0 {
					return value.module.constructEnumCase(class, enumCase, nil, value.module.globals, expr.Span)
				}
				return enumCaseRef{module: value.module, enumName: value.name, caseName: expr.Name}, nil
			}
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	case *nativeList, *nativeArray, *nativeOption, *nativeSet, *nativeMap, *nativeTerm:
		if in.nativeHasMethod(receiver, expr.Name) {
			return nil, RuntimeError{Message: "method '" + expr.Name + "' must be called with ()", Span: expr.Span}
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	case *nativeTuple:
		for i, name := range value.names {
			if name == expr.Name {
				if i < len(value.items) {
					return value.items[i], nil
				}
				break
			}
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	default:
		return nil, RuntimeError{Message: "member access expects class instance", Span: expr.Span}
	}
}

type nativeCallResult struct {
	value Value
	err   error
}

type namedValueArg struct {
	Name  string
	Value Value
	Span  parser.Span
}

func namedArgValues(args []namedValueArg) []Value {
	values := make([]Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return values
}

func hasNamedParserArgs(args []parser.CallArg) bool {
	for _, arg := range args {
		if arg.Name != "" {
			return true
		}
	}
	return false
}

func hasNamedValueArgs(args []namedValueArg) bool {
	for _, arg := range args {
		if arg.Name != "" {
			return true
		}
	}
	return false
}

func reorderNamedValueArgs(params []parser.Parameter, args []namedValueArg, span parser.Span, callable string) ([]Value, error) {
	if len(params) > 0 && params[len(params)-1].Variadic {
		return nil, RuntimeError{Message: "named arguments are not supported for variadic " + callable, Span: span}
	}
	ordered := make([]Value, len(params))
	filled := make([]bool, len(params))
	seenNamed := false
	pos := 0
	for _, arg := range args {
		if arg.Name == "" {
			if seenNamed {
				return nil, RuntimeError{Message: "positional arguments cannot follow named arguments", Span: arg.Span}
			}
			if pos >= len(params) {
				return nil, RuntimeError{Message: fmt.Sprintf("%s expects %d arguments, got %d", callable, len(params), len(args)), Span: span}
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
			return nil, RuntimeError{Message: "unknown named argument '" + arg.Name + "'", Span: arg.Span}
		}
		if filled[paramIndex] {
			return nil, RuntimeError{Message: "argument '" + arg.Name + "' was provided more than once", Span: arg.Span}
		}
		ordered[paramIndex] = arg.Value
		filled[paramIndex] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, RuntimeError{Message: "missing argument '" + params[i].Name + "' in " + callable, Span: span}
		}
	}
	return ordered, nil
}

func (in *Interpreter) callBuiltin(name string, argExprs []parser.CallArg, args []Value, local *env, span parser.Span) (Value, error) {
	if hasNamedParserArgs(argExprs) {
		return nil, RuntimeError{Message: "named arguments are not supported for builtin constructors", Span: span}
	}
	switch name {
	case "List":
		if args == nil {
			args = make([]Value, len(argExprs))
			for i, arg := range argExprs {
				value, err := in.evalExpr(arg.Value, local)
				if err != nil {
					return nil, err
				}
				args[i] = value
			}
		}
		return &nativeList{items: append([]Value(nil), args...)}, nil
	case "Array":
		if len(argExprs) != 1 {
			return nil, RuntimeError{Message: fmt.Sprintf("Array constructor expects 1 argument, got %d", len(argExprs)), Span: span}
		}
		if args == nil {
			args = make([]Value, len(argExprs))
			for i, arg := range argExprs {
				value, err := in.evalExpr(arg.Value, local)
				if err != nil {
					return nil, err
				}
				args[i] = value
			}
		}
		length, ok := args[0].(int64)
		if !ok {
			return nil, RuntimeError{Message: "Array constructor length must be Int", Span: exprSpan(argExprs[0].Value)}
		}
		if length < 0 {
			return nil, RuntimeError{Message: "Array constructor length must be non-negative", Span: exprSpan(argExprs[0].Value)}
		}
		return &nativeArray{items: make([]Value, int(length))}, nil
	case "Set":
		if args == nil {
			args = make([]Value, len(argExprs))
			for i, arg := range argExprs {
				value, err := in.evalExpr(arg.Value, local)
				if err != nil {
					return nil, err
				}
				args[i] = value
			}
		}
		s := &nativeSet{keys: map[string]Value{}, order: []string{}}
		for _, arg := range args {
			key, err := nativeKey(arg, span, local, in)
			if err != nil {
				return nil, err
			}
			if _, ok := s.keys[key]; !ok {
				s.order = append(s.order, key)
			}
			s.keys[key] = arg
		}
		return s, nil
	case "Some":
		if len(argExprs) != 1 {
			return nil, RuntimeError{Message: fmt.Sprintf("Some constructor expects 1 argument, got %d", len(argExprs)), Span: span}
		}
		if args == nil {
			args = make([]Value, len(argExprs))
			for i, arg := range argExprs {
				value, err := in.evalExpr(arg.Value, local)
				if err != nil {
					return nil, err
				}
				args[i] = value
			}
		}
		return in.constructStdlibOption(args[0], true, local, span)
	case "None":
		if len(argExprs) != 0 {
			return nil, RuntimeError{Message: fmt.Sprintf("None constructor expects 0 arguments, got %d", len(argExprs)), Span: span}
		}
		return in.constructStdlibOption(nil, false, local, span)
	case "Map":
		m := &nativeMap{items: map[string]Value{}, keys: map[string]Value{}, order: []string{}}
		for _, argExpr := range argExprs {
			pair, ok := argExpr.Value.(*parser.BinaryExpr)
			if !ok || pair.Operator != ":" {
				return nil, RuntimeError{Message: "Map constructor expects key : value pairs", Span: span}
			}
			keyValue, err := in.evalExpr(pair.Left, local)
			if err != nil {
				return nil, err
			}
			valueValue, err := in.evalExpr(pair.Right, local)
			if err != nil {
				return nil, err
			}
			key, err := nativeKey(keyValue, exprSpan(pair.Left), local, in)
			if err != nil {
				return nil, err
			}
			if _, ok := m.items[key]; !ok {
				m.order = append(m.order, key)
				m.keys[key] = keyValue
			}
			m.items[key] = valueValue
		}
		return m, nil
	default:
		return nil, RuntimeError{Message: "value is not callable", Span: span}
	}
}

func (in *Interpreter) nativeHasMethod(receiver Value, name string) bool {
	switch receiver.(type) {
	case *nativeList:
		return name == "append" || name == "get" || name == "size" || name == "iterator"
	case *nativeListIterator:
		return name == "hasNext" || name == "next"
	case *nativeArray:
		return name == "size"
	case *nativeOption:
		return name == "isSet" || name == "get" || name == "getOr"
	case *nativeSet:
		return name == "add" || name == "contains" || name == "size"
	case *nativeMap:
		return name == "set" || name == "get" || name == "contains" || name == "size"
	case *nativeTerm:
		return name == "print" || name == "println"
	default:
		return false
	}
}

func nativeMethodParams(receiver Value, name string) ([]parser.Parameter, bool) {
	switch receiver.(type) {
	case *nativeList:
		switch name {
		case "append":
			return []parser.Parameter{{Name: "value"}}, true
		case "get":
			return []parser.Parameter{{Name: "index"}}, true
		case "size":
			return nil, true
		case "iterator":
			return nil, true
		}
	case *nativeListIterator:
		switch name {
		case "hasNext", "next":
			return nil, true
		}
	case *nativeArray:
		if name == "size" {
			return nil, true
		}
	case *nativeOption:
		switch name {
		case "isSet", "get":
			return nil, true
		case "getOr":
			return []parser.Parameter{{Name: "defaultValue"}}, true
		}
	case *nativeSet:
		switch name {
		case "add", "contains":
			return []parser.Parameter{{Name: "value"}}, true
		case "size":
			return nil, true
		}
	case *nativeMap:
		switch name {
		case "set":
			return []parser.Parameter{{Name: "key"}, {Name: "value"}}, true
		case "get", "contains":
			return []parser.Parameter{{Name: "key"}}, true
		case "size":
			return nil, true
		}
	case *nativeTerm:
		switch name {
		case "print":
			return []parser.Parameter{{Name: "value"}}, true
		case "println":
			return []parser.Parameter{{Name: "value", Variadic: true}}, true
		}
	}
	return nil, false
}

func (in *Interpreter) callNativeMethod(receiver Value, name string, args []namedValueArg, local *env, span parser.Span) (nativeCallResult, bool) {
	ordered := namedArgValues(args)
	if hasNamedValueArgs(args) {
		params, ok := nativeMethodParams(receiver, name)
		if !ok {
			return nativeCallResult{}, false
		}
		reordered, err := reorderNamedValueArgs(params, args, span, "method '"+name+"'")
		if err != nil {
			return nativeCallResult{err: err}, true
		}
		ordered = reordered
	}
	switch value := receiver.(type) {
	case *nativeList:
		switch name {
		case "append":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "append expects 1 argument", Span: span}}, true
			}
			value.items = append(value.items, ordered[0])
			return nativeCallResult{value: value}, true
		case "get":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "get expects 1 argument", Span: span}}, true
			}
			index, ok := ordered[0].(int64)
			if !ok {
				return nativeCallResult{err: RuntimeError{Message: "get index must be Int", Span: span}}, true
			}
			if index < 0 || index >= int64(len(value.items)) {
				result, err := in.constructStdlibOption(nil, false, local, span)
				return nativeCallResult{value: result, err: err}, true
			}
			result, err := in.constructStdlibOption(value.items[index], true, local, span)
			return nativeCallResult{value: result, err: err}, true
		case "size":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "size expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: int64(len(value.items))}, true
		case "iterator":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "iterator expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: &nativeListIterator{items: append([]Value(nil), value.items...)}}, true
		}
	case *nativeListIterator:
		switch name {
		case "hasNext":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "hasNext expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: value.index < len(value.items)}, true
		case "next":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "next expects 0 arguments", Span: span}}, true
			}
			if value.index >= len(value.items) {
				return nativeCallResult{err: RuntimeError{Message: "iterator exhausted", Span: span}}, true
			}
			item := value.items[value.index]
			value.index++
			return nativeCallResult{value: item}, true
		}
	case *nativeArray:
		switch name {
		case "size":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "size expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: int64(len(value.items))}, true
		}
	case *nativeOption:
		switch name {
		case "isSet":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "isSet expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: value.set}, true
		case "get":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "get expects 0 arguments", Span: span}}, true
			}
			if !value.set {
				return nativeCallResult{err: RuntimeError{Message: "Option has no value", Span: span}}, true
			}
			return nativeCallResult{value: value.value}, true
		case "getOr":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "getOr expects 1 argument", Span: span}}, true
			}
			if value.set {
				return nativeCallResult{value: value.value}, true
			}
			return nativeCallResult{value: ordered[0]}, true
		}
	case *nativeSet:
		switch name {
		case "add":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "add expects 1 argument", Span: span}}, true
			}
			key, err := nativeKey(ordered[0], span, local, in)
			if err != nil {
				return nativeCallResult{err: err}, true
			}
			if _, ok := value.keys[key]; !ok {
				value.order = append(value.order, key)
			}
			value.keys[key] = ordered[0]
			return nativeCallResult{value: value}, true
		case "contains":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "contains expects 1 argument", Span: span}}, true
			}
			key, err := nativeKey(ordered[0], span, local, in)
			if err != nil {
				return nativeCallResult{err: err}, true
			}
			_, ok := value.keys[key]
			return nativeCallResult{value: ok}, true
		case "size":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "size expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: int64(len(value.keys))}, true
		}
	case *nativeMap:
		switch name {
		case "set":
			if len(ordered) != 2 {
				return nativeCallResult{err: RuntimeError{Message: "set expects 2 arguments", Span: span}}, true
			}
			key, err := nativeKey(ordered[0], span, local, in)
			if err != nil {
				return nativeCallResult{err: err}, true
			}
			if _, ok := value.items[key]; !ok {
				value.order = append(value.order, key)
				value.keys[key] = ordered[0]
			}
			value.items[key] = ordered[1]
			return nativeCallResult{value: value}, true
		case "get":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "get expects 1 argument", Span: span}}, true
			}
			key, err := nativeKey(ordered[0], span, local, in)
			if err != nil {
				return nativeCallResult{err: err}, true
			}
			result, ok := value.items[key]
			if !ok {
				opt, err := in.constructStdlibOption(nil, false, local, span)
				return nativeCallResult{value: opt, err: err}, true
			}
			opt, err := in.constructStdlibOption(result, true, local, span)
			return nativeCallResult{value: opt, err: err}, true
		case "contains":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "contains expects 1 argument", Span: span}}, true
			}
			key, err := nativeKey(ordered[0], span, local, in)
			if err != nil {
				return nativeCallResult{err: err}, true
			}
			_, ok := value.items[key]
			return nativeCallResult{value: ok}, true
		case "size":
			if len(ordered) != 0 {
				return nativeCallResult{err: RuntimeError{Message: "size expects 0 arguments", Span: span}}, true
			}
			return nativeCallResult{value: int64(len(value.items))}, true
		}
	case *nativeTerm:
		switch name {
		case "print":
			if len(ordered) != 1 {
				return nativeCallResult{err: RuntimeError{Message: "print expects 1 argument", Span: span}}, true
			}
			fmt.Print(fmt.Sprint(ordered[0]))
			return nativeCallResult{value: value}, true
		case "println":
			parts := make([]any, len(ordered))
			for i, arg := range ordered {
				parts[i] = fmt.Sprint(arg)
			}
			fmt.Println(parts...)
			return nativeCallResult{value: value}, true
		}
	}
	return nativeCallResult{}, false
}

func nativeKey(value Value, span parser.Span, local *env, in *Interpreter) (string, error) {
	switch v := value.(type) {
	case int64:
		return fmt.Sprintf("i:%d", v), nil
	case float64:
		return fmt.Sprintf("f:%g", v), nil
	case bool:
		if v {
			return "b:true", nil
		}
		return "b:false", nil
	case string:
		return "s:" + v, nil
	case rune:
		return fmt.Sprintf("r:%d", v), nil
	case *instance:
		return fmt.Sprintf("o:%p", v), nil
	default:
		return "", RuntimeError{Message: "unsupported Map/Set key type", Span: span}
	}
}

type moduleRef struct{ name string }
type functionRef struct {
	module *Interpreter
	name   string
}
type classRef struct {
	module *Interpreter
	name   string
}

type enumCaseRef struct {
	module   *Interpreter
	enumName string
	caseName string
}

func newEnv(parent *env) *env {
	return &env{parent: parent, values: map[string]slot{}}
}

func (e *env) define(name string, value Value, mutable bool) {
	e.values[name] = slot{value: value, mutable: mutable}
}

func (e *env) get(name string) (slot, bool) {
	if value, ok := e.values[name]; ok {
		return value, true
	}
	if e.parent != nil {
		return e.parent.get(name)
	}
	return slot{}, false
}

func (e *env) resolve(name string) (*slotRef, bool) {
	if _, ok := e.values[name]; ok {
		return &slotRef{owner: e, name: name}, true
	}
	if e.parent != nil {
		return e.parent.resolve(name)
	}
	return nil, false
}

type slotRef struct {
	owner *env
	name  string
}

func (r *slotRef) mutable() bool { return r.owner.values[r.name].mutable }
func (r *slotRef) set(value Value) error {
	slot := r.owner.values[r.name]
	slot.value = value
	r.owner.values[r.name] = slot
	return nil
}
func (r *slotRef) value() Value { return r.owner.values[r.name].value }

func acceptsArgCount(params []parser.Parameter, count int) bool {
	if len(params) == 0 {
		return count == 0
	}
	if params[len(params)-1].Variadic {
		return count >= len(params)-1
	}
	return count == len(params)
}

func expectedCallableArgs(params []parser.Parameter) string {
	if len(params) > 0 && params[len(params)-1].Variadic {
		return fmt.Sprintf("at least %d", len(params)-1)
	}
	return fmt.Sprintf("%d", len(params))
}

func acceptsClosureArgCount(fn *closure, count int) bool {
	if fn.variadic {
		return count >= len(fn.params)-1
	}
	return count == len(fn.params)
}

func expectedClosureArgs(fn *closure) string {
	if fn.variadic {
		return fmt.Sprintf("at least %d", len(fn.params)-1)
	}
	return fmt.Sprintf("%d", len(fn.params))
}

func (in *Interpreter) runtimeMethodMatches(method *parser.MethodDecl, args []Value) bool {
	if !acceptsArgCount(method.Parameters, len(args)) {
		return false
	}
	for i, arg := range args {
		paramIndex := i
		if len(method.Parameters) > 0 && method.Parameters[len(method.Parameters)-1].Variadic && paramIndex >= len(method.Parameters)-1 {
			paramIndex = len(method.Parameters) - 1
		}
		if paramIndex >= len(method.Parameters) {
			return false
		}
		if !in.runtimeValueMatchesType(arg, method.Parameters[paramIndex].Type) {
			return false
		}
	}
	return true
}

func (in *Interpreter) runtimeValueMatchesType(value Value, ref *parser.TypeRef) bool {
	if ref == nil {
		return true
	}
	if len(ref.TupleElements) > 0 {
		tuple, ok := value.(*nativeTuple)
		return ok && len(ref.TupleElements) == len(tuple.items)
	}
	switch ref.Name {
	case "", "Unit", "Int", "Float", "Bool", "String", "Rune", "List", "Iterable", "Iterator", "Set", "Map", "Option", "Array", "Term":
	default:
		if _, ok := in.classes[ref.Name]; !ok {
			if _, ok := in.interfaces[ref.Name]; !ok {
				return true
			}
		}
	}
	switch ref.Name {
	case "Unit":
		return value == nil
	case "Int":
		_, ok := value.(int64)
		return ok
	case "Float":
		_, ok := value.(float64)
		return ok
	case "Bool":
		_, ok := value.(bool)
		return ok
	case "String":
		_, ok := value.(string)
		return ok
	case "Rune":
		_, ok := value.(rune)
		return ok
	case "List":
		_, ok := value.(*nativeList)
		return ok
	case "Iterable":
		switch value.(type) {
		case *nativeList, *nativeArray, *nativeSet, *nativeListIterator:
			return true
		}
		return false
	case "Iterator":
		_, ok := value.(*nativeListIterator)
		return ok
	case "Set":
		_, ok := value.(*nativeSet)
		return ok
	case "Map":
		_, ok := value.(*nativeMap)
		return ok
	case "Option":
		if instanceValue, ok := value.(*instance); ok && instanceValue.class.Name == "Option" {
			return true
		}
		_, ok := value.(*nativeOption)
		return ok
	case "Array":
		_, ok := value.(*nativeArray)
		return ok
	case "Term":
		_, ok := value.(*nativeTerm)
		return ok
	default:
		if instanceValue, ok := value.(*instance); ok {
			if instanceValue.class.Name == ref.Name {
				return true
			}
			for _, impl := range instanceValue.class.Implements {
				if impl.Name == ref.Name || in.interfaceExtends(impl.Name, ref.Name, map[string]bool{}) {
					return true
				}
			}
		}
		return false
	}
}

func (in *Interpreter) interfaceExtends(name, target string, seen map[string]bool) bool {
	if name == target {
		return true
	}
	if seen[name] {
		return false
	}
	seen[name] = true
	iface, ok := in.interfaces[name]
	if !ok {
		return false
	}
	for _, parent := range iface.Extends {
		if parent.Name == target || in.interfaceExtends(parent.Name, target, seen) {
			return true
		}
	}
	return false
}

func applyBinary(op string, left, right Value, span parser.Span) (Value, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		return applyArithmetic(op, left, right, span)
	case "<", "<=", ">", ">=":
		return compareOrdered(op, left, right, span)
	case "&&":
		lb, err := asBool(left, span)
		if err != nil {
			return nil, err
		}
		rb, err := asBool(right, span)
		if err != nil {
			return nil, err
		}
		if op == "&&" {
			return lb && rb, nil
		}
	case "||":
		lb, err := asBool(left, span)
		if err != nil {
			return nil, err
		}
		rb, err := asBool(right, span)
		if err != nil {
			return nil, err
		}
		return lb || rb, nil
	case "..":
		start, ok1 := left.(int64)
		end, ok2 := right.(int64)
		if !ok1 || !ok2 {
			return nil, RuntimeError{Message: "range operands must be Int", Span: span}
		}
		var out []Value
		if start <= end {
			for i := start; i < end; i++ {
				out = append(out, i)
			}
		} else {
			for i := start; i > end; i-- {
				out = append(out, i)
			}
		}
		return out, nil
	}
	return nil, RuntimeError{Message: "unsupported operator '" + op + "'", Span: span}
}

func applyArithmetic(op string, left, right Value, span parser.Span) (Value, error) {
	if ls, ok := left.(string); ok && op == "+" {
		return ls + fmt.Sprint(right), nil
	}
	if rs, ok := right.(string); ok && op == "+" {
		return fmt.Sprint(left) + rs, nil
	}
	if li, ok := left.(int64); ok {
		if ri, ok := right.(int64); ok {
			switch op {
			case "+":
				return li + ri, nil
			case "-":
				return li - ri, nil
			case "*":
				return li * ri, nil
			case "/":
				return li / ri, nil
			case "%":
				return li % ri, nil
			}
		}
	}
	lf, lok := toFloat(left)
	rf, rok := toFloat(right)
	if lok && rok {
		switch op {
		case "+":
			return lf + rf, nil
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		case "/":
			return lf / rf, nil
		case "%":
			return math.Mod(lf, rf), nil
		}
	}
	return nil, RuntimeError{Message: "invalid arithmetic operands", Span: span}
}

func compareOrdered(op string, left, right Value, span parser.Span) (Value, error) {
	if li, ok := left.(int64); ok {
		if ri, ok := right.(int64); ok {
			return compareInts(op, li, ri), nil
		}
	}
	if lf, lok := toFloat(left); lok {
		if rf, rok := toFloat(right); rok {
			return compareFloats(op, lf, rf), nil
		}
	}
	if ls, ok := left.(string); ok {
		if rs, ok := right.(string); ok {
			return compareStrings(op, ls, rs), nil
		}
	}
	if lr, ok := left.(rune); ok {
		if rr, ok := right.(rune); ok {
			return compareInts(op, int64(lr), int64(rr)), nil
		}
	}
	return nil, RuntimeError{Message: "invalid comparison operands", Span: span}
}

func compareInts(op string, left, right int64) bool {
	switch op {
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	default:
		return false
	}
}

func compareFloats(op string, left, right float64) bool {
	switch op {
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	default:
		return false
	}
}

func compareStrings(op, left, right string) bool {
	switch op {
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	default:
		return false
	}
}

func (in *Interpreter) valuesEqual(left, right Value, span parser.Span, local *env) (bool, error) {
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		return ok && l == r, nil
	case float64:
		r, ok := toFloat(right)
		return ok && l == r, nil
	case bool:
		r, ok := right.(bool)
		return ok && l == r, nil
	case string:
		r, ok := right.(string)
		return ok && l == r, nil
	case rune:
		r, ok := right.(rune)
		return ok && l == r, nil
	case *instance:
		r, ok := right.(*instance)
		if !ok || l.class.Name != r.class.Name {
			return false, nil
		}
		if l.class.Enum || r.class.Enum {
			if l.caseName != r.caseName {
				return false, nil
			}
			for name, leftValue := range l.fields {
				equal, err := in.valuesEqual(leftValue, r.fields[name], span, local)
				if err != nil {
					return false, err
				}
				if !equal {
					return false, nil
				}
			}
			return true, nil
		}
		for _, method := range l.class.Methods {
			if method.Name == "equals" && len(method.Parameters) == 1 {
				value, err := in.callMethod(l, method, []Value{r}, local)
				if err != nil {
					return false, err
				}
				if result, ok := value.(bool); ok {
					return result, nil
				}
				return false, RuntimeError{Message: "equals must return Bool", Span: span}
			}
		}
		return false, nil
	default:
		return left == right, nil
	}
}

func toFloat(value Value) (float64, bool) {
	switch n := value.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func asBool(value Value, span parser.Span) (bool, error) {
	b, ok := value.(bool)
	if !ok {
		return false, RuntimeError{Message: "expected Bool", Span: span}
	}
	return b, nil
}

func (in *Interpreter) iterableToSlice(value Value, local *env, span parser.Span) ([]Value, bool) {
	switch items := value.(type) {
	case []Value:
		return items, true
	case *nativeList:
		return items.items, true
	case *nativeArray:
		return items.items, true
	case *nativeSet:
		out := make([]Value, 0, len(items.order))
		for _, key := range items.order {
			out = append(out, items.keys[key])
		}
		return out, true
	}
	iterator, err := in.invokeMethod(value, "iterator", nil, local, span)
	if err != nil {
		return nil, false
	}
	var out []Value
	for {
		hasNextValue, err := in.invokeMethod(iterator, "hasNext", nil, local, span)
		if err != nil {
			return nil, false
		}
		hasNext, ok := hasNextValue.(bool)
		if !ok || !hasNext {
			break
		}
		nextValue, err := in.invokeMethod(iterator, "next", nil, local, span)
		if err != nil {
			return nil, false
		}
		out = append(out, nextValue)
	}
	return out, true
}

func (in *Interpreter) invokeMethod(receiver Value, name string, args []Value, local *env, span parser.Span) (Value, error) {
	named := make([]namedValueArg, len(args))
	for i, arg := range args {
		named[i] = namedValueArg{Value: arg, Span: span}
	}
	if native, ok := in.callNativeMethod(receiver, name, named, local, span); ok {
		return native.value, native.err
	}
	obj, ok := receiver.(*instance)
	if !ok {
		return nil, RuntimeError{Message: "member call requires class instance", Span: span}
	}
	for _, method := range obj.class.Methods {
		if method.Name != name {
			continue
		}
		if in.runtimeMethodMatches(method, args) {
			return in.callMethod(obj, method, args, local)
		}
	}
	return nil, RuntimeError{Message: "unknown method '" + name + "'", Span: span}
}

func (in *Interpreter) constructStdlibOption(value Value, set bool, local *env, span parser.Span) (Value, error) {
	class, ok := in.classes["Option"]
	if !ok {
		if set {
			return &nativeOption{value: value, set: true}, nil
		}
		return &nativeOption{set: false}, nil
	}
	if set {
		return in.construct(class, []Value{value}, local)
	}
	return in.construct(class, nil, local)
}

func (in *Interpreter) bindingValues(bindings []parser.Binding, values []parser.Expr, local *env, span parser.Span) ([]Value, error) {
	if len(bindings) == 0 || len(values) == 0 {
		return nil, nil
	}
	if len(bindings) == len(values) {
		out := make([]Value, len(values))
		for i, expr := range values {
			if expr == nil {
				out[i] = deferredValue{}
				continue
			}
			value, err := in.evalExpr(expr, local)
			if err != nil {
				return nil, err
			}
			value = in.coerceValueForBinding(bindingTypeRef(bindings, i), value)
			out[i] = value
		}
		return out, nil
	}
	if len(values) == 1 {
		if values[0] == nil {
			return nil, RuntimeError{Message: "cannot destructure deferred value", Span: span}
		}
		value, err := in.evalExpr(values[0], local)
		if err != nil {
			return nil, err
		}
		tuple, ok := value.(*nativeTuple)
		if !ok {
			return nil, RuntimeError{Message: fmt.Sprintf("binding expects %d values, got 1", len(bindings)), Span: span}
		}
		if len(tuple.items) != len(bindings) {
			return nil, RuntimeError{Message: fmt.Sprintf("binding expects %d tuple values, got %d", len(bindings), len(tuple.items)), Span: span}
		}
		return append([]Value(nil), tuple.items...), nil
	}
	return nil, RuntimeError{Message: fmt.Sprintf("binding expects %d values, got %d", len(bindings), len(values)), Span: span}
}

func bindingTypeRef(bindings []parser.Binding, index int) *parser.TypeRef {
	if index < 0 || index >= len(bindings) {
		return nil
	}
	return bindings[index].Type
}

func (in *Interpreter) coerceValueForBinding(ref *parser.TypeRef, value Value) Value {
	return in.coerceValueForTypeRef(ref, value)
}

func isUnitTypeRef(ref *parser.TypeRef) bool {
	return ref != nil && ref.Name == "Unit" && len(ref.Arguments) == 0 && len(ref.TupleElements) == 0 && ref.ReturnType == nil
}

func (in *Interpreter) coerceValueForTypeRef(ref *parser.TypeRef, value Value) Value {
	if ref == nil {
		return value
	}
	if isUnitTypeRef(ref) {
		return nil
	}
	tuple, ok := value.(*nativeTuple)
	if !ok || len(ref.TupleElements) == 0 {
		return value
	}
	renamed := append([]string(nil), ref.TupleNames...)
	return &nativeTuple{items: append([]Value(nil), tuple.items...), names: renamed}
}

func preserveTupleNames(current Value, next Value) Value {
	currentTuple, ok := current.(*nativeTuple)
	if !ok {
		return next
	}
	nextTuple, ok := next.(*nativeTuple)
	if !ok {
		return next
	}
	return &nativeTuple{items: append([]Value(nil), nextTuple.items...), names: append([]string(nil), currentTuple.names...)}
}

func (in *Interpreter) assignmentValues(targetCount int, values []parser.Expr, local *env, span parser.Span) ([]Value, error) {
	if targetCount == len(values) {
		out := make([]Value, len(values))
		for i, expr := range values {
			value, err := in.evalExpr(expr, local)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	}
	if len(values) == 1 {
		value, err := in.evalExpr(values[0], local)
		if err != nil {
			return nil, err
		}
		tuple, ok := value.(*nativeTuple)
		if !ok {
			return nil, RuntimeError{Message: fmt.Sprintf("assignment expects %d values, got 1", targetCount), Span: span}
		}
		if len(tuple.items) != targetCount {
			return nil, RuntimeError{Message: fmt.Sprintf("assignment expects %d tuple values, got %d", targetCount, len(tuple.items)), Span: span}
		}
		return append([]Value(nil), tuple.items...), nil
	}
	return nil, RuntimeError{Message: fmt.Sprintf("assignment expects %d values, got %d", targetCount, len(values)), Span: span}
}

func indexedItems(value Value) ([]Value, bool) {
	switch items := value.(type) {
	case []Value:
		return items, true
	case *nativeList:
		return items.items, true
	case *nativeArray:
		return items.items, true
	default:
		return nil, false
	}
}

func zeroValue(ref *parser.TypeRef) Value {
	if ref == nil {
		return nil
	}
	switch ref.Name {
	case "Int", "Int64":
		return int64(0)
	case "Float", "Float64":
		return float64(0)
	case "Bool":
		return false
	case "String":
		return ""
	case "Rune":
		return rune(0)
	case "Unit":
		return nil
	case "List":
		return &nativeList{items: []Value{}}
	case "Set":
		return &nativeSet{keys: map[string]Value{}, order: []string{}}
	case "Map":
		return &nativeMap{items: map[string]Value{}, keys: map[string]Value{}, order: []string{}}
	case "Array":
		return &nativeArray{items: []Value{}}
	case "Tuple":
		items := make([]Value, len(ref.TupleElements))
		for i, elem := range ref.TupleElements {
			items[i] = zeroValue(elem)
		}
		return &nativeTuple{items: items, names: append([]string(nil), ref.TupleNames...)}
	case "Option":
		return &nativeOption{set: false}
	case "Term":
		return &nativeTerm{}
	default:
		return nil
	}
}

func stmtSpan(stmt parser.Statement) parser.Span {
	switch s := stmt.(type) {
	case *parser.ValStmt:
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
	case *parser.TupleLiteral:
		return e.Span
	case *parser.CallExpr:
		return e.Span
	case *parser.MemberExpr:
		return e.Span
	case *parser.IndexExpr:
		return e.Span
	case *parser.RecordUpdateExpr:
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
