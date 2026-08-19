package backend

import (
	"bytes"
	"cupid/compiler/hir"
	"cupid/compiler/mir"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Backend struct {
	prog         *mir.MIRProgram
	fasmDir      string
	release      bool
	debug        bool
	strTable     map[string]string // string content -> label
	strCounter   int
	labelCounter int
}

func New(prog *mir.MIRProgram, fasmDir string, release bool, debug bool) *Backend {
	return &Backend{
		prog:     prog,
		fasmDir:  fasmDir,
		release:  release,
		debug:    debug,
		strTable: make(map[string]string),
	}
}

func GenerateAssembly(prog *mir.MIRProgram, fasmDir string, release bool, debug bool) (string, error) {
	b := New(prog, fasmDir, release, debug)
	return b.Generate()
}

func (b *Backend) Generate() (string, error) {
	var codeBuf bytes.Buffer
	var dataBuf bytes.Buffer

	// Pre-scan all string literals to build string table
	for _, fn := range b.prog.Functions {
		for _, blk := range fn.Blocks {
			for _, stmt := range blk.Statements {
				b.collectStringsFromStmt(stmt)
			}
		}
	}
	for _, g := range b.prog.Globals {
		if g.Value != nil && g.Value.Type() != nil && g.Value.Type().Kind == hir.TypeString {
			if lit, ok := g.Value.(*hir.HIRLiteral); ok {
				b.getStringLabel(lit.Value)
			}
		}
	}

	// Generate Code Section
	for _, fn := range b.prog.Functions {
		b.generateFunction(fn, &codeBuf)
	}

	// Generate Data Section
	dataBuf.WriteString("section '.data' data readable writeable\n")
	dataBuf.WriteString("    _cupid_stdout dq 0\n")
	dataBuf.WriteString("    _cupid_bytes_written dq 0\n")
	dataBuf.WriteString("    _cupid_crlf db 13, 10, 0\n")
	dataBuf.WriteString("    _cupid_true_str db 'true', 0\n")
	dataBuf.WriteString("    _cupid_false_str db 'false', 0\n")
	dataBuf.WriteString("    _cupid_fmt_i64 db '%lld', 0\n")
	dataBuf.WriteString("    _cupid_fmt_u64 db '%llu', 0\n")
	dataBuf.WriteString("    _cupid_fmt_f64 db '%f', 0\n")
	dataBuf.WriteString("    _cupid_fmt_str db '%s', 0\n")

	for strVal, label := range b.strTable {
		escaped := b.formatFasmString(strVal)
		dataBuf.WriteString(fmt.Sprintf("    %s db %s, 0\n", label, escaped))
		dataBuf.WriteString(fmt.Sprintf("    %s_len dq %d\n", label, len(strVal)))
	}

	// Globals
	for _, g := range b.prog.Globals {
		dataBuf.WriteString(fmt.Sprintf("    global_%s dq 0\n", g.Name))
	}

	// Assemble complete FASM source
	var out bytes.Buffer
	out.WriteString("format PE64 console\n")
	out.WriteString("entry start\n\n")

	// Include standard FASM import macro
	macroPath := strings.TrimRight(b.fasmDir, "\\/") + "\\INCLUDE\\macro\\import64.inc"
	out.WriteString(fmt.Sprintf("include '%s'\n\n", macroPath))

	out.WriteString("section '.text' code readable executable\n\n")

	// Entry Point
	out.WriteString("start:\n")
	out.WriteString("    sub rsp, 40\n")
	out.WriteString("    mov ecx, -11 ; STD_OUTPUT_HANDLE\n")
	out.WriteString("    call [GetStdHandle]\n")
	out.WriteString("    mov [_cupid_stdout], rax\n\n")
	out.WriteString("    call cupid_main\n")
	out.WriteString("    mov ecx, eax\n")
	out.WriteString("    call [ExitProcess]\n\n")

	// Emit built-in Cupid runtime helpers
	b.emitRuntimeHelpers(&out)

	// Emit user functions
	out.Write(codeBuf.Bytes())

	// Emit data section
	out.Write(dataBuf.Bytes())

	// Emit imports
	out.WriteString("\nsection '.idata' import data readable writeable\n")
	out.WriteString("library kernel32,'KERNEL32.DLL', \\\n")
	out.WriteString("        msvcrt,'MSVCRT.DLL'\n\n")
	out.WriteString("import kernel32, \\\n")
	out.WriteString("       ExitProcess,'ExitProcess', \\\n")
	out.WriteString("       GetStdHandle,'GetStdHandle', \\\n")
	out.WriteString("       WriteFile,'WriteFile', \\\n")
	out.WriteString("       ReadFile,'ReadFile', \\\n")
	out.WriteString("       GetProcessHeap,'GetProcessHeap', \\\n")
	out.WriteString("       HeapAlloc,'HeapAlloc', \\\n")
	out.WriteString("       HeapFree,'HeapFree', \\\n")
	out.WriteString("       Sleep,'Sleep'\n\n")
	out.WriteString("import msvcrt, \\\n")
	out.WriteString("       printf,'printf', \\\n")
	out.WriteString("       sprintf,'sprintf'\n")

	return out.String(), nil
}

func (b *Backend) collectStringsFromStmt(stmt mir.Statement) {
	if assign, ok := stmt.(*mir.AssignStmt); ok {
		if use, ok := assign.Src.(*mir.UseRvalue); ok && use.Op.Kind == mir.OpConst {
			if assign.Dest.Type != nil && assign.Dest.Type.Kind == hir.TypeString {
				b.getStringLabel(use.Op.Constant)
			}
		} else if call, ok := assign.Src.(*mir.CallRvalue); ok {
			for _, arg := range call.Args {
				if arg.Kind == mir.OpConst && arg.Type != nil && arg.Type.Kind == hir.TypeString {
					b.getStringLabel(arg.Constant)
				}
			}
		}
	}
}

func (b *Backend) getStringLabel(val string) string {
	cleanVal := strings.Trim(val, "\"")
	if label, exists := b.strTable[cleanVal]; exists {
		return label
	}
	b.strCounter++
	label := fmt.Sprintf("_cupid_str_%d", b.strCounter)
	b.strTable[cleanVal] = label
	return label
}

func (b *Backend) formatFasmString(s string) string {
	if s == "" {
		return "0"
	}
	parts := make([]string, 0)
	var current []rune
	for _, r := range s {
		if r >= 32 && r <= 126 && r != '\'' {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				parts = append(parts, fmt.Sprintf("'%s'", string(current)))
				current = nil
			}
			parts = append(parts, fmt.Sprintf("%d", int(r)))
		}
	}
	if len(current) > 0 {
		parts = append(parts, fmt.Sprintf("'%s'", string(current)))
	}
	return strings.Join(parts, ", ")
}

