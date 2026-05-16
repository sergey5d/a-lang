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
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const defaultClassName = "Generated"

type Generator struct {
	b                 strings.Builder
	indent            int
	tempID            int
	moduleClass       string
	packageName       string
	classes           map[string]*lower.Class
	objects           map[string]*lower.Class
	records           map[string]*typecheck.Type
	inObject          bool
	thisClass         *lower.Class
	currentReturnType *typecheck.Type
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
		records:     map[string]*typecheck.Type{},
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
	g.collectRecordTypes(program)
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
    public interface Mapper<T, R> {
        R apply(T value);
    }

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

    public <R> Option<R> map(Mapper<? super T, ? extends R> mapper) {
        if (!this.set) {
            return Option.none();
        }
        return Option.some(mapper.apply(this.value));
    }
}
`,
		filepath.Join("alang", "stdlib", "OS.java"): `package alang.stdlib;

public final class OS {
    private OS() {}

    public static void print(Object... values) {
        for (Object value : values) {
            System.out.print(String.valueOf(value));
        }
    }

    public static void println(Object... values) {
        System.out.println(join(values));
    }

    public static void printf(String format, Object... values) {
        System.out.printf(format, values);
    }

    public static void panic(Object value) {
        throw new RuntimeException(String.valueOf(value));
    }

