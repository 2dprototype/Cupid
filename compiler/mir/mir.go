package mir

import (
	"cupid/compiler/hir"
	"fmt"
	"strconv"
)

type OperandKind int

const (
	OpLocal OperandKind = iota
	OpConst
)

type Operand struct {
	Kind     OperandKind
	LocalID  int
	Constant string // literal representation
	Type     *hir.HIRType
}

func (op Operand) String() string {
	if op.Kind == OpLocal {
		return fmt.Sprintf("%%%d", op.LocalID)
	}
	return op.Constant
}

type Local struct {
	ID      int
	Name    string // optional debug name
	Type    *hir.HIRType
	IsParam bool
}

type Rvalue interface {
	rvalue()
	String() string
}

type UseRvalue struct {
	Op Operand
}

func (r *UseRvalue) rvalue()        {}
func (r *UseRvalue) String() string { return r.Op.String() }

type BinaryRvalue struct {
	Op    string
	Left  Operand
	Right Operand
	Type  *hir.HIRType
}

func (r *BinaryRvalue) rvalue() {}
func (r *BinaryRvalue) String() string {
	return fmt.Sprintf("%s %s %s", r.Left.String(), r.Op, r.Right.String())
}

type UnaryRvalue struct {
	Op    string
	Right Operand
	Type  *hir.HIRType
}

func (r *UnaryRvalue) rvalue() {}
func (r *UnaryRvalue) String() string {
	return fmt.Sprintf("%s%s", r.Op, r.Right.String())
}

type CallRvalue struct {
	FuncName string
	Args     []Operand
	Type     *hir.HIRType
}

func (r *CallRvalue) rvalue() {}
func (r *CallRvalue) String() string {
	argsStr := ""
	for i, a := range r.Args {
		if i > 0 {
			argsStr += ", "
		}
		argsStr += a.String()
	}
	return fmt.Sprintf("call %s(%s)", r.FuncName, argsStr)
}

type FieldAccessRvalue struct {
	Base   Operand
	Offset int
	Type   *hir.HIRType
}

func (r *FieldAccessRvalue) rvalue() {}
func (r *FieldAccessRvalue) String() string {
	return fmt.Sprintf("%s + offset(%d)", r.Base.String(), r.Offset)
}

type RefRvalue struct {
	Target Operand
	Type   *hir.HIRType
}

func (r *RefRvalue) rvalue() {}
func (r *RefRvalue) String() string {
	return fmt.Sprintf("&%s", r.Target.String())
}

type DerefRvalue struct {
	Target Operand
	Type   *hir.HIRType
}

func (r *DerefRvalue) rvalue() {}
func (r *DerefRvalue) String() string {
	return fmt.Sprintf("*%s", r.Target.String())
}

type Statement interface {
	statement()
	String() string
}

type AssignStmt struct {
	Dest Local
	Src  Rvalue
}

func (s *AssignStmt) statement() {}
func (s *AssignStmt) String() string {
	return fmt.Sprintf("  %%%d = %s", s.Dest.ID, s.Src.String())
}

type StoreStmt struct {
	Ptr Operand
	Val Operand
}

func (s *StoreStmt) statement() {}
func (s *StoreStmt) String() string {
	return fmt.Sprintf("  store %s -> [%s]", s.Val.String(), s.Ptr.String())
}

type SetFieldStmt struct {
	Base   Operand
	Offset int
	Val    Operand
}

func (s *SetFieldStmt) statement() {}
func (s *SetFieldStmt) String() string {
	return fmt.Sprintf("  set_field [%s + %d] = %s", s.Base.String(), s.Offset, s.Val.String())
}

type AsmStmt struct {
	Assembly string
}

func (s *AsmStmt) statement() {}
func (s *AsmStmt) String() string {
	return fmt.Sprintf("  asm: %s", s.Assembly)
}

type IndexRvalue struct {
	Base  Operand
	Index Operand
	Type  *hir.HIRType
}

func (r *IndexRvalue) rvalue() {}
func (r *IndexRvalue) String() string {
	return fmt.Sprintf("%s[%s]", r.Base.String(), r.Index.String())
}

type SetIndexStmt struct {
	Base  Operand
	Index Operand
	Val   Operand
}

func (s *SetIndexStmt) statement() {}
func (s *SetIndexStmt) String() string {
	return fmt.Sprintf("  set_index %s[%s] = %s", s.Base.String(), s.Index.String(), s.Val.String())
}

type Terminator interface {
	terminator()
	String() string
}

type ReturnTerminator struct {
	Value *Operand
}

func (t *ReturnTerminator) terminator() {}
func (t *ReturnTerminator) String() string {
	if t.Value != nil {
		return fmt.Sprintf("  return %s", t.Value.String())
	}
	return "  return"
}

