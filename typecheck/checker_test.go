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

func TestAnalyzeMethodOverloadResolution(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def value() Int {
		return this.count
	}

	def add(value Int) Int {
		return this.count + value
	}

	def add(left Int, right Int) Int {
		return left + right
	}
}

def run() Int {
	let counter Counter = Counter(2)
	return counter.add(1)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeNoMatchingMethodOverload(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def add(value Int) Int {
		return this.count + value
	}
}

def run() Int {
	let counter Counter = Counter(2)
	return counter.add(true)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "no_matching_overload" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeMethodReferenceWithoutCall(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def add(value Int) Int {
		return this.count + value
	}
}

def run() Int {
	let counter Counter = Counter(2)
	let f Int = counter.add
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_member_access" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeDuplicateConstructorOverload(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def init(value Int) {
		this.count = value
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected duplicate constructor diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "duplicate_constructor" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeImplicitConstructorRequiresMutableOnlyFields(t *testing.T) {
	src := `
class Counter {
	private let count Int
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "constructor_required" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeConstructorMustInitializeImmutableFields(t *testing.T) {
	src := `
class Counter {
	private let count Int
	private var seen Bool

	def init() {
		this.seen = false
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "uninitialized_field" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeLambdaFunctionValue(t *testing.T) {
	src := `
def run() Int {
	let add = (x Int) -> x + 1
	return add(2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeBlockLambdaFunctionValue(t *testing.T) {
	src := `
def run() Int {
	let add = (x Int) -> {
		let y Int = x + 1
		return y
	}
	return add(2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeLambdaCannotCaptureVar(t *testing.T) {
	src := `
def run() Int {
	var total Int = 1
	let add = (x Int) -> x + total
	return add(2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_lambda_capture" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeLambdaCanUseEnclosingGenericType(t *testing.T) {
	src := `
class Box[T] {
	private let value T

	def init(value T) {
		this.value = value
	}

	def transform() T {
		let id = (item T) -> item
		return id(this.value)
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeFunctionTypeBindingAndContextualLambda(t *testing.T) {
	src := `
def run() Int {
	let add (Int) -> Int = x -> x + 1
	return add(2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeContextualLambdaInFunctionCall(t *testing.T) {
	src := `
def apply(value Int, f (Int) -> Int) Int {
	return f(value)
}

def run() Int {
	return apply(2, x -> x + 1)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeUntypedLambdaWithoutContextFails(t *testing.T) {
	src := `
def run() Int {
	let add = x -> x + 1
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_lambda_type" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeEqSupportsClassEquality(t *testing.T) {
	src := `
class Counter implements Eq[Counter] {
	private let count Int

	def init(count Int) {
		this.count = count
	}

	def equals(other Counter) Bool {
		return this.count == other.count
	}
}

def run() Bool {
	let left Counter = Counter(1)
	let right Counter = Counter(1)
	return left == right
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeClassEqualityRequiresEq(t *testing.T) {
	src := `
class Counter {
	private let count Int

	def init(count Int) {
		this.count = count
	}
}

def run() Bool {
	let left Counter = Counter(1)
	let right Counter = Counter(1)
	return left == right
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_binary_operand" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}
