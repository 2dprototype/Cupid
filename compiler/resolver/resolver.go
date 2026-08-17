package resolver

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/modules"
	"fmt"
)

type SymbolKind int

const (
	SymVar SymbolKind = iota
	SymFunc
	SymStruct
	SymTrait
	SymConst
	SymTypeVar
)

type Symbol struct {
	Name     string
	Kind     SymbolKind
	DeclNode ast.Node // e.g. *ast.LetStmt, Param, *ast.FuncDecl, *ast.StructDecl, GenericParam
	IsMut    bool
}

type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Parent:  parent,
		Symbols: make(map[string]*Symbol),
	}
}

func (s *Scope) Insert(sym *Symbol) bool {
	if _, exists := s.Symbols[sym.Name]; exists {
		return false
	}
	s.Symbols[sym.Name] = sym
	return true
}

func (s *Scope) Lookup(name string) *Symbol {
	curr := s
	for curr != nil {
		if sym, exists := curr.Symbols[name]; exists {
			return sym
		}
		curr = curr.Parent
	}
	return nil
}

type Resolver struct {
	modules     map[string]*modules.Module
	errors      []error
	Resolutions map[ast.Node]ast.Node
}

func NewResolver(mods map[string]*modules.Module) *Resolver {
	return &Resolver{
		modules:     mods,
		errors:      []error{},
		Resolutions: make(map[ast.Node]ast.Node),
	}
}

func (r *Resolver) Errors() []error {
	return r.errors
}

func (r *Resolver) ResolveAll() bool {
	for _, mod := range r.modules {
		r.resolveModule(mod)
	}
	return len(r.errors) == 0
}

func (r *Resolver) resolveModule(mod *modules.Module) {
	// Build module scope
	modScope := NewScope(nil)

	// Add local declarations
	for _, decl := range mod.AST.Decls {
		var sym *Symbol
		var name string
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name = d.Name
			sym = &Symbol{Name: name, Kind: SymFunc, DeclNode: d}
		case *ast.StructDecl:
			name = d.Name
			sym = &Symbol{Name: name, Kind: SymStruct, DeclNode: d}
		case *ast.TraitDecl:
			name = d.Name
			sym = &Symbol{Name: name, Kind: SymTrait, DeclNode: d}
		case *ast.GlobalConstDecl:
			name = d.Name
			sym = &Symbol{Name: name, Kind: SymConst, DeclNode: d}
		}

		if sym != nil {
			if !modScope.Insert(sym) {
				r.reportError(decl.Pos(), fmt.Sprintf("duplicate declaration of %q in module %q", name, mod.Name), "E300", len(name))
			}
		}
	}

	// Walk declarations
	for _, decl := range mod.AST.Decls {
		r.resolveDecl(decl, mod, modScope)
	}
}

func (r *Resolver) resolveDecl(decl ast.Decl, mod *modules.Module, scope *Scope) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		r.resolveFuncDecl(d, mod, scope)
	case *ast.ImplDecl:
		r.resolveImplDecl(d, mod, scope)
	case *ast.StructDecl:
		r.resolveStructDecl(d, mod, scope)
	case *ast.TraitDecl:
		// Trait method signatures only have types, but they might have parameters
		for _, m := range d.Methods {
			funcScope := NewScope(scope)
			// Add generics/lifetimes if any?
			for _, p := range m.Params {
				r.resolveType(p.Type, mod, funcScope)
			}
			if m.ReturnType != nil {
				r.resolveType(m.ReturnType, mod, funcScope)
			}
		}
	case *ast.GlobalConstDecl:
		if d.Type != nil {
			r.resolveType(d.Type, mod, scope)
		}
		r.resolveExpr(d.Value, mod, scope)
	}
}

func (r *Resolver) resolveFuncDecl(fd *ast.FuncDecl, mod *modules.Module, parentScope *Scope) {
	funcScope := NewScope(parentScope)

	// Add generic parameters to scope as type variables
	for _, gp := range fd.Generics {
		funcScope.Insert(&Symbol{
			Name:     gp.Name,
			Kind:     SymTypeVar,
			DeclNode: fd, // dummy reference back to function
		})
	}

	// Resolve parameter types and add params to scope
	for _, p := range fd.Params {
		r.resolveType(p.Type, mod, funcScope)
		sym := &Symbol{
			Name:     p.Name,
			Kind:     SymVar,
			DeclNode: p.Type, // Type is the definition
		}
		if p.Name == "self" {
			// self is special, check if it's mutable
			if _, ok := p.Type.(*ast.PointerType); ok {
				sym.IsMut = true
			}
		}
		funcScope.Insert(sym)
		// Map the parameter usage to itself
		r.Resolutions[p.Type] = p.Type
	}

	if fd.ReturnType != nil {
		r.resolveType(fd.ReturnType, mod, funcScope)
	}

	if fd.Body != nil {
		r.resolveBlockStmt(fd.Body, mod, funcScope)
	}
}

