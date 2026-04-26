package interpreter

import "a-lang/parser"

func nativeMapSet(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.set receiver mismatch", Span: span}
	}
	if len(args) != 2 {
		return nil, RuntimeError{Message: "set expects 2 arguments", Span: span}
	}
	key, err := nativeKey(args[0], span, local, in)
	if err != nil {
		return nil, err
	}
	if _, ok := value.items[key]; !ok {
		value.order = append(value.order, key)
		value.keys[key] = args[0]
	}
	value.items[key] = args[1]
	return value, nil
}

func nativeMapIterator(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.iterator receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "iterator expects 0 arguments", Span: span}
	}
	items := make([]Value, 0, len(value.order))
	for _, key := range value.order {
		items = append(items, &nativeTuple{items: []Value{value.keys[key], value.items[key]}})
	}
	return &nativeListIterator{items: items}, nil
}

func nativeMapMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	out := &nativeList{items: make([]Value, 0, len(value.order))}
	for _, key := range value.order {
		mapped, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		out.items = append(out.items, mapped)
	}
	return out, nil
}

func nativeMapFlatMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.flatMap receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "flatMap expects 1 argument", Span: span}
	}
	out := &nativeList{items: []Value{}}
	for _, key := range value.order {
		mapped, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		listValue, ok := mapped.(*nativeList)
		if !ok {
			return nil, RuntimeError{Message: "flatMap function must return List", Span: span}
		}
		out.items = append(out.items, listValue.items...)
	}
	return out, nil
}

func nativeMapForEach(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.forEach receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "forEach expects 1 argument", Span: span}
	}
	for _, key := range value.order {
		if _, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func nativeMapGet(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.get receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "get expects 1 argument", Span: span}
	}
	key, err := nativeKey(args[0], span, local, in)
	if err != nil {
		return nil, err
	}
	result, ok := value.items[key]
	if !ok {
		return in.constructStdlibOption(nil, false, local, span)
	}
	return in.constructStdlibOption(result, true, local, span)
}

func nativeMapContains(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.contains receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "contains expects 1 argument", Span: span}
	}
	key, err := nativeKey(args[0], span, local, in)
	if err != nil {
		return nil, err
	}
	_, ok = value.items[key]
	return ok, nil
}

func nativeMapSize(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.size receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "size expects 0 arguments", Span: span}
	}
	return int64(len(value.items)), nil
}
