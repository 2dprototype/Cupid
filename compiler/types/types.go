package types

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
	"cupid/compiler/modules"
	"fmt"
	"strings"
)

type TypeChecker struct {
	modules     map[string]*modules.Module
	resolutions map[ast.Node]ast.Node
	errors      []error
	specialized map[string]ast.Decl
	ExprTypes   map[ast.Expr]ast.Type
}

func NewTypeChecker(mods map[string]*modules.Module, res map[ast.Node]ast.Node) *TypeChecker {
	return &TypeChecker{
		modules:     mods,
		resolutions: res,
		errors:      []error{},
		specialized: make(map[string]ast.Decl),
		ExprTypes:   make(map[ast.Expr]ast.Type),
	}
}

func (tc *TypeChecker) Errors() []error {
	return tc.errors
}

func isIntType(name string) bool {
	switch name {
	case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "int", "uint", "usize", "isize":
		return true
	}
	return false
}

func (tc *TypeChecker) TypesEqual(t1, t2 ast.Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	switch pt1 := t1.(type) {
	case *ast.PrimitiveType:
		pt2, ok := t2.(*ast.PrimitiveType)
		if !ok {
			return false
		}
		if pt1.Name == pt2.Name {
			return true
		}
		if isIntType(pt1.Name) && isIntType(pt2.Name) {
			return true
		}
		return false
	case *ast.PointerType:
		pt2, ok := t2.(*ast.PointerType)
		return ok && pt1.Mutable == pt2.Mutable && tc.TypesEqual(pt1.To, pt2.To)
	case *ast.ArrayType:
		pt2, ok := t2.(*ast.ArrayType)
		return ok && pt1.Size == pt2.Size && tc.TypesEqual(pt1.Element, pt2.Element)
	case *ast.DynTraitType:
		pt2, ok := t2.(*ast.DynTraitType)
		return ok && pt1.Trait == pt2.Trait
	case *ast.GenericType:
		pt2, ok := t2.(*ast.GenericType)
		if !ok || pt1.BaseName != pt2.BaseName || len(pt1.Params) != len(pt2.Params) {
			return false
		}
		for i := range pt1.Params {
			if !tc.TypesEqual(pt1.Params[i], pt2.Params[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (tc *TypeChecker) TypeCheckAll() bool {
	for _, mod := range tc.modules {
		// Type-check all declarations in the module
		for _, decl := range mod.AST.Decls {
			tc.typeCheckDecl(decl, mod)
		}
	}
	return len(tc.errors) == 0
}

func (tc *TypeChecker) typeCheckDecl(decl ast.Decl, mod *modules.Module) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		// Skip generic functions since they are monomorphized on call
		if len(d.Generics) > 0 {
			return
		}
		tc.typeCheckFuncDecl(d, mod)
	case *ast.ImplDecl:
		if len(d.Generics) > 0 {
			return
		}
		for _, m := range d.Methods {
			tc.typeCheckFuncDecl(m, mod)
		}
	case *ast.StructDecl:
		if len(d.Generics) > 0 {
			return
		}
		for i, f := range d.Fields {
			d.Fields[i].Type = tc.resolveAndValidateType(f.Type, mod)
		}
	case *ast.GlobalConstDecl:
		var expectedType ast.Type
		if d.Type != nil {
			d.Type = tc.resolveAndValidateType(d.Type, mod)
			expectedType = d.Type
		}
		valType := tc.TypeCheckExpr(d.Value, mod)
		if expectedType != nil && valType != nil {
			if !tc.TypesEqual(expectedType, valType) {
				tc.reportError(d.Pos(), fmt.Sprintf("type mismatch in global constant %q: expected %q, got %q", d.Name, expectedType.String(), valType.String()), "E401", len(d.Name))
			}
		}
	}
}

func (tc *TypeChecker) typeCheckFuncDecl(fd *ast.FuncDecl, mod *modules.Module) {
	// Type check parameters
	for i, p := range fd.Params {
		fd.Params[i].Type = tc.resolveAndValidateType(p.Type, mod)
	}

	var expectedRet ast.Type = &ast.PrimitiveType{Position: fd.Position, Name: "void"}
	if fd.ReturnType != nil {
		fd.ReturnType = tc.resolveAndValidateType(fd.ReturnType, mod)
		expectedRet = fd.ReturnType
	}

	if fd.Body != nil {
		tc.typeCheckBlockStmt(fd.Body, expectedRet, mod)
	}
}

func (tc *TypeChecker) typeCheckBlockStmt(bs *ast.BlockStmt, expectedRet ast.Type, mod *modules.Module) {
	for _, stmt := range bs.Stmts {
		tc.typeCheckStmt(stmt, expectedRet, mod)
	}
}

func (tc *TypeChecker) typeCheckStmt(stmt ast.Stmt, expectedRet ast.Type, mod *modules.Module) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		var expectedType ast.Type
		if s.Type != nil {
			s.Type = tc.resolveAndValidateType(s.Type, mod)
			expectedType = s.Type
		}
		valType := tc.TypeCheckExpr(s.Value, mod)
		if expectedType != nil && valType != nil {
			if !tc.TypesEqual(expectedType, valType) {
				tc.reportError(s.Pos(), fmt.Sprintf("type mismatch in variable binding %q: expected %q, got %q", s.Name, expectedType.String(), valType.String()), "E401", len(s.Name))
			}
		}
	case *ast.ReturnStmt:
		var actualRet ast.Type = &ast.PrimitiveType{Position: s.Pos(), Name: "void"}
		if s.Value != nil {
			actualRet = tc.TypeCheckExpr(s.Value, mod)
		}
		if actualRet != nil && !tc.TypesEqual(expectedRet, actualRet) {
			tc.reportError(s.Pos(), fmt.Sprintf("type mismatch in return statement: expected %q, got %q", expectedRet.String(), actualRet.String()), "E401", 6)
		}
	case *ast.ExprStmt:
		tc.TypeCheckExpr(s.Expression, mod)
	case *ast.IfStmt:
		condType := tc.TypeCheckExpr(s.Condition, mod)
		if condType != nil && condType.String() != "bool" {
			tc.reportError(s.Condition.Pos(), fmt.Sprintf("if condition must be bool, got %q", condType.String()), "E402", len(s.Condition.String()))
		}
		tc.typeCheckBlockStmt(s.ThenBlock, expectedRet, mod)
		if s.ElseBlock != nil {
			tc.typeCheckStmt(s.ElseBlock, expectedRet, mod)
		}
	case *ast.ForStmt:
		if s.RangeExpr != nil {
			rangeType := tc.TypeCheckExpr(s.RangeExpr, mod)
			_ = rangeType // range validation (e.g. arrays or iterators)
		}
		tc.typeCheckBlockStmt(s.Body, expectedRet, mod)
	case *ast.UnsafeBlock:
		tc.typeCheckBlockStmt(s.Block, expectedRet, mod)
	case *ast.AsmBlock:
		// inline assembly is untyped
	}
}

func (tc *TypeChecker) TypeCheckExpr(expr ast.Expr, mod *modules.Module) ast.Type {
	if expr == nil {
		return nil
	}

	if cached, exists := tc.ExprTypes[expr]; exists {
		return cached
	}

	var t ast.Type
	switch e := expr.(type) {
	case *ast.IdentExpr:
		t = tc.typeCheckIdentExpr(e, mod)
	case *ast.LiteralExpr:
		t = tc.typeCheckLiteralExpr(e)
	case *ast.BinaryExpr:
		t = tc.typeCheckBinaryExpr(e, mod)
	case *ast.UnaryExpr:
		t = tc.typeCheckUnaryExpr(e, mod)
	case *ast.RefExpr:
		targetType := tc.TypeCheckExpr(e.Target, mod)
		if targetType != nil {
			t = &ast.PointerType{Position: e.Position, To: targetType, Mutable: e.Mutable}
		}
	case *ast.DerefExpr:
		targetType := tc.TypeCheckExpr(e.Target, mod)
		if targetType != nil {
			if ptr, ok := targetType.(*ast.PointerType); ok {
				t = ptr.To
			} else {
				tc.reportError(e.Pos(), fmt.Sprintf("cannot dereference non-pointer type %q", targetType.String()), "E403", 1)
			}
		}
	case *ast.SelectorExpr:
		t = tc.typeCheckSelectorExpr(e, mod)
	case *ast.CallExpr:
		t = tc.typeCheckCallExpr(e, mod)
	case *ast.IndexExpr:
		targetType := tc.TypeCheckExpr(e.Target, mod)
		indexType := tc.TypeCheckExpr(e.Index, mod)
		if indexType != nil && indexType.String() != "i32" && indexType.String() != "i64" && indexType.String() != "usize" && indexType.String() != "int" {
			tc.reportError(e.Index.Pos(), fmt.Sprintf("array index must be integer, got %q", indexType.String()), "E404", len(e.Index.String()))
		}
		if targetType != nil {
			if arr, ok := targetType.(*ast.ArrayType); ok {
				t = arr.Element
			} else {
				tc.reportError(e.Pos(), fmt.Sprintf("cannot index non-array type %q", targetType.String()), "E405", len(e.Target.String()))
			}
		}
	case *ast.ArrayLiteralExpr:
		var elemType ast.Type
		for _, elem := range e.Elements {
			et := tc.TypeCheckExpr(elem, mod)
			if elemType == nil {
				elemType = et
			} else if et != nil && !tc.TypesEqual(elemType, et) {
				tc.reportError(elem.Pos(), fmt.Sprintf("array element type mismatch: expected %q, got %q", elemType.String(), et.String()), "E401", len(elem.String()))
			}
		}
		if elemType == nil {
			elemType = &ast.PrimitiveType{Position: e.Position, Name: "i64"}
		}
		t = &ast.ArrayType{Position: e.Position, Element: elemType, Size: len(e.Elements)}
	case *ast.StructInitExpr:
		t = tc.typeCheckStructInitExpr(e, mod)
	case *ast.MatchExpr:
		targetType := tc.TypeCheckExpr(e.Target, mod)
		for _, c := range e.Cases {
			if ident, ok := c.Pattern.(*ast.IdentExpr); ok && ident.Name == "_" {
				// Wildcard matches all target types
			} else {
				patType := tc.TypeCheckExpr(c.Pattern, mod)
				if targetType != nil && patType != nil && !tc.TypesEqual(targetType, patType) {
					tc.reportError(c.Pattern.Pos(), fmt.Sprintf("match pattern type mismatch: expected %q, got %q", targetType.String(), patType.String()), "E401", len(c.Pattern.String()))
				}
			}
			tc.typeCheckBlockStmt(c.Body, &ast.PrimitiveType{Name: "void"}, mod)
		}
		t = &ast.PrimitiveType{Position: e.Position, Name: "void"}
	case *ast.QuestionExpr:
		targetType := tc.TypeCheckExpr(e.Target, mod)
		t = targetType // simplify for now
	}

	if t != nil {
		tc.ExprTypes[expr] = t
	}
	return t
}

func (tc *TypeChecker) typeCheckIdentExpr(ie *ast.IdentExpr, mod *modules.Module) ast.Type {
	if ie.Name == "true" || ie.Name == "false" {
		return &ast.PrimitiveType{Position: ie.Position, Name: "bool"}
	}
	if ie.Name == "_" {
		return &ast.PrimitiveType{Position: ie.Position, Name: "void"}
	}
	if ie.Name == "print" || ie.Name == "println" || ie.Name == "len" || ie.Name == "sizeof" || ie.Name == "alignof" {
		return &ast.PrimitiveType{Position: ie.Position, Name: "fn"}
	}

	decl := tc.resolutions[ie]
	if decl == nil {
		// Might be a built-in type or unresolved
		return nil
	}

	switch d := decl.(type) {
	case *ast.LetStmt:
		if d.Type != nil {
			return d.Type
		}
		// Infer type from value
		return tc.TypeCheckExpr(d.Value, mod)
	case ast.Type:
		// Resolved to a type parameter or type definition
		return d
	case *ast.GlobalConstDecl:
		if d.Type != nil {
			return d.Type
		}
		return tc.TypeCheckExpr(d.Value, mod)
	case *ast.FuncDecl:
		return &ast.PrimitiveType{Position: d.Position, Name: "fn"}
	}
	return nil
}

func (tc *TypeChecker) typeCheckLiteralExpr(le *ast.LiteralExpr) ast.Type {
	switch le.Type {
	case lexer.INT:
		return &ast.PrimitiveType{Position: le.Position, Name: "i32"}
	case lexer.FLOAT:
		return &ast.PrimitiveType{Position: le.Position, Name: "f64"}
	case lexer.STRING:
		return &ast.PrimitiveType{Position: le.Position, Name: "string"}
	case lexer.CHAR:
		return &ast.PrimitiveType{Position: le.Position, Name: "char"}
	}
	return nil
}

func (tc *TypeChecker) typeCheckBinaryExpr(be *ast.BinaryExpr, mod *modules.Module) ast.Type {
	leftType := tc.TypeCheckExpr(be.Left, mod)
	rightType := tc.TypeCheckExpr(be.Right, mod)

	if leftType == nil || rightType == nil {
		return nil
	}

	if !tc.TypesEqual(leftType, rightType) {
		tc.reportError(be.Pos(), fmt.Sprintf("type mismatch in binary expression: left has %q, right has %q", leftType.String(), rightType.String()), "E401", len(be.Op.String()))
		return nil
	}

	switch be.Op {
	case lexer.EQ, lexer.NEQ, lexer.LT, lexer.LTE, lexer.GT, lexer.GTE:
		return &ast.PrimitiveType{Position: be.Position, Name: "bool"}
	default:
		return leftType
	}
}

func (tc *TypeChecker) typeCheckUnaryExpr(ue *ast.UnaryExpr, mod *modules.Module) ast.Type {
	rightType := tc.TypeCheckExpr(ue.Right, mod)
	if rightType == nil {
		return nil
	}

	switch ue.Op {
	case lexer.NOT:
		if rightType.String() != "bool" {
			tc.reportError(ue.Pos(), fmt.Sprintf("logical not expects bool, got %q", rightType.String()), "E406", 1)
		}
		return &ast.PrimitiveType{Position: ue.Position, Name: "bool"}
	case lexer.SUB:
		// unary minus expects numeric type
		return rightType
	}
	return rightType
}

func (tc *TypeChecker) typeCheckSelectorExpr(se *ast.SelectorExpr, mod *modules.Module) ast.Type {
	// If it is resolved to a declaration (e.g. module function), return its type
	if decl, exists := tc.resolutions[se]; exists {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			if fd.ReturnType != nil {
				return fd.ReturnType
			}
			return &ast.PrimitiveType{Position: se.Position, Name: "void"}
		}
	}

	targetType := tc.TypeCheckExpr(se.Target, mod)
	if targetType == nil {
		return nil
	}

	// Auto-dereference pointer types for member access
	underlying := targetType
	if ptr, ok := underlying.(*ast.PointerType); ok {
		underlying = ptr.To
	}

	// Resolve the underlying custom type (struct)
	if prim, ok := underlying.(*ast.PrimitiveType); ok {
		// Look up struct declaration
		structDecl := tc.findStructDecl(prim.Name, mod)
		if structDecl != nil {
			for _, f := range structDecl.Fields {
				if f.Name == se.Field {
					return f.Type
				}
			}
			tc.reportError(se.Pos(), fmt.Sprintf("field %q not found in struct %q", se.Field, prim.Name), "E407", len(se.Field))
		}
	} else if gt, ok := underlying.(*ast.GenericType); ok {
		// Monomorphized or generic struct
		structDecl := tc.findStructDecl(gt.BaseName, mod)
		if structDecl != nil {
			// Monomorphize structural fields if generic
			if len(structDecl.Generics) > 0 {
				 specializedStruct := tc.monomorphizeStruct(structDecl, gt.Params, mod)
				 if specializedStruct != nil {
					 for _, f := range specializedStruct.Fields {
						 if f.Name == se.Field {
							 return f.Type
						 }
					 }
				 }
			}
		}
	}

	return nil
}

