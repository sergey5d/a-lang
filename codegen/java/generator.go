package java

import (
	"a-lang/lower"
	"a-lang/parser"
	"a-lang/predef"
	"a-lang/typecheck"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const defaultClassName = "Generated"

type Generator struct {
	b           strings.Builder
	indent      int
	tempID      int
	moduleClass string
	packageName string
	classes     map[string]*lower.Class
	objects     map[string]*lower.Class
	inObject    bool
	thisClass   *lower.Class
}

func Generate(program *lower.Program) ([]byte, error) {
	return GenerateForPackage(program, "")
}

func GenerateNamed(program *lower.Program, className string) ([]byte, error) {
	return GenerateForPackageNamed(program, "", className)
}

func GenerateForPackage(program *lower.Program, packageName string) ([]byte, error) {
	return GenerateForPackageNamed(program, packageName, ModuleClassName(packageName))
}

func GenerateForPackageSource(program *lower.Program, packageName, sourcePath string) ([]byte, error) {
	return GenerateForPackageNamed(program, packageName, ModuleClassNameFor(packageName, sourcePath))
}

func GenerateForPackageNamed(program *lower.Program, packageName, className string) ([]byte, error) {
	g := &Generator{
		moduleClass: sanitizeTypeName(className),
		packageName: javaPackageName(packageName),
		classes:     map[string]*lower.Class{},
		objects:     map[string]*lower.Class{},
	}
	if g.moduleClass == "" {
		g.moduleClass = defaultClassName
	}
	for _, class := range program.Classes {
		if class.Object {
			g.objects[class.Name] = class
			continue
		}
		g.classes[class.Name] = class
	}
	if err := g.writeProgram(program); err != nil {
		return nil, err
	}
	return []byte(g.b.String()), nil
}

func WriteStdlibSupport(baseDir string) error {
	for rel, src := range stdlibSources() {
		path := filepath.Join(baseDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create stdlib output dir: %w", err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			return fmt.Errorf("write stdlib source %s: %w", rel, err)
		}
	}
	return nil
}

func stdlibSources() map[string]string {
	sources := map[string]string{
		filepath.Join("alang", "stdlib", "Option.java"): `package alang.stdlib;

public final class Option<T> {
    private static final Option<?> NONE = new Option<>(false, null);

    private final boolean set;
    private final T value;

    private Option(boolean set, T value) {
        this.set = set;
        this.value = value;
    }

    public static <T> Option<T> some(T value) {
        return new Option<>(true, value);
    }

    @SuppressWarnings("unchecked")
    public static <T> Option<T> none() {
        return (Option<T>) NONE;
    }

    public boolean isSet() {
        return this.set;
    }

    public boolean isEmpty() {
        return !this.set;
    }

    public T expect() {
        if (!this.set) {
            throw new IllegalStateException("Option.None");
        }
        return this.value;
    }

    public T getOr(T defaultValue) {
        return this.set ? this.value : defaultValue;
    }

    public T getOrElse(T defaultValue) {
        return this.getOr(defaultValue);
    }
}
`,
		filepath.Join("alang", "stdlib", "OS.java"): `package alang.stdlib;

public final class OS {
    private OS() {}

    public static void print(Object value) {
        System.out.print(String.valueOf(value));
    }

    public static void println(Object value) {
        System.out.println(String.valueOf(value));
    }

    public static void printf(String format, Object... values) {
        System.out.printf(format, values);
    }

    public static void panic(Object value) {
        throw new RuntimeException(String.valueOf(value));
    }
}
`,
	}

	if src, err := bundledStdlibFile(filepath.Join("alang", "stdlib", "List.java")); err == nil {
		sources[filepath.Join("alang", "stdlib", "List.java")] = src
	}
	if src, err := bundledStdlibFile(filepath.Join("alang", "stdlib", "Set.java")); err == nil {
		sources[filepath.Join("alang", "stdlib", "Set.java")] = src
	}
	if src, err := bundledStdlibFile(filepath.Join("alang", "stdlib", "Map.java")); err == nil {
		sources[filepath.Join("alang", "stdlib", "Map.java")] = src
	}

	if registry, err := predef.Load(); err == nil {
		for arity := 2; arity <= 10; arity++ {
			name := fmt.Sprintf("Tuple%d", arity)
			decl, ok := registry.Types[name]
			if !ok || decl.Kind != predef.KindRecord {
				continue
			}
			if src, err := tupleJavaSourceFromDescriptor(decl); err == nil {
				sources[filepath.Join("alang", "stdlib", name+".java")] = src
			}
		}
	}

	return sources
}

func tupleJavaSourceFromDescriptor(desc predef.TypeDescriptor) (string, error) {
	if desc.Kind != predef.KindRecord {
		return "", fmt.Errorf("%s is not a record", desc.Name)
	}
	typeParams := make([]string, len(desc.TypeParameters))
	for i, param := range desc.TypeParameters {
		typeParams[i] = param.Name
	}
	fields := make([]string, len(desc.Fields))
	params := make([]string, len(desc.Fields))
	assignments := make([]string, len(desc.Fields))
	for i, field := range desc.Fields {
		fieldType, err := javaTypeRefFromTypeRef(field.Type)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", desc.Name, field.Name, err)
		}
		fields[i] = fmt.Sprintf("    public final %s %s;", fieldType, field.Name)
		params[i] = fmt.Sprintf("%s %s", fieldType, field.Name)
		assignments[i] = fmt.Sprintf("        this.%s = %s;", field.Name, field.Name)
	}
	typeParamSuffix := ""
	if len(typeParams) > 0 {
		typeParamSuffix = "<" + strings.Join(typeParams, ", ") + ">"
	}
	return fmt.Sprintf(`package alang.stdlib;

public final class %s%s {
%s

    public %s(%s) {
%s
    }
}
`, desc.Name, typeParamSuffix, strings.Join(fields, "\n"), desc.Name, strings.Join(params, ", "), strings.Join(assignments, "\n")), nil
}

