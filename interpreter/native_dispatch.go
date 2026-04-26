package interpreter

import "a-lang/parser"

type nativeMethodHandler func(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error)

func nativeMethodHandlers() map[string]map[string]nativeMethodHandler {
	return map[string]map[string]nativeMethodHandler{
		"Array": {
			"size": nativeArraySize,
		},
		"List": {
			"append":   nativeListAppend,
			"map":      nativeListMap,
			"flatMap":  nativeListFlatMap,
			"forEach":  nativeListForEach,
			"sort":     nativeListSort,
			"get":      nativeListGet,
			"head":     nativeListHead,
			"tail":     nativeListTail,
			"remove":   nativeListRemove,
			"size":     nativeListSize,
			"iterator": nativeListIteratorMethod,
		},
		"Iterator": {
			"hasNext": nativeIteratorHasNext,
			"next":    nativeIteratorNext,
		},
		"Option": {
			"isSet":   nativeOptionIsSet,
			"isEmpty": nativeOptionIsEmpty,
			"get":     nativeOptionGet,
			"getOr":   nativeOptionGetOr,
		},
		"Set": {
			"add":      nativeSetAdd,
			"iterator": nativeSetIterator,
			"map":      nativeSetMap,
			"flatMap":  nativeSetFlatMap,
			"forEach":  nativeSetForEach,
			"contains": nativeSetContains,
			"size":     nativeSetSize,
		},
		"Map": {
			"set":      nativeMapSet,
			"iterator": nativeMapIterator,
			"map":      nativeMapMap,
			"flatMap":  nativeMapFlatMap,
			"forEach":  nativeMapForEach,
			"get":      nativeMapGet,
			"contains": nativeMapContains,
			"size":     nativeMapSize,
		},
		"Term": {
			"print":   nativeTermPrint,
			"println": nativeTermPrintln,
		},
	}
}

func lookupNativeMethodHandler(receiver Value, name string) (nativeMethodHandler, bool) {
	typeName, ok := nativeBuiltinTypeName(receiver)
	if !ok {
		return nil, false
	}
	methods, ok := nativeMethodHandlers()[typeName]
	if !ok {
		return nil, false
	}
	handler, ok := methods[name]
	if !ok {
		return nil, false
	}
	if typeName == "Array" {
		return handler, true
	}
	if _, ok := lookupBuiltinMethodDescriptor(typeName, name); !ok {
		return nil, false
	}
	return handler, true
}

func asNativeList(receiver Value) (*nativeList, bool) {
	value, ok := receiver.(*nativeList)
	return value, ok
}

func asNativeListIterator(receiver Value) (*nativeListIterator, bool) {
	value, ok := receiver.(*nativeListIterator)
	return value, ok
}

func asNativeArray(receiver Value) (*nativeArray, bool) {
	value, ok := receiver.(*nativeArray)
	return value, ok
}

func asNativeOption(receiver Value) (*nativeOption, bool) {
	value, ok := receiver.(*nativeOption)
	return value, ok
}

func asNativeSet(receiver Value) (*nativeSet, bool) {
	value, ok := receiver.(*nativeSet)
	return value, ok
}

func asNativeMap(receiver Value) (*nativeMap, bool) {
	value, ok := receiver.(*nativeMap)
	return value, ok
}

func asNativeTerm(receiver Value) (*nativeTerm, bool) {
	value, ok := receiver.(*nativeTerm)
	return value, ok
}