    private static String join(Object... values) {
        if (values.length == 0) {
            return "";
        }
        StringBuilder out = new StringBuilder();
        for (int i = 0; i < values.length; i++) {
            if (i > 0) {
                out.append(" ");
            }
            out.append(String.valueOf(values[i]));
        }
        return out.toString();
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

	for _, recordType := range g.orderedRecordTypes() {
		g.line("")
		if err := g.writeRecordClass(recordType); err != nil {
			return err
		}
	}

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
	if class.Enum {
		return g.writeEnumClass(class)
	}
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
	} else if class.Record {
		if err := g.writeRecordDefaultConstructor(class); err != nil {
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

func (g *Generator) writeRecordClass(recordType *typecheck.Type) error {
	name := g.recordClassName(recordType)
	g.linef("final class %s {", name)
	g.indent++
	for _, field := range recordType.Fields {
		typ, err := g.javaType(field.Type, false)
		if err != nil {
			return err
		}
		g.linef("public final %s %s;", typ, javaIdent(field.Name))
	}
	if len(recordType.Fields) > 0 {
		g.line("")
	}
	params := make([]string, len(recordType.Fields))
	for i, field := range recordType.Fields {
		typ, err := g.javaType(field.Type, false)
		if err != nil {
			return err
		}
		params[i] = typ + " " + javaIdent(field.Name)
	}
	g.linef("public %s(%s) {", name, strings.Join(params, ", "))
	g.indent++
	for _, field := range recordType.Fields {
		g.linef("this.%s = %s;", javaIdent(field.Name), javaIdent(field.Name))
	}
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeEnumClass(class *lower.Class) error {
	name := g.className(class)
	typeDecl := g.typeParamsDecl(class.TypeParameters)
	typeUse := g.typeParamsUse(class.TypeParameters)
	if len(class.Fields) == 0 {
		g.linef("interface %s%s {", name, typeDecl)
		g.indent++
		for _, method := range class.Methods {
			if err := g.writeEnumInterfaceMethod(class, method); err != nil {
				return err
			}
			g.line("")
		}
		for _, enumCase := range class.Cases {
			if len(enumCase.Fields) == 0 {
				fieldType := name
				if len(class.TypeParameters) == 0 {
					fieldType += typeUse
				}
				g.linef("%s %s = %s.INSTANCE;", fieldType, javaIdent(enumCase.Name), g.enumCaseClassName(class, enumCase.Name))
				if len(class.TypeParameters) > 0 {
					methodTypeDecl := typeDecl
					if methodTypeDecl != "" {
						methodTypeDecl += " "
					}
					g.linef("@SuppressWarnings(\"unchecked\")")
					g.linef("static %s%s%s %s() {", methodTypeDecl, name, typeUse, javaIdent(enumCase.Name))
					g.indent++
					g.linef("return (%s%s) %s;", name, typeUse, javaIdent(enumCase.Name))
					g.indent--
					g.line("}")
				}
				continue
			}
			params := make([]lower.Parameter, len(enumCase.Fields))
			for i, field := range enumCase.Fields {
				params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
			}
			methodTypeDecl := g.typeParamsDecl(class.TypeParameters)
			if methodTypeDecl != "" {
				methodTypeDecl += " "
			}
			g.linef("static %s%s%s %s(%s) {", methodTypeDecl, name, typeUse, javaIdent(enumCase.Name), g.params(params))
			g.indent++
			g.linef("return new %s%s(%s);", g.enumCaseClassName(class, enumCase.Name), typeUse, g.enumCasePayloadArgs(enumCase.Fields))
			g.indent--
			g.line("}")
		}
		g.indent--
		g.line("}")
	} else {
		g.linef("abstract class %s%s {", name, typeDecl)
		g.indent++
		for _, field := range class.Fields {
			if err := g.writeField(field); err != nil {
				return err
			}
		}
		g.line("")
		params := make([]lower.Parameter, len(class.Fields))
		for i, field := range class.Fields {
			params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
		}
		g.linef("protected %s(%s) {", name, g.params(params))
		g.indent++
		for _, field := range class.Fields {
			g.linef("this.%s = %s;", javaIdent(field.Name), javaIdent(field.Name))
		}
		g.indent--
		g.line("}")
		g.line("")
		for i, method := range class.Methods {
			if err := g.writeMethod(class, method); err != nil {
				return err
			}
			if i < len(class.Methods)-1 || len(class.Cases) > 0 {
				g.line("")
			}
		}
		for _, enumCase := range class.Cases {
			if len(enumCase.Fields) == 0 {
				fieldType := name
				if len(class.TypeParameters) == 0 {
					fieldType += typeUse
				}
				g.linef("static final %s %s = %s.INSTANCE;", fieldType, javaIdent(enumCase.Name), g.enumCaseClassName(class, enumCase.Name))
				if len(class.TypeParameters) > 0 {
					methodTypeDecl := typeDecl
					if methodTypeDecl != "" {
						methodTypeDecl += " "
					}
					g.linef("@SuppressWarnings(\"unchecked\")")
					g.linef("public static %s%s%s %s() {", methodTypeDecl, name, typeUse, javaIdent(enumCase.Name))
					g.indent++
					g.linef("return (%s%s) %s;", name, typeUse, javaIdent(enumCase.Name))
					g.indent--
					g.line("}")
				}
				continue
			}
			params := make([]lower.Parameter, len(enumCase.Fields))
			for i, field := range enumCase.Fields {
				params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
			}
			methodTypeDecl := g.typeParamsDecl(class.TypeParameters)
			if methodTypeDecl != "" {
				methodTypeDecl += " "
			}
			g.linef("public static %s%s%s %s(%s) {", methodTypeDecl, name, typeUse, javaIdent(enumCase.Name), g.params(params))
			g.indent++
			g.linef("return new %s%s(%s);", g.enumCaseClassName(class, enumCase.Name), typeUse, g.enumCasePayloadArgs(enumCase.Fields))
			g.indent--
			g.line("}")
		}
		g.indent--
		g.line("}")
	}
	for _, enumCase := range class.Cases {
		g.line("")
		if err := g.writeEnumCaseClass(class, enumCase); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) writeEnumInterfaceMethod(class *lower.Class, fn *lower.Function) error {
	returnType, err := g.javaType(fn.ReturnType, true)
	if err != nil {
		return err
	}
	header := "default " + returnType + " " + javaIdent(fn.Name) + "(" + g.params(fn.Parameters) + ")"
	g.linef("%s {", header)
	g.indent++
	prevClass := g.thisClass
	prevInObject := g.inObject
	g.thisClass = class
	g.inObject = false
	defer func() {
		g.thisClass = prevClass
		g.inObject = prevInObject
	}()
	if err := g.writeCallableBody(fn.Body, fn.ReturnType, false); err != nil {
		return err
	}
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) writeEnumCaseClass(class *lower.Class, enumCase lower.EnumCase) error {
	caseName := g.enumCaseClassName(class, enumCase.Name)
	baseName := g.className(class)
	typeDecl := g.typeParamsDecl(class.TypeParameters)
	typeUse := g.typeParamsUse(class.TypeParameters)
	if len(class.Fields) == 0 {
		g.linef("final class %s%s implements %s%s {", caseName, typeDecl, baseName, typeUse)
	} else {
		g.linef("final class %s%s extends %s%s {", caseName, typeDecl, baseName, typeUse)
	}
	g.indent++
	for _, field := range enumCase.Fields {
		if err := g.writeField(field); err != nil {
			return err
		}
	}
	if len(enumCase.Fields) > 0 {
		g.line("")
	}
	if len(enumCase.Fields) == 0 {
		instanceType := caseName
		constructorType := caseName
		if len(class.TypeParameters) == 0 {
			instanceType += typeUse
			constructorType += typeUse
		}
		g.linef("static final %s INSTANCE = new %s();", instanceType, constructorType)
		g.line("")
	}
	header := caseName + "(" + g.params(g.enumCaseParams(enumCase.Fields)) + ")"
	if len(enumCase.Fields) == 0 {
		header = "private " + header
	} else {
		header = "public " + header
	}
	g.linef("%s {", header)
	g.indent++
	if len(class.Fields) > 0 {
		g.linef("super(%s);", g.enumCaseBaseArgs(class, enumCase))
	}
	for _, field := range enumCase.Fields {
		g.linef("this.%s = %s;", javaIdent(field.Name), javaIdent(field.Name))
	}
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	return nil
}

func (g *Generator) enumCaseClassName(class *lower.Class, caseName string) string {
	return sanitizeTypeName(class.Name + "_" + caseName)
}

func (g *Generator) typeParamsDecl(params []string) string {
	if len(params) == 0 {
		return ""
	}
	names := make([]string, len(params))
	for i, param := range params {
		names[i] = sanitizeTypeName(param)
	}
	return "<" + strings.Join(names, ", ") + ">"
}

func (g *Generator) typeParamsUse(params []string) string {
	return g.typeParamsDecl(params)
}

func (g *Generator) enumCaseParams(fields []*lower.Field) []lower.Parameter {
	params := make([]lower.Parameter, len(fields))
	for i, field := range fields {
		params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
	}
	return params
}

func (g *Generator) enumCasePayloadArgs(fields []*lower.Field) string {
	args := make([]string, len(fields))
	for i, field := range fields {
		args[i] = javaIdent(field.Name)
	}
	return strings.Join(args, ", ")
}

func (g *Generator) enumCaseBaseArgs(class *lower.Class, enumCase lower.EnumCase) string {
	assignments := map[string]lower.Expr{}
	for _, assignment := range enumCase.Assignments {
		assignments[assignment.Name] = assignment.Value
	}
	args := make([]string, len(class.Fields))
	for i, field := range class.Fields {
		if value, ok := assignments[field.Name]; ok {
			rendered, err := g.exprWithExpected(value, field.Type)
			if err != nil {
				args[i] = g.zeroValueJava(field.Type)
			} else {
				args[i] = rendered
			}
			continue
		}
		args[i] = g.zeroValueJava(field.Type)
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

func (g *Generator) writeRecordDefaultConstructor(class *lower.Class) error {
	params := make([]lower.Parameter, len(class.Fields))
	for i, field := range class.Fields {
		params[i] = lower.Parameter{Name: field.Name, Type: field.Type}
	}
	g.linef("public %s(%s) {", g.className(class), g.params(params))
	g.indent++
	for _, field := range class.Fields {
		g.linef("this.%s = %s;", javaIdent(field.Name), javaIdent(field.Name))
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
	prevReturnType := g.currentReturnType
	g.currentReturnType = returnType
	defer func() {
		g.currentReturnType = prevReturnType
	}()
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
	case *lower.While:
		cond, err := g.expr(s.Condition)
		if err != nil {
			return err
		}
		g.linef("while (%s) {", unwrapGroupedJavaExpr(cond))
		g.indent++
		if err := g.writeStmtBlock(s.Body); err != nil {
			return err
		}
		g.indent--
		g.line("}")
	case *lower.Loop:
		if ok, err := g.writeWhileLoop(s); err != nil {
			return err
		} else if ok {
			return nil
		}
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
		if isUnitType(g.currentReturnType) {
			if _, ok := s.Value.(*lower.UnitLiteral); ok {
				g.line("return;")
				return nil
			}
			value, err := g.expr(s.Value)
			if err != nil {
				return err
			}
			g.linef("%s;", value)
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

func (g *Generator) writeWhileLoop(loop *lower.Loop) (bool, error) {
	if len(loop.Body) != 1 {
		return false, nil
	}
	guard, ok := loop.Body[0].(*lower.If)
	if !ok {
		return false, nil
	}
	if len(guard.Else) != 1 {
		return false, nil
	}
	if _, ok := guard.Else[0].(*lower.Break); !ok {
		return false, nil
	}
	cond, err := g.expr(guard.Condition)
	if err != nil {
		return false, err
	}
	g.linef("while (%s) {", unwrapGroupedJavaExpr(cond))
	g.indent++
	if err := g.writeStmtBlock(guard.Then); err != nil {
		return false, err
	}
	g.indent--
	g.line("}")
	return true, nil
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
	case *lower.UnitLiteral:
		return "null", nil
	case *lower.RuneLiteral:
		return strconv.FormatInt(int64(e.Value), 10) + "L", nil
	case *lower.ListLiteral:
		return g.listLiteral(e)
	case *lower.TupleLiteral:
		return g.tupleLiteral(e)
	case *lower.RecordLiteral:
		return g.recordLiteral(e)
	case *lower.TypeIs:
		value, err := g.expr(e.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s instanceof %s)", value, sanitizeTypeName(e.Target)), nil
	case *lower.Cast:
		value, err := g.expr(e.Value)
		if err != nil {
			return "", err
		}
		targetType, err := g.javaType(e.Type, false)
		if err != nil {
			targetType = sanitizeTypeName(e.Target)
		}
		return fmt.Sprintf("((%s) %s)", targetType, value), nil
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
			return g.optionCallExpr("Some", e.Args, nil)
		}
		if e.Name == "None" {
			return g.optionCallExpr("None", e.Args, nil)
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
		args := make([]string, len(e.Args))
		for i, arg := range e.Args {
			value, err := g.exprWithExpected(arg, g.constructArgType(class, i))
			if err != nil {
				return "", err
			}
			args[i] = value
		}
		return fmt.Sprintf("new %s(%s)", g.className(class), strings.Join(args, ", ")), nil
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
			return g.optionCallExpr("Some", e.Args, nil)
		}
		if callee, ok := e.Callee.(*lower.VarRef); ok && callee.Name == "None" {
			return g.optionCallExpr("None", e.Args, nil)
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

func (g *Generator) optionCallExpr(name string, args []lower.Expr, expected *typecheck.Type) (string, error) {
	switch name {
	case "Some":
		if len(args) != 1 {
			return "", fmt.Errorf("Some expects 1 argument")
		}
		value, err := g.expr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Option.some(%s)", value), nil
	case "None":
		if len(args) != 0 {
			return "", fmt.Errorf("None expects 0 arguments")
		}
		if expected != nil && expected.Name == "Option" && len(expected.Args) == 1 {
			argType, err := g.javaReferenceType(expected.Args[0])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Option.<%s>none()", argType), nil
		}
		return "Option.none()", nil
	default:
		return "", fmt.Errorf("unsupported option call %s", name)
	}
}

func (g *Generator) exprWithExpected(expr lower.Expr, expected *typecheck.Type) (string, error) {
	switch e := expr.(type) {
	case *lower.MethodCall:
		if receiver, ok := e.Receiver.(*lower.VarRef); ok && receiver.Name == "Array" && e.Method == "ofLength" {
			return g.arrayOfLengthWithExpected(e, expected)
		}
	case *lower.FunctionCall:
		if e.Name == "Some" || e.Name == "None" {
			return g.optionCallExpr(e.Name, e.Args, expected)
		}
	case *lower.Invoke:
		if callee, ok := e.Callee.(*lower.VarRef); ok && (callee.Name == "Some" || callee.Name == "None") {
			return g.optionCallExpr(callee.Name, e.Args, expected)
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

func (g *Generator) constructArgType(class *lower.Class, index int) *typecheck.Type {
	if class == nil {
		return nil
	}
	if class.Constructor != nil {
		if index >= 0 && index < len(class.Constructor.Parameters) {
			return class.Constructor.Parameters[index].Type
		}
		return nil
	}
	if class.Record {
		if index >= 0 && index < len(class.Fields) {
			return class.Fields[index].Type
		}
	}
	return nil
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

func (g *Generator) recordLiteral(record *lower.RecordLiteral) (string, error) {
	if record.Type == nil || record.Type.Kind != typecheck.TypeRecord {
		return "", fmt.Errorf("record literal requires resolved record type")
	}
	args := make([]string, len(record.Fields))
	for i, field := range record.Fields {
		value, err := g.exprWithExpected(field.Value, record.Type.Fields[i].Type)
		if err != nil {
			return "", err
		}
		args[i] = value
	}
	return fmt.Sprintf("new %s(%s)", g.recordClassName(record.Type), strings.Join(args, ", ")), nil
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
	case *lower.UnitLiteral:
		return e.Type
	case *lower.RuneLiteral:
		return e.Type
	case *lower.ListLiteral:
		return e.Type
	case *lower.TupleLiteral:
		return e.Type
	case *lower.RecordLiteral:
		return e.Type
	case *lower.TypeIs:
		return e.Type
	case *lower.Cast:
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
			if len(t.Args) == 0 {
				return g.className(class), nil
			}
			args := make([]string, len(t.Args))
			for i, arg := range t.Args {
				argType, err := g.javaReferenceType(arg)
				if err != nil {
					return "", err
				}
				args[i] = argType
			}
			return g.className(class) + "<" + strings.Join(args, ", ") + ">", nil
		}
		if len(t.Args) == 0 {
			return sanitizeTypeName(t.Name), nil
		}
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			argType, err := g.javaReferenceType(arg)
			if err != nil {
				return "", err
			}
			args[i] = argType
		}
		return sanitizeTypeName(t.Name) + "<" + strings.Join(args, ", ") + ">", nil
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
	case typecheck.TypeRecord:
		return g.recordClassName(t), nil
	case typecheck.TypeParam:
		return sanitizeTypeName(t.Name), nil
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
	return "Obj_" + sanitizeTypeName(name)
}

func (g *Generator) collectRecordTypes(program *lower.Program) {
	for _, global := range program.Globals {
		g.collectRecordType(global.Type)
		g.collectRecordExpr(global.Init)
	}
	for _, fn := range program.Functions {
		g.collectRecordFunction(fn)
	}
	for _, class := range program.Classes {
		for _, field := range class.Fields {
			g.collectRecordType(field.Type)
			g.collectRecordExpr(field.Init)
		}
		if class.Constructor != nil {
			g.collectRecordFunction(class.Constructor)
		}
		for _, method := range class.Methods {
			g.collectRecordFunction(method)
		}
	}
}

func (g *Generator) collectRecordFunction(fn *lower.Function) {
	for _, param := range fn.Parameters {
		g.collectRecordType(param.Type)
	}
	g.collectRecordType(fn.ReturnType)
	for _, stmt := range fn.Body {
		g.collectRecordStmt(stmt)
	}
}

func (g *Generator) collectRecordStmt(stmt lower.Stmt) {
	switch s := stmt.(type) {
	case *lower.VarDecl:
		g.collectRecordType(s.Type)
		g.collectRecordExpr(s.Init)
	case *lower.Assign:
		g.collectRecordExpr(s.Target)
		g.collectRecordExpr(s.Value)
	case *lower.If:
		g.collectRecordExpr(s.Condition)
		for _, stmt := range s.Then {
			g.collectRecordStmt(stmt)
		}
		for _, stmt := range s.Else {
			g.collectRecordStmt(stmt)
		}
	case *lower.ForEach:
		g.collectRecordExpr(s.Iterable)
		for _, stmt := range s.Body {
			g.collectRecordStmt(stmt)
		}
	case *lower.While:
		g.collectRecordExpr(s.Condition)
		for _, stmt := range s.Body {
			g.collectRecordStmt(stmt)
		}
	case *lower.Loop:
		for _, stmt := range s.Body {
			g.collectRecordStmt(stmt)
		}
	case *lower.Return:
		g.collectRecordExpr(s.Value)
	case *lower.ExprStmt:
		g.collectRecordExpr(s.Expr)
	}
}

func (g *Generator) collectRecordExpr(expr lower.Expr) {
	if expr == nil {
		return
	}
	g.collectRecordType(exprType(expr))
	switch e := expr.(type) {
	case *lower.ListLiteral:
		for _, elem := range e.Elements {
			g.collectRecordExpr(elem)
		}
	case *lower.TupleLiteral:
		for _, elem := range e.Elements {
			g.collectRecordExpr(elem)
		}
	case *lower.RecordLiteral:
		for _, field := range e.Fields {
			g.collectRecordExpr(field.Value)
		}
	case *lower.Unary:
		g.collectRecordExpr(e.Right)
	case *lower.Binary:
		g.collectRecordExpr(e.Left)
		g.collectRecordExpr(e.Right)
	case *lower.IfExpr:
		g.collectRecordExpr(e.Condition)
		for _, stmt := range e.ThenPrefix {
			g.collectRecordStmt(stmt)
		}
		g.collectRecordExpr(e.ThenValue)
		for _, stmt := range e.ElsePrefix {
			g.collectRecordStmt(stmt)
		}
		g.collectRecordExpr(e.ElseValue)
	case *lower.FunctionCall:
		for _, arg := range e.Args {
			g.collectRecordExpr(arg)
		}
	case *lower.Construct:
		for _, arg := range e.Args {
			g.collectRecordExpr(arg)
		}
	case *lower.FieldGet:
		g.collectRecordExpr(e.Receiver)
	case *lower.IndexGet:
		g.collectRecordExpr(e.Receiver)
		g.collectRecordExpr(e.Index)
	case *lower.MethodCall:
		g.collectRecordExpr(e.Receiver)
		for _, arg := range e.Args {
			g.collectRecordExpr(arg)
		}
	case *lower.Lambda:
		for _, param := range e.Parameters {
			g.collectRecordType(param.Type)
		}
		g.collectRecordType(e.ReturnType)
		for _, stmt := range e.Body {
			g.collectRecordStmt(stmt)
		}
	case *lower.Invoke:
		g.collectRecordExpr(e.Callee)
		for _, arg := range e.Args {
			g.collectRecordExpr(arg)
		}
	}
}

func (g *Generator) collectRecordType(t *typecheck.Type) {
	if t == nil {
		return
	}
	if t.Kind == typecheck.TypeRecord {
		key := g.recordTypeKey(t)
		if _, ok := g.records[key]; !ok {
			g.records[key] = t
		}
	}
	for _, arg := range t.Args {
		g.collectRecordType(arg)
	}
	if t.Signature != nil {
		for _, param := range t.Signature.Parameters {
			g.collectRecordType(param)
		}
		g.collectRecordType(t.Signature.ReturnType)
	}
	for _, field := range t.Fields {
		g.collectRecordType(field.Type)
	}
}

func (g *Generator) orderedRecordTypes() []*typecheck.Type {
	keys := make([]string, 0, len(g.records))
	for key := range g.records {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]*typecheck.Type, len(keys))
	for i, key := range keys {
		out[i] = g.records[key]
	}
	return out
}

func (g *Generator) recordClassName(t *typecheck.Type) string {
	return "Record_" + sanitizeTypeName(g.recordTypeKey(t))
}

func (g *Generator) recordTypeKey(t *typecheck.Type) string {
	parts := make([]string, len(t.Fields))
	for i, field := range t.Fields {
		parts[i] = field.Name + "_" + sanitizeTypeName(g.typeKey(field.Type))
	}
	return strings.Join(parts, "_")
}

func (g *Generator) typeKey(t *typecheck.Type) string {
	if t == nil {
		return "Unknown"
	}
	switch t.Kind {
	case typecheck.TypeBuiltin, typecheck.TypeClass, typecheck.TypeInterface, typecheck.TypeObject, typecheck.TypeParam:
		if len(t.Args) == 0 {
			return t.Name
		}
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			args[i] = g.typeKey(arg)
		}
		return t.Name + "_" + strings.Join(args, "_")
	case typecheck.TypeTuple:
		args := make([]string, len(t.Args))
		for i, arg := range t.Args {
			args[i] = g.typeKey(arg)
		}
		return "Tuple_" + strings.Join(args, "_")
	case typecheck.TypeRecord:
		return "Record_" + g.recordTypeKey(t)
	case typecheck.TypeFunction:
		if t.Signature == nil {
			return "Func"
		}
		args := make([]string, len(t.Signature.Parameters))
		for i, arg := range t.Signature.Parameters {
			args[i] = g.typeKey(arg)
		}
		return "Func_" + strings.Join(args, "_") + "_To_" + g.typeKey(t.Signature.ReturnType)
	default:
		return t.Name
	}
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
	if ifExpr, ok := ret.Value.(*lower.IfExpr); ok && (len(ifExpr.ThenPrefix) > 0 || len(ifExpr.ElsePrefix) > 0) {
		var body strings.Builder
		body.WriteString(param)
		body.WriteString(" -> {\n")
		if err := g.writeLambdaReturnBlock(&body, 1, ifExpr); err != nil {
			return "", err
		}
		body.WriteString("}")
		return body.String(), nil
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

func (g *Generator) writeLambdaReturnBlock(b *strings.Builder, indent int, expr lower.Expr) error {
	if ifExpr, ok := expr.(*lower.IfExpr); ok {
		cond, err := g.expr(ifExpr.Condition)
		if err != nil {
			return err
		}
		writeIndentedLine(b, indent, fmt.Sprintf("if (%s) {", unwrapGroupedJavaExpr(cond)))
		for _, stmt := range ifExpr.ThenPrefix {
			if err := g.writeStmtIntoBuilder(b, indent+1, stmt); err != nil {
				return err
			}
		}
		if err := g.writeLambdaReturnBlock(b, indent+1, ifExpr.ThenValue); err != nil {
			return err
		}
		writeIndentedLine(b, indent, "} else {")
		for _, stmt := range ifExpr.ElsePrefix {
			if err := g.writeStmtIntoBuilder(b, indent+1, stmt); err != nil {
				return err
			}
		}
		if err := g.writeLambdaReturnBlock(b, indent+1, ifExpr.ElseValue); err != nil {
			return err
		}
		writeIndentedLine(b, indent, "}")
		return nil
	}
	value, err := g.expr(expr)
	if err != nil {
		return err
	}
	writeIndentedLine(b, indent, fmt.Sprintf("return %s;", value))
	return nil
}

func (g *Generator) writeStmtIntoBuilder(b *strings.Builder, indent int, stmt lower.Stmt) error {
	switch s := stmt.(type) {
	case *lower.VarDecl:
		typ, err := g.javaType(s.Type, false)
		if err != nil {
			return err
		}
		if s.Init == nil {
			writeIndentedLine(b, indent, fmt.Sprintf("%s %s;", typ, javaIdent(s.Name)))
			return nil
		}
		initExpr, err := g.exprWithExpected(s.Init, s.Type)
		if err != nil {
			return err
		}
		writeIndentedLine(b, indent, fmt.Sprintf("%s %s = %s;", typ, javaIdent(s.Name), initExpr))
		return nil
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
		writeIndentedLine(b, indent, fmt.Sprintf("%s %s %s;", target, op, value))
		return nil
	default:
		return fmt.Errorf("unsupported lambda prefix statement %T", stmt)
	}
}

func writeIndentedLine(b *strings.Builder, indent int, line string) {
	for i := 0; i < indent; i++ {
		b.WriteString("    ")
	}
	b.WriteString(line)
	b.WriteByte('\n')
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
	indent := strings.Repeat("    ", g.indent)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			g.b.WriteByte('\n')
		}
		g.b.WriteString(indent)
		g.b.WriteString(line)
	}
	g.b.WriteByte('\n')
}

func (g *Generator) linef(format string, args ...any) {
	g.line(fmt.Sprintf(format, args...))
}
