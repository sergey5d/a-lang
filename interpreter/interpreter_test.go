package interpreter

import (
	"testing"

	"a-lang/parser"
)

func parseProgram(t *testing.T, src string) *parser.Program {
	t.Helper()
	program, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return program
}

func TestCallFunction(t *testing.T) {
	src := `
def add(a Int, b Int) Int {
	return a + b
}

def run(input Int) Int {
	let total Int = add(input, 2)
	var copy Int = total
	copy += 3

	if copy > 10 {
		return copy
	}

	return total
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run", int64(5))
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(7) {
		t.Fatalf("expected 7, got %#v", value)
	}
}

func TestForAndRange(t *testing.T) {
	src := `
def run() Int {
	var total Int = 0

	for item <- [1, 2, 3] {
		total += item
	}

	for step <- range(1, 5, 2) {
		total += step
	}

	return total
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(10) {
		t.Fatalf("expected 10, got %#v", value)
	}
}

func TestClassesAndMethods(t *testing.T) {
	src := `
class Counter {
	private var count Int

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.count += 1
		return this.count
	}
}

def run() Int {
	let counter Counter = Counter(1)
	counter.inc()
	return counter.inc()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(3) {
		t.Fatalf("expected 3, got %#v", value)
	}
}