func (b *Backend) functionLabel(name string) string {
	if name == "main" {
		return "cupid_main"
	}
	return "cu_" + name
}

func (b *Backend) generateFunction(fn *mir.MIRFunction, out *bytes.Buffer) {
	label := b.functionLabel(fn.Name)

	out.WriteString(fmt.Sprintf("%s:\n", label))
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")

	// Calculate stack size for locals + 32 bytes shadow space
	offsets, totalLocalBytes := b.calculateLocalOffsets(fn)
	frameSize := totalLocalBytes + 48
	if frameSize%16 != 0 {
		frameSize += 16 - (frameSize % 16)
	}

	out.WriteString(fmt.Sprintf("    sub rsp, %d\n", frameSize))

	// Map params to stack slots
	// Register args in Win64: RCX, RDX, R8, R9
	paramRegs := []string{"rcx", "rdx", "r8", "r9"}
	for i, param := range fn.Params {
		offset := offsets[param.ID]
		if param.Type != nil && (param.Type.Kind == hir.TypeStruct || param.Type.Kind == hir.TypeArray) && param.Type.ByteSize() > 8 {
			if i < len(paramRegs) {
				reg := paramRegs[i]
				numWords := (param.Type.ByteSize() + 7) / 8
				for w := 0; w < numWords; w++ {
					out.WriteString(fmt.Sprintf("    mov rax, [%s + %d]\n", reg, w*8))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", offset, w*8))
				}
			}
		} else {
			if i < len(paramRegs) {
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], %s\n", offset, paramRegs[i]))
			} else {
				callerOffset := 16 + 32 + (i-4)*8
				out.WriteString(fmt.Sprintf("    mov rax, [rbp + %d]\n", callerOffset))
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", offset))
			}
		}
	}

	// Generate basic blocks
	for _, blk := range fn.Blocks {
		out.WriteString(fmt.Sprintf(".bb_%s_%d:\n", b.functionLabel(fn.Name), blk.ID))
		for _, stmt := range blk.Statements {
			b.generateStatement(stmt, fn, offsets, out)
		}
		if blk.Terminator != nil {
			b.generateTerminator(blk.Terminator, fn, offsets, out)
		}
	}

	out.WriteString(fmt.Sprintf(".epilogue_%s:\n", b.functionLabel(fn.Name)))
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")
}

