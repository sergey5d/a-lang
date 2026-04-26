package interpreter

import "a-lang/parser"

func nativeOptionIsSet(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeOption(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.isSet receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isSet expects 0 arguments", Span: span}
	}
	return value.set, nil
}

func nativeOptionIsEmpty(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeOption(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.isEmpty receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isEmpty expects 0 arguments", Span: span}
	}
	return !value.set, nil
}

func nativeOptionGet(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeOption(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.get receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "get expects 0 arguments", Span: span}
	}
	if !value.set {
		return nil, RuntimeError{Message: "Option has no value", Span: span}
	}
	return value.value, nil
}

func nativeOptionGetOr(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeOption(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.getOr receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "getOr expects 1 argument", Span: span}
	}
	if value.set {
		return value.value, nil
	}
	return args[0], nil
}
