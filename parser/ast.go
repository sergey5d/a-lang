package parser

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Program struct {
	Functions  []*FunctionDecl  `json:"functions"`
	Interfaces []*InterfaceDecl `json:"interfaces,omitempty"`
	Classes    []*ClassDecl     `json:"classes,omitempty"`
	Statements []Statement      `json:"statements,omitempty"`
	Span       Span             `json:"span"`
}

type TypeRef struct {
	Name      string     `json:"name"`
	Arguments []*TypeRef `json:"arguments,omitempty"`
	Span      Span       `json:"span"`
}

type TypeParameter struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

type FunctionDecl struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType *TypeRef    `json:"returnType"`
	Body       *BlockStmt  `json:"body"`
	Span       Span        `json:"span"`
}

type InterfaceDecl struct {
	Name           string            `json:"name"`
	TypeParameters []TypeParameter   `json:"typeParameters,omitempty"`
	Methods        []InterfaceMethod `json:"methods"`
	Span           Span              `json:"span"`
}

type InterfaceMethod struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType *TypeRef    `json:"returnType"`
	Span       Span        `json:"span"`
}

type ClassDecl struct {
	Name           string          `json:"name"`
	TypeParameters []TypeParameter `json:"typeParameters,omitempty"`
	Implements     []*TypeRef      `json:"implements,omitempty"`
	Fields         []FieldDecl     `json:"fields,omitempty"`
	Methods        []*MethodDecl   `json:"methods,omitempty"`
	Span           Span            `json:"span"`
}

type FieldDecl struct {
	Name    string   `json:"name"`
	Type    *TypeRef `json:"type"`
	Mutable bool     `json:"mutable,omitempty"`
	Private bool     `json:"private,omitempty"`
	Span    Span     `json:"span"`
}

type MethodDecl struct {
	Name        string      `json:"name"`
	Parameters  []Parameter `json:"parameters"`
	ReturnType  *TypeRef    `json:"returnType,omitempty"`
	Body        *BlockStmt  `json:"body"`
	Private     bool        `json:"private,omitempty"`
	Constructor bool        `json:"constructor,omitempty"`
	Span        Span        `json:"span"`
}

type Parameter struct {
	Name string   `json:"name"`
	Type *TypeRef `json:"type"`
	Span Span     `json:"span"`
}

type BlockStmt struct {
	Statements []Statement `json:"statements"`
	Span       Span        `json:"span"`
}

type Statement interface {
	statementNode()
}

type Expr interface {
	exprNode()
}

type ValStmt struct {
	Bindings []Binding `json:"bindings"`
	Values   []Expr    `json:"values"`
	Span     Span      `json:"span"`
}

type Binding struct {
	Name    string   `json:"name"`
	Type    *TypeRef `json:"type,omitempty"`
	Mutable bool     `json:"mutable,omitempty"`
	Span    Span     `json:"span"`
}

func (*ValStmt) statementNode() {}

type AssignmentStmt struct {
	Target   Expr   `json:"target"`
	Operator string `json:"operator"`
	Value    Expr   `json:"value"`
	Span     Span   `json:"span"`
}

func (*AssignmentStmt) statementNode() {}

type IfStmt struct {
	Condition Expr       `json:"condition"`
	Then      *BlockStmt `json:"then"`
	ElseIf    *IfStmt    `json:"elseIf,omitempty"`
	Else      *BlockStmt `json:"else,omitempty"`
	Span      Span       `json:"span"`
}

func (*IfStmt) statementNode() {}

type ForStmt struct {
	Bindings  []ForBinding `json:"bindings,omitempty"`
	Body      *BlockStmt   `json:"body,omitempty"`
	YieldBody *BlockStmt   `json:"yieldBody,omitempty"`
	Span      Span         `json:"span"`
}

func (*ForStmt) statementNode() {}

type ForBinding struct {
	Name     string `json:"name"`
	Iterable Expr   `json:"iterable"`
	Span     Span   `json:"span"`
}

type ReturnStmt struct {
	Value Expr `json:"value"`
	Span  Span `json:"span"`
}

func (*ReturnStmt) statementNode() {}

type BreakStmt struct {
	Span Span `json:"span"`
}

func (*BreakStmt) statementNode() {}

type ExprStmt struct {
	Expr Expr `json:"expr"`
	Span Span `json:"span"`
}

func (*ExprStmt) statementNode() {}

type Identifier struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

func (*Identifier) exprNode() {}

type PlaceholderExpr struct {
	Span Span `json:"span"`
}

func (*PlaceholderExpr) exprNode() {}

type IntegerLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*IntegerLiteral) exprNode() {}

type FloatLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*FloatLiteral) exprNode() {}

type RuneLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*RuneLiteral) exprNode() {}

type BoolLiteral struct {
	Value bool `json:"value"`
	Span  Span `json:"span"`
}

func (*BoolLiteral) exprNode() {}

type StringLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*StringLiteral) exprNode() {}

type ListLiteral struct {
	Elements []Expr `json:"elements"`
	Span     Span   `json:"span"`
}

func (*ListLiteral) exprNode() {}

type MapLiteral struct {
	Span Span `json:"span"`
}

func (*MapLiteral) exprNode() {}

type CallExpr struct {
	Callee Expr   `json:"callee"`
	Args   []Expr `json:"args"`
	Span   Span   `json:"span"`
}

func (*CallExpr) exprNode() {}

type MemberExpr struct {
	Receiver Expr   `json:"receiver"`
	Name     string `json:"name"`
	Span     Span   `json:"span"`
}

func (*MemberExpr) exprNode() {}

type LambdaParameter struct {
	Name string   `json:"name"`
	Type *TypeRef `json:"type,omitempty"`
	Span Span     `json:"span"`
}

type LambdaExpr struct {
	Parameters []LambdaParameter `json:"parameters"`
	Body       Expr              `json:"body"`
	Span       Span              `json:"span"`
}

func (*LambdaExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr   `json:"left"`
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
	Span     Span   `json:"span"`
}

func (*BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
	Span     Span   `json:"span"`
}

func (*UnaryExpr) exprNode() {}

type GroupExpr struct {
	Inner Expr `json:"inner"`
	Span  Span `json:"span"`
}

func (*GroupExpr) exprNode() {}
