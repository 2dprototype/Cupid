package hir

import (
	"cupid/compiler/ast"
	"cupid/compiler/modules"
	"cupid/compiler/resolver"
	"cupid/compiler/types"
	"fmt"
	"strings"
)

// HIR Type representations
type HIRTypeKind int

const (
	TypeVoid HIRTypeKind = iota
	TypeBool
	TypeChar
	TypeI8
	TypeI16
	TypeI32
	TypeI64
	TypeU8
	TypeU16
	TypeU32
	TypeU64
	TypeF32
	TypeF64
	TypeString
	TypePointer
	TypeArray
	TypeStruct
	TypeDynTrait
)

type HIRType struct {
	Kind     HIRTypeKind
	Name     string
	ElemType *HIRType
	Mutable  bool
	Size     int // array size or byte size
	Fields   []HIRField
}

type HIRField struct {
	Name   string
	Type   *HIRType
	Offset int
}

func (t *HIRType) String() string {
	if t == nil {
		return "void"
	}
	switch t.Kind {
	case TypeVoid:
		return "void"
	case TypeBool:
		return "bool"
	case TypeChar:
		return "char"
	case TypeI8:
		return "i8"
	case TypeI16:
		return "i16"
	case TypeI32:
		return "i32"
	case TypeI64:
		return "i64"
	case TypeU8:
		return "u8"
	case TypeU16:
		return "u16"
	case TypeU32:
		return "u32"
	case TypeU64:
		return "u64"
	case TypeF32:
		return "f32"
	case TypeF64:
		return "f64"
	case TypeString:
		return "string"
	case TypePointer:
		if t.Mutable {
			return "&mut " + t.ElemType.String()
		}
		return "&" + t.ElemType.String()
	case TypeArray:
		if t.Size == -1 {
			return "[]" + t.ElemType.String()
		}
		return fmt.Sprintf("[%d]%s", t.Size, t.ElemType.String())
	case TypeStruct:
		return t.Name
	case TypeDynTrait:
		return "dyn " + t.Name
	}
	return t.Name
}

func (t *HIRType) ByteSize() int {
	if t == nil {
		return 0
	}
	switch t.Kind {
	case TypeVoid:
		return 0
	case TypeBool, TypeI8, TypeU8:
		return 1
	case TypeI16, TypeU16:
		return 2
	case TypeChar, TypeI32, TypeU32, TypeF32:
		return 4
	case TypeI64, TypeU64, TypeF64, TypePointer, TypeString:
		return 8
	case TypeArray:
		if t.Size <= 0 {
			return 16 // slice: ptr + len
		}
		return t.Size * t.ElemType.ByteSize()
	case TypeStruct:
		total := 0
		for _, f := range t.Fields {
			sz := f.Type.ByteSize()
			total += sz
		}
		if total == 0 {
			return 8
		}
		return total
	case TypeDynTrait:
		return 16 // fat pointer: data + vtable
	}
	return 8
}

// HIR AST
type HIRProgram struct {
	Structs   map[string]*HIRStruct
	Functions []*HIRFunc
	Globals   []*HIRGlobal
}

type HIRStruct struct {
	Name   string
	Type   *HIRType
	Fields []HIRField
}

type HIRGlobal struct {
	Name  string
	Type  *HIRType
	Value HIRExpr
}

type HIRFunc struct {
	Name       string
	Params     []HIRParam
	ReturnType *HIRType
	Body       *HIRBlock
}

type HIRParam struct {
	Name string
	Type *HIRType
}

type HIRStmt interface {
	hirStmt()
}

type HIRBlock struct {
	Stmts []HIRStmt
}

func (b *HIRBlock) hirStmt() {}

type HIRLetStmt struct {
	Name    string
	Mutable bool
	Type    *HIRType
	Value   HIRExpr
}

func (s *HIRLetStmt) hirStmt() {}

type HIRAssignStmt struct {
	Target HIRExpr
	Value  HIRExpr
}

