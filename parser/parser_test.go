package parser

import "testing"

const sampleProgram = `
def doSomeWork(a Int, b Int) Bool {

	let list = [a, b, c]
	let set = Set()
	let map = {}
	let map2 = Map(a : b)
	let tuple = Set(a, b)
	let tuple2 = a : b : c
	let tuple21 = a : b : c
	let array = Array(1, 2, 3)
	let string String = "xxx"
	let a int = 65

	let a Int, b Int64 = 1, 3

	if a == b {

	} else {

	}

	for a <- list {

		if a == 7 {
			break
		}
	}

	for a <- [1..100] {

	}

	do (
		a <- list,
		c <- map
	) yield {
		a + c
	}

	a match {
		list List: list.append(5)
		5 : a + 7
		[0..100].contains(_): a * 5
	}

	ret a == 5
}
`

func TestParseSampleProgram(t *testing.T) {
	program, err := Parse(sampleProgram)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(program.Functions))
	}
	fn := program.Functions[0]
	if fn.Name != "doSomeWork" {
		t.Fatalf("unexpected function name %q", fn.Name)
	}
	if got := len(fn.Body.Statements); got != 17 {
		t.Fatalf("expected 17 statements in body, got %d", got)
	}
	if _, ok := fn.Body.Statements[len(fn.Body.Statements)-1].(*ReturnStmt); !ok {
		t.Fatalf("expected final statement to be return, got %T", fn.Body.Statements[len(fn.Body.Statements)-1])
	}
}

func TestParseExtendedOperators(t *testing.T) {
	src := `
def ops(a Int, b Int) Bool {
	let x = a - b / 2 * 3 % 5
	let y = !a == b
	let z = a != b
	let c = a < b
	let d = a <= b
	let e = a > b
	let f = a >= b
	ret a == b || a != b && !(a < b)
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(program.Functions))
	}
}

func TestParseMutableBindings(t *testing.T) {
	src := `
def vars() Bool {
	mut count Int = 1
	mut left Int, right Int = 1, 2
	ret count == right
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first, ok := fn.Body.Statements[0].(*ValStmt)
	if !ok {
		t.Fatalf("expected first statement to be binding, got %T", fn.Body.Statements[0])
	}
	if !first.Bindings[0].Mutable {
		t.Fatalf("expected first binding to be mutable")
	}

	second, ok := fn.Body.Statements[1].(*ValStmt)
	if !ok {
		t.Fatalf("expected second statement to be binding, got %T", fn.Body.Statements[1])
	}
	if !second.Bindings[0].Mutable {
		t.Fatalf("expected first binding in second statement to be mutable")
	}
	if !second.Bindings[1].Mutable {
		t.Fatalf("expected second binding in second statement to be mutable")
	}
}

func TestParseImmutableBindings(t *testing.T) {
	src := `
def vars() Bool {
	let count Int = 1
	let left Int, right Int = 1, 2
	ret count == right
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first := fn.Body.Statements[0].(*ValStmt)
	if first.Bindings[0].Mutable {
		t.Fatalf("expected let binding to be immutable")
	}

	second := fn.Body.Statements[1].(*ValStmt)
	if second.Bindings[0].Mutable || second.Bindings[1].Mutable {
		t.Fatalf("expected let bindings to be immutable")
	}
}

func TestParseUntypedBindings(t *testing.T) {
	src := `
def vars() Bool {
	let a = "some string"
	mut counter = 0
	ret counter == 0
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]

	letStmt := fn.Body.Statements[0].(*ValStmt)
	if letStmt.Bindings[0].Type != "" {
		t.Fatalf("expected untyped let binding, got type %q", letStmt.Bindings[0].Type)
	}
	if letStmt.Bindings[0].Mutable {
		t.Fatalf("expected let binding to be immutable")
	}

	mutStmt := fn.Body.Statements[1].(*ValStmt)
	if mutStmt.Bindings[0].Type != "" {
		t.Fatalf("expected untyped mut binding, got type %q", mutStmt.Bindings[0].Type)
	}
	if !mutStmt.Bindings[0].Mutable {
		t.Fatalf("expected mut binding to be mutable")
	}
}

func TestParseFunctionInvocationBinding(t *testing.T) {
	src := `
def vars(b Int) Bool {
	let a = function(b)
	ret a == b
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	stmt := fn.Body.Statements[0].(*ValStmt)
	call, ok := stmt.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected binding value to be call expression, got %T", stmt.Values[0])
	}
	callee, ok := call.Callee.(*Identifier)
	if !ok || callee.Name != "function" {
		t.Fatalf("expected callee to be identifier 'function', got %#v", call.Callee)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 call argument, got %d", len(call.Args))
	}
}

func TestParseLambdaBindings(t *testing.T) {
	src := `
def vars() Bool {
	let a = Map(1 : "string").map((key, value) -> key.toString() + value)
	let b = Set(1).map(key -> key.toString())
	let c = Map(1 : 2).map((key Int, value Int) -> key + value)
	let d = Set(1).map(key Int -> key.toString())
	ret 1 == 1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]

	first := fn.Body.Statements[0].(*ValStmt)
	firstCall, ok := first.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected first binding value to be call expression, got %T", first.Values[0])
	}
	firstLambda, ok := firstCall.Args[0].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected map argument to be lambda, got %T", firstCall.Args[0])
	}
	if len(firstLambda.Parameters) != 2 {
		t.Fatalf("expected 2 lambda parameters, got %d", len(firstLambda.Parameters))
	}

	second := fn.Body.Statements[1].(*ValStmt)
	secondCall, ok := second.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected second binding value to be call expression, got %T", second.Values[0])
	}
	secondLambda, ok := secondCall.Args[0].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected map argument to be lambda, got %T", secondCall.Args[0])
	}
	if len(secondLambda.Parameters) != 1 || secondLambda.Parameters[0].Name != "key" || secondLambda.Parameters[0].Type != "" {
		t.Fatalf("unexpected single-parameter lambda: %#v", secondLambda.Parameters)
	}

	third := fn.Body.Statements[2].(*ValStmt)
	thirdCall, ok := third.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected third binding value to be call expression, got %T", third.Values[0])
	}
	thirdLambda, ok := thirdCall.Args[0].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected typed map argument to be lambda, got %T", thirdCall.Args[0])
	}
	if thirdLambda.Parameters[0].Type != "Int" || thirdLambda.Parameters[1].Type != "Int" {
		t.Fatalf("expected typed lambda parameters, got %#v", thirdLambda.Parameters)
	}

	fourth := fn.Body.Statements[3].(*ValStmt)
	fourthCall, ok := fourth.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected fourth binding value to be call expression, got %T", fourth.Values[0])
	}
	fourthLambda, ok := fourthCall.Args[0].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected typed single-parameter lambda, got %T", fourthCall.Args[0])
	}
	if len(fourthLambda.Parameters) != 1 || fourthLambda.Parameters[0].Name != "key" || fourthLambda.Parameters[0].Type != "Int" {
		t.Fatalf("unexpected typed single-parameter lambda: %#v", fourthLambda.Parameters)
	}
}

