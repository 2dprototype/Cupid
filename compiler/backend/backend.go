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
	dataBuf.WriteString("    _cupid_fmt_str_len db '%.*s', 0\n")
	dataBuf.WriteString("    _cupid_fmt_bounds_panic db 'panic: string index out of bounds: index %lld, len %lld', 13, 10, 0\n")
	dataBuf.WriteString("    _cupid_lbracket db '[', 0\n")
	dataBuf.WriteString("    _cupid_rbracket db ']', 0\n")
	dataBuf.WriteString("    _cupid_lparen db '(', 0\n")
	dataBuf.WriteString("    _cupid_rparen db ')', 0\n")
	dataBuf.WriteString("    _cupid_lbrace db ' { ', 0\n")
	dataBuf.WriteString("    _cupid_rbrace db ' }', 0\n")
	dataBuf.WriteString("    _cupid_colon_space db ': ', 0\n")
	dataBuf.WriteString("    _cupid_ampersand db '&', 0\n")
	dataBuf.WriteString("    _cupid_comma_space db ', ', 0\n")
	dataBuf.WriteString("    _cupid_single_quote db 39, 0\n")
	dataBuf.WriteString("    _cupid_double_quote db 34, 0\n")

	for strVal, label := range b.strTable {
		escaped := b.formatFasmString(strVal)
		dataBuf.WriteString(fmt.Sprintf("    %s db %s, 0\n", label, escaped))
		dataBuf.WriteString(fmt.Sprintf("    %s_len dq %d\n", label, len(strVal)))
		dataBuf.WriteString(fmt.Sprintf("    %s_hdr dq %s, %d\n", label, label, len(strVal)))
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
	b.emitGlobalInitializers(&out)
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
	out.WriteString("        user32,'USER32.DLL', \\\n")
	out.WriteString("        msvcrt,'MSVCRT.DLL'\n\n")
	out.WriteString("import kernel32, \\\n")
	out.WriteString("       ExitProcess,'ExitProcess', \\\n")
	out.WriteString("       GetStdHandle,'GetStdHandle', \\\n")
	out.WriteString("       WriteFile,'WriteFile', \\\n")
	out.WriteString("       ReadFile,'ReadFile', \\\n")
	out.WriteString("       CreateFileA,'CreateFileA', \\\n")
	out.WriteString("       GetFileSizeEx,'GetFileSizeEx', \\\n")
	out.WriteString("       DeleteFileA,'DeleteFileA', \\\n")
	out.WriteString("       GetFileAttributesA,'GetFileAttributesA', \\\n")
	out.WriteString("       GetTickCount64,'GetTickCount64', \\\n")
	out.WriteString("       Beep,'Beep', \\\n")
	out.WriteString("       GetProcessHeap,'GetProcessHeap', \\\n")
	out.WriteString("       HeapAlloc,'HeapAlloc', \\\n")
	out.WriteString("       HeapFree,'HeapFree', \\\n")
	out.WriteString("       Sleep,'Sleep', \\\n")
	out.WriteString("       CreateThread,'CreateThread', \\\n")
	out.WriteString("       WaitForSingleObject,'WaitForSingleObject', \\\n")
	out.WriteString("       CreateSemaphoreA,'CreateSemaphoreA', \\\n")
	out.WriteString("       ReleaseSemaphore,'ReleaseSemaphore', \\\n")
	out.WriteString("       CreateMutexA,'CreateMutexA', \\\n")
	out.WriteString("       ReleaseMutex,'ReleaseMutex', \\\n")
	out.WriteString("       CloseHandle,'CloseHandle'\n\n")
	out.WriteString("import user32, \\\n")
	out.WriteString("       MessageBoxA,'MessageBoxA', \\\n")
	out.WriteString("       GetSystemMetrics,'GetSystemMetrics'\n\n")
	out.WriteString("import msvcrt, \\\n")
	out.WriteString("       printf,'printf', \\\n")
	out.WriteString("       sprintf,'sprintf'\n")

	return out.String(), nil
}

func (b *Backend) collectStringsFromOp(op mir.Operand) {
	if op.Kind == mir.OpConst && op.Type != nil && op.Type.Kind == hir.TypeString {
		b.getStringLabel(op.Constant)
	}
}

