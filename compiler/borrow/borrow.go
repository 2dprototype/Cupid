package borrow

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/modules"
	"cupid/compiler/lexer"
	"fmt"
)

type BorrowKind int

const (
	BorrowShared BorrowKind = iota
	BorrowMut
)

type Borrow struct {
	Kind       BorrowKind
	BorrowedBy string
	Pos        ast.Position
}

type BorrowChecker struct {
	modules     map[string]*modules.Module
	resolutions map[ast.Node]ast.Node
	exprTypes   map[ast.Expr]ast.Type
	errors      []error
}

func NewBorrowChecker(
	mods map[string]*modules.Module,
	res map[ast.Node]ast.Node,
	exprTypes map[ast.Expr]ast.Type,
) *BorrowChecker {
	return &BorrowChecker{
		modules:     mods,
		resolutions: res,
		exprTypes:   exprTypes,
		errors:      []error{},
	}
}

func (bc *BorrowChecker) Errors() []error {
	return bc.errors
}

func (bc *BorrowChecker) BorrowCheckAll() bool {
	for _, mod := range bc.modules {
		bc.borrowCheckModule(mod)
	}
	return len(bc.errors) == 0
}

func (bc *BorrowChecker) borrowCheckModule(mod *modules.Module) {
	for _, decl := range mod.AST.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			bc.borrowCheckFunc(fd, mod)
		}
	}
}

func (bc *BorrowChecker) isCopyType(t ast.Type) bool {
	if t == nil {
		return true // default to copy for safety
	}
	switch pt := t.(type) {
	case *ast.PrimitiveType:
		switch pt.Name {
		case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64", "bool", "char", "void", "int", "uint", "usize", "isize":
			return true
		default:
			return false
		}
	case *ast.PointerType:
		return true // references themselves are copy
	case *ast.ArrayType:
		return bc.isCopyType(pt.Element)
	case *ast.TupleType:
		for _, el := range pt.Elements {
			if !bc.isCopyType(el) {
				return false
			}
		}
		return true
	case *ast.ChannelType:
		return false
	}
	return false
}

type borrowEnv struct {
	moved           map[string]bool
	borrows         map[string][]Borrow
	declaredInBlock map[int][]string // block nesting level -> declared variables
	nestingLevel    int
}

func (bc *BorrowChecker) borrowCheckFunc(fd *ast.FuncDecl, mod *modules.Module) {
	if fd.Body == nil {
		return
	}

	env := &borrowEnv{
		moved:           make(map[string]bool),
		borrows:         make(map[string][]Borrow),
		declaredInBlock: make(map[int][]string),
		nestingLevel:    0,
	}

	// Add receiver and parameters to declared variables of block level 0
	if fd.Receiver != nil {
		env.declaredInBlock[0] = append(env.declaredInBlock[0], fd.Receiver.Name)
	}
	for _, p := range fd.Params {
		env.declaredInBlock[0] = append(env.declaredInBlock[0], p.Name)
	}

	bc.checkBlockStmt(fd.Body, env, mod)
}

func (bc *BorrowChecker) checkBlockStmt(bs *ast.BlockStmt, env *borrowEnv, mod *modules.Module) {
	env.nestingLevel++
	defer func() {
		// Clean up variables declared at this block level
		vars := env.declaredInBlock[env.nestingLevel]
		for _, v := range vars {
			// Remove borrows by this variable
			for bName, bList := range env.borrows {
				newList := []Borrow{}
				for _, b := range bList {
					if b.BorrowedBy != v {
						newList = append(newList, b)
					}
				}
				if len(newList) == 0 {
					delete(env.borrows, bName)
				} else {
					env.borrows[bName] = newList
				}
			}
			// Remove moved status of local variables
			delete(env.moved, v)
		}
		delete(env.declaredInBlock, env.nestingLevel)
		env.nestingLevel--
	}()

	for i, stmt := range bs.Stmts {
		// Clean up dead/expired borrows (Non-Lexical Lifetimes)
		for bName, bList := range env.borrows {
			newList := []Borrow{}
			for _, b := range bList {
				if b.BorrowedBy == "" {
					continue // temporary borrow expired after its statement
				}
				if isIdentUsedInStmts(b.BorrowedBy, bs.Stmts[i:]) {
					newList = append(newList, b)
				}
			}
			if len(newList) == 0 {
				delete(env.borrows, bName)
			} else {
				env.borrows[bName] = newList
			}
		}

		bc.checkStmt(stmt, env, mod)

		// Clean up temporary anonymous borrows created during this statement
		for bName, bList := range env.borrows {
			newList := []Borrow{}
			for _, b := range bList {
				if b.BorrowedBy != "" {
					newList = append(newList, b)
				}
			}
			if len(newList) == 0 {
				delete(env.borrows, bName)
			} else {
				env.borrows[bName] = newList
			}
		}
	}
}