func (b *Backend) calculateLocalOffsets(fn *mir.MIRFunction) (map[int]int, int) {
	offsets := make(map[int]int)
	currentOffset := 0
	for _, l := range fn.Locals {
		sz := 8
		if l.Type != nil && (l.Type.Kind == hir.TypeStruct || l.Type.Kind == hir.TypeArray) {
			sz = l.Type.ByteSize()
			if sz < 8 {
				sz = 8
			}
			if sz%8 != 0 {
				sz += 8 - (sz % 8)
			}
		}
		currentOffset += sz
		offsets[l.ID] = currentOffset
	}
	return offsets, currentOffset
}

func (b *Backend) generateStatement(stmt mir.Statement, fn *mir.MIRFunction, offsets map[int]int, out *bytes.Buffer) {
	switch s := stmt.(type) {
	case *mir.AssignStmt:
		destOffset := offsets[s.Dest.ID]
		switch src := s.Src.(type) {
		case *mir.UseRvalue:
			if s.Dest.Type != nil && (s.Dest.Type.Kind == hir.TypeStruct || s.Dest.Type.Kind == hir.TypeArray) && s.Dest.Type.ByteSize() > 8 && src.Op.Kind == mir.OpLocal {
				srcOffset := offsets[src.Op.LocalID]
				numWords := (s.Dest.Type.ByteSize() + 7) / 8
				for w := 0; w < numWords; w++ {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + %d]\n", srcOffset, w*8))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", destOffset, w*8))
				}
			} else {
				b.loadOperandToReg(src.Op, "rax", offsets, out)
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
			}
		case *mir.BinaryRvalue:
			b.generateBinary(src, destOffset, offsets, out)
		case *mir.UnaryRvalue:
			b.generateUnary(src, destOffset, offsets, out)
		case *mir.CallRvalue:
			b.generateCall(src, destOffset, offsets, out)
		case *mir.FieldAccessRvalue:
			if src.Base.Kind == mir.OpLocal && src.Base.Type != nil && src.Base.Type.Kind == hir.TypeStruct {
				baseOffset := offsets[src.Base.LocalID]
				out.WriteString(fmt.Sprintf("    mov rcx, [rbp - %d + %d]\n", baseOffset, src.Offset))
			} else {
				b.loadOperandToReg(src.Base, "rax", offsets, out)
				out.WriteString(fmt.Sprintf("    mov rcx, [rax + %d]\n", src.Offset))
			}
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rcx\n", destOffset))
		case *mir.IndexRvalue:
			b.loadOperandToReg(src.Index, "rcx", offsets, out)
			out.WriteString("    shl rcx, 3\n")
			if src.Base.Kind == mir.OpLocal {
				baseOffset := offsets[src.Base.LocalID]
				out.WriteString(fmt.Sprintf("    mov rax, rbp\n"))
				out.WriteString(fmt.Sprintf("    sub rax, %d\n", baseOffset))
				out.WriteString("    add rax, rcx\n")
				out.WriteString("    mov rdx, [rax]\n")
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rdx\n", destOffset))
			}
		case *mir.RefRvalue:
			if src.Target.Kind == mir.OpLocal {
				targetOffset := offsets[src.Target.LocalID]
				out.WriteString(fmt.Sprintf("    lea rax, [rbp - %d]\n", targetOffset))
			} else {
				b.loadOperandToReg(src.Target, "rax", offsets, out)
			}
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		case *mir.DerefRvalue:
			b.loadOperandToReg(src.Target, "rax", offsets, out)
			out.WriteString("    mov rcx, [rax]\n")
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rcx\n", destOffset))
		case *mir.CastRvalue:
			b.generateCast(src, destOffset, offsets, out)
		}
	case *mir.StoreStmt:
		b.loadOperandToReg(s.Ptr, "rax", offsets, out)
		b.loadOperandToReg(s.Val, "rcx", offsets, out)
		out.WriteString("    mov [rax], rcx\n")
	case *mir.SetFieldStmt:
		b.loadOperandToReg(s.Val, "rcx", offsets, out)
		if s.Base.Kind == mir.OpLocal && s.Base.Type != nil && s.Base.Type.Kind == hir.TypeStruct {
			baseOffset := offsets[s.Base.LocalID]
			out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rcx\n", baseOffset, s.Offset))
		} else {
			b.loadOperandToReg(s.Base, "rax", offsets, out)
			out.WriteString(fmt.Sprintf("    mov [rax + %d], rcx\n", s.Offset))
		}
	case *mir.SetIndexStmt:
		b.loadOperandToReg(s.Val, "rdx", offsets, out)
		b.loadOperandToReg(s.Index, "rcx", offsets, out)
		out.WriteString("    shl rcx, 3\n")
		if s.Base.Kind == mir.OpLocal {
			baseOffset := offsets[s.Base.LocalID]
			out.WriteString(fmt.Sprintf("    mov rax, rbp\n"))
			out.WriteString(fmt.Sprintf("    sub rax, %d\n", baseOffset))
			out.WriteString("    add rax, rcx\n")
			out.WriteString("    mov [rax], rdx\n")
		}
	case *mir.AsmStmt:
		out.WriteString(fmt.Sprintf("    %s\n", s.Assembly))
	}
}

