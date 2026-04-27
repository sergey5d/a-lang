package interpreter

import "a-lang/parser"

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
