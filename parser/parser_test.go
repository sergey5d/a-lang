package parser

import "testing"

const sampleProgram = `
def doSomeWork(a Int, b Int) Bool {

	list = [a, b, c]
	set = Set()
	map = Map()
	map2 = Map(a : b)
	tuple = Set(a, b)
	tuple2 = a : b : c
	tuple21 = a : b : c
	array = Array(1, 2, 3)
	string String = "xxx"
	a int = 65

	a Int, b Int64 = 1, 3

	if a == b {

	} else {

	}

	for a <- list {

		if a == 7 {
			break
		}
	}

	for a <- Range(1, 100) {

	}

	for {
		a <- list,
		c <- map
	} yield {
		a + c
	}

	return a == 5
}
`

func assertTypeRef(t *testing.T, ref *TypeRef, name string, argNames ...string) {
	t.Helper()
	if ref == nil {
		t.Fatalf("expected type %q, got nil", name)
	}
	if ref.Name != name {
		t.Fatalf("expected type %q, got %q", name, ref.Name)
	}
	if len(ref.Arguments) != len(argNames) {
		t.Fatalf("expected %d type arguments for %q, got %d", len(argNames), name, len(ref.Arguments))
	}
	for i, argName := range argNames {
		if ref.Arguments[i].Name != argName {
			t.Fatalf("expected type argument %d for %q to be %q, got %q", i, name, argName, ref.Arguments[i].Name)
		}
	}
}

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
	if got := len(fn.Body.Statements); got != 16 {
		t.Fatalf("expected 16 statements in body, got %d", got)
	}
	if _, ok := fn.Body.Statements[len(fn.Body.Statements)-1].(*ReturnStmt); !ok {
		t.Fatalf("expected final statement to be return, got %T", fn.Body.Statements[len(fn.Body.Statements)-1])
	}
}

