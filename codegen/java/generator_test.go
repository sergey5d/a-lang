package java

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func lowerProgram(t *testing.T, src string) *lower.Program {
	t.Helper()
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
	return lowered
}

func TestGenerateCompilesWithJavac(t *testing.T) {
	src := `
class Counter {
	hidden var count Int
}

impl Counter {
	def init(count Int) {
		this.count = count
	}

	def bump(delta Int) Int {
		this.count += delta
		return this.count
	}
}

object MathBox {
	hidden base Int = 2

	def add(x Int) Int = x + this.base
}

seed Int = 3

def accumulate(values Array[Int]) Int {
	var total Int = 0
	for value <- values {
		total += value
	}
	return total
}

def run() Int {
	counter Counter = Counter(seed)
	values Array[Int] = Array(1, 2, 3)
	values[0] := values[0] + 1
	if seed > 0 {
		return MathBox.add(counter.bump(accumulate(values)))
	}
	return 0
}
`

	lowered := lowerProgram(t, src)
	source, err := GenerateForPackage(lowered, "")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	text := string(source)
	if !strings.Contains(text, "public final class Pkg_Default") {
		t.Fatalf("expected package holder class in generated Java, got:\n%s", text)
	}
	if !strings.Contains(text, "public static void main(String[] args)") {
		t.Fatalf("expected java main bridge in generated Java, got:\n%s", text)
	}
	if !strings.Contains(text, "final class Counter") || !strings.Contains(text, "final class Object_MathBox") {
		t.Fatalf("expected object backing class in generated Java, got:\n%s", text)
	}
	if !strings.Contains(text, "new long[] {1L, 2L, 3L}") {
		t.Fatalf("expected Array(...) lowering in generated Java, got:\n%s", text)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "Pkg_Default.java")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatalf("write generated source: %v", err)
	}

	cmd := exec.Command("javac", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("javac failed: %v\n%s\n%s", err, text, string(output))
	}
}

func TestGenerateRejectsUnsupportedListLiteral(t *testing.T) {
	src := `
def run() Int {
	values = [1, 2, 3]
	return 1
}
`

	lowered := lowerProgram(t, src)
	_, err := GenerateForPackage(lowered, "")
	if err == nil {
		t.Fatalf("expected Generate to reject list literals")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported-expression error, got %v", err)
	}
}

func TestOutputPathUsesPackageStructure(t *testing.T) {
	path := OutputPath("bin/java/src", "model/pubdemo")
	expected := filepath.Join("bin/java/src", "model", "pubdemo", "Pkg_Model_pubdemo.java")
	if path != expected {
		t.Fatalf("expected output path %q, got %q", expected, path)
	}
}
