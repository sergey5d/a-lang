package interpreter

import (
	"fmt"
	"os"

	"a-lang/parser"
)

func nativePrinterPrint(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativePrinter(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Printer.print receiver mismatch", Span: span}
	}
	writer := os.Stdout
	if value.stderr {
		writer = os.Stderr
	}
	for _, arg := range args {
		if _, err := fmt.Fprint(writer, fmt.Sprint(arg)); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func nativePrinterPrintln(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativePrinter(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Printer.println receiver mismatch", Span: span}
	}
	writer := os.Stdout
	if value.stderr {
		writer = os.Stderr
	}
	parts := make([]any, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprint(arg)
	}
	if _, err := fmt.Fprintln(writer, parts...); err != nil {
		return nil, err
	}
	return nil, nil
}

func nativeOSPrint(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeOS(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native OS.print receiver mismatch", Span: span}
	}
	return nativePrinterPrint(in, value.out, args, local, span)
}

func nativeOSPrintln(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeOS(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native OS.println receiver mismatch", Span: span}
	}
	return nativePrinterPrintln(in, value.out, args, local, span)
}
