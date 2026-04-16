package interpreter

import (
	"fmt"
	"math"
	"strconv"

	"a-lang/parser"
)

type Value any

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
}

type instance struct {
	class  *parser.ClassDecl
	fields map[string]Value
}

type closure struct {
	params    []parser.LambdaParameter
	body      parser.Expr
	blockBody *parser.BlockStmt
	env       *env
}

type returnSignal struct {
	value Value
}

type breakSignal struct{}

func New(program *parser.Program) *Interpreter {
	in := &Interpreter{
		program:   program,
		functions: map[string]*parser.FunctionDecl{},
		classes:   map[string]*parser.ClassDecl{},
	}
	for _, fn := range program.Functions {
		in.functions[fn.Name] = fn
	}
	for _, class := range program.Classes {
		in.classes[class.Name] = class
	}
	return in
}

func (in *Interpreter) Call(function string, args ...Value) (Value, error) {
	global := newEnv(nil)
	if err := in.execTopLevel(global); err != nil {
		return nil, err
	}
	return in.callFunctionByName(function, args, global)
}

func (in *Interpreter) execTopLevel(global *env) error {
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
	if len(args) != len(fn.Parameters) {
		return nil, RuntimeError{Message: fmt.Sprintf("function '%s' expects %d args, got %d", fn.Name, len(fn.Parameters), len(args)), Span: fn.Span}
	}
	local := newEnv(parent)
	for i, param := range fn.Parameters {
		local.define(param.Name, args[i], false)
	}
	value, signal, err := in.execBlock(fn.Body, local, nil)
	if err != nil {
		return nil, err
	}
	if ret, ok := signal.(returnSignal); ok {
		return ret.value, nil
	}
	return value, nil
}

func (in *Interpreter) callMethod(receiver *instance, method *parser.MethodDecl, args []Value, parent *env) (Value, error) {
	if len(args) != len(method.Parameters) {
		return nil, RuntimeError{Message: fmt.Sprintf("method '%s' expects %d args, got %d", method.Name, len(method.Parameters), len(args)), Span: method.Span}
	}
	local := newEnv(parent)
	local.define("this", receiver, false)
	for i, param := range method.Parameters {
		local.define(param.Name, args[i], false)
	}
	_, signal, err := in.execBlock(method.Body, local, receiver)
	if err != nil {
		return nil, err
	}
	if ret, ok := signal.(returnSignal); ok {
		return ret.value, nil
	}
	return nil, nil
}

func (in *Interpreter) construct(class *parser.ClassDecl, args []Value, parent *env) (Value, error) {
	obj := &instance{class: class, fields: map[string]Value{}}
	for _, field := range class.Fields {
		obj.fields[field.Name] = zeroValue(field.Type)
	}
	for _, method := range class.Methods {
		if method.Constructor {
			if _, err := in.callMethod(obj, method, args, parent); err != nil {
				return nil, err
			}
			return obj, nil
		}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: fmt.Sprintf("constructor '%s' expects 0 args, got %d", class.Name, len(args)), Span: class.Span}
	}
	return obj, nil
}

func (in *Interpreter) execBlock(block *parser.BlockStmt, parent *env, self *instance) (Value, any, error) {
	local := newEnv(parent)
	if self != nil {
		local.define("this", self, false)
	}
	for _, stmt := range block.Statements {
		value, signal, err := in.execStmt(stmt, local, self)
		if err != nil || signal != nil {
			return value, signal, err
		}
	}
	return nil, nil, nil
}