func (b *Backend) collectStringsFromStmt(stmt mir.Statement) {
	switch s := stmt.(type) {
	case *mir.AssignStmt:
		switch src := s.Src.(type) {
		case *mir.UseRvalue:
			b.collectStringsFromOp(src.Op)
		case *mir.BinaryRvalue:
			b.collectStringsFromOp(src.Left)
			b.collectStringsFromOp(src.Right)
		case *mir.UnaryRvalue:
			b.collectStringsFromOp(src.Right)
		case *mir.CallRvalue:
			for _, a := range src.Args {
				b.collectStringsFromOp(a)
			}
		case *mir.IndexRvalue:
			b.collectStringsFromOp(src.Base)
			b.collectStringsFromOp(src.Index)
		case *mir.SliceRvalue:
			b.collectStringsFromOp(src.Base)
			if src.Low != nil {
				b.collectStringsFromOp(*src.Low)
			}
			if src.High != nil {
				b.collectStringsFromOp(*src.High)
			}
		case *mir.CastRvalue:
			b.collectStringsFromOp(src.Value)
		}
	case *mir.SetFieldStmt:
		b.collectStringsFromOp(s.Base)
		b.collectStringsFromOp(s.Val)
	case *mir.SetIndexStmt:
		b.collectStringsFromOp(s.Base)
		b.collectStringsFromOp(s.Index)
		b.collectStringsFromOp(s.Val)
	case *mir.StoreStmt:
		b.collectStringsFromOp(s.Ptr)
		b.collectStringsFromOp(s.Val)
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
	if strings.HasPrefix(name, "_cupid_") {
		return name
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
		if param.Type != nil && param.Type.ByteSize() > 8 {
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
		if l.Type != nil {
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
			if s.Dest.Type != nil && s.Dest.Type.ByteSize() > 8 {
				if src.Op.Kind == mir.OpLocal {
					srcOffset := offsets[src.Op.LocalID]
					numWords := (s.Dest.Type.ByteSize() + 7) / 8
					for w := 0; w < numWords; w++ {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + %d]\n", srcOffset, w*8))
						out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", destOffset, w*8))
					}
				} else if src.Op.Kind == mir.OpConst && s.Dest.Type.Kind == hir.TypeString {
					cleanVal := strings.Trim(src.Op.Constant, "\"")
					label := b.getStringLabel(cleanVal)
					out.WriteString(fmt.Sprintf("    lea rax, [%s]\n", label))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
					out.WriteString(fmt.Sprintf("    mov qword [rbp - %d + 8], %d\n", destOffset, len(cleanVal)))
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
			if src.Type != nil && src.Type.ByteSize() > 8 {
				numWords := (src.Type.ByteSize() + 7) / 8
				if src.Base.Kind == mir.OpLocal && src.Base.Type != nil && (src.Base.Type.Kind == hir.TypeStruct || src.Base.Type.Kind == hir.TypeTuple) {
					baseOffset := offsets[src.Base.LocalID]
					for w := 0; w < numWords; w++ {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + %d]\n", baseOffset, src.Offset+w*8))
						out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", destOffset, w*8))
					}
				} else {
					b.loadOperandToReg(src.Base, "rax", offsets, out)
					for w := 0; w < numWords; w++ {
						out.WriteString(fmt.Sprintf("    mov rcx, [rax + %d]\n", src.Offset+w*8))
						out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rcx\n", destOffset, w*8))
					}
				}
				break
			}
			fKind := hir.TypeI64
			if src.Type != nil {
				fKind = src.Type.Kind
			}
			addr := ""
			if src.Base.Kind == mir.OpLocal && src.Base.Type != nil && (src.Base.Type.Kind == hir.TypeStruct || src.Base.Type.Kind == hir.TypeTuple) {
				baseOffset := offsets[src.Base.LocalID]
				addr = fmt.Sprintf("[rbp - %d + %d]", baseOffset, src.Offset)
			} else {
				b.loadOperandToReg(src.Base, "rax", offsets, out)
				addr = fmt.Sprintf("[rax + %d]", src.Offset)
			}

			switch fKind {
			case hir.TypeBool, hir.TypeU8:
				out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
			case hir.TypeI8:
				out.WriteString(fmt.Sprintf("    movsx rcx, byte %s\n", addr))
			case hir.TypeU16:
				out.WriteString(fmt.Sprintf("    movzx rcx, word %s\n", addr))
			case hir.TypeI16:
				out.WriteString(fmt.Sprintf("    movsx rcx, word %s\n", addr))
			case hir.TypeU32, hir.TypeChar, hir.TypeF32:
				out.WriteString(fmt.Sprintf("    mov ecx, dword %s\n", addr))
			case hir.TypeI32:
				out.WriteString(fmt.Sprintf("    movsxd rcx, dword %s\n", addr))
			default:
				out.WriteString(fmt.Sprintf("    mov rcx, qword %s\n", addr))
			}
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rcx\n", destOffset))
		case *mir.IndexRvalue:
			if src.Base.Type != nil && (src.Base.Type.Kind == hir.TypeString || (src.Base.Type.Kind == hir.TypePointer && src.Base.Type.ElemType != nil && src.Base.Type.ElemType.Kind == hir.TypeString)) {
				baseOffset := offsets[src.Base.LocalID]
				b.loadOperandToReg(src.Index, "rcx", offsets, out)
				if src.Base.Type.Kind == hir.TypePointer {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
					out.WriteString("    mov rax, [rax]\n")
				} else {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
				}
				out.WriteString("    movzx rax, byte [rax + rcx]\n")
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
				break
			}
			b.loadOperandToReg(src.Index, "rcx", offsets, out)
			b.emitElemScaledIndex("rcx", src.Type, out)
			if src.Base.Kind == mir.OpLocal {
				baseOffset := offsets[src.Base.LocalID]
				if src.Base.Type != nil && (src.Base.Type.Kind == hir.TypePointer || (src.Base.Type.Kind == hir.TypeArray && src.Base.Type.Size <= 0)) {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
				} else {
					out.WriteString(fmt.Sprintf("    mov rax, rbp\n"))
					out.WriteString(fmt.Sprintf("    sub rax, %d\n", baseOffset))
				}
				out.WriteString("    add rax, rcx\n")

				fKind := hir.TypeI64
				if src.Type != nil {
					fKind = src.Type.Kind
				}
				switch fKind {
				case hir.TypeBool, hir.TypeU8:
					out.WriteString("    movzx rdx, byte [rax]\n")
				case hir.TypeI8:
					out.WriteString("    movsx rdx, byte [rax]\n")
				case hir.TypeU16:
					out.WriteString("    movzx rdx, word [rax]\n")
				case hir.TypeI16:
					out.WriteString("    movsx rdx, word [rax]\n")
				case hir.TypeU32, hir.TypeChar, hir.TypeF32:
					out.WriteString("    mov edx, dword [rax]\n")
				case hir.TypeI32:
					out.WriteString("    movsxd rdx, dword [rax]\n")
				default:
					out.WriteString("    mov rdx, [rax]\n")
				}
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rdx\n", destOffset))
			} else {
				b.loadOperandToReg(src.Base, "rax", offsets, out)
				out.WriteString("    add rax, rcx\n")
				out.WriteString("    mov rdx, [rax]\n")
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rdx\n", destOffset))
			}
		case *mir.SliceRvalue:
			if (src.Type != nil && src.Type.Kind == hir.TypeString) || (src.Base.Type != nil && (src.Base.Type.Kind == hir.TypeString || (src.Base.Type.Kind == hir.TypePointer && src.Base.Type.ElemType != nil && src.Base.Type.ElemType.Kind == hir.TypeString))) {
				if src.Base.Kind == mir.OpLocal {
					baseOffset := offsets[src.Base.LocalID]
					if src.Base.Type != nil && src.Base.Type.Kind == hir.TypePointer {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
						if src.High != nil {
							b.loadOperandToReg(*src.High, "r8", offsets, out)
						} else {
							out.WriteString("    mov r8, [rax + 8]\n")
						}
						out.WriteString("    mov rax, [rax]\n")
					} else {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
						if src.High != nil {
							b.loadOperandToReg(*src.High, "r8", offsets, out)
						} else {
							out.WriteString(fmt.Sprintf("    mov r8, [rbp - %d + 8]\n", baseOffset))
						}
					}
					if src.Low != nil {
						b.loadOperandToReg(*src.Low, "rdx", offsets, out)
					} else {
						out.WriteString("    xor rdx, rdx\n")
					}
					out.WriteString("    add rax, rdx\n")
					out.WriteString("    sub r8, rdx\n")
					out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], r8\n", destOffset))
				}
			} else if src.Base.Type != nil && src.Base.Type.Kind == hir.TypeArray {
				if src.Base.Kind == mir.OpLocal {
					baseOffset := offsets[src.Base.LocalID]
					elemSize := 8
					if src.Base.Type.ElemType != nil {
						elemSize = src.Base.Type.ElemType.ByteSize()
					}
					if src.Base.Type.Size <= 0 {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
						if src.High != nil {
							b.loadOperandToReg(*src.High, "r8", offsets, out)
						} else {
							out.WriteString(fmt.Sprintf("    mov r8, [rbp - %d + 8]\n", baseOffset))
						}
					} else {
						out.WriteString(fmt.Sprintf("    mov rax, rbp\n"))
						out.WriteString(fmt.Sprintf("    sub rax, %d\n", baseOffset))
						if src.High != nil {
							b.loadOperandToReg(*src.High, "r8", offsets, out)
						} else {
							out.WriteString(fmt.Sprintf("    mov r8, %d\n", src.Base.Type.Size))
						}
					}
					if src.Low != nil {
						b.loadOperandToReg(*src.Low, "rdx", offsets, out)
					} else {
						out.WriteString("    xor rdx, rdx\n")
					}
					out.WriteString("    sub r8, rdx\n")
					if elemSize > 1 {
						out.WriteString(fmt.Sprintf("    imul rdx, %d\n", elemSize))
					}
					out.WriteString("    add rax, rdx\n")
					out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], r8\n", destOffset))
				}
			}
		case *mir.RefRvalue:
			if src.Target.Kind == mir.OpLocal {
				targetOffset := offsets[src.Target.LocalID]
				out.WriteString(fmt.Sprintf("    lea rax, [rbp - %d]\n", targetOffset))
			} else {
				out.WriteString(fmt.Sprintf("    lea rax, [global_%s]\n", src.Target.Constant))
			}
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		case *mir.DerefRvalue:
			b.loadOperandToReg(src.Target, "rax", offsets, out)
			sz := 8
			if src.Type != nil {
				sz = src.Type.ByteSize()
			} else if src.Target.Type != nil && src.Target.Type.ElemType != nil {
				// fall back to the pointee's declared size if Type wasn't propagated
				sz = src.Target.Type.ElemType.ByteSize()
			}
			if sz > 8 {
				numWords := (sz + 7) / 8
				for w := 0; w < numWords; w++ {
					out.WriteString(fmt.Sprintf("    mov rcx, [rax + %d]\n", w*8))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rcx\n", destOffset, w*8))
				}
				break
			}
			fKind := hir.TypeI64
			if src.Type != nil {
				fKind = src.Type.Kind
			}
			switch fKind {
			case hir.TypeBool, hir.TypeU8:
				out.WriteString("    movzx rcx, byte [rax]\n")
			case hir.TypeI8:
				out.WriteString("    movsx rcx, byte [rax]\n")
			case hir.TypeU16:
				out.WriteString("    movzx rcx, word [rax]\n")
			case hir.TypeI16:
				out.WriteString("    movsx rcx, word [rax]\n")
			case hir.TypeU32, hir.TypeChar, hir.TypeF32:
				out.WriteString("    mov ecx, dword [rax]\n")
			case hir.TypeI32:
				out.WriteString("    movsxd rcx, dword [rax]\n")
			default:
				out.WriteString("    mov rcx, [rax]\n")
			}
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rcx\n", destOffset))
		case *mir.CastRvalue:
			b.generateCast(src, destOffset, offsets, out)
		}
	case *mir.StoreStmt:
		b.loadOperandToReg(s.Ptr, "rax", offsets, out)
		b.loadOperandToReg(s.Val, "rcx", offsets, out)
		valKind := hir.TypeI64
		if s.Val.Type != nil {
			valKind = s.Val.Type.Kind
		}
		switch valKind {
		case hir.TypeBool, hir.TypeI8, hir.TypeU8:
			out.WriteString("    mov byte [rax], cl\n")
		case hir.TypeI16, hir.TypeU16:
			out.WriteString("    mov word [rax], cx\n")
		case hir.TypeI32, hir.TypeU32, hir.TypeChar, hir.TypeF32:
			out.WriteString("    mov dword [rax], ecx\n")
		default:
			out.WriteString("    mov qword [rax], rcx\n")
		}
	case *mir.SetFieldStmt:
		if s.Val.Type != nil && s.Val.Type.ByteSize() > 8 {
			numWords := (s.Val.Type.ByteSize() + 7) / 8
			if s.Val.Kind == mir.OpLocal {
				valOffset := offsets[s.Val.LocalID]
				if s.Base.Kind == mir.OpLocal && s.Base.Type != nil && (s.Base.Type.Kind == hir.TypeStruct || s.Base.Type.Kind == hir.TypeTuple) {
					baseOffset := offsets[s.Base.LocalID]
					for w := 0; w < numWords; w++ {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + %d]\n", valOffset, w*8))
						out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", baseOffset, s.Offset+w*8))
					}
				} else {
					b.loadOperandToReg(s.Base, "rdx", offsets, out)
					for w := 0; w < numWords; w++ {
						out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + %d]\n", valOffset, w*8))
						out.WriteString(fmt.Sprintf("    mov [rdx + %d], rax\n", s.Offset+w*8))
					}
				}
			} else {
				// OpConst string
				cleanVal := strings.Trim(s.Val.Constant, "\"")
				strLabel := b.getStringLabel(s.Val.Constant)
				if s.Base.Kind == mir.OpLocal && s.Base.Type != nil && (s.Base.Type.Kind == hir.TypeStruct || s.Base.Type.Kind == hir.TypeTuple) {
					baseOffset := offsets[s.Base.LocalID]
					out.WriteString(fmt.Sprintf("    lea rax, [%s]\n", strLabel))
					out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rax\n", baseOffset, s.Offset))
					out.WriteString(fmt.Sprintf("    mov qword [rbp - %d + %d + 8], %d\n", baseOffset, s.Offset, len(cleanVal)))
				} else {
					b.loadOperandToReg(s.Base, "rdx", offsets, out)
					out.WriteString(fmt.Sprintf("    lea rax, [%s]\n", strLabel))
					out.WriteString(fmt.Sprintf("    mov [rdx + %d], rax\n", s.Offset))
					out.WriteString(fmt.Sprintf("    mov qword [rdx + %d + 8], %d\n", s.Offset, len(cleanVal)))
				}
			}
			break
		}
		b.loadOperandToReg(s.Val, "rcx", offsets, out)
		addr := ""
		if s.Base.Kind == mir.OpLocal && s.Base.Type != nil && (s.Base.Type.Kind == hir.TypeStruct || s.Base.Type.Kind == hir.TypeTuple) {
			baseOffset := offsets[s.Base.LocalID]
			addr = fmt.Sprintf("[rbp - %d + %d]", baseOffset, s.Offset)
		} else {
			b.loadOperandToReg(s.Base, "rax", offsets, out)
			addr = fmt.Sprintf("[rax + %d]", s.Offset)
		}

		fKind := hir.TypeI64
		if s.Val.Type != nil {
			fKind = s.Val.Type.Kind
		}
		switch fKind {
		case hir.TypeBool, hir.TypeI8, hir.TypeU8:
			out.WriteString(fmt.Sprintf("    mov byte %s, cl\n", addr))
		case hir.TypeI16, hir.TypeU16:
			out.WriteString(fmt.Sprintf("    mov word %s, cx\n", addr))
		case hir.TypeI32, hir.TypeU32, hir.TypeChar, hir.TypeF32:
			out.WriteString(fmt.Sprintf("    mov dword %s, ecx\n", addr))
		default:
			out.WriteString(fmt.Sprintf("    mov qword %s, rcx\n", addr))
		}
	case *mir.SetIndexStmt:
		b.loadOperandToReg(s.Val, "rdx", offsets, out)
		b.loadOperandToReg(s.Index, "rcx", offsets, out)
		b.emitElemScaledIndex("rcx", s.Val.Type, out)
		if s.Base.Kind == mir.OpLocal {
			baseOffset := offsets[s.Base.LocalID]
			if s.Base.Type != nil && (s.Base.Type.Kind == hir.TypePointer || (s.Base.Type.Kind == hir.TypeArray && s.Base.Type.Size == 0)) {
				out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", baseOffset))
			} else {
				out.WriteString(fmt.Sprintf("    mov rax, rbp\n"))
				out.WriteString(fmt.Sprintf("    sub rax, %d\n", baseOffset))
			}
			out.WriteString("    add rax, rcx\n")

			fKind := hir.TypeI64
			if s.Val.Type != nil {
				fKind = s.Val.Type.Kind
			}
			switch fKind {
			case hir.TypeBool, hir.TypeI8, hir.TypeU8:
				out.WriteString("    mov byte [rax], dl\n")
			case hir.TypeI16, hir.TypeU16:
				out.WriteString("    mov word [rax], dx\n")
			case hir.TypeI32, hir.TypeU32, hir.TypeChar, hir.TypeF32:
				out.WriteString("    mov dword [rax], edx\n")
			default:
				out.WriteString("    mov [rax], rdx\n")
			}
		}
	case *mir.AsmStmt:
		out.WriteString(fmt.Sprintf("    %s\n", s.Assembly))
	}
}

