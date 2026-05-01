package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"a-lang/parser"
)

type LoadedModule struct {
	Path          string
	SourceProgram *parser.Program
	Program       *parser.Program
	Imports       map[string]*LoadedModule
	ImportPaths   map[string]string
	SymbolImports map[string]ImportedSymbol
	Dependencies  map[string]*LoadedModule
}

type ImportedSymbol struct {
	LocalName    string
	OriginalName string
	IsInterface  bool
	Module       *LoadedModule
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
	sourceProgram, err := parser.Parse(string(src))
	if err != nil {
		return nil, err
	}
	program := sourceProgram
	stdlibDir, _ := findStdlibDir(filepath.Dir(abs))
	if stdlibDir != "" {
		preludePrograms, err := loadPreludePrograms(stdlibDir, abs)
		if err != nil {
			return nil, err
		}
		program = mergePrelude(program, preludePrograms)
	}

	mod := &LoadedModule{
		Path:          abs,
		SourceProgram: sourceProgram,
		Program:       program,
		Imports:       map[string]*LoadedModule{},
		ImportPaths:   map[string]string{},
		SymbolImports: map[string]ImportedSymbol{},
		Dependencies:  map[string]*LoadedModule{},
	}
	cache[abs] = mod

	baseDir := filepath.Dir(abs)
	for _, imp := range program.Imports {
		childPath := filepath.Join(baseDir, filepath.FromSlash(imp.Path)+".al")
		child, err := load(childPath, cache, loading)
		if err != nil {
			return nil, fmt.Errorf("load import %q: %w", imp.Path, err)
		}
		mod.Dependencies[child.Path] = child
		if len(imp.Symbols) == 0 && !imp.Wildcard {
			alias := filepath.Base(imp.Path)
			if existing, ok := mod.ImportPaths[alias]; ok && existing != imp.Path {
				return nil, fmt.Errorf("duplicate import alias '%s' for paths '%s' and '%s'", alias, existing, imp.Path)
			}
			if _, ok := mod.SymbolImports[alias]; ok {
				return nil, fmt.Errorf("module import alias '%s' conflicts with imported symbol", alias)
			}
			if child.Program.PackageName != "" && child.Program.PackageName != alias {
				return nil, fmt.Errorf("import %q expected package '%s', got '%s'", imp.Path, alias, child.Program.PackageName)
			}
			mod.Imports[alias] = child
			mod.ImportPaths[alias] = imp.Path
			continue
		}
		symbols := imp.Symbols
		if imp.Wildcard {
			symbols = exportedSymbols(child, program.PackageName)
		}
		samePackage := program.PackageName != "" && child.Program.PackageName == program.PackageName
		for _, symbol := range symbols {
			resolved, ok := resolveImportedSymbol(child, symbol.Name, samePackage)
			if !ok {
				return nil, fmt.Errorf("import %q has no public symbol '%s'", imp.Path, symbol.Name)
			}
			localName := symbol.Name
			if symbol.Alias != "" {
				localName = symbol.Alias
			}
			if _, ok := mod.Imports[localName]; ok {
				return nil, fmt.Errorf("imported symbol '%s' conflicts with module import alias", localName)
			}
			if existing, ok := mod.SymbolImports[localName]; ok && (existing.Module.Path != child.Path || existing.OriginalName != resolved.OriginalName) {
				return nil, fmt.Errorf("duplicate imported symbol '%s'", localName)
			}
			resolved.LocalName = localName
			mod.SymbolImports[localName] = resolved
		}
	}

	return mod, nil
}

func exportedSymbols(mod *LoadedModule, currentPackage string) []parser.ImportSymbol {
	samePackage := currentPackage != "" && mod.SourceProgram.PackageName == currentPackage
	out := []parser.ImportSymbol{}
	for _, decl := range mod.SourceProgram.Classes {
		if decl.Private && !samePackage {
			continue
		}
		out = append(out, parser.ImportSymbol{Name: decl.Name})
	}
	for _, decl := range mod.SourceProgram.Interfaces {
		if decl.Private && !samePackage {
			continue
		}
		out = append(out, parser.ImportSymbol{Name: decl.Name})
	}
	return out
}

func resolveImportedSymbol(mod *LoadedModule, name string, samePackage bool) (ImportedSymbol, bool) {
	for _, decl := range mod.SourceProgram.Classes {
		if decl.Name != name {
			continue
		}
		if decl.Private && !samePackage {
			return ImportedSymbol{}, false
		}
		return ImportedSymbol{OriginalName: name, Module: mod}, true
	}
	for _, decl := range mod.SourceProgram.Interfaces {
		if decl.Name != name {
			continue
		}
		if decl.Private && !samePackage {
			return ImportedSymbol{}, false
		}
		return ImportedSymbol{OriginalName: name, Module: mod, IsInterface: true}, true
	}
	return ImportedSymbol{}, false
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
	if _, err := os.Stat(filepath.Join(stdlibDir, "list.al")); err != nil {
		predefList := filepath.Join(stdlibDir, "predef", "list.al")
		if _, statErr := os.Stat(predefList); statErr == nil && predefList != currentFile {
			paths = append(paths, predefList)
		}
	}

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
