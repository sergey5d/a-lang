package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

// parameterBuilder converts parser parameters and type parameters into typed ones.
type parameterBuilder struct {
	ctx   *buildContext
	types *typeRefBuilder
}

// buildParameters converts parser parameters into typed parameters with symbols.
func (b *parameterBuilder) buildParameters(params []parser.Parameter) []Parameter {
	out := make([]Parameter, len(params))
	for i, param := range params {
		out[i] = Parameter{
			Name:   param.Name,
			Type:   b.types.BuildType(param.Type),
			Symbol: b.ctx.newSymbol(SymbolParameter, param.Name, "", param.Span),
			Span:   param.Span,
		}
	}
	return out
}

// buildTypeParameters converts parser type parameters into typed type parameters.
func (b *parameterBuilder) buildTypeParameters(params []parser.TypeParameter) []TypeParameter {
	out := make([]TypeParameter, len(params))
	for i, param := range params {
		out[i] = TypeParameter{Name: param.Name, Span: param.Span}
	}
	return out
}

// typeRefBuilder converts parser type references and resolves semantic call targets.
type typeRefBuilder struct {
	ctx *buildContext
}

// Build satisfies Builder for parser type refs.
func (b *typeRefBuilder) Build(ref *parser.TypeRef) (*typecheck.Type, error) {
	return b.BuildType(ref), nil
}

// BuildType converts a parser type reference into a semantic type.
func (b *typeRefBuilder) BuildType(ref *parser.TypeRef) *typecheck.Type {
	if ref == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = b.BuildType(param)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: b.BuildType(ref.ReturnType),
			},
		}
	}
	if len(ref.TupleElements) > 0 {
		args := make([]*typecheck.Type, len(ref.TupleElements))
		for i, arg := range ref.TupleElements {
			args[i] = b.BuildType(arg)
		}
		return &typecheck.Type{Kind: typecheck.TypeTuple, Name: "Tuple", Args: args, TupleNames: append([]string(nil), ref.TupleNames...)}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = b.BuildType(arg)
	}
	return &typecheck.Type{Kind: b.kindOf(ref.Name), Name: ref.Name, Args: args}
}

// kindOf classifies a type name using the known declarations in context.
func (b *typeRefBuilder) kindOf(name string) typecheck.TypeKind {
	switch name {
	case "List", "Map", "Set", "Term", "Eq", "Option":
		return typecheck.TypeInterface
	}
	switch name {
	case "Int", "Float", "Bool", "String", "Rune", "Decimal", "Array", "Unit":
		return typecheck.TypeBuiltin
	}
	if _, ok := b.ctx.classes[name]; ok {
		return typecheck.TypeClass
	}
	if _, ok := b.ctx.interfaces[name]; ok {
		return typecheck.TypeInterface
	}
	return typecheck.TypeUnknown
}

// lookupMethodSymbol finds the stable symbol assigned to a specific method declaration.
func (b *typeRefBuilder) lookupMethodSymbol(className string, method *parser.MethodDecl) SymbolRef {
	for _, candidate := range b.ctx.methodSymbols[className][method.Name] {
		if candidate.decl == method {
			return candidate.symbol
		}
	}
	return SymbolRef{}
}

// resolveFieldSymbol resolves a class field symbol from a receiver type and name.
func (b *typeRefBuilder) resolveFieldSymbol(receiverType *typecheck.Type, name string) *SymbolRef {
	if receiverType == nil || receiverType.Kind != typecheck.TypeClass {
		return nil
	}
	if fields, ok := b.ctx.fieldSymbols[receiverType.Name]; ok {
		if symbol, ok := fields[name]; ok {
			return &symbol
		}
	}
	return nil
}

// resolveConstructorSymbol chooses the constructor symbol matching lowered argument types.
func (b *typeRefBuilder) resolveConstructorSymbol(className string, args []Expr) *SymbolRef {
	class, ok := b.ctx.classes[className]
	if !ok {
		return nil
	}
	subst := b.substForClass(class, nil)
	for _, candidates := range b.ctx.methodSymbols[className] {
		for _, candidate := range candidates {
			if !candidate.decl.Constructor {
				continue
			}
			if b.methodMatches(candidate.decl, subst, args) {
				symbol := candidate.symbol
				return &symbol
			}
		}
	}
	return nil
}

// resolveMethodTarget resolves a method target symbol and dispatch kind for a call.
func (b *typeRefBuilder) resolveMethodTarget(receiverType *typecheck.Type, name string, args []Expr) (*SymbolRef, CallDispatch) {
	if receiverType == nil {
		return nil, DispatchStatic
	}
	switch receiverType.Kind {
	case typecheck.TypeClass:
		class, ok := b.ctx.classes[receiverType.Name]
		if !ok {
			return nil, DispatchStatic
		}
		subst := b.substForClass(class, receiverType.Args)
		for _, candidate := range b.ctx.methodSymbols[receiverType.Name][name] {
			if b.methodMatches(candidate.decl, subst, args) {
				symbol := candidate.symbol
				return &symbol, DispatchStatic
			}
		}
	case typecheck.TypeInterface:
		if iface, ok := b.ctx.interfaces[receiverType.Name]; ok {
			for _, method := range iface.Methods {
				if method.Name == name {
					symbol := b.ctx.newSymbol(SymbolMethod, method.Name, iface.Name, method.Span)
					return &symbol, DispatchVirtual
				}
			}
		}
	}
	return nil, DispatchStatic
}

