package typed

import (
	"fmt"

	"a-lang/parser"
)

func (b *exprBuilder) buildAnonymousRecordExpr(expr parser.Expr, record *parser.AnonymousRecordExpr) (Expr, error) {
	if len(record.Values) > 0 {
		return nil, fmt.Errorf("anonymous positional record literal was not resolved during type checking")
	}
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