func (b *Backend) generateBinary(bin *mir.BinaryRvalue, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	b.loadOperandToReg(bin.Left, "rax", offsets, out)
	b.loadOperandToReg(bin.Right, "rcx", offsets, out)

	switch bin.Op {
	case "+":
		out.WriteString("    add rax, rcx\n")
	case "-":
		out.WriteString("    sub rax, rcx\n")
	case "*":
		out.WriteString("    imul rax, rcx\n")
	case "/":
		out.WriteString("    cqo\n")
		out.WriteString("    idiv rcx\n")
	case "%":
		out.WriteString("    cqo\n")
		out.WriteString("    idiv rcx\n")
		out.WriteString("    mov rax, rdx\n")
	case "==", "!=", "<", "<=", ">", ">=":
		out.WriteString("    cmp rax, rcx\n")
		setInstr := ""
		switch bin.Op {
		case "==":
			setInstr = "sete"
		case "!=":
			setInstr = "setne"
		case "<":
			setInstr = "setl"
		case "<=":
			setInstr = "setle"
		case ">":
			setInstr = "setg"
		case ">=":
			setInstr = "setge"
		}
		out.WriteString(fmt.Sprintf("    %s al\n", setInstr))
		out.WriteString("    movzx rax, al\n")
	case "&":
		out.WriteString("    and rax, rcx\n")
	case "|":
		out.WriteString("    or rax, rcx\n")
	case "^":
		out.WriteString("    xor rax, rcx\n")
	case "<<":
		out.WriteString("    shl rax, cl\n")
	case ">>":
		out.WriteString("    shr rax, cl\n")
	case "&&":
		out.WriteString("    and rax, rcx\n")
	case "||":
		out.WriteString("    or rax, rcx\n")
	}

	out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
}

func (b *Backend) generateUnary(un *mir.UnaryRvalue, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	b.loadOperandToReg(un.Right, "rax", offsets, out)
	switch un.Op {
	case "-":
		out.WriteString("    neg rax\n")
	case "!":
		out.WriteString("    cmp rax, 0\n")
		out.WriteString("    sete al\n")
		out.WriteString("    movzx rax, al\n")
	}
	out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
}

func (b *Backend) generateCall(call *mir.CallRvalue, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	// Handle built-ins: print, println, len
	if call.FuncName == "len" {
		if len(call.Args) > 0 && call.Args[0].Type != nil && call.Args[0].Type.Kind == hir.TypeArray {
			out.WriteString(fmt.Sprintf("    mov rax, %d\n", call.Args[0].Type.Size))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
			return
		}
	}

	if call.FuncName == "print" || call.FuncName == "println" {
		if len(call.Args) > 0 {
			arg := call.Args[0]
			if arg.Type != nil && arg.Type.Kind == hir.TypeString {
				b.loadOperandToReg(arg, "rcx", offsets, out)
				out.WriteString("    call _cupid_print_str\n")
			} else if arg.Type != nil && arg.Type.Kind == hir.TypeBool {
				b.loadOperandToReg(arg, "rcx", offsets, out)
				out.WriteString("    call _cupid_print_bool\n")
			} else {
				b.loadOperandToReg(arg, "rcx", offsets, out)
				out.WriteString("    call _cupid_print_i64\n")
			}
		}
		if call.FuncName == "println" {
			out.WriteString("    call _cupid_println\n")
		}
		return
	}

	// Standard function call ABI
	paramRegs := []string{"rcx", "rdx", "r8", "r9"}
	for i, arg := range call.Args {
		if i < len(paramRegs) {
			if arg.Type != nil && (arg.Type.Kind == hir.TypeStruct || arg.Type.Kind == hir.TypeArray) && arg.Type.ByteSize() > 8 && arg.Kind == mir.OpLocal {
				argOffset := offsets[arg.LocalID]
				out.WriteString(fmt.Sprintf("    lea %s, [rbp - %d]\n", paramRegs[i], argOffset))
			} else {
				b.loadOperandToReg(arg, paramRegs[i], offsets, out)
			}
		} else {
			stackOffset := 32 + (i-4)*8
			b.loadOperandToReg(arg, "rax", offsets, out)
			out.WriteString(fmt.Sprintf("    mov [rsp + %d], rax\n", stackOffset))
		}
	}

	targetName := b.functionLabel(call.FuncName)
	out.WriteString(fmt.Sprintf("    call %s\n", targetName))
	if call.Type != nil && (call.Type.Kind == hir.TypeStruct || call.Type.Kind == hir.TypeArray) && call.Type.ByteSize() > 8 {
		numWords := (call.Type.ByteSize() + 7) / 8
		for w := 0; w < numWords; w++ {
			out.WriteString(fmt.Sprintf("    mov rcx, [rax + %d]\n", w*8))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rcx\n", destOffset, w*8))
		}
	} else {
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
	}
}

