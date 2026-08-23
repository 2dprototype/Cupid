package ast

import (
	"bytes"
	"fmt"
	"strings"
	"cupid/compiler/lexer"
)

// Position tracks where in the source file a node came from
type Position struct {
	File string
	Line int
	Col  int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col)
}

type Node interface {
	Pos() Position
	String() string
}

type Expr interface {
	Node
	exprNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Decl interface {
	Node
	declNode()
}

// ---------------- Types in the AST ----------------

type Type interface {
	Node
	typeNode()
}

type PrimitiveType struct {
	Position Position
	Name     string // bool, string, i32, f64, etc.
}
func (pt *PrimitiveType) Pos() Position { return pt.Position }
func (pt *PrimitiveType) String() string { return pt.Name }
func (pt *PrimitiveType) typeNode() {}

type PointerType struct {
	Position Position
	To       Type
	Mutable  bool
}
func (pt *PointerType) Pos() Position { return pt.Position }
func (pt *PointerType) String() string {
	if pt.Mutable {
		return "&mut " + pt.To.String()
	}
	return "&" + pt.To.String()
}
func (pt *PointerType) typeNode() {}

type ArrayType struct {
	Position Position
	Element  Type
	Size     int // -1 for slice
}
func (at *ArrayType) Pos() Position { return at.Position }
func (at *ArrayType) String() string {
	if at.Size == -1 {
		return "[]" + at.Element.String()
	}
	return fmt.Sprintf("[%d]%s", at.Size, at.Element.String())
}
func (at *ArrayType) typeNode() {}

type GenericType struct {
	Position   Position
	BaseName   string
	Params     []Type
	Lifetimes  []string
}
func (gt *GenericType) Pos() Position { return gt.Position }
func (gt *GenericType) String() string {
	var sb strings.Builder
	sb.WriteString(gt.BaseName)
	if len(gt.Lifetimes) > 0 || len(gt.Params) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range gt.Lifetimes {
			parts = append(parts, "'" + lt)
		}
		for _, p := range gt.Params {
			parts = append(parts, p.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	return sb.String()
}
func (gt *GenericType) typeNode() {}

type ChannelType struct {
	Position Position
	Value    Type
}
func (ct *ChannelType) Pos() Position { return ct.Position }
func (ct *ChannelType) String() string { return "channel<" + ct.Value.String() + ">" }
func (ct *ChannelType) typeNode() {}

type DynTraitType struct {
	Position Position
	Trait    string
}
func (dt *DynTraitType) Pos() Position { return dt.Position }
func (dt *DynTraitType) String() string { return "dyn " + dt.Trait }
func (dt *DynTraitType) typeNode() {}

// ---------------- Declarations ----------------

type Program struct {
	Position Position
	Decls    []Decl
}
func (p *Program) Pos() Position { return p.Position }
func (p *Program) String() string {
	var out bytes.Buffer
	for _, d := range p.Decls {
		out.WriteString(d.String())
		out.WriteString("\n")
	}
	return out.String()
}

type ImportDecl struct {
	Position    Position
	Path        string // e.g. "math"
	Alias       string // e.g. "gfx" (math as gfx)
	Symbols     []ImportSymbol
	FromModule  string // if imported from, e.g. "entities"
}
func (id *ImportDecl) Pos() Position { return id.Position }
func (id *ImportDecl) String() string {
	if len(id.Symbols) > 0 {
		syms := []string{}
		for _, s := range id.Symbols {
			if s.Alias != "" {
				syms = append(syms, s.Name+" as "+s.Alias)
			} else {
				syms = append(syms, s.Name)
			}
		}
		return fmt.Sprintf("import {%s} from %q", strings.Join(syms, ", "), id.FromModule)
	}
	if id.Alias != "" {
		return fmt.Sprintf("import %q as %s", id.Path, id.Alias)
	}
	return fmt.Sprintf("import %q", id.Path)
}
func (id *ImportDecl) declNode() {}

type ImportSymbol struct {
	Name  string
	Alias string
}

type FuncDecl struct {
	Position     Position
	Exported     bool
	Receiver     *Param // optional method receiver e.g. (p: Point)
	Name         string
	Lifetimes    []string
	Generics     []GenericParam
	Params       []Param
	ReturnType   Type // nil means no return value
	WhereClauses []WhereClause
	Body         *BlockStmt
}
func (fd *FuncDecl) Pos() Position { return fd.Position }
func (fd *FuncDecl) String() string {
	var sb strings.Builder
	if fd.Exported {
		sb.WriteString("export ")
	}
	sb.WriteString("fn ")
	if fd.Receiver != nil {
		sb.WriteString("(" + fd.Receiver.String() + ") ")
	}
	sb.WriteString(fd.Name)
	if len(fd.Lifetimes) > 0 || len(fd.Generics) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range fd.Lifetimes {
			parts = append(parts, "'"+lt)
		}
		for _, g := range fd.Generics {
			parts = append(parts, g.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	sb.WriteString("(")
	params := []string{}
	for _, p := range fd.Params {
		params = append(params, p.Name+": "+p.Type.String())
	}
	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(")")
	if fd.ReturnType != nil {
		sb.WriteString(" -> " + fd.ReturnType.String())
	}
	if len(fd.WhereClauses) > 0 {
		sb.WriteString(" where ")
		clauses := []string{}
		for _, w := range fd.WhereClauses {
			clauses = append(clauses, w.String())
		}
		sb.WriteString(strings.Join(clauses, ", "))
	}
	if fd.Body != nil {
		sb.WriteString(" " + fd.Body.String())
	}
	return sb.String()
}
func (fd *FuncDecl) declNode() {}

type GenericParam struct {
	Name   string
	Bounds []string // trait bounds
}
func (gp GenericParam) String() string {
	if len(gp.Bounds) > 0 {
		return gp.Name + ": " + strings.Join(gp.Bounds, " + ")
	}
	return gp.Name
}

type Param struct {
	Position Position
	Name     string
	Mutable  bool
	Type     Type
}
func (p *Param) Pos() Position { return p.Position }
func (p *Param) String() string {
	if p.Mutable {
		return "mut " + p.Name + ": " + p.Type.String()
	}
	return p.Name + ": " + p.Type.String()
}

type WhereClause struct {
	ParamName string
	Bounds    []string
}
func (wc WhereClause) String() string {
	return wc.ParamName + ": " + strings.Join(wc.Bounds, " + ")
}

type StructDecl struct {
	Position  Position
	Exported  bool
	Name      string
	Lifetimes []string
	Generics  []GenericParam
	Fields    []StructField
}
func (sd *StructDecl) Pos() Position { return sd.Position }
func (sd *StructDecl) String() string {
	var sb strings.Builder
	if sd.Exported {
		sb.WriteString("export ")
	}
	sb.WriteString("struct " + sd.Name)
	if len(sd.Lifetimes) > 0 || len(sd.Generics) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range sd.Lifetimes {
			parts = append(parts, "'"+lt)
		}
		for _, g := range sd.Generics {
			parts = append(parts, g.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	sb.WriteString(" {\n")
	for _, f := range sd.Fields {
		sb.WriteString(fmt.Sprintf("    %s %s\n", f.Name, f.Type.String()))
	}
	sb.WriteString("}")
	return sb.String()
}
func (sd *StructDecl) declNode() {}

type StructField struct {
	Position Position
	Name     string
	Type     Type
}
func (sf *StructField) Pos() Position { return sf.Position }
func (sf *StructField) String() string { return sf.Name + " " + sf.Type.String() }

type ImplDecl struct {
	Position  Position
	TraitName string // optional
	Target    Type
	Lifetimes []string
	Generics  []GenericParam
	Methods   []*FuncDecl
}
func (id *ImplDecl) Pos() Position { return id.Position }
func (id *ImplDecl) String() string {
	var sb strings.Builder
	sb.WriteString("impl")
	if len(id.Lifetimes) > 0 || len(id.Generics) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range id.Lifetimes {
			parts = append(parts, "'"+lt)
		}
		for _, g := range id.Generics {
			parts = append(parts, g.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	if id.TraitName != "" {
		sb.WriteString(" " + id.TraitName + " for")
	}
	sb.WriteString(" " + id.Target.String() + " {\n")
	for _, m := range id.Methods {
		sb.WriteString(m.String())
		sb.WriteString("\n")
	}
	sb.WriteString("}")
	return sb.String()
}
func (id *ImplDecl) declNode() {}

type TraitDecl struct {
	Position  Position
	Exported  bool
	Name      string
	Lifetimes []string
	Generics  []GenericParam
	Methods   []TraitMethodSignature
}
func (td *TraitDecl) Pos() Position { return td.Position }
func (td *TraitDecl) String() string {
	var sb strings.Builder
	if td.Exported {
		sb.WriteString("export ")
	}
	sb.WriteString("trait " + td.Name)
	if len(td.Lifetimes) > 0 || len(td.Generics) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range td.Lifetimes {
			parts = append(parts, "'"+lt)
		}
		for _, g := range td.Generics {
			parts = append(parts, g.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	sb.WriteString(" {\n")
	for _, m := range td.Methods {
		sb.WriteString("    fn " + m.Name + "(")
		params := []string{}
		for _, p := range m.Params {
			params = append(params, p.Name+": "+p.Type.String())
		}
		sb.WriteString(strings.Join(params, ", "))
		sb.WriteString(")")
		if m.ReturnType != nil {
			sb.WriteString(" -> " + m.ReturnType.String())
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}")
	return sb.String()
}
func (td *TraitDecl) declNode() {}

type TraitMethodSignature struct {
	Name       string
	Params     []Param
	ReturnType Type
}

type GlobalConstDecl struct {
	Position Position
	Exported bool
	Name     string
	Type     Type // optional
	Value    Expr
}
func (gc *GlobalConstDecl) Pos() Position { return gc.Position }
func (gc *GlobalConstDecl) String() string {
	var sb strings.Builder
	if gc.Exported {
		sb.WriteString("export ")
	}
	sb.WriteString("const " + gc.Name)
	if gc.Type != nil {
		sb.WriteString(": " + gc.Type.String())
	}
	sb.WriteString(" = " + gc.Value.String())
	return sb.String()
}
func (gc *GlobalConstDecl) declNode() {}

type GlobalVarDecl struct {
	Position Position
	Exported bool
	Mutable  bool
	Name     string
	Type     Type // optional
	Value    Expr
}
func (gv *GlobalVarDecl) Pos() Position { return gv.Position }
func (gv *GlobalVarDecl) String() string {
	kw := "let"
	if gv.Mutable {
		kw = "mut"
	}
	var sb strings.Builder
	if gv.Exported {
		sb.WriteString("export ")
	}
	sb.WriteString(kw + " " + gv.Name)
	if gv.Type != nil {
		sb.WriteString(": " + gv.Type.String())
	}
	sb.WriteString(" = " + gv.Value.String())
	return sb.String()
}
func (gv *GlobalVarDecl) declNode() {}

// ---------------- Statements ----------------

type LetStmt struct {
	Position Position
	Name     string
	Mutable  bool
	Type     Type // optional
	Value    Expr
}
func (ls *LetStmt) Pos() Position { return ls.Position }
func (ls *LetStmt) String() string {
	keyword := "let"
	if ls.Mutable {
		keyword = "mut" // Note: in spec, "mut score = 0" or "let mut score = 0"? Let's check plan.md.
		// plan.md line 542: "`let` defines immutable bindings", "`mut` defines mutable bindings".
		// line 550: "let name = \"Tokio\" \n mut score = 0"
		// So mutable binding is "mut score = 0", immutable is "let score = 0"
	}
	var sb strings.Builder
	sb.WriteString(keyword + " " + ls.Name)
	if ls.Type != nil {
		sb.WriteString(": " + ls.Type.String())
	}
	sb.WriteString(" = " + ls.Value.String())
	return sb.String()
}
func (ls *LetStmt) stmtNode() {}

type ConstStmt struct {
	Position Position
	Name     string
	Type     Type // optional
	Value    Expr
}
func (cs *ConstStmt) Pos() Position { return cs.Position }
func (cs *ConstStmt) String() string {
	var sb strings.Builder
	sb.WriteString("const " + cs.Name)
	if cs.Type != nil {
		sb.WriteString(": " + cs.Type.String())
	}
	sb.WriteString(" = " + cs.Value.String())
	return sb.String()
}
func (cs *ConstStmt) stmtNode() {}

type ExprStmt struct {
	Position Position
	Expression Expr
}
func (es *ExprStmt) Pos() Position { return es.Position }
func (es *ExprStmt) String() string { return es.Expression.String() }
func (es *ExprStmt) stmtNode() {}

type ReturnStmt struct {
	Position Position
	Value    Expr // optional
}
func (rs *ReturnStmt) Pos() Position { return rs.Position }
func (rs *ReturnStmt) String() string {
	if rs.Value != nil {
		return "return " + rs.Value.String()
	}
	return "return"
}
func (rs *ReturnStmt) stmtNode() {}

type BreakStmt struct {
	Position Position
}
func (bs *BreakStmt) Pos() Position { return bs.Position }
func (bs *BreakStmt) String() string { return "break" }
func (bs *BreakStmt) stmtNode() {}

type ContinueStmt struct {
	Position Position
}
func (cs *ContinueStmt) Pos() Position { return cs.Position }
func (cs *ContinueStmt) String() string { return "continue" }
func (cs *ContinueStmt) stmtNode() {}

type GoStmt struct {
	Position Position
	Call     *CallExpr
}
func (gs *GoStmt) Pos() Position { return gs.Position }
func (gs *GoStmt) String() string { return "go " + gs.Call.String() }
func (gs *GoStmt) stmtNode() {}

type SelectStmt struct {
	Position Position
	Cases    []SelectCase
	Default  *BlockStmt // nil if no default case
}
func (ss *SelectStmt) Pos() Position { return ss.Position }
func (ss *SelectStmt) String() string {
	var sb strings.Builder
	sb.WriteString("select {\n")
	for _, c := range ss.Cases {
		sb.WriteString("    case ")
		if c.VarName != "" {
			sb.WriteString(c.VarName + " = ")
		}
		sb.WriteString(c.ChannelOp.String() + ":\n")
		sb.WriteString("        " + c.Body.String() + "\n")
	}
	if ss.Default != nil {
		sb.WriteString("    default:\n")
		sb.WriteString("        " + ss.Default.String() + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}
func (ss *SelectStmt) stmtNode() {}

type SelectCase struct {
	VarName   string // e.g. "value" in "case value = jobs.recv()"
	ChannelOp Expr   // jobs.recv()
	Body      *BlockStmt
}

type BlockStmt struct {
	Position Position
	Stmts    []Stmt
}
func (bs *BlockStmt) Pos() Position { return bs.Position }
func (bs *BlockStmt) String() string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, s := range bs.Stmts {
		sb.WriteString("    " + s.String() + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}
func (bs *BlockStmt) stmtNode() {}

type UnsafeBlock struct {
	Position Position
	Block    *BlockStmt
}
func (ub *UnsafeBlock) Pos() Position { return ub.Position }
func (ub *UnsafeBlock) String() string { return "unsafe " + ub.Block.String() }
func (ub *UnsafeBlock) stmtNode() {}

type AsmBlock struct {
	Position Position
	RawText  string
}
func (ab *AsmBlock) Pos() Position { return ab.Position }
func (ab *AsmBlock) String() string { return "asm {\n" + ab.RawText + "}" }
func (ab *AsmBlock) stmtNode() {}

type IfStmt struct {
	Position  Position
	Condition Expr
	ThenBlock *BlockStmt
	ElseBlock Stmt // *BlockStmt or *IfStmt or nil
}
func (is *IfStmt) Pos() Position { return is.Position }
func (is *IfStmt) String() string {
	var sb strings.Builder
	sb.WriteString("if " + is.Condition.String() + " " + is.ThenBlock.String())
	if is.ElseBlock != nil {
		sb.WriteString(" else " + is.ElseBlock.String())
	}
	return sb.String()
}
func (is *IfStmt) stmtNode() {}

type ForStmt struct {
	Position  Position
	VarName   string // optional loop variable
	RangeExpr Expr   // optional condition or iterator expression
	Body      *BlockStmt
}
func (fs *ForStmt) Pos() Position { return fs.Position }
func (fs *ForStmt) String() string {
	var sb strings.Builder
	sb.WriteString("for ")
	if fs.VarName != "" {
		sb.WriteString(fs.VarName + " in ")
	}
	if fs.RangeExpr != nil {
		sb.WriteString(fs.RangeExpr.String() + " ")
	}
	sb.WriteString(fs.Body.String())
	return sb.String()
}
func (fs *ForStmt) stmtNode() {}

// ---------------- Expressions ----------------

type IdentExpr struct {
	Position Position
	Name     string
}
func (ie *IdentExpr) Pos() Position { return ie.Position }
func (ie *IdentExpr) String() string { return ie.Name }
func (ie *IdentExpr) exprNode() {}

type LiteralExpr struct {
	Position Position
	Type     lexer.TokenType // INT, FLOAT, STRING, CHAR, or boolean via IDENT
	Value    string
}
func (le *LiteralExpr) Pos() Position { return le.Position }
func (le *LiteralExpr) String() string {
	if le.Type == lexer.STRING {
		return `"` + le.Value + `"`
	}
	return le.Value
}
func (le *LiteralExpr) exprNode() {}

type BinaryExpr struct {
	Position Position
	Op       lexer.TokenType
	Left     Expr
	Right    Expr
}
func (be *BinaryExpr) Pos() Position { return be.Position }
func (be *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", be.Left.String(), be.Op.String(), be.Right.String())
}
func (be *BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Position Position
	Op       lexer.TokenType
	Right    Expr
}
func (ue *UnaryExpr) Pos() Position { return ue.Position }
func (ue *UnaryExpr) String() string {
	return ue.Op.String() + ue.Right.String()
}
func (ue *UnaryExpr) exprNode() {}

type CallExpr struct {
	Position  Position
	Function  Expr // IdentExpr or SelectorExpr
	Lifetimes []string
	Generics  []Type
	Args      []Expr
}
func (ce *CallExpr) Pos() Position { return ce.Position }
func (ce *CallExpr) String() string {
	var sb strings.Builder
	sb.WriteString(ce.Function.String())
	if len(ce.Lifetimes) > 0 || len(ce.Generics) > 0 {
		sb.WriteString("<")
		parts := []string{}
		for _, lt := range ce.Lifetimes {
			parts = append(parts, "'"+lt)
		}
		for _, g := range ce.Generics {
			parts = append(parts, g.String())
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString(">")
	}
	sb.WriteString("(")
	args := []string{}
	for _, a := range ce.Args {
		args = append(args, a.String())
	}
	sb.WriteString(strings.Join(args, ", "))
	sb.WriteString(")")
	return sb.String()
}
func (ce *CallExpr) exprNode() {}

type IndexExpr struct {
	Position Position
	Target   Expr
	Index    Expr
}
func (ie *IndexExpr) Pos() Position { return ie.Position }
func (ie *IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", ie.Target.String(), ie.Index.String())
}
func (ie *IndexExpr) exprNode() {}

type SliceExpr struct {
	Position Position
	Target   Expr
	Low      Expr // optional (nil if omitted)
	High     Expr // optional (nil if omitted)
}
func (se *SliceExpr) Pos() Position { return se.Position }
func (se *SliceExpr) String() string {
	lowStr := ""
	if se.Low != nil {
		lowStr = se.Low.String()
	}
	highStr := ""
	if se.High != nil {
		highStr = se.High.String()
	}
	return fmt.Sprintf("%s[%s:%s]", se.Target.String(), lowStr, highStr)
}
func (se *SliceExpr) exprNode() {}

type SelectorExpr struct {
	Position Position
	Target   Expr
	Field    string
}
func (se *SelectorExpr) Pos() Position { return se.Position }
func (se *SelectorExpr) String() string {
	return se.Target.String() + "." + se.Field
}
func (se *SelectorExpr) exprNode() {}

type StructInitExpr struct {
	Position Position
	Struct   Type
	Fields   []StructInitField
}
func (si *StructInitExpr) Pos() Position { return si.Position }
func (si *StructInitExpr) String() string {
	var sb strings.Builder
	sb.WriteString(si.Struct.String() + "{\n")
	for _, f := range si.Fields {
		sb.WriteString(fmt.Sprintf("    %s: %s\n", f.Name, f.Value.String()))
	}
	sb.WriteString("}")
	return sb.String()
}
func (si *StructInitExpr) exprNode() {}

type StructInitField struct {
	Name  string
	Value Expr
}

type RefExpr struct {
	Position Position
	Mutable  bool
	Target   Expr
}
func (re *RefExpr) Pos() Position { return re.Position }
func (re *RefExpr) String() string {
	if re.Mutable {
		return "&mut " + re.Target.String()
	}
	return "&" + re.Target.String()
}
func (re *RefExpr) exprNode() {}

type DerefExpr struct {
	Position Position
	Target   Expr
}
func (de *DerefExpr) Pos() Position { return de.Position }
func (de *DerefExpr) String() string {
	return "*" + de.Target.String()
}
func (de *DerefExpr) exprNode() {}

type QuestionExpr struct {
	Position Position
	Target   Expr
}
func (qe *QuestionExpr) Pos() Position { return qe.Position }
func (qe *QuestionExpr) String() string { return qe.Target.String() + "?" }
func (qe *QuestionExpr) exprNode() {}

type MatchExpr struct {
	Position Position
	Target   Expr
	Cases    []MatchCase
}
func (me *MatchExpr) Pos() Position { return me.Position }
func (me *MatchExpr) String() string {
	var sb strings.Builder
	sb.WriteString("match " + me.Target.String() + " {\n")
	for _, c := range me.Cases {
		sb.WriteString("    " + c.Pattern.String() + " => " + c.Body.String() + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}
func (me *MatchExpr) exprNode() {}
func (me *MatchExpr) stmtNode() {}

type MatchCase struct {
	Pattern Expr
	Body    *BlockStmt
}

type ArrayLiteralExpr struct {
	Position Position
	Elements []Expr
}

func (ale *ArrayLiteralExpr) Pos() Position { return ale.Position }
func (ale *ArrayLiteralExpr) String() string {
	var parts []string
	for _, e := range ale.Elements {
		parts = append(parts, e.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
func (ale *ArrayLiteralExpr) exprNode() {}

// TypeCastExpr represents type casting: i32(7), f64(3.14), etc.
// Can also be used for struct initialization with type name
type TypeCastExpr struct {
	Position Position
	TargetType Type // The target type being cast to
	Value    Expr  // The value being cast
}
func (tc *TypeCastExpr) Pos() Position { return tc.Position }
func (tc *TypeCastExpr) String() string {
	return tc.TargetType.String() + "(" + tc.Value.String() + ")"
}
func (tc *TypeCastExpr) exprNode() {}

// TypeofExpr represents the typeof() operator: typeof(x)
// Returns the type of the expression
type TypeofExpr struct {
	Position Position
	Value    Expr // The expression to get the type of
}
func (te *TypeofExpr) Pos() Position { return te.Position }
func (te *TypeofExpr) String() string {
	return "typeof(" + te.Value.String() + ")"
}
func (te *TypeofExpr) exprNode() {}