// methodMatches checks whether a method declaration matches the provided argument types.
func (b *typeRefBuilder) methodMatches(method *parser.MethodDecl, subst map[string]*typecheck.Type, args []Expr) bool {
	if len(method.Parameters) != len(args) {
		return false
	}
	for i, param := range method.Parameters {
		paramType := b.instantiateTypeRef(param.Type, subst)
		if !sameType(paramType, args[i].GetType()) {
			return false
		}
	}
	return true
}

// substForClass builds a substitution map for a generic class instance.
func (b *typeRefBuilder) substForClass(class *parser.ClassDecl, args []*typecheck.Type) map[string]*typecheck.Type {
	if len(class.TypeParameters) == 0 {
		return nil
	}
	subst := map[string]*typecheck.Type{}
	for i, param := range class.TypeParameters {
		if i < len(args) && args[i] != nil {
			subst[param.Name] = args[i]
			continue
		}
		subst[param.Name] = &typecheck.Type{Kind: typecheck.TypeParam, Name: param.Name}
	}
	return subst
}

// instantiateTypeRef applies a type-parameter substitution to a parser type reference.
func (b *typeRefBuilder) instantiateTypeRef(ref *parser.TypeRef, subst map[string]*typecheck.Type) *typecheck.Type {
	if ref == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = b.instantiateTypeRef(param, subst)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: b.instantiateTypeRef(ref.ReturnType, subst),
			},
		}
	}
	if len(ref.TupleElements) > 0 {
		args := make([]*typecheck.Type, len(ref.TupleElements))
		for i, arg := range ref.TupleElements {
			args[i] = b.instantiateTypeRef(arg, subst)
		}
		return &typecheck.Type{Kind: typecheck.TypeTuple, Name: "Tuple", Args: args, TupleNames: append([]string(nil), ref.TupleNames...)}
	}
	if subst != nil {
		if resolved, ok := subst[ref.Name]; ok && len(ref.Arguments) == 0 {
			return resolved
		}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = b.instantiateTypeRef(arg, subst)
	}
	return &typecheck.Type{Kind: b.kindOf(ref.Name), Name: ref.Name, Args: args}
}

// modeFromMutable maps parser mutability into typed binding mode.
func modeFromMutable(mutable bool) BindingMode {
	if mutable {
		return BindingMutable
	}
	return BindingImmutable
}

// initMode decides whether a typed declaration is immediate or deferred.
func initMode(deferred bool, init Expr) InitMode {
	if deferred || init == nil {
		return InitDeferred
	}
	return InitImmediate
}

// elementType extracts the element type for iterable-like typed values.
func elementType(typ *typecheck.Type) *typecheck.Type {
	if typ == nil {
		return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
	}
	if len(typ.Args) > 0 {
		return typ.Args[0]
	}
	return &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
}

// iterableElementType extracts the item type for iterable runtime values.
func (b *typeRefBuilder) iterableElementType(typ *typecheck.Type) *typecheck.Type {
	return elementType(typ)
}

// exprSpan returns the parser span for a parser expression node.
func exprSpan(expr parser.Expr) parser.Span {
	switch e := expr.(type) {
	case *parser.Identifier:
		return e.Span
	case *parser.PlaceholderExpr:
		return e.Span
	case *parser.IntegerLiteral:
		return e.Span
	case *parser.FloatLiteral:
		return e.Span
	case *parser.RuneLiteral:
		return e.Span
	case *parser.BoolLiteral:
		return e.Span
	case *parser.StringLiteral:
		return e.Span
	case *parser.ListLiteral:
		return e.Span
	case *parser.TupleLiteral:
		return e.Span
	case *parser.CallExpr:
		return e.Span
	case *parser.MemberExpr:
		return e.Span
	case *parser.IndexExpr:
		return e.Span
	case *parser.IfExpr:
		return e.Span
	case *parser.ForYieldExpr:
		return e.Span
	case *parser.LambdaExpr:
		return e.Span
	case *parser.BinaryExpr:
		return e.Span
	case *parser.UnaryExpr:
		return e.Span
	case *parser.GroupExpr:
		return e.Span
	default:
		return parser.Span{}
	}
}

// sameType compares semantic types for exact typed-builder matching needs.
func sameType(left, right *typecheck.Type) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Kind == typecheck.TypeUnknown || right.Kind == typecheck.TypeUnknown {
		return true
	}
	if left.Kind == typecheck.TypeTuple && right.Kind == typecheck.TypeTuple {
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
	if left.Kind != right.Kind || left.Name != right.Name || len(left.Args) != len(right.Args) {
		if left.Kind == typecheck.TypeFunction && right.Kind == typecheck.TypeFunction {
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
		return false
	}
	for i := range left.Args {
		if !sameType(left.Args[i], right.Args[i]) {
			return false
		}
	}
	return true
}
