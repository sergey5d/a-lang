package typed

import (
	"fmt"

	"a-lang/parser"
	"a-lang/typecheck"
)

// stmtBuilder builds typed statements and depends only on expression and block builders.
type stmtBuilder struct {
	ctx    *buildContext
	exprs  Builder[parser.Expr, Expr]
	blocks Builder[*parser.BlockStmt, *BlockStmt]
}

// Build converts a parser statement into its typed equivalent.
func (b *stmtBuilder) Build(stmt parser.Statement) (Stmt, error) {
	switch s := stmt.(type) {
	case *parser.ValStmt:
		bindings := make([]BindingDecl, len(s.Bindings))
		for i, binding := range s.Bindings {
			var init Expr
			var err error
			if i < len(s.Values) && s.Values[i] != nil {
				init, err = b.exprs.Build(s.Values[i])
				if err != nil {
					return nil, err
				}
			}
			symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			typ := init.GetType()
			if binding.Type != nil {
				typ = (&typeRefBuilder{ctx: b.ctx}).BuildType(binding.Type)
			}
			if typ == nil {
				typ = &typecheck.Type{Kind: typecheck.TypeUnknown, Name: "<unknown>"}
			}
			bindings[i] = BindingDecl{
				Name:     binding.Name,
				Type:     typ,
				Mode:     modeFromMutable(binding.Mutable),
				InitMode: initMode(binding.Deferred, init),
				Init:     init,
				Symbol:   symbol,
				Span:     binding.Span,
			}
			b.ctx.defineSymbol(symbol)
		}
		return &BindingStmt{Bindings: bindings, Span: s.Span}, nil
	case *parser.AssignmentStmt:
		target, err := b.exprs.Build(s.Target)
		if err != nil {
			return nil, err
		}
		value, err := b.exprs.Build(s.Value)
		if err != nil {
			return nil, err
		}
		return &AssignmentStmt{Target: target, Operator: s.Operator, Value: value, Span: s.Span}, nil
	case *parser.IfStmt:
		cond, err := b.exprs.Build(s.Condition)
		if err != nil {
			return nil, err
		}
		thenBlock, err := b.blocks.Build(s.Then)
		if err != nil {
			return nil, err
		}
		var elseIf *IfStmt
		if s.ElseIf != nil {
			built, err := b.Build(s.ElseIf)
			if err != nil {
				return nil, err
			}
			elseIf = built.(*IfStmt)
		}
		elseBlock, err := b.blocks.Build(s.Else)
		if err != nil {
			return nil, err
		}
		return &IfStmt{Condition: cond, Then: thenBlock, ElseIf: elseIf, Else: elseBlock, Span: s.Span}, nil
	case *parser.ForStmt:
		bindings := make([]ForBinding, len(s.Bindings))
		for i, binding := range s.Bindings {
			iterable, err := b.exprs.Build(binding.Iterable)
			if err != nil {
				return nil, err
			}
			symbol := b.ctx.newSymbol(SymbolBinding, binding.Name, "", binding.Span)
			bindings[i] = ForBinding{
				Name:     binding.Name,
				Type:     elementType(iterable.GetType()),
				Iterable: iterable,
				Symbol:   symbol,
				Span:     binding.Span,
			}
		}

		b.ctx.pushScope()
		for _, binding := range bindings {
			b.ctx.defineSymbol(binding.Symbol)
		}
		body, err := b.blocks.Build(s.Body)
		b.ctx.popScope()
		if err != nil {
			return nil, err
		}

		b.ctx.pushScope()
		for _, binding := range bindings {
			b.ctx.defineSymbol(binding.Symbol)
		}
		yieldBody, err := b.blocks.Build(s.YieldBody)
		b.ctx.popScope()
		if err != nil {
			return nil, err
		}

		return &ForStmt{Bindings: bindings, Body: body, YieldBody: yieldBody, Span: s.Span}, nil
	case *parser.ReturnStmt:
		value, err := b.exprs.Build(s.Value)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{Value: value, Span: s.Span}, nil
	case *parser.BreakStmt:
		return &BreakStmt{Span: s.Span}, nil
	case *parser.ExprStmt:
		expr, err := b.exprs.Build(s.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: expr, Span: s.Span}, nil
	default:
		return nil, fmt.Errorf("unsupported statement type %T", stmt)
	}
}
