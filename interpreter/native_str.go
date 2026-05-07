package interpreter

import (
	"strings"

	"a-lang/parser"
)

func nativeStrSize(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := receiver.(string)
	if !ok {
		return nil, RuntimeError{Message: "native Str.size receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "size expects 0 arguments", Span: span}
	}
	return int64(len([]rune(value))), nil
}

func nativeStrSplit(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := receiver.(string)
	if !ok {
		return nil, RuntimeError{Message: "native Str.split receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "split expects 1 argument", Span: span}
	}
	separator, ok := args[0].(string)
	if !ok {
		return nil, RuntimeError{Message: "split expects separator of type Str", Span: span}
	}
	parts := strings.Split(value, separator)
	items := make([]Value, len(parts))
	for i, part := range parts {
		items[i] = part
	}
	return &nativeArray{items: items}, nil
}

func nativeStrIndexOf(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := receiver.(string)
	if !ok {
		return nil, RuntimeError{Message: "native Str.indexOf receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "indexOf expects 1 argument", Span: span}
	}
	part, ok := args[0].(string)
	if !ok {
		return nil, RuntimeError{Message: "indexOf expects substring of type Str", Span: span}
	}
	byteIndex := strings.Index(value, part)
	if byteIndex < 0 {
		return int64(-1), nil
	}
	return int64(len([]rune(value[:byteIndex]))), nil
}
