package parser

import "testing"

const sampleProgram = `
def doSomeWork(a Int, b Int) Bool {

	let list = [a, b, c]
	let set = Set()
	let map = {}
	let map2 = Map(a : b)
	let tuple = Set(a, b)
	let tuple2 = a : b : c
	let tuple21 = a : b : c
	let array = Array(1, 2, 3)
	let string String = "xxx"
	let a int = 65

	let a Int, b Int64 = 1, 3

	if a == b {

	} else {

	}

	for a <- list {

		if a == 7 {
			break
		}
	}

	for a <- [1..100] {

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

	for {
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

	third := fn.Body.Statements[2].(*ForStmt)
	if len(third.Bindings) != 0 || third.Body == nil {
		t.Fatalf("expected infinite for loop, got %#v", third)
	}

	fourth := fn.Body.Statements[3].(*ForStmt)
	if len(fourth.Bindings) != 2 || fourth.YieldBody == nil {
		t.Fatalf("expected yield-style for loop, got %#v", fourth)
	}
}

func TestParseExtendedOperators(t *testing.T) {
	src := `
def ops(a Int, b Int) Bool {
	let x = a - b / 2 * 3 % 5
	let y = !a == b
	let z = a != b
	let c = a < b
	let d = a <= b
	let e = a > b
	let f = a >= b
	let yes = true
	let no = false
	let pi = 1.1
	let whole = 1.
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
	let yes = true
	let no = false
	let a = 1.1
	let b = 1.
	let c = 'x'
	let d = '\n'
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
	def toString() String
}

class Box[T] implements Mapper[T, Stringable] {
	private let value T

	def init(value T) {
		this.value = value
	}

	def map(key T) Stringable {
		return this
	}
}

class SolidWork implements Stringable {
	private let a List[Int]
	private var b Map[String, Bool]

	def init(a Int, b Bool) {
		this.a = a
		this.b = b
	}

	def toString() String {
	}

	def addOne(one Int) Int {
	}

	private def buildLabel() String {
	}
}

class RecordKeeper {
	let record Set[String]
	private let approved Bool
}

let recordKeeper = RecordKeeper("test record", true)
let solidWork = SolidWork(1, false)
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

func TestRejectInvalidRuneLiterals(t *testing.T) {
	cases := []string{
		"def bad() Bool { let a = '' return true }",
		"def bad() Bool { let a = 'ab' return true }",
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
	var count Int = 1
	var left Int, right Int = 1, 2
	count = count + 1
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
	var counter = 0
	counter = counter + 1
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
	if assign.Operator != "=" {
		t.Fatalf("expected plain assignment operator, got %q", assign.Operator)
	}

	compound, ok := fn.Body.Statements[2].(*AssignmentStmt)
	if !ok || compound.Operator != "+=" {
		t.Fatalf("expected compound assignment with +=, got %#v", fn.Body.Statements[2])
	}
}

func TestParseImmutableBindings(t *testing.T) {
	src := `
def vars() Bool {
	let count Int = 1
	let left Int, right Int = 1, 2
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
		t.Fatalf("expected let binding to be immutable")
	}

	second := fn.Body.Statements[1].(*ValStmt)
	if second.Bindings[0].Mutable || second.Bindings[1].Mutable {
		t.Fatalf("expected let bindings to be immutable")
	}
}

func TestParseUntypedBindings(t *testing.T) {
	src := `
def vars() Bool {
	let a = "some string"
	var counter = 0
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
		t.Fatalf("expected untyped let binding, got type %#v", letStmt.Bindings[0].Type)
	}
	if letStmt.Bindings[0].Mutable {
		t.Fatalf("expected let binding to be immutable")
	}

	varStmt := fn.Body.Statements[1].(*ValStmt)
	if varStmt.Bindings[0].Type != nil {
		t.Fatalf("expected untyped var binding, got type %#v", varStmt.Bindings[0].Type)
	}
	if !varStmt.Bindings[0].Mutable {
		t.Fatalf("expected var binding to be mutable")
	}
}

func TestParseFunctionInvocationBinding(t *testing.T) {
	src := `
def vars(b Int) Bool {
	let a = function(b)
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

func TestParseLambdaBindings(t *testing.T) {
	src := `
def vars() Bool {
	let a = Map(1 : "string").map((key, value) -> key.toString() + value)
	let b = Set(1).map(key -> key.toString())
	let c = Map(1 : 2).map((key Int, value Int) -> key + value)
	let d = Set(1).map(key Int -> key.toString())
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
	firstLambda, ok := firstCall.Args[0].(*LambdaExpr)
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
	secondLambda, ok := secondCall.Args[0].(*LambdaExpr)
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
	thirdLambda, ok := thirdCall.Args[0].(*LambdaExpr)
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
	fourthLambda, ok := fourthCall.Args[0].(*LambdaExpr)
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
	let add = (x Int) -> {
		let y Int = x + 1
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
	let values List[T]

	def init(values List[T]) {
	}
}

def wrap(input Map[String, List[Int]]) List[Map[String, Int]] {
	let cache Map[String, List[Int]] = input
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
def apply(value Int, f (Int) -> Int) Int {
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

func TestAttachSourceSpans(t *testing.T) {
	src := `
def sample(a Int) Bool {
	let value = function(a)
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
		t.Fatalf("expected let statement to start on line 3, got %#v", stmt.Span)
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
