package lower

import "a-lang/typecheck"

type Program struct {
	Globals   []*Global
	Functions []*Function
	Classes   []*Class
}

type Global struct {
	Name    string
	Mutable bool
	Type    *typecheck.Type
	Init    Expr
}

type Class struct {
	Name        string
	Fields      []*Field
	Constructor *Function
	Methods     []*Function
}

type Field struct {
	Name    string
	Mutable bool
	Private bool
	Type    *typecheck.Type
}

type Function struct {
	Name        string
	Parameters  []Parameter
	ReturnType  *typecheck.Type
	Body        []Stmt
	Receiver    string
	Private     bool
	Constructor bool
}

type Parameter struct {
	Name string
	Type *typecheck.Type
}

type Stmt interface{ stmtNode() }
type Expr interface{ exprNode() }

type VarDecl struct {
	Name    string
	Mutable bool
	Type    *typecheck.Type
	Init    Expr
}

func (*VarDecl) stmtNode() {}

type Assign struct {
	Target   Expr
	Operator string
	Value    Expr
}

func (*Assign) stmtNode() {}

type If struct {
	Condition Expr
	Then      []Stmt
	Else      []Stmt
}

func (*If) stmtNode() {}

type ForEach struct {
	Name     string
	Iterable Expr
	Body     []Stmt
}

func (*ForEach) stmtNode() {}

type Loop struct {
	Body []Stmt
}

func (*Loop) stmtNode() {}

type Return struct {
	Value Expr
}

func (*Return) stmtNode() {}

type Break struct{}

func (*Break) stmtNode() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmtNode() {}

type VarRef struct {
	Name string
	Type *typecheck.Type
}

func (*VarRef) exprNode() {}

type ThisRef struct {
	Type *typecheck.Type
}

func (*ThisRef) exprNode() {}

type IntLiteral struct {
	Value int64
	Type  *typecheck.Type
}

func (*IntLiteral) exprNode() {}

type FloatLiteral struct {
	Value float64
	Type  *typecheck.Type
}

func (*FloatLiteral) exprNode() {}

type BoolLiteral struct {
	Value bool
	Type  *typecheck.Type
}

func (*BoolLiteral) exprNode() {}

type StringLiteral struct {
	Value string
	Type  *typecheck.Type
}

func (*StringLiteral) exprNode() {}

type RuneLiteral struct {
	Value rune
	Type  *typecheck.Type
}

func (*RuneLiteral) exprNode() {}

type ListLiteral struct {
	Elements []Expr
	Type     *typecheck.Type
}

func (*ListLiteral) exprNode() {}

type Unary struct {
	Operator string
	Right    Expr
	Type     *typecheck.Type
}

func (*Unary) exprNode() {}

type Binary struct {
	Left     Expr
	Operator string
	Right    Expr
	Type     *typecheck.Type
}

func (*Binary) exprNode() {}

type FunctionCall struct {
	Name string
	Args []Expr
	Type *typecheck.Type
}

func (*FunctionCall) exprNode() {}

type BuiltinCall struct {
	Name string
	Args []Expr
	Type *typecheck.Type
}

func (*BuiltinCall) exprNode() {}

type Construct struct {
	Class string
	Args  []Expr
	Type  *typecheck.Type
}

func (*Construct) exprNode() {}

type FieldGet struct {
	Receiver Expr
	Name     string
	Type     *typecheck.Type
}

func (*FieldGet) exprNode() {}

type MethodCall struct {
	Receiver Expr
	Method   string
	Args     []Expr
	Type     *typecheck.Type
}

func (*MethodCall) exprNode() {}