func (b *Backend) emitElemScaledIndex(indexReg string, elemType *hir.HIRType, out *bytes.Buffer) {
	sz := 8
	if elemType != nil && elemType.ByteSize() > 0 {
		sz = elemType.ByteSize()
	}
	switch sz {
	case 1:
		// no scaling needed
	case 2:
		out.WriteString(fmt.Sprintf("    shl %s, 1\n", indexReg))
	case 4:
		out.WriteString(fmt.Sprintf("    shl %s, 2\n", indexReg))
	case 8:
		out.WriteString(fmt.Sprintf("    shl %s, 3\n", indexReg))
	default:
		out.WriteString(fmt.Sprintf("    imul %s, %d\n", indexReg, sz))
	}
}

func parseAnyInteger(s string) (int64, error) {
	s = strings.ReplaceAll(s, "_", "")
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	var val uint64
	var err error
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err = strconv.ParseUint(s[2:], 16, 64)
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		val, err = strconv.ParseUint(s[2:], 2, 64)
	} else if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		val, err = strconv.ParseUint(s[2:], 8, 64)
	} else {
		val, err = strconv.ParseUint(s, 10, 64)
	}
	if err != nil {
		return 0, err
	}
	if neg {
		return -int64(val), nil
	}
	return int64(val), nil
}

