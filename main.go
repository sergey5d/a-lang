package main

import (
	"encoding/json"
	"fmt"
	"os"

	"a-lang/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: a-lang-parser <file>")
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

	out, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal ast: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(out))
}
