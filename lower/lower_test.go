package lower

import (
	"testing"

	"a-lang/parser"
	"a-lang/typecheck"
)

func parseProgram(t *testing.T, src string) *parser.Program {
	t.Helper()
	program, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return program
}

func TestProgramFromAST(t *testing.T) {
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

seed Int = 1

def run(input Int) Int {
	counter Counter = Counter(input)
	if input > 0 {
		return counter.inc()
	}
	return seed
}
`

	program := parseProgram(t, src)
	types := typecheck.Analyze(program)
	if len(types.Diagnostics) != 0 {
		t.Fatalf("expected no type diagnostics, got %#v", types.Diagnostics)
	}

	lowered, err := ProgramFromAST(program, types)
	if err != nil {
		t.Fatalf("ProgramFromAST returned error: %v", err)
	}

	if len(lowered.Globals) != 1 {
		t.Fatalf("expected 1 global, got %d", len(lowered.Globals))
	}
	if len(lowered.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(lowered.Functions))
	}
	if len(lowered.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(lowered.Classes))
	}
	if lowered.Globals[0].Name != "seed" {
		t.Fatalf("unexpected global %#v", lowered.Globals[0])
	}
	if lowered.Classes[0].Constructor == nil {
		t.Fatalf("expected constructor to be lowered")
	}
	if len(lowered.Classes[0].Methods) != 1 || lowered.Classes[0].Methods[0].Name != "inc" {
		t.Fatalf("unexpected lowered methods %#v", lowered.Classes[0].Methods)
	}
	first, ok := lowered.Functions[0].Body[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected first statement to be var decl, got %T", lowered.Functions[0].Body[0])
	}
	if _, ok := first.Init.(*Construct); !ok {
		t.Fatalf("expected constructor init, got %T", first.Init)
	}
	ifStmt, ok := lowered.Functions[0].Body[1].(*If)
	if !ok {
		t.Fatalf("expected second statement to be if, got %T", lowered.Functions[0].Body[1])
	}
	if _, ok := ifStmt.Then[0].(*Return); !ok {
		t.Fatalf("expected return in then branch, got %T", ifStmt.Then[0])
	}
}

func TestProgramFromASTRejectsUnsupportedYieldLoop(t *testing.T) {
	src := `
def run(values List[Int]) Int {
	for {
		x <- values
	} yield {
		x
	}
	return 0
}
`

	program := parseProgram(t, src)
	types := typecheck.Analyze(program)
	_, err := ProgramFromAST(program, types)
	if err == nil {
		t.Fatalf("expected lowering error for yield loop")
	}
}
