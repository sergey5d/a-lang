package semantic

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

func TestAnalyzeValidScopes(t *testing.T) {
	src := `
def run(input Int) Bool {
	value = helper(input)
	acc := 0
	acc := acc + 1
	item = input

	for item <- [1, 2, 3] {
		if item == input {
			break
		}
	}

	for item <- [1, 3] {
		if item == input {
			break
		}
	}

	for {
		if acc == input {
			break
		}
	}

	for {
		x <- [value],
		y <- [input]
	} yield {
		x + y
	}

	mapper = (x Int) -> x + value
	return mapper(input) == helper(value)
}

def helper(x Int) Int {
	return x
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestAnalyzeUndefinedName(t *testing.T) {
	src := `
def run() Bool {
	return missing == 1
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "undefined_name" {
		t.Fatalf("unexpected diagnostic code %#v", diagnostics[0])
	}
}

func TestAnalyzeDuplicateBinding(t *testing.T) {
	src := `
def run() Bool {
	value = 1
	value Int = 2
	return value == 2
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics for duplicate binding, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "duplicate_binding" {
		t.Fatalf("unexpected first diagnostic %#v", diagnostics[0])
	}
}

func TestAnalyzeDuplicateParameter(t *testing.T) {
	src := `
def run(value Int, value Int) Bool {
	return value == 1
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics for duplicate parameter, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "duplicate_parameter" {
		t.Fatalf("unexpected first diagnostic %#v", diagnostics[0])
	}
}

func TestAnalyzeBreakOutsideLoop(t *testing.T) {
	src := `
def run() Bool {
	break
	return 1 == 1
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "invalid_break" {
		t.Fatalf("unexpected diagnostic %#v", diagnostics[0])
	}
}

func TestAnalyzeDuplicateFunction(t *testing.T) {
	src := `
def run() Bool {
	return 1 == 1
}

def run() Bool {
	return 1 == 1
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics for duplicate function, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "duplicate_function" {
		t.Fatalf("unexpected first diagnostic %#v", diagnostics[0])
	}
}

func TestAnalyzeAssignmentToImmutableBinding(t *testing.T) {
	src := `
def run() Bool {
	value = 1
	value = 2
	return value == 2
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "duplicate_binding" {
		t.Fatalf("unexpected diagnostic %#v", diagnostics[0])
	}
}

func TestAnalyzeAssignmentToMutableBinding(t *testing.T) {
	src := `
def run() Bool {
	value := 1
	value := value + 1
	value += 1
	value -= 1
	value *= 2
	value /= 2
	value %= 2
	return value == 2
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestAnalyzeTopLevelGlobalBinding(t *testing.T) {
	src := `
seed Int = 1

def run() Int {
	return seed
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestAnalyzeClassesAndInterfaces(t *testing.T) {
	src := `
interface Mapper[K, V] {
	def map(value K) V
}

interface Stringable {
	def show() String
}

class Box[T] with Mapper[T, Stringable] {
	private value T

	def init(value T) {
		this.value = value
	}

	def map(value T) Stringable {
		return this
	}
}

class SolidWork with Stringable {
	private a List[Int]
	private b Map[String, Bool] := deferred

	def init(a Int, b Bool) {
		this.a = a
		this.b = b
	}

	def show() String {
		return this.buildLabel()
	}

	private def buildLabel() String {
		return this.a.show()
	}
}

solidWork = SolidWork(1, false)
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestAnalyzeGenericTypes(t *testing.T) {
	src := `
class Store[T] {
	values List[T]
}

def useStore(input Map[String, List[Int]]) List[Map[String, Int]] {
	store Store[Int] = Store(input)
	bad Unknown[Int] = store
	wrong List[Int, String] = []
	return [Map()]
}
`

	diagnostics := Analyze(parseProgram(t, src))
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", diagnostics)
	}
	if diagnostics[0].Code != "undefined_type" {
		t.Fatalf("unexpected first diagnostic %#v", diagnostics[0])
	}
	if diagnostics[1].Code != "invalid_type_arity" {
		t.Fatalf("unexpected second diagnostic %#v", diagnostics[1])
	}
}
