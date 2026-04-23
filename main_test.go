package main

import (
	"strings"
	"testing"

	"a-lang/parser"
	"a-lang/semantic"
)

func TestParseExpectedOutput(t *testing.T) {
	src := `# EXPECT:
# a
# b

def main() Unit {
}`

	expected, ok, err := parseExpectedOutput(src)
	if err != nil {
		t.Fatalf("parseExpectedOutput returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected # EXPECT header to be found")
	}
	if expected != "a\nb" {
		t.Fatalf("expected two-line output, got %q", expected)
	}
}

func TestParseExpectedOutputMissing(t *testing.T) {
	src := `def main() Unit {}`

	_, ok, err := parseExpectedOutput(src)
	if err != nil {
		t.Fatalf("parseExpectedOutput returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected no # EXPECT header")
	}
}

func TestParseCLIArgsScalars(t *testing.T) {
	fn := &parser.FunctionDecl{
		Name: "main",
		Parameters: []parser.Parameter{
			{Name: "count", Type: &parser.TypeRef{Name: "Int"}},
			{Name: "ratio", Type: &parser.TypeRef{Name: "Float"}},
			{Name: "flag", Type: &parser.TypeRef{Name: "Bool"}},
			{Name: "name", Type: &parser.TypeRef{Name: "Str"}},
			{Name: "mark", Type: &parser.TypeRef{Name: "Rune"}},
		},
	}

	args, err := parseCLIArgs(fn, []string{"5", "1.5", "true", "alice", "x"})
	if err != nil {
		t.Fatalf("parseCLIArgs returned error: %v", err)
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
	if args[0] != int64(5) {
		t.Fatalf("expected Int arg 5, got %#v", args[0])
	}
	if args[1] != float64(1.5) {
		t.Fatalf("expected Float arg 1.5, got %#v", args[1])
	}
	if args[2] != true {
		t.Fatalf("expected Bool arg true, got %#v", args[2])
	}
	if args[3] != "alice" {
		t.Fatalf("expected Str arg alice, got %#v", args[3])
	}
	if args[4] != rune('x') {
		t.Fatalf("expected Rune arg 'x', got %#v", args[4])
	}
}

func TestParseCLIArgsRejectsUnsupportedType(t *testing.T) {
	fn := &parser.FunctionDecl{
		Name: "main",
		Parameters: []parser.Parameter{
			{Name: "items", Type: &parser.TypeRef{Name: "List", Arguments: []*parser.TypeRef{{Name: "Int"}}}},
		},
	}

	_, err := parseCLIArgs(fn, []string{"[1,2]"})
	if err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestParseCLIArgsRejectsWrongArity(t *testing.T) {
	fn := &parser.FunctionDecl{
		Name: "main",
		Parameters: []parser.Parameter{
			{Name: "count", Type: &parser.TypeRef{Name: "Int"}},
		},
	}

	_, err := parseCLIArgs(fn, nil)
	if err == nil {
		t.Fatalf("expected arity error")
	}
}

func TestFormatDiagnosticIncludesSourceExcerpt(t *testing.T) {
	src := "x = 1\nmissing\nz = 3\n"
	diagnostic := semantic.Diagnostic{
		Code:    "undefined_name",
		Message: "undefined name 'missing'",
		Span: parser.Span{
			Start: parser.Position{Line: 2, Column: 1},
			End:   parser.Position{Line: 2, Column: 8},
		},
	}

	formatted := formatDiagnostic(diagnostic, src)
	if !strings.Contains(formatted, "undefined_name at 2:1: undefined name 'missing'") {
		t.Fatalf("expected headline in formatted diagnostic, got %q", formatted)
	}
	if !strings.Contains(formatted, "1 │ x = 1") {
		t.Fatalf("expected previous line in formatted diagnostic, got %q", formatted)
	}
	if !strings.Contains(formatted, "2 │ missing") {
		t.Fatalf("expected target line in formatted diagnostic, got %q", formatted)
	}
	if !strings.Contains(formatted, "│ ^^^^^^^") {
		t.Fatalf("expected caret underline in formatted diagnostic, got %q", formatted)
	}
}