type BranchTerminator struct {
	TargetBlock int
}

func (t *BranchTerminator) terminator() {}
func (t *BranchTerminator) String() string {
	return fmt.Sprintf("  goto bb%d", t.TargetBlock)
}

type CondBranchTerminator struct {
	Cond       Operand
	ThenBlock  int
	ElseBlock  int
}

func (t *CondBranchTerminator) terminator() {}
func (t *CondBranchTerminator) String() string {
	return fmt.Sprintf("  if %s goto bb%d else goto bb%d", t.Cond.String(), t.ThenBlock, t.ElseBlock)
}

type BasicBlock struct {
	ID         int
	Statements []Statement
	Terminator Terminator
}

func (bb *BasicBlock) String() string {
	out := fmt.Sprintf("bb%d:\n", bb.ID)
	for _, s := range bb.Statements {
		out += s.String() + "\n"
	}
	if bb.Terminator != nil {
		out += bb.Terminator.String() + "\n"
	}
	return out
}

type MIRFunction struct {
	Name        string
	Params      []Local
	ReturnType  *hir.HIRType
	Locals      []Local
	Blocks      []*BasicBlock
	nextLocalID int
	nextBlockID int
}

func (fn *MIRFunction) NewLocal(typ *hir.HIRType, name string) Local {
	l := Local{
		ID:   fn.nextLocalID,
		Name: name,
		Type: typ,
	}
	fn.nextLocalID++
	fn.Locals = append(fn.Locals, l)
	return l
}

func (fn *MIRFunction) NewBlock() *BasicBlock {
	b := &BasicBlock{
		ID:         fn.nextBlockID,
		Statements: make([]Statement, 0),
	}
	fn.nextBlockID++
	fn.Blocks = append(fn.Blocks, b)
	return b
}

type MIRProgram struct {
	Structs   map[string]*hir.HIRStruct
	Globals   []*hir.HIRGlobal
	Functions []*MIRFunction
}

// Lowering from HIR to MIR
type MIRLowerer struct {
	hirProg *hir.HIRProgram
}

func LowerHIR(hirProg *hir.HIRProgram) *MIRProgram {
	l := &MIRLowerer{hirProg: hirProg}
	return l.lower()
}

func (l *MIRLowerer) lower() *MIRProgram {
	prog := &MIRProgram{
		Structs:   l.hirProg.Structs,
		Globals:   l.hirProg.Globals,
		Functions: make([]*MIRFunction, 0, len(l.hirProg.Functions)),
	}

	for _, fn := range l.hirProg.Functions {
		mirFn := l.lowerFunction(fn)
		prog.Functions = append(prog.Functions, mirFn)
	}

	return prog
}

type fnContext struct {
	mirFn      *MIRFunction
	varMap     map[string]Local
	currBlock  *BasicBlock
	loopStarts []int
	loopEnds   []int
}

func (l *MIRLowerer) lowerFunction(fn *hir.HIRFunc) *MIRFunction {
	mirFn := &MIRFunction{
		Name:       fn.Name,
		Params:     make([]Local, 0, len(fn.Params)),
		ReturnType: fn.ReturnType,
		Locals:     make([]Local, 0),
		Blocks:     make([]*BasicBlock, 0),
	}

	ctx := &fnContext{
		mirFn:  mirFn,
		varMap: make(map[string]Local),
	}

	// Create param locals
	for _, p := range fn.Params {
		paramLocal := mirFn.NewLocal(p.Type, p.Name)
		paramLocal.IsParam = true
		mirFn.Params = append(mirFn.Params, paramLocal)
		ctx.varMap[p.Name] = paramLocal
	}

	entryBlock := mirFn.NewBlock()
	ctx.currBlock = entryBlock

	if fn.Body != nil {
		l.lowerBlock(fn.Body, ctx)
	}

	// Ensure entry block or current block has a terminator
	if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
		ctx.currBlock.Terminator = &ReturnTerminator{Value: nil}
	}

	return mirFn
}

func (l *MIRLowerer) lowerBlock(block *hir.HIRBlock, ctx *fnContext) {
	if block == nil {
		return
	}
	for _, s := range block.Stmts {
		l.lowerStmt(s, ctx)
	}
}

