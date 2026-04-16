package typecheck

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

func TestAnalyzeValidProgram(t *testing.T) {
	src := `
def add(a Int, b Int) Int {
	return a + b
}

def run(input Int) Bool {
	let total Int = add(input, 1)
	var copy Int = total
	copy += 1

	if copy > input {
		return true
	}

	return false
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeTypeMismatch(t *testing.T) {
	src := `
def run() Bool {
	let count Int = "hello"
	return true
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "type_mismatch" {
		t.Fatalf("expected type_mismatch diagnostic, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeInvalidConditionAndReturn(t *testing.T) {
	src := `
def run() Bool {
	if 1 {
		return 7
	}

	return false
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_condition_type" {
		t.Fatalf("unexpected first diagnostic %#v", result.Diagnostics[0])
	}
	if result.Diagnostics[1].Code != "invalid_return_type" {
		t.Fatalf("unexpected second diagnostic %#v", result.Diagnostics[1])
	}
}

func TestAnalyzeInvalidCallAndAssignment(t *testing.T) {
	src := `
def add(a Int, b Int) Int {
	return a + b
}

def run() Int {
	let value Int = add(true)
	let fixed Int = 1
	fixed = 2
	return value
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 3 {
		t.Fatalf("expected 3 diagnostics, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_argument_count" {
		t.Fatalf("unexpected first diagnostic %#v", result.Diagnostics[0])
	}
	if result.Diagnostics[1].Code != "invalid_argument_type" {
		t.Fatalf("unexpected second diagnostic %#v", result.Diagnostics[1])
	}
	if result.Diagnostics[2].Code != "assign_immutable" {
		t.Fatalf("unexpected third diagnostic %#v", result.Diagnostics[2])
	}
}

func TestAnalyzeClassMembersAndConstructors(t *testing.T) {
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
	return counter.inc()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeInterfaceImplementation(t *testing.T) {
	src := `
interface Stringable {
	def toString() String
}

class Good implements Stringable {
	def init() {
	}

	def toString() String {
		return "ok"
	}
}

class Bad implements Stringable {
	def init() {
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "interface_not_implemented" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeImmutableFieldAssignmentInInit(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def read() Int {
		return this.count
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeImmutableFieldAssignmentOutsideInit(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def bump() Int {
		this.count = this.count + 1
		return this.count
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "assign_immutable" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzePrivateAccessOutsideClass(t *testing.T) {
	src := `
class SecretBox {
	private let value Int

	def init(value Int) {
		this.value = value
	}

	private def reveal() Int {
		return this.value
	}
}

def run() Int {
	let box SecretBox = SecretBox(7)
	let x Int = box.value
	return box.reveal()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "private_access" {
		t.Fatalf("unexpected first diagnostic %#v", result.Diagnostics[0])
	}
	if result.Diagnostics[1].Code != "private_access" {
		t.Fatalf("unexpected second diagnostic %#v", result.Diagnostics[1])
	}
}

func TestAnalyzePrivateAccessInsideClass(t *testing.T) {
	src := `
class SecretBox {
	private let value Int

	def init(value Int) {
		this.value = value
	}

	private def reveal() Int {
		return this.value
	}

	def expose() Int {
		return this.reveal()
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}