func TestParseElseIf(t *testing.T) {
	src := `
def classify(a Int) Bool {
	if a == 1 {
		ret a == 1
	} else if a == 2 {
		ret a == 2
	} else {
		ret a == 3
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	ifStmt, ok := fn.Body.Statements[0].(*IfStmt)
	if !ok {
		t.Fatalf("expected first statement to be if, got %T", fn.Body.Statements[0])
	}
	if ifStmt.ElseIf == nil {
		t.Fatalf("expected else-if branch to be present")
	}
	if ifStmt.ElseIf.Else == nil {
		t.Fatalf("expected final else block on else-if chain")
	}
}

func TestAttachSourceSpans(t *testing.T) {
	src := `
def sample(a Int) Bool {
	let value = function(a)
	ret value == a
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if program.Span.Start.Line == 0 || program.Span.End.Line == 0 {
		t.Fatalf("expected program span to be populated, got %#v", program.Span)
	}

	fn := program.Functions[0]
	if fn.Span.Start.Line != 2 {
		t.Fatalf("expected function to start on line 2, got %#v", fn.Span)
	}

	stmt := fn.Body.Statements[0].(*ValStmt)
	if stmt.Span.Start.Line != 3 {
		t.Fatalf("expected let statement to start on line 3, got %#v", stmt.Span)
	}

	call := stmt.Values[0].(*CallExpr)
	if call.Span.Start.Line != 3 || call.Span.End.Line != 3 {
		t.Fatalf("expected call span on line 3, got %#v", call.Span)
	}

	retStmt := fn.Body.Statements[1].(*ReturnStmt)
	if retStmt.Span.Start.Line != 4 {
		t.Fatalf("expected return statement span on line 4, got %#v", retStmt.Span)
	}
}
