package driver

import (
	"cupid/compiler/backend"
	"cupid/compiler/borrow"
	"cupid/compiler/diagnostics"
	"cupid/compiler/hir"
	"cupid/compiler/mir"
	"cupid/compiler/modules"
	"cupid/compiler/optimizer"
	"cupid/compiler/resolver"
	"cupid/compiler/types"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CompileOptions struct {
	InputFile string
	OutputFile string
	StdlibDir string
	FasmDir   string
	EmitAsm   bool
	Release   bool
	Debug     bool
}

type Driver struct {
	opts CompileOptions
}

func New(opts CompileOptions) *Driver {
	return &Driver{opts: opts}
}

func (d *Driver) Compile() (string, error) {
	absInput, err := filepath.Abs(d.opts.InputFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of %q: %w", d.opts.InputFile, err)
	}

	// 1. Resolve modules & Parse
	modResolver := modules.NewModuleResolver(d.opts.StdlibDir)
	// rootMod, err := modResolver.ResolveGraph(absInput)
	_, err = modResolver.ResolveGraph(absInput)
	if err != nil {
		for _, e := range modResolver.Errors() {
			if diag, ok := e.(diagnostics.Diagnostic); ok {
				fmt.Fprintln(os.Stderr, diagnostics.FormatError(rootModSource(modResolver, diag.File), diag))
			} else {
				fmt.Fprintln(os.Stderr, e)
			}
		}
		return "", err
	}

	mods := modResolver.Modules()

	// 2. Name & Symbol Resolution
	symbolResolver := resolver.NewResolver(mods)
	if !symbolResolver.ResolveAll() {
		for _, e := range symbolResolver.Errors() {
			if diag, ok := e.(diagnostics.Diagnostic); ok {
				fmt.Fprintln(os.Stderr, diagnostics.FormatError(rootModSource(modResolver, diag.File), diag))
			} else {
				fmt.Fprintln(os.Stderr, e)
			}
		}
		return "", fmt.Errorf("name resolution failed with %d errors", len(symbolResolver.Errors()))
	}

	// 3. Type Checking
	typeChecker := types.NewTypeChecker(mods, symbolResolver.Resolutions)
	if !typeChecker.TypeCheckAll() {
		for _, e := range typeChecker.Errors() {
			if diag, ok := e.(diagnostics.Diagnostic); ok {
				fmt.Fprintln(os.Stderr, diagnostics.FormatError(rootModSource(modResolver, diag.File), diag))
			} else {
				fmt.Fprintln(os.Stderr, e)
			}
		}
		return "", fmt.Errorf("type checking failed with %d errors", len(typeChecker.Errors()))
	}

	// 4. Lower AST to HIR
	hirProg := hir.LowerAST(mods, typeChecker, symbolResolver)

	// 5. Borrow Checking
	borrowChecker := borrow.NewBorrowChecker(mods, symbolResolver.Resolutions, typeChecker.ExprTypes)
	if !borrowChecker.BorrowCheckAll() {
		for _, e := range borrowChecker.Errors() {
			if diag, ok := e.(diagnostics.Diagnostic); ok {
				fmt.Fprintln(os.Stderr, diagnostics.FormatError(rootModSource(modResolver, diag.File), diag))
			} else {
				fmt.Fprintln(os.Stderr, e)
			}
		}
		return "", fmt.Errorf("borrow checking failed with %d errors", len(borrowChecker.Errors()))
	}

	// 6. Lower HIR to MIR
	mirProg := mir.LowerHIR(hirProg)

	// 7. Optimizer
	mirProg = optimizer.Optimize(mirProg, d.opts.Release)

	// 8. Native Backend Code Generation (FASM x86-64)
	fasmDir := d.opts.FasmDir
	if fasmDir == "" {
		fasmDir = filepath.Join(filepath.Dir(absInput), "fasm")
	}

	asmCode, err := backend.GenerateAssembly(mirProg, fasmDir, d.opts.Release, d.opts.Debug)
	if err != nil {
		return "", fmt.Errorf("code generation failed: %w", err)
	}

	// Determine output paths
	baseName := strings.TrimSuffix(filepath.Base(absInput), filepath.Ext(absInput))
	dir := filepath.Dir(absInput)
	asmFile := filepath.Join(dir, baseName+".asm")
	exeFile := filepath.Join(dir, baseName+".exe")
	if d.opts.OutputFile != "" {
		exeFile = d.opts.OutputFile
	}

	// Write assembly
	if err := os.WriteFile(asmFile, []byte(asmCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write assembly file: %w", err)
	}

	// 9. Invoke FASM
	fasmExe := filepath.Join(fasmDir, "FASM.EXE")
	fasmCmd := exec.Command(fasmExe, asmFile, exeFile)
	output, err := fasmCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("assembler error: %v\n%s", err, string(output))
	}

	// Cleanup temporary asm unless emitAsm was requested
	if !d.opts.EmitAsm {
		_ = os.Remove(asmFile)
	}

	return exeFile, nil
}

func rootModSource(mr *modules.ModuleResolver, filePath string) string {
	for _, mod := range mr.Modules() {
		if mod.Path == filePath || strings.EqualFold(filepath.Clean(mod.Path), filepath.Clean(filePath)) {
			return mod.Source
		}
	}
	return ""
}
