package interpreter

import "a-lang/parser"

func nativeArrayGet(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.get receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "get expects 1 argument", Span: span}
	}
	index, ok := args[0].(int64)
	if !ok {
		return nil, RuntimeError{Message: "get index must be Int", Span: span}
	}
	if index < 0 || index >= int64(len(value.items)) {
		return in.constructStdlibOption(nil, false, local, span)
	}
	return in.constructStdlibOption(value.items[index], true, local, span)
}

func nativeArrayFirst(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.first receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "first expects 0 arguments", Span: span}
	}
	if len(value.items) == 0 {
		return in.constructStdlibOption(nil, false, local, span)
	}
	return in.constructStdlibOption(value.items[0], true, local, span)
}

func nativeArrayLast(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.last receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "last expects 0 arguments", Span: span}
	}
	if len(value.items) == 0 {
		return in.constructStdlibOption(nil, false, local, span)
	}
	return in.constructStdlibOption(value.items[len(value.items)-1], true, local, span)
}

func nativeArrayMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	out := &nativeArray{items: make([]Value, len(value.items))}
	for i, item := range value.items {
		mapped, err := in.invokeCallableValue(args[0], []Value{item}, local, span)
		if err != nil {
			return nil, err
		}
		out.items[i] = mapped
	}
	return out, nil
}

func nativeArrayExists(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.exists receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "exists expects 1 argument", Span: span}
	}
	for _, item := range value.items {
		matched, err := in.invokeCallableValue(args[0], []Value{item}, local, span)
		if err != nil {
			return nil, err
		}
		keep, err := boolResult(matched, "exists", span)
		if err != nil {
			return nil, err
		}
		if keep {
			return true, nil
		}
	}
	return false, nil
}

func nativeArrayForAll(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.forAll receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "forAll expects 1 argument", Span: span}
	}
	for _, item := range value.items {
		matched, err := in.invokeCallableValue(args[0], []Value{item}, local, span)
		if err != nil {
			return nil, err
		}
		keep, err := boolResult(matched, "forAll", span)
		if err != nil {
			return nil, err
		}
		if !keep {
			return false, nil
		}
	}
	return true, nil
}

func nativeArrayForEach(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.forEach receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "forEach expects 1 argument", Span: span}
	}
	for _, item := range value.items {
		if _, err := in.invokeCallableValue(args[0], []Value{item}, local, span); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func nativeArraySize(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.size receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "size expects 0 arguments", Span: span}
	}
	return int64(len(value.items)), nil
}

func nativeArrayZip(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.zip receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "zip expects 1 argument", Span: span}
	}
	other, ok := args[0].(*nativeArray)
	if !ok {
		return nil, RuntimeError{Message: "zip expects Array argument", Span: span}
	}
	limit := len(value.items)
	if len(other.items) < limit {
		limit = len(other.items)
	}
	out := &nativeArray{items: make([]Value, limit)}
	for i := 0; i < limit; i++ {
		out.items[i] = &nativeTuple{items: []Value{value.items[i], other.items[i]}}
	}
	return out, nil
}

func nativeArrayZipWithIndex(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeArray(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Array.zipWithIndex receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "zipWithIndex expects 0 arguments", Span: span}
	}
	out := &nativeArray{items: make([]Value, len(value.items))}
	for i, item := range value.items {
		out.items[i] = &nativeTuple{items: []Value{item, int64(i)}}
	}
	return out, nil
}
