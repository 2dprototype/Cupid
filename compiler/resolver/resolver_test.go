package resolver

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
	"cupid/compiler/modules"
	"cupid/compiler/parser"
	"os"
	"path/filepath"
	"testing"
)

func TestResolver_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_resolver_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	rootContent := `
struct Player {
	name string
	hp i32
}

fn add(a: i32, b: i32) -> i32 {
	mut x: i32 = a + b
	return x
}

fn main() {
	mut p: Player = Player{ name: "hero", hp: 100 }
	mut score = add(10, 20)
}
`
	rootPath := filepath.Join(tmpDir, "main.cu")
	if err := os.WriteFile(rootPath, []byte(rootContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := modules.NewModuleResolver("")
	_, err = mr.ResolveGraph(rootPath)
	if err != nil {
		t.Fatalf("unexpected error resolving module graph: %v", err)
	}

	res := NewResolver(mr.Modules())
	success := res.ResolveAll()
	if !success {
		t.Fatalf("resolver failed with errors: %v", res.errors)
	}

	// Verify that variable usages are resolved
	// Specifically, check that 'a' and 'b' in 'a + b' resolve to their parameter definitions
	var mainMod *modules.Module
	for _, m := range mr.Modules() {
		mainMod = m
		break
	}

	// Find the 'add' function and trace its resolutions
	var addFunc *ast.FuncDecl
	for _, decl := range mainMod.AST.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name == "add" {
			addFunc = fd
			break
		}
	}

	if addFunc == nil {
		t.Fatal("expected 'add' function to be found")
	}

	// The block has let stmt: 'mut x = a + b'
	letStmt := addFunc.Body.Stmts[0].(*ast.LetStmt)
	binExpr := letStmt.Value.(*ast.BinaryExpr)
	leftIdent := binExpr.Left.(*ast.IdentExpr)
	rightIdent := binExpr.Right.(*ast.IdentExpr)

	resolvedLeft := res.Resolutions[leftIdent]
	resolvedRight := res.Resolutions[rightIdent]

	if resolvedLeft == nil {
		t.Error("expected left identifier 'a' to be resolved")
	}
	if resolvedRight == nil {
		t.Error("expected right identifier 'b' to be resolved")
	}
}

func TestResolver_UnresolvedName(t *testing.T) {
	l := lexer.New("fn main() { x = 10 }", "test.cu")
	p := parser.New(l, "fn main() { x = 10 }")
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
	res := NewResolver(mods)
	success := res.ResolveAll()
	if success {
		t.Fatal("expected resolver to fail due to unresolved name 'x'")
	}

	foundE301 := false
	for _, err := range res.errors {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E301" {
			foundE301 = true
		}
	}

	if !foundE301 {
		t.Errorf("expected E301 error code, got errors: %v", res.errors)
	}
}

func TestResolver_DuplicateDeclaration(t *testing.T) {
	input := `
fn add() {}
fn add() {}
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
	res := NewResolver(mods)
	success := res.ResolveAll()
	if success {
		t.Fatal("expected resolver to fail due to duplicate declarations")
	}

	foundE300 := false
	for _, err := range res.errors {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E300" {
			foundE300 = true
		}
	}

	if !foundE300 {
		t.Errorf("expected E300 error code, got errors: %v", res.errors)
	}
}

func TestResolver_UnresolvedType(t *testing.T) {
	input := `
fn main() {
	mut p: UnknownStruct = 0
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
	res := NewResolver(mods)
	success := res.ResolveAll()
	if success {
		t.Fatal("expected resolver to fail due to unresolved type")
	}

	foundE304 := false
	for _, err := range res.errors {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E304" {
			foundE304 = true
		}
	}

	if !foundE304 {
		t.Errorf("expected E304 error code, got errors: %v", res.errors)
	}
}