func (b *Backend) generateTerminator(term mir.Terminator, fn *mir.MIRFunction, offsets map[int]int, out *bytes.Buffer) {
	fnLabel := b.functionLabel(fn.Name)
	switch t := term.(type) {
	case *mir.ReturnTerminator:
		if t.Value != nil {
			if fn.ReturnType != nil && (fn.ReturnType.Kind == hir.TypeStruct || fn.ReturnType.Kind == hir.TypeArray) && fn.ReturnType.ByteSize() > 8 && t.Value.Kind == mir.OpLocal {
				valOffset := offsets[t.Value.LocalID]
				out.WriteString(fmt.Sprintf("    lea rax, [rbp - %d]\n", valOffset))
			} else {
				b.loadOperandToReg(*t.Value, "rax", offsets, out)
			}
		}
		out.WriteString(fmt.Sprintf("    jmp .epilogue_%s\n", fnLabel))
	case *mir.BranchTerminator:
		out.WriteString(fmt.Sprintf("    jmp .bb_%s_%d\n", fnLabel, t.TargetBlock))
	case *mir.CondBranchTerminator:
		b.loadOperandToReg(t.Cond, "rax", offsets, out)
		out.WriteString("    cmp rax, 0\n")
		out.WriteString(fmt.Sprintf("    jne .bb_%s_%d\n", fnLabel, t.ThenBlock))
		out.WriteString(fmt.Sprintf("    jmp .bb_%s_%d\n", fnLabel, t.ElseBlock))
	}
}

func (b *Backend) loadOperandToReg(op mir.Operand, reg string, offsets map[int]int, out *bytes.Buffer) {
	if op.Kind == mir.OpLocal {
		offset := offsets[op.LocalID]
		out.WriteString(fmt.Sprintf("    mov %s, [rbp - %d]\n", reg, offset))
	} else {
		// Constant
		val := op.Constant
		if op.Type != nil && op.Type.Kind == hir.TypeString {
			cleanVal := strings.Trim(val, "\"")
			label := b.getStringLabel(cleanVal)
			out.WriteString(fmt.Sprintf("    lea %s, [%s]\n", reg, label))
		} else if op.Type != nil && op.Type.Kind == hir.TypeChar {
			// Char literals carry their raw source text (e.g. "a" or "\n"),
			// not a decimal number - decode to a code point immediate.
			out.WriteString(fmt.Sprintf("    mov %s, %d\n", reg, charLiteralCodePoint(val)))
		} else if val == "true" {
			out.WriteString(fmt.Sprintf("    mov %s, 1\n", reg))
		} else if val == "false" {
			out.WriteString(fmt.Sprintf("    mov %s, 0\n", reg))
		} else {
			if _, err := strconv.ParseInt(val, 10, 64); err == nil {
				out.WriteString(fmt.Sprintf("    mov %s, %s\n", reg, val))
			} else {
				// Global or symbol reference
				out.WriteString(fmt.Sprintf("    mov %s, [global_%s]\n", reg, val))
			}
		}
	}
}

// charLiteralCodePoint decodes a lexer char-literal's raw text (e.g. "a",
// "\n", "\\") into its Unicode code point, matching lexer.readCharLiteral.
func charLiteralCodePoint(raw string) int64 {
	if raw == "" {
		return 0
	}
	if raw[0] == '\\' && len(raw) >= 2 {
		switch raw[1] {
		case 'n':
			return int64('\n')
		case 't':
			return int64('\t')
		case 'r':
			return int64('\r')
		case '0':
			return 0
		case '\\':
			return int64('\\')
		case '\'':
			return int64('\'')
		case '"':
			return int64('"')
		}
	}
	r, _ := utf8.DecodeRuneInString(raw)
	return int64(r)
}