func (b *Backend) generateBinary(bin *mir.BinaryRvalue, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	isFloat := (bin.Left.Type != nil && isFloatKind(bin.Left.Type.Kind)) ||
		(bin.Right.Type != nil && isFloatKind(bin.Right.Type.Kind)) ||
		(bin.Type != nil && isFloatKind(bin.Type.Kind))

	if isFloat {
		fKind := hir.TypeF64
		if (bin.Left.Type != nil && bin.Left.Type.Kind == hir.TypeF32) ||
			(bin.Right.Type != nil && bin.Right.Type.Kind == hir.TypeF32) ||
			(bin.Type != nil && bin.Type.Kind == hir.TypeF32) {
			fKind = hir.TypeF32
		}

		b.loadOperandToXMM(bin.Left, "xmm0", fKind, offsets, out)
		b.loadOperandToXMM(bin.Right, "xmm1", fKind, offsets, out)

		switch bin.Op {
		case "+":
			if fKind == hir.TypeF32 {
				out.WriteString("    addss xmm0, xmm1\n")
			} else {
				out.WriteString("    addsd xmm0, xmm1\n")
			}
			b.storeXMMToLocal("xmm0", fKind, destOffset, out)
		case "-":
			if fKind == hir.TypeF32 {
				out.WriteString("    subss xmm0, xmm1\n")
			} else {
				out.WriteString("    subsd xmm0, xmm1\n")
			}
			b.storeXMMToLocal("xmm0", fKind, destOffset, out)
		case "*":
			if fKind == hir.TypeF32 {
				out.WriteString("    mulss xmm0, xmm1\n")
			} else {
				out.WriteString("    mulsd xmm0, xmm1\n")
			}
			b.storeXMMToLocal("xmm0", fKind, destOffset, out)
		case "/":
			if fKind == hir.TypeF32 {
				out.WriteString("    divss xmm0, xmm1\n")
			} else {
				out.WriteString("    divsd xmm0, xmm1\n")
			}
			b.storeXMMToLocal("xmm0", fKind, destOffset, out)
		case "==", "!=", "<", "<=", ">", ">=":
			if fKind == hir.TypeF32 {
				out.WriteString("    ucomiss xmm0, xmm1\n")
			} else {
				out.WriteString("    ucomisd xmm0, xmm1\n")
			}
			setInstr := ""
			switch bin.Op {
			case "==":
				setInstr = "sete"
			case "!=":
				setInstr = "setne"
			case "<":
				setInstr = "setb"
			case "<=":
				setInstr = "setbe"
			case ">":
				setInstr = "seta"
			case ">=":
				setInstr = "setae"
			}
			out.WriteString(fmt.Sprintf("    %s al\n", setInstr))
			out.WriteString("    movzx rax, al\n")
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		}
		return
	}

	isString := (bin.Left.Type != nil && bin.Left.Type.Kind == hir.TypeString) ||
		(bin.Right.Type != nil && bin.Right.Type.Kind == hir.TypeString)

	if isString {
		if bin.Left.Kind == mir.OpLocal {
			leftOffset := offsets[bin.Left.LocalID]
			if bin.Left.Type != nil && bin.Left.Type.Kind == hir.TypePointer {
				out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", leftOffset))
				out.WriteString("    mov rcx, [rax]\n")
				out.WriteString("    mov rdx, [rax + 8]\n")
			} else {
				out.WriteString(fmt.Sprintf("    mov rcx, [rbp - %d]\n", leftOffset))
				out.WriteString(fmt.Sprintf("    mov rdx, [rbp - %d + 8]\n", leftOffset))
			}
		} else {
			b.loadOperandToReg(bin.Left, "rcx", offsets, out)
			cleanVal := strings.Trim(bin.Left.Constant, "\"")
			out.WriteString(fmt.Sprintf("    mov rdx, %d\n", len(cleanVal)))
		}
		if bin.Right.Kind == mir.OpLocal {
			rightOffset := offsets[bin.Right.LocalID]
			if bin.Right.Type != nil && bin.Right.Type.Kind == hir.TypePointer {
				out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", rightOffset))
				out.WriteString("    mov r8, [rax]\n")
				out.WriteString("    mov r9, [rax + 8]\n")
			} else {
				out.WriteString(fmt.Sprintf("    mov r8, [rbp - %d]\n", rightOffset))
				out.WriteString(fmt.Sprintf("    mov r9, [rbp - %d + 8]\n", rightOffset))
			}
		} else {
			b.loadOperandToReg(bin.Right, "r8", offsets, out)
			cleanVal := strings.Trim(bin.Right.Constant, "\"")
			out.WriteString(fmt.Sprintf("    mov r9, %d\n", len(cleanVal)))
		}
		switch bin.Op {
		case "+":
			out.WriteString("    call _cupid_str_concat_slice\n")
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], rdx\n", destOffset))
		case "==":
			out.WriteString("    call _cupid_str_eq_slice\n")
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		case "!=":
			out.WriteString("    call _cupid_str_eq_slice\n")
			out.WriteString("    xor rax, 1\n")
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		}
		return
	}

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
	isFloat := (un.Right.Type != nil && isFloatKind(un.Right.Type.Kind)) ||
		(un.Type != nil && isFloatKind(un.Type.Kind))

	if isFloat {
		fKind := hir.TypeF64
		if (un.Right.Type != nil && un.Right.Type.Kind == hir.TypeF32) ||
			(un.Type != nil && un.Type.Kind == hir.TypeF32) {
			fKind = hir.TypeF32
		}

		b.loadOperandToXMM(un.Right, "xmm0", fKind, offsets, out)
		if un.Op == "-" {
			if fKind == hir.TypeF32 {
				out.WriteString("    xorps xmm1, xmm1\n")
				out.WriteString("    subss xmm1, xmm0\n")
				out.WriteString("    movaps xmm0, xmm1\n")
			} else {
				out.WriteString("    xorpd xmm1, xmm1\n")
				out.WriteString("    subsd xmm1, xmm0\n")
				out.WriteString("    movapd xmm0, xmm1\n")
			}
			b.storeXMMToLocal("xmm0", fKind, destOffset, out)
		}
		return
	}

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
		if len(call.Args) > 0 {
			arg := call.Args[0]
			if arg.Type != nil && arg.Type.Kind == hir.TypeArray && arg.Type.Size > 0 {
				out.WriteString(fmt.Sprintf("    mov rax, %d\n", arg.Type.Size))
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
				return
			}
			if arg.Kind == mir.OpLocal {
				argOffset := offsets[arg.LocalID]
				if arg.Type != nil && arg.Type.Kind == hir.TypePointer {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", argOffset))
					out.WriteString("    mov rax, [rax + 8]\n")
				} else {
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + 8]\n", argOffset))
				}
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
				return
			} else if arg.Kind == mir.OpConst && arg.Type != nil && arg.Type.Kind == hir.TypeString {
				cleanVal := strings.Trim(arg.Constant, "\"")
				out.WriteString(fmt.Sprintf("    mov rax, %d\n", len(cleanVal)))
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
				return
			} else {
				b.loadOperandToReg(arg, "rax", offsets, out)
				out.WriteString("    mov rax, [rax + 8]\n")
				out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
				return
			}
		}
		out.WriteString(fmt.Sprintf("    mov qword [rbp - %d], 0\n", destOffset))
		return
	}

	if call.FuncName == "print" || call.FuncName == "println" {
		for _, arg := range call.Args {
			b.generatePrintOperand(arg, offsets, out)
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
			if arg.Type != nil && isFloatKind(arg.Type.Kind) {
				xmmReg := fmt.Sprintf("xmm%d", i)
				b.loadOperandToXMM(arg, xmmReg, arg.Type.Kind, offsets, out)
				if arg.Type.Kind == hir.TypeF32 {
					out.WriteString(fmt.Sprintf("    movd %s, %s\n", paramRegs[i], xmmReg))
				} else {
					out.WriteString(fmt.Sprintf("    movq %s, %s\n", paramRegs[i], xmmReg))
				}
			} else if arg.Type != nil && arg.Type.ByteSize() > 8 {
				if arg.Kind == mir.OpLocal {
					argOffset := offsets[arg.LocalID]
					out.WriteString(fmt.Sprintf("    lea %s, [rbp - %d]\n", paramRegs[i], argOffset))
				} else if arg.Kind == mir.OpConst && arg.Type.Kind == hir.TypeString {
					cleanVal := strings.Trim(arg.Constant, "\"")
					label := b.getStringLabel(cleanVal)
					out.WriteString(fmt.Sprintf("    lea %s, [%s_hdr]\n", paramRegs[i], label))
				} else {
					b.loadOperandToReg(arg, paramRegs[i], offsets, out)
				}
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
	if call.Type != nil && isFloatKind(call.Type.Kind) {
		b.storeXMMToLocal("xmm0", call.Type.Kind, destOffset, out)
	} else if call.Type != nil && call.Type.ByteSize() == 16 {
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], rdx\n", destOffset))
	} else if call.Type != nil && call.Type.ByteSize() > 8 {
		numWords := (call.Type.ByteSize() + 7) / 8
		for w := 0; w < numWords; w++ {
			out.WriteString(fmt.Sprintf("    mov rcx, [rax + %d]\n", w*8))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d + %d], rcx\n", destOffset, w*8))
		}
	} else {
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
	}
}

func (b *Backend) generatePrintOperand(arg mir.Operand, offsets map[int]int, out *bytes.Buffer) {
	if arg.Type == nil {
		b.loadOperandToReg(arg, "rcx", offsets, out)
		out.WriteString("    call _cupid_print_i64\n")
		return
	}

	switch arg.Type.Kind {
	case hir.TypeString:
		if arg.Kind == mir.OpLocal {
			argOffset := offsets[arg.LocalID]
			out.WriteString(fmt.Sprintf("    mov rcx, [rbp - %d]\n", argOffset))
			out.WriteString(fmt.Sprintf("    mov rdx, [rbp - %d + 8]\n", argOffset))
		} else {
			cleanVal := strings.Trim(arg.Constant, "\"")
			label := b.getStringLabel(cleanVal)
			out.WriteString(fmt.Sprintf("    lea rcx, [%s]\n", label))
			out.WriteString(fmt.Sprintf("    mov rdx, %d\n", len(cleanVal)))
		}
		out.WriteString("    call _cupid_print_str_len\n")

	case hir.TypeChar:
		b.loadOperandToReg(arg, "rcx", offsets, out)
		out.WriteString("    call _cupid_print_char\n")

	case hir.TypeBool:
		b.loadOperandToReg(arg, "rcx", offsets, out)
		out.WriteString("    call _cupid_print_bool\n")

	case hir.TypeF32, hir.TypeF64:
		b.loadOperandToXMM(arg, "xmm0", arg.Type.Kind, offsets, out)
		if arg.Type.Kind == hir.TypeF32 {
			out.WriteString("    cvtss2sd xmm0, xmm0\n")
		}
		out.WriteString("    call _cupid_print_f64\n")

	case hir.TypeArray:
		b.generatePrintArray(arg, offsets, out)

	case hir.TypeTuple:
		b.generatePrintTuple(arg, offsets, out)

	case hir.TypeStruct:
		b.generatePrintStruct(arg, offsets, out)

	case hir.TypePointer:
		if arg.Type.ElemType != nil {
			switch arg.Type.ElemType.Kind {
			case hir.TypeString:
				if arg.Kind == mir.OpLocal {
					argOffset := offsets[arg.LocalID]
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", argOffset))
					out.WriteString("    mov rcx, [rax]\n")
					out.WriteString("    mov rdx, [rax + 8]\n")
					out.WriteString("    call _cupid_print_str_len\n")
				}
				return
			case hir.TypeStruct:
				out.WriteString("    lea rcx, [_cupid_ampersand]\n")
				out.WriteString("    call _cupid_print_str\n")
				b.generatePrintStruct(arg, offsets, out)
				return
			case hir.TypeTuple:
				out.WriteString("    lea rcx, [_cupid_ampersand]\n")
				out.WriteString("    call _cupid_print_str\n")
				b.generatePrintTuple(arg, offsets, out)
				return
			case hir.TypeArray:
				out.WriteString("    lea rcx, [_cupid_ampersand]\n")
				out.WriteString("    call _cupid_print_str\n")
				b.generatePrintArray(arg, offsets, out)
				return
			}
		}
		if arg.Type.ElemType != nil {
			if _, ok := b.prog.Structs[arg.Type.ElemType.Name]; ok {
				out.WriteString("    lea rcx, [_cupid_ampersand]\n")
				out.WriteString("    call _cupid_print_str\n")
				b.generatePrintStruct(arg, offsets, out)
				return
			}
		}
		if _, ok := b.prog.Structs[arg.Type.Name]; ok {
			out.WriteString("    lea rcx, [_cupid_ampersand]\n")
			out.WriteString("    call _cupid_print_str\n")
			b.generatePrintStruct(arg, offsets, out)
			return
		}
		b.loadOperandToReg(arg, "rcx", offsets, out)
		out.WriteString("    call _cupid_print_i64\n")

	default:
		b.loadOperandToReg(arg, "rcx", offsets, out)
		out.WriteString("    call _cupid_print_i64\n")
	}
}

