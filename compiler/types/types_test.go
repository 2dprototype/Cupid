package types

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
	"cupid/compiler/modules"
	"cupid/compiler/parser"
	"cupid/compiler/resolver"
	"os"
	"path/filepath"
	"testing"
)

func TestTypeChecker_BasicSuccess(t *testing.T) {
	input := `
fn add(a: i32, b: i32) -> i32 {
	mut x: i32 = a + b
	return x
}
`
	l := lexer.New(input, "test.cu")
	p := parser.New(l, input)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}

	mod := &modules.Module{
		Path:    "test.cu",
		Name:    "test",
		AST:     prog,
		Exports: make(map[string]ast.Decl),
	}

	mods := map[string]*modules.Module{"test.cu": mod}
	res := resolver.NewResolver(mods)
	if !res.ResolveAll() {
		t.Fatalf("resolver failed: %v", res.Errors())
	}

	tc := NewTypeChecker(mods, res.Resolutions)
	success := tc.TypeCheckAll()
	if !success {
		t.Fatalf("type checker failed: %v", tc.Errors())
	}
}

func TestTypeChecker_TypeMismatch(t *testing.T) {
	input := `
fn bad() {
	mut x: i32 = "hello"
}
`
	l := lexer.New(input, "test.cu")
	p := parser.New(l, input)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.Errors())
	}

	mod := &modules.Module{
		Path:    "test.cu",
		Name:    "test",
		AST:     prog,
		Exports: make(map[string]ast.Decl),
	}

	mods := map[string]*modules.Module{"test.cu": mod}
	res := resolver.NewResolver(mods)
	res.ResolveAll()

	tc := NewTypeChecker(mods, res.Resolutions)
	success := tc.TypeCheckAll()
	if success {
		t.Fatal("expected type mismatch error, got success")
	}

	foundE401 := false
	for _, err := range tc.Errors() {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E401" {
			foundE401 = true
		}
	}

	if !foundE401 {
		t.Errorf("expected E401 error, got errors: %v", tc.Errors())
	}
}

func TestTypeChecker_MonomorphizationFunc(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_types_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := `
fn identity<T>(x: T) -> T {
	return x
}

fn main() {
	mut val: i32 = identity<i32>(42)
}
`
	rootPath := filepath.Join(tmpDir, "main.cu")
	if err := os.WriteFile(rootPath, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := modules.NewModuleResolver("")
	_, err = mr.ResolveGraph(rootPath)
	if err != nil {
		t.Fatalf("unexpected error resolving module graph: %v", err)
	}

	res := resolver.NewResolver(mr.Modules())
	if !res.ResolveAll() {
		t.Fatalf("resolver failed: %v", res.Errors())
	}

	tc := NewTypeChecker(mr.Modules(), res.Resolutions)
	success := tc.TypeCheckAll()
	if !success {
		t.Fatalf("type checker failed: %v", tc.Errors())
	}

	// Verify that the function was specialized
	var mainMod *modules.Module
	for _, m := range mr.Modules() {
		mainMod = m
		break
	}

	// There should be a specialized declaration in the AST now
	foundSpecialized := false
	var specializedDecl *ast.FuncDecl
	for _, decl := range mainMod.AST.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name == "identity__i32" {
			foundSpecialized = true
			specializedDecl = fd
			break
		}
	}

	if !foundSpecialized {
		t.Fatal("expected specialized function 'identity__i32' to be added to AST")
	}

	// Check that the parameter in specializedDecl has type i32
	if specializedDecl.Params[0].Type.String() != "i32" {
		t.Errorf("expected specialized parameter type 'i32', got %q", specializedDecl.Params[0].Type.String())
	}
	if specializedDecl.ReturnType.String() != "i32" {
		t.Errorf("expected specialized return type 'i32', got %q", specializedDecl.ReturnType.String())
	}
}

func TestTypeChecker_MonomorphizationStruct(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_types_test_struct")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := `
struct Box<T> {
	value T
}

fn main() {
	mut b: Box<i32> = Box<i32>{ value: 42 }
}
`
	rootPath := filepath.Join(tmpDir, "main.cu")
	if err := os.WriteFile(rootPath, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := modules.NewModuleResolver("")
	_, err = mr.ResolveGraph(rootPath)
	if err != nil {
		t.Logf("Errors in module resolver:")
		for _, e := range mr.Errors() {
			t.Logf("  %v", e)
		}
		t.Fatalf("unexpected error resolving module graph: %v", err)
	}

	res := resolver.NewResolver(mr.Modules())
	if !res.ResolveAll() {
		t.Fatalf("resolver failed: %v", res.Errors())
	}

	tc := NewTypeChecker(mr.Modules(), res.Resolutions)
	success := tc.TypeCheckAll()
	if !success {
		t.Fatalf("type checker failed: %v", tc.Errors())
	}

	var mainMod *modules.Module
	for _, m := range mr.Modules() {
		mainMod = m
		break
	}

	// Verify specialized StructDecl
	foundSpecialized := false
	var specializedStruct *ast.StructDecl
	for _, decl := range mainMod.AST.Decls {
		if sd, ok := decl.(*ast.StructDecl); ok && sd.Name == "Box__i32" {
			foundSpecialized = true
			specializedStruct = sd
			break
		}
	}

	if !foundSpecialized {
		t.Fatal("expected specialized struct 'Box__i32' to be added to AST")
	}

	if specializedStruct.Fields[0].Type.String() != "i32" {
		t.Errorf("expected specialized field type 'i32', got %q", specializedStruct.Fields[0].Type.String())
	}
}