func (tc *TypeChecker) typeCheckCallExpr(ce *ast.CallExpr, mod *modules.Module) ast.Type {
	// Handle builtins
	if ident, ok := ce.Function.(*ast.IdentExpr); ok {
		switch ident.Name {
		case "print", "println":
			for _, arg := range ce.Args {
				tc.TypeCheckExpr(arg, mod)
			}
			return &ast.PrimitiveType{Position: ce.Position, Name: "void"}
		case "len":
			if len(ce.Args) != 1 {
				tc.reportError(ce.Pos(), "len expects 1 argument", "E410", 3)
				return nil
			}
			tc.TypeCheckExpr(ce.Args[0], mod)
			return &ast.PrimitiveType{Position: ce.Position, Name: "i64"}
		case "sizeof", "alignof":
			return &ast.PrimitiveType{Position: ce.Position, Name: "i64"}
		}
	}

	// Find function declaration
	var fd *ast.FuncDecl

	decl := tc.resolutions[ce.Function]
	if decl != nil {
		fd, _ = decl.(*ast.FuncDecl)
	}

	isMethodCall := false
	if fd == nil {
		// Might be a selector expression (module.func or struct.method)
		if se, ok := ce.Function.(*ast.SelectorExpr); ok {
			if d, ok := tc.resolutions[se]; ok {
				fd, _ = d.(*ast.FuncDecl)
			} else {
				targetType := tc.TypeCheckExpr(se.Target, mod)
				if targetType != nil {
					typeName := targetType.String()
					if ptr, ok := targetType.(*ast.PointerType); ok {
						typeName = ptr.To.String()
					}
					// Look up in impl blocks across modules
					for _, m := range tc.modules {
						for _, decl := range m.AST.Decls {
							if id, ok := decl.(*ast.ImplDecl); ok && id.Target.String() == typeName {
								for _, method := range id.Methods {
									if method.Name == se.Field {
										fd = method
										tc.resolutions[se] = fd
										isMethodCall = true
										break
									}
								}
							}
							if fd != nil {
								break
							}
						}
						if fd != nil {
							break
						}
					}
				}
			}
		}
	}

	if fd == nil {
		tc.reportError(ce.Pos(), "cannot call non-function symbol", "E408", len(ce.Function.String()))
		return nil
	}

	// Monomorphize generic function if needed
	if len(fd.Generics) > 0 {
		if len(ce.Generics) != len(fd.Generics) {
			tc.reportError(ce.Pos(), fmt.Sprintf("generic parameter count mismatch: expected %d, got %d", len(fd.Generics), len(ce.Generics)), "E409", 1)
			return nil
		}
		fd = tc.monomorphizeFunc(fd, ce.Generics, mod)
		tc.resolutions[ce.Function] = fd
		if ident, ok := ce.Function.(*ast.IdentExpr); ok {
			ident.Name = fd.Name
			tc.resolutions[ident] = fd
		} else if se, ok := ce.Function.(*ast.SelectorExpr); ok {
			se.Field = fd.Name
			tc.resolutions[se] = fd
		}
	}

	// Validate arguments
	paramStart := 0
	if isMethodCall && len(fd.Params) > 0 && fd.Params[0].Name == "self" {
		paramStart = 1
	}

	expectedArgs := len(fd.Params) - paramStart
	if len(ce.Args) != expectedArgs {
		tc.reportError(ce.Pos(), fmt.Sprintf("argument count mismatch: expected %d, got %d", expectedArgs, len(ce.Args)), "E410", 1)
		return nil
	}

	for i, arg := range ce.Args {
		paramType := fd.Params[i+paramStart].Type
		argType := tc.TypeCheckExpr(arg, mod)
		if argType != nil && !tc.TypesEqual(paramType, argType) {
			tc.reportError(arg.Pos(), fmt.Sprintf("argument type mismatch: expected %q, got %q", paramType.String(), argType.String()), "E401", len(arg.String()))
		}
	}

	if fd.ReturnType != nil {
		return fd.ReturnType
	}
	return &ast.PrimitiveType{Position: ce.Position, Name: "void"}
}

