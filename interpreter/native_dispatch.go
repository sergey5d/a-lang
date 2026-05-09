package interpreter

import "a-lang/parser"

type nativeMethodHandler func(in *Interpreter, receiver Value, args []Value, local *env, span parser.Span) (Value, error)

func nativeMethodHandlers() map[string]map[string]nativeMethodHandler {
	return map[string]map[string]nativeMethodHandler{
		"Printer": {
			"print":   nativePrinterPrint,
			"println": nativePrinterPrintln,
			"printf":  nativePrinterPrintf,
		},
		"OS": {
			"print":   nativeOSPrint,
			"println": nativeOSPrintln,
			"printf":  nativeOSPrintf,
			"panic":   nativeOSPanic,
		},
		"Str": {
			"size":    nativeStrSize,
			"split":   nativeStrSplit,
			"indexOf": nativeStrIndexOf,
		},
		"Array": {
			"get":          nativeArrayGet,
			"first":        nativeArrayFirst,
			"last":         nativeArrayLast,
			"map":          nativeArrayMap,
			"exists":       nativeArrayExists,
			"forAll":       nativeArrayForAll,
			"forEach":      nativeArrayForEach,
			"size":         nativeArraySize,
			"zip":          nativeArrayZip,
			"zipWithIndex": nativeArrayZipWithIndex,
		},
		"List": {
			"append":       nativeListAppend,
			"map":          nativeListMap,
			"flatMap":      nativeListFlatMap,
			"filter":       nativeListFilter,
			"fold":         nativeListFold,
			"reduce":       nativeListReduce,
			"exists":       nativeListExists,
			"forAll":       nativeListForAll,
			"forEach":      nativeListForEach,
			"sort":         nativeListSort,
			"zip":          nativeListZip,
			"zipWithIndex": nativeListZipWithIndex,
			"get":          nativeListGet,
			"head":         nativeListHead,
			"tail":         nativeListTail,
			"isEmpty":      nativeListIsEmpty,
			"remove":       nativeListRemove,
			"removeLast":   nativeListRemoveLast,
			"size":         nativeListSize,
			"iterator":     nativeListIteratorMethod,
		},
		"Iterator": {
			"hasNext": nativeIteratorHasNext,
			"next":    nativeIteratorNext,
		},
		"Option": {
			"isSet":     nativeOptionIsSet,
			"isEmpty":   nativeOptionIsEmpty,
			"isFailure": nativeOptionIsFailure,
			"get":       nativeOptionGet,
			"unwrap":    nativeOptionUnwrap,
			"getOr":     nativeOptionGetOr,
			"getOrElse": nativeOptionGetOrElse,
			"map":       nativeOptionMap,
		},
		"Result": {
			"isOk":      nativeResultIsOk,
			"isErr":     nativeResultIsErr,
			"isFailure": nativeResultIsFailure,
			"unwrap":    nativeResultUnwrap,
			"getError":  nativeResultGetError,
			"getOr":     nativeResultGetOr,
			"map":       nativeResultMap,
		},
		"Either": {
			"isLeft":    nativeEitherIsLeft,
			"isRight":   nativeEitherIsRight,
			"isFailure": nativeEitherIsFailure,
			"unwrap":    nativeEitherUnwrap,
			"getLeft":   nativeEitherGetLeft,
			"getOr":     nativeEitherGetOr,
			"map":       nativeEitherMap,
		},
		"Set": {
			"add":      nativeSetAdd,
			"iterator": nativeSetIterator,
			"map":      nativeSetMap,
			"flatMap":  nativeSetFlatMap,
			"filter":   nativeSetFilter,
			"fold":     nativeSetFold,
			"reduce":   nativeSetReduce,
			"exists":   nativeSetExists,
			"forAll":   nativeSetForAll,
			"forEach":  nativeSetForEach,
			"contains": nativeSetContains,
			"size":     nativeSetSize,
		},
		"Map": {
			"set":       nativeMapSet,
			"iterator":  nativeMapIterator,
			"map":       nativeMapMap,
			"mapValues": nativeMapMapValues,
			"flatMap":   nativeMapFlatMap,
			"filter":    nativeMapFilter,
			"fold":      nativeMapFold,
			"reduce":    nativeMapReduce,
			"exists":    nativeMapExists,
			"forAll":    nativeMapForAll,
			"forEach":   nativeMapForEach,
			"get":       nativeMapGet,
			"contains":  nativeMapContains,
			"size":      nativeMapSize,
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
	if typeName == "Array" || typeName == "Str" {
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

func asNativeResult(receiver Value) (*nativeResult, bool) {
	value, ok := receiver.(*nativeResult)
	return value, ok
}

func asNativeEither(receiver Value) (*nativeEither, bool) {
	value, ok := receiver.(*nativeEither)
	return value, ok
}

func asNativePrinter(receiver Value) (*nativePrinter, bool) {
	value, ok := receiver.(*nativePrinter)
	return value, ok
}

func asNativeOS(receiver Value) (*nativeOS, bool) {
	value, ok := receiver.(*nativeOS)
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

func boolResult(value Value, methodName string, span parser.Span) (bool, error) {
	result, ok := value.(bool)
	if !ok {
		return false, RuntimeError{Message: methodName + " function must return Bool", Span: span}
	}
	return result, nil
}

func tupleEntry(key Value, value Value) *nativeTuple {
	return &nativeTuple{items: []Value{key, value}}
}