// ---------------- Type cast codegen ----------------
//
// Every scalar local occupies a fixed 8-byte stack slot (see
// calculateLocalOffsets), and the invariant maintained everywhere in this
// backend is that a slot always holds its value sign/zero-extended to fill
// all 64 bits per the value's own type (this is how bool, already stored
// as a clean 0/1, and every arithmetic op that reads a full "mov rax,
// [rbp-N]" already behave). Casts must preserve that invariant.
//
// f32/f64 locals reuse the same 8-byte slots but hold IEEE-754 bit
// patterns instead (low 32 bits for f32, all 64 for f64), moved in/out via
// SSE2 register instructions instead of GPR ones.

func isFloatKind(k hir.HIRTypeKind) bool {
	return k == hir.TypeF32 || k == hir.TypeF64
}

func isIntKind(k hir.HIRTypeKind) bool {
	switch k {
	case hir.TypeI8, hir.TypeI16, hir.TypeI32, hir.TypeI64,
		hir.TypeU8, hir.TypeU16, hir.TypeU32, hir.TypeU64,
		hir.TypeBool, hir.TypeChar:
		return true
	}
	return false
}

func isUnsignedKind(k hir.HIRTypeKind) bool {
	switch k {
	case hir.TypeU8, hir.TypeU16, hir.TypeU32, hir.TypeU64:
		return true
	}
	return false
}

// truncateAndExtendRax narrows rax to `kind`'s width using the matching
// x86 sub-register (al/ax/eax) and re-extends to 64 bits with the correct
// signedness, in a single instruction. Because al/ax/eax always alias the
// low bits of rax regardless of what was there before, this single step
// correctly implements truncate-then-extend for ANY source width, so the
// source kind never needs to be consulted here - the target kind alone
// determines the instruction.
func (b *Backend) truncateAndExtendRax(kind hir.HIRTypeKind, out *bytes.Buffer) {
	switch kind {
	case hir.TypeI8:
		out.WriteString("    movsx rax, al\n")
	case hir.TypeI16:
		out.WriteString("    movsx rax, ax\n")
	case hir.TypeI32:
		out.WriteString("    movsxd rax, eax\n")
	case hir.TypeU8:
		out.WriteString("    movzx rax, al\n")
	case hir.TypeU16:
		out.WriteString("    movzx rax, ax\n")
	case hir.TypeU32, hir.TypeChar:
		out.WriteString("    mov eax, eax\n") // writing eax zero-extends to rax
	// TypeI64 / TypeU64: already the full 64 bits, nothing to do.
	default:
	}
}

// finalizeIntInRax stores a raw integer result in rax into `toKind`'s
// representation. bool needs true 0-vs-nonzero normalization (a truncating
// mov would wrongly turn 256 into false); every other integer/char kind
// just needs truncateAndExtendRax.
func (b *Backend) finalizeIntInRax(toKind hir.HIRTypeKind, out *bytes.Buffer) {
	if toKind == hir.TypeBool {
		out.WriteString("    cmp rax, 0\n")
		out.WriteString("    setne al\n")
		out.WriteString("    movzx rax, al\n")
		return
	}
	b.truncateAndExtendRax(toKind, out)
}

// loadOperandToXMM loads a float-typed operand into an XMM register.
// Constants are materialized without touching the data section at all:
// their IEEE-754 bit pattern is moved as a GPR immediate and then punned
// into the XMM register with movd/movq, since FASM has no "mov xmm, imm".
func (b *Backend) loadOperandToXMM(op mir.Operand, xmmReg string, kind hir.HIRTypeKind, offsets map[int]int, out *bytes.Buffer) {
	if op.Kind == mir.OpLocal {
		offset := offsets[op.LocalID]
		if kind == hir.TypeF32 {
			out.WriteString(fmt.Sprintf("    movd %s, [rbp - %d]\n", xmmReg, offset))
		} else {
			out.WriteString(fmt.Sprintf("    movsd %s, [rbp - %d]\n", xmmReg, offset))
		}
		return
	}

	f, err := strconv.ParseFloat(op.Constant, 64)
	if err != nil {
		f = 0
	}
	if kind == hir.TypeF32 {
		bits := math.Float32bits(float32(f))
		out.WriteString(fmt.Sprintf("    mov eax, %d\n", bits))
		out.WriteString(fmt.Sprintf("    movd %s, eax\n", xmmReg))
	} else {
		bits := math.Float64bits(f)
		out.WriteString(fmt.Sprintf("    mov rax, %d\n", bits))
		out.WriteString(fmt.Sprintf("    movq %s, rax\n", xmmReg))
	}
}

