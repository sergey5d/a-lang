package parser

// Position identifies a source location using 1-based line and column numbers.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Span describes the source range covered by a parsed node.
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Program is the root parser AST node for a source file.
type Program struct {
	Functions  []*FunctionDecl  `json:"functions"`
	Interfaces []*InterfaceDecl `json:"interfaces,omitempty"`
	Classes    []*ClassDecl     `json:"classes,omitempty"`
	Statements []Statement      `json:"statements,omitempty"`
	Span       Span             `json:"span"`
}

// TypeRef represents a named, generic, or function type in source.
type TypeRef struct {
	Name           string     `json:"name,omitempty"`
	Arguments      []*TypeRef `json:"arguments,omitempty"`
	ParameterTypes []*TypeRef `json:"parameterTypes,omitempty"`
	ReturnType     *TypeRef   `json:"returnType,omitempty"`
	Span           Span       `json:"span"`
}

// TypeParameter declares a generic type parameter name.
type TypeParameter struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

// FunctionDecl describes a top-level function declaration.
type FunctionDecl struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType *TypeRef    `json:"returnType"`
	Body       *BlockStmt  `json:"body"`
	Span       Span        `json:"span"`
}

// InterfaceDecl describes an interface declaration and its methods.
type InterfaceDecl struct {
	Name           string            `json:"name"`
	TypeParameters []TypeParameter   `json:"typeParameters,omitempty"`
	Methods        []InterfaceMethod `json:"methods"`
	Span           Span              `json:"span"`
}

// InterfaceMethod describes a method signature inside an interface.
type InterfaceMethod struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnType *TypeRef    `json:"returnType"`
	Span       Span        `json:"span"`
}

// ClassDecl describes a class declaration, its fields, and its methods.
type ClassDecl struct {
	Name           string          `json:"name"`
	TypeParameters []TypeParameter `json:"typeParameters,omitempty"`
	Implements     []*TypeRef      `json:"implements,omitempty"`
	Fields         []FieldDecl     `json:"fields,omitempty"`
	Methods        []*MethodDecl   `json:"methods,omitempty"`
	Span           Span            `json:"span"`
}

// FieldDecl describes a class field declaration.
type FieldDecl struct {
	Name        string   `json:"name"`
	Type        *TypeRef `json:"type"`
	Initializer Expr     `json:"initializer,omitempty"`
	Mutable     bool     `json:"mutable,omitempty"`
	Deferred    bool     `json:"deferred,omitempty"`
	Private     bool     `json:"private,omitempty"`
	Span        Span     `json:"span"`
}

// MethodDecl describes a class method or constructor declaration.
type MethodDecl struct {
	Name        string      `json:"name"`
	Parameters  []Parameter `json:"parameters"`
	ReturnType  *TypeRef    `json:"returnType,omitempty"`
	Body        *BlockStmt  `json:"body"`
	Private     bool        `json:"private,omitempty"`
	Constructor bool        `json:"constructor,omitempty"`
	Span        Span        `json:"span"`
}

// Parameter describes a named typed parameter in a callable signature.
type Parameter struct {
	Name     string   `json:"name"`
	Type     *TypeRef `json:"type"`
	Variadic bool     `json:"variadic,omitempty"`
	Span     Span     `json:"span"`
}

// BlockStmt is a braced sequence of statements.
type BlockStmt struct {
	Statements []Statement `json:"statements"`
	Span       Span        `json:"span"`
}

// Statement is implemented by all parser AST statement nodes.
type Statement interface {
	statementNode()
}

// LocalFunctionStmt declares a named local function inside a block.
type LocalFunctionStmt struct {
	Function *FunctionDecl `json:"function"`
	Span     Span          `json:"span"`
}

func (*LocalFunctionStmt) statementNode() {}

// Expr is implemented by all parser AST expression nodes.
type Expr interface {
	exprNode()
}

// ValStmt declares one or more bindings with matching initializer expressions.
type ValStmt struct {
	Bindings []Binding `json:"bindings"`
	Values   []Expr    `json:"values"`
	Span     Span      `json:"span"`
}

// Binding describes a single declared name in a binding statement.
type Binding struct {
	Name     string   `json:"name"`
	Type     *TypeRef `json:"type,omitempty"`
	Mutable  bool     `json:"mutable,omitempty"`
	Deferred bool     `json:"deferred,omitempty"`
	Span     Span     `json:"span"`
}

func (*ValStmt) statementNode() {}

// AssignmentStmt writes a value to an existing assignment target.
type AssignmentStmt struct {
	Target   Expr   `json:"target"`
	Operator string `json:"operator"`
	Value    Expr   `json:"value"`
	Span     Span   `json:"span"`
}

func (*AssignmentStmt) statementNode() {}

// IfStmt represents an if / else-if / else chain.
type IfStmt struct {
	Condition Expr       `json:"condition"`
	Then      *BlockStmt `json:"then"`
	ElseIf    *IfStmt    `json:"elseIf,omitempty"`
	Else      *BlockStmt `json:"else,omitempty"`
	Span      Span       `json:"span"`
}

