package modules

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
	"cupid/compiler/parser"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ImportedSymbol struct {
	ModulePath string
	SymbolName string
}

type Module struct {
	Path            string
	Name            string
	AST             *ast.Program
	Source          string
	Exports         map[string]ast.Decl
	ImportedSymbols map[string]ImportedSymbol
	ImportedModules map[string]string // alias/baseName -> canonical path
}

type ModuleResolver struct {
	stdlibDir string
	modules   map[string]*Module
	order     []string
	errors    []error
}

func NewModuleResolver(stdlibDir string) *ModuleResolver {
	return &ModuleResolver{
		stdlibDir: stdlibDir,
		modules:   make(map[string]*Module),
		order:     []string{},
		errors:    []error{},
	}
}

func (mr *ModuleResolver) Modules() map[string]*Module {
	return mr.modules
}

func (mr *ModuleResolver) Order() []string {
	return mr.order
}

func (mr *ModuleResolver) Errors() []error {
	return mr.errors
}

// ResolveGraph resolves all modules starting from a root file
func (mr *ModuleResolver) ResolveGraph(rootPath string) (*Module, error) {
	canonicalRoot, err := mr.canonicalizePath(rootPath)
	if err != nil {
		return nil, err
	}

	visiting := make(map[string]*ast.ImportDecl)
	visited := make(map[string]bool)

	rootMod, err := mr.resolveModuleRec(canonicalRoot, visiting, visited, nil)
	if err != nil {
		return nil, err
	}

	// Now validate that all imported symbols in all modules are exported in their target modules
	mr.validateVisibility()

	if len(mr.errors) > 0 {
		return nil, fmt.Errorf("module resolution failed with %d errors", len(mr.errors))
	}

	return rootMod, nil
}

func (mr *ModuleResolver) canonicalizePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	cleaned := filepath.Clean(abs)
	return strings.ToLower(filepath.ToSlash(cleaned)), nil
}

// FindModule locates a module on the filesystem
func (mr *ModuleResolver) findModulePath(importPath string, currentFileDir string) (string, error) {
	// Try standard library first if it's a bare module name
	isLocal := strings.HasPrefix(importPath, ".") || strings.HasPrefix(importPath, "..") || filepath.IsAbs(importPath)

	if !isLocal && mr.stdlibDir != "" {
		// Look in stdlib
		candidates := []string{
			filepath.Join(mr.stdlibDir, importPath+".cu"),
			filepath.Join(mr.stdlibDir, importPath, importPath+".cu"),
			filepath.Join(mr.stdlibDir, importPath, "main.cu"),
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				return mr.canonicalizePath(cand)
			}
		}
	}

	// Try local directory or relative directory
	baseDir := currentFileDir
	if baseDir == "" {
		baseDir = "."
	}
	candidates := []string{
		filepath.Join(baseDir, importPath),
		filepath.Join(baseDir, importPath+".cu"),
		filepath.Join(baseDir, importPath, filepath.Base(importPath)+".cu"),
		filepath.Join(baseDir, importPath, "main.cu"),
	}

	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return mr.canonicalizePath(cand)
		}
	}

	return "", fmt.Errorf("module not found")
}