func (l *MIRLowerer) lowerStmt(stmt hir.HIRStmt, ctx *fnContext) {
	if ctx.currBlock == nil {
		return
	}

	switch s := stmt.(type) {
	case *hir.HIRLetStmt:
		op := l.lowerExpr(s.Value, ctx)
		loc := ctx.mirFn.NewLocal(s.Type, s.Name)
		ctx.varMap[s.Name] = loc
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: loc,
			Src:  &UseRvalue{Op: op},
		})
	case *hir.HIRAssignStmt:
		valOp := l.lowerExpr(s.Value, ctx)
		if v, ok := s.Target.(*hir.HIRVar); ok {
			if loc, exists := ctx.varMap[v.Name]; exists {
				ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
					Dest: loc,
					Src:  &UseRvalue{Op: valOp},
				})
			}
		} else if fa, ok := s.Target.(*hir.HIRFieldAccessExpr); ok {
			baseOp := l.lowerExpr(fa.Target, ctx)
			ctx.currBlock.Statements = append(ctx.currBlock.Statements, &SetFieldStmt{
				Base:   baseOp,
				Offset: fa.Offset,
				Val:    valOp,
			})
		} else if idx, ok := s.Target.(*hir.HIRIndexExpr); ok {
			baseOp := l.lowerExpr(idx.Target, ctx)
			idxOp := l.lowerExpr(idx.Index, ctx)
			ctx.currBlock.Statements = append(ctx.currBlock.Statements, &SetIndexStmt{
				Base:  baseOp,
				Index: idxOp,
				Val:   valOp,
			})
		}
	case *hir.HIRReturnStmt:
		var retOp *Operand
		if s.Value != nil {
			op := l.lowerExpr(s.Value, ctx)
			retOp = &op
		}
		ctx.currBlock.Terminator = &ReturnTerminator{Value: retOp}
		// Any subsequent statements in this block are unreachable
		ctx.currBlock = nil
	case *hir.HIRExprStmt:
		l.lowerExpr(s.Expr, ctx)
	case *hir.HIRIfStmt:
		condOp := l.lowerExpr(s.Condition, ctx)
		thenBlk := ctx.mirFn.NewBlock()
		elseBlk := ctx.mirFn.NewBlock()
		mergeBlk := ctx.mirFn.NewBlock()

		ctx.currBlock.Terminator = &CondBranchTerminator{
			Cond:      condOp,
			ThenBlock: thenBlk.ID,
			ElseBlock: elseBlk.ID,
		}

		// Then block
		ctx.currBlock = thenBlk
		l.lowerBlock(s.ThenBlock, ctx)
		if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
			ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: mergeBlk.ID}
		}

		// Else block
		ctx.currBlock = elseBlk
		if s.ElseBlock != nil {
			l.lowerBlock(s.ElseBlock, ctx)
		}
		if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
			ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: mergeBlk.ID}
		}

		ctx.currBlock = mergeBlk
	case *hir.HIRForStmt:
		// for i in 0..10 or for condition
		condBlk := ctx.mirFn.NewBlock()
		bodyBlk := ctx.mirFn.NewBlock()
		exitBlk := ctx.mirFn.NewBlock()

		// Initial branch to cond
		ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: condBlk.ID}

		ctx.loopStarts = append(ctx.loopStarts, condBlk.ID)
		ctx.loopEnds = append(ctx.loopEnds, exitBlk.ID)

		// Cond block
		ctx.currBlock = condBlk
		if s.End != nil {
			endOp := l.lowerExpr(s.End, ctx)
			condBlk.Terminator = &CondBranchTerminator{
				Cond:      endOp,
				ThenBlock: bodyBlk.ID,
				ElseBlock: exitBlk.ID,
			}
		} else {
			condBlk.Terminator = &BranchTerminator{TargetBlock: bodyBlk.ID}
		}

		// Body block
		ctx.currBlock = bodyBlk
		l.lowerBlock(s.Body, ctx)
		if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
			ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: condBlk.ID}
		}

		ctx.loopStarts = ctx.loopStarts[:len(ctx.loopStarts)-1]
		ctx.loopEnds = ctx.loopEnds[:len(ctx.loopEnds)-1]

		ctx.currBlock = exitBlk
	case *hir.HIRAsmStmt:
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AsmStmt{
			Assembly: s.Assembly,
		})
	case *hir.HIRMatchStmt:
		targetOp := l.lowerExpr(s.Target, ctx)
		mergeBlk := ctx.mirFn.NewBlock()

		for _, c := range s.Cases {
			caseBodyBlk := ctx.mirFn.NewBlock()
			nextCaseBlk := ctx.mirFn.NewBlock()

			if c.Pattern == nil {
				// Wildcard default case '_'
				ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: caseBodyBlk.ID}
			} else {
				patOp := l.lowerExpr(c.Pattern, ctx)
				condLoc := ctx.mirFn.NewLocal(&hir.HIRType{Kind: hir.TypeBool, Name: "bool"}, "tmp_match_cond")
				ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
					Dest: condLoc,
					Src: &BinaryRvalue{
						Op:    "==",
						Left:  targetOp,
						Right: patOp,
						Type:  &hir.HIRType{Kind: hir.TypeBool, Name: "bool"},
					},
				})
				ctx.currBlock.Terminator = &CondBranchTerminator{
					Cond:      Operand{Kind: OpLocal, LocalID: condLoc.ID, Type: condLoc.Type},
					ThenBlock: caseBodyBlk.ID,
					ElseBlock: nextCaseBlk.ID,
				}
			}

			ctx.currBlock = caseBodyBlk
			l.lowerBlock(c.Body, ctx)
			if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
				ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: mergeBlk.ID}
			}

			ctx.currBlock = nextCaseBlk
		}

		if ctx.currBlock != nil && ctx.currBlock.Terminator == nil {
			ctx.currBlock.Terminator = &BranchTerminator{TargetBlock: mergeBlk.ID}
		}
		ctx.currBlock = mergeBlk
	}
}

