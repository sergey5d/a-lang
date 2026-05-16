package interpreter

import "a-lang/parser"

func nativeEitherIsLeft(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.isLeft receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isLeft expects 0 arguments", Span: span}
	}
	return !value.rightSet, nil
}

func nativeEitherIsRight(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.isRight receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isRight expects 0 arguments", Span: span}
	}
	return value.rightSet, nil
}

func nativeEitherIsFailure(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.isFailure receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isFailure expects 0 arguments", Span: span}
	}
	return !value.rightSet, nil
}

func nativeEitherUnwrap(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.unwrap receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "unwrap expects 0 arguments", Span: span}
	}
	if !value.rightSet {
		return nil, RuntimeError{Message: "Either has no right value", Span: span}
	}
	return value.right, nil
}

func nativeEitherGetLeft(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.getLeft receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "getLeft expects 0 arguments", Span: span}
	}
	if value.rightSet {
		return nil, RuntimeError{Message: "Either has no left value", Span: span}
	}
	return value.left, nil
}

func nativeEitherGetOr(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.getOr receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "getOr expects 1 argument", Span: span}
	}
	if value.rightSet {
		return value.right, nil
	}
	return args[0], nil
}

func nativeEitherMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	if !value.rightSet {
		return in.constructStdlibEither(value.left, nil, false, local, span)
	}
	mapped, err := in.invokeCallableValue(args[0], []Value{value.right}, local, span)
	if err != nil {
		return nil, err
	}
	return in.constructStdlibEither(nil, mapped, true, local, span)
}
