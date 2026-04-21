package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"a-lang/parser"
)

type LoadedModule struct {
	Path        string
	Program     *parser.Program
	Imports     map[string]*LoadedModule
	ImportPaths map[string]string
}

func Load(path string) (*LoadedModule, error) {
	cache := map[string]*LoadedModule{}
	loading := map[string]bool{}
	return load(path, cache, loading)
}

func load(path string, cache map[string]*LoadedModule, loading map[string]bool) (*LoadedModule, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if mod, ok := cache[abs]; ok {
		return mod, nil
	}
	if loading[abs] {
		return nil, fmt.Errorf("import cycle detected at %s", abs)
	}
	loading[abs] = true
	defer delete(loading, abs)

	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	program, err := parser.Parse(string(src))
	if err != nil {
		return nil, err
	}
	stdlibDir, _ := findStdlibDir(filepath.Dir(abs))
	if stdlibDir != "" {
		preludePrograms, err := loadPreludePrograms(stdlibDir, abs)
		if err != nil {
			return nil, err
		}
		program = mergePrelude(program, preludePrograms)
	}

	mod := &LoadedModule{
		Path:        abs,
		Program:     program,
		Imports:     map[string]*LoadedModule{},
		ImportPaths: map[string]string{},
	}
	cache[abs] = mod

	baseDir := filepath.Dir(abs)
	for _, imp := range program.Imports {
		alias := filepath.Base(imp.Path)
		if existing, ok := mod.ImportPaths[alias]; ok && existing != imp.Path {
			return nil, fmt.Errorf("duplicate import alias '%s' for paths '%s' and '%s'", alias, existing, imp.Path)
		}
		childPath := filepath.Join(baseDir, filepath.FromSlash(imp.Path)+".al")
		child, err := load(childPath, cache, loading)
		if err != nil {
			return nil, fmt.Errorf("load import %q: %w", imp.Path, err)
		}
		if child.Program.PackageName != "" && child.Program.PackageName != alias {
			return nil, fmt.Errorf("import %q expected package '%s', got '%s'", imp.Path, alias, child.Program.PackageName)
		}
		mod.Imports[alias] = child
		mod.ImportPaths[alias] = imp.Path
	}

	return mod, nil
}

func findStdlibDir(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "stdlib")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func loadPreludePrograms(stdlibDir, currentFile string) ([]*parser.Program, error) {
	entries, err := os.ReadDir(stdlibDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".al" {
			continue
		}
		path := filepath.Join(stdlibDir, entry.Name())
		if path == currentFile {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	out := make([]*parser.Program, 0, len(paths))
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		program, err := parser.Parse(string(src))
		if err != nil {
			return nil, fmt.Errorf("parse stdlib %q: %w", filepath.Base(path), err)
		}
		out = append(out, program)
	}
	return out, nil
}

func mergePrelude(program *parser.Program, prelude []*parser.Program) *parser.Program {
	if len(prelude) == 0 {
		return program
	}
	merged := &parser.Program{
		PackageName: program.PackageName,
		PackageSpan: program.PackageSpan,
		Imports:     append([]parser.ImportDecl(nil), program.Imports...),
		Functions:   []*parser.FunctionDecl{},
		Interfaces:  []*parser.InterfaceDecl{},
		Classes:     []*parser.ClassDecl{},
		Statements:  append([]parser.Statement(nil), program.Statements...),
		Span:        program.Span,
	}
	for _, std := range prelude {
		merged.Functions = append(merged.Functions, std.Functions...)
		merged.Interfaces = append(merged.Interfaces, std.Interfaces...)
		merged.Classes = append(merged.Classes, std.Classes...)
	}
	merged.Functions = append(merged.Functions, program.Functions...)
	merged.Interfaces = append(merged.Interfaces, program.Interfaces...)
	merged.Classes = append(merged.Classes, program.Classes...)
	return merged
}
