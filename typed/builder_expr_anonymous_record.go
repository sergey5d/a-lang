package typed

import "a-lang/parser"

func (b *exprBuilder) buildAnonymousRecordExpr(expr parser.Expr, record *parser.AnonymousRecordExpr) (Expr, error) {
	fields := make([]RecordUpdateField, len(record.Fields))
	for i, field := range record.Fields {
		value, err := b.Build(field.Value)
		if err != nil {
			return nil, err
		}
		fields[i] = RecordUpdateField{Name: field.Name, Value: value}
	}
	return &AnonymousRecordExpr{baseExpr: b.base(expr), Fields: fields}, nil
}