func (s *HIRAssignStmt) hirStmt() {}

type HIRReturnStmt struct {
	Value HIRExpr
}

func (s *HIRReturnStmt) hirStmt() {}

type HIRExprStmt struct {
	Expr HIRExpr
}

func (s *HIRExprStmt) hirStmt() {}

type HIRIfStmt struct {
	Condition HIRExpr
	ThenBlock *HIRBlock
	ElseBlock *HIRBlock
}

func (s *HIRIfStmt) hirStmt() {}

type HIRForStmt struct {
	VarName string
	Start   HIRExpr
	End     HIRExpr
	Body    *HIRBlock
}

func (s *HIRForStmt) hirStmt() {}

type HIRBreakStmt struct{}
func (s *HIRBreakStmt) hirStmt() {}

type HIRContinueStmt struct{}
func (s *HIRContinueStmt) hirStmt() {}

type HIRAsmStmt struct {
	Assembly string
}

func (s *HIRAsmStmt) hirStmt() {}

type HIRMatchCase struct {
	Pattern HIRExpr
	Body    *HIRBlock
}

type HIRMatchStmt struct {
	Target HIRExpr
	Cases  []HIRMatchCase
}

func (s *HIRMatchStmt) hirStmt() {}

// HIR Expressions
type HIRExpr interface {
	hirExpr()
	Type() *HIRType
}

type HIRLiteral struct {
	Typ   *HIRType
	Value string
}

func (e *HIRLiteral) hirExpr()        {}
func (e *HIRLiteral) Type() *HIRType { return e.Typ }

type HIRVar struct {
	Name string
	Typ  *HIRType
}

func (e *HIRVar) hirExpr()        {}
func (e *HIRVar) Type() *HIRType { return e.Typ }

type HIRBinaryExpr struct {
	Op    string
	Left  HIRExpr
	Right HIRExpr
	Typ   *HIRType
}

func (e *HIRBinaryExpr) hirExpr()        {}
func (e *HIRBinaryExpr) Type() *HIRType { return e.Typ }

type HIRUnaryExpr struct {
	Op    string
	Right HIRExpr
	Typ   *HIRType
}

func (e *HIRUnaryExpr) hirExpr()        {}
func (e *HIRUnaryExpr) Type() *HIRType { return e.Typ }

type HIRCallExpr struct {
	FuncName string
	Args     []HIRExpr
	Typ      *HIRType
}

func (e *HIRCallExpr) hirExpr()        {}
func (e *HIRCallExpr) Type() *HIRType { return e.Typ }

type HIRStructInitExpr struct {
	StructType *HIRType
	Fields     map[string]HIRExpr
}

func (e *HIRStructInitExpr) hirExpr()        {}
func (e *HIRStructInitExpr) Type() *HIRType { return e.StructType }

type HIRFieldAccessExpr struct {
	Target HIRExpr
	Field  string
	Offset int
	Typ    *HIRType
}

func (e *HIRFieldAccessExpr) hirExpr()        {}
func (e *HIRFieldAccessExpr) Type() *HIRType { return e.Typ }

type HIRRefExpr struct {
	Target  HIRExpr
	Mutable bool
	Typ     *HIRType
}

func (e *HIRRefExpr) hirExpr()        {}
func (e *HIRRefExpr) Type() *HIRType { return e.Typ }

type HIRDerefExpr struct {
	Target HIRExpr
	Typ    *HIRType
}

func (e *HIRDerefExpr) hirExpr()        {}
func (e *HIRDerefExpr) Type() *HIRType { return e.Typ }

type HIRIndexExpr struct {
	Target HIRExpr
	Index  HIRExpr
	Typ    *HIRType
}

func (e *HIRIndexExpr) hirExpr()        {}
func (e *HIRIndexExpr) Type() *HIRType { return e.Typ }