func (bc *BorrowChecker) checkStmt(stmt ast.Stmt, env *borrowEnv, mod *modules.Module) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		// 1. Check value expression
		bc.checkExpr(s.Value, env, mod, false)

		// 2. Register variable(s)
		if s.Pattern != nil {
			bc.collectPatternVars(s.Pattern, env)
		} else {
			env.declaredInBlock[env.nestingLevel] = append(env.declaredInBlock[env.nestingLevel], s.Name)
		}

		// 3. Detect if this variable is a borrow creation
		if ref, ok := s.Value.(*ast.RefExpr); ok {
			if ident, ok := ref.Target.(*ast.IdentExpr); ok {
				kind := BorrowShared
				if ref.Mutable {
					kind = BorrowMut
				}

				// Check borrowing rules
				varName := s.Name
				if s.Pattern != nil {
					varName = s.Pattern.String()
				}
				bc.addBorrow(ident.Name, varName, kind, ref.Position, env)
			}
		}

	case *ast.ReturnStmt:
		if s.Value != nil {
			bc.checkExpr(s.Value, env, mod, true) // return moves value
		}

	case *ast.BreakStmt, *ast.ContinueStmt:
		// Control flow statements have no variable moves or borrows

	case *ast.ExprStmt:
		// Check assignments
		if bin, ok := s.Expression.(*ast.BinaryExpr); ok && isAssignmentOp(bin.Op) {
			// Left is assignment target
			bc.checkExpr(bin.Right, env, mod, true)
			bc.checkAssignTarget(bin.Left, env, mod)
		} else {
			bc.checkExpr(s.Expression, env, mod, false)
		}

	case *ast.IfStmt:
		bc.checkExpr(s.Condition, env, mod, false)
		bc.checkBlockStmt(s.ThenBlock, env, mod)
		if s.ElseBlock != nil {
			bc.checkStmt(s.ElseBlock, env, mod)
		}

	case *ast.ForStmt:
		if s.VarName != "" {
			env.declaredInBlock[env.nestingLevel] = append(env.declaredInBlock[env.nestingLevel], s.VarName)
		}
		if s.RangeExpr != nil {
			bc.checkExpr(s.RangeExpr, env, mod, false)
		}
		bc.checkBlockStmt(s.Body, env, mod)

	case *ast.UnsafeBlock:
		bc.checkBlockStmt(s.Block, env, mod)

	case *ast.AsmBlock:
		// Raw assembly statement

	case *ast.GoStmt:
		bc.checkExpr(s.Call, env, mod, false)

	case *ast.SelectStmt:
		for _, c := range s.Cases {
			if c.ChannelOp != nil {
				bc.checkExpr(c.ChannelOp, env, mod, false)
			}
			if c.VarName != "" {
				env.declaredInBlock[env.nestingLevel] = append(env.declaredInBlock[env.nestingLevel], c.VarName)
			}
			if c.Body != nil {
				bc.checkBlockStmt(c.Body, env, mod)
			}
		}
		if s.Default != nil {
			bc.checkBlockStmt(s.Default, env, mod)
		}
	}
}