// storeXMMToLocal spills an XMM register into a local's stack slot.
func (b *Backend) storeXMMToLocal(xmmReg string, kind hir.HIRTypeKind, destOffset int, out *bytes.Buffer) {
	if kind == hir.TypeF32 {
		out.WriteString(fmt.Sprintf("    movd [rbp - %d], %s\n", destOffset, xmmReg))
	} else {
		out.WriteString(fmt.Sprintf("    movsd [rbp - %d], %s\n", destOffset, xmmReg))
	}
}

// generateCast emits code for a TypeCastExpr: typename(value).
func (b *Backend) generateCast(c *mir.CastRvalue, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	fromKind := hir.TypeI64
	if c.FromType != nil {
		fromKind = c.FromType.Kind
	}
	toKind := hir.TypeI64
	if c.ToType != nil {
		toKind = c.ToType.Kind
	}

	if toKind == hir.TypeString {
		b.generateCastToString(c, fromKind, destOffset, offsets, out)
		return
	}
	if fromKind == hir.TypeString {
		// The type checker only allows string -> string here; identity copy.
		b.loadOperandToReg(c.Value, "rax", offsets, out)
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		return
	}

	switch {
	case isFloatKind(fromKind) && isFloatKind(toKind):
		b.loadOperandToXMM(c.Value, "xmm0", fromKind, offsets, out)
		if fromKind == hir.TypeF64 && toKind == hir.TypeF32 {
			out.WriteString("    cvtsd2ss xmm0, xmm0\n")
		} else if fromKind == hir.TypeF32 && toKind == hir.TypeF64 {
			out.WriteString("    cvtss2sd xmm0, xmm0\n")
		}
		b.storeXMMToLocal("xmm0", toKind, destOffset, out)

	case isFloatKind(fromKind) && isIntKind(toKind):
		b.loadOperandToXMM(c.Value, "xmm0", fromKind, offsets, out)
		if fromKind == hir.TypeF32 {
			out.WriteString("    cvttss2si rax, xmm0\n") // truncates toward zero
		} else {
			out.WriteString("    cvttsd2si rax, xmm0\n")
		}
		b.finalizeIntInRax(toKind, out)
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))

	case isIntKind(fromKind) && isFloatKind(toKind):
		b.loadOperandToReg(c.Value, "rax", offsets, out)
		b.truncateAndExtendRax(fromKind, out)
		if toKind == hir.TypeF32 {
			out.WriteString("    cvtsi2ss xmm0, rax\n")
		} else {
			out.WriteString("    cvtsi2sd xmm0, rax\n")
		}
		b.storeXMMToLocal("xmm0", toKind, destOffset, out)

	default: // int/bool/char <-> int/bool/char
		b.loadOperandToReg(c.Value, "rax", offsets, out)
		b.finalizeIntInRax(toKind, out)
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
	}
}

// generateCastToString formats any scalar into a heap-allocated,
// null-terminated string (giving it the same representation as any other
// TypeString value elsewhere in the backend: a pointer in a stack slot).
func (b *Backend) generateCastToString(c *mir.CastRvalue, fromKind hir.HIRTypeKind, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	switch {
	case fromKind == hir.TypeString:
		b.loadOperandToReg(c.Value, "rax", offsets, out)
	case fromKind == hir.TypeBool:
		b.loadOperandToReg(c.Value, "rcx", offsets, out)
		out.WriteString("    call _cupid_bool_to_str\n")
	case fromKind == hir.TypeChar:
		b.loadOperandToReg(c.Value, "rcx", offsets, out)
		out.WriteString("    call _cupid_char_to_str\n")
	case isFloatKind(fromKind):
		b.loadOperandToXMM(c.Value, "xmm0", fromKind, offsets, out)
		if fromKind == hir.TypeF32 {
			out.WriteString("    cvtss2sd xmm0, xmm0\n")
		}
		out.WriteString("    call _cupid_f64_to_str\n")
	default: // remaining integer kinds
		b.loadOperandToReg(c.Value, "rax", offsets, out)
		b.truncateAndExtendRax(fromKind, out)
		out.WriteString("    mov rcx, rax\n")
		if isUnsignedKind(fromKind) {
			out.WriteString("    call _cupid_u64_to_str\n")
		} else {
			out.WriteString("    call _cupid_i64_to_str\n")
		}
	}
	out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
}

