package parser

type Program struct {
	Functions []*FunctionDecl `json:"functions"`
}

type FunctionDecl struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType string      `json:"returnType"`
	Body       *BlockStmt  `json:"body"`
}

type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type BlockStmt struct {
	Statements []Statement `json:"statements"`
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
}

type Binding struct {
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Mutable bool   `json:"mutable,omitempty"`
}

func (*ValStmt) statementNode() {}

type IfStmt struct {
	Condition Expr       `json:"condition"`
	Then      *BlockStmt `json:"then"`
	ElseIf    *IfStmt    `json:"elseIf,omitempty"`
	Else      *BlockStmt `json:"else,omitempty"`
}

func (*IfStmt) statementNode() {}

type ForStmt struct {
	Name     string     `json:"name"`
	Iterable Expr       `json:"iterable"`
	Body     *BlockStmt `json:"body"`
}

func (*ForStmt) statementNode() {}

type DoYieldStmt struct {
	Bindings []ForBinding `json:"bindings"`
	Body     *BlockStmt   `json:"body"`
}

func (*DoYieldStmt) statementNode() {}

type ForBinding struct {
	Name     string `json:"name"`
	Iterable Expr   `json:"iterable"`
}

type MatchStmt struct {
	Target Expr       `json:"target"`
	Arms   []MatchArm `json:"arms"`
}

func (*MatchStmt) statementNode() {}

type MatchArm struct {
	Pattern     Expr   `json:"pattern"`
	PatternType string `json:"patternType,omitempty"`
	Result      Expr   `json:"result"`
}

type ReturnStmt struct {
	Value Expr `json:"value"`
}

func (*ReturnStmt) statementNode() {}

type BreakStmt struct{}

func (*BreakStmt) statementNode() {}

type ExprStmt struct {
	Expr Expr `json:"expr"`
}

func (*ExprStmt) statementNode() {}

type Identifier struct {
	Name string `json:"name"`
}

func (*Identifier) exprNode() {}

type PlaceholderExpr struct{}

func (*PlaceholderExpr) exprNode() {}

type IntegerLiteral struct {
	Value string `json:"value"`
}

func (*IntegerLiteral) exprNode() {}

type StringLiteral struct {
	Value string `json:"value"`
}

func (*StringLiteral) exprNode() {}

type ListLiteral struct {
	Elements []Expr `json:"elements"`
}

func (*ListLiteral) exprNode() {}

type MapLiteral struct{}

func (*MapLiteral) exprNode() {}

type CallExpr struct {
	Callee Expr   `json:"callee"`
	Args   []Expr `json:"args"`
}

func (*CallExpr) exprNode() {}

type MemberExpr struct {
	Receiver Expr   `json:"receiver"`
	Name     string `json:"name"`
}

func (*MemberExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr   `json:"left"`
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
}

func (*BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
}

func (*UnaryExpr) exprNode() {}

type GroupExpr struct {
	Inner Expr `json:"inner"`
}

func (*GroupExpr) exprNode() {}