func (r *Resolver) resolveImplDecl(id *ast.ImplDecl, mod *modules.Module, parentScope *Scope) {
	implScope := NewScope(parentScope)

	// Add generic parameters
	for _, gp := range id.Generics {
		implScope.Insert(&Symbol{
			Name:     gp.Name,
			Kind:     SymTypeVar,
			DeclNode: id,
		})
	}

	r.resolveType(id.Target, mod, implScope)

	// Add "self" and "Self" pointing to id.Target
	implScope.Insert(&Symbol{
		Name:     "self",
		Kind:     SymTypeVar,
		DeclNode: id.Target,
	})
	implScope.Insert(&Symbol{
		Name:     "Self",
		Kind:     SymTypeVar,
		DeclNode: id.Target,
	})

	for _, m := range id.Methods {
		// Methods inside impl can reference target self and its lifetimes
		r.resolveFuncDecl(m, mod, implScope)
	}
}

func (r *Resolver) resolveStructDecl(sd *ast.StructDecl, mod *modules.Module, parentScope *Scope) {
	structScope := NewScope(parentScope)
	for _, gp := range sd.Generics {
		structScope.Insert(&Symbol{
			Name:     gp.Name,
			Kind:     SymTypeVar,
			DeclNode: sd,
		})
	}

	for _, f := range sd.Fields {
		r.resolveType(f.Type, mod, structScope)
	}
}

func (r *Resolver) resolveBlockStmt(bs *ast.BlockStmt, mod *modules.Module, parentScope *Scope) {
	blockScope := NewScope(parentScope)
	for _, stmt := range bs.Stmts {
		r.resolveStmt(stmt, mod, blockScope)
	}
}

func (r *Resolver) resolveStmt(stmt ast.Stmt, mod *modules.Module, scope *Scope) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		if s.Type != nil {
			r.resolveType(s.Type, mod, scope)
		}
		r.resolveExpr(s.Value, mod, scope)
		// Register variable in current scope
		sym := &Symbol{
			Name:     s.Name,
			Kind:     SymVar,
			DeclNode: s,
			IsMut:    s.Mutable,
		}
		if !scope.Insert(sym) {
			r.reportError(s.Pos(), fmt.Sprintf("redefinition of variable %q in block scope", s.Name), "E302", len(s.Name))
		}
	case *ast.ReturnStmt:
		if s.Value != nil {
			r.resolveExpr(s.Value, mod, scope)
		}
	case *ast.ExprStmt:
		r.resolveExpr(s.Expression, mod, scope)
	case *ast.IfStmt:
		r.resolveExpr(s.Condition, mod, scope)
		r.resolveBlockStmt(s.ThenBlock, mod, scope)
		if s.ElseBlock != nil {
			r.resolveStmt(s.ElseBlock, mod, scope)
		}
	case *ast.ForStmt:
		loopScope := NewScope(scope)
		if s.VarName != "" {
			loopScope.Insert(&Symbol{
				Name:     s.VarName,
				Kind:     SymVar,
				DeclNode: s,
			})
		}
		if s.RangeExpr != nil {
			r.resolveExpr(s.RangeExpr, mod, loopScope)
		}
		r.resolveBlockStmt(s.Body, mod, loopScope)
	case *ast.UnsafeBlock:
		r.resolveBlockStmt(s.Block, mod, scope)
	case *ast.AsmBlock:
		// asm block has no variables/types to resolve
	}
}

func (r *Resolver) resolveExpr(expr ast.Expr, mod *modules.Module, scope *Scope) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.IdentExpr:
		r.resolveIdentExpr(e, mod, scope)
	case *ast.LiteralExpr:
		// Nothing to resolve for literals
	case *ast.BinaryExpr:
		r.resolveExpr(e.Left, mod, scope)
		r.resolveExpr(e.Right, mod, scope)
	case *ast.UnaryExpr:
		r.resolveExpr(e.Right, mod, scope)
	case *ast.SelectorExpr:
		// e.g. target.field or module.func
		// If target is an identifier, check if it refers to an imported module
		if ident, ok := e.Target.(*ast.IdentExpr); ok {
			if depPath, exists := mod.ImportedModules[ident.Name]; exists {
				// It's a module reference! E.g. math.sin
				depMod := r.modules[depPath]
				if depMod != nil {
					if decl, ok := depMod.Exports[e.Field]; ok {
						r.Resolutions[e] = decl
						r.Resolutions[ident] = decl // resolve prefix too
						return
					} else {
						r.reportError(e.Pos(), fmt.Sprintf("symbol %q is not exported by module %q", e.Field, depMod.Name), "E601", len(e.Field))
					}
				}
			}
		}
		// Otherwise, it's a field or method selector, type checker will resolve it.
		r.resolveExpr(e.Target, mod, scope)
	case *ast.CallExpr:
		r.resolveExpr(e.Function, mod, scope)
		for _, arg := range e.Args {
			r.resolveExpr(arg, mod, scope)
		}
	case *ast.IndexExpr:
		r.resolveExpr(e.Target, mod, scope)
		r.resolveExpr(e.Index, mod, scope)
	case *ast.StructInitExpr:
		r.resolveType(e.Struct, mod, scope)
		for _, f := range e.Fields {
			r.resolveExpr(f.Value, mod, scope)
		}
	case *ast.MatchExpr:
		r.resolveExpr(e.Target, mod, scope)
		for _, c := range e.Cases {
			caseScope := NewScope(scope)
			// Patterns might introduce variables, but currently Cupid patterns are just constants
			// If patterns have variables in future, we would bind them here.
			r.resolveExpr(c.Pattern, mod, caseScope)
			r.resolveBlockStmt(c.Body, mod, caseScope)
		}
	case *ast.QuestionExpr:
		r.resolveExpr(e.Target, mod, scope)
	}
}

