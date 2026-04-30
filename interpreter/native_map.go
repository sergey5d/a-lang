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

func nativeMapMapValues(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.mapValues receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "mapValues expects 1 argument", Span: span}
	}
	out := &nativeMap{items: map[string]Value{}, keys: map[string]Value{}, order: make([]string, 0, len(value.order))}
	for _, key := range value.order {
		mapped, err := in.invokeCallableValue(args[0], []Value{value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		out.order = append(out.order, key)
		out.keys[key] = value.keys[key]
		out.items[key] = mapped
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

func nativeMapFilter(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.filter receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "filter expects 1 argument", Span: span}
	}
	out := &nativeMap{items: map[string]Value{}, keys: map[string]Value{}, order: []string{}}
	for _, key := range value.order {
		matched, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		keep, err := boolResult(matched, "filter", span)
		if err != nil {
			return nil, err
		}
		if keep {
			out.order = append(out.order, key)
			out.keys[key] = value.keys[key]
			out.items[key] = value.items[key]
		}
	}
	return out, nil
}

func nativeMapFold(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.fold receiver mismatch", Span: span}
	}
	if len(args) != 2 {
		return nil, RuntimeError{Message: "fold expects 2 arguments", Span: span}
	}
	acc := args[0]
	for _, key := range value.order {
		next, err := in.invokeCallableValue(args[1], []Value{acc, value.keys[key], value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		acc = next
	}
	return acc, nil
}

func nativeMapReduce(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.reduce receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "reduce expects 1 argument", Span: span}
	}
	if len(value.order) == 0 {
		return in.constructStdlibOption(nil, false, local, span)
	}
	acc := tupleEntry(value.keys[value.order[0]], value.items[value.order[0]])
	for _, key := range value.order[1:] {
		next, err := in.invokeCallableValue(args[0], []Value{acc.items[0], acc.items[1], value.keys[key], value.items[key]}, local, span)
		if err != nil {
			return nil, err
		}
		tuple, ok := next.(*nativeTuple)
		if !ok || len(tuple.items) != 2 {
			return nil, RuntimeError{Message: "reduce function must return a (K, V) tuple", Span: span}
		}
		acc = tuple
	}
	return in.constructStdlibOption(acc, true, local, span)
}

func nativeMapExists(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.exists receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "exists expects 1 argument", Span: span}
	}
	for _, key := range value.order {
		matched, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span)
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

func nativeMapForAll(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeMap(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Map.forAll receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "forAll expects 1 argument", Span: span}
	}
	for _, key := range value.order {
		matched, err := in.invokeCallableValue(args[0], []Value{value.keys[key], value.items[key]}, local, span)
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
