package backend

import (
	"bytes"
	"cupid/compiler/hir"
	"cupid/compiler/mir"
	"fmt"
	"strconv"
	"strings"
)

type Backend struct {
	prog       *mir.MIRProgram
	fasmDir    string
	release    bool
	debug      bool
	strTable   map[string]string // string content -> label
	strCounter int
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
	out.WriteString("       printf,'printf'\n")

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
}
