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
	let mut count Int = 1
	let mut left Int, right Int = 1, 2
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
		t.Fatalf("expected first statement to be let binding, got %T", fn.Body.Statements[0])
	}
	if !first.Bindings[0].Mutable {
		t.Fatalf("expected first binding to be mutable")
	}

	second, ok := fn.Body.Statements[1].(*ValStmt)
	if !ok {
		t.Fatalf("expected second statement to be let binding, got %T", fn.Body.Statements[1])
	}
	if !second.Bindings[0].Mutable {
		t.Fatalf("expected first binding in second statement to be mutable")
	}
	if second.Bindings[1].Mutable {
		t.Fatalf("expected second binding in second statement to be immutable")
	}
}
