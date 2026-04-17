package typed

import (
	"a-lang/parser"
	"a-lang/typecheck"
)

type BindingMode string

const (
	BindingImmutable BindingMode = "immutable"
	BindingMutable   BindingMode = "mutable"
)

type InitMode string

const (
	InitImmediate InitMode = "immediate"
	InitDeferred  InitMode = "deferred"
)

type Program struct {
	Functions  []*FunctionDecl
	Interfaces []*InterfaceDecl
	Classes    []*ClassDecl
	Globals    []Stmt
	Span       parser.Span
}

type TypeParameter struct {
	Name string
	Span parser.Span
}

type Parameter struct {
	Name string
	Type *typecheck.Type
	Span parser.Span
}

type FunctionDecl struct {
	Name       string
	Parameters []Parameter
	ReturnType *typecheck.Type
	Body       *BlockStmt
	Span       parser.Span
}

type InterfaceMethod struct {
	Name       string
	Parameters []Parameter
	ReturnType *typecheck.Type
	Span       parser.Span
}

type InterfaceDecl struct {
	Name           string
	TypeParameters []TypeParameter
	Methods        []InterfaceMethod
	Span           parser.Span
}

type FieldDecl struct {
	Name     string
	Type     *typecheck.Type
	Mode     BindingMode
	InitMode InitMode
	Init     Expr
	Private  bool
	Span     parser.Span
}

type MethodDecl struct {
	Name        string
	Parameters  []Parameter
	ReturnType  *typecheck.Type
	Body        *BlockStmt
	Private     bool
	Constructor bool
	Span        parser.Span
}

type ClassDecl struct {
	Name           string
	TypeParameters []TypeParameter
	Interfaces     []*typecheck.Type
	Fields         []FieldDecl
	Methods        []*MethodDecl
	Span           parser.Span
}

type Stmt interface {
	stmtNode()
	GetSpan() parser.Span
}

type Expr interface {
	exprNode()
	GetSpan() parser.Span
	GetType() *typecheck.Type
}

type BlockStmt struct {
	Statements []Stmt
	Span       parser.Span
}

type BindingDecl struct {
	Name     string
	Type     *typecheck.Type
	Mode     BindingMode
	InitMode InitMode
	Init     Expr
	Span     parser.Span
}

type BindingStmt struct {
	Bindings []BindingDecl
	Span     parser.Span
}

func (*BindingStmt) stmtNode() {}
func (s *BindingStmt) GetSpan() parser.Span { return s.Span }

type AssignmentStmt struct {
	Target   Expr
	Operator string
	Value    Expr
	Span     parser.Span
}

func (*AssignmentStmt) stmtNode() {}
func (s *AssignmentStmt) GetSpan() parser.Span { return s.Span }

type IfStmt struct {
	Condition Expr
	Then      *BlockStmt
	ElseIf    *IfStmt
	Else      *BlockStmt
	Span      parser.Span
}

func (*IfStmt) stmtNode() {}
func (s *IfStmt) GetSpan() parser.Span { return s.Span }

type ForBinding struct {
	Name     string
	Type     *typecheck.Type
	Iterable Expr
	Span     parser.Span
}

type ForStmt struct {
	Bindings  []ForBinding
	Body      *BlockStmt
	YieldBody *BlockStmt
	Span      parser.Span
}

func (*ForStmt) stmtNode() {}
func (s *ForStmt) GetSpan() parser.Span { return s.Span }

type ReturnStmt struct {
	Value Expr
	Span  parser.Span
}

func (*ReturnStmt) stmtNode() {}
func (s *ReturnStmt) GetSpan() parser.Span { return s.Span }

type BreakStmt struct {
	Span parser.Span
}

func (*BreakStmt) stmtNode() {}
func (s *BreakStmt) GetSpan() parser.Span { return s.Span }

type ExprStmt struct {
	Expr Expr
	Span parser.Span
}

func (*ExprStmt) stmtNode() {}
func (s *ExprStmt) GetSpan() parser.Span { return s.Span }

type baseExpr struct {
	Type *typecheck.Type
	Span parser.Span
}

func (e *baseExpr) GetType() *typecheck.Type { return e.Type }
func (e *baseExpr) GetSpan() parser.Span     { return e.Span }

type IdentifierExpr struct {
	baseExpr
	Name string
}

func (*IdentifierExpr) exprNode() {}

type PlaceholderExpr struct {
	baseExpr
}

func (*PlaceholderExpr) exprNode() {}

type IntegerLiteral struct {
	baseExpr
	Value string
}

func (*IntegerLiteral) exprNode() {}

type FloatLiteral struct {
	baseExpr
	Value string
}

func (*FloatLiteral) exprNode() {}

type RuneLiteral struct {
	baseExpr
	Value string
}

func (*RuneLiteral) exprNode() {}

type BoolLiteral struct {
	baseExpr
	Value bool
}

func (*BoolLiteral) exprNode() {}

type StringLiteral struct {
	baseExpr
	Value string
}

func (*StringLiteral) exprNode() {}

type ListLiteral struct {
	baseExpr
	Elements []Expr
}

func (*ListLiteral) exprNode() {}

type GroupExpr struct {
	baseExpr
	Inner Expr
}

func (*GroupExpr) exprNode() {}

type UnaryExpr struct {
	baseExpr
	Operator string
	Right    Expr
}

func (*UnaryExpr) exprNode() {}

type BinaryExpr struct {
	baseExpr
	Left     Expr
	Operator string
	Right    Expr
}

func (*BinaryExpr) exprNode() {}

type FieldExpr struct {
	baseExpr
	Receiver Expr
	Name     string
}

func (*FieldExpr) exprNode() {}

type FunctionCallExpr struct {
	baseExpr
	Name string
	Args []Expr
}

func (*FunctionCallExpr) exprNode() {}

type ConstructorCallExpr struct {
	baseExpr
	Class string
	Args  []Expr
}

func (*ConstructorCallExpr) exprNode() {}

type MethodCallExpr struct {
	baseExpr
	Receiver Expr
	Method   string
	Args     []Expr
}

func (*MethodCallExpr) exprNode() {}

type InvokeExpr struct {
	baseExpr
	Callee Expr
	Args   []Expr
}

func (*InvokeExpr) exprNode() {}

type LambdaParameter struct {
	Name string
	Type *typecheck.Type
	Span parser.Span
}

type LambdaExpr struct {
	baseExpr
	Parameters []LambdaParameter
	Body       Expr
	BlockBody  *BlockStmt
}

func (*LambdaExpr) exprNode() {}