func (tc *TypeChecker) typeCheckStructInitExpr(se *ast.StructInitExpr, mod *modules.Module) ast.Type {
	var sname string
	var typeParams []ast.Type
	var structDecl *ast.StructDecl

	if prim, ok := se.Struct.(*ast.PrimitiveType); ok {
		sname = prim.Name
		structDecl = tc.findStructDecl(sname, mod)
	} else if gt, ok := se.Struct.(*ast.GenericType); ok {
		sname = gt.BaseName
		typeParams = gt.Params
		structDecl = tc.findStructDecl(sname, mod)
	}

	if structDecl == nil {
		tc.reportError(se.Pos(), fmt.Sprintf("unresolved struct type %q", sname), "E304", len(sname))
		return nil
	}

	// Monomorphize struct if generic
	if len(structDecl.Generics) > 0 {
		if len(typeParams) != len(structDecl.Generics) {
			tc.reportError(se.Pos(), fmt.Sprintf("generic struct parameter count mismatch: expected %d, got %d", len(structDecl.Generics), len(typeParams)), "E409", 1)
			return nil
		}
		structDecl = tc.monomorphizeStruct(structDecl, typeParams, mod)
		se.Struct = &ast.PrimitiveType{Position: se.Struct.Pos(), Name: structDecl.Name}
	}

	// Validate fields
	fieldsMap := make(map[string]ast.Type)
	for _, f := range structDecl.Fields {
		fieldsMap[f.Name] = f.Type
	}

	for _, f := range se.Fields {
		expectedType, ok := fieldsMap[f.Name]
		if !ok {
			tc.reportError(f.Value.Pos(), fmt.Sprintf("field %q does not exist in struct %q", f.Name, structDecl.Name), "E407", len(f.Name))
			continue
		}
		valType := tc.TypeCheckExpr(f.Value, mod)
		if valType != nil && !tc.TypesEqual(expectedType, valType) {
			tc.reportError(f.Value.Pos(), fmt.Sprintf("field %q type mismatch: expected %q, got %q", f.Name, expectedType.String(), valType.String()), "E401", len(f.Value.String()))
		}
	}

	return se.Struct
}