func (in *Interpreter) execStmt(stmt parser.Statement, local *env, self *instance) (Value, any, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		values := make([]Value, len(s.Values))
		for i, expr := range s.Values {
			value, err := in.evalExpr(expr, local)
			if err != nil {
				return nil, nil, err
			}
			values[i] = value
		}
		for i, binding := range s.Bindings {
			var value Value
			if i < len(values) {
				value = values[i]
			}
			local.define(binding.Name, value, binding.Mutable)
		}
		return nil, nil, nil
	case *parser.AssignmentStmt:
		value, err := in.evalExpr(s.Value, local)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, in.assign(s.Target, s.Operator, value, local)
	case *parser.IfStmt:
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
		if s.ElseIf != nil {
			return in.execStmt(s.ElseIf, local, self)
		}
		if s.Else != nil {
			return in.execBlock(s.Else, local, self)
		}
		return nil, nil, nil
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

func (in *Interpreter) execFor(stmt *parser.ForStmt, local *env, self *instance) (Value, any, error) {
	if stmt.YieldBody != nil {
		return nil, nil, RuntimeError{Message: "yield loops are not implemented in interpreter yet", Span: stmt.Span}
	}
	if len(stmt.Bindings) == 0 {
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
	if len(stmt.Bindings) != 1 {
		return nil, nil, RuntimeError{Message: "multi-binding for loops are not implemented in interpreter yet", Span: stmt.Span}
	}
	iterable, err := in.evalExpr(stmt.Bindings[0].Iterable, local)
	if err != nil {
		return nil, nil, err
	}
	items, ok := iterable.([]Value)
	if !ok {
		return nil, nil, RuntimeError{Message: "for loop expects iterable list value", Span: stmt.Bindings[0].Span}
	}
	for _, item := range items {
		loopEnv := newEnv(local)
		loopEnv.define(stmt.Bindings[0].Name, item, false)
		_, signal, err := in.execBlock(stmt.Body, loopEnv, self)
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
	return nil, nil, nil
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
		if operator != "=" {
			updated, err := applyBinary(operator[:len(operator)-1], current.value(), value, t.Span)
			if err != nil {
				return err
			}
			value = updated
		}
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
		if operator != "=" {
			updated, err := applyBinary(operator[:len(operator)-1], current, value, t.Span)
			if err != nil {
				return err
			}
			value = updated
		}
		obj.fields[t.Name] = value
		return nil
	default:
		return RuntimeError{Message: "invalid assignment target", Span: exprSpan(target)}
	}
}

func (in *Interpreter) evalExpr(expr parser.Expr, local *env) (Value, error) {
	switch e := expr.(type) {
	case *parser.Identifier:
		if binding, ok := local.get(e.Name); ok {
			return binding.value, nil
		}
		if _, ok := in.functions[e.Name]; ok {
			return e.Name, nil
		}
		if _, ok := in.classes[e.Name]; ok {
			return classRef{name: e.Name}, nil
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
	case *parser.ListLiteral:
		items := make([]Value, len(e.Elements))
		for i, item := range e.Elements {
			value, err := in.evalExpr(item, local)
			if err != nil {
				return nil, err
			}
			items[i] = value
		}
		return items, nil
	case *parser.MapLiteral:
		return map[string]Value{}, nil
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
	case *parser.CallExpr:
		return in.evalCall(e, local)
	case *parser.MemberExpr:
		receiver, err := in.evalExpr(e.Receiver, local)
		if err != nil {
			return nil, err
		}
		return in.evalMember(receiver, e)
	case *parser.LambdaExpr:
		return &closure{params: e.Parameters, body: e.Body, blockBody: e.BlockBody, env: local}, nil
	case *parser.PlaceholderExpr:
		return nil, RuntimeError{Message: "placeholder is not supported here", Span: e.Span}
	default:
		return nil, RuntimeError{Message: "unsupported expression", Span: exprSpan(expr)}
	}
}

func (in *Interpreter) evalCall(call *parser.CallExpr, local *env) (Value, error) {
	if member, ok := call.Callee.(*parser.MemberExpr); ok {
		return in.evalMethodCall(member, call.Args, local)
	}
	callee, err := in.evalExpr(call.Callee, local)
	if err != nil {
		return nil, err
	}
	args := make([]Value, len(call.Args))
	for i, arg := range call.Args {
		value, err := in.evalExpr(arg, local)
		if err != nil {
			return nil, err
		}
		args[i] = value
	}
	switch fn := callee.(type) {
	case string:
		return in.callFunctionByName(fn, args, local)
	case classRef:
		class, ok := in.classes[fn.name]
		if !ok {
			return nil, RuntimeError{Message: "undefined class '" + fn.name + "'", Span: call.Span}
		}
		return in.construct(class, args, local)
	case *closure:
		return in.callClosure(fn, args)
	default:
		return nil, RuntimeError{Message: "value is not callable", Span: call.Span}
	}
}

func (in *Interpreter) callClosure(fn *closure, args []Value) (Value, error) {
	if len(args) != len(fn.params) {
		return nil, RuntimeError{Message: fmt.Sprintf("lambda expects %d args, got %d", len(fn.params), len(args)), Span: parser.Span{}}
	}
	local := newEnv(fn.env)
	for i, param := range fn.params {
		local.define(param.Name, args[i], false)
	}
	if fn.body != nil {
		return in.evalExpr(fn.body, local)
	}
	if fn.blockBody != nil {
		_, signal, err := in.execBlock(fn.blockBody, local, nil)
		if err != nil {
			return nil, err
		}
		if ret, ok := signal.(returnSignal); ok {
			return ret.value, nil
		}
		return nil, nil
	}
	return nil, nil
}

func (in *Interpreter) evalMethodCall(member *parser.MemberExpr, argExprs []parser.Expr, local *env) (Value, error) {
	receiver, err := in.evalExpr(member.Receiver, local)
	if err != nil {
		return nil, err
	}
	obj, ok := receiver.(*instance)
	if !ok {
		return nil, RuntimeError{Message: "member call requires class instance", Span: member.Span}
	}
	args := make([]Value, len(argExprs))
	for i, arg := range argExprs {
		value, err := in.evalExpr(arg, local)
		if err != nil {
			return nil, err
		}
		args[i] = value
	}
	for _, method := range obj.class.Methods {
		if method.Name == member.Name && len(method.Parameters) == len(args) {
			return in.callMethod(obj, method, args, local)
		}
	}
	return nil, RuntimeError{Message: "unknown method '" + member.Name + "'", Span: member.Span}
}

func (in *Interpreter) evalMember(receiver Value, expr *parser.MemberExpr) (Value, error) {
	switch value := receiver.(type) {
	case *instance:
		if field, ok := value.fields[expr.Name]; ok {
			return field, nil
		}
		for _, method := range value.class.Methods {
			if method.Name == expr.Name {
				return nil, RuntimeError{Message: "method '" + expr.Name + "' must be called with ()", Span: expr.Span}
			}
		}
		return nil, RuntimeError{Message: "unknown member '" + expr.Name + "'", Span: expr.Span}
	default:
		return nil, RuntimeError{Message: "member access expects class instance", Span: expr.Span}
	}
}

type classRef struct{ name string }

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
	case "List", "Set", "Array":
		return []Value{}
	case "Map":
		return map[string]Value{}
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
