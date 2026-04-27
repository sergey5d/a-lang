package interpreter

import "a-lang/parser"

func nativeResultIsOk(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.isOk receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isOk expects 0 arguments", Span: span}
	}
	return value.ok, nil
}

func nativeResultIsErr(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.isErr receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isErr expects 0 arguments", Span: span}
	}
	return !value.ok, nil
}

func nativeResultIsFailure(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.isFailure receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isFailure expects 0 arguments", Span: span}
	}
	return !value.ok, nil
}

func nativeResultUnwrap(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.unwrap receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "unwrap expects 0 arguments", Span: span}
	}
	if !value.ok {
		return nil, RuntimeError{Message: "Result has no success value", Span: span}
	}
	return value.value, nil
}

func nativeResultGetError(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.getError receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "getError expects 0 arguments", Span: span}
	}
	if value.ok {
		return nil, RuntimeError{Message: "Result has no error value", Span: span}
	}
	return value.err, nil
}

func nativeResultGetOr(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.getOr receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "getOr expects 1 argument", Span: span}
	}
	if value.ok {
		return value.value, nil
	}
	return args[0], nil
}

func nativeResultMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeResult(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Result.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	if !value.ok {
		return in.constructStdlibResult(nil, value.err, false, local)
	}
	mapped, err := in.invokeCallableValue(args[0], []Value{value.value}, local, span)
	if err != nil {
		return nil, err
	}
	return in.constructStdlibResult(mapped, nil, true, local)
}