func (tc *TypeChecker) resolveAndValidateType(t ast.Type, mod *modules.Module) ast.Type {
	if t == nil {
		return nil
	}
	switch pt := t.(type) {
	case *ast.PrimitiveType:
		// If custom struct/trait, verify it exists
		switch pt.Name {
		case "i32", "i64", "u32", "u64", "f32", "f64", "bool", "string", "char", "void", "self", "Self":
			return pt
		}
		// If in resolutions, it is resolved
		if _, ok := tc.resolutions[pt]; ok {
			return pt
		}
		// Otherwise find it
		sd := tc.findStructDecl(pt.Name, mod)
		if sd == nil {
			tc.reportError(pt.Pos(), fmt.Sprintf("unresolved type %q", pt.Name), "E304", len(pt.Name))
		} else {
			tc.resolutions[pt] = sd
		}
		return pt
	case *ast.PointerType:
		pt.To = tc.resolveAndValidateType(pt.To, mod)
		return pt
	case *ast.ArrayType:
		pt.Element = tc.resolveAndValidateType(pt.Element, mod)
		return pt
	case *ast.GenericType:
		// Monomorphize generic type references
		sd := tc.findStructDecl(pt.BaseName, mod)
		if sd != nil && len(sd.Generics) > 0 {
			spec := tc.monomorphizeStruct(sd, pt.Params, mod)
			ret := &ast.PrimitiveType{Position: pt.Position, Name: spec.Name}
			tc.resolutions[ret] = spec
			return ret
		}
		return pt
	}
	return t
}

