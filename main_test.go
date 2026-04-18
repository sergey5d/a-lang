package main

import (
	"testing"

	"a-lang/parser"
)

func TestParseCLIArgsScalars(t *testing.T) {
	fn := &parser.FunctionDecl{
		Name: "main",
		Parameters: []parser.Parameter{
			{Name: "count", Type: &parser.TypeRef{Name: "Int"}},
			{Name: "ratio", Type: &parser.TypeRef{Name: "Float"}},
			{Name: "flag", Type: &parser.TypeRef{Name: "Bool"}},
			{Name: "name", Type: &parser.TypeRef{Name: "String"}},
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
		t.Fatalf("expected String arg alice, got %#v", args[3])
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
