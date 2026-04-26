package interpreter

import "a-lang/parser"

func nativeIteratorHasNext(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeListIterator(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Iterator.hasNext receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "hasNext expects 0 arguments", Span: span}
	}
	return value.index < len(value.items), nil
}

func nativeIteratorNext(_ *Interpreter, receiver Value, args []Value, _ *env, span parser.Span) (Value, error) {
	value, ok := asNativeListIterator(receiver)
	if !ok {
		return nil, RuntimeError{Message: "native Iterator.next receiver mismatch", Span: span}
	}
	if len(args) != 0 {
		return nil, RuntimeError{Message: "next expects 0 arguments", Span: span}
	}
	if value.index >= len(value.items) {
		return nil, RuntimeError{Message: "iterator exhausted", Span: span}
	}
	item := value.items[value.index]
	value.index++
	return item, nil
}
