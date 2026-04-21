package typed

import "a-lang/parser"

func (b *exprBuilder) buildRecordUpdateExpr(expr parser.Expr, update *parser.RecordUpdateExpr) (Expr, error) {
	receiver, err := b.Build(update.Receiver)
	if err != nil {
		return nil, err
	}
	fields := make([]RecordUpdateField, len(update.Updates))
	for i, item := range update.Updates {
		value, err := b.Build(item.Value)
		if err != nil {
			return nil, err
		}
		fields[i] = RecordUpdateField{Name: item.Name, Value: value}
	}
	return &RecordUpdateExpr{baseExpr: b.base(expr), Receiver: receiver, Updates: fields}, nil
}
