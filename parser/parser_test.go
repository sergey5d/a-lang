package parser

import "testing"

const sampleProgram = `
def doSomeWork(a Int, b Int) Bool {

	val list = [a, b, c]
	val set = Set()
	val map = {}
	val map2 = Map(a : b)
	val tuple = Set(a, b)
	val tuple2 = a -> b -> c
	val tuple21 = a : b : c
	val array = Array(1, 2, 3)
	val string String = "xxx"
	val a int = 65

	val a Int, b Int64 = 1, 3

	if a == b {

	} else {

	}

	for a from list {

		if a == 7 {
			break
		}
	}

	for a from [1..100] {

	}

	do (
		a from list,
		c from map
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
	val x = a - b / 2 * 3 % 5
	val y = !a == b
	val z = a != b
	val c = a < b
	val d = a <= b
	val e = a > b
	val f = a >= b
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
