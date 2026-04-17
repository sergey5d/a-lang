package golang

import (
	"a-lang/lower"
	"a-lang/typecheck"
	"bytes"
	"fmt"
	"go/format"
	"strconv"
	"strings"
)

type Generator struct {
	b            strings.Builder
	indent       int
	currentClass string
}

func Generate(program *lower.Program) ([]byte, error) {
	g := &Generator{}
	if err := g.writeProgram(program); err != nil {
		return nil, err
	}
	src := []byte(g.b.String())
	formatted, err := format.Source(src)
	if err != nil {
		return src, fmt.Errorf("format generated go: %w", err)
	}
	return formatted, nil
}

func (g *Generator) writeProgram(program *lower.Program) error {
	g.line("package generated")
	g.line("")

	for _, global := range program.Globals {
		if err := g.writeGlobal(global); err != nil {
			return err
		}
		g.line("")
	}

	for _, class := range program.Classes {
		if err := g.writeClass(class); err != nil {
			return err
		}
		g.line("")
	}

	for _, fn := range program.Functions {
		if err := g.writeFunction(fn); err != nil {
			return err
		}
		g.line("")
	}

	return nil
}

func (g *Generator) writeGlobal(global *lower.Global) error {
	name := goIdent(global.Name)
	typ := goType(global.Type)
	if global.Init == nil {
		g.linef("var %s %s", name, typ)
		return nil
	}
	initExpr, err := g.expr(global.Init)
	if err != nil {
		return err
	}
	g.linef("var %s %s = %s", name, typ, initExpr)
	return nil
}

