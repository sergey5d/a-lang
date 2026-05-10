package interpreter

import "a-lang/parser"

func optionState(receiver Value) (bool, Value, bool) {
	switch value := receiver.(type) {
	case *nativeOption:
		return value.set, value.value, true
	case *instance:
		if value.class.Name != "Option" {
			return false, nil, false
		}
		switch value.caseName {
		case "Some":
			return true, value.fields["value"], true
		case "None":
			return false, nil, true
		default:
			return false, nil, false
		}
	default:
		return false, nil, false
	}
}

func nativeOptionIsSet(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	set, _, ok := optionState(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.isSet receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isSet expects 0 arguments", Span: span}
	}
	return set, nil
}

func nativeOptionIsEmpty(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	set, _, ok := optionState(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.isEmpty receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "isEmpty expects 0 arguments", Span: span}
	}
	return !set, nil
}

func nativeOptionExpect(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	set, value, ok := optionState(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.expect receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "expect expects 0 arguments", Span: span}
	}
	if !set {
		return nil, RuntimeError{Message: "Option has no value", Span: span}
	}
	return value, nil
}

func nativeOptionGetOr(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	set, value, ok := optionState(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.getOr receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "getOr expects 1 argument", Span: span}
	}
	if set {
		return value, nil
	}
	return args[0], nil
}

func nativeOptionGetOrElse(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	return nativeOptionGetOr(in, receiver, args, local, span)
}

func nativeOptionMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	set, value, ok := optionState(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Option.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	if !set {
		return in.constructStdlibOption(nil, false, local, span)
	}
	mapped, err := in.invokeCallableValue(args[0], []Value{value}, local, span)
	if err != nil {
		return nil, err
	}
	return in.constructStdlibOption(mapped, true, local, span)
}