type HIRSliceExpr struct {
	Target HIRExpr
	Low    HIRExpr
	High   HIRExpr
	Typ    *HIRType
}

func (e *HIRSliceExpr) hirExpr()        {}
func (e *HIRSliceExpr) Type() *HIRType { return e.Typ }

type HIRArrayInitExpr struct {
	Elements []HIRExpr
	Typ      *HIRType
}

func (e *HIRArrayInitExpr) hirExpr()        {}
func (e *HIRArrayInitExpr) Type() *HIRType { return e.Typ }

// HIRCastExpr represents a runtime type conversion produced by
// `typename(value)` syntax (e.g. i64(239), f32(2323), string(x)).
// `typeof(x)` does NOT lower to this node — since Cupid is statically
// typed, typeof is resolved to a constant string literal at lowering time
// (see Lowerer.lowerExpr's *ast.TypeofExpr case) and needs no runtime
// support at all.
type HIRCastExpr struct {
	Value HIRExpr
	Typ   *HIRType
}

func (e *HIRCastExpr) hirExpr()        {}
func (e *HIRCastExpr) Type() *HIRType { return e.Typ }

// Lowering AST to HIR
type Lowerer struct {
	modules     map[string]*modules.Module
	tc          *types.TypeChecker
	resolver    *resolver.Resolver
	structTypes map[string]*HIRType
}

func LowerAST(mods map[string]*modules.Module, tc *types.TypeChecker, res *resolver.Resolver) *HIRProgram {
	l := &Lowerer{
		modules:     mods,
		tc:          tc,
		resolver:    res,
		structTypes: make(map[string]*HIRType),
	}
	return l.lower()
}

func (l *Lowerer) lower() *HIRProgram {
	prog := &HIRProgram{
		Structs:   make(map[string]*HIRStruct),
		Functions: make([]*HIRFunc, 0),
		Globals:   make([]*HIRGlobal, 0),
	}

	// First pass: collect struct definitions
	for _, mod := range l.modules {
		for _, decl := range mod.AST.Decls {
			if sd, ok := decl.(*ast.StructDecl); ok {
				st := &HIRType{
					Kind: TypeStruct,
					Name: sd.Name,
				}
				l.structTypes[sd.Name] = st
			}
		}
	}

	// Fill struct fields and calculate offsets
	for _, mod := range l.modules {
		for _, decl := range mod.AST.Decls {
			if sd, ok := decl.(*ast.StructDecl); ok {
				st := l.structTypes[sd.Name]
				offset := 0
				fields := make([]HIRField, 0, len(sd.Fields))
				for _, f := range sd.Fields {
					fType := l.convertType(f.Type)
					fields = append(fields, HIRField{
						Name:   f.Name,
						Type:   fType,
						Offset: offset,
					})
					offset += fType.ByteSize()
				}
				st.Fields = fields
				st.Size = offset
				prog.Structs[sd.Name] = &HIRStruct{
					Name:   sd.Name,
					Type:   st,
					Fields: fields,
				}
			}
		}
	}

	// Lower globals, functions, and impl methods
	for _, mod := range l.modules {
		for _, decl := range mod.AST.Decls {
			switch d := decl.(type) {
			case *ast.GlobalConstDecl:
				gType := l.convertType(d.Type)
				val := l.lowerExpr(d.Value, mod)
				if gType == nil && val != nil {
					gType = val.Type()
				}
				prog.Globals = append(prog.Globals, &HIRGlobal{
					Name:  d.Name,
					Type:  gType,
					Value: val,
				})
			case *ast.FuncDecl:
				fn := l.lowerFuncDecl(d, "", mod)
				prog.Functions = append(prog.Functions, fn)
			case *ast.ImplDecl:
				targetName := d.Target.String()
				for _, m := range d.Methods {
					fn := l.lowerFuncDecl(m, targetName+"_", mod)
					prog.Functions = append(prog.Functions, fn)
				}
			}
		}
	}

	return prog
}