func (tc *TypeChecker) findStructDecl(name string, mod *modules.Module) *ast.StructDecl {
	// Look up in current module
	for _, decl := range mod.AST.Decls {
		if sd, ok := decl.(*ast.StructDecl); ok && sd.Name == name {
			return sd
		}
	}
	// Look up in imported symbols
	if impSym, exists := mod.ImportedSymbols[name]; exists {
		targetMod := tc.modules[impSym.ModulePath]
		if targetMod != nil {
			if decl, exists := targetMod.Exports[impSym.SymbolName]; exists {
				if sd, ok := decl.(*ast.StructDecl); ok {
					return sd
				}
			}
		}
	}
	return nil
}

// ---------------- Monomorphization (Specialization) ----------------

func (tc *TypeChecker) monomorphizeFunc(fd *ast.FuncDecl, typeArgs []ast.Type, mod *modules.Module) *ast.FuncDecl {
	// Build specialized name
	var argStrings []string
	for _, arg := range typeArgs {
		argStrings = append(argStrings, strings.ReplaceAll(arg.String(), " ", "_"))
	}
	mangledName := fmt.Sprintf("%s__%s", fd.Name, strings.Join(argStrings, "_"))

	// Check if already specialized
	if spec, exists := tc.specialized[mangledName]; exists {
		return spec.(*ast.FuncDecl)
	}

	// Create type map
	typeMap := make(map[string]ast.Type)
	for i, gp := range fd.Generics {
		typeMap[gp.Name] = typeArgs[i]
	}

	sub := &Substituter{
		typeMap:     typeMap,
		resolutions: tc.resolutions,
		newRes:      make(map[ast.Node]ast.Node),
	}

	specializedFunc := sub.cloneAndSubstitute(fd).(*ast.FuncDecl)
	specializedFunc.Name = mangledName

	// Copy new resolutions
	for k, v := range sub.newRes {
		tc.resolutions[k] = v
	}

	// Add to module declarations so it gets compiled
	mod.AST.Decls = append(mod.AST.Decls, specializedFunc)
	tc.specialized[mangledName] = specializedFunc

	// Recursively type check the specialized function
	tc.typeCheckFuncDecl(specializedFunc, mod)

	return specializedFunc
}

