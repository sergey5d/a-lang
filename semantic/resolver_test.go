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
	let value = helper(input)
	mut acc = 0

	for item <- [1, 2, 3] {
		if item == input {
			break
		}
	}

	for {
		if item == input {
			break
		}
	}

	for {
		x <- [value],
		y <- [input]
	} yield {
		x + y
	}

	let mapper = (x Int) -> x + value
	ret mapper(input) == helper(value)
}

def helper(x Int) Int {
	ret x
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
	ret missing == 1
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
	let value = 1
	let value = 2
	ret value == 2
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
	ret value == 1
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
	ret 1 == 1
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
	ret 1 == 1
}

def run() Bool {
	ret 1 == 1
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
