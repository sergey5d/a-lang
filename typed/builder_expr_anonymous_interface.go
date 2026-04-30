package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

func (b *exprBuilder) buildAnonymousInterfaceExpr(expr parser.Expr, anon *parser.AnonymousInterfaceExpr) (Expr, error) {
	interfaces := make([]*typecheck.Type, len(anon.Interfaces))
	for i, iface := range anon.Interfaces {
		interfaces[i] = b.types.BuildType(iface)
	}
	return &AnonymousInterfaceExpr{baseExpr: b.base(expr), Interfaces: interfaces}, nil
}