func (tc *TypeChecker) monomorphizeStruct(sd *ast.StructDecl, typeArgs []ast.Type, mod *modules.Module) *ast.StructDecl {
	var argStrings []string
	for _, arg := range typeArgs {
		argStrings = append(argStrings, strings.ReplaceAll(arg.String(), " ", "_"))
	}
	mangledName := fmt.Sprintf("%s__%s", sd.Name, strings.Join(argStrings, "_"))

	if spec, exists := tc.specialized[mangledName]; exists {
		return spec.(*ast.StructDecl)
	}

	typeMap := make(map[string]ast.Type)
	for i, gp := range sd.Generics {
		typeMap[gp.Name] = typeArgs[i]
	}

	sub := &Substituter{
		typeMap:     typeMap,
		resolutions: tc.resolutions,
		newRes:      make(map[ast.Node]ast.Node),
	}

	specializedStruct := sub.cloneAndSubstitute(sd).(*ast.StructDecl)
	specializedStruct.Name = mangledName

	for k, v := range sub.newRes {
		tc.resolutions[k] = v
	}

	mod.AST.Decls = append(mod.AST.Decls, specializedStruct)
	tc.specialized[mangledName] = specializedStruct

	// Validate fields in specialized struct
	for _, f := range specializedStruct.Fields {
		tc.resolveAndValidateType(f.Type, mod)
	}

	return specializedStruct
}

func (tc *TypeChecker) reportError(pos ast.Position, msg string, code string, spanLen int) {
	diag := diagnostics.Diagnostic{
		Code:    code,
		Message: msg,
		File:    pos.File,
		Line:    pos.Line,
		Column:  pos.Col,
		SpanLen: spanLen,
	}
	tc.errors = append(tc.errors, diag)
}

// ---------------- AST Substituter for Specialization ----------------