func (b *Backend) generatePrintStruct(arg mir.Operand, offsets map[int]int, out *bytes.Buffer) {
	stType := arg.Type
	isPointer := false
	if stType != nil && stType.Kind == hir.TypePointer && stType.ElemType != nil {
		stType = stType.ElemType
		isPointer = true
	}

	stName := "Struct"
	if stType != nil && stType.Name != "" {
		stName = stType.Name
	}

	label := b.getStringLabel(stName)
	out.WriteString(fmt.Sprintf("    lea rcx, [%s]\n", label))
	out.WriteString("    call _cupid_print_str\n")

	out.WriteString("    lea rcx, [_cupid_lbrace]\n")
	out.WriteString("    call _cupid_print_str\n")

	var fields []hir.HIRField
	if stType != nil && len(stType.Fields) > 0 {
		fields = stType.Fields
	} else if stDef, ok := b.prog.Structs[stName]; ok && len(stDef.Fields) > 0 {
		fields = stDef.Fields
	}

	baseOffset := 0
	if arg.Kind == mir.OpLocal {
		baseOffset = offsets[arg.LocalID]
	}

	for i, f := range fields {
		if i > 0 {
			out.WriteString("    lea rcx, [_cupid_comma_space]\n")
			out.WriteString("    call _cupid_print_str\n")
		}

		fLabel := b.getStringLabel(f.Name)
		out.WriteString(fmt.Sprintf("    lea rcx, [%s]\n", fLabel))
		out.WriteString("    call _cupid_print_str\n")

		out.WriteString("    lea rcx, [_cupid_colon_space]\n")
		out.WriteString("    call _cupid_print_str\n")

		if isPointer {
			out.WriteString(fmt.Sprintf("    mov rdi, [rbp - %d]\n", baseOffset))
			addr := fmt.Sprintf("[rdi + %d]", f.Offset)
			b.generatePrintFieldAt(addr, f.Type, offsets, out)
		} else {
			addr := fmt.Sprintf("[rbp - %d + %d]", baseOffset, f.Offset)
			b.generatePrintFieldAt(addr, f.Type, offsets, out)
		}
	}

	out.WriteString("    lea rcx, [_cupid_rbrace]\n")
	out.WriteString("    call _cupid_print_str\n")
}

func (b *Backend) generatePrintTuple(arg mir.Operand, offsets map[int]int, out *bytes.Buffer) {
	tType := arg.Type
	isPointer := false
	if tType != nil && tType.Kind == hir.TypePointer && tType.ElemType != nil {
		tType = tType.ElemType
		isPointer = true
	}

	out.WriteString("    lea rcx, [_cupid_lparen]\n")
	out.WriteString("    call _cupid_print_str\n")

	var fields []hir.HIRField
	if tType != nil {
		fields = tType.Fields
	}

	baseOffset := 0
	if arg.Kind == mir.OpLocal {
		baseOffset = offsets[arg.LocalID]
	}

	for i, f := range fields {
		if i > 0 {
			out.WriteString("    lea rcx, [_cupid_comma_space]\n")
			out.WriteString("    call _cupid_print_str\n")
		}

		if isPointer {
			out.WriteString(fmt.Sprintf("    mov rdi, [rbp - %d]\n", baseOffset))
			addr := fmt.Sprintf("[rdi + %d]", f.Offset)
			b.generatePrintFieldAt(addr, f.Type, offsets, out)
		} else {
			addr := fmt.Sprintf("[rbp - %d + %d]", baseOffset, f.Offset)
			b.generatePrintFieldAt(addr, f.Type, offsets, out)
		}
	}

	out.WriteString("    lea rcx, [_cupid_rparen]\n")
	out.WriteString("    call _cupid_print_str\n")
}

