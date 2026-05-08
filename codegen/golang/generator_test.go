package golang

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"go/types"
	"testing"

	"a-lang/lower"
	"a-lang/parser"
	"a-lang/typecheck"
	"a-lang/typed"
)

func parseProgram(t *testing.T, src string) *parser.Program {
	t.Helper()
	program, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return program
}

func TestGenerate(t *testing.T) {
	src := `
class Counter {
	private var count Int
}

impl Counter {
	def init(count Int) {
		this.count = count
	}

	def inc(delta Int) Int {
		this.count += delta
		return this.count
	}
}

seed Int = 1

def sum(values Array[Int]) Int {
	var total Int = 0
	for item <- values {
		total += item
	}
	return total
}

def run(values Array[Int]) Int {
	bump Int -> Int = x -> x + 1
	counter Counter = Counter(seed)
	values[0] := values[0] + 1
	for {
		left <- values,
		right <- values
	} yield {
		left + right
	}
	if values[0] > 0 {
		return bump(counter.inc(sum(values)))
	}
	return seed
}
`

	program := parseProgram(t, src)
	typesResult := typecheck.Analyze(program)
	if len(typesResult.Diagnostics) != 0 {
		t.Fatalf("expected no type diagnostics, got %#v", typesResult.Diagnostics)
	}

	typedProgram, err := typed.Build(program, typesResult)
	if err != nil {
		t.Fatalf("typed.Build returned error: %v", err)
	}

	lowered, err := lower.ProgramFromTyped(typedProgram)
	if err != nil {
		t.Fatalf("ProgramFromTyped returned error: %v", err)
	}

	source, err := Generate(lowered)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "generated.go", source, goparser.AllErrors)
	if err != nil {
		t.Fatalf("generated source did not parse: %v\n%s", err, source)
	}

	conf := types.Config{}
	if _, err := conf.Check("generated", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("generated source did not type-check: %v\n%s", err, source)
	}
}
