package lower

import "a-lang/typed"

func (l *Lowerer) lowerProgram(program *typed.Program) (*Program, error) {
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
	out := &Class{Name: class.Name}
	for _, field := range class.Fields {
		out.Fields = append(out.Fields, &Field{
			Name:    field.Name,
			Mutable: field.Mode == typed.BindingMutable,
			Private: field.Private,
			Type:    field.Type,
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