func (mr *ModuleResolver) resolveModuleRec(
	canonicalPath string,
	visiting map[string]*ast.ImportDecl,
	visited map[string]bool,
	importStmt *ast.ImportDecl,
) (*Module, error) {
	// Circular dependency check
	if visiting[canonicalPath] != nil {
		firstImport := visiting[canonicalPath]
		diag := diagnostics.Diagnostic{
			Code:    "E602",
			Message: fmt.Sprintf("circular import between %q and %q", filepath.Base(firstImport.Position.File), filepath.Base(canonicalPath)),
			File:    importStmt.Position.File,
			Line:    importStmt.Position.Line,
			Column:  importStmt.Position.Col,
			SpanLen: len(importStmt.String()),
		}
		mr.errors = append(mr.errors, diag)
		return nil, diag
	}

	// If already fully visited, return cached
	if visited[canonicalPath] {
		return mr.modules[canonicalPath], nil
	}

	// Read and parse file
	bytes, err := os.ReadFile(canonicalPath)
	if err != nil {
		var diag diagnostics.Diagnostic
		if importStmt != nil {
			diag = diagnostics.Diagnostic{
				Code:    "E603",
				Message: fmt.Sprintf("module file not found: %s", canonicalPath),
				File:    importStmt.Position.File,
				Line:    importStmt.Position.Line,
				Column:  importStmt.Position.Col,
				SpanLen: len(importStmt.String()),
			}
		} else {
			diag = diagnostics.Diagnostic{
				Code:    "E603",
				Message: fmt.Sprintf("failed to read file: %s", canonicalPath),
				File:    canonicalPath,
				Line:    1,
				Column:  1,
			}
		}
		mr.errors = append(mr.errors, diag)
		return nil, diag
	}

	source := string(bytes)
	l := lexer.New(source, canonicalPath)
	p := parser.New(l, source)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		mr.errors = append(mr.errors, p.Errors()...)
		return nil, fmt.Errorf("parse errors in %s", canonicalPath)
	}

	// Create module
	baseName := strings.TrimSuffix(filepath.Base(canonicalPath), ".cu")
	mod := &Module{
		Path:            canonicalPath,
		Name:            baseName,
		AST:             prog,
		Source:          source,
		Exports:         make(map[string]ast.Decl),
		ImportedSymbols: make(map[string]ImportedSymbol),
		ImportedModules: make(map[string]string),
	}

	// Register exports
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Exported {
				mod.Exports[d.Name] = d
			}
		case *ast.StructDecl:
			if d.Exported {
				mod.Exports[d.Name] = d
			}
		case *ast.TraitDecl:
			if d.Exported {
				mod.Exports[d.Name] = d
			}
		case *ast.GlobalConstDecl:
			if d.Exported {
				mod.Exports[d.Name] = d
			}
		}
	}

	// Visit imports
	mr.modules[canonicalPath] = mod
	visiting[canonicalPath] = importStmt

	currentDir := filepath.Dir(canonicalPath)
	for _, decl := range prog.Decls {
		imp, ok := decl.(*ast.ImportDecl)
		if !ok {
			continue
		}

		importPath := imp.Path
		if importPath == "" {
			importPath = imp.FromModule
		}

		depPath, err := mr.findModulePath(importPath, currentDir)
		if err != nil {
			diag := diagnostics.Diagnostic{
				Code:    "E603",
				Message: fmt.Sprintf("module %q not found", importPath),
				File:    imp.Position.File,
				Line:    imp.Position.Line,
				Column:  imp.Position.Col,
				SpanLen: len(imp.String()),
			}
			mr.errors = append(mr.errors, diag)
			continue
		}

		_, err = mr.resolveModuleRec(depPath, visiting, visited, imp)
		if err != nil {
			// error already added to mr.errors
			continue
		}

		// Save imported module path mapping
		if len(imp.Symbols) > 0 {
			for _, sym := range imp.Symbols {
				localName := sym.Alias
				if localName == "" {
					localName = sym.Name
				}
				mod.ImportedSymbols[localName] = ImportedSymbol{
					ModulePath: depPath,
					SymbolName: sym.Name,
				}
			}
		} else {
			aliasOrName := imp.Alias
			if aliasOrName == "" {
				aliasOrName = strings.TrimSuffix(filepath.Base(depPath), ".cu")
			}
			mod.ImportedModules[aliasOrName] = depPath
		}
	}

	delete(visiting, canonicalPath)
	visited[canonicalPath] = true
	mr.order = append(mr.order, canonicalPath)

	return mod, nil
}

func (mr *ModuleResolver) validateVisibility() {
	for _, mod := range mr.modules {
		// Validate named imported symbols
		for localName, impSym := range mod.ImportedSymbols {
			targetMod := mr.modules[impSym.ModulePath]
			if targetMod == nil {
				continue
			}

			decl, ok := targetMod.Exports[impSym.SymbolName]
			if !ok {
				// The symbol is either private or not found in targetMod
				// We need to locate the ImportDecl that imported this symbol to report the diagnostic correctly
				var targetImport *ast.ImportDecl
				for _, decl := range mod.AST.Decls {
					if imp, ok := decl.(*ast.ImportDecl); ok {
						for _, s := range imp.Symbols {
							name := s.Alias
							if name == "" {
								name = s.Name
							}
							if name == localName {
								targetImport = imp
								break
							}
						}
					}
					if targetImport != nil {
						break
					}
				}

				pos := ast.Position{File: mod.Path, Line: 1, Col: 1}
				spanLen := 1
				if targetImport != nil {
					pos = targetImport.Position
					spanLen = len(targetImport.String())
				}

				diag := diagnostics.Diagnostic{
					Code:    "E601",
					Message: fmt.Sprintf("'%s' is private to module '%s'", impSym.SymbolName, targetMod.Name),
					File:    pos.File,
					Line:    pos.Line,
					Column:  pos.Col,
					SpanLen: spanLen,
				}
				mr.errors = append(mr.errors, diag)
			} else {
				_ = decl
			}
		}
	}
}
