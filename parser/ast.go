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
	Functions []*FunctionDecl `json:"functions"`
	Span      Span            `json:"span"`
}

type FunctionDecl struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType string      `json:"returnType"`
	Body       *BlockStmt  `json:"body"`
	Span       Span        `json:"span"`
}

type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Span Span   `json:"span"`
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
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Mutable bool   `json:"mutable,omitempty"`
	Span    Span   `json:"span"`
}

func (*ValStmt) statementNode() {}

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

type MatchStmt struct {
	Target Expr       `json:"target"`
	Arms   []MatchArm `json:"arms"`
	Span   Span       `json:"span"`
}

func (*MatchStmt) statementNode() {}

type MatchArm struct {
	Pattern     Expr   `json:"pattern"`
	PatternType string `json:"patternType,omitempty"`
	Result      Expr   `json:"result"`
	Span        Span   `json:"span"`
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
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Span Span   `json:"span"`
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
