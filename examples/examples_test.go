package examples_test

import (
	"a-lang/interpreter"
	"a-lang/parser"
	"a-lang/semantic"
	"a-lang/typecheck"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamples(t *testing.T) {
	paths, err := filepath.Glob("*.al")
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("expected at least one example in examples/*.al")
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Logf("running %s", filepath.Base(path))
			srcBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile returned error: %v", err)
			}
			src := string(srcBytes)
			if shouldIgnoreExample(src) {
				t.Skipf("skipping %s: marked with # IGNORE", filepath.Base(path))
			}
			expected, err := parseExpectedOutput(src)
			if err != nil {
				if err == errMissingExpectedHeader {
					t.Skipf("skipping %s: no # EXPECT header", filepath.Base(path))
				}
				t.Fatalf("parseExpectedOutput returned error: %v", err)
			}

			program, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}

			diagnostics := semantic.Analyze(program)
			typeResult := typecheck.Analyze(program)
			diagnostics = append(diagnostics, typeResult.Diagnostics...)
			if len(diagnostics) > 0 {
				var messages []string
				for _, diagnostic := range diagnostics {
					messages = append(messages, diagnostic.Error())
				}
				t.Fatalf("expected no diagnostics, got:\n%s", strings.Join(messages, "\n"))
			}

			oldStdout := os.Stdout
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("Pipe returned error: %v", err)
			}
			os.Stdout = writer

			in := interpreter.New(program)
			value, callErr := in.Call("main")

			_ = writer.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(reader)
			_ = reader.Close()

			if callErr != nil {
				t.Fatalf("Call returned error: %v", callErr)
			}

			actual := string(output)
			if value != nil {
				actual += fmt.Sprintln(value)
			}

			if normalizeExampleOutput(actual) != normalizeExampleOutput(expected) {
				t.Fatalf("unexpected output\nexpected:\n%s\nactual:\n%s", expected, actual)
			}
			t.Logf("passed %s", filepath.Base(path))
		})
	}
}

var errMissingExpectedHeader = fmt.Errorf("missing '# EXPECT:' header")

func parseExpectedOutput(src string) (string, error) {
	lines := strings.Split(src, "\n")
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "# EXPECT:" {
			start = i + 1
			break
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}
	}
	if start == -1 {
		return "", errMissingExpectedHeader
	}

	var out []string
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
		content := strings.TrimPrefix(trimmed, "#")
		if strings.HasPrefix(content, " ") {
			content = content[1:]
		}
		out = append(out, content)
	}
	return strings.Join(out, "\n"), nil
}

func normalizeExampleOutput(s string) string {
	return strings.TrimRight(s, "\n")
}

func shouldIgnoreExample(src string) bool {
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "# IGNORE" {
			return true
		}
		if !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return false
}
