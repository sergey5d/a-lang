package parser

func Parse(input string) (*Program, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}

	parser := &Parser{
		tokens: tokens,
		scopes: []map[string]struct{}{{}},
	}
	return parser.parseProgram()
}

type Parser struct {
	tokens []Token
	pos    int
	scopes []map[string]struct{}
}

func tokenSpan(token Token) Span {
	return Span{
		Start: Position{Line: token.Line, Column: token.Column},
		End:   Position{Line: token.EndLine, Column: token.EndColumn},
	}
}

func mergeSpans(start, end Span) Span {
	return Span{Start: start.Start, End: end.End}
}

func typeSpan(ref *TypeRef) Span {
	if ref == nil {
		return Span{}
	}
	return ref.Span
}

func implicitUnitType(span Span) *TypeRef {
	return &TypeRef{Name: "Unit", Span: span}
}

func isZeroArgFunctionType(ref *TypeRef) bool {
	return ref != nil && ref.ReturnType != nil && len(ref.ParameterTypes) == 0
}

func wrapThunkExpr(ref *TypeRef, expr Expr) Expr {
	if !isZeroArgFunctionType(ref) {
		return expr
	}
	if _, ok := expr.(*LambdaExpr); ok {
		return expr
	}
	return &LambdaExpr{
		Parameters: []LambdaParameter{},
		Body:       expr,
		Span:       exprSpan(expr),
	}
}

func exprSpan(expr Expr) Span {
	switch e := expr.(type) {
	case *Identifier:
		return e.Span
	case *PlaceholderExpr:
		return e.Span
	case *IntegerLiteral:
		return e.Span
	case *FloatLiteral:
		return e.Span
	case *RuneLiteral:
		return e.Span
	case *BoolLiteral:
		return e.Span
	case *StringLiteral:
		return e.Span
	case *UnitLiteral:
		return e.Span
	case *ListLiteral:
		return e.Span
	case *TupleLiteral:
		return e.Span
	case *CallExpr:
		return e.Span
	case *MemberExpr:
		return e.Span
	case *IndexExpr:
		return e.Span
	case *IfExpr:
		return e.Span
	case *ForYieldExpr:
		return e.Span
	case *LambdaExpr:
		return e.Span
	case *BinaryExpr:
		return e.Span
	case *UnaryExpr:
		return e.Span
	case *GroupExpr:
		return e.Span
	default:
		return Span{}
	}
}

func stmtSpan(stmt Statement) Span {
	switch s := stmt.(type) {
	case *ValStmt:
		return s.Span
	case *LocalFunctionStmt:
		return s.Span
	case *AssignmentStmt:
		return s.Span
	case *MultiAssignmentStmt:
		return s.Span
	case *IfStmt:
		return s.Span
	case *ForStmt:
		return s.Span
	case *ReturnStmt:
		return s.Span
	case *BreakStmt:
		return s.Span
	case *ExprStmt:
		return s.Span
	default:
		return Span{}
	}
}
