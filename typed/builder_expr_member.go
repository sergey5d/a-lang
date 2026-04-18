package typed

import "a-lang/parser"

// buildMemberExpr converts parser member access into typed field access.
func (b *exprBuilder) buildMemberExpr(expr parser.Expr, member *parser.MemberExpr) (Expr, error) {
	receiver, err := b.Build(member.Receiver)
	if err != nil {
		return nil, err
	}
	field := &FieldExpr{baseExpr: b.base(expr), Receiver: receiver, Name: member.Name}
	field.Field = b.types.resolveFieldSymbol(receiver.GetType(), member.Name)
	return field, nil
}