func (r *Resolver) resolveIdentExpr(ie *ast.IdentExpr, mod *modules.Module, scope *Scope) {
	// 0. Built-in functions and literals
	switch ie.Name {
	case "print", "println", "len", "sizeof", "alignof", "true", "false":
		return
	}

	// 1. Look up in local lexical scopes
	if sym := scope.Lookup(ie.Name); sym != nil {
		r.Resolutions[ie] = sym.DeclNode
		return
	}

	// 2. Look up in module-level imported symbols
	if impSym, exists := mod.ImportedSymbols[ie.Name]; exists {
		targetMod := r.modules[impSym.ModulePath]
		if targetMod != nil {
			if decl, exists := targetMod.Exports[impSym.SymbolName]; exists {
				r.Resolutions[ie] = decl
				return
			}
		}
	}

	// 3. Look up in module-level declarations
	for _, decl := range mod.AST.Decls {
		var dName string
		switch d := decl.(type) {
		case *ast.FuncDecl:
			dName = d.Name
		case *ast.StructDecl:
			dName = d.Name
		case *ast.TraitDecl:
			dName = d.Name
		case *ast.GlobalConstDecl:
			dName = d.Name
		}
		if dName == ie.Name {
			r.Resolutions[ie] = decl
			return
		}
	}

	// If it's a module name (e.g. "math"), it is resolved inside SelectorExpr, but if used alone, it's an error
	if _, isModule := mod.ImportedModules[ie.Name]; isModule {
		r.reportError(ie.Pos(), fmt.Sprintf("module %q used as an expression", ie.Name), "E303", len(ie.Name))
		return
	}

	// Unresolved name
	r.reportError(ie.Pos(), fmt.Sprintf("unresolved name %q", ie.Name), "E301", len(ie.Name))
}

func (r *Resolver) resolveType(t ast.Type, mod *modules.Module, scope *Scope) {
	if t == nil {
		return
	}

	switch pt := t.(type) {
	case *ast.PrimitiveType:
		// If it's a built-in type (i32, f64, bool, string, etc.), do nothing
		switch pt.Name {
		case "i32", "i64", "u32", "u64", "f32", "f64", "bool", "string", "char", "void", "self", "Self":
			return
		}

		// 1. Check local generic type variables/lifetimes
		if sym := scope.Lookup(pt.Name); sym != nil && sym.Kind == SymTypeVar {
			r.Resolutions[pt] = sym.DeclNode
			return
		}

		// 2. Look up in module-level imported symbols
		if impSym, exists := mod.ImportedSymbols[pt.Name]; exists {
			targetMod := r.modules[impSym.ModulePath]
			if targetMod != nil {
				if decl, exists := targetMod.Exports[impSym.SymbolName]; exists {
					r.Resolutions[pt] = decl
					return
				}
			}
		}

		// 3. Look up custom struct/trait type in module declarations
		for _, decl := range mod.AST.Decls {
			switch d := decl.(type) {
			case *ast.StructDecl:
				if d.Name == pt.Name {
					r.Resolutions[pt] = d
					return
				}
			case *ast.TraitDecl:
				if d.Name == pt.Name {
					r.Resolutions[pt] = d
					return
				}
			}
		}

		r.reportError(pt.Pos(), fmt.Sprintf("unresolved type %q", pt.Name), "E304", len(pt.Name))

	case *ast.PointerType:
		r.resolveType(pt.To, mod, scope)
	case *ast.ArrayType:
		r.resolveType(pt.Element, mod, scope)
	}
}

func (r *Resolver) reportError(pos ast.Position, msg string, code string, spanLen int) {
	diag := diagnostics.Diagnostic{
		Code:    code,
		Message: msg,
		File:    pos.File,
		Line:    pos.Line,
		Column:  pos.Col,
		SpanLen: spanLen,
	}
	r.errors = append(r.errors, diag)
}