func (b *Backend) generatePrintFieldAt(addr string, ft *hir.HIRType, offsets map[int]int, out *bytes.Buffer) {
	if ft == nil {
		out.WriteString(fmt.Sprintf("    mov rcx, %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
		return
	}
	switch ft.Kind {
	case hir.TypeString:
		out.WriteString(fmt.Sprintf("    mov rcx, %s\n", addr))
		out.WriteString("    call _cupid_print_str_literal\n")
	case hir.TypeChar:
		out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
		out.WriteString("    call _cupid_print_char_literal\n")
	case hir.TypeBool:
		out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
		out.WriteString("    call _cupid_print_bool\n")
	case hir.TypeF32:
		out.WriteString(fmt.Sprintf("    movss xmm0, %s\n", addr))
		out.WriteString("    cvtss2sd xmm0, xmm0\n")
		out.WriteString("    call _cupid_print_f64\n")
	case hir.TypeF64:
		out.WriteString(fmt.Sprintf("    movsd xmm0, %s\n", addr))
		out.WriteString("    call _cupid_print_f64\n")
	case hir.TypeI8:
		out.WriteString(fmt.Sprintf("    movsx rcx, byte %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	case hir.TypeU8:
		out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	case hir.TypeI16:
		out.WriteString(fmt.Sprintf("    movsx rcx, word %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	case hir.TypeU16:
		out.WriteString(fmt.Sprintf("    movzx rcx, word %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	case hir.TypeI32:
		out.WriteString(fmt.Sprintf("    movsxd rcx, dword %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	case hir.TypeU32:
		out.WriteString(fmt.Sprintf("    mov ecx, dword %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	default:
		out.WriteString(fmt.Sprintf("    mov rcx, %s\n", addr))
		out.WriteString("    call _cupid_print_i64\n")
	}
}

func (b *Backend) generatePrintArray(arg mir.Operand, offsets map[int]int, out *bytes.Buffer) {
	elemType := &hir.HIRType{Kind: hir.TypeI64}
	if arg.Type != nil && arg.Type.ElemType != nil {
		elemType = arg.Type.ElemType
	}
	elemSize := elemType.ByteSize()
	size := 0
	if arg.Type != nil {
		size = arg.Type.Size
	}

	out.WriteString("    lea rcx, [_cupid_lbracket]\n")
	out.WriteString("    call _cupid_print_str\n")

	baseOffset := 0
	if arg.Kind == mir.OpLocal {
		baseOffset = offsets[arg.LocalID]
	}

	for i := 0; i < size; i++ {
		if i > 0 {
			out.WriteString("    lea rcx, [_cupid_comma_space]\n")
			out.WriteString("    call _cupid_print_str\n")
		}

		addr := fmt.Sprintf("[rbp - %d + %d]", baseOffset, i*elemSize)

		switch elemType.Kind {
		case hir.TypeChar:
			out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
			out.WriteString("    call _cupid_print_char_literal\n")
		case hir.TypeString:
			out.WriteString(fmt.Sprintf("    mov rcx, %s\n", addr))
			out.WriteString("    call _cupid_print_str_literal\n")
		case hir.TypeBool:
			out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
			out.WriteString("    call _cupid_print_bool\n")
		case hir.TypeF32:
			out.WriteString(fmt.Sprintf("    movss xmm0, %s\n", addr))
			out.WriteString("    cvtss2sd xmm0, xmm0\n")
			out.WriteString("    call _cupid_print_f64\n")
		case hir.TypeF64:
			out.WriteString(fmt.Sprintf("    movsd xmm0, %s\n", addr))
			out.WriteString("    call _cupid_print_f64\n")
		case hir.TypeI8:
			out.WriteString(fmt.Sprintf("    movsx rcx, byte %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		case hir.TypeU8:
			out.WriteString(fmt.Sprintf("    movzx rcx, byte %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		case hir.TypeI16:
			out.WriteString(fmt.Sprintf("    movsx rcx, word %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		case hir.TypeU16:
			out.WriteString(fmt.Sprintf("    movzx rcx, word %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		case hir.TypeI32:
			out.WriteString(fmt.Sprintf("    movsxd rcx, dword %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		case hir.TypeU32:
			out.WriteString(fmt.Sprintf("    mov ecx, dword %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		default:
			out.WriteString(fmt.Sprintf("    mov rcx, %s\n", addr))
			out.WriteString("    call _cupid_print_i64\n")
		}
	}

	out.WriteString("    lea rcx, [_cupid_rbracket]\n")
	out.WriteString("    call _cupid_print_str\n")
}

func (b *Backend) generateTerminator(term mir.Terminator, fn *mir.MIRFunction, offsets map[int]int, out *bytes.Buffer) {
	fnLabel := b.functionLabel(fn.Name)
	switch t := term.(type) {
	case *mir.ReturnTerminator:
		if t.Value != nil {
			if fn.ReturnType != nil && isFloatKind(fn.ReturnType.Kind) {
				b.loadOperandToXMM(*t.Value, "xmm0", fn.ReturnType.Kind, offsets, out)
				if fn.ReturnType.Kind == hir.TypeF32 {
					out.WriteString("    movd rax, xmm0\n")
				} else {
					out.WriteString("    movq rax, xmm0\n")
				}
			} else if fn.ReturnType != nil && fn.ReturnType.ByteSize() == 16 {
				if t.Value.Kind == mir.OpLocal {
					valOffset := offsets[t.Value.LocalID]
					out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", valOffset))
					out.WriteString(fmt.Sprintf("    mov rdx, [rbp - %d + 8]\n", valOffset))
				} else if t.Value.Kind == mir.OpConst && fn.ReturnType.Kind == hir.TypeString {
					cleanVal := strings.Trim(t.Value.Constant, "\"")
					label := b.getStringLabel(cleanVal)
					out.WriteString(fmt.Sprintf("    lea rax, [%s]\n", label))
					out.WriteString(fmt.Sprintf("    mov rdx, %d\n", len(cleanVal)))
				}
			} else if fn.ReturnType != nil && fn.ReturnType.ByteSize() > 8 && t.Value.Kind == mir.OpLocal {
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
		} else if op.Type != nil && isFloatKind(op.Type.Kind) {
			cleanVal := strings.ReplaceAll(val, "_", "")
			f, err := strconv.ParseFloat(cleanVal, 64)
			if err != nil {
				if intVal, err2 := parseAnyInteger(val); err2 == nil {
					f = float64(intVal)
				}
			}
			if op.Type.Kind == hir.TypeF32 {
				bits := math.Float32bits(float32(f))
				out.WriteString(fmt.Sprintf("    mov %s, %d\n", reg, bits))
			} else {
				bits := math.Float64bits(f)
				out.WriteString(fmt.Sprintf("    mov %s, %d\n", reg, bits))
			}
		} else if f, err := strconv.ParseFloat(strings.ReplaceAll(val, "_", ""), 64); err == nil && (strings.Contains(val, ".") || strings.Contains(val, "e") || strings.Contains(val, "E")) {
			bits := math.Float64bits(f)
			out.WriteString(fmt.Sprintf("    mov %s, %d\n", reg, bits))
		} else if intVal, err := parseAnyInteger(val); err == nil {
			out.WriteString(fmt.Sprintf("    mov %s, %d\n", reg, intVal))
		} else {
			// Global or function symbol reference
			isFn := false
			for _, fn := range b.prog.Functions {
				if fn.Name == val {
					isFn = true
					break
				}
			}
			if isFn {
				out.WriteString(fmt.Sprintf("    lea %s, [%s]\n", reg, b.functionLabel(val)))
			} else {
				out.WriteString(fmt.Sprintf("    mov %s, [global_%s]\n", reg, val))
			}
		}
	}
}

// charLiteralCodePoint decodes a lexer char-literal's raw text (e.g. "a",
// "\n", "\\") into its Unicode code point, matching lexer.readCharLiteral.
func charLiteralCodePoint(raw string) int64 {
	raw = strings.Trim(raw, "'")
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '\\' && len(raw) >= 2 {
		switch raw[1] {
		case 'n':
			return '\n'
		case 'r':
			return '\r'
		case 't':
			return '\t'
		case '\\':
			return '\\'
		case '\'':
			return '\''
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

	cleanVal := strings.ReplaceAll(op.Constant, "_", "")
	f, err := strconv.ParseFloat(cleanVal, 64)
	if err != nil {
		if intVal, err2 := parseAnyInteger(op.Constant); err2 == nil {
			f = float64(intVal)
		} else {
			f = 0
		}
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
// generateCastToString formats any scalar into a 16-byte {ptr, len} string.
func (b *Backend) generateCastToString(c *mir.CastRvalue, fromKind hir.HIRTypeKind, destOffset int, offsets map[int]int, out *bytes.Buffer) {
	switch {
	case fromKind == hir.TypeString:
		if c.Value.Kind == mir.OpLocal {
			srcOffset := offsets[c.Value.LocalID]
			out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d]\n", srcOffset))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
			out.WriteString(fmt.Sprintf("    mov rax, [rbp - %d + 8]\n", srcOffset))
			out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], rax\n", destOffset))
			return
		}
		b.loadOperandToReg(c.Value, "rax", offsets, out)
		out.WriteString(fmt.Sprintf("    mov [rbp - %d], rax\n", destOffset))
		return
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
	out.WriteString(fmt.Sprintf("    mov [rbp - %d + 8], rdx\n", destOffset))
}

func (b *Backend) emitRuntimeHelpers(out *bytes.Buffer) {
	// _cupid_print_str_len: RCX = ptr, RDX = len
	out.WriteString("_cupid_print_str_len:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov r8, rcx\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str_len]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

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

	// _cupid_print_char: RCX contains character byte
	out.WriteString("_cupid_print_char:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], cl\n")
	out.WriteString("    mov byte [rbp - 7], 0\n")
	out.WriteString("    lea rdx, [rbp - 8]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_print_char_literal: RCX contains character byte -> prints 'c'
	out.WriteString("_cupid_print_char_literal:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], cl\n")
	out.WriteString("    mov byte [rbp - 7], 0\n")
	out.WriteString("    lea rdx, [_cupid_single_quote]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    lea rdx, [rbp - 8]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    lea rdx, [_cupid_single_quote]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_print_str_literal: RCX contains string pointer -> prints "str"
	out.WriteString("_cupid_print_str_literal:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    lea rdx, [_cupid_double_quote]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rdx, [rbp - 8]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    lea rdx, [_cupid_double_quote]\n")
	out.WriteString("    lea rcx, [_cupid_fmt_str]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_panic_bounds: RCX = index, RDX = len
	out.WriteString("_cupid_panic_bounds:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov r8, rdx\n")
	out.WriteString("    mov rdx, rcx\n")
	out.WriteString("    lea rcx, [_cupid_fmt_bounds_panic]\n")
	out.WriteString("    call [printf]\n")
	out.WriteString("    mov ecx, 1\n")
	out.WriteString("    call [ExitProcess]\n\n")

	// _cupid_str_index: RCX = str, RDX = index -> RAX = char
	out.WriteString("_cupid_str_index:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    cmp rdx, 0\n")
	out.WriteString("    jl .csi_bounds_err\n")
	out.WriteString("    call _cupid_str_len\n")
	out.WriteString("    mov rdx, [rbp - 16]\n")
	out.WriteString("    cmp rdx, rax\n")
	out.WriteString("    jge .csi_bounds_err\n")
	out.WriteString("    mov rcx, [rbp - 8]\n")
	out.WriteString("    movzx rax, byte [rcx + rdx]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n")
	out.WriteString(".csi_bounds_err:\n")
	out.WriteString("    mov rcx, [rbp - 16]\n")
	out.WriteString("    mov rdx, rax\n")
	out.WriteString("    call _cupid_panic_bounds\n\n")

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

	// _cupid_print_f64: XMM0 contains double
	out.WriteString("_cupid_print_f64:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    movsd [rbp - 8], xmm0\n")
	out.WriteString("    movsd xmm1, [rbp - 8]\n")
	out.WriteString("    movq rdx, xmm1\n")
	out.WriteString("    lea rcx, [_cupid_fmt_f64]\n")
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
	out.WriteString("    mov [rbp - 8], rcx ; save bytes\n")
	out.WriteString("    call [GetProcessHeap]\n")
	out.WriteString("    mov rcx, rax ; hHeap\n")
	out.WriteString("    mov edx, 8   ; HEAP_ZERO_MEMORY\n")
	out.WriteString("    mov r8, [rbp - 8] ; bytes\n")
	out.WriteString("    call [HeapAlloc]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_free: RCX contains memory pointer
	out.WriteString("_cupid_free:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx ; save lpMem\n")
	out.WriteString("    call [GetProcessHeap]\n")
	out.WriteString("    mov rcx, rax ; hHeap\n")
	out.WriteString("    xor edx, edx ; dwFlags\n")
	out.WriteString("    mov r8, [rbp - 8] ; lpMem\n")
	out.WriteString("    call [HeapFree]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_len: RCX = string ptr -> RAX = byte length
	out.WriteString("_cupid_str_len:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 32\n")
	out.WriteString("    mov rax, rcx\n")
	out.WriteString("    cmp rax, 0\n")
	out.WriteString("    je .csl_zero\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString(".csl_loop:\n")
	out.WriteString("    cmp byte [rax + rdx], 0\n")
	out.WriteString("    je .csl_done\n")
	out.WriteString("    inc rdx\n")
	out.WriteString("    jmp .csl_loop\n")
	out.WriteString(".csl_done:\n")
	out.WriteString("    mov rax, rdx\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n")
	out.WriteString(".csl_zero:\n")
	out.WriteString("    xor rax, rax\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_concat_slice: RCX = ptr1, RDX = len1, R8 = ptr2, R9 = len2 -> RAX = ptr, RDX = len
	out.WriteString("_cupid_str_concat_slice:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 64\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    mov [rbp - 24], r8\n")
	out.WriteString("    mov [rbp - 32], r9\n")
	out.WriteString("    mov rcx, rdx\n")
	out.WriteString("    add rcx, r9\n")
	out.WriteString("    inc rcx\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 40], rax\n")
	out.WriteString("    mov r10, [rbp - 8]\n")
	out.WriteString("    mov r11, [rbp - 40]\n")
	out.WriteString("    mov rcx, [rbp - 16]\n")
	out.WriteString("    test rcx, rcx\n")
	out.WriteString("    jz .cscs_s2\n")
	out.WriteString(".cscs_s1_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov [r11], al\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .cscs_s1_loop\n")
	out.WriteString(".cscs_s2:\n")
	out.WriteString("    mov r10, [rbp - 24]\n")
	out.WriteString("    mov rcx, [rbp - 32]\n")
	out.WriteString("    test rcx, rcx\n")
	out.WriteString("    jz .cscs_done\n")
	out.WriteString(".cscs_s2_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov [r11], al\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .cscs_s2_loop\n")
	out.WriteString(".cscs_done:\n")
	out.WriteString("    mov byte [r11], 0\n")
	out.WriteString("    mov rax, [rbp - 40]\n")
	out.WriteString("    mov rdx, [rbp - 16]\n")
	out.WriteString("    add rdx, [rbp - 32]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_eq_slice: RCX = ptr1, RDX = len1, R8 = ptr2, R9 = len2 -> RAX = 1 or 0
	out.WriteString("_cupid_str_eq_slice:\n")
	out.WriteString("    cmp rdx, r9\n")
	out.WriteString("    jne .cseqs_false\n")
	out.WriteString("    test rdx, rdx\n")
	out.WriteString("    jz .cseqs_true\n")
	out.WriteString("    mov r10, rcx\n")
	out.WriteString("    mov r11, r8\n")
	out.WriteString("    mov rcx, rdx\n")
	out.WriteString(".cseqs_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    cmp al, [r11]\n")
	out.WriteString("    jne .cseqs_false\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .cseqs_loop\n")
	out.WriteString(".cseqs_true:\n")
	out.WriteString("    mov rax, 1\n")
	out.WriteString("    ret\n")
	out.WriteString(".cseqs_false:\n")
	out.WriteString("    xor rax, rax\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_concat: RCX = s1, RDX = s2 -> RAX = new heap string
	out.WriteString("_cupid_str_concat:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 64\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    call _cupid_str_len\n")
	out.WriteString("    mov [rbp - 24], rax\n")
	out.WriteString("    mov rcx, [rbp - 16]\n")
	out.WriteString("    call _cupid_str_len\n")
	out.WriteString("    mov [rbp - 32], rax\n")
	out.WriteString("    mov rcx, [rbp - 24]\n")
	out.WriteString("    add rcx, [rbp - 32]\n")
	out.WriteString("    inc rcx\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 40], rax\n")
	out.WriteString("    mov r10, [rbp - 8]\n")
	out.WriteString("    mov r11, [rbp - 40]\n")
	out.WriteString("    mov rcx, [rbp - 24]\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .csc_s2\n")
	out.WriteString(".csc_s1_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov [r11], al\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .csc_s1_loop\n")
	out.WriteString(".csc_s2:\n")
	out.WriteString("    mov r10, [rbp - 16]\n")
	out.WriteString("    mov rcx, [rbp - 32]\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .csc_done\n")
	out.WriteString(".csc_s2_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov [r11], al\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .csc_s2_loop\n")
	out.WriteString(".csc_done:\n")
	out.WriteString("    mov byte [r11], 0\n")
	out.WriteString("    mov rax, [rbp - 40]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_eq: RCX = s1, RDX = s2 -> RAX = 1 (true) or 0 (false)
	out.WriteString("_cupid_str_eq:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 32\n")
	out.WriteString("    mov r10, rcx\n")
	out.WriteString("    mov r11, rdx\n")
	out.WriteString("    cmp r10, r11\n")
	out.WriteString("    je .cseq_true\n")
	out.WriteString("    cmp r10, 0\n")
	out.WriteString("    je .cseq_false\n")
	out.WriteString("    cmp r11, 0\n")
	out.WriteString("    je .cseq_false\n")
	out.WriteString(".cseq_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov dl, [r11]\n")
	out.WriteString("    cmp al, dl\n")
	out.WriteString("    jne .cseq_false\n")
	out.WriteString("    cmp al, 0\n")
	out.WriteString("    je .cseq_true\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    jmp .cseq_loop\n")
	out.WriteString(".cseq_true:\n")
	out.WriteString("    mov rax, 1\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n")
	out.WriteString(".cseq_false:\n")
	out.WriteString("    xor rax, rax\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_str_slice: RCX = str, RDX = low, R8 = high -> RAX = new substring
	out.WriteString("_cupid_str_slice:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 64\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    mov [rbp - 24], r8\n")
	out.WriteString("    call _cupid_str_len\n")
	out.WriteString("    mov r9, rax\n")
	out.WriteString("    mov rdx, [rbp - 16]\n")
	out.WriteString("    cmp rdx, 0\n")
	out.WriteString("    jge .css_low_ok\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString(".css_low_ok:\n")
	out.WriteString("    cmp rdx, r9\n")
	out.WriteString("    jle .css_low_bounded\n")
	out.WriteString("    mov rdx, r9\n")
	out.WriteString(".css_low_bounded:\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    mov r8, [rbp - 24]\n")
	out.WriteString("    cmp r8, -1\n")
	out.WriteString("    je .css_high_is_len\n")
	out.WriteString("    cmp r8, r9\n")
	out.WriteString("    jle .css_high_bounded\n")
	out.WriteString(".css_high_is_len:\n")
	out.WriteString("    mov r8, r9\n")
	out.WriteString(".css_high_bounded:\n")
	out.WriteString("    cmp r8, rdx\n")
	out.WriteString("    jge .css_high_ok\n")
	out.WriteString("    mov r8, rdx\n")
	out.WriteString(".css_high_ok:\n")
	out.WriteString("    mov [rbp - 24], r8\n")
	out.WriteString("    mov rcx, r8\n")
	out.WriteString("    sub rcx, rdx\n")
	out.WriteString("    mov [rbp - 32], rcx\n")
	out.WriteString("    inc rcx\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov [rbp - 40], rax\n")
	out.WriteString("    mov r10, [rbp - 8]\n")
	out.WriteString("    add r10, [rbp - 16]\n")
	out.WriteString("    mov r11, [rbp - 40]\n")
	out.WriteString("    mov rcx, [rbp - 32]\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .css_done\n")
	out.WriteString(".css_copy_loop:\n")
	out.WriteString("    mov al, [r10]\n")
	out.WriteString("    mov [r11], al\n")
	out.WriteString("    inc r10\n")
	out.WriteString("    inc r11\n")
	out.WriteString("    dec rcx\n")
	out.WriteString("    jnz .css_copy_loop\n")
	out.WriteString(".css_done:\n")
	out.WriteString("    mov byte [r11], 0\n")
	out.WriteString("    mov rax, [rbp - 40]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// ---- Helpers backing typename(x) -> string casts ----

	// _cupid_bool_to_str: RCX = 0 or 1 -> RAX = pointer, RDX = len
	out.WriteString("_cupid_bool_to_str:\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    je .cbts_false\n")
	out.WriteString("    lea rax, [_cupid_true_str]\n")
	out.WriteString("    mov rdx, 4\n")
	out.WriteString("    ret\n")
	out.WriteString(".cbts_false:\n")
	out.WriteString("    lea rax, [_cupid_false_str]\n")
	out.WriteString("    mov rdx, 5\n")
	out.WriteString("    ret\n\n")

	// _cupid_char_to_str: RCX = code point -> RAX = pointer, RDX = 1
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
	out.WriteString("    mov rdx, 1\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_i64_to_str: RCX = signed 64-bit value -> RAX = ptr, RDX = len
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
	out.WriteString("    movsxd rdx, eax\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_u64_to_str: RCX = unsigned 64-bit value -> RAX = ptr, RDX = len
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
	out.WriteString("    movsxd rdx, eax\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_f64_to_str: XMM0 = double value -> RAX = ptr, RDX = len
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
	out.WriteString("    movsxd rdx, eax\n")
	out.WriteString("    mov rax, [rbp - 16]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_go_spawn: RCX = func_ptr, RDX = arg
	out.WriteString("_cupid_go_spawn:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov [rbp - 8], rcx\n")
	out.WriteString("    mov [rbp - 16], rdx\n")
	out.WriteString("    xor rcx, rcx\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString("    mov r8, [rbp - 8]\n")
	out.WriteString("    mov r9, [rbp - 16]\n")
	out.WriteString("    mov qword [rsp + 32], 0\n")
	out.WriteString("    mov qword [rsp + 40], 0\n")
	out.WriteString("    call [CreateThread]\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_chan_new: RCX = elem_size, RDX = capacity -> RAX = chan_ptr
	out.WriteString("_cupid_chan_new:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    push rbx\n")
	out.WriteString("    push rsi\n")
	out.WriteString("    push rdi\n")
	out.WriteString("    push r12\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    cmp rdx, 0\n")
	out.WriteString("    jg .ccn_cap_ok\n")
	out.WriteString("    mov rdx, 16\n")
	out.WriteString(".ccn_cap_ok:\n")
	out.WriteString("    cmp rcx, 0\n")
	out.WriteString("    jg .ccn_size_ok\n")
	out.WriteString("    mov rcx, 8\n")
	out.WriteString(".ccn_size_ok:\n")
	out.WriteString("    mov [rbp - 40], rcx\n")
	out.WriteString("    mov [rbp - 48], rdx\n")
	out.WriteString("    mov rax, rcx\n")
	out.WriteString("    imul rax, rdx\n")
	out.WriteString("    add rax, 64\n")
	out.WriteString("    mov rcx, rax\n")
	out.WriteString("    call _cupid_alloc\n")
	out.WriteString("    mov rbx, rax\n")
	out.WriteString("    xor rcx, rcx\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString("    xor r8, r8\n")
	out.WriteString("    xor r9, r9\n")
	out.WriteString("    call [CreateMutexA]\n")
	out.WriteString("    mov [rbx], rax\n")
	out.WriteString("    xor rcx, rcx\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString("    mov r8, [rbp - 48]\n")
	out.WriteString("    xor r9, r9\n")
	out.WriteString("    call [CreateSemaphoreA]\n")
	out.WriteString("    mov [rbx + 8], rax\n")
	out.WriteString("    xor rcx, rcx\n")
	out.WriteString("    mov rdx, [rbp - 48]\n")
	out.WriteString("    mov r8, [rbp - 48]\n")
	out.WriteString("    xor r9, r9\n")
	out.WriteString("    call [CreateSemaphoreA]\n")
	out.WriteString("    mov [rbx + 16], rax\n")
	out.WriteString("    mov rax, [rbp - 48]\n")
	out.WriteString("    mov [rbx + 24], rax\n")
	out.WriteString("    mov rax, [rbp - 40]\n")
	out.WriteString("    mov [rbx + 32], rax\n")
	out.WriteString("    mov qword [rbx + 40], 0\n")
	out.WriteString("    mov qword [rbx + 48], 0\n")
	out.WriteString("    mov qword [rbx + 56], 0\n")
	out.WriteString("    mov rax, rbx\n")
	out.WriteString("    add rsp, 48\n")
	out.WriteString("    pop r12\n")
	out.WriteString("    pop rdi\n")
	out.WriteString("    pop rsi\n")
	out.WriteString("    pop rbx\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_chan_send: RCX = chan_ptr, RDX = val
	out.WriteString("_cupid_chan_send:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    push rbx\n")
	out.WriteString("    push rsi\n")
	out.WriteString("    push rdi\n")
	out.WriteString("    push r12\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov rbx, rcx\n")
	out.WriteString("    mov r12, rdx\n")
	out.WriteString("    mov rcx, [rbx + 16]\n")
	out.WriteString("    mov rdx, -1\n")
	out.WriteString("    call [WaitForSingleObject]\n")
	out.WriteString("    mov rcx, [rbx]\n")
	out.WriteString("    mov rdx, -1\n")
	out.WriteString("    call [WaitForSingleObject]\n")
	out.WriteString("    mov rsi, [rbx + 48]\n")
	out.WriteString("    mov rax, [rbx + 32]\n")
	out.WriteString("    imul rsi, rax\n")
	out.WriteString("    lea rdi, [rbx + 64 + rsi]\n")
	out.WriteString("    mov [rdi], r12\n")
	out.WriteString("    mov rax, [rbx + 48]\n")
	out.WriteString("    inc rax\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString("    mov rcx, [rbx + 24]\n")
	out.WriteString("    div rcx\n")
	out.WriteString("    mov [rbx + 48], rdx\n")
	out.WriteString("    inc qword [rbx + 56]\n")
	out.WriteString("    mov rcx, [rbx]\n")
	out.WriteString("    call [ReleaseMutex]\n")
	out.WriteString("    mov rcx, [rbx + 8]\n")
	out.WriteString("    mov rdx, 1\n")
	out.WriteString("    xor r8, r8\n")
	out.WriteString("    call [ReleaseSemaphore]\n")
	out.WriteString("    add rsp, 48\n")
	out.WriteString("    pop r12\n")
	out.WriteString("    pop rdi\n")
	out.WriteString("    pop rsi\n")
	out.WriteString("    pop rbx\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")

	// _cupid_chan_recv: RCX = chan_ptr -> RAX = val
	out.WriteString("_cupid_chan_recv:\n")
	out.WriteString("    push rbp\n")
	out.WriteString("    mov rbp, rsp\n")
	out.WriteString("    push rbx\n")
	out.WriteString("    push rsi\n")
	out.WriteString("    push rdi\n")
	out.WriteString("    push r12\n")
	out.WriteString("    sub rsp, 48\n")
	out.WriteString("    mov rbx, rcx\n")
	out.WriteString("    mov rcx, [rbx + 8]\n")
	out.WriteString("    mov rdx, -1\n")
	out.WriteString("    call [WaitForSingleObject]\n")
	out.WriteString("    mov rcx, [rbx]\n")
	out.WriteString("    mov rdx, -1\n")
	out.WriteString("    call [WaitForSingleObject]\n")
	out.WriteString("    mov rsi, [rbx + 40]\n")
	out.WriteString("    mov rax, [rbx + 32]\n")
	out.WriteString("    imul rsi, rax\n")
	out.WriteString("    lea rdi, [rbx + 64 + rsi]\n")
	out.WriteString("    mov r12, [rdi]\n")
	out.WriteString("    mov rax, [rbx + 40]\n")
	out.WriteString("    inc rax\n")
	out.WriteString("    xor rdx, rdx\n")
	out.WriteString("    mov rcx, [rbx + 24]\n")
	out.WriteString("    div rcx\n")
	out.WriteString("    mov [rbx + 40], rdx\n")
	out.WriteString("    dec qword [rbx + 56]\n")
	out.WriteString("    mov rcx, [rbx]\n")
	out.WriteString("    call [ReleaseMutex]\n")
	out.WriteString("    mov rcx, [rbx + 16]\n")
	out.WriteString("    mov rdx, 1\n")
	out.WriteString("    xor r8, r8\n")
	out.WriteString("    call [ReleaseSemaphore]\n")
	out.WriteString("    mov rax, r12\n")
	out.WriteString("    add rsp, 48\n")
	out.WriteString("    pop r12\n")
	out.WriteString("    pop rdi\n")
	out.WriteString("    pop rsi\n")
	out.WriteString("    pop rbx\n")
	out.WriteString("    mov rsp, rbp\n")
	out.WriteString("    pop rbp\n")
	out.WriteString("    ret\n\n")
}

func (b *Backend) emitGlobalInitializers(out *bytes.Buffer) {
	for _, g := range b.prog.Globals {
		if g.Value == nil {
			continue
		}
		b.emitGlobalInitExpr(g.Name, g.Value, out)
	}
}

func (b *Backend) emitGlobalInitExpr(globalName string, expr hir.HIRExpr, out *bytes.Buffer) {
	switch e := expr.(type) {
	case *hir.HIRLiteral:
		if e.Typ != nil && e.Typ.Kind == hir.TypeString {
			cleanVal := strings.Trim(e.Value, "\"")
			label := b.getStringLabel(cleanVal)
			out.WriteString(fmt.Sprintf("    lea rax, [%s]\n", label))
		} else if intVal, err := parseAnyInteger(e.Value); err == nil {
			out.WriteString(fmt.Sprintf("    mov rax, %d\n", intVal))
		} else {
			out.WriteString(fmt.Sprintf("    mov rax, %s\n", e.Value))
		}
		out.WriteString(fmt.Sprintf("    mov [global_%s], rax\n", globalName))
	case *hir.HIRCallExpr:
		paramRegs := []string{"rcx", "rdx", "r8", "r9"}
		for i, arg := range e.Args {
			if i < len(paramRegs) {
				if lit, ok := arg.(*hir.HIRLiteral); ok {
					if intVal, err := parseAnyInteger(lit.Value); err == nil {
						out.WriteString(fmt.Sprintf("    mov %s, %d\n", paramRegs[i], intVal))
					} else {
						out.WriteString(fmt.Sprintf("    mov %s, %s\n", paramRegs[i], lit.Value))
					}
				}
			}
		}
		targetName := b.functionLabel(e.FuncName)
		out.WriteString(fmt.Sprintf("    call %s\n", targetName))
		out.WriteString(fmt.Sprintf("    mov [global_%s], rax\n", globalName))
	}
}