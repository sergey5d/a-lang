package interpreter

import (
	"io"
	"os"
	"strings"
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
	total Int = add(input, 2)
	copy Int := total
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

func TestForLoops(t *testing.T) {
	src := `
def run() Int {
	total Int := 0

	for item <- [1, 2, 3] {
		total += item
	}

	for step <- [1, 3] {
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

func TestYieldLoops(t *testing.T) {
	src := `
def run() Int {
	total Int := 0

	for {
		left <- [1, 2],
		right <- [3, 4]
	} yield {
		total += left + right
		left + right
	}

	return total
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(20) {
		t.Fatalf("expected 20, got %#v", value)
	}
}

func TestIndexing(t *testing.T) {
	src := `
def run() Int {
	values = [1, 2, 3]
	values[1] := values[0] + 4
	return values[1]
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(5) {
		t.Fatalf("expected 5, got %#v", value)
	}
}

func TestClassesAndMethods(t *testing.T) {
	src := `
class Counter {
	private count Int := deferred

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.count += 1
		return this.count
	}
}

def run() Int {
	counter Counter = Counter(1)
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

func TestMethodOverloadDispatch(t *testing.T) {
	src := `
class Adder {
	def add(value Int) Int {
		return value + 1
	}

	def add(value String) Int {
		return 99
	}
}

def run() Int {
	adder Adder = Adder()
	return adder.add("hehe")
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(99) {
		t.Fatalf("expected 99, got %#v", value)
	}
}

func TestMethodWithoutReturnTypeDoesNotImplicitlyReturn(t *testing.T) {
	src := `
class Counter {
	private count Int := deferred

	def init(count Int) {
		this.count = count
	}

	def touch() {
		this.count + 1
	}
}

def run() Bool {
	counter Counter = Counter(1)
	return counter.touch() == 2
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != false {
		t.Fatalf("expected false, got %#v", value)
	}
}

func TestExpressionBodiedMethod(t *testing.T) {
	src := `
class Counter {
	def value() Int = 7
}

def run() Int {
	counter Counter = Counter()
	counter.value()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(7) {
		t.Fatalf("expected 7, got %#v", value)
	}
}

func TestFunctionImplicitReturn(t *testing.T) {
	src := `
def suffix(value String) String {
	value + "!"
}

def run() String {
	suffix("hello")
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "hello!" {
		t.Fatalf("expected hello!, got %#v", value)
	}
}

func TestFunctionWithoutReturnTypeDoesNotImplicitlyReturn(t *testing.T) {
	src := `
def suffix(value String) {
	value + "!"
}

def run() Bool {
	return suffix("hello") == "hello!"
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != false {
		t.Fatalf("expected false, got %#v", value)
	}
}

func TestLocalFunctionStmt(t *testing.T) {
	src := `
def run() Int {
	boost Int = 2
	def add(value Int) Int = value + boost
	add(5)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(7) {
		t.Fatalf("expected 7, got %#v", value)
	}
}

func TestVariadicFunctionAndMethod(t *testing.T) {
	src := `
class Printer {
	def count(values String...) Int = values.size()
}

def sum(values Int...) Int {
	total Int := 0
	for value <- values {
		total += value
	}
	total
}

def run() Int {
	printer Printer = Printer()
	return sum(1, 2, 3) + printer.count("a", "b")
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(8) {
		t.Fatalf("expected 8, got %#v", value)
	}
}

func TestMultiAssignmentStmt(t *testing.T) {
	src := `
def run() Int {
	a Int := 1
	b Int := 2
	a, b := b, a + b
	a + b
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(5) {
		t.Fatalf("expected 5, got %#v", value)
	}
}

func TestTupleDestructuring(t *testing.T) {
	src := `
def run() Int {
	a (value Int, size Int) = (1, 2)
	b (Int, Int) = a
	c = a
	d (left Int, right Int) = a
	a.value + c.value + d.left + d.right
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(5) {
		t.Fatalf("expected 5, got %#v", value)
	}
}

func TestMethodReferenceRequiresCall(t *testing.T) {
	src := `
class Counter {
	private count Int := deferred

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.count += 1
		return this.count
	}
}

def run() Int {
	counter Counter = Counter(1)
	bad = counter.inc
	return 0
}
`

	in := New(parseProgram(t, src))
	_, err := in.Call("run")
	if err == nil {
		t.Fatalf("expected runtime error")
	}
	if !strings.Contains(err.Error(), "must be called with ()") {
		t.Fatalf("unexpected runtime error: %v", err)
	}
}

func TestLambdaFunctionValue(t *testing.T) {
	src := `
def run() Int {
	add = (x Int) -> x + 1
	return add(2)
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

func TestBlockLambdaFunctionValue(t *testing.T) {
	src := `
def run() Int {
	add = (x Int) -> {
		y Int = x + 1
		return y
	}
	return add(2)
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

func TestClassEqualityUsesEquals(t *testing.T) {
	src := `
class Counter {
	private count Int

	def init(count Int) {
		this.count = count
	}

	def equals(other Counter) Bool {
		return this.count == other.count
	}
}

def run() Bool {
	left Counter = Counter(1)
	right Counter = Counter(1)
	return left == right
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != true {
		t.Fatalf("expected true, got %#v", value)
	}
}

func TestBuiltinCollectionsAndTerm(t *testing.T) {
	src := `
def run() Int {
	items = List(1, 2)
	items.append(3)

	values = Map("a" : 1)
	values.set("b", 2)

	seen = Set(1, 2)
	if seen.contains(2) {
		Term.println("ok", "done")
	}

	return items.get(0).getOr(0) + values.get("a").getOr(0) + values.size() + seen.size()
}
`

	in := New(parseProgram(t, src))

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}
	os.Stdout = writer

	value, callErr := in.Call("run")

	_ = writer.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(reader)
	_ = reader.Close()

	if callErr != nil {
		t.Fatalf("Call returned error: %v", callErr)
	}
	if value != int64(6) {
		t.Fatalf("expected 6, got %#v", value)
	}
	if strings.TrimSpace(string(output)) != "ok done" {
		t.Fatalf("expected Term output 'ok', got %q", string(output))
	}
}

func TestOptionRuntime(t *testing.T) {
	src := `
def run() Int {
	found = Some(5)
	missing = None()
	if found.isSet() {
		return found.get() + missing.getOr(2)
	}
	return 0
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(7) {
		t.Fatalf("expected 7, got %#v", value)
	}
}

func TestStringConcatenation(t *testing.T) {
	src := `
def run() String {
	count Int = 10
	return "counter " + count
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "counter 10" {
		t.Fatalf("expected %q, got %#v", "counter 10", value)
	}
}

func TestIfAndForYieldExpressions(t *testing.T) {
	src := `
def run() Int {
	values = [1, 2, 3]
	label Int = if true {
		10
	} else {
		20
	}
	items = for {
		x <- values,
		y <- values
	} yield {
		x + y
	}
	Term.println("items " + items.size())
	return label + items.size()
}
`

	in := New(parseProgram(t, src))

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}
	os.Stdout = writer

	value, callErr := in.Call("run")

	_ = writer.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(reader)
	_ = reader.Close()

	if callErr != nil {
		t.Fatalf("Call returned error: %v", callErr)
	}
	if value != int64(19) {
		t.Fatalf("expected 19, got %#v", value)
	}
	if strings.TrimSpace(string(output)) != "items 9" {
		t.Fatalf("expected output %q, got %q", "items 9", string(output))
	}
}

func TestArrayConstructorAndSize(t *testing.T) {
	src := `
def run() Int {
	values = Array(5)
	values[1] := 7
	return values.size()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(5) {
		t.Fatalf("expected 5, got %#v", value)
	}
}
