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
)

type Type struct {
	Kind      TypeKind
	Name      string
	Args      []*Type
	Signature *Signature
}

var unknownType = &Type{Kind: TypeUnknown, Name: "<unknown>"}

func (t *Type) String() string {
	if t == nil {
		return "<nil>"
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
	if left.Kind != right.Kind || left.Name != right.Name || len(left.Args) != len(right.Args) {
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
