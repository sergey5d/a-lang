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
	total Int = add(input, 1)
	copy Int := total
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
	count Int = "hello"
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
	value Int = add(true)
	fixed Int = 1
	fixed := 2
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

func TestAnalyzeConstructorFieldAssignmentAllowsEquals(t *testing.T) {
	src := `
class Box {
	value Int

	def init(value Int) {
		this.value = value
	}
}

def run() Int {
	box Box = Box(2)
	return box.value
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeTupleDestructuring(t *testing.T) {
	src := `
def run() Int {
	a (value Int, size Int) = (1, 2)
	b (Int, Int) = a
	c = a
	d (left Int, right Int) = a
	x Int = a.value
	y Int = c.value
	z Int = d.left
	b.value
	return x + y + z
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "unknown_member" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeArrayIndexing(t *testing.T) {
	src := `
def run(values Array[Int]) Int {
	first Int = values[0]
	values[1] := first + 2
	return values[1]
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeInvalidIndexing(t *testing.T) {
	src := `
def fromList(values List[Int]) Int {
	return values[0]
}

def fromArray(values Array[Int]) Int {
	return values[true]
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_index_target" {
		t.Fatalf("unexpected first diagnostic %#v", result.Diagnostics[0])
	}
	if result.Diagnostics[1].Code != "invalid_index_type" {
		t.Fatalf("unexpected second diagnostic %#v", result.Diagnostics[1])
	}
}

func TestAnalyzeClassMembersAndConstructors(t *testing.T) {
	src := `
class Counter {
	private count Int := ?

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
	return counter.inc()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeImplicitPrimaryConstructorAndThisDelegation(t *testing.T) {
	src := `
class Counter {
	count Int
	label String
	private seen Bool := false

	def this(seed Int) {
		this(count = seed, label = "ok")
	}
}

def run() Int {
	left Counter = Counter(1, "x")
	right Counter = Counter(seed = 3)
	return left.count + right.count
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeRecordRejectsMutableFields(t *testing.T) {
	src := `
record Amount {
	amount Int := 1
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics for mutable record field")
	}
	if result.Diagnostics[0].Code != "invalid_record_field" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeRecordUpdateExpr(t *testing.T) {
	src := `
record Amount {
	amount Int
	description String
}

def run() Int {
	value = Amount(10, "x")
	updated = value with { amount = 42 }
	return updated.amount
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
	def show() String
}

class Good with Stringable {
	def init() {
	}

	def show() String {
		return "ok"
	}
}

class Bad with Stringable {
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

func TestAnalyzeInterfaceInheritance(t *testing.T) {
	src := `
interface Hopper {
	def hop() String
}

interface Jumper {
	def jump(steps Int) String
}

interface Acrobat with Hopper, Jumper {
}

class Rabbit with Acrobat {
	def hop() String = "hop"
	def jump(steps Int) String = "jump " + steps
}

def run() String {
	rabbit = Rabbit()
	return rabbit.hop() + " " + rabbit.jump(3)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeImmutableFieldAssignmentInInit(t *testing.T) {
	src := `
class Counter {
	private count Int

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
	private count Int

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
	private value Int

	def init(value Int) {
		this.value = value
	}

	private def reveal() Int {
		return this.value
	}
}

def run() Int {
	box SecretBox = SecretBox(7)
	x Int = box.value
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
	private value Int

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
	private count Int

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
	counter Counter = Counter(2)
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
	private count Int

	def init(count Int) {
		this.count = count
	}

	def add(value Int) Int {
		return this.count + value
	}
}

def run() Int {
	counter Counter = Counter(2)
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
	private count Int

	def init(count Int) {
		this.count = count
	}

	def add(value Int) Int {
		return this.count + value
	}
}

def run() Int {
	counter Counter = Counter(2)
	f Int = counter.add
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
	private count Int

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
	private count Int
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
	private count Int
	private seen Bool := ?

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
	add = (x Int) -> x + 1
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
	add = (x Int) -> {
		y Int = x + 1
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
	total Int := 1
	add = (x Int) -> x + total
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
	private value T

	def init(value T) {
		this.value = value
	}

	def transform() T {
		id = (item T) -> item
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
	add Int -> Int = x -> x + 1
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
def apply(value Int, f Int -> Int) Int {
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
	add = x -> x + 1
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
class Counter with Eq[Counter] {
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

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeClassEqualityRequiresEq(t *testing.T) {
	src := `
class Counter {
	private count Int

	def init(count Int) {
		this.count = count
	}
}

def run() Bool {
	left Counter = Counter(1)
	right Counter = Counter(1)
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

func TestAnalyzeBuiltinCollectionAndTermInterfaces(t *testing.T) {
	src := `
def run() Int {
	items List[Int] = List(1, 2)
	items.append(3)

	values Map[String, Int] = Map("a" : 1)
	values.set("b", 2)

	seen Set[Int] = Set(1, 2)
	if seen.contains(2) {
		Term.println("ok")
	}

	return items.get(0).getOr(0) + values.get("a").getOr(0)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeTermPrintlnAnyTypes(t *testing.T) {
	src := `
def run() Int {
	Term.println("count", 10, true)
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeOptionConstructorsAndMethods(t *testing.T) {
	src := `
def run() Int {
	found Option[Int] = Some(5)
	missing Option[Int] = None()
	if found.isSet() {
		return found.get()
	}
	return missing.getOr(7)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeStringConcatenation(t *testing.T) {
	src := `
def run() String {
	count Int = 10
	return "counter " + count
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeIfAndForYieldExpressions(t *testing.T) {
	src := `
def run(values List[Int], flag Bool) Int {
	label Int = if flag {
		1
	} else {
		2
	}
	items List[Int] = for {
		x <- values,
		y <- values
	} yield {
		x + y
	}
	items2 List[Int] = for item <- values yield {
		item + 1
	}
	return label + items.size() + items2.size()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeIsExpression(t *testing.T) {
	src := `
class Counter {
}

def run() Bool {
	value = Counter()
	return value is Counter
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeNamedCallArguments(t *testing.T) {
	src := `
class Counter {
	def set(value Int, label String) Int {
		return value
	}
}

def doSomething(a String, b Int) Int {
	return b
}

def run() Int {
	counter = Counter()
	left Int = doSomething(b = 5, a = "crap")
	right Int = counter.set(label = "ok", value = 7)
	return left + right
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeUnitLambda(t *testing.T) {
	src := `
def run() Int {
	action () -> Unit = () -> Term.println("hi")
	action()
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeUnitLiteralAndGroupedExpr(t *testing.T) {
	src := `
def run() Bool {
	unit Unit = ()
	value Int = (1)
	return value == 1
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeExplicitReturnValueInUnitFunctionFails(t *testing.T) {
	src := `
def run() Unit {
	return 1
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "invalid_return_type" {
		t.Fatalf("expected invalid_return_type, got %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeZeroArgFunctionBindingSugar(t *testing.T) {
	src := `
def run() Int {
	action () -> Int = 1
	return action()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeFunctionImplicitReturn(t *testing.T) {
	src := `
def suffix(value String) String {
	value + "!"
}

def run() String {
	suffix("hello")
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeLocalFunctionStmt(t *testing.T) {
	src := `
def run() Int {
	boost Int = 2
	def add(value Int) Int = value + boost
	add(5)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeVariadicFunction(t *testing.T) {
	src := `
def sum(values Int...) Int {
	total Int := 0
	for value <- values {
		total += value
	}
	total
}

def run() Int {
	sum(1, 2, 3)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeMultiAssignmentStmt(t *testing.T) {
	src := `
def run() Int {
	a Int := 0
	b String := ""
	a, b := 1, "ok"
	a
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeArrayConstructorAndSize(t *testing.T) {
	src := `
def run() Int {
	values Array[Int] = Array(5)
	return values.size()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeInvalidArrayConstructor(t *testing.T) {
	src := `
def run() Array[Int] {
	return Array(1, 2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "invalid_argument_count" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}