type Substituter struct {
	typeMap     map[string]ast.Type
	resolutions map[ast.Node]ast.Node
	newRes      map[ast.Node]ast.Node
	paramMap    map[ast.Node]ast.Node
}

func (s *Substituter) cloneAndSubstitute(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	if s.paramMap == nil {
		s.paramMap = make(map[ast.Node]ast.Node)
	}

	var cloned ast.Node
	switch n := node.(type) {
	case *ast.PrimitiveType:
		if subst, exists := s.typeMap[n.Name]; exists {
			cloned = s.cloneType(subst)
		} else {
			cloned = &ast.PrimitiveType{Position: n.Position, Name: n.Name}
		}
	case *ast.PointerType:
		cloned = &ast.PointerType{
			Position: n.Position,
			To:       s.cloneAndSubstitute(n.To).(ast.Type),
			Mutable:  n.Mutable,
		}
	case *ast.ArrayType:
		cloned = &ast.ArrayType{
			Position: n.Position,
			Element:  s.cloneAndSubstitute(n.Element).(ast.Type),
			Size:     n.Size,
		}
	case *ast.GenericType:
		newParams := []ast.Type{}
		for _, p := range n.Params {
			newParams = append(newParams, s.cloneAndSubstitute(p).(ast.Type))
		}
		cloned = &ast.GenericType{
			Position:  n.Position,
			BaseName:  n.BaseName,
			Params:    newParams,
			Lifetimes: n.Lifetimes,
		}
	case *ast.BlockStmt:
		newStmts := []ast.Stmt{}
		for _, st := range n.Stmts {
			newStmts = append(newStmts, s.cloneAndSubstitute(st).(ast.Stmt))
		}
		cloned = &ast.BlockStmt{Position: n.Position, Stmts: newStmts}
	case *ast.LetStmt:
		var newType ast.Type
		if n.Type != nil {
			newType = s.cloneAndSubstitute(n.Type).(ast.Type)
		}
		clonedLet := &ast.LetStmt{
			Position: n.Position,
			Name:     n.Name,
			Type:     newType,
			Value:    s.cloneAndSubstitute(n.Value).(ast.Expr),
			Mutable:  n.Mutable,
		}
		s.paramMap[n] = clonedLet
		cloned = clonedLet
	case *ast.ReturnStmt:
		var newVal ast.Expr
		if n.Value != nil {
			newVal = s.cloneAndSubstitute(n.Value).(ast.Expr)
		}
		cloned = &ast.ReturnStmt{Position: n.Position, Value: newVal}
	case *ast.ExprStmt:
		cloned = &ast.ExprStmt{Position: n.Position, Expression: s.cloneAndSubstitute(n.Expression).(ast.Expr)}
	case *ast.IfStmt:
		var newElse ast.Stmt
		if n.ElseBlock != nil {
			newElse = s.cloneAndSubstitute(n.ElseBlock).(ast.Stmt)
		}
		cloned = &ast.IfStmt{
			Position:  n.Position,
			Condition: s.cloneAndSubstitute(n.Condition).(ast.Expr),
			ThenBlock: s.cloneAndSubstitute(n.ThenBlock).(*ast.BlockStmt),
			ElseBlock: newElse,
		}
	case *ast.ForStmt:
		clonedFor := &ast.ForStmt{
			Position:  n.Position,
			VarName:   n.VarName,
			RangeExpr: s.cloneAndSubstitute(n.RangeExpr).(ast.Expr),
			Body:      s.cloneAndSubstitute(n.Body).(*ast.BlockStmt),
		}
		s.paramMap[n] = clonedFor
		cloned = clonedFor
	case *ast.UnsafeBlock:
		cloned = &ast.UnsafeBlock{Position: n.Position, Block: s.cloneAndSubstitute(n.Block).(*ast.BlockStmt)}
	case *ast.AsmBlock:
		cloned = &ast.AsmBlock{Position: n.Position, RawText: n.RawText}
	case *ast.IdentExpr:
		cloned = &ast.IdentExpr{Position: n.Position, Name: n.Name}
	case *ast.LiteralExpr:
		cloned = &ast.LiteralExpr{Position: n.Position, Type: n.Type, Value: n.Value}
	case *ast.BinaryExpr:
		cloned = &ast.BinaryExpr{
			Position: n.Position,
			Op:       n.Op,
			Left:     s.cloneAndSubstitute(n.Left).(ast.Expr),
			Right:    s.cloneAndSubstitute(n.Right).(ast.Expr),
		}
	case *ast.UnaryExpr:
		cloned = &ast.UnaryExpr{
			Position: n.Position,
			Op:       n.Op,
			Right:    s.cloneAndSubstitute(n.Right).(ast.Expr),
		}
	case *ast.RefExpr:
		cloned = &ast.RefExpr{
			Position: n.Position,
			Mutable:  n.Mutable,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
		}
	case *ast.DerefExpr:
		cloned = &ast.DerefExpr{
			Position: n.Position,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
		}
	case *ast.SelectorExpr:
		cloned = &ast.SelectorExpr{
			Position: n.Position,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
			Field:    n.Field,
		}
	case *ast.CallExpr:
		newArgs := []ast.Expr{}
		for _, a := range n.Args {
			newArgs = append(newArgs, s.cloneAndSubstitute(a).(ast.Expr))
		}
		newGen := []ast.Type{}
		for _, g := range n.Generics {
			newGen = append(newGen, s.cloneAndSubstitute(g).(ast.Type))
		}
		cloned = &ast.CallExpr{
			Position:  n.Position,
			Function:  s.cloneAndSubstitute(n.Function).(ast.Expr),
			Generics:  newGen,
			Args:      newArgs,
			Lifetimes: n.Lifetimes,
		}
	case *ast.IndexExpr:
		cloned = &ast.IndexExpr{
			Position: n.Position,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
			Index:    s.cloneAndSubstitute(n.Index).(ast.Expr),
		}
	case *ast.StructInitExpr:
		newFields := []ast.StructInitField{}
		for _, f := range n.Fields {
			newFields = append(newFields, ast.StructInitField{
				Name:  f.Name,
				Value: s.cloneAndSubstitute(f.Value).(ast.Expr),
			})
		}
		cloned = &ast.StructInitExpr{
			Position: n.Position,
			Struct:   s.cloneAndSubstitute(n.Struct).(ast.Type),
			Fields:   newFields,
		}
	case *ast.MatchExpr:
		newCases := []ast.MatchCase{}
		for _, c := range n.Cases {
			newCases = append(newCases, ast.MatchCase{
				Pattern: s.cloneAndSubstitute(c.Pattern).(ast.Expr),
				Body:    s.cloneAndSubstitute(c.Body).(*ast.BlockStmt),
			})
		}
		cloned = &ast.MatchExpr{
			Position: n.Position,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
			Cases:    newCases,
		}
	case *ast.QuestionExpr:
		cloned = &ast.QuestionExpr{
			Position: n.Position,
			Target:   s.cloneAndSubstitute(n.Target).(ast.Expr),
		}
	case *ast.FuncDecl:
		newParams := []ast.Param{}
		for _, p := range n.Params {
			clonedParamType := s.cloneAndSubstitute(p.Type).(ast.Type)
			newParams = append(newParams, ast.Param{
				Name: p.Name,
				Type: clonedParamType,
			})
			s.paramMap[p.Type] = clonedParamType
		}
		var newRet ast.Type
		if n.ReturnType != nil {
			newRet = s.cloneAndSubstitute(n.ReturnType).(ast.Type)
		}
		var newBody *ast.BlockStmt
		if n.Body != nil {
			newBody = s.cloneAndSubstitute(n.Body).(*ast.BlockStmt)
		}
		cloned = &ast.FuncDecl{
			Position:   n.Position,
			Exported:   n.Exported,
			Name:       n.Name,
			Params:     newParams,
			ReturnType: newRet,
			Body:       newBody,
		}
	case *ast.StructDecl:
		newFields := []ast.StructField{}
		for _, f := range n.Fields {
			clonedFieldType := s.cloneAndSubstitute(f.Type).(ast.Type)
			newFields = append(newFields, ast.StructField{
				Name: f.Name,
				Type: clonedFieldType,
			})
			s.paramMap[f.Type] = clonedFieldType
		}
		cloned = &ast.StructDecl{
			Position: sdPos(n),
			Exported: n.Exported,
			Name:     n.Name,
			Fields:   newFields,
		}
	}

	// Re-map resolutions
	if target, exists := s.resolutions[node]; exists {
		// If the target has been cloned inside this substituter, resolve to the clone
		if clonedTarget, ok := s.paramMap[target]; ok {
			s.newRes[cloned] = clonedTarget
		} else {
			s.newRes[cloned] = target
		}
	}

	return cloned
}

func sdPos(sd *ast.StructDecl) ast.Position {
	return sd.Position
}

func (s *Substituter) cloneType(t ast.Type) ast.Type {
	switch n := t.(type) {
	case *ast.PrimitiveType:
		return &ast.PrimitiveType{Position: n.Position, Name: n.Name}
	case *ast.PointerType:
		return &ast.PointerType{Position: n.Position, To: s.cloneType(n.To), Mutable: n.Mutable}
	case *ast.ArrayType:
		return &ast.ArrayType{Position: n.Position, Element: s.cloneType(n.Element), Size: n.Size}
	}
	return t
}