func (b *Backend) emitRuntimeHelpers(out *bytes.Buffer) {
	// _cupid_print_str: RCX contains null-terminated string pointer
	out.WriteString("_cupid_print_str:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov rdx, rcx\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_print_i64: RCX contains 64-bit integer
	out.WriteString("_cupid_print_i64:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov rdx, rcx\n")
	out.WriteString("    lea rcx, [_cupid_fmt_i64]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_print_bool: RCX contains 0 or 1
	out.WriteString("_cupid_print_bool:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .print_false\n")
	out.WriteString("    lea rdx, [_cupid_true_str]\n")
	out.WriteString("    jmp .do_print_bool\n")
	out.WriteString(".print_false:\n")
	out.WriteString("    lea rdx, [_cupid_false_str]\n")
	out.WriteString(".do_print_bool:\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_println: prints \r\n
	out.WriteString("_cupid_println:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    lea rdx, [_cupid_crlf]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_alloc: RCX contains number of bytes
	out.WriteString("_cupid_alloc:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov r8, rcx ; bytes\n")
	out.WriteString("    call [GetProcessHeap]\n")
	out.WriteString("    mov rcx, rax ; hHeap\n")
	out.WriteString("    mov edx, 8   ; HEAP_ZERO_MEMORY\n")
	out.WriteString("    call [HeapAlloc]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_free: RCX contains memory pointer
	out.WriteString("_cupid_free:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov r8, rcx ; lpMem\n")
	out.WriteString("    call [GetProcessHeap]\n")
	out.WriteString("    mov rcx, rax ; hHeap\n")
	out.WriteString("    xor edx, edx ; dwFlags\n")
	out.WriteString("    call [HeapFree]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// ---- Helpers backing typename(x) -> string casts ----

	// _cupid_bool_to_str: RCX = 0 or 1 -> RAX = pointer to "false"/"true".
	// No allocation needed: these are the same static strings print() uses.
	out.WriteString("_cupid_bool_to_str:\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .cbts_false\n")
	out.WriteString("    lea rax, [_cupid_true_str]\n")
	out.WriteString("    ret\n")
	out.WriteString(".cbts_false:\n")
	out.WriteString("    lea rax, [_cupid_false_str]\n")
	out.WriteString("    ret\n\n")

	// _cupid_char_to_str: RCX = code point -> RAX = pointer to a
	// freshly-allocated 2-byte "<char>\0" buffer.
	out.WriteString("_cupid_char_to_str:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov rcx, 2\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov rcx, [rbp - 8]\n")
	out.WriteString("    mov byte [rax], cl\n")
	out.WriteString("    mov byte [rax + 1], 0\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_i64_to_str: RCX = signed 64-bit value -> RAX = formatted buffer.
	out.WriteString("_cupid_i64_to_str:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov rcx, 24\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 16], rax\n")
	out.WriteString("    mov rcx, rax\n")
	out.WriteString("    lea rdx, [_cupid_fmt_i64]\n")
	out.WriteString("    mov r8, [rbp - 8]\n")
	out.WriteString("    call [sprintf]\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_u64_to_str: RCX = unsigned 64-bit value -> RAX = formatted buffer.
	out.WriteString("_cupid_u64_to_str:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov rcx, 24\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 16], rax\n")
	out.WriteString("    mov rcx, rax\n")
	out.WriteString("    lea rdx, [_cupid_fmt_u64]\n")
	out.WriteString("    mov r8, [rbp - 8]\n")
	out.WriteString("    call [sprintf]\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_f64_to_str: XMM0 = double value -> RAX = formatted buffer.
	// sprintf is variadic, and the Windows x64 ABI requires variadic
	// floating-point args to be duplicated into the positional GP register
	// (R8 here, since buf/fmt already occupy RCX/RDX) as well as XMM2, in
	// case the callee reads va_args through the integer registers.
	out.WriteString("_cupid_f64_to_str:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    movsd [rbp - 8], xmm0\n")
	out.WriteString("    mov rcx, 32\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 16], rax\n")
	out.WriteString("    mov rcx, rax\n")
	out.WriteString("    lea rdx, [_cupid_fmt_f64]\n")
	out.WriteString("    movsd xmm2, [rbp - 8]\n")
	out.WriteString("    movq r8, xmm2\n")
	out.WriteString("    call [sprintf]\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")
}