func (bc *BorrowChecker) checkExpr(expr ast.Expr, env *borrowEnv, mod *modules.Module, isMoveContext bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.IdentExpr:
		// Check use after move
		if env.moved[e.Name] {
			bc.reportError(e.Pos(), fmt.Sprintf("use of moved value %q", e.Name), "E202", len(e.Name))
			return
		}

		// If this is a move context and the type is not Copy, mark it as moved
		if isMoveContext {
			t := bc.exprTypes[e]
			if !bc.isCopyType(t) {
				// Verify it has no active borrows
				if len(env.borrows[e.Name]) > 0 {
					bc.reportError(e.Pos(), fmt.Sprintf("cannot move %q because it is borrowed", e.Name), "E204", len(e.Name))
				} else {
					env.moved[e.Name] = true
				}
			}
		}

	case *ast.BinaryExpr:
		// Check both operands
		bc.checkExpr(e.Left, env, mod, isMoveContext)
		bc.checkExpr(e.Right, env, mod, isMoveContext)

	case *ast.UnaryExpr:
		bc.checkExpr(e.Right, env, mod, isMoveContext)

	case *ast.RefExpr:
		// Address-of expression. E.g. &x or &mut x
		if ident, ok := e.Target.(*ast.IdentExpr); ok {
			decl := bc.resolutions[ident]
			if e.Mutable {
				if letStmt, ok := decl.(*ast.LetStmt); ok && !letStmt.Mutable {
					bc.reportError(e.Pos(), fmt.Sprintf("cannot borrow immutable variable %q as mutable (declare with 'mut' to make it mutable)", ident.Name), "E201", len(ident.Name))
				} else if param, ok := decl.(*ast.Param); ok && !param.Mutable {
					bc.reportError(e.Pos(), fmt.Sprintf("cannot borrow immutable parameter %q as mutable", param.Name), "E201", len(param.Name))
				}
			}
			kind := BorrowShared
			if e.Mutable {
				kind = BorrowMut
			}
			bc.addBorrow(ident.Name, "", kind, e.Position, env)
		} else {
			bc.checkExpr(e.Target, env, mod, false)
		}

	case *ast.DerefExpr:
		bc.checkExpr(e.Target, env, mod, false)

	case *ast.SelectorExpr:
		bc.checkExpr(e.Target, env, mod, false)

	case *ast.CallExpr:
		// Call expression. If an argument is passed by value (non-pointer), it might be moved!
		decl := bc.resolutions[e.Function]
		var fd *ast.FuncDecl
		if decl != nil {
			fd, _ = decl.(*ast.FuncDecl)
		}

		for i, arg := range e.Args {
			// Check if parameter is a reference or a value
			isArgMove := false
			if fd != nil && i < len(fd.Params) {
				paramType := fd.Params[i].Type
				if _, ok := paramType.(*ast.PointerType); !ok {
					// Passed by value!
					isArgMove = true
				}
			}
			bc.checkExpr(arg, env, mod, isArgMove)
		}

	case *ast.IndexExpr:
		bc.checkExpr(e.Target, env, mod, false)
		bc.checkExpr(e.Index, env, mod, false)

	case *ast.TupleExpr:
		for _, el := range e.Elements {
			bc.checkExpr(el, env, mod, isMoveContext)
		}

	case *ast.StructInitExpr:
		for _, f := range e.Fields {
			bc.checkExpr(f.Value, env, mod, isMoveContext)
		}

	case *ast.MatchExpr:
		bc.checkExpr(e.Target, env, mod, false)
		for _, c := range e.Cases {
			bc.checkExpr(c.Pattern, env, mod, false)
			bc.checkBlockStmt(c.Body, env, mod)
		}

	case *ast.TypeCastExpr:
		// Cast sources are always scalar Copy types (enforced by the type
		// checker), so this is never a move context, but we still need to
		// walk into the operand to catch use-after-move / borrow errors.
		bc.checkExpr(e.Value, env, mod, false)

	case *ast.TypeofExpr:
		bc.checkExpr(e.Value, env, mod, false)
	}
}

