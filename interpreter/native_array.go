package interpreter

import "a-lang/parser"

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
