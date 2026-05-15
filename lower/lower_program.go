package lower

import (
	"a-lang/parser"
	"a-lang/typecheck"
	"a-lang/typed"
)

func (l *Lowerer) lowerProgram(program *typed.Program) (*Program, error) {
	l.classes = map[string]*typed.ClassDecl{}
	for _, class := range program.Classes {
		l.classes[class.Name] = class
	}
	out := &Program{}
	for _, stmt := range program.Globals {
		globals, err := l.lowerGlobal(stmt)
		if err != nil {
			return nil, err
		}
		out.Globals = append(out.Globals, globals...)
	}
	for _, fn := range program.Functions {
		lowered, err := l.lowerFunction(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, lowered)
	}
	for _, class := range program.Classes {
		lowered, err := l.lowerClass(class)
		if err != nil {
			return nil, err
		}
		out.Classes = append(out.Classes, lowered)
	}
	return out, nil
}

func (l *Lowerer) lowerGlobal(stmt typed.Stmt) ([]*Global, error) {
	switch s := stmt.(type) {
	case *typed.BindingStmt:
		var globals []*Global
		for _, binding := range s.Bindings {
			var init Expr
			var err error
			if binding.Init != nil {
				init, err = l.lowerExpr(binding.Init)
				if err != nil {
					return nil, err
				}
			}
			globals = append(globals, &Global{
				Name:    binding.Name,
				Mutable: binding.Mode == typed.BindingMutable,
				Type:    binding.Type,
				Init:    init,
			})
		}
		return globals, nil
	default:
		return nil, unsupportedTopLevelErr(stmt)
	}
}

func (l *Lowerer) lowerClass(class *typed.ClassDecl) (*Class, error) {
	prevTypeParams := l.typeParams
	l.typeParams = map[string]struct{}{}
	for _, param := range class.TypeParameters {
		l.typeParams[param.Name] = struct{}{}
	}
	defer func() {
		l.typeParams = prevTypeParams
	}()

	out := &Class{
		Name:           class.Name,
		Object:         class.Object,
		Record:         class.Record,
		Enum:           class.Enum,
		TypeParameters: make([]string, len(class.TypeParameters)),
	}
	for i, param := range class.TypeParameters {
		out.TypeParameters[i] = param.Name
	}
	for _, field := range class.Fields {
		var init Expr
		var err error
		if field.Init != nil {
			init, err = l.lowerExpr(field.Init)
			if err != nil {
				return nil, err
			}
		}
		out.Fields = append(out.Fields, &Field{
			Name:    field.Name,
			Mutable: field.Mode == typed.BindingMutable,
			Private: field.Private,
			Type:    field.Type,
			Init:    init,
		})
	}
	for _, method := range class.Methods {
		lowered, err := l.lowerMethod(class.Name, method)
		if err != nil {
			return nil, err
		}
		if method.Constructor {
			out.Constructor = lowered
		} else {
			out.Methods = append(out.Methods, lowered)
		}
	}
	for _, enumCase := range class.Cases {
		loweredCase := EnumCase{Name: enumCase.Name}
		for _, field := range enumCase.Fields {
			loweredCase.Fields = append(loweredCase.Fields, &Field{
				Name:    field.Name,
				Mutable: field.Mutable,
				Private: field.Private,
				Type:    l.resolveFieldType(field.Type),
			})
		}
		for _, assignment := range enumCase.Assignments {
			value, err := l.lowerExprFromParser(assignment.Value)
			if err != nil {
				return nil, err
			}
			loweredCase.Assignments = append(loweredCase.Assignments, EnumCaseAssignment{
				Name:  assignment.Name,
				Value: value,
			})
		}
		out.Cases = append(out.Cases, loweredCase)
	}
	return out, nil
}

func (l *Lowerer) lowerFunction(fn *typed.FunctionDecl) (*Function, error) {
	body, err := l.lowerBlock(fn.Body)
	if err != nil {
		return nil, err
	}
	return &Function{
		Name:       fn.Name,
		Parameters: l.lowerParams(fn.Parameters),
		ReturnType: fn.ReturnType,
		Body:       body,
	}, nil
}

func (l *Lowerer) lowerMethod(receiver string, method *typed.MethodDecl) (*Function, error) {
	body, err := l.lowerBlock(method.Body)
	if err != nil {
		return nil, err
	}
	return &Function{
		Name:        method.Name,
		Parameters:  l.lowerParams(method.Parameters),
		ReturnType:  method.ReturnType,
		Body:        body,
		Receiver:    receiver,
		Private:     method.Private,
		Constructor: method.Constructor,
	}, nil
}

func (l *Lowerer) lowerParams(params []typed.Parameter) []Parameter {
	out := make([]Parameter, len(params))
	for i, param := range params {
		out[i] = Parameter{Name: param.Name, Type: param.Type}
	}
	return out
}

func (l *Lowerer) resolveFieldType(ref *parser.TypeRef) *typecheck.Type {
	if ref == nil {
		return unknownType()
	}
	if ref.ReturnType != nil {
		params := make([]*typecheck.Type, len(ref.ParameterTypes))
		for i, param := range ref.ParameterTypes {
			params[i] = l.resolveFieldType(param)
		}
		return &typecheck.Type{
			Kind: typecheck.TypeFunction,
			Name: "func",
			Signature: &typecheck.Signature{
				Parameters: params,
				ReturnType: l.resolveFieldType(ref.ReturnType),
			},
		}
	}
	if len(ref.TupleElements) > 0 {
		args := make([]*typecheck.Type, len(ref.TupleElements))
		for i, arg := range ref.TupleElements {
			args[i] = l.resolveFieldType(arg)
		}
		return &typecheck.Type{Kind: typecheck.TypeTuple, Name: "Tuple", Args: args, TupleNames: append([]string(nil), ref.TupleNames...)}
	}
	if len(ref.RecordFields) > 0 {
		fields := make([]typecheck.RecordField, len(ref.RecordFields))
		for i, field := range ref.RecordFields {
			fields[i] = typecheck.RecordField{Name: field.Name, Type: l.resolveFieldType(field.Type)}
		}
		return &typecheck.Type{Kind: typecheck.TypeRecord, Name: "Record", Fields: fields}
	}
	args := make([]*typecheck.Type, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		args[i] = l.resolveFieldType(arg)
	}
	kind := typecheck.TypeUnknown
	if _, ok := l.typeParams[ref.Name]; ok {
		kind = typecheck.TypeParam
	} else if _, ok := l.classes[ref.Name]; ok {
		kind = typecheck.TypeClass
	} else {
		switch ref.Name {
		case "Int", "Float", "Bool", "Str", "Rune", "Decimal", "Array", "Unit":
			kind = typecheck.TypeBuiltin
		case "List", "Map", "Set", "Printer", "Eq", "Option", "Result", "Either":
			kind = typecheck.TypeInterface
		}
	}
	return &typecheck.Type{Kind: kind, Name: ref.Name, Args: args}
}