func (bc *BorrowChecker) collectPatternVars(pat ast.Expr, env *borrowEnv) {
	if pat == nil {
		return
	}
	switch p := pat.(type) {
	case *ast.IdentExpr:
		if p.Name != "_" {
			env.declaredInBlock[env.nestingLevel] = append(env.declaredInBlock[env.nestingLevel], p.Name)
		}
	case *ast.TupleExpr:
		for _, el := range p.Elements {
			bc.collectPatternVars(el, env)
		}
	case *ast.CallExpr:
		for _, arg := range p.Args {
			bc.collectPatternVars(arg, env)
		}
	}
}

func (bc *BorrowChecker) checkAssignTarget(target ast.Expr, env *borrowEnv, mod *modules.Module) {
	if ident, ok := target.(*ast.IdentExpr); ok {
		decl := bc.resolutions[ident]
		if letStmt, ok := decl.(*ast.LetStmt); ok && !letStmt.Mutable {
			bc.reportError(ident.Pos(), fmt.Sprintf("cannot assign to immutable variable %q (declare with 'mut' to make it mutable)", ident.Name), "E201", len(ident.Name))
			return
		}
		if param, ok := decl.(*ast.Param); ok && !param.Mutable {
			bc.reportError(ident.Pos(), fmt.Sprintf("cannot assign to immutable parameter %q (declare with 'mut' to make it mutable)", param.Name), "E201", len(param.Name))
			return
		}
		if _, ok := decl.(*ast.GlobalConstDecl); ok {
			bc.reportError(ident.Pos(), fmt.Sprintf("cannot assign to constant %q", ident.Name), "E201", len(ident.Name))
			return
		}
		if gv, ok := decl.(*ast.GlobalVarDecl); ok && !gv.Mutable {
			bc.reportError(ident.Pos(), fmt.Sprintf("cannot assign to immutable variable %q (declare with 'mut' to make it mutable)", ident.Name), "E201", len(ident.Name))
			return
		}

		// Verify variable is not borrowed
		if len(env.borrows[ident.Name]) > 0 {
			bc.reportError(ident.Pos(), fmt.Sprintf("cannot assign to %q because it is borrowed", ident.Name), "E203", len(ident.Name))
		}
		// Re-assignment makes a moved variable initialized again!
		delete(env.moved, ident.Name)
	} else if selector, ok := target.(*ast.SelectorExpr); ok {
		bc.checkAssignTarget(selector.Target, env, mod)
	} else if idx, ok := target.(*ast.IndexExpr); ok {
		bc.checkAssignTarget(idx.Target, env, mod)
	}
}

func isAssignmentOp(op lexer.TokenType) bool {
	return op == lexer.ASSIGN || op == lexer.ADD_ASSIGN || op == lexer.SUB_ASSIGN ||
		op == lexer.MUL_ASSIGN || op == lexer.DIV_ASSIGN || op == lexer.MOD_ASSIGN
}

func (bc *BorrowChecker) addBorrow(varName string, borrowedBy string, kind BorrowKind, pos ast.Position, env *borrowEnv) {
	if env.moved[varName] {
		bc.reportError(pos, fmt.Sprintf("cannot borrow moved value %q", varName), "E202", len(varName))
		return
	}

	activeBorrows := env.borrows[varName]
	for _, b := range activeBorrows {
		// Mut borrow conflicts with everything
		if kind == BorrowMut || b.Kind == BorrowMut {
			bc.reportError(pos, fmt.Sprintf("cannot borrow %q as mutable because it is already borrowed", varName), "E201", len(varName))
			return
		}
	}

	env.borrows[varName] = append(env.borrows[varName], Borrow{
		Kind:       kind,
		BorrowedBy: borrowedBy,
		Pos:        pos,
	})
}