func (l *Lowerer) lowerFuncDecl(fd *ast.FuncDecl, prefix string, mod *modules.Module) *HIRFunc {
	fnName := prefix + fd.Name
	retType := l.convertType(fd.ReturnType)
	if retType == nil {
		retType = &HIRType{Kind: TypeVoid, Name: "void"}
	}

	targetStructName := strings.TrimSuffix(prefix, "_")

	params := make([]HIRParam, 0, len(fd.Params))
	for _, p := range fd.Params {
		pType := l.convertType(p.Type)
		if p.Name == "self" && targetStructName != "" {
			if st, ok := l.structTypes[targetStructName]; ok {
				if pType != nil && pType.Kind == TypePointer {
					pType.ElemType = st
				} else {
					pType = st
				}
			}
		}
		params = append(params, HIRParam{
			Name: p.Name,
			Type: pType,
		})
	}

	var body *HIRBlock
	if fd.Body != nil {
		body = l.lowerBlock(fd.Body, mod)
	}

	return &HIRFunc{
		Name:       fnName,
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

func (l *Lowerer) lowerBlock(bs *ast.BlockStmt, mod *modules.Module) *HIRBlock {
	if bs == nil {
		return &HIRBlock{}
	}
	block := &HIRBlock{Stmts: make([]HIRStmt, 0, len(bs.Stmts))}
	for _, s := range bs.Stmts {
		stmt := l.lowerStmt(s, mod)
		if stmt != nil {
			block.Stmts = append(block.Stmts, stmt)
		}
	}
	return block
}

func (l *Lowerer) lowerStmt(stmt ast.Stmt, mod *modules.Module) HIRStmt {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		val := l.lowerExpr(s.Value, mod)
		typ := l.convertType(s.Type)
		if typ == nil && val != nil {
			typ = val.Type()
		}
		return &HIRLetStmt{
			Name:    s.Name,
			Mutable: s.Mutable,
			Type:    typ,
			Value:   val,
		}
	case *ast.ConstStmt:
		val := l.lowerExpr(s.Value, mod)
		typ := l.convertType(s.Type)
		if typ == nil && val != nil {
			typ = val.Type()
		}
		return &HIRLetStmt{
			Name:    s.Name,
			Mutable: false,
			Type:    typ,
			Value:   val,
		}
	case *ast.ReturnStmt:
		var val HIRExpr
		if s.Value != nil {
			val = l.lowerExpr(s.Value, mod)
		}
		return &HIRReturnStmt{Value: val}
	case *ast.BreakStmt:
		return &HIRBreakStmt{}
	case *ast.ContinueStmt:
		return &HIRContinueStmt{}
	case *ast.ExprStmt:
		// Check for assignment expression desugared in AST
		if bin, ok := s.Expression.(*ast.BinaryExpr); ok && isAssignmentOp(bin.Op.String()) {
			lhs := l.lowerExpr(bin.Left, mod)
			rhs := l.lowerExpr(bin.Right, mod)
			if bin.Op.String() != "=" {
				// desugar +=, -=, etc.
				op := bin.Op.String()[:len(bin.Op.String())-1]
				rhs = &HIRBinaryExpr{
					Op:    op,
					Left:  lhs,
					Right: rhs,
					Typ:   lhs.Type(),
				}
			}
			return &HIRAssignStmt{
				Target: lhs,
				Value:  rhs,
			}
		}
		if me, ok := s.Expression.(*ast.MatchExpr); ok {
			return l.lowerStmt(me, mod)
		}
		return &HIRExprStmt{Expr: l.lowerExpr(s.Expression, mod)}
	case *ast.IfStmt:
		cond := l.lowerExpr(s.Condition, mod)
		thenBlk := l.lowerBlock(s.ThenBlock, mod)
		var elseBlk *HIRBlock
		if s.ElseBlock != nil {
			if eb, ok := s.ElseBlock.(*ast.BlockStmt); ok {
				elseBlk = l.lowerBlock(eb, mod)
			} else if elif, ok := s.ElseBlock.(*ast.IfStmt); ok {
				elseBlk = &HIRBlock{Stmts: []HIRStmt{l.lowerStmt(elif, mod)}}
			}
		}
		return &HIRIfStmt{
			Condition: cond,
			ThenBlock: thenBlk,
			ElseBlock: elseBlk,
		}
	case *ast.ForStmt:
		body := l.lowerBlock(s.Body, mod)
		var startExpr, endExpr HIRExpr
		if s.RangeExpr != nil {
			endExpr = l.lowerExpr(s.RangeExpr, mod)
		}
		return &HIRForStmt{
			VarName: s.VarName,
			Start:   startExpr,
			End:     endExpr,
			Body:    body,
		}
	case *ast.UnsafeBlock:
		return l.lowerBlock(s.Block, mod)
	case *ast.AsmBlock:
		return &HIRAsmStmt{Assembly: s.RawText}
	case *ast.MatchExpr:
		target := l.lowerExpr(s.Target, mod)
		cases := make([]HIRMatchCase, 0, len(s.Cases))
		for _, c := range s.Cases {
			var pat HIRExpr
			if ident, ok := c.Pattern.(*ast.IdentExpr); ok && ident.Name == "_" {
				pat = nil
			} else {
				pat = l.lowerExpr(c.Pattern, mod)
			}
			body := l.lowerBlock(c.Body, mod)
			cases = append(cases, HIRMatchCase{
				Pattern: pat,
				Body:    body,
			})
		}
		return &HIRMatchStmt{
			Target: target,
			Cases:  cases,
		}
	}
	return nil
}

func (l *Lowerer) lowerExpr(expr ast.Expr, mod *modules.Module) HIRExpr {
	if expr == nil {
		return nil
	}

	astType := l.tc.ExprTypes[expr]
	hirType := l.convertType(astType)

	switch e := expr.(type) {
	case *ast.LiteralExpr:
		if hirType == nil {
			hirType = l.literalType(e.Type.String(), e.Value)
		}
		return &HIRLiteral{
			Typ:   hirType,
			Value: e.Value,
		}
	case *ast.IdentExpr:
		return &HIRVar{
			Name: e.Name,
			Typ:  hirType,
		}
	case *ast.BinaryExpr:
		left := l.lowerExpr(e.Left, mod)
		right := l.lowerExpr(e.Right, mod)
		if hirType == nil && left != nil {
			hirType = left.Type()
		}
		return &HIRBinaryExpr{
			Op:    e.Op.String(),
			Left:  left,
			Right: right,
			Typ:   hirType,
		}
	case *ast.UnaryExpr:
		right := l.lowerExpr(e.Right, mod)
		if hirType == nil && right != nil {
			hirType = right.Type()
		}
		return &HIRUnaryExpr{
			Op:    e.Op.String(),
			Right: right,
			Typ:   hirType,
		}
	case *ast.CallExpr:
		funcName := ""
		args := make([]HIRExpr, 0, len(e.Args))
		if sel, ok := e.Function.(*ast.SelectorExpr); ok {
			if decl, ok := l.resolver.Resolutions[sel]; ok {
				if fd, isFunc := decl.(*ast.FuncDecl); isFunc {
					funcName = fd.Name
				} else {
					funcName = sel.Field
				}
			} else {
				target := l.lowerExpr(sel.Target, mod)
				targetType := target.Type()
				typeName := ""
				if targetType != nil {
					if targetType.Kind == TypePointer {
						typeName = targetType.ElemType.Name
					} else {
						typeName = targetType.Name
					}
				}
				funcName = typeName + "_" + sel.Field
				args = append(args, target)
			}
		} else if ident, ok := e.Function.(*ast.IdentExpr); ok {
			if decl, ok := l.resolver.Resolutions[ident]; ok {
				if fd, isFunc := decl.(*ast.FuncDecl); isFunc {
					funcName = fd.Name
				} else {
					funcName = ident.Name
				}
			} else {
				funcName = ident.Name
			}
		} else {
			funcName = e.Function.String()
		}

		for _, a := range e.Args {
			args = append(args, l.lowerExpr(a, mod))
		}
		return &HIRCallExpr{
			FuncName: funcName,
			Args:     args,
			Typ:      hirType,
		}
	case *ast.StructInitExpr:
		stType := l.convertType(e.Struct)
		fields := make(map[string]HIRExpr)
		for _, f := range e.Fields {
			fields[f.Name] = l.lowerExpr(f.Value, mod)
		}
		return &HIRStructInitExpr{
			StructType: stType,
			Fields:     fields,
		}
	case *ast.SelectorExpr:
		if decl, ok := l.resolver.Resolutions[e]; ok {
			if gc, isConst := decl.(*ast.GlobalConstDecl); isConst {
				return l.lowerExpr(gc.Value, mod)
			}
		}
		target := l.lowerExpr(e.Target, mod)
		offset := 0
		if target.Type() != nil {
			st := target.Type()
			if st.Kind == TypePointer {
				st = st.ElemType
			}
			for _, f := range st.Fields {
				if f.Name == e.Field {
					offset = f.Offset
					if hirType == nil {
						hirType = f.Type
					}
					break
				}
			}
		}
		return &HIRFieldAccessExpr{
			Target: target,
			Field:  e.Field,
			Offset: offset,
			Typ:    hirType,
		}
	case *ast.RefExpr:
		target := l.lowerExpr(e.Target, mod)
		var refType *HIRType
		if target.Type() != nil {
			refType = &HIRType{
				Kind:     TypePointer,
				ElemType: target.Type(),
				Mutable:  e.Mutable,
			}
		}
		return &HIRRefExpr{
			Target:  target,
			Mutable: e.Mutable,
			Typ:     refType,
		}
	case *ast.DerefExpr:
		target := l.lowerExpr(e.Target, mod)
		var elemType *HIRType
		if target.Type() != nil && target.Type().ElemType != nil {
			elemType = target.Type().ElemType
		}
		return &HIRDerefExpr{
			Target: target,
			Typ:    elemType,
		}
	case *ast.IndexExpr:
		target := l.lowerExpr(e.Target, mod)
		idx := l.lowerExpr(e.Index, mod)
		var elemType *HIRType
		if target.Type() != nil {
			if target.Type().Kind == TypeString {
				elemType = &HIRType{Kind: TypeU8}
			} else if target.Type().ElemType != nil {
				elemType = target.Type().ElemType
			}
		}
		return &HIRIndexExpr{
			Target: target,
			Index:  idx,
			Typ:    elemType,
		}
	case *ast.SliceExpr:
		target := l.lowerExpr(e.Target, mod)
		var low HIRExpr
		if e.Low != nil {
			low = l.lowerExpr(e.Low, mod)
		}
		var high HIRExpr
		if e.High != nil {
			high = l.lowerExpr(e.High, mod)
		}
		sliceType := hirType
		if sliceType == nil && target.Type() != nil {
			if target.Type().Kind == TypeString {
				sliceType = &HIRType{Kind: TypeString}
			} else if target.Type().Kind == TypeArray {
				sliceType = &HIRType{Kind: TypeArray, ElemType: target.Type().ElemType, Size: 0}
			}
		}
		return &HIRSliceExpr{
			Target: target,
			Low:    low,
			High:   high,
			Typ:    sliceType,
		}
	case *ast.ArrayLiteralExpr:
		elements := make([]HIRExpr, 0, len(e.Elements))
		for _, elem := range e.Elements {
			elements = append(elements, l.lowerExpr(elem, mod))
		}
		return &HIRArrayInitExpr{
			Elements: elements,
			Typ:      hirType,
		}
	case *ast.TypeCastExpr:
		val := l.lowerExpr(e.Value, mod)
		castType := hirType
		if castType == nil {
			castType = l.convertType(e.TargetType)
		}
		return &HIRCastExpr{
			Value: val,
			Typ:   castType,
		}
	case *ast.TypeofExpr:
		// Cupid has no runtime type info: the operand's static type is
		// known here, at compile time, so typeof folds straight to a
		// string constant instead of generating any code for its operand.
		typeName := "void"
		if vt, ok := l.tc.ExprTypes[e.Value]; ok && vt != nil {
			typeName = vt.String()
		}
		strType := hirType
		if strType == nil {
			strType = &HIRType{Kind: TypeString, Name: "string"}
		}
		return &HIRLiteral{
			Typ:   strType,
			Value: typeName,
		}
	}

	return nil
}

func (l *Lowerer) convertType(astType ast.Type) *HIRType {
	if astType == nil {
		return nil
	}

	switch t := astType.(type) {
	case *ast.PrimitiveType:
		switch t.Name {
		case "void":
			return &HIRType{Kind: TypeVoid, Name: "void"}
		case "bool":
			return &HIRType{Kind: TypeBool, Name: "bool"}
		case "char":
			return &HIRType{Kind: TypeChar, Name: "char"}
		case "string":
			return &HIRType{Kind: TypeString, Name: "string"}
		case "i8":
			return &HIRType{Kind: TypeI8, Name: "i8"}
		case "i16":
			return &HIRType{Kind: TypeI16, Name: "i16"}
		case "i32", "int":
			return &HIRType{Kind: TypeI32, Name: "i32"}
		case "i64", "isize":
			return &HIRType{Kind: TypeI64, Name: "i64"}
		case "u8":
			return &HIRType{Kind: TypeU8, Name: "u8"}
		case "u16":
			return &HIRType{Kind: TypeU16, Name: "u16"}
		case "u32", "uint":
			return &HIRType{Kind: TypeU32, Name: "u32"}
		case "u64", "usize":
			return &HIRType{Kind: TypeU64, Name: "u64"}
		case "f32":
			return &HIRType{Kind: TypeF32, Name: "f32"}
		case "f64":
			return &HIRType{Kind: TypeF64, Name: "f64"}
		default:
			if st, ok := l.structTypes[t.Name]; ok {
				return st
			}
			return &HIRType{Kind: TypeStruct, Name: t.Name}
		}
	case *ast.PointerType:
		return &HIRType{
			Kind:     TypePointer,
			ElemType: l.convertType(t.To),
			Mutable:  t.Mutable,
		}
	case *ast.ArrayType:
		return &HIRType{
			Kind:     TypeArray,
			ElemType: l.convertType(t.Element),
			Size:     t.Size,
		}
	case *ast.GenericType:
		return &HIRType{
			Kind: TypeStruct,
			Name: t.BaseName,
		}
	case *ast.DynTraitType:
		return &HIRType{
			Kind: TypeDynTrait,
			Name: t.Trait,
		}
	}

	return &HIRType{Kind: TypeVoid, Name: "void"}
}

func (l *Lowerer) literalType(tokenType string, value string) *HIRType {
	switch tokenType {
	case "INT":
		return &HIRType{Kind: TypeI32, Name: "i32"}
	case "FLOAT":
		return &HIRType{Kind: TypeF64, Name: "f64"}
	case "STRING":
		return &HIRType{Kind: TypeString, Name: "string"}
	case "CHAR":
		return &HIRType{Kind: TypeChar, Name: "char"}
	case "true", "false", "bool":
		return &HIRType{Kind: TypeBool, Name: "bool"}
	}
	return &HIRType{Kind: TypeI32, Name: "i32"}
}

func isAssignmentOp(op string) bool {
	return op == "=" || op == "+=" || op == "-=" || op == "*=" || op == "/=" || op == "%="
}