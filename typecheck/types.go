package typecheck

import (
	"strings"

	"a-lang/parser"
)

type TypeKind string

const (
	TypeUnknown   TypeKind = "unknown"
	TypeBuiltin   TypeKind = "builtin"
	TypeClass     TypeKind = "class"
	TypeInterface TypeKind = "interface"
	TypeParam     TypeKind = "type_param"
	TypeFunction  TypeKind = "function"
	TypeTuple     TypeKind = "tuple"
	TypeModule    TypeKind = "module"
)

type Type struct {
	Kind       TypeKind
	Name       string
	Args       []*Type
	TupleNames []string
	Signature  *Signature
}

var unknownType = &Type{Kind: TypeUnknown, Name: "<unknown>"}

func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	if t.Kind == TypeFunction && t.Signature != nil {
		parts := make([]string, len(t.Signature.Parameters))
		for i, param := range t.Signature.Parameters {
			parts[i] = param.String()
		}
		return "(" + strings.Join(parts, ", ") + ") -> " + t.Signature.ReturnType.String()
	}
	if t.Kind == TypeTuple {
		parts := make([]string, len(t.Args))
		for i, arg := range t.Args {
			if i < len(t.TupleNames) && t.TupleNames[i] != "" {
				parts[i] = t.TupleNames[i] + " " + arg.String()
			} else {
				parts[i] = arg.String()
			}
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
	if len(t.Args) == 0 {
		return t.Name
	}
	parts := make([]string, len(t.Args))
	for i, arg := range t.Args {
		parts[i] = arg.String()
	}
	return t.Name + "[" + strings.Join(parts, ", ") + "]"
}

func isUnknown(t *Type) bool {
	return t == nil || t.Kind == TypeUnknown
}

func sameType(left, right *Type) bool {
	if isUnknown(left) || isUnknown(right) {
		return true
	}
	if left.Kind != right.Kind {
		return false
	}
	if left.Kind == TypeFunction {
		if left.Signature == nil || right.Signature == nil {
			return left.Signature == right.Signature
		}
		if len(left.Signature.Parameters) != len(right.Signature.Parameters) {
			return false
		}
		for i := range left.Signature.Parameters {
			if !sameType(left.Signature.Parameters[i], right.Signature.Parameters[i]) {
				return false
			}
		}
		return sameType(left.Signature.ReturnType, right.Signature.ReturnType)
	}
	if left.Kind == TypeTuple {
		if len(left.Args) != len(right.Args) {
			return false
		}
		for i := range left.Args {
			if !sameType(left.Args[i], right.Args[i]) {
				return false
			}
		}
		return true
	}
	if left.Name != right.Name || len(left.Args) != len(right.Args) {
		return false
	}
	for i := range left.Args {
		if !sameType(left.Args[i], right.Args[i]) {
			return false
		}
	}
	return true
}

func fromTypeRef(ref *parser.TypeRef, lookup typeLookup) *Type {
	if ref == nil {
		return unknownType
	}
	if ref.ReturnType != nil {
		params := make([]*Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = fromTypeRef(param, lookup)
		}
		return &Type{
			Kind: TypeFunction,
			Name: "func",
			Signature: &Signature{
				Parameters: params,
				ReturnType: fromTypeRef(ref.ReturnType, lookup),
			},
		}
	}
	if len(ref.TupleElements) > 0 {
		args := make([]*Type, len(ref.TupleElements))
		for i, arg := range ref.TupleElements {
			args[i] = fromTypeRef(arg, lookup)
		}
		return &Type{Kind: TypeTuple, Name: "Tuple", Args: args, TupleNames: append([]string(nil), ref.TupleNames...)}
	}
	args := make([]*Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = fromTypeRef(arg, lookup)
	}
	kind := lookup.kindOf(ref.Name)
	if kind == "" {
		kind = TypeUnknown
	}
	return &Type{Kind: kind, Name: ref.Name, Args: args}
}