func javaTypeRefFromTypeRef(ref *parser.TypeRef) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("nil type ref")
	}
	switch ref.Name {
	case "Int", "Int64", "Rune":
		return "Long", nil
	case "Float", "Float64", "Decimal":
		return "Double", nil
	case "Bool":
		return "Boolean", nil
	case "Str":
		return "String", nil
	}
	if len(ref.Arguments) == 0 {
		return ref.Name, nil
	}
	args := make([]string, len(ref.Arguments))
	for i, arg := range ref.Arguments {
		rendered, err := javaTypeRefFromTypeRef(arg)
		if err != nil {
			return "", err
		}
		args[i] = rendered
	}
	return ref.Name + "<" + strings.Join(args, ", ") + ">", nil
}

func bundledStdlibFile(rel string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve generator file path")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "java", "stdlib", rel)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (g *Generator) writeProgram(program *lower.Program) error {
	if g.packageName != "" {
		g.linef("package %s;", g.packageName)
		g.line("")
	}
	g.line("import alang.stdlib.*;")
	g.line("")

	g.linef("public final class %s {", g.moduleClass)
	g.indent++

	for _, global := range program.Globals {
		if err := g.writeGlobal(global); err != nil {
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

	if entry := g.findJavaEntry(program.Functions); entry != nil {
		g.line("public static void main(String[] args) {")
		g.indent++
		g.linef("%s();", javaIdent(entry.Name))
		g.indent--
		g.line("}")
		g.line("")
	}

	g.indent--
	g.line("}")

	for _, class := range program.Classes {
		g.line("")
		if err := g.writeClass(class); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) writeGlobal(global *lower.Global) error {
	typ, err := g.javaType(global.Type, false)
	if err != nil {
		return err
	}
	modifier := "public static"
	if !global.Mutable {
		modifier += " final"
	}
	name := javaIdent(global.Name)
	if global.Init == nil {
		g.linef("%s %s %s;", modifier, typ, name)
		return nil
	}
	initExpr, err := g.exprWithExpected(global.Init, global.Type)
	if err != nil {
		return err
	}
	g.linef("%s %s %s = %s;", modifier, typ, name, initExpr)
	return nil
}

func (g *Generator) writeClass(class *lower.Class) error {
	name := g.className(class)
	g.linef("final class %s {", name)
	g.indent++

	prevClass := g.thisClass
	prevInObject := g.inObject
	g.thisClass = class
	g.inObject = class.Object
	defer func() {
		g.thisClass = prevClass
		g.inObject = prevInObject
	}()

	if class.Enum {
		return g.writeEnumClass(class)
	}

	if class.Object {
		g.linef("static final %s INSTANCE = new %s();", name, name)
		g.line("")
	}

	for _, field := range class.Fields {
		if err := g.writeField(field); err != nil {
			return err
		}
	}
	if len(class.Fields) > 0 {
		g.line("")
	}

	if class.Object {
		g.linef("private %s() {}", name)
		g.line("")
	} else if class.Constructor != nil {
		if err := g.writeConstructor(class); err != nil {
			return err
		}
		g.line("")
	} else {
		g.linef("public %s() {}", name)
		g.line("")
	}

	for i, method := range class.Methods {
		if err := g.writeMethod(class, method); err != nil {
			return err
		}
		if i < len(class.Methods)-1 {
			g.line("")
		}
	}

	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeEnumClass(class *lower.Class) error {
	name := g.className(class)
	allFields := g.enumUnionFields(class)
	g.line(`private final String __tag;`)
	for _, field := range allFields {
		if err := g.writeField(field); err != nil {
			return err
		}
	}
	g.line("")

	constructorParams := []string{"String __tag"}
	for _, field := range allFields {
		typ, err := g.javaType(field.Type, false)
		if err != nil {
			return err
		}
		constructorParams = append(constructorParams, typ+" "+javaIdent(field.Name))
	}
	g.linef("private %s(%s) {", name, strings.Join(constructorParams, ", "))
	g.indent++
	g.line("this.__tag = __tag;")
	for _, field := range allFields {
		g.linef("this.%s = %s;", javaIdent(field.Name), javaIdent(field.Name))
	}
	g.indent--
	g.line("}")
	g.line("")

	for _, enumCase := range class.Cases {
		if len(enumCase.Fields) == 0 {
			g.linef("static final %s %s = new %s(%s);", name, javaIdent(enumCase.Name), name, g.enumConstructorArgs(enumCase.Name, allFields, nil))
			continue
		}
		params := make([]lower.Parameter, len(enumCase.Fields))
		for i, field := range enumCase.Fields {
			params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
		}
		g.linef("public static %s %s(%s) {", name, javaIdent(enumCase.Name), g.params(params))
		g.indent++
		g.linef("return new %s(%s);", name, g.enumConstructorArgs(enumCase.Name, allFields, enumCase.Fields))
		g.indent--
		g.line("}")
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) enumUnionFields(class *lower.Class) []*lower.Field {
	seen := map[string]bool{}
	var out []*lower.Field
	for _, field := range class.Fields {
		if seen[field.Name] {
			continue
		}
		seen[field.Name] = true
		out = append(out, field)
	}
	for _, enumCase := range class.Cases {
		for _, field := range enumCase.Fields {
			if seen[field.Name] {
				continue
			}
			seen[field.Name] = true
			out = append(out, field)
		}
	}
	return out
}

func (g *Generator) enumConstructorArgs(caseName string, unionFields []*lower.Field, caseFields []*lower.Field) string {
	args := []string{strconv.Quote(caseName)}
	caseFieldMap := map[string]*lower.Field{}
	for _, field := range caseFields {
		caseFieldMap[field.Name] = field
	}
	for _, field := range unionFields {
		if _, ok := caseFieldMap[field.Name]; ok {
			args = append(args, javaIdent(field.Name))
			continue
		}
		args = append(args, g.zeroValueJava(field.Type))
	}
	return strings.Join(args, ", ")
}

func (g *Generator) writeField(field *lower.Field) error {
	typ, err := g.javaType(field.Type, false)
	if err != nil {
		return err
	}
	name := javaIdent(field.Name)
	prefix := "public " + typ + " " + name
	if field.Private {
		prefix = typ + " " + name
	}
	if !field.Mutable {
		if field.Private {
			prefix = "final " + typ + " " + name
		} else {
			prefix = "public final " + typ + " " + name
		}
	}
	if field.Init == nil {
		g.linef("%s;", prefix)
		return nil
	}
	initExpr, err := g.exprWithExpected(field.Init, field.Type)
	if err != nil {
		return err
	}
	g.linef("%s = %s;", prefix, initExpr)
	return nil
}

func (g *Generator) writeConstructor(class *lower.Class) error {
	fn := class.Constructor
	header := "public " + g.className(class) + "(" + g.params(fn.Parameters) + ")"
	if fn.Private {
		header = g.className(class) + "(" + g.params(fn.Parameters) + ")"
	}
	g.linef("%s {", header)
	g.indent++
	if err := g.writeCallableBody(fn.Body, fn.ReturnType, true); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeMethod(class *lower.Class, fn *lower.Function) error {
	returnType, err := g.javaType(fn.ReturnType, true)
	if err != nil {
		return err
	}
	header := "public " + returnType + " " + javaIdent(fn.Name) + "(" + g.params(fn.Parameters) + ")"
	if fn.Private {
		header = returnType + " " + javaIdent(fn.Name) + "(" + g.params(fn.Parameters) + ")"
	}
	g.linef("%s {", header)
	g.indent++
	if err := g.writeCallableBody(fn.Body, fn.ReturnType, false); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeFunction(fn *lower.Function) error {
	returnType, err := g.javaType(fn.ReturnType, true)
	if err != nil {
		return err
	}
	access := "public static"
	if fn.Private {
		access = "private static"
	}
	g.linef("%s %s %s(%s) {", access, returnType, javaIdent(fn.Name), g.params(fn.Parameters))
	g.indent++
	if err := g.writeCallableBody(fn.Body, fn.ReturnType, false); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) findJavaEntry(functions []*lower.Function) *lower.Function {
	for _, fn := range functions {
		if fn.Name == "run" && len(fn.Parameters) == 0 {
			return fn
		}
	}
	for _, fn := range functions {
		if fn.Name == "main" && len(fn.Parameters) == 0 {
			return fn
		}
	}
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

func (g *Generator) writeCallableBody(stmts []lower.Stmt, returnType *typecheck.Type, constructor bool) error {
	for i, stmt := range stmts {
		if !constructor && !isUnitType(returnType) && i == len(stmts)-1 {
			if exprStmt, ok := stmt.(*lower.ExprStmt); ok {
				value, err := g.expr(exprStmt.Expr)
				if err != nil {
					return err
				}
				g.linef("return %s;", value)
				return nil
			}
		}
		if err := g.writeStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) writeStmt(stmt lower.Stmt) error {
	switch s := stmt.(type) {
	case *lower.VarDecl:
		typ, err := g.javaType(s.Type, false)
		if err != nil {
			return err
		}
		if s.Init == nil {
			g.linef("%s %s;", typ, javaIdent(s.Name))
			return nil
		}
		initExpr, err := g.exprWithExpected(s.Init, s.Type)
		if err != nil {
			return err
		}
		g.linef("%s %s = %s;", typ, javaIdent(s.Name), initExpr)
	case *lower.Assign:
		target, err := g.expr(s.Target)
		if err != nil {
			return err
		}
		value, err := g.exprWithExpected(s.Value, g.assignmentTargetType(s.Target))
		if err != nil {
			return err
		}
		op := s.Operator
		if op == ":=" {
			op = "="
		}
		g.linef("%s %s %s;", target, op, value)
	case *lower.If:
		cond, err := g.expr(s.Condition)
		if err != nil {
			return err
		}
		g.linef("if (%s) {", unwrapGroupedJavaExpr(cond))
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
		if ok, err := g.writeRangeLoop(s); err != nil {
			return err
		} else if ok {
			return nil
		}
		iterable, err := g.expr(s.Iterable)
		if err != nil {
			return err
		}
		itemType, err := g.foreachItemType(s.Iterable)
		if err != nil {
			return err
		}
		g.linef("for (%s %s : %s) {", itemType, javaIdent(s.Name), iterable)
		g.indent++
		if err := g.writeStmtBlock(s.Body); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.Loop:
		g.line("while (true) {")
		g.indent++
		if err := g.writeStmtBlock(s.Body); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.Return:
		if s.Value == nil {
			g.line("return;")
			return nil
		}
		value, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.linef("return %s;", value)
	case *lower.Break:
		g.line("break;")
	case *lower.ExprStmt:
		expr, err := g.expr(s.Expr)
		if err != nil {
			return err
		}
		g.linef("%s;", expr)
	default:
		return fmt.Errorf("unsupported lowered statement %T", stmt)
	}
	return nil
}

func (g *Generator) writeRangeLoop(loop *lower.ForEach) (bool, error) {
	start, end, err := g.rangeBounds(loop.Iterable)
	if err != nil {
		return false, err
	}
	if start == "" && end == "" {
		return false, nil
	}
	g.linef("for (long %s = %s; %s < %s; %s++) {", javaIdent(loop.Name), start, javaIdent(loop.Name), end, javaIdent(loop.Name))
	g.indent++
	if err := g.writeStmtBlock(loop.Body); err != nil {
		return false, err
	}
	g.indent--
	g.line("}")
	return true, nil
}

func (g *Generator) rangeBounds(iterable lower.Expr) (string, string, error) {
	t := exprType(iterable)
	if t == nil || t.Kind != typecheck.TypeTuple || len(t.Args) != 2 {
		return "", "", nil
	}
	for _, arg := range t.Args {
		if arg == nil || arg.Kind != typecheck.TypeBuiltin || (arg.Name != "Int" && arg.Name != "Int64") {
			return "", "", nil
		}
	}
	switch e := iterable.(type) {
	case *lower.TupleLiteral:
		start, err := g.expr(e.Elements[0])
		if err != nil {
			return "", "", err
		}
		end, err := g.expr(e.Elements[1])
		if err != nil {
			return "", "", err
		}
		return start, end, nil
	default:
		iterableExpr, err := g.expr(iterable)
		if err != nil {
			return "", "", err
		}
		temp := g.nextTemp("range")
		typeName, err := g.javaType(t, false)
		if err != nil {
			return "", "", err
		}
		g.linef("%s %s = %s;", typeName, temp, iterableExpr)
		return temp + "._1", temp + "._2", nil
	}
}

func (g *Generator) expr(expr lower.Expr) (string, error) {
	switch e := expr.(type) {
	case *lower.VarRef:
		if e.Name == "OS" {
			return "OS", nil
		}
		if e.Type != nil && e.Type.Kind == typecheck.TypeObject {
			if _, ok := g.objects[e.Name]; ok {
				return g.className(g.objects[e.Name]) + ".INSTANCE", nil
			}
		}
		return javaIdent(e.Name), nil
	case *lower.ThisRef:
		return "this", nil
	case *lower.IntLiteral:
		return strconv.FormatInt(e.Value, 10) + "L", nil
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
		return strconv.FormatInt(int64(e.Value), 10) + "L", nil
	case *lower.ListLiteral:
		return g.listLiteral(e)
	case *lower.TupleLiteral:
		return g.tupleLiteral(e)
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
	case *lower.IfExpr:
		if len(e.ThenPrefix) > 0 || len(e.ElsePrefix) > 0 {
			return "", fmt.Errorf("unsupported lowered expression %T with branch statements", expr)
		}
		condition, err := g.expr(e.Condition)
		if err != nil {
			return "", err
		}
		thenValue, err := g.expr(e.ThenValue)
		if err != nil {
			return "", err
		}
		elseValue, err := g.expr(e.ElseValue)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("((%s) ? %s : %s)", condition, thenValue, elseValue), nil
	case *lower.FunctionCall:
		if e.Name == "Array" {
			return g.arrayLiteral(e)
		}
		if e.Name == "List" {
			return g.collectionLiteral("List", e.Args)
		}
		if e.Name == "Set" {
			return g.collectionLiteral("Set", e.Args)
		}
		if e.Name == "Some" {
			if len(e.Args) != 1 {
				return "", fmt.Errorf("Some expects 1 argument")
			}
			value, err := g.expr(e.Args[0])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Option.some(%s)", value), nil
		}
		if e.Name == "None" {
			if len(e.Args) != 0 {
				return "", fmt.Errorf("None expects 0 arguments")
			}
			return "Option.none()", nil
		}
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s(%s)", g.moduleClass, javaIdent(e.Name), args), nil
	case *lower.BuiltinCall:
		return "", fmt.Errorf("unsupported lowered expression %T", expr)
	case *lower.Construct:
		class, ok := g.classes[e.Class]
		if !ok {
			return "", fmt.Errorf("unknown class %q", e.Class)
		}
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("new %s(%s)", g.className(class), args), nil
	case *lower.FieldGet:
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s", receiver, javaIdent(e.Name)), nil
	case *lower.IndexGet:
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		index, err := g.expr(e.Index)
		if err != nil {
			return "", err
		}
		if receiverType := exprType(e.Receiver); receiverType != nil && receiverType.Kind == typecheck.TypeInterface && receiverType.Name == "List" && len(receiverType.Args) == 1 {
			return fmt.Sprintf("%s.get(%s).expect()", receiver, index), nil
		}
		return fmt.Sprintf("%s[(int)(%s)]", receiver, index), nil
	case *lower.MethodCall:
		if receiver, ok := e.Receiver.(*lower.VarRef); ok && receiver.Name == "Array" && e.Method == "ofLength" {
			return g.arrayOfLength(e)
		}
		if receiverType := exprType(e.Receiver); receiverType != nil && receiverType.Kind == typecheck.TypeInterface && receiverType.Name == "List" && e.Method == "sort" && len(e.Args) == 1 {
			return g.listSortCall(e)
		}
		if receiverType := exprType(e.Receiver); receiverType != nil && receiverType.Kind == typecheck.TypeBuiltin && receiverType.Name == "Array" {
			if e.Method == "size" && len(e.Args) == 0 {
				receiver, err := g.expr(e.Receiver)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s.length", receiver), nil
			}
		}
		receiver, err := g.expr(e.Receiver)
		if err != nil {
			return "", err
		}
		args, err := g.exprList(e.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s(%s)", receiver, javaIdent(e.Method), args), nil
	case *lower.Lambda:
		return g.lambdaExpr(e)
	case *lower.Invoke:
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "Array" {
			return g.arrayLiteralFromArgs(e.Args, e.Type)
		}
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "List" {
			return g.collectionLiteral("List", e.Args)
		}
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "Set" {
			return g.collectionLiteral("Set", e.Args)
		}
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "Some" {
			if len(e.Args) != 1 {
				return "", fmt.Errorf("Some expects 1 argument")
			}
			value, err := g.expr(e.Args[0])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Option.some(%s)", value), nil
		}
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "None" {
			if len(e.Args) != 0 {
				return "", fmt.Errorf("None expects 0 arguments")
			}
			return "Option.none()", nil
		}
		return "", fmt.Errorf("unsupported lowered expression %T", expr)
	default:
		return "", fmt.Errorf("unsupported lowered expression %T", expr)
	}
}

func unwrapGroupedJavaExpr(expr string) string {
	expr = strings.TrimSpace(expr)
	if len(expr) < 2 || expr[0] != '(' || expr[len(expr)-1] != ')' {
		return expr
	}
	depth := 0
	for i, r := range expr {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i != len(expr)-1 {
				return expr
			}
		}
		if depth < 0 {
			return expr
		}
	}
	if depth != 0 {
		return expr
	}
	return expr[1 : len(expr)-1]
}

func (g *Generator) exprWithExpected(expr lower.Expr, expected *typecheck.Type) (string, error) {
	switch e := expr.(type) {
	case *lower.MethodCall:
		if receiver, ok := e.Receiver.(*lower.VarRef); ok && receiver.Name == "Array" && e.Method == "ofLength" {
			return g.arrayOfLengthWithExpected(e, expected)
		}
	}
	return g.expr(expr)
}

func (g *Generator) params(params []lower.Parameter) string {
	parts := make([]string, len(params))
	for i, param := range params {
		typ, err := g.javaType(param.Type, false)
		if err != nil {
			parts[i] = "Object " + javaIdent(param.Name)
			continue
		}
		parts[i] = typ + " " + javaIdent(param.Name)
	}
	return strings.Join(parts, ", ")
}

func (g *Generator) exprList(args []lower.Expr) (string, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		value, err := g.expr(arg)
		if err != nil {
			return "", err
		}
		parts[i] = value
	}
	return strings.Join(parts, ", "), nil
}

func (g *Generator) arrayLiteral(call *lower.FunctionCall) (string, error) {
	return g.arrayLiteralFromArgs(call.Args, call.Type)
}

func (g *Generator) listLiteral(list *lower.ListLiteral) (string, error) {
	items, err := g.exprList(list.Elements)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("List.of(%s)", items), nil
}

func (g *Generator) listSortCall(call *lower.MethodCall) (string, error) {
	receiver, err := g.expr(call.Receiver)
	if err != nil {
		return "", err
	}
	ordering, err := g.expr(call.Args[0])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.sort((left, right) -> (int)(%s.compare(left, right)))", receiver, ordering), nil
}

func (g *Generator) collectionLiteral(className string, args []lower.Expr) (string, error) {
	items, err := g.exprList(args)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.of(%s)", className, items), nil
}

func (g *Generator) arrayLiteralFromArgs(args []lower.Expr, typ *typecheck.Type) (string, error) {
	if typ == nil || typ.Kind != typecheck.TypeBuiltin || typ.Name != "Array" || len(typ.Args) != 1 {
		return "", fmt.Errorf("Array(...) requires resolved Array[T] type")
	}
	elemType, err := g.javaType(typ.Args[0], false)
	if err != nil {
		return "", err
	}
	items, err := g.exprList(args)
	if err != nil {
		return "", err
	}
	if strings.Contains(elemType, "<") {
		return fmt.Sprintf("(%s[]) new %s[] {%s}", elemType, eraseJavaGenerics(elemType), items), nil
	}
	return fmt.Sprintf("new %s[] {%s}", elemType, items), nil
}

func (g *Generator) arrayOfLength(call *lower.MethodCall) (string, error) {
	return g.arrayOfLengthWithExpected(call, call.Type)
}

func (g *Generator) arrayOfLengthWithExpected(call *lower.MethodCall, expected *typecheck.Type) (string, error) {
	if len(call.Args) != 1 {
		return "", fmt.Errorf("Array.ofLength expects 1 argument")
	}
	arrayType := call.Type
	if arrayType == nil || arrayType.Kind == typecheck.TypeUnknown || hasUnknownTypeArg(arrayType) {
		arrayType = expected
	}
	if arrayType == nil || arrayType.Kind != typecheck.TypeBuiltin || arrayType.Name != "Array" || len(arrayType.Args) != 1 {
		return "", fmt.Errorf("Array.ofLength requires resolved Array[T] type")
	}
	elemType, err := g.javaType(arrayType.Args[0], false)
	if err != nil {
		return "", err
	}
	length, err := g.expr(call.Args[0])
	if err != nil {
		return "", err
	}
	if strings.Contains(elemType, "<") {
		return fmt.Sprintf("(%s[]) new %s[(int)(%s)]", elemType, eraseJavaGenerics(elemType), length), nil
	}
	return fmt.Sprintf("new %s[(int)(%s)]", elemType, length), nil
}

func eraseJavaGenerics(typeName string) string {
	if idx := strings.IndexByte(typeName, '<'); idx >= 0 {
		return typeName[:idx]
	}
	return typeName
}

func hasUnknownTypeArg(t *typecheck.Type) bool {
	if t == nil {
		return true
	}
	for _, arg := range t.Args {
		if arg == nil || arg.Kind == typecheck.TypeUnknown {
			return true
		}
	}
	return false
}

func (g *Generator) assignmentTargetType(target lower.Expr) *typecheck.Type {
	switch t := target.(type) {
	case *lower.VarRef:
		return t.Type
	case *lower.FieldGet:
		if t.Type != nil && t.Type.Kind != typecheck.TypeUnknown {
			return t.Type
		}
		receiverType := exprType(t.Receiver)
		if receiverType == nil {
			return t.Type
		}
		switch receiverType.Kind {
		case typecheck.TypeClass:
			if class, ok := g.classes[receiverType.Name]; ok {
				for _, field := range class.Fields {
					if field.Name == t.Name {
						return field.Type
					}
				}
			}
		case typecheck.TypeObject:
			if obj, ok := g.objects[receiverType.Name]; ok {
				for _, field := range obj.Fields {
					if field.Name == t.Name {
						return field.Type
					}
				}
			}
		}
		return t.Type
	case *lower.IndexGet:
		receiverType := exprType(t.Receiver)
		if receiverType == nil {
			return t.Type
		}
		if receiverType.Kind == typecheck.TypeBuiltin && receiverType.Name == "Array" && len(receiverType.Args) == 1 {
			return receiverType.Args[0]
		}
		if receiverType.Kind == typecheck.TypeInterface && receiverType.Name == "List" && len(receiverType.Args) == 1 {
			return receiverType.Args[0]
		}
		return t.Type
	default:
		return exprType(target)
	}
}

func (g *Generator) tupleLiteral(tuple *lower.TupleLiteral) (string, error) {
	if tuple.Type == nil || tuple.Type.Kind != typecheck.TypeTuple {
		return "", fmt.Errorf("tuple literal requires resolved tuple type")
	}
	arity := len(tuple.Elements)
	if arity < 2 || arity > 10 {
		return "", fmt.Errorf("tuple arity %d is unsupported", arity)
	}
	args, err := g.exprList(tuple.Elements)
	if err != nil {
		return "", err
	}
	typeName, err := g.javaType(tuple.Type, false)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("new %s(%s)", typeName, args), nil
}

func (g *Generator) foreachItemType(iterable lower.Expr) (string, error) {
	switch e := iterable.(type) {
	case interface{ GetType() *typecheck.Type }:
		_ = e
	}
	switch t := exprType(iterable); {
	case t == nil:
		return "", fmt.Errorf("foreach iterable has no type")
	case t.Kind == typecheck.TypeBuiltin && t.Name == "Array" && len(t.Args) == 1:
		return g.javaType(t.Args[0], false)
	case t.Kind == typecheck.TypeInterface && (t.Name == "List" || t.Name == "Set" || t.Name == "Iterable") && len(t.Args) == 1:
		return g.javaReferenceType(t.Args[0])
	default:
		return "", fmt.Errorf("unsupported foreach iterable type %s", t.String())
	}
}

func exprType(expr lower.Expr) *typecheck.Type {
	switch e := expr.(type) {
	case *lower.VarRef:
		return e.Type
	case *lower.ThisRef:
		return e.Type
	case *lower.IntLiteral:
		return e.Type
	case *lower.FloatLiteral:
		return e.Type
	case *lower.BoolLiteral:
		return e.Type
	case *lower.StringLiteral:
		return e.Type
	case *lower.RuneLiteral:
		return e.Type
	case *lower.ListLiteral:
		return e.Type
	case *lower.TupleLiteral:
		return e.Type
	case *lower.Unary:
		return e.Type
	case *lower.Binary:
		return e.Type
	case *lower.IfExpr:
		return e.Type
	case *lower.FunctionCall:
		return e.Type
	case *lower.BuiltinCall:
		return e.Type
	case *lower.Construct:
		return e.Type
	case *lower.FieldGet:
		return e.Type
	case *lower.IndexGet:
		return e.Type
	case *lower.MethodCall:
		return e.Type
	case *lower.Lambda:
		return e.Type
	case *lower.Invoke:
		return e.Type
	default:
		return nil
	}
}

func isUnitType(t *typecheck.Type) bool {
	return t != nil && t.Kind == typecheck.TypeBuiltin && t.Name == "Unit"
}

func (g *Generator) javaType(t *typecheck.Type, allowVoid bool) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil type")
	}
	switch t.Kind {
	case typecheck.TypeBuiltin:
		switch t.Name {
		case "Unit":
			if allowVoid {
				return "void", nil
			}
			return "", fmt.Errorf("Unit is only valid as a return type")
		case "Int", "Int64", "Rune":
			return "long", nil
		case "Float", "Float64", "Decimal":
			return "double", nil
		case "Bool":
			return "boolean", nil
		case "Str":
			return "String", nil
		case "Array":
			if len(t.Args) != 1 {
				return "", fmt.Errorf("Array requires one type argument")
			}
			elemType, err := g.javaType(t.Args[0], false)
			if err != nil {
				return "", err
			}
			return elemType + "[]", nil
		default:
			return "", fmt.Errorf("unsupported builtin type %s", t.Name)
		}
	case typecheck.TypeClass:
		if t.Name == "Option" {
			return g.optionType(t)
		}
		if class, ok := g.classes[t.Name]; ok {
			return g.className(class), nil
		}
		return sanitizeTypeName(t.Name), nil
	case typecheck.TypeObject:
		if class, ok := g.objects[t.Name]; ok {
			return g.className(class), nil
		}
		return g.objectClassName(t.Name), nil
	case typecheck.TypeInterface:
		if t.Name == "Option" {
			return g.optionType(t)
		}
		if (t.Name == "List" || t.Name == "Set" || t.Name == "Iterable") && len(t.Args) == 1 {
			argType, err := g.javaReferenceType(t.Args[0])
			if err != nil {
				return "", err
			}
			switch t.Name {
			case "List":
				return "List<" + argType + ">", nil
			case "Set":
				return "Set<" + argType + ">", nil
			default:
				return "Iterable<" + argType + ">", nil
			}
		}
		return "", fmt.Errorf("unsupported interface type %s", t.String())
	case typecheck.TypeTuple:
		return g.tupleType(t)
	default:
		return "", fmt.Errorf("unsupported type %s", t.String())
	}
}

func (g *Generator) optionType(t *typecheck.Type) (string, error) {
	if len(t.Args) != 1 {
		return "", fmt.Errorf("Option requires one type argument")
	}
	argType, err := g.javaReferenceType(t.Args[0])
	if err != nil {
		return "", err
	}
	return "Option<" + argType + ">", nil
}

func (g *Generator) tupleType(t *typecheck.Type) (string, error) {
	arity := len(t.Args)
	if arity < 2 || arity > 10 {
		return "", fmt.Errorf("tuple arity %d is unsupported", arity)
	}
	parts := make([]string, arity)
	for i, arg := range t.Args {
		part, err := g.javaReferenceType(arg)
		if err != nil {
			return "", err
		}
		parts[i] = part
	}
	return fmt.Sprintf("Tuple%d<%s>", arity, strings.Join(parts, ", ")), nil
}

func (g *Generator) javaReferenceType(t *typecheck.Type) (string, error) {
	base, err := g.javaType(t, false)
	if err != nil {
		return "", err
	}
	switch base {
	case "long":
		return "Long", nil
	case "double":
		return "Double", nil
	case "boolean":
		return "Boolean", nil
	default:
		return base, nil
	}
}

func (g *Generator) zeroValueJava(t *typecheck.Type) string {
	if t == nil {
		return "null"
	}
	switch t.Kind {
	case typecheck.TypeBuiltin:
		switch t.Name {
		case "Int", "Int64", "Rune":
			return "0L"
		case "Float", "Float64", "Decimal":
			return "0.0"
		case "Bool":
			return "false"
		case "Str":
			return `""`
		case "Array":
			return "null"
		default:
			return "null"
		}
	default:
		return "null"
	}
}

func (g *Generator) className(class *lower.Class) string {
	if class.Object {
		return g.objectClassName(class.Name)
	}
	return sanitizeTypeName(class.Name)
}

func (g *Generator) objectClassName(name string) string {
	return "Object_" + sanitizeTypeName(name)
}

func sanitizeTypeName(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range name {
		switch {
		case unicode.IsLetter(r) || r == '_':
			if i == 0 {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(r)
			}
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteRune('_')
			}
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func ModuleClassName(packageName string) string {
	if packageName == "" {
		return "Pkg_Default"
	}
	return "Pkg_" + sanitizeTypeName(packageName)
}

func ModuleClassNameFor(packageName, sourcePath string) string {
	if packageName != "" {
		return ModuleClassName(packageName)
	}
	stem := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	name := sanitizeFileStem(stem)
	if name == "" {
		return "Pkg_Default"
	}
	return "Pkg_" + name
}

func (g *Generator) nextTemp(prefix string) string {
	g.tempID++
	return "__" + prefix + strconv.Itoa(g.tempID)
}

func (g *Generator) lambdaExpr(lambda *lower.Lambda) (string, error) {
	if len(lambda.Parameters) != 1 {
		return "", fmt.Errorf("unsupported lambda arity %d", len(lambda.Parameters))
	}
	param := javaIdent(lambda.Parameters[0].Name)
	if len(lambda.Body) != 1 {
		return "", fmt.Errorf("unsupported lambda body with %d statements", len(lambda.Body))
	}
	ret, ok := lambda.Body[0].(*lower.Return)
	if !ok || ret.Value == nil {
		return "", fmt.Errorf("unsupported lambda body %T", lambda.Body[0])
	}
	value, err := g.expr(ret.Value)
	if err != nil {
		return "", err
	}
	if isUnitType(lambda.ReturnType) {
		return fmt.Sprintf("%s -> { %s; }", param, value), nil
	}
	return fmt.Sprintf("%s -> %s", param, value), nil
}

func OutputPath(baseDir, packageName string) string {
	parts := packagePathParts(packageName)
	parts = append(parts, ModuleClassName(packageName)+".java")
	all := append([]string{baseDir}, parts...)
	return filepath.Join(all...)
}

func OutputPathFor(baseDir, packageName, sourcePath string) string {
	parts := packagePathParts(packageName)
	parts = append(parts, ModuleClassNameFor(packageName, sourcePath)+".java")
	all := append([]string{baseDir}, parts...)
	return filepath.Join(all...)
}

func javaPackageName(packageName string) string {
	if packageName == "" {
		return ""
	}
	parts := packagePathParts(packageName)
	return strings.Join(parts, ".")
}

func sanitizeFileStem(name string) string {
	if name == "" {
		return ""
	}
	var out strings.Builder
	upperNext := true
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if upperNext && unicode.IsLetter(r) {
				out.WriteRune(unicode.ToUpper(r))
			} else {
				out.WriteRune(r)
			}
			upperNext = false
		default:
			upperNext = true
		}
	}
	return out.String()
}

func packagePathParts(packageName string) []string {
	if packageName == "" {
		return nil
	}
	parts := strings.FieldsFunc(packageName, func(r rune) bool {
		return r == '/' || r == '\\' || r == '.'
	})
	for i, part := range parts {
		part = strings.ToLower(javaIdent(part))
		part = strings.TrimPrefix(part, "_")
		if part == "" {
			part = "pkg"
		}
		parts[i] = part
	}
	return parts
}

func javaIdent(name string) string {
	if name == "" {
		return "_"
	}
	var b strings.Builder
	for i, r := range name {
		switch {
		case unicode.IsLetter(r) || r == '_':
			b.WriteRune(r)
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteRune('_')
			}
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	switch out {
	case "class", "public", "private", "protected", "static", "final", "this", "new", "return", "if", "else", "for", "while", "break", "long", "double", "boolean", "void":
		return "_" + out
	default:
		return out
	}
}

func (g *Generator) line(s string) {
	g.b.WriteString(strings.Repeat("    ", g.indent))
	g.b.WriteString(s)
	g.b.WriteByte('\n')
}

func (g *Generator) linef(format string, args ...any) {
	g.line(fmt.Sprintf(format, args...))
}