func (l *MIRLowerer) lowerExpr(expr hir.HIRExpr, ctx *fnContext) Operand {
	if expr == nil {
		return Operand{Kind: OpConst, Constant: "0", Type: &hir.HIRType{Kind: hir.TypeVoid}}
	}

	switch e := expr.(type) {
	case *hir.HIRLiteral:
		return Operand{
			Kind:     OpConst,
			Constant: e.Value,
			Type:     e.Typ,
		}
	case *hir.HIRVar:
		if loc, exists := ctx.varMap[e.Name]; exists {
			return Operand{
				Kind:    OpLocal,
				LocalID: loc.ID,
				Type:    loc.Type,
			}
		}
		// If it's a global constant or function reference
		return Operand{
			Kind:     OpConst,
			Constant: e.Name,
			Type:     e.Typ,
		}
	case *hir.HIRBinaryExpr:
		leftOp := l.lowerExpr(e.Left, ctx)
		rightOp := l.lowerExpr(e.Right, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_bin")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &BinaryRvalue{
				Op:    e.Op,
				Left:  leftOp,
				Right: rightOp,
				Type:  e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRUnaryExpr:
		rightOp := l.lowerExpr(e.Right, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_un")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &UnaryRvalue{
				Op:    e.Op,
				Right: rightOp,
				Type:  e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRCallExpr:
		args := make([]Operand, 0, len(e.Args))
		for _, a := range e.Args {
			args = append(args, l.lowerExpr(a, ctx))
		}
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_call")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &CallRvalue{
				FuncName: e.FuncName,
				Args:     args,
				Type:     e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRFieldAccessExpr:
		baseOp := l.lowerExpr(e.Target, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_field")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &FieldAccessRvalue{
				Base:   baseOp,
				Offset: e.Offset,
				Type:   e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRStructInitExpr:
		destLoc := ctx.mirFn.NewLocal(e.StructType, "tmp_struct")
		// Initialise struct fields
		if e.StructType != nil {
			for _, f := range e.StructType.Fields {
				if fVal, ok := e.Fields[f.Name]; ok {
					valOp := l.lowerExpr(fVal, ctx)
					ctx.currBlock.Statements = append(ctx.currBlock.Statements, &SetFieldStmt{
						Base:   Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type},
						Offset: f.Offset,
						Val:    valOp,
					})
				}
			}
		}
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRRefExpr:
		targetOp := l.lowerExpr(e.Target, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_ref")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &RefRvalue{
				Target: targetOp,
				Type:   e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRDerefExpr:
		targetOp := l.lowerExpr(e.Target, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_deref")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &DerefRvalue{
				Target: targetOp,
				Type:   e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRIndexExpr:
		baseOp := l.lowerExpr(e.Target, ctx)
		idxOp := l.lowerExpr(e.Index, ctx)
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_index")
		ctx.currBlock.Statements = append(ctx.currBlock.Statements, &AssignStmt{
			Dest: destLoc,
			Src: &IndexRvalue{
				Base:  baseOp,
				Index: idxOp,
				Type:  e.Typ,
			},
		})
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	case *hir.HIRArrayInitExpr:
		destLoc := ctx.mirFn.NewLocal(e.Typ, "tmp_array")
		for i, elem := range e.Elements {
			elemOp := l.lowerExpr(elem, ctx)
			ctx.currBlock.Statements = append(ctx.currBlock.Statements, &SetIndexStmt{
				Base:  Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type},
				Index: Operand{Kind: OpConst, Constant: strconv.Itoa(i), Type: &hir.HIRType{Kind: hir.TypeI64, Name: "i64"}},
				Val:   elemOp,
			})
		}
		return Operand{Kind: OpLocal, LocalID: destLoc.ID, Type: destLoc.Type}
	}

	return Operand{Kind: OpConst, Constant: "0", Type: &hir.HIRType{Kind: hir.TypeVoid}}
}