func (*IfStmt) statementNode() {}

// ForStmt represents loop forms including foreach, infinite, and yield-style loops.
type ForStmt struct {
	Bindings  []ForBinding `json:"bindings,omitempty"`
	Body      *BlockStmt   `json:"body,omitempty"`
	YieldBody *BlockStmt   `json:"yieldBody,omitempty"`
	Span      Span         `json:"span"`
}

func (*ForStmt) statementNode() {}

// ForBinding binds a loop variable to an iterable expression.
type ForBinding struct {
	Name     string `json:"name"`
	Iterable Expr   `json:"iterable"`
	Span     Span   `json:"span"`
}

// ReturnStmt returns a value from a callable body.
type ReturnStmt struct {
	Value Expr `json:"value"`
	Span  Span `json:"span"`
}

func (*ReturnStmt) statementNode() {}

// BreakStmt exits the nearest loop.
type BreakStmt struct {
	Span Span `json:"span"`
}

func (*BreakStmt) statementNode() {}

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Expr Expr `json:"expr"`
	Span Span `json:"span"`
}

func (*ExprStmt) statementNode() {}

// Identifier is a name reference expression.
type Identifier struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

func (*Identifier) exprNode() {}

// PlaceholderExpr represents the `_` placeholder expression.
type PlaceholderExpr struct {
	Span Span `json:"span"`
}

func (*PlaceholderExpr) exprNode() {}

// IntegerLiteral is a source integer literal.
type IntegerLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*IntegerLiteral) exprNode() {}

// FloatLiteral is a source floating-point literal.
type FloatLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*FloatLiteral) exprNode() {}

// RuneLiteral is a source rune literal.
type RuneLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*RuneLiteral) exprNode() {}

// BoolLiteral is a source boolean literal.
type BoolLiteral struct {
	Value bool `json:"value"`
	Span  Span `json:"span"`
}

func (*BoolLiteral) exprNode() {}

// StringLiteral is a source string literal.
type StringLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*StringLiteral) exprNode() {}

// ListLiteral is a bracketed list literal expression.
type ListLiteral struct {
	Elements []Expr `json:"elements"`
	Span     Span   `json:"span"`
}

func (*ListLiteral) exprNode() {}

// CallExpr applies a callee expression to argument expressions.
type CallExpr struct {
	Callee Expr   `json:"callee"`
	Args   []Expr `json:"args"`
	Span   Span   `json:"span"`
}

func (*CallExpr) exprNode() {}

// MemberExpr reads a named member from a receiver expression.
type MemberExpr struct {
	Receiver Expr   `json:"receiver"`
	Name     string `json:"name"`
	Span     Span   `json:"span"`
}

func (*MemberExpr) exprNode() {}

// IndexExpr indexes into a receiver expression with another expression.
type IndexExpr struct {
	Receiver Expr `json:"receiver"`
	Index    Expr `json:"index"`
	Span     Span `json:"span"`
}

func (*IndexExpr) exprNode() {}

// IfExpr is an if / else expression whose value comes from the chosen branch.
type IfExpr struct {
	Condition Expr       `json:"condition"`
	Then      *BlockStmt `json:"then"`
	Else      *BlockStmt `json:"else"`
	Span      Span       `json:"span"`
}

func (*IfExpr) exprNode() {}

// ForYieldExpr is a yield-style for expression that evaluates to a list.
type ForYieldExpr struct {
	Bindings  []ForBinding `json:"bindings"`
	YieldBody *BlockStmt   `json:"yieldBody"`
	Span      Span         `json:"span"`
}

func (*ForYieldExpr) exprNode() {}

// LambdaParameter describes a single lambda parameter.
type LambdaParameter struct {
	Name string   `json:"name"`
	Type *TypeRef `json:"type,omitempty"`
	Span Span     `json:"span"`
}

// LambdaExpr represents an expression-bodied or block-bodied lambda.
type LambdaExpr struct {
	Parameters []LambdaParameter `json:"parameters"`
	Body       Expr              `json:"body,omitempty"`
	BlockBody  *BlockStmt        `json:"blockBody,omitempty"`
	Span       Span              `json:"span"`
}

func (*LambdaExpr) exprNode() {}

// BinaryExpr applies a binary operator to two expressions.
type BinaryExpr struct {
	Left     Expr   `json:"left"`
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
	Span     Span   `json:"span"`
}

func (*BinaryExpr) exprNode() {}

// UnaryExpr applies a unary operator to one expression.
type UnaryExpr struct {
	Operator string `json:"operator"`
	Right    Expr   `json:"right"`
	Span     Span   `json:"span"`
}

func (*UnaryExpr) exprNode() {}

// GroupExpr preserves parenthesized expression structure.
type GroupExpr struct {
	Inner Expr `json:"inner"`
	Span  Span `json:"span"`
}

func (*GroupExpr) exprNode() {}
