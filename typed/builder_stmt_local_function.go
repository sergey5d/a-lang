package typed

import (
	"fmt"

	"a-lang/parser"
	"a-lang/typecheck"
)

// localFunctionStmtBuilder builds typed local function statements as immutable lambda bindings.
type localFunctionStmtBuilder struct {
	ctx    *buildContext
	blocks Builder[*parser.BlockStmt, *BlockStmt]
	types  *typeRefBuilder
}

// Build converts a parser local function into a typed binding statement backed by a lambda.
func (b *localFunctionStmtBuilder) Build(stmt *parser.LocalFunctionStmt) (Stmt, error) {
	fn := stmt.Function
	if len(fn.TypeParameters) > 0 {
		return nil, fmt.Errorf("unsupported local function type parameters on %s", fn.Name)
	}

	paramTypes := make([]*typecheck.Type, len(fn.Parameters))
	params := make([]LambdaParameter, len(fn.Parameters))
	for i, param := range fn.Parameters {
		paramType := b.types.BuildType(param.Type)
		paramTypes[i] = paramType
		params[i] = LambdaParameter{
			Name:   param.Name,
			Type:   paramType,
			Symbol: b.ctx.newSymbol(SymbolParameter, param.Name, "", param.Span),
			Span:   param.Span,
		}
	}

	returnType := b.types.BuildType(fn.ReturnType)
	lambdaType := &typecheck.Type{
		Kind: typecheck.TypeFunction,
		Name: "func",
		Signature: &typecheck.Signature{
			Parameters: paramTypes,
			ReturnType: returnType,
			Variadic:   len(fn.Parameters) > 0 && fn.Parameters[len(fn.Parameters)-1].Variadic,
		},
	}

	symbol := b.ctx.newSymbol(SymbolBinding, fn.Name, "", stmt.Span)
	if fn.Name != "_" {
		b.ctx.defineSymbol(symbol)
	}

	b.ctx.pushScope()
	defer b.ctx.popScope()
	for _, param := range params {
		b.ctx.defineSymbol(param.Symbol)
	}

	body, err := b.blocks.Build(fn.Body)
	if err != nil {
		return nil, err
	}

	return &BindingStmt{
		Bindings: []BindingDecl{
			{
				Name:     fn.Name,
				Type:     lambdaType,
				Mode:     BindingImmutable,
				InitMode: InitImmediate,
				Init: &LambdaExpr{
					baseExpr:   baseExpr{Type: lambdaType, Span: fn.Span},
					Parameters: params,
					BlockBody:  body,
				},
				Symbol: symbol,
				Span:   stmt.Span,
			},
		},
		Span: stmt.Span,
	}, nil
}
