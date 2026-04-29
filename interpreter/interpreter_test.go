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

func TestListArrayAndRangeZipMethods(t *testing.T) {
	src := `
def run() Int {
	items = List(1, 2, 3)
	pairs = items.zip(List("a", "b"))
	indexed = items.zipWithIndex()

	values = Array(3)
	values[0] := 4
	values[1] := 5
	values[2] := 6
	other = Array(2)
	other[0] := "x"
	other[1] := "y"
	valuePairs = values.zip(other)
	valueIndexed = values.zipWithIndex()

	firstLeft, firstRight = pairs.get(0).get()
	indexedValue, indexedPos = indexed.get(2).get()
	arrayLeft, arrayRight = valuePairs[1]
	arrayIndexedValue, arrayIndexedPos = valueIndexed[2]
	total := 0

	for left, right <- pairs {
		if right == "b" {
			total += left
		}
	}
	for value, index <- indexed {
		total += value + index
	}
	for left, right <- valuePairs {
		if right == "y" {
			total += left
		}
	}
	for value, index <- valueIndexed {
		if index == 2 {
			total += value
		}
	}

	if firstRight == "a" && arrayRight == "y" {
		return firstLeft + indexedValue + indexedPos + arrayLeft + arrayIndexedValue + arrayIndexedPos + pairs.size() + valuePairs.size() + total
	}
	return 0
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(45) {
		t.Fatalf("expected 45, got %#v", value)
	}
}

func TestClassesAndMethods(t *testing.T) {
	src := `
class Counter {
	private count Int := ?

	def this(count Int) {
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

	def add(value Str) Int {
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
	private count Int := ?

	def this(count Int) {
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

func TestForDestructuring(t *testing.T) {
	src := `
def run() Int {
	total Int := 0

	for left Int, right Str <- [(1, "x"), (2, "y"), (3, "x")] {
		if right == "x" {
			total += left
		}
	}

	return total
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(4) {
		t.Fatalf("expected 4, got %#v", value)
	}
}

func TestMatchStmt(t *testing.T) {
	src := `
enum OptionX[T] {
	case NoneX
	case SomeX {
		value T
	}
}

def run() Int {
	value OptionX[Int] = OptionX.SomeX(7)
	total Int := 0

	match value {
		SomeX(item) => {
			total += item
		}
		OptionX.NoneX => {
			total += 100
		}
	}

	pair = (1, 2)
	match pair {
		(left, right) => {
			total += left + right
		}
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

func TestMatchTypePattern(t *testing.T) {
	src := `
interface WorkerLike {
	def doWork() Int
}

class Worker with WorkerLike {
	impl def doWork() Int = 7
}

class Other with WorkerLike {
	impl def doWork() Int = 3
}

def run() Int {
	value WorkerLike = Worker()

	return match value {
		worker Worker => worker.doWork()
		_ Other => 100
		_ => 0
	}
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

func TestMatchClassExtractor(t *testing.T) {
	src := `
class PairBox {
	left Int
	right Int
}

def run() Int {
	value PairBox = PairBox(4, 9)
	return match value {
		PairBox(left, right) => left + right
	}
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(13) {
		t.Fatalf("expected 13, got %#v", value)
	}
}

func TestMatchRecordExtractor(t *testing.T) {
	src := `
record Amount {
	count Int
	label Str
}

def run() Int {
	value Amount = Amount(42, "hello")
	return match value {
		Amount(count, label) => count
	}
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(42) {
		t.Fatalf("expected 42, got %#v", value)
	}
}

func TestFunctionImplicitReturn(t *testing.T) {
	src := `
def suffix(value Str) Str {
	value + "!"
}

def run() Str {
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

func TestUnitLambda(t *testing.T) {
	src := `
def run() Int {
	action () -> Unit = () -> Term.println("hi")
	action()
	return 0
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(0) {
		t.Fatalf("expected 0, got %#v", value)
	}
}

func TestExplicitUnitFunction(t *testing.T) {
	src := `
def log() Unit = "hello!"

def run() Int {
	log()
	return 0
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(0) {
		t.Fatalf("expected 0, got %#v", value)
	}
}

func TestUnitLiteralAndGroupedExpr(t *testing.T) {
	src := `
def run() Bool {
	unit Unit = ()
	value Int = (1)
	return value == 1
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

func TestZeroArgFunctionBindingSugar(t *testing.T) {
	src := `
def run() Int {
	action () -> Int = 1
	return action()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(1) {
		t.Fatalf("expected 1, got %#v", value)
	}
}

func TestFunctionWithoutReturnTypeDoesNotImplicitlyReturn(t *testing.T) {
	src := `
def suffix(value Str) {
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
	def count(values Str...) Int = values.size()
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

func TestNamedTupleAccessAfterMethodReturn(t *testing.T) {
	src := `
class Counter {
	def pair() (value Int, size Int) = (2, 3)
}

def run() Int {
	counter = Counter()
	pair = counter.pair()
	return pair.value + pair.size
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
	private count Int := ?

	def this(count Int) {
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

	def this(count Int) {
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
object Ascending with Ordering[Int] {
	impl def compare(left Int, right Int) Int = left - right
}

def run() Int {
	items = List(1, 2)
	items.append(3)
	items.sort(Ascending)

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

func TestListSortWithOrdering(t *testing.T) {
	src := `
object Descending with Ordering[Int] {
	impl def compare(left Int, right Int) Int = right - left
}

def run() Int {
	items = List(3, 1, 4, 2)
	items.sort(Descending)
	return items.get(0).getOr(0) * 1000 + items.get(1).getOr(0) * 100 + items.get(2).getOr(0) * 10 + items.get(3).getOr(0)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(4321) {
		t.Fatalf("expected 4321, got %#v", value)
	}
}

func TestCustomAndCollectionOperators(t *testing.T) {
	src := `
class Vec {
	private items Array[Int] := ?

	def this(left Int, right Int) {
		this.items := Array(2)
		this.items[0] := left
		this.items[1] := right
	}

	def [](index Int) Int = items[index]
	def +(other Vec) Vec = Vec(this[0] + other[0], this[1] + other[1])
	def -() Vec = Vec(-this[0], -this[1])
	def :-(other Vec) Vec = Vec(this[0] - other[0], this[1] - other[1])
	def --(other Vec) Vec = Vec(this[0] - other[0] - 1, this[1] - other[1] - 1)
}

def run() Int {
	left Vec = Vec(1, 2)
	right Vec = Vec(3, 4)
	total Vec = left + right
	neg Vec = -total
	diff Vec = total :- left
	trimmed Vec = total -- left

	items = List(1, 2)
	items2 = items :+ 3
	merged = items2 ++ List(4, 5)

	seen = Set(1, 2)
	all = seen ++ Set(3)

	return neg[0] + diff[0] + trimmed[1] + merged[4] + all.size()
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

func TestListMapFlatMapForEach(t *testing.T) {
	src := `
def run() Int {
	items = List(1, 2, 3)
	doubled = items.map((item Int) -> item * 2)
	doubled2 = items.map(item -> item * 2)
	expanded = items.flatMap((item Int) -> List(item, item + 10))
	doubled.forEach((item Int) -> Term.println("item " + item))
	return doubled.get(2).getOr(0) * 100 + doubled2.get(1).getOr(0) * 10 + expanded.get(5).getOr(0)
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
	if value != int64(653) {
		t.Fatalf("expected 653, got %#v", value)
	}
	if strings.TrimSpace(string(output)) != "item 2\nitem 4\nitem 6" {
		t.Fatalf("unexpected output %q", string(output))
	}
}

func TestSetAndMapHigherOrderMethods(t *testing.T) {
	src := `
def run() Int {
	seen = Set(1, 2, 3)
	doubled = seen.map((item Int) -> item * 2)
	expanded = seen.flatMap((item Int) -> Set(item, item + 10))
	filtered = seen.filter((item Int) -> item > 1)
	setTotal = seen.fold(0, (acc Int, item Int) -> acc + item)
	setReduced = seen.reduce((left Int, right Int) -> left + right)
	setHasBig = seen.exists((item Int) -> item > 2)
	setAllPositive = seen.forAll((item Int) -> item > 0)
	seen.forEach((item Int) -> Term.println("set " + item))

	values = Map("a" : 1, "b" : 2)
	mapped = values.map((key Str, value Int) -> value * 10)
	expandedValues = values.flatMap((key Str, value Int) -> List(value, value + 10))
	filteredMap = values.filter((key Str, value Int) -> value > 1)
	mapTotal = values.fold(0, (acc Int, key Str, value Int) -> acc + value)
	mapReduced = values.reduce((leftKey Str, leftValue Int, rightKey Str, rightValue Int) -> (rightKey, rightValue))
	mapHasB = values.exists((key Str, value Int) -> key == "b")
	mapAllSmall = values.forAll((key Str, value Int) -> value < 3)
	values.forEach((key Str, value Int) -> Term.println("pair " + key + " " + value))

	total := 0
	for item Int <- seen {
		total += item
	}
	for key Str, value Int <- values {
		total += value
	}

	reducedKey, reducedValue = mapReduced.get()
	if expanded.contains(12) && setHasBig && setAllPositive && mapHasB && mapAllSmall {
		if reducedKey == "b" {
			return total * 1000000 + mapped.get(0).getOr(0) * 100000 + expandedValues.get(3).getOr(0) * 10000 + doubled.size() * 1000 + filtered.size() * 100 + setTotal * 10 + setReduced.getOr(0) + filteredMap.size() + mapTotal + reducedValue
		}
	}
	return 0
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
	if value != int64(10123272) {
		t.Fatalf("expected 10123272, got %#v", value)
	}
	if strings.TrimSpace(string(output)) != "set 1\nset 2\nset 3\npair a 1\npair b 2" {
		t.Fatalf("unexpected output %q", string(output))
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

func TestBuiltinRuntimeUsesPredefMethodSurface(t *testing.T) {
	src := `
def run() Int {
	items = List(10, 20, 30)
	head = items.head().getOr(0)
	rest = items.tail()
	removed = items.remove(1)
	empty = None()

	Term.print("left", "-", "right")

	if empty.isEmpty() {
		return head * 1000 + rest.size() * 100 + removed.getOr(0)
	}
	return 0
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
	if value != int64(10220) {
		t.Fatalf("expected 10220, got %#v", value)
	}
	if string(output) != "left-right" {
		t.Fatalf("unexpected output %q", string(output))
	}
}

func TestStringConcatenation(t *testing.T) {
	src := `
def run() Str {
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

func TestStringInterpolation(t *testing.T) {
	src := `
def run() Str {
	name Str = "world"
	count Int = 2
	return "hello $name ${count + 1} \$done"
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "hello world 3 $done" {
		t.Fatalf("expected %q, got %#v", "hello world 3 $done", value)
	}
}

func TestMultilineString(t *testing.T) {
	src := `
def run() Str {
	return """
hello
$name
\n
"""
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "\nhello\n$name\n\n\n" {
		t.Fatalf("expected %q, got %#v", "\nhello\n$name\n\n\n", value)
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
	items2 = for item <- values yield {
		item + 1
	}
	Term.println("items " + items.size() + " " + items2.size())
	return label + items.size() + items2.size()
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
	if value != int64(22) {
		t.Fatalf("expected 22, got %#v", value)
	}
	if strings.TrimSpace(string(output)) != "items 9 3" {
		t.Fatalf("expected output %q, got %q", "items 9 3", string(output))
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

func TestIsExpression(t *testing.T) {
	src := `
class Counter {
}

def run() Bool {
	counter = Counter()
	return counter is Counter && "x" is Str && !(counter is Str)
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

func TestInterfaceInheritanceIsExpression(t *testing.T) {
	src := `
interface Hopper {
	def hop() Str
}

interface Jumper {
	def jump(steps Int) Str
}

interface Acrobat with Hopper, Jumper {
}

class Rabbit with Acrobat {
	impl def hop() Str = "hop"
	impl def jump(steps Int) Str = "jump " + steps
}

def run() Bool {
	rabbit = Rabbit()
	return rabbit is Acrobat && rabbit is Hopper && rabbit is Jumper
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

func TestRecordUpdateExpr(t *testing.T) {
	src := `
record Amount {
	amount Int
	description Str
}

def run() Bool {
	value = Amount(10, "x")
	updated = value with { amount = 42 }
	return value.amount == 10 && updated.amount == 42 && updated.description == "x"
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

func TestRecordAndClassDestructuring(t *testing.T) {
	src := `
record Pair {
	left Int
	right Str
}

class Box {
	value Int
	label Str
}

def run() Int {
	a Int, b Str = Pair(5, "x")
	c Int, d Str = Box(7, "y")
	return a + c
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(12) {
		t.Fatalf("expected 12, got %#v", value)
	}
}

func TestDestructuringSkipBinding(t *testing.T) {
	src := `
record Triple {
	first Int
	middle Str
	last Str
}

def run() Str {
	a Int, _, c Str = Triple(1, "drop", "keep")
	return c
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "keep" {
		t.Fatalf("expected keep, got %#v", value)
	}
}

func TestNamedCallArguments(t *testing.T) {
	src := `
class Counter {
	def set(value Int, label Str) Int {
		return value
	}
}

def doSomething(a Str, b Int) Int {
	return b
}

def run() Int {
	counter = Counter()
	left Int = doSomething(b = 5, a = "go")
	right Int = counter.set(label = "ok", value = 7)
	return left + right
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(12) {
		t.Fatalf("expected 12, got %#v", value)
	}
}

func TestObjectSingletonAccess(t *testing.T) {
	src := `
object A {
	count Int := 2

	def value() Int = count

	def test(a Int) Int {
		return a + this.value()
	}
}

def run() Int {
	return A.test(5)
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

func TestObjectApplyCall(t *testing.T) {
	src := `
object Range {
	def apply(end Int) Int = end
}

def run() Int {
	return Range.apply(5)
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

func TestObjectDirectCall(t *testing.T) {
	src := `
object Range {
	def apply(end Int) Int = end
}

def run() Int {
	return Range(5)
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

func TestClassApplyCall(t *testing.T) {
	src := `
class Adder {
	amount Int

	def apply(value Int) Int = amount + value
}

def run() Int {
	adder Adder = Adder(5)
	return adder(7)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(12) {
		t.Fatalf("expected 12, got %#v", value)
	}
}

func TestRecordApplyCall(t *testing.T) {
	src := `
record Adder {
	amount Int

	def apply(value Int) Int = amount + value
}

def run() Int {
	adder Adder = Adder(5)
	return adder(7)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(12) {
		t.Fatalf("expected 12, got %#v", value)
	}
}

func TestImplicitPrimaryConstructorAndThisDelegation(t *testing.T) {
	src := `
class Counter {
	count Int
	label Str
	private seen Bool := false

	def this(seed Int) {
		this(count = seed, label = "ok")
	}

	def value() Int = count
}

def run() Int {
	left Counter = Counter(1, "x")
	right Counter = Counter(seed = 3)
	return left.value() + right.value()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(4) {
		t.Fatalf("expected 4, got %#v", value)
	}
}

func TestYieldLoopsWithImmutableBindings(t *testing.T) {
	src := `
def run() Int {
	items = for {
		item <- [1, 2, 3]
		next = item + 1
	} yield {
		next
	}

	return items.get(0).get() + items.get(1).get() + items.get(2).get()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(9) {
		t.Fatalf("expected 9, got %#v", value)
	}
}

func TestConditionalForLoop(t *testing.T) {
	src := `
def run() Int {
	total Int := 0
	for total < 3 {
		total += 1
	}
	return total
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

func TestYieldLoopsWithMutableBindings(t *testing.T) {
	src := `
def run() Int {
	items = for {
		item <- [1, 2, 3]
		total := item
	} yield {
		total += 1
		total
	}

	return items.get(0).get() + items.get(1).get() + items.get(2).get()
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(9) {
		t.Fatalf("expected 9, got %#v", value)
	}
}

func TestIfOptionBinding(t *testing.T) {
	src := `
def run() Int {
	found Option[Int] = Some(5)
	missing Option[Int] = None()
	total Int := 0
	if item <- found {
		total := total + item
	}
	if item <- missing {
		total := total + item
	}
	return total
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

func TestIfOptionDestructuring(t *testing.T) {
	src := `
def run() Str {
	found Option[(Int, Str, Bool)] = Some((1, "ok", true))
	if _, name Str, _ <- found {
		return name
	}
	return "missing"
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "ok" {
		t.Fatalf("expected %q, got %#v", "ok", value)
	}
}

func TestUnwrapStmtOptionResultEither(t *testing.T) {
	src := `
def plusOneOption(value Option[Int]) Option[Int] {
	item <- value
	return Some(item + 1)
}

def plusOneResult(value Result[Int, Str]) Result[Int, Str] {
	item <- value
	return Ok(item + 1)
}

def plusOneEither(value Either[Str, Int]) Either[Str, Int] {
	item <- value
	return Right(item + 1)
}

def twoEithers(value Either[Str, Int], value2 Either[Str, Str]) Either[Str, Int] {
	item <- value
	size <- value2.map((s Str) -> s.size())
	return Right(item + size)
}

def run() Str {
	optionValue = plusOneOption(Some(4)).getOr(0)
	resultValue = plusOneResult(Ok(5)).getOr(0)
	eitherValue = plusOneEither(Right(6)).getOr(0)
	comboValue = twoEithers(Right(7), Right("abc")).getOr(0)

	if plusOneOption(None()).isEmpty() &&
	   plusOneResult(Err("bad")).isErr() &&
	   plusOneEither(Left("nope")).isLeft() &&
	   twoEithers(Right(7), Left("size bad")).isLeft() {
		return "${optionValue}-${resultValue}-${eitherValue}-${comboValue}"
	}
	return "broken"
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != "5-6-7-10" {
		t.Fatalf("expected %q, got %#v", "5-6-7-10", value)
	}
}

func TestStrSize(t *testing.T) {
	src := `
def run() Int {
	return "hello".size() + "мир".size()
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

func TestNestedBlockExpressions(t *testing.T) {
	src := `
def run() Int {
	a1 = {
		1 + 7
	}
	{
		Term.println("xxx")
	}
	v := {
		a = 5
		{
			a + 1
		}
	}
	return a1 + v
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(14) {
		t.Fatalf("expected 14, got %#v", value)
	}
}

func TestNestedBlockValueStatements(t *testing.T) {
	src := `
def run() Int {
	fromIf = {
		if false {
			10
		} else {
			20
		}
	}
	fromYield = {
		for item <- [1, 2, 3] yield {
			if item % 2 == 0 {
				item * 10
			} else {
				item
			}
		}
	}
	return fromIf + fromYield.get(1).getOr(0)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(40) {
		t.Fatalf("expected 40, got %#v", value)
	}
}

func TestPlaceholderLambdaShorthand(t *testing.T) {
	src := `
def applyTwice(f (Int) -> Int, value Int) Int = f(f(value))

def run() Int {
	inc (Int) -> Int = _ + 1
	items = List(1, 2, 3)
	mapped = items.map(_ + 1)
	return applyTwice(inc, mapped.get(0).getOr(0))
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(4) {
		t.Fatalf("expected 4, got %#v", value)
	}
}

func TestTupleDestructuringLambdas(t *testing.T) {
	src := `
def run() Int {
	pairs = List(("a", 1), ("bb", 2))
	pairMapped = pairs.map((key, value) -> key.size() + value)
	pairKeys = pairs.map((key, _) -> key)
	pairIgnored = pairs.map((_, value) -> value * 2)
	tuple4s = List((1, 2, 3, 4), (4, 5, 6, 7))
	tuple4Mapped = tuple4s.map((first, _, third, _) -> first + third)
	entries = Map("a": 1, "bbb": 2)
	mapMapped = entries.map((key, value) -> key.size() + value)
	return pairMapped.get(0).getOr(0) +
		pairMapped.get(1).getOr(0) +
		pairKeys.get(1).getOr("").size() +
		mapMapped.get(1).getOr(0) +
		pairIgnored.get(1).getOr(0) +
		tuple4Mapped.get(1).getOr(0)
}
`

	in := New(parseProgram(t, src))
	value, err := in.Call("run")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if value != int64(27) {
		t.Fatalf("expected 27, got %#v", value)
	}
}