func TestParseHashComments(t *testing.T) {
	src := `
# top-level comment
def run() Int {
	# before binding
	value Int = 1 # trailing comment
	# before return
	return value
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(program.Functions))
	}
	fn := program.Functions[0]
	if got := len(fn.Body.Statements); got != 2 {
		t.Fatalf("expected 2 statements in body, got %d", got)
	}
	if _, ok := fn.Body.Statements[0].(*ValStmt); !ok {
		t.Fatalf("expected first statement to be binding, got %T", fn.Body.Statements[0])
	}
	if _, ok := fn.Body.Statements[1].(*ReturnStmt); !ok {
		t.Fatalf("expected second statement to be return, got %T", fn.Body.Statements[1])
	}
}

func TestParseForForms(t *testing.T) {
	src := `
def loops(input Int, value Int) Bool {
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

	loop {
		break
	}

	for {
		x <- [value],
		y <- [input]
	} yield {
		x + y
	}

	return value == input
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first := fn.Body.Statements[0].(*ForStmt)
	if len(first.Bindings) != 1 || first.Body == nil {
		t.Fatalf("expected single-binding for loop, got %#v", first)
	}

	third := fn.Body.Statements[2].(*LoopStmt)
	if third.Body == nil {
		t.Fatalf("expected infinite loop, got %#v", third)
	}

	fourth := fn.Body.Statements[3].(*ForStmt)
	if len(fourth.Bindings) != 2 || fourth.YieldBody == nil {
		t.Fatalf("expected yield-style for loop, got %#v", fourth)
	}
}

func TestParseIfAndForYieldExpressions(t *testing.T) {
	src := `
def run(values List[Int], flag Bool) Int {
	label = if flag {
		1
	} else {
		2
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

	return label
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first := fn.Body.Statements[0].(*ValStmt)
	if _, ok := first.Values[0].(*IfExpr); !ok {
		t.Fatalf("expected first binding value to be if expression, got %T", first.Values[0])
	}
	second := fn.Body.Statements[1].(*ValStmt)
	if _, ok := second.Values[0].(*ForYieldExpr); !ok {
		t.Fatalf("expected second binding value to be yield expression, got %T", second.Values[0])
	}
	third := fn.Body.Statements[2].(*ValStmt)
	if yieldExpr, ok := third.Values[0].(*ForYieldExpr); !ok {
		t.Fatalf("expected third binding value to be short yield expression, got %T", third.Values[0])
	} else if len(yieldExpr.Bindings) != 1 {
		t.Fatalf("expected short yield expression to have 1 binding, got %d", len(yieldExpr.Bindings))
	}
}

func TestParseIsExpression(t *testing.T) {
	src := `
def run(value Any) Bool {
	return value is String
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	ret := fn.Body.Statements[0].(*ReturnStmt)
	isExpr, ok := ret.Value.(*IsExpr)
	if !ok {
		t.Fatalf("expected return value to be is expression, got %T", ret.Value)
	}
	if isExpr.Target == nil || isExpr.Target.Name != "String" {
		t.Fatalf("expected is target String, got %#v", isExpr.Target)
	}
}

func TestParsePackageAndImports(t *testing.T) {
	src := `
package app
import util
import model/user

def main() Unit {
	()
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if program.PackageName != "app" {
		t.Fatalf("expected package app, got %q", program.PackageName)
	}
	if len(program.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(program.Imports))
	}
	if program.Imports[0].Path != "util" || program.Imports[1].Path != "model/user" {
		t.Fatalf("unexpected imports %#v", program.Imports)
	}
}

func TestParseMethodWithoutReturnType(t *testing.T) {
	src := `
class Counter {
	def touch() {
		1
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	method := program.Classes[0].Methods[0]
	if method.Constructor {
		t.Fatalf("expected ordinary method, got constructor %#v", method)
	}
	if method.ReturnType == nil || method.ReturnType.Name != "Unit" {
		t.Fatalf("expected omitted return type to normalize to Unit, got %#v", method.ReturnType)
	}
}

func TestParseThisConstructorAndFieldOrder(t *testing.T) {
	src := `
class Counter {
	count Int

	def this(seed Int) {
		this(count = seed)
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	method := program.Classes[0].Methods[0]
	if !method.Constructor || method.Name != "this" {
		t.Fatalf("expected def this to be marked as constructor, got %#v", method)
	}

	bad := `
class Broken {
	def run() Unit {
	}
	count Int
}
`
	if _, err := Parse(bad); err == nil {
		t.Fatalf("expected parse error for field after method")
	}
}

func TestParseExpressionBodiedDefs(t *testing.T) {
	src := `
def suffix(value String) String = value + "!"

class Counter {
	def value() Int = 1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in function body, got %d", len(fn.Body.Statements))
	}
	if _, ok := fn.Body.Statements[0].(*ExprStmt); !ok {
		t.Fatalf("expected expression-bodied function to parse as ExprStmt block, got %T", fn.Body.Statements[0])
	}

	method := program.Classes[0].Methods[0]
	if len(method.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in method body, got %d", len(method.Body.Statements))
	}
	if _, ok := method.Body.Statements[0].(*ExprStmt); !ok {
		t.Fatalf("expected expression-bodied method to parse as ExprStmt block, got %T", method.Body.Statements[0])
	}
}

func TestParseExplicitUnitDefs(t *testing.T) {
	src := `
def printIt() Unit = Term.println("x")

class Counter {
	def print() Unit = Term.println("y")
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if program.Functions[0].ReturnType == nil || program.Functions[0].ReturnType.Name != "Unit" {
		t.Fatalf("expected Unit return type for function, got %#v", program.Functions[0].ReturnType)
	}
	method := program.Classes[0].Methods[0]
	if method.ReturnType == nil || method.ReturnType.Name != "Unit" {
		t.Fatalf("expected Unit return type for method, got %#v", method.ReturnType)
	}
}

func TestParseFunctionWithoutReturnType(t *testing.T) {
	src := `
def printIt() {
	1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if fn.ReturnType == nil || fn.ReturnType.Name != "Unit" {
		t.Fatalf("expected Unit return type, got %#v", fn.ReturnType)
	}
}

func TestParseLocalFunctionStmt(t *testing.T) {
	src := `
def run() Int {
	def x(term Int) = term + 1
	x(5)
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if len(fn.Body.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(fn.Body.Statements))
	}
	if _, ok := fn.Body.Statements[0].(*LocalFunctionStmt); !ok {
		t.Fatalf("expected first statement to be local function, got %T", fn.Body.Statements[0])
	}
}

func TestParseEqualsBlockBodiedDefs(t *testing.T) {
	src := `
def top() = {
	1
}

class Counter {
	def method() = {
		1
	}
}

def run() {
	def local() = {
		1
	}
	local()
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(program.Functions[0].Body.Statements) != 1 {
		t.Fatalf("expected top function block body to contain 1 statement, got %d", len(program.Functions[0].Body.Statements))
	}
	if len(program.Classes[0].Methods[0].Body.Statements) != 1 {
		t.Fatalf("expected method block body to contain 1 statement, got %d", len(program.Classes[0].Methods[0].Body.Statements))
	}
	run := program.Functions[1]
	localStmt, ok := run.Body.Statements[0].(*LocalFunctionStmt)
	if !ok {
		t.Fatalf("expected first run statement to be local function, got %T", run.Body.Statements[0])
	}
	if len(localStmt.Function.Body.Statements) != 1 {
		t.Fatalf("expected local function block body to contain 1 statement, got %d", len(localStmt.Function.Body.Statements))
	}
}

func TestParseVariadicParameter(t *testing.T) {
	src := `
def printAll(values String...) {
	values.size()
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	param := program.Functions[0].Parameters[0]
	if !param.Variadic {
		t.Fatalf("expected parameter to be variadic")
	}
}

func TestParseMultiAssignmentStmt(t *testing.T) {
	src := `
def run() Int {
	a Int := 0
	b Int := 0
	a, b := 1, 2
	a
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if _, ok := fn.Body.Statements[2].(*MultiAssignmentStmt); !ok {
		t.Fatalf("expected multi assignment statement, got %T", fn.Body.Statements[2])
	}
}

func TestParseDeclaredNamesWithEqualsAsBindings(t *testing.T) {
	src := `
def run() Int {
	a Int := 0
	b Int := 0
	a, b = 1, 2
	a
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if _, ok := fn.Body.Statements[2].(*ValStmt); !ok {
		t.Fatalf("expected binding statement, got %T", fn.Body.Statements[2])
	}
}

func TestParseTupleLiteralAndType(t *testing.T) {
	src := `
def run() (value Int, label String) {
	pair (value Int, label String) = (1, "ok")
	pair
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if fn.ReturnType == nil || len(fn.ReturnType.TupleElements) != 2 {
		t.Fatalf("expected tuple return type, got %#v", fn.ReturnType)
	}
	if fn.ReturnType.TupleNames[0] != "value" || fn.ReturnType.TupleNames[1] != "label" {
		t.Fatalf("expected tuple names to be preserved, got %#v", fn.ReturnType.TupleNames)
	}
	stmt := fn.Body.Statements[0].(*ValStmt)
	if stmt.Bindings[0].Type == nil || len(stmt.Bindings[0].Type.TupleElements) != 2 {
		t.Fatalf("expected tuple binding type, got %#v", stmt.Bindings[0].Type)
	}
	if stmt.Bindings[0].Type.TupleNames[0] != "value" || stmt.Bindings[0].Type.TupleNames[1] != "label" {
		t.Fatalf("expected named tuple binding type, got %#v", stmt.Bindings[0].Type.TupleNames)
	}
	if _, ok := stmt.Values[0].(*TupleLiteral); !ok {
		t.Fatalf("expected tuple literal, got %T", stmt.Values[0])
	}
}

func TestParseUnitAndGroupedParenExpr(t *testing.T) {
	src := `
def run() Unit {
	unit = ()
	value = (1)
	pair = (1, 2)
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	body := program.Functions[0].Body.Statements
	unitStmt := body[0].(*ValStmt)
	if _, ok := unitStmt.Values[0].(*UnitLiteral); !ok {
		t.Fatalf("expected unit literal, got %T", unitStmt.Values[0])
	}

	groupStmt := body[1].(*ValStmt)
	if _, ok := groupStmt.Values[0].(*GroupExpr); !ok {
		t.Fatalf("expected grouped expression, got %T", groupStmt.Values[0])
	}

	tupleStmt := body[2].(*ValStmt)
	if _, ok := tupleStmt.Values[0].(*TupleLiteral); !ok {
		t.Fatalf("expected tuple literal, got %T", tupleStmt.Values[0])
	}
}

func TestParseZeroArgFunctionBindingSugar(t *testing.T) {
	src := `
def run() Unit {
	action () -> Unit = Term.println("x")
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	stmt := program.Functions[0].Body.Statements[0].(*ValStmt)
	if _, ok := stmt.Values[0].(*LambdaExpr); !ok {
		t.Fatalf("expected lambda expression, got %T", stmt.Values[0])
	}
}

func TestParseUnitFunctionType(t *testing.T) {
	src := `
def run(action () -> Unit) Unit {
	action()
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fnType := program.Functions[0].Parameters[0].Type
	if fnType.ReturnType == nil || fnType.ReturnType.Name != "Unit" {
		t.Fatalf("expected Unit return type, got %#v", fnType)
	}
	if len(fnType.ParameterTypes) != 0 {
		t.Fatalf("expected zero-arg function type, got %#v", fnType.ParameterTypes)
	}
}

func TestParseExtendedOperators(t *testing.T) {
	src := `
def ops(a Int, b Int) Bool {
	x = a - b / 2 * 3 % 5
	y = !a == b
	z = a != b
	c = a < b
	d = a <= b
	e = a > b
	f = a >= b
	yes = true
	no = false
	pi = 1.1
	whole = 1.
	return a == b || a != b && !(a < b)
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

func TestParseBoolAndFloatLiterals(t *testing.T) {
	src := `
def literals() Bool {
	yes = true
	no = false
	a = 1.1
	b = 1.
	c = 'x'
	d = '\n'
	return yes == true
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	if _, ok := fn.Body.Statements[0].(*ValStmt).Values[0].(*BoolLiteral); !ok {
		t.Fatalf("expected true to parse as bool literal")
	}
	if _, ok := fn.Body.Statements[1].(*ValStmt).Values[0].(*BoolLiteral); !ok {
		t.Fatalf("expected false to parse as bool literal")
	}
	if _, ok := fn.Body.Statements[2].(*ValStmt).Values[0].(*FloatLiteral); !ok {
		t.Fatalf("expected 1.1 to parse as float literal")
	}
	if literal, ok := fn.Body.Statements[3].(*ValStmt).Values[0].(*FloatLiteral); !ok || literal.Value != "1." {
		t.Fatalf("expected 1. to parse as float literal, got %#v", fn.Body.Statements[3].(*ValStmt).Values[0])
	}
	if literal, ok := fn.Body.Statements[4].(*ValStmt).Values[0].(*RuneLiteral); !ok || literal.Value != "x" {
		t.Fatalf("expected 'x' to parse as rune literal, got %#v", fn.Body.Statements[4].(*ValStmt).Values[0])
	}
	if literal, ok := fn.Body.Statements[5].(*ValStmt).Values[0].(*RuneLiteral); !ok || literal.Value != "\\n" {
		t.Fatalf("expected '\\n' to parse as escaped rune literal, got %#v", fn.Body.Statements[5].(*ValStmt).Values[0])
	}
}

func TestParseInterfacesAndClasses(t *testing.T) {
	src := `
interface Mapper[K, V] {
	def map(key K) V
}

interface Stringable {
	def show() String
}

class Box[T] with Mapper[T, Stringable] {
	private value T

	def init(value T) {
		this.value = value
	}

	def map(key T) Stringable {
		return this
	}
}

class SolidWork with Stringable {
	private a List[Int]
	private b Map[String, Bool] := ?

	def init(a Int, b Bool) {
		this.a = a
		this.b = b
	}

	def show() String {
	}

	def addOne(one Int) Int {
	}

	private def buildLabel() String {
	}
}

class RecordKeeper {
	entries Set[String]
	private approved Bool
}

recordKeeper = RecordKeeper("test record", true)
solidWork = SolidWork(1, false)
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(program.Interfaces))
	}
	if len(program.Classes) != 3 {
		t.Fatalf("expected 3 classes, got %d", len(program.Classes))
	}
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 top-level statements, got %d", len(program.Statements))
	}
	mapper := program.Interfaces[0]
	if mapper.Name != "Mapper" || len(mapper.TypeParameters) != 2 {
		t.Fatalf("unexpected generic interface %#v", mapper)
	}
	assertTypeRef(t, mapper.Methods[0].Parameters[0].Type, "K")
	assertTypeRef(t, mapper.Methods[0].ReturnType, "V")

	box := program.Classes[0]
	if box.Name != "Box" || len(box.TypeParameters) != 1 {
		t.Fatalf("unexpected generic class %#v", box)
	}
	if len(box.Implements) != 1 || box.Implements[0].Name != "Mapper" {
		t.Fatalf("unexpected generic implements clause %#v", box.Implements)
	}
	assertTypeRef(t, box.Implements[0], "Mapper", "T", "Stringable")
	assertTypeRef(t, box.Fields[0].Type, "T")

	cls := program.Classes[1]
	if cls.Name != "SolidWork" || len(cls.Implements) != 1 || cls.Implements[0].Name != "Stringable" {
		t.Fatalf("unexpected class declaration %#v", cls)
	}
	if len(cls.Fields) != 2 || len(cls.Methods) != 4 {
		t.Fatalf("unexpected class members %#v", cls)
	}
	assertTypeRef(t, cls.Fields[0].Type, "List", "Int")
	assertTypeRef(t, cls.Fields[1].Type, "Map", "String", "Bool")
	if !cls.Fields[0].Private || !cls.Fields[1].Private {
		t.Fatalf("expected first class fields to be private")
	}
	if !cls.Methods[3].Private {
		t.Fatalf("expected helper method to be private")
	}
	if !cls.Methods[0].Constructor {
		t.Fatalf("expected init to be marked as constructor")
	}
}

func TestParseRecordDecl(t *testing.T) {
	src := `
record Amount {
	amount Int
	description String

	def label() String = description
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Classes) != 1 {
		t.Fatalf("expected 1 aggregate decl, got %d", len(program.Classes))
	}
	if !program.Classes[0].Record {
		t.Fatalf("expected declaration to be marked as record")
	}
}

func TestParseObjectDecl(t *testing.T) {
	src := `
object A {
	def value() Unit = ()
	def test(a Int) Int = a + 5
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Classes) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Classes))
	}
	if !program.Classes[0].Object {
		t.Fatalf("expected declaration to be marked as object")
	}
	if len(program.Classes[0].Methods) != 2 {
		t.Fatalf("expected 2 object methods, got %#v", program.Classes[0].Methods)
	}
}

func TestParseObjectShortApplyDecl(t *testing.T) {
	src := `
object Range {
	def (end Int) Int = end
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Classes) != 1 || len(program.Classes[0].Methods) != 1 {
		t.Fatalf("unexpected object declaration %#v", program.Classes)
	}
	if program.Classes[0].Methods[0].Name != "apply" {
		t.Fatalf("expected short object method to normalize to apply, got %#v", program.Classes[0].Methods[0])
	}
}

func TestParsePrivateClassDecl(t *testing.T) {
	src := `
private class Hidden {
	def value() Int = 1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Classes) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Classes))
	}
	if !program.Classes[0].Private {
		t.Fatalf("expected class to be marked private")
	}
}

func TestParseDestructuringSkipBinding(t *testing.T) {
	src := `
def run() Int {
	a Int, _, c String = (1, 2, "x")
	return a
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	stmt, ok := program.Functions[0].Body.Statements[0].(*ValStmt)
	if !ok {
		t.Fatalf("expected binding statement, got %#v", program.Functions[0].Body.Statements[0])
	}
	if len(stmt.Bindings) != 3 || stmt.Bindings[1].Name != "_" {
		t.Fatalf("expected skip binding in middle, got %#v", stmt.Bindings)
	}
}

func TestParseInterfaceInheritance(t *testing.T) {
	src := `
interface Hopper {
	def hop() String
}

interface Acrobat with Hopper {
	def land() String
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(program.Interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(program.Interfaces))
	}
	acrobat := program.Interfaces[1]
	if acrobat.Name != "Acrobat" {
		t.Fatalf("unexpected interface %#v", acrobat)
	}
	if len(acrobat.Extends) != 1 || acrobat.Extends[0].Name != "Hopper" {
		t.Fatalf("unexpected extends clause %#v", acrobat.Extends)
	}
}

func TestParseRecordUpdateExpr(t *testing.T) {
	src := `
record Amount {
	amount Int
	description String
}

def run() Amount {
	value = Amount(10, "x")
	return value with { amount = 42, description = "y" }
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	run := program.Functions[0]
	ret, ok := run.Body.Statements[1].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %#v", run.Body.Statements[1])
	}
	update, ok := ret.Value.(*RecordUpdateExpr)
	if !ok {
		t.Fatalf("expected record update expr, got %#v", ret.Value)
	}
	if len(update.Updates) != 2 || update.Updates[0].Name != "amount" || update.Updates[1].Name != "description" {
		t.Fatalf("unexpected record updates %#v", update.Updates)
	}
}

func TestRejectInvalidRuneLiterals(t *testing.T) {
	cases := []string{
		"def bad() Bool { a = '' return true }",
		"def bad() Bool { a = 'ab' return true }",
	}

	for _, src := range cases {
		if _, err := Parse(src); err == nil {
			t.Fatalf("expected parse error for invalid rune literal in %q", src)
		}
	}
}

func TestParseMutableBindings(t *testing.T) {
	src := `
def vars() Bool {
	count Int := 1
	left Int, right Int := 1, 2
	count := count + 1
	return count == right
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first, ok := fn.Body.Statements[0].(*ValStmt)
	if !ok {
		t.Fatalf("expected first statement to be binding, got %T", fn.Body.Statements[0])
	}
	if !first.Bindings[0].Mutable {
		t.Fatalf("expected first binding to be mutable")
	}

	second, ok := fn.Body.Statements[1].(*ValStmt)
	if !ok {
		t.Fatalf("expected second statement to be binding, got %T", fn.Body.Statements[1])
	}
	if !second.Bindings[0].Mutable {
		t.Fatalf("expected first binding in second statement to be mutable")
	}
	if !second.Bindings[1].Mutable {
		t.Fatalf("expected second binding in second statement to be mutable")
	}
}

func TestParseAssignmentStatement(t *testing.T) {
	src := `
def vars() Bool {
	counter := 0
	counter := counter + 1
	counter += 2
	counter -= 1
	counter *= 3
	counter /= 2
	counter %= 2
	return counter == 1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	assign, ok := fn.Body.Statements[1].(*AssignmentStmt)
	if !ok {
		t.Fatalf("expected second statement to be assignment, got %T", fn.Body.Statements[1])
	}
	if _, ok := assign.Target.(*Identifier); !ok {
		t.Fatalf("expected assignment target to be identifier, got %T", assign.Target)
	}
	if assign.Operator != ":=" {
		t.Fatalf("expected mutable assignment operator, got %q", assign.Operator)
	}

	compound, ok := fn.Body.Statements[2].(*AssignmentStmt)
	if !ok || compound.Operator != "+=" {
		t.Fatalf("expected compound assignment with +=, got %#v", fn.Body.Statements[2])
	}
}

func TestParseIndexExpressionAndAssignment(t *testing.T) {
	src := `
def run(values Array[Int]) Int {
	item Int = values[0]
	values[1] := item
	return values[1]
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first, ok := fn.Body.Statements[0].(*ValStmt)
	if !ok {
		t.Fatalf("expected first statement to be binding, got %T", fn.Body.Statements[0])
	}
	index, ok := first.Values[0].(*IndexExpr)
	if !ok {
		t.Fatalf("expected binding value to be index expression, got %T", first.Values[0])
	}
	if _, ok := index.Receiver.(*Identifier); !ok {
		t.Fatalf("expected index receiver to be identifier, got %T", index.Receiver)
	}

	assign, ok := fn.Body.Statements[1].(*AssignmentStmt)
	if !ok {
		t.Fatalf("expected second statement to be assignment, got %T", fn.Body.Statements[1])
	}
	if _, ok := assign.Target.(*IndexExpr); !ok {
		t.Fatalf("expected assignment target to be index expression, got %T", assign.Target)
	}
	if assign.Operator != ":=" {
		t.Fatalf("expected mutable assignment operator, got %q", assign.Operator)
	}
}

func TestParseImmutableBindings(t *testing.T) {
	src := `
def vars() Bool {
	count Int = 1
	left Int, right Int = 1, 2
	return count == right
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	first := fn.Body.Statements[0].(*ValStmt)
	if first.Bindings[0].Mutable {
		t.Fatalf("expected immutable binding")
	}

	second := fn.Body.Statements[1].(*ValStmt)
	if second.Bindings[0].Mutable || second.Bindings[1].Mutable {
		t.Fatalf("expected immutable bindings")
	}
}

func TestParseUntypedBindings(t *testing.T) {
	src := `
def vars() Bool {
	a = "some string"
	counter := 0
	return counter == 0
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]

	letStmt := fn.Body.Statements[0].(*ValStmt)
	if letStmt.Bindings[0].Type != nil {
		t.Fatalf("expected untyped immutable binding, got type %#v", letStmt.Bindings[0].Type)
	}
	if letStmt.Bindings[0].Mutable {
		t.Fatalf("expected immutable binding")
	}

	mutableStmt := fn.Body.Statements[1].(*ValStmt)
	if mutableStmt.Bindings[0].Type != nil {
		t.Fatalf("expected untyped mutable binding, got type %#v", mutableStmt.Bindings[0].Type)
	}
	if !mutableStmt.Bindings[0].Mutable {
		t.Fatalf("expected mutable binding to be mutable")
	}
}

func TestParseFunctionInvocationBinding(t *testing.T) {
	src := `
def vars(b Int) Bool {
	a = function(b)
	return a == b
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	stmt := fn.Body.Statements[0].(*ValStmt)
	call, ok := stmt.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected binding value to be call expression, got %T", stmt.Values[0])
	}
	callee, ok := call.Callee.(*Identifier)
	if !ok || callee.Name != "function" {
		t.Fatalf("expected callee to be identifier 'function', got %#v", call.Callee)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 call argument, got %d", len(call.Args))
	}
}

func TestParseNamedCallArguments(t *testing.T) {
	src := `
def doSomething(a String, b Int) Unit {
}

def main() Unit {
	doSomething(a = "crap", b = 5)
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	stmt := program.Functions[1].Body.Statements[0].(*ExprStmt)
	call := stmt.Expr.(*CallExpr)
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 call arguments, got %d", len(call.Args))
	}
	if call.Args[0].Name != "a" || call.Args[1].Name != "b" {
		t.Fatalf("expected named arguments a, b, got %#v", call.Args)
	}
}

func TestParseLambdaBindings(t *testing.T) {
	src := `
def vars() Bool {
	a = Map(1 : "string").map((key, value) -> key.show() + value)
	b = Set(1).map(key -> key.show())
	c = Map(1 : 2).map((key Int, value Int) -> key + value)
	d = Set(1).map(key Int -> key.show())
	return 1 == 1
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]

	first := fn.Body.Statements[0].(*ValStmt)
	firstCall, ok := first.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected first binding value to be call expression, got %T", first.Values[0])
	}
	firstLambda, ok := firstCall.Args[0].Value.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected map argument to be lambda, got %T", firstCall.Args[0])
	}
	if len(firstLambda.Parameters) != 2 {
		t.Fatalf("expected 2 lambda parameters, got %d", len(firstLambda.Parameters))
	}

	second := fn.Body.Statements[1].(*ValStmt)
	secondCall, ok := second.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected second binding value to be call expression, got %T", second.Values[0])
	}
	secondLambda, ok := secondCall.Args[0].Value.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected map argument to be lambda, got %T", secondCall.Args[0])
	}
	if len(secondLambda.Parameters) != 1 || secondLambda.Parameters[0].Name != "key" || secondLambda.Parameters[0].Type != nil {
		t.Fatalf("unexpected single-parameter lambda: %#v", secondLambda.Parameters)
	}

	third := fn.Body.Statements[2].(*ValStmt)
	thirdCall, ok := third.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected third binding value to be call expression, got %T", third.Values[0])
	}
	thirdLambda, ok := thirdCall.Args[0].Value.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected typed map argument to be lambda, got %T", thirdCall.Args[0])
	}
	if thirdLambda.Parameters[0].Type == nil || thirdLambda.Parameters[0].Type.Name != "Int" || thirdLambda.Parameters[1].Type == nil || thirdLambda.Parameters[1].Type.Name != "Int" {
		t.Fatalf("expected typed lambda parameters, got %#v", thirdLambda.Parameters)
	}

	fourth := fn.Body.Statements[3].(*ValStmt)
	fourthCall, ok := fourth.Values[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected fourth binding value to be call expression, got %T", fourth.Values[0])
	}
	fourthLambda, ok := fourthCall.Args[0].Value.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected typed single-parameter lambda, got %T", fourthCall.Args[0])
	}
	if len(fourthLambda.Parameters) != 1 || fourthLambda.Parameters[0].Name != "key" || fourthLambda.Parameters[0].Type == nil || fourthLambda.Parameters[0].Type.Name != "Int" {
		t.Fatalf("unexpected typed single-parameter lambda: %#v", fourthLambda.Parameters)
	}
}

func TestParseBlockLambda(t *testing.T) {
	src := `
def vars() Int {
	add = (x Int) -> {
		y Int = x + 1
		return y
	}
	return add(1)
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	stmt := fn.Body.Statements[0].(*ValStmt)
	lambda, ok := stmt.Values[0].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected lambda expression, got %T", stmt.Values[0])
	}
	if lambda.BlockBody == nil || lambda.Body != nil {
		t.Fatalf("expected block-bodied lambda, got %#v", lambda)
	}
	if len(lambda.BlockBody.Statements) != 2 {
		t.Fatalf("expected 2 statements in lambda block, got %d", len(lambda.BlockBody.Statements))
	}
}

func TestParseGenericTypeRefs(t *testing.T) {
	src := `
interface Pairer[K, V] {
	def pair(left K, right V) Map[K, V]
}

class Store[T] {
	values List[T]

	def init(values List[T]) {
	}
}

def wrap(input Map[String, List[Int]]) List[Map[String, Int]] {
	cache Map[String, List[Int]] = input
	return [cache]
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	assertTypeRef(t, program.Interfaces[0].Methods[0].ReturnType, "Map", "K", "V")
	assertTypeRef(t, program.Classes[0].Fields[0].Type, "List", "T")
	assertTypeRef(t, program.Functions[0].Parameters[0].Type, "Map", "String", "List")
	if len(program.Functions[0].Parameters[0].Type.Arguments[1].Arguments) != 1 || program.Functions[0].Parameters[0].Type.Arguments[1].Arguments[0].Name != "Int" {
		t.Fatalf("expected nested generic type argument, got %#v", program.Functions[0].Parameters[0].Type)
	}
	assertTypeRef(t, program.Functions[0].ReturnType, "List", "Map")
	if len(program.Functions[0].ReturnType.Arguments[0].Arguments) != 2 {
		t.Fatalf("expected nested return type arguments, got %#v", program.Functions[0].ReturnType)
	}
}

func TestParseFunctionTypeRefs(t *testing.T) {
	src := `
def apply(value Int, f Int -> Int) Int {
	return f(value)
}

def pair(f (Int, String) -> Bool) Bool {
	return false
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fnType := program.Functions[0].Parameters[1].Type
	if fnType.ReturnType == nil || len(fnType.ParameterTypes) != 1 {
		t.Fatalf("expected single-parameter function type, got %#v", fnType)
	}
	assertTypeRef(t, fnType.ParameterTypes[0], "Int")
	assertTypeRef(t, fnType.ReturnType, "Int")

	pairType := program.Functions[1].Parameters[0].Type
	if pairType.ReturnType == nil || len(pairType.ParameterTypes) != 2 {
		t.Fatalf("expected two-parameter function type, got %#v", pairType)
	}
	assertTypeRef(t, pairType.ParameterTypes[0], "Int")
	assertTypeRef(t, pairType.ParameterTypes[1], "String")
	assertTypeRef(t, pairType.ReturnType, "Bool")
}

func TestParseElseIf(t *testing.T) {
	src := `
def classify(a Int) Bool {
	if a == 1 {
		return a == 1
	} else if a == 2 {
		return a == 2
	} else {
		return a == 3
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	ifStmt, ok := fn.Body.Statements[0].(*IfStmt)
	if !ok {
		t.Fatalf("expected first statement to be if, got %T", fn.Body.Statements[0])
	}
	if ifStmt.ElseIf == nil {
		t.Fatalf("expected else-if branch to be present")
	}
	if ifStmt.ElseIf.Else == nil {
		t.Fatalf("expected final else block on else-if chain")
	}
}

func TestParseIfOptionBinding(t *testing.T) {
	src := `
def run(value Option[Int]) Unit {
	if item <- value {
		Term.println(item)
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	fn := program.Functions[0]
	ifStmt, ok := fn.Body.Statements[0].(*IfStmt)
	if !ok {
		t.Fatalf("expected first statement to be if, got %T", fn.Body.Statements[0])
	}
	if len(ifStmt.Bindings) != 1 || ifStmt.Bindings[0].Name != "item" {
		t.Fatalf("expected binding name 'item', got %#v", ifStmt.Bindings)
	}
	if ifStmt.BindingValue == nil {
		t.Fatalf("expected binding value to be populated")
	}
	if ifStmt.Condition != nil {
		t.Fatalf("expected boolean condition to be empty for option-binding if")
	}
}

func TestParseIfOptionDestructuring(t *testing.T) {
	src := `
def run(value Option[(Int, String, Bool)]) Unit {
	if _, name String, _ <- value {
		Term.println(name)
	}
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	ifStmt := program.Functions[0].Body.Statements[0].(*IfStmt)
	if len(ifStmt.Bindings) != 3 {
		t.Fatalf("expected 3 bindings, got %#v", ifStmt.Bindings)
	}
	if ifStmt.Bindings[0].Name != "_" || ifStmt.Bindings[1].Name != "name" || ifStmt.Bindings[2].Name != "_" {
		t.Fatalf("unexpected bindings %#v", ifStmt.Bindings)
	}
	if ifStmt.Bindings[1].Type == nil || ifStmt.Bindings[1].Type.Name != "String" {
		t.Fatalf("expected explicit String binding type, got %#v", ifStmt.Bindings[1].Type)
	}
}

func TestAttachSourceSpans(t *testing.T) {
	src := `
def sample(a Int) Bool {
	value = function(a)
	return value == a
}
`

	program, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if program.Span.Start.Line == 0 || program.Span.End.Line == 0 {
		t.Fatalf("expected program span to be populated, got %#v", program.Span)
	}

	fn := program.Functions[0]
	if fn.Span.Start.Line != 2 {
		t.Fatalf("expected function to start on line 2, got %#v", fn.Span)
	}

	stmt := fn.Body.Statements[0].(*ValStmt)
	if stmt.Span.Start.Line != 3 {
		t.Fatalf("expected immutable binding statement to start on line 3, got %#v", stmt.Span)
	}

	call := stmt.Values[0].(*CallExpr)
	if call.Span.Start.Line != 3 || call.Span.End.Line != 3 {
		t.Fatalf("expected call span on line 3, got %#v", call.Span)
	}

	retStmt := fn.Body.Statements[1].(*ReturnStmt)
	if retStmt.Span.Start.Line != 4 {
		t.Fatalf("expected return statement span on line 4, got %#v", retStmt.Span)
	}
}
