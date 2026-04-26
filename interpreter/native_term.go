package interpreter

import (
	"fmt"

	"a-lang/parser"
)

func nativeTermPrint(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeTerm(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Term.print receiver mismatch", Span: span}
	}
	for _, arg := range args {
		fmt.Print(fmt.Sprint(arg))
	}
	return value, nil
}

func nativeTermPrintln(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeTerm(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Term.println receiver mismatch", Span: span}
	}
	parts := make([]any, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprint(arg)
	}
	fmt.Println(parts...)
	return value, nil
}
