package typed

import (
	"testing"

	"a-lang/parser"
	"a-lang/typecheck"
)

func parseProgram(t *testing.T, src string) *parser.Program {
	t.Helper()
	program, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return program
}

func TestBuildTypedProgram(t *testing.T) {
	src := `
class Counter {
	private count Int
	private ticks Int := 0

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.ticks += 1
		return this.count + this.ticks
	}
}

seed Int = 1

def run(input Int) Int {
	counter Counter = Counter(input)
	return counter.inc()
}
`

	program := parseProgram(t, src)
	types := typecheck.Analyze(program)
	if len(types.Diagnostics) != 0 {
		t.Fatalf("expected no type diagnostics, got %#v", types.Diagnostics)
	}

	typedProgram, err := Build(program, types)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if len(typedProgram.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(typedProgram.Classes))
	}
	if len(typedProgram.Globals) != 1 {
		t.Fatalf("expected 1 global, got %d", len(typedProgram.Globals))
	}
	counter := typedProgram.Classes[0]
	if counter.Fields[0].Mode != BindingImmutable || counter.Fields[0].InitMode != InitDeferred {
		t.Fatalf("expected immutable deferred field, got %#v", counter.Fields[0])
	}
	if counter.Fields[1].Mode != BindingMutable || counter.Fields[1].InitMode != InitImmediate {
		t.Fatalf("expected mutable initialized field, got %#v", counter.Fields[1])
	}
	if counter.Fields[1].Init == nil {
		t.Fatalf("expected mutable field initializer")
	}

	global, ok := typedProgram.Globals[0].(*BindingStmt)
	if !ok || global.Bindings[0].Name != "seed" {
		t.Fatalf("unexpected globals %#v", typedProgram.Globals)
	}

	run := typedProgram.Functions[0]
	first, ok := run.Body.Statements[0].(*BindingStmt)
	if !ok {
		t.Fatalf("expected first statement to be binding, got %T", run.Body.Statements[0])
	}
	call, ok := first.Bindings[0].Init.(*ConstructorCallExpr)
	if !ok || call.Class != "Counter" {
		t.Fatalf("expected constructor call, got %#v", first.Bindings[0].Init)
	}

	ret, ok := run.Body.Statements[1].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %T", run.Body.Statements[1])
	}
	methodCall, ok := ret.Value.(*MethodCallExpr)
	if !ok || methodCall.Method != "inc" {
		t.Fatalf("expected method call, got %#v", ret.Value)
	}
}

func TestBuildTypedLambdaAndInvoke(t *testing.T) {
	src := `
def run() Int {
	add Int -> Int = x -> x + 1
	return add(2)
}
`

	program := parseProgram(t, src)
	types := typecheck.Analyze(program)
	if len(types.Diagnostics) != 0 {
		t.Fatalf("expected no type diagnostics, got %#v", types.Diagnostics)
	}

	typedProgram, err := Build(program, types)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	run := typedProgram.Functions[0]
	stmt := run.Body.Statements[0].(*BindingStmt)
	lambda, ok := stmt.Bindings[0].Init.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected lambda initializer, got %T", stmt.Bindings[0].Init)
	}
	if lambda.GetType() == nil || lambda.GetType().Kind != typecheck.TypeFunction {
		t.Fatalf("expected lambda function type, got %#v", lambda.GetType())
	}

	ret := run.Body.Statements[1].(*ReturnStmt)
	invoke, ok := ret.Value.(*InvokeExpr)
	if !ok {
		t.Fatalf("expected invoke expression, got %T", ret.Value)
	}
	if _, ok := invoke.Callee.(*IdentifierExpr); !ok {
		t.Fatalf("expected invoke callee identifier, got %T", invoke.Callee)
	}
}