func (g *Generator) writeClass(class *lower.Class) error {
	g.linef("type %s struct {", goTypeName(class.Name))
	g.indent++
	for _, field := range class.Fields {
		g.linef("%s %s", goIdent(field.Name), goType(field.Type))
	}
	g.indent--
	g.line("}")
	g.line("")

	if err := g.writeConstructor(class); err != nil {
		return err
	}
	for _, method := range class.Methods {
		g.line("")
		if err := g.writeMethod(class.Name, method); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) writeConstructor(class *lower.Class) error {
	fn := class.Constructor
	name := constructorName(class.Name)
	if fn == nil {
		g.linef("func %s() *%s {", name, goTypeName(class.Name))
		g.indent++
		g.linef("return &%s{}", goTypeName(class.Name))
		g.indent--
		g.line("}")
		return nil
	}

	params := g.params(fn.Parameters)
	g.linef("func %s(%s) *%s {", name, params, goTypeName(class.Name))
	g.indent++
	g.linef("this := &%s{}", goTypeName(class.Name))
	prev := g.currentClass
	g.currentClass = class.Name
	if err := g.writeStmtBlock(fn.Body); err != nil {
		return err
	}
	g.currentClass = prev
	g.line("return this")
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeMethod(className string, fn *lower.Function) error {
	receiver := goIdent("this")
	g.linef("func (%s *%s) %s(%s) %s {", receiver, goTypeName(className), goIdent(fn.Name), g.params(fn.Parameters), goType(fn.ReturnType))
	g.indent++
	prev := g.currentClass
	g.currentClass = className
	if err := g.writeStmtBlock(fn.Body); err != nil {
		return err
	}
	g.currentClass = prev
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeFunction(fn *lower.Function) error {
	g.linef("func %s(%s) %s {", goIdent(fn.Name), g.params(fn.Parameters), goType(fn.ReturnType))
	g.indent++
	if err := g.writeStmtBlock(fn.Body); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeStmtBlock(stmts []lower.Stmt) error {
	for _, stmt := range stmts {
		if err := g.writeStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) writeStmt(stmt lower.Stmt) error {
	switch s := stmt.(type) {
	case *lower.VarDecl:
		name := goIdent(s.Name)
		typ := goType(s.Type)
		if s.Init == nil {
			g.linef("var %s %s", name, typ)
			return nil
		}
		initExpr, err := g.expr(s.Init)
		if err != nil {
			return err
		}
		g.linef("var %s %s = %s", name, typ, initExpr)
	case *lower.Assign:
		target, err := g.expr(s.Target)
		if err != nil {
			return err
		}
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		op := s.Operator
		if op == ":=" {
			op = "="
		}
		g.linef("%s %s %s", target, op, value)
	case *lower.If:
		cond, err := g.expr(s.Condition)
		if err != nil {
			return err
		}
		g.linef("if %s {", cond)
		g.indent++
		if err := g.writeStmtBlock(s.Then); err != nil {
			return err
		}
		g.indent--
		if len(s.Else) == 0 {
			g.line("}")
			return nil
		}
		g.line("} else {")
		g.indent++
		if err := g.writeStmtBlock(s.Else); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.ForEach:
		iterable, err := g.expr(s.Iterable)
		if err != nil {
			return err
		}
		g.linef("for _, %s := range %s {", goIdent(s.Name), iterable)
		g.indent++
		if err := g.writeStmtBlock(s.Body); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.Loop:
		g.line("for {")
		g.indent++
		if err := g.writeStmtBlock(s.Body); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.Return:
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.linef("return %s", value)
	case *lower.Break:
		g.line("break")
	case *lower.ExprStmt:
		expr, err := g.expr(s.Expr)
		if err != nil {
			return err
		}
		g.line(expr)
	default:
		return fmt.Errorf("unsupported lowered statement %T", stmt)
	}
	return nil
}

func (g *Generator) expr(expr lower.Expr) (string, error) {
	switch e := expr.(type) {
	case *lower.VarRef:
		return goIdent(e.Name), nil
	case *lower.ThisRef:
		return "this", nil
	case *lower.IntLiteral:
		return strconv.FormatInt(e.Value, 10), nil
	case *lower.FloatLiteral:
		return strconv.FormatFloat(e.Value, 'g', -1, 64), nil
	case *lower.BoolLiteral:
		if e.Value {
			return "true", nil
		}
		return "false", nil
	case *lower.StringLiteral:
		return strconv.Quote(e.Value), nil
	case *lower.RuneLiteral:
		return strconv.QuoteRune(e.Value), nil
	case *lower.ListLiteral:
		items := make([]string, len(e.Elements))
		for i, item := range e.Elements {
			value, err := g.expr(item)
			if err != nil {
				return "", err
			}
			items[i] = value
		}
		return fmt.Sprintf("%s{%s}", goCompositeType(e.Type), strings.Join(items, ", ")), nil
	case *lower.Unary:
		right, err := g.expr(e.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s%s)", e.Operator, right), nil
	case *lower.Binary:
		left, err := g.expr(e.Left)
		if err != nil {
			return "", err
		}
		right, err := g.expr(e.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, e.Operator, right), nil
	case *lower.FunctionCall:
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", goIdent(e.Name), strings.Join(args, ", ")), nil
	case *lower.BuiltinCall:
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		switch e.Name {
		case "append":
			return fmt.Sprintf("append(%s)", strings.Join(args, ", ")), nil
		default:
			return "", fmt.Errorf("builtin call %q is not supported by Go generation yet", e.Name)
		}
	case *lower.Construct:
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", constructorName(e.Class), strings.Join(args, ", ")), nil
	case *lower.FieldGet:
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s", receiver, goIdent(e.Name)), nil
	case *lower.IndexGet:
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		index, err := g.expr(e.Index)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s[%s]", receiver, index), nil
	case *lower.MethodCall:
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s(%s)", receiver, goIdent(e.Method), strings.Join(args, ", ")), nil
	case *lower.Lambda:
		return g.lambdaExpr(e)
	case *lower.Invoke:
		callee, err := g.expr(e.Callee)
		if err != nil {
			return "", err
		}
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", callee, strings.Join(args, ", ")), nil
	default:
		return "", fmt.Errorf("unsupported lowered expression %T", expr)
	}
}

func (g *Generator) lambdaExpr(expr *lower.Lambda) (string, error) {
	var b strings.Builder
	b.WriteString("func(")
	for i, param := range expr.Parameters {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(goIdent(param.Name))
		b.WriteByte(' ')
		b.WriteString(goType(param.Type))
	}
	b.WriteString(")")
	if expr.ReturnType != nil {
		ret := goType(expr.ReturnType)
		if ret != "" {
			b.WriteByte(' ')
			b.WriteString(ret)
		}
	}
	b.WriteString(" {\n")

	sub := &Generator{
		indent:       1,
		currentClass: g.currentClass,
	}
	if err := sub.writeStmtBlock(expr.Body); err != nil {
		return "", err
	}
	b.WriteString(sub.b.String())
	b.WriteString("}")
	return b.String(), nil
}

func (g *Generator) exprList(exprs []lower.Expr) ([]string, error) {
	out := make([]string, len(exprs))
	for i, expr := range exprs {
		value, err := g.expr(expr)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

func (g *Generator) params(params []lower.Parameter) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, param := range params {
		parts[i] = fmt.Sprintf("%s %s", goIdent(param.Name), goType(param.Type))
	}
	return strings.Join(parts, ", ")
}

func (g *Generator) line(text string) {
	g.b.WriteString(strings.Repeat("\t", g.indent))
	g.b.WriteString(text)
	g.b.WriteByte('\n')
}

func (g *Generator) linef(format string, args ...any) {
	g.line(fmt.Sprintf(format, args...))
}

func goType(t *typecheck.Type) string {
	if t == nil {
		return "any"
	}
	switch t.Kind {
	case typecheck.TypeBuiltin:
		switch t.Name {
		case "Int", "Int64":
			return "int64"
		case "Float", "Float64", "Decimal":
			return "float64"
		case "Bool":
			return "bool"
		case "String":
			return "string"
		case "Rune":
			return "rune"
		case "Array", "List", "Set":
			if len(t.Args) == 1 {
				return "[]" + goType(t.Args[0])
			}
			return "[]any"
		case "Map":
			if len(t.Args) == 2 {
				return "map[" + goType(t.Args[0]) + "]" + goType(t.Args[1])
			}
			return "map[any]any"
		default:
			return goTypeName(t.Name)
		}
	case typecheck.TypeClass:
		return "*" + goTypeName(t.Name)
	case typecheck.TypeInterface:
		return goTypeName(t.Name)
	case typecheck.TypeFunction:
		if t.Signature == nil {
			return "func()"
		}
		params := make([]string, len(t.Signature.Parameters))
		for i, param := range t.Signature.Parameters {
			params[i] = goType(param)
		}
		return "func(" + strings.Join(params, ", ") + ") " + goType(t.Signature.ReturnType)
	default:
		return "any"
	}
}

func goCompositeType(t *typecheck.Type) string {
	if t == nil {
		return "[]any"
	}
	if t.Kind == typecheck.TypeBuiltin && (t.Name == "Array" || t.Name == "List" || t.Name == "Set") {
		return goType(t)
	}
	return goType(t)
}

func constructorName(class string) string {
	return "new" + goTypeName(class)
}

func goTypeName(name string) string {
	return sanitize(name)
}

func goIdent(name string) string {
	return sanitize(name)
}

func sanitize(name string) string {
	if name == "" {
		return "_"
	}
	var buf bytes.Buffer
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			buf.WriteRune(r)
			continue
		}
		if i == 0 && r >= '0' && r <= '9' {
			buf.WriteByte('_')
			buf.WriteRune(r)
			continue
		}
		buf.WriteByte('_')
	}
	out := buf.String()
	if out == "" {
		out = "_"
	}
	switch out {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return out + "_"
	default:
		return out
	}
}
