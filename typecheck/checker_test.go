package typecheck

import (
	"os"
	"path/filepath"
	"testing"

	"a-lang/module"
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

func writeModuleFile(t *testing.T, path, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
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

func TestAnalyzeMatchStmt(t *testing.T) {
	src := `
enum OptionX[T] {
	case NoneX
	case SomeX {
		value T
	}
}

def run(value OptionX[Int]) Int {
	match value {
		SomeX(x) => {
			return x
		}
		OptionX.NoneX => {
			return 0
		}
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeMatchStmtRequiresEnumExhaustiveness(t *testing.T) {
	src := `
enum OptionX[T] {
	case NoneX
	case SomeX {
		value T
	}
}

def run(value OptionX[Int]) Int {
	match value {
		SomeX(x) => {
			return x
		}
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "non_exhaustive_match" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeMatchExprRequiresEnumExhaustiveness(t *testing.T) {
	src := `
enum OptionX[T] {
	case NoneX
	case SomeX {
		value T
	}
}

def run(value OptionX[Int]) Int =
	match value {
		SomeX(x) => x
	}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "non_exhaustive_match" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeMatchClassExtractor(t *testing.T) {
	src := `
class PairBox {
	left Int
	right Int
}

def run(value PairBox) Int {
	match value {
		PairBox(left, right) => {
			return left + right
		}
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeMatchRecordExtractor(t *testing.T) {
	src := `
record Amount {
	count Int
	label Str
}

def run(value Amount) Int {
	match value {
		Amount(count, label) => {
			return count
		}
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeMatchTypePattern(t *testing.T) {
	src := `
interface WorkerLike {
}

class Worker with WorkerLike {
}

class Other with WorkerLike {
}

def run(value WorkerLike) Int {
	match value {
		worker Worker => {
			return 1
		}
		_ Other => {
			return 2
		}
		_ => {
			return 3
		}
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeConstructorFieldAssignmentAllowsEquals(t *testing.T) {
	src := `
class Box {
	value Int

	def this(value Int) {
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
def fromSet(values Set[Int]) Int {
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
	label Str
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
	description Str
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

func TestAnalyzeRecordAndClassDestructuring(t *testing.T) {
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

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeDestructuringSkipBinding(t *testing.T) {
	src := `
record Triple {
	first Int
	middle Str
	last Str
}

def run() Int {
	a Int, _, c Str = Triple(1, "drop", "keep")
	return a
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeClassDestructuringRejectsPrivateFields(t *testing.T) {
	src := `
class Box {
	value Int
	private hidden Str = "x"
}

def run() Int {
	a Int, b Str = Box(7)
	return a
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	found := false
	for _, diag := range result.Diagnostics {
		if diag.Code == "invalid_binding_count" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected invalid_binding_count diagnostic, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeObjectSingletonAccess(t *testing.T) {
	src := `
object A {
	def value() Int = 5
	def test(a Int) Int = a + this.value()
}

def run() Int {
	return A.test(2)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeObjectApplyCall(t *testing.T) {
	src := `
object Range {
	def apply(end Int) Int = end
}

def run() Int {
	return Range.apply(5)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeObjectDirectCall(t *testing.T) {
	src := `
object Range {
	def apply(end Int) Int = end
}

def run() Int {
	return Range(5)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeClassApplyCall(t *testing.T) {
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

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeRecordApplyCall(t *testing.T) {
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

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeModuleAllowsPrivateClassWithinSamePackage(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, filepath.Join(dir, "app.al"), `
package app

private class Hidden {
	def value() Int = 7
}
`)
	writeModuleFile(t, filepath.Join(dir, "main.al"), `
package app

import app

def run() Int {
	value Hidden = app.Hidden()
	return value.value()
}
`)

	mod, err := module.Load(filepath.Join(dir, "main.al"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	result := AnalyzeModule(mod)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeModuleRejectsPrivateClassAcrossPackages(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, filepath.Join(dir, "lib.al"), `
package lib

private class Hidden {
	def value() Int = 7
}
`)
	writeModuleFile(t, filepath.Join(dir, "main.al"), `
package app

import lib

def run() Int {
	value = lib.Hidden()
	return 0
}
`)

	mod, err := module.Load(filepath.Join(dir, "main.al"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	result := AnalyzeModule(mod)
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	found := false
	for _, diag := range result.Diagnostics {
		if diag.Code == "unknown_member" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unknown_member diagnostic, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeModuleAllowsPrivateFunctionWithinSamePackage(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, filepath.Join(dir, "app.al"), `
package app

private def hidden() Int = 7
`)
	writeModuleFile(t, filepath.Join(dir, "main.al"), `
package app

import app

def run() Int {
	return app.hidden()
}
`)

	mod, err := module.Load(filepath.Join(dir, "main.al"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	result := AnalyzeModule(mod)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeModuleRejectsPrivateFunctionAcrossPackages(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, filepath.Join(dir, "lib.al"), `
package lib

private def hidden() Int = 7
`)
	writeModuleFile(t, filepath.Join(dir, "main.al"), `
package app

import lib

def run() Int {
	return lib.hidden()
}
`)

	mod, err := module.Load(filepath.Join(dir, "main.al"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	result := AnalyzeModule(mod)
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	found := false
	for _, diag := range result.Diagnostics {
		if diag.Code == "unknown_member" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unknown_member diagnostic, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeModuleExtendedImports(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, filepath.Join(dir, "model", "things.al"), `
package things

interface Named {
	def label() Str
}

class A with Named {
	impl def label() Str = "A"
}

class B with Named {
	impl def label() Str = "B"
}

object C {
	def apply(value Int) Int = value + 1
}
`)
	writeModuleFile(t, filepath.Join(dir, "main.al"), `
package app

import model/things
import model/things/A
import model/things/A as AliasA
import model/things/{B as AliasB, Named}
import model/things/*

def run() Int {
	value A = A()
	aliasA AliasA = AliasA()
	valueB AliasB = AliasB()
	named Named = value
	total Int = C(4)
	return total + things.C(5)
}
`)

	mod, err := module.Load(filepath.Join(dir, "main.al"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	result := AnalyzeModule(mod)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeInterfaceImplementation(t *testing.T) {
	src := `
interface Stringable {
	def show() Str
}

class Good with Stringable {
	def init() {
	}

	impl def show() Str {
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

func TestAnalyzeInterfaceImplementationRequiresImplDef(t *testing.T) {
	src := `
interface Stringable {
	def show() Str
}

class Bad with Stringable {
	def show() Str {
		return "ok"
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "interface_method_requires_impl" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeInterfaceInheritance(t *testing.T) {
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

def run() Str {
	rabbit = Rabbit()
	return rabbit.hop() + " " + rabbit.jump(3)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeImmutableFieldAssignmentInConstructor(t *testing.T) {
	src := `
class Counter {
	private count Int

	def this(count Int) {
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

func TestAnalyzeImmutableFieldAssignmentInInitMethodFails(t *testing.T) {
	src := `
class Counter {
	private count Int

	def init(count Int) {
		this.count = count
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "constructor_required" {
		t.Fatalf("unexpected first diagnostic %#v", result.Diagnostics[0])
	}
	if result.Diagnostics[1].Code != "assign_immutable" {
		t.Fatalf("unexpected second diagnostic %#v", result.Diagnostics[1])
	}
}

func TestAnalyzeImmutableFieldAssignmentOutsideConstructor(t *testing.T) {
	src := `
class Counter {
	private count Int

	def this(count Int) {
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

	def this(value Int) {
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

	def this(value Int) {
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

	def this(count Int) {
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

	def this(count Int) {
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

	def this(count Int) {
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

	def this(count Int) {
		this.count = count
	}

	def this(value Int) {
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

	def this() {
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

	def this(value T) {
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

func TestAnalyzeGenericFunctionAndMethodCalls(t *testing.T) {
	src := `
def id[T](value T) T = value

class Mapper {
	def map[X](value Int, fn Int -> X) X {
		fn(value)
	}
}

def run() Str {
	mapper Mapper = Mapper()
	mapped Str = mapper.map(5, (x Int) -> "value=" + x)
	return id(mapped)
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

	def this(count Int) {
		this.count = count
	}

	impl def equals(other Counter) Bool {
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

	def this(count Int) {
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
object Ascending with Ordering[Int] {
	impl def compare(left Int, right Int) Int = left - right
}

def run() Int {
	items List[Int] = List(1, 2)
	items.append(3)
	items.sort(Ascending)

	values Map[Str, Int] = Map("a" : 1)
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

func TestAnalyzeListSortWithOrdering(t *testing.T) {
	src := `
object Descending with Ordering[Int] {
	impl def compare(left Int, right Int) Int = right - left
}

def run() Int {
	items List[Int] = List(3, 1, 2)
	items.sort(Descending)
	return items.get(0).getOr(0) * 100 + items.get(1).getOr(0) * 10 + items.get(2).getOr(0)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeListMapFlatMapForEach(t *testing.T) {
	src := `
def run() Int {
	items List[Int] = List(1, 2, 3)
	doubled List[Int] = items.map((item Int) -> item * 2)
	doubled2 List[Int] = items.map(item -> item * 2)
	expanded List[Int] = items.flatMap((item Int) -> List(item, item + 10))
	filtered List[Int] = items.filter((item Int) -> item > 1)
	total Int = items.fold(0, (acc Int, item Int) -> acc + item)
	reduced Option[Int] = items.reduce((left Int, right Int) -> left + right)
	hasBig Bool = items.exists((item Int) -> item > 2)
	allPositive Bool = items.forAll((item Int) -> item > 0)
	doubled.forEach((item Int) -> Term.println(item))
	if hasBig && allPositive {
		return doubled.get(2).getOr(0) + doubled2.get(1).getOr(0) + expanded.get(5).getOr(0) + filtered.size() + total + reduced.getOr(0)
	}
	return 0
}
	`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeSetAndMapHigherOrderMethods(t *testing.T) {
	src := `
def run() Int {
	seen Set[Int] = Set(1, 2, 3)
	doubled Set[Int] = seen.map((item Int) -> item * 2)
	expanded Set[Int] = seen.flatMap((item Int) -> Set(item, item + 10))
	filtered Set[Int] = seen.filter((item Int) -> item > 1)
	setTotal Int = seen.fold(0, (acc Int, item Int) -> acc + item)
	setReduced Option[Int] = seen.reduce((left Int, right Int) -> left + right)
	setHasBig Bool = seen.exists((item Int) -> item > 2)
	setAllPositive Bool = seen.forAll((item Int) -> item > 0)
	seen.forEach((item Int) -> Term.println(item))

	values Map[Str, Int] = Map("a" : 1, "b" : 2)
	mapped List[Int] = values.map((key Str, value Int) -> value * 10)
	expandedValues List[Int] = values.flatMap((key Str, value Int) -> List(value, value + 10))
	filteredMap Map[Str, Int] = values.filter((key Str, value Int) -> value > 1)
	mapTotal Int = values.fold(0, (acc Int, key Str, value Int) -> acc + value)
	mapReduced Option[(Str, Int)] = values.reduce((leftKey Str, leftValue Int, rightKey Str, rightValue Int) -> (rightKey, rightValue))
	mapHasB Bool = values.exists((key Str, value Int) -> key == "b")
	mapAllSmall Bool = values.forAll((key Str, value Int) -> value < 3)
	values.forEach((key Str, value Int) -> Term.println(key))

	total Int := 0
	for item Int <- seen {
		total += item
	}
	for key Str, value Int <- values {
		total += value
	}

	reducedKey Str, reducedValue Int = mapReduced.get()
	if expanded.contains(12) && setHasBig && setAllPositive && mapHasB && mapAllSmall {
		if reducedKey == "b" {
			return total + mapped.get(0).getOr(0) + expandedValues.get(3).getOr(0) + doubled.size() + filtered.size() + setTotal + setReduced.getOr(0) + filteredMap.size() + mapTotal + reducedValue
		}
	}
	return 0
}
	`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeCustomAndCollectionOperators(t *testing.T) {
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
}

def run() Int {
	left Vec = Vec(1, 2)
	right Vec = Vec(3, 4)
	total Vec = left + right
	neg Vec = -total

	items List[Int] = List(1, 2)
	items2 List[Int] = items :+ 3
	merged List[Int] = items2 ++ List(4, 5)

	seen Set[Int] = Set(1, 2)
	all Set[Int] = seen ++ Set(3)

	return neg[0] + merged[4] + all.size()
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeGenericBounds(t *testing.T) {
	src := `
object Ascending with Ordering[Int] {
	impl def compare(left Int, right Int) Int = left - right
}

class Box[T with Ordering[T]] {
	value T
}

class Mapper {
	def pick[X with Ordering[X]](value X) X = value
}

def pick[T with Ordering[T]](value T) T = value
def useBox(value Box[Int]) Int = value.value

def run() Int {
	mapper Mapper = Mapper()
	value Int = pick(3)
	return value + mapper.pick(4)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeGenericBoundsRejectMissingWitness(t *testing.T) {
src := `
class Box[T with Ordering[T]] {
	value T
}

def pick[T with Ordering[T]](value T) T = value
def badBox(value Box[Str]) Int = 0

def run() Int {
	value Str = pick("x")
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics for missing Ordering[Str] witness")
	}
	if result.Diagnostics[0].Code != "type_argument_bound" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeRejectDeferredBindingsOutsideClasses(t *testing.T) {
	src := `
value Int = ?

def run() Int {
	local Int := ?
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %#v", result.Diagnostics)
	}
	for _, diag := range result.Diagnostics {
		if diag.Code != "invalid_deferred" {
			t.Fatalf("unexpected diagnostic %#v", diag)
		}
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
def run() Str {
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

func TestAnalyzeForDestructuring(t *testing.T) {
	src := `
def run(rows List[(Int, Str)]) Int {
	total Int := 0
	for left Int, right Str <- rows {
		if right == "x" {
			total += left
		}
	}
	return total
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeConditionalForLoop(t *testing.T) {
	src := `
def run(limit Int) Int {
	total Int := 0
	for total < limit {
		total += 1
	}
	return total
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeForYieldImmutableBindings(t *testing.T) {
	src := `
def run(values List[Int]) List[Int] {
	return for {
		item <- values
		next = item + 1
	} yield {
		next
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeForYieldMutableBindings(t *testing.T) {
	src := `
def run(values List[Int]) List[Int] {
	return for {
		item <- values
		total := item
	} yield {
		total += 1
		total
	}
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeRejectsUselessExpressionStatement(t *testing.T) {
	src := `
def run() Int {
	value = 1
	value
	return value
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "useless_expression" {
		t.Fatalf("expected useless_expression, got %#v", result.Diagnostics[0])
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
	def set(value Int, label Str) Int {
		return value
	}
}

def doSomething(a Str, b Int) Int {
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
def suffix(value Str) Str {
	value + "!"
}

def run() Str {
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
	b Str := ""
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

func TestAnalyzeIfOptionBinding(t *testing.T) {
	src := `
def run(value Option[Int]) Int {
	if item <- value {
		return item
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeIfOptionBindingRejectsNonOption(t *testing.T) {
	src := `
def run(value Int) Int {
	if item <- value {
		return item
	}
	return 0
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "invalid_condition_type" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeIfOptionDestructuring(t *testing.T) {
	src := `
def run(value Option[(Int, Str, Bool)]) Str {
	if _, name Str, _ <- value {
		return name
	}
	return "missing"
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeUnwrapStmtOptionResultEither(t *testing.T) {
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
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
}

func TestAnalyzeUnwrapStmtRejectsIncompatibleReturn(t *testing.T) {
	src := `
def run(value Result[Int, Str]) Option[Int] {
	item <- value
	return Some(item)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "invalid_unwrap" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}

func TestAnalyzeUnwrapStmtRejectsNonUnwrappable(t *testing.T) {
	src := `
def run(value Int) Option[Int] {
	item <- value
	return Some(item)
}
`

	result := Analyze(parseProgram(t, src))
	if len(result.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}
	if result.Diagnostics[0].Code != "invalid_unwrap" {
		t.Fatalf("unexpected diagnostic %#v", result.Diagnostics[0])
	}
}
