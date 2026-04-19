package module

import (
	"fmt"
	"os"
	"path/filepath"

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
