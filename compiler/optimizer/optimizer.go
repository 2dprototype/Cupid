package optimizer

import (
	"cupid/compiler/hir"
	"cupid/compiler/mir"
	"strconv"
)

type Optimizer struct {
	prog    *mir.MIRProgram
	release bool
}

func Optimize(prog *mir.MIRProgram, release bool) *mir.MIRProgram {
	opt := &Optimizer{
		prog:    prog,
		release: release,
	}
	return opt.run()
}

func (opt *Optimizer) run() *mir.MIRProgram {
	for _, fn := range opt.prog.Functions {
		opt.optimizeFunction(fn)
	}
	return opt.prog
}

func (opt *Optimizer) optimizeFunction(fn *mir.MIRFunction) {
	// 1. Constant folding
	opt.foldConstants(fn)

	// 2. Dead block elimination
	opt.eliminateDeadBlocks(fn)
}

func (opt *Optimizer) foldConstants(fn *mir.MIRFunction) {
	for _, blk := range fn.Blocks {
		for _, stmt := range blk.Statements {
			if assign, ok := stmt.(*mir.AssignStmt); ok {
				if bin, ok := assign.Src.(*mir.BinaryRvalue); ok {
					if bin.Left.Kind == mir.OpConst && bin.Right.Kind == mir.OpConst {
						if folded, ok := opt.evalBinaryConst(bin.Op, bin.Left.Constant, bin.Right.Constant); ok {
							assign.Src = &mir.UseRvalue{
								Op: mir.Operand{
									Kind:     mir.OpConst,
									Constant: folded,
									Type:     assign.Dest.Type,
								},
							}
						}
					}
				} else if cast, ok := assign.Src.(*mir.CastRvalue); ok {
					if folded, ok := opt.evalConstCast(cast); ok {
						assign.Src = &mir.UseRvalue{
							Op: mir.Operand{
								Kind:     mir.OpConst,
								Constant: folded,
								Type:     assign.Dest.Type,
							},
						}
					}
				}
			}
		}
	}
}

// evalConstCast folds i64(7), u8(300), bool(0), etc. when the operand is
// already a literal integer constant, so obviously-constant casts don't
// cost an instruction at runtime. Float and string casts are left for
// codegen: they need real conversion/formatting instructions, not just
// text rewriting.
func (opt *Optimizer) evalConstCast(cast *mir.CastRvalue) (string, bool) {
	if cast.Value.Kind != mir.OpConst || cast.FromType == nil || cast.ToType == nil {
		return "", false
	}
	if !isConstFoldableIntKind(cast.FromType.Kind) || !isConstFoldableIntKind(cast.ToType.Kind) {
		return "", false
	}
	v, err := strconv.ParseInt(cast.Value.Constant, 10, 64)
	if err != nil {
		return "", false
	}
	return truncateToKind(v, cast.ToType.Kind), true
}

func isConstFoldableIntKind(k hir.HIRTypeKind) bool {
	switch k {
	case hir.TypeI8, hir.TypeI16, hir.TypeI32, hir.TypeI64,
		hir.TypeU8, hir.TypeU16, hir.TypeU32, hir.TypeU64, hir.TypeBool:
		return true
	}
	return false
}

// truncateToKind applies the same wraparound/sign-extension semantics as
// the backend's runtime cast instructions, but at compile time.
func truncateToKind(v int64, k hir.HIRTypeKind) string {
	switch k {
	case hir.TypeBool:
		if v != 0 {
			return "true"
		}
		return "false"
	case hir.TypeI8:
		return strconv.FormatInt(int64(int8(v)), 10)
	case hir.TypeI16:
		return strconv.FormatInt(int64(int16(v)), 10)
	case hir.TypeI32:
		return strconv.FormatInt(int64(int32(v)), 10)
	case hir.TypeU8:
		return strconv.FormatUint(uint64(uint8(v)), 10)
	case hir.TypeU16:
		return strconv.FormatUint(uint64(uint16(v)), 10)
	case hir.TypeU32:
		return strconv.FormatUint(uint64(uint32(v)), 10)
	case hir.TypeU64:
		return strconv.FormatUint(uint64(v), 10)
	default: // TypeI64
		return strconv.FormatInt(v, 10)
	}
}

func (opt *Optimizer) evalBinaryConst(op string, left string, right string) (string, bool) {
	lInt, errL := strconv.ParseInt(left, 10, 64)
	rInt, errR := strconv.ParseInt(right, 10, 64)

	if errL == nil && errR == nil {
		switch op {
		case "+":
			return strconv.FormatInt(lInt+rInt, 10), true
		case "-":
			return strconv.FormatInt(lInt-rInt, 10), true
		case "*":
			return strconv.FormatInt(lInt*rInt, 10), true
		case "/":
			if rInt != 0 {
				return strconv.FormatInt(lInt/rInt, 10), true
			}
		case "%":
			if rInt != 0 {
				return strconv.FormatInt(lInt%rInt, 10), true
			}
		case "==":
			if lInt == rInt {
				return "true", true
			}
			return "false", true
		case "!=":
			if lInt != rInt {
				return "true", true
			}
			return "false", true
		case "<":
			if lInt < rInt {
				return "true", true
			}
			return "false", true
		case "<=":
			if lInt <= rInt {
				return "true", true
			}
			return "false", true
		case ">":
			if lInt > rInt {
				return "true", true
			}
			return "false", true
		case ">=":
			if lInt >= rInt {
				return "true", true
			}
			return "false", true
		}
	}

	return "", false
}

func (opt *Optimizer) eliminateDeadBlocks(fn *mir.MIRFunction) {
	if len(fn.Blocks) == 0 {
		return
	}

	reachable := make(map[int]bool)
	queue := []int{fn.Blocks[0].ID}
	reachable[fn.Blocks[0].ID] = true

	blockMap := make(map[int]*mir.BasicBlock)
	for _, b := range fn.Blocks {
		blockMap[b.ID] = b
	}

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]

		b := blockMap[currID]
		if b == nil || b.Terminator == nil {
			continue
		}

		switch term := b.Terminator.(type) {
		case *mir.BranchTerminator:
			if !reachable[term.TargetBlock] {
				reachable[term.TargetBlock] = true
				queue = append(queue, term.TargetBlock)
			}
		case *mir.CondBranchTerminator:
			if !reachable[term.ThenBlock] {
				reachable[term.ThenBlock] = true
				queue = append(queue, term.ThenBlock)
			}
			if !reachable[term.ElseBlock] {
				reachable[term.ElseBlock] = true
				queue = append(queue, term.ElseBlock)
			}
		}
	}

	newBlocks := make([]*mir.BasicBlock, 0, len(fn.Blocks))
	for _, b := range fn.Blocks {
		if reachable[b.ID] {
			newBlocks = append(newBlocks, b)
		}
	}
	fn.Blocks = newBlocks
}