func (bc *BorrowChecker) reportError(pos ast.Position, msg string, code string, spanLen int) {
	diag := diagnostics.Diagnostic{
		Code:    code,
		Message: msg,
		File:    pos.File,
		Line:    pos.Line,
		Column:  pos.Col,
		SpanLen: spanLen,
	}
	bc.errors = append(bc.errors, diag)
}

func isIdentUsedInExpr(name string, expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return e.Name == name
	case *ast.BinaryExpr:
		return isIdentUsedInExpr(name, e.Left) || isIdentUsedInExpr(name, e.Right)
	case *ast.UnaryExpr:
		return isIdentUsedInExpr(name, e.Right)
	case *ast.RefExpr:
		return isIdentUsedInExpr(name, e.Target)
	case *ast.DerefExpr:
		return isIdentUsedInExpr(name, e.Target)
	case *ast.SelectorExpr:
		return isIdentUsedInExpr(name, e.Target)
	case *ast.IndexExpr:
		return isIdentUsedInExpr(name, e.Target) || isIdentUsedInExpr(name, e.Index)
	case *ast.SliceExpr:
		return isIdentUsedInExpr(name, e.Target) || isIdentUsedInExpr(name, e.Low) || isIdentUsedInExpr(name, e.High)
	case *ast.CallExpr:
		if isIdentUsedInExpr(name, e.Function) {
			return true
		}
		for _, arg := range e.Args {
			if isIdentUsedInExpr(name, arg) {
				return true
			}
		}
		return false
	case *ast.StructInitExpr:
		for _, f := range e.Fields {
			if isIdentUsedInExpr(name, f.Value) {
				return true
			}
		}
		return false
	case *ast.ArrayLiteralExpr:
		for _, el := range e.Elements {
			if isIdentUsedInExpr(name, el) {
				return true
			}
		}
		return false
	case *ast.TypeCastExpr:
		return isIdentUsedInExpr(name, e.Value)
	}
	return false
}

func isIdentUsedInStmt(name string, stmt ast.Stmt) bool {
	if stmt == nil {
		return false
	}
	switch s := stmt.(type) {
	case *ast.LetStmt:
		return isIdentUsedInExpr(name, s.Value)
	case *ast.ConstStmt:
		return isIdentUsedInExpr(name, s.Value)
	case *ast.ReturnStmt:
		return isIdentUsedInExpr(name, s.Value)
	case *ast.ExprStmt:
		return isIdentUsedInExpr(name, s.Expression)
	case *ast.IfStmt:
		if isIdentUsedInExpr(name, s.Condition) || isIdentUsedInBlock(name, s.ThenBlock) {
			return true
		}
		if s.ElseBlock != nil {
			return isIdentUsedInStmt(name, s.ElseBlock)
		}
		return false
	case *ast.ForStmt:
		if isIdentUsedInExpr(name, s.RangeExpr) || isIdentUsedInBlock(name, s.Body) {
			return true
		}
		return false
	case *ast.BlockStmt:
		return isIdentUsedInBlock(name, s)
	case *ast.UnsafeBlock:
		return isIdentUsedInBlock(name, s.Block)
	case *ast.AsmBlock:
		return false
	case *ast.GoStmt:
		return isIdentUsedInExpr(name, s.Call)
	case *ast.SelectStmt:
		for _, c := range s.Cases {
			if isIdentUsedInExpr(name, c.ChannelOp) || isIdentUsedInBlock(name, c.Body) {
				return true
			}
		}
		if s.Default != nil {
			return isIdentUsedInBlock(name, s.Default)
		}
		return false
	}
	return false
}

func isIdentUsedInBlock(name string, bs *ast.BlockStmt) bool {
	if bs == nil {
		return false
	}
	for _, s := range bs.Stmts {
		if isIdentUsedInStmt(name, s) {
			return true
		}
	}
	return false
}

func isIdentUsedInStmts(name string, stmts []ast.Stmt) bool {
	for _, s := range stmts {
		if isIdentUsedInStmt(name, s) {
			return true
		}
	}
	return false
}