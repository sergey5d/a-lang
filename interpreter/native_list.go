package interpreter

import (
	"sort"

	"a-lang/parser"
)

func nativeListAppend(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.append receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "append expects 1 argument", Span: span}
	}
	value.items = append(value.items, args[0])
	return value, nil
}

func nativeListMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.map receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "map expects 1 argument", Span: span}
	}
	out := &nativeList{items: make([]Value, 0, len(value.items))}
	for _, item := range value.items {
		mapped, err := in.invokeCallableValue(args[0], []Value{item}, local, span)
		if err != nil {
			return nil, err
		}
		out.items = append(out.items, mapped)
	}
	return out, nil
}

func nativeListFlatMap(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.flatMap receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "flatMap expects 1 argument", Span: span}
	}
	out := &nativeList{items: []Value{}}
	for _, item := range value.items {
		mapped, err := in.invokeCallableValue(args[0], []Value{item}, local, span)
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

func nativeListForEach(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.forEach receiver mismatch", Span: span}
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

func nativeListSort(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.sort receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "sort expects 1 argument", Span: span}
	}
	ordering := args[0]
	var sortErr error
	sort.SliceStable(value.items, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		compared, err := in.invokeMethod(ordering, "compare", []Value{value.items[i], value.items[j]}, local, span)
		if err != nil {
			sortErr = err
			return false
		}
		result, ok := compared.(int64)
		if !ok {
			sortErr = RuntimeError{Message: "Ordering.compare must return Int", Span: span}
			return false
		}
		return result < 0
	})
	if sortErr != nil {
		return nil, sortErr
	}
	return value, nil
}

func nativeListGet(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.get receiver mismatch", Span: span}
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

func nativeListHead(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.head receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "head expects 0 arguments", Span: span}
	}
	if len(value.items) == 0 {
		return in.constructStdlibOption(nil, false, local, span)
	}
	return in.constructStdlibOption(value.items[0], true, local, span)
}

func nativeListTail(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.tail receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "tail expects 0 arguments", Span: span}
	}
	if len(value.items) <= 1 {
		return &nativeList{items: []Value{}}, nil
	}
	return &nativeList{items: append([]Value(nil), value.items[1:]...)}, nil
}

func nativeListRemove(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.remove receiver mismatch", Span: span}
	}
	if len(args) != 1 {
		return nil, RuntimeError{Message: "remove expects 1 argument", Span: span}
	}
	index, ok := args[0].(int64)
	if !ok {
		return nil, RuntimeError{Message: "remove index must be Int", Span: span}
	}
	if index < 0 || index >= int64(len(value.items)) {
		return in.constructStdlibOption(nil, false, local, span)
	}
	removed := value.items[index]
	value.items = append(value.items[:index], value.items[index+1:]...)
	return in.constructStdlibOption(removed, true, local, span)
}

func nativeListSize(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.size receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "size expects 0 arguments", Span: span}
	}
	return int64(len(value.items)), nil
}

func nativeListIteratorMethod(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeList(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native List.iterator receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "iterator expects 0 arguments", Span: span}
	}
	return &nativeListIterator{items: append([]Value(nil), value.items...)}, nil
}
