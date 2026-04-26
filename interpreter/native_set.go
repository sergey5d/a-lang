package interpreter

import "a-lang/parser"

func nativeSetAdd(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.add receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "add expects 1 argument", Span: span}
	}
	key, err := nativeKey(args[0], span, local, in)
	if err != nil {
		return nil, err
	}
	if _, ok := value.keys[key]; !ok {
		value.order = append(value.order, key)
	}
	value.keys[key] = args[0]
	return value, nil
}

func nativeSetIterator(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.iterator receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "iterator expects 0 arguments", Span: span}
	}
	items := make([]Value, 0, len(value.order))
	for _, key := range value.order {
		items = append(items, value.keys[key])
	}
	return &nativeListIterator{items: items}, nil
}

func nativeSetMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	out := &nativeSet{keys: map[string]Value{}, order: []string{}}
	for _, key := range value.order {
		mapped, err := in.invokeCallableValue(args[0], []Value{value.keys[key]}, local, span)
		if err != nil {
			return nil, err
		}
		mappedKey, err := nativeKey(mapped, span, local, in)
		if err != nil {
			return nil, err
		}
		if _, exists := out.keys[mappedKey]; !exists {
			out.order = append(out.order, mappedKey)
		}
		out.keys[mappedKey] = mapped
	}
	return out, nil
}

func nativeSetFlatMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.flatMap receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "flatMap expects 1 argument", Span: span}
	}
	out := &nativeSet{keys: map[string]Value{}, order: []string{}}
	for _, key := range value.order {
		mapped, err := in.invokeCallableValue(args[0], []Value{value.keys[key]}, local, span)
		if err != nil {
			return nil, err
		}
		setValue, ok := mapped.(*nativeSet)
		if !ok {
			return nil, RuntimeError{Message: "flatMap function must return Set", Span: span}
		}
		for _, nestedKey := range setValue.order {
			nestedValue := setValue.keys[nestedKey]
			outKey, err := nativeKey(nestedValue, span, local, in)
			if err != nil {
				return nil, err
			}
			if _, exists := out.keys[outKey]; !exists {
				out.order = append(out.order, outKey)
			}
			out.keys[outKey] = nestedValue
		}
	}
	return out, nil
}

func nativeSetForEach(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.forEach receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "forEach expects 1 argument", Span: span}
	}
	for _, key := range value.order {
		if _, err := in.invokeCallableValue(args[0], []Value{value.keys[key]}, local, span); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func nativeSetContains(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.contains receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "contains expects 1 argument", Span: span}
	}
	key, err := nativeKey(args[0], span, local, in)
	if err != nil {
		return nil, err
	}
	_, ok = value.keys[key]
	return ok, nil
}

func nativeSetSize(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeSet(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Set.size receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "size expects 0 arguments", Span: span}
	}
	return int64(len(value.keys)), nil
}
