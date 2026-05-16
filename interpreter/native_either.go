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

func nativeEitherExpectRight(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.expectRight receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "expectRight expects 0 arguments", Span: span}
	}
	if !value.rightSet {
		return nil, RuntimeError{Message: "Either has no right value", Span: span}
	}
	return value.right, nil
}

func nativeEitherExpectLeft(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.expectLeft receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "expectLeft expects 0 arguments", Span: span}
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

func nativeEitherMapLeft(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.mapLeft receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "mapLeft expects 1 argument", Span: span}
	}
	if value.rightSet {
		return in.constructStdlibEither(nil, value.right, true, local, span)
	}
	mapped, err := in.invokeCallableValue(args[0], []Value{value.left}, local, span)
	if err != nil {
		return nil, err
	}
	return in.constructStdlibEither(mapped, nil, false, local, span)
}

func nativeEitherFlatMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.flatMap receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "flatMap expects 1 argument", Span: span}
	}
	if !value.rightSet {
		return in.constructStdlibEither(value.left, nil, false, local, span)
	}
	return in.invokeCallableValue(args[0], []Value{value.right}, local, span)
}

func nativeEitherToOption(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.toOption receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "toOption expects 0 arguments", Span: span}
	}
	if value.rightSet {
		return in.constructStdlibOption(value.right, true, local, span)
	}
	return in.constructStdlibOption(nil, false, local, span)
}

func nativeEitherToResult(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeEither(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Either.toResult receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "toResult expects 0 arguments", Span: span}
	}
	if value.rightSet {
		return in.constructStdlibResult(value.right, nil, true, local, span)
	}
	return in.constructStdlibResult(nil, value.left, false, local, span)
}
