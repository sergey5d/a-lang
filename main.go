package main

import (
	"a-lang/interpreter"
	"a-lang/parser"
	"a-lang/semantic"
	"a-lang/typecheck"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: a-lang <file> [run [entry [args...]]|ast]")
		os.Exit(1)
	}

	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	program, err := parser.Parse(string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	diagnostics := semantic.Analyze(program)
	typeResult := typecheck.Analyze(program)
	diagnostics = append(diagnostics, typeResult.Diagnostics...)
	if len(diagnostics) > 0 {
		for _, diagnostic := range diagnostics {
			fmt.Fprintln(os.Stderr, diagnostic.Error())
		}
		os.Exit(1)
	}

	mode := "run"
	if len(os.Args) >= 3 {
		mode = os.Args[2]
	}

	switch mode {
	case "ast":
		out, err := json.MarshalIndent(program, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal ast: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	case "run":
		entry := "run"
		var rawArgs []string
		if len(os.Args) >= 4 {
			entry = os.Args[3]
			rawArgs = os.Args[4:]
		} else {
			rawArgs = os.Args[3:3]
		}
		entryFn := findFunction(program, entry)
		if entryFn == nil {
			fmt.Fprintf(os.Stderr, "unknown entry %q\n", entry)
			os.Exit(1)
		}
		args, err := parseCLIArgs(entryFn, rawArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		in := interpreter.New(program)
		value, err := in.Call(entry, args...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if value != nil {
			fmt.Println(value)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", mode)
		os.Exit(1)
	}
}

func findFunction(program *parser.Program, name string) *parser.FunctionDecl {
	for _, fn := range program.Functions {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

func parseCLIArgs(fn *parser.FunctionDecl, raw []string) ([]interpreter.Value, error) {
	if len(raw) != len(fn.Parameters) {
		return nil, fmt.Errorf("entry %q expects %d arguments, got %d", fn.Name, len(fn.Parameters), len(raw))
	}
	args := make([]interpreter.Value, len(raw))
	for i, param := range fn.Parameters {
		value, err := parseCLIArg(param.Type, raw[i])
		if err != nil {
			return nil, fmt.Errorf("argument %d (%s): %w", i+1, param.Name, err)
		}
		args[i] = value
	}
	return args, nil
}

func parseCLIArg(ref *parser.TypeRef, raw string) (interpreter.Value, error) {
	if ref == nil {
		return nil, fmt.Errorf("missing parameter type")
	}
	switch ref.Name {
	case "Int", "Int64":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected Int, got %q", raw)
		}
		return value, nil
	case "Float", "Float64":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("expected Float, got %q", raw)
		}
		return value, nil
	case "Bool":
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("expected Bool, got %q", raw)
		}
		return value, nil
	case "String":
		return raw, nil
	case "Rune":
		runes := []rune(raw)
		if len(runes) != 1 {
			return nil, fmt.Errorf("expected Rune, got %q", raw)
		}
		return runes[0], nil
	default:
		return nil, fmt.Errorf("CLI arguments do not support type %s yet", ref.Name)
	}
}
