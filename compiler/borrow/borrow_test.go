package borrow

import (
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
	"cupid/compiler/modules"
	"cupid/compiler/parser"
	"cupid/compiler/resolver"
	"cupid/compiler/types"
	"testing"
)

func TestBorrowChecker_Success(t *testing.T) {
	input := `
struct Player {
	name string
}

fn main() {
	mut p = Player{ name: "hero" }
	let r1 = &p
	let r2 = &p
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

	tc := types.NewTypeChecker(mods, res.Resolutions)
	tc.TypeCheckAll()

	bc := NewBorrowChecker(mods, res.Resolutions, tc.ExprTypes)
	success := bc.BorrowCheckAll()
	if !success {
		t.Fatalf("borrow checker failed: %v", bc.Errors())
	}
}

func TestBorrowChecker_MutableConflict(t *testing.T) {
	input := `
struct Player {
	name string
}

fn main() {
	mut p = Player{ name: "hero" }
	let r1 = &mut p
	let r2 = &p
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

	tc := types.NewTypeChecker(mods, res.Resolutions)
	tc.TypeCheckAll()

	bc := NewBorrowChecker(mods, res.Resolutions, tc.ExprTypes)
	success := bc.BorrowCheckAll()
	if success {
		t.Fatal("expected borrow conflict, got success")
	}

	foundE201 := false
	for _, err := range bc.Errors() {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E201" {
			foundE201 = true
		}
	}

	if !foundE201 {
		t.Errorf("expected E201 error, got: %v", bc.Errors())
	}
}

func TestBorrowChecker_UseAfterMove(t *testing.T) {
	input := `
struct Player {
	name string
}

fn consume(p: Player) {}

fn main() {
	let p = Player{ name: "hero" }
	consume(p)
	let name = p.name
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

	tc := types.NewTypeChecker(mods, res.Resolutions)
	tc.TypeCheckAll()

	bc := NewBorrowChecker(mods, res.Resolutions, tc.ExprTypes)
	success := bc.BorrowCheckAll()
	if success {
		t.Fatal("expected use-after-move error, got success")
	}

	foundE202 := false
	for _, err := range bc.Errors() {
		if diag, ok := err.(diagnostics.Diagnostic); ok && diag.Code == "E202" {
			foundE202 = true
		}
	}

	if !foundE202 {
		t.Errorf("expected E202 error, got: %v", bc.Errors())
	}
}

func TestBorrowChecker_ReassignAfterMove(t *testing.T) {
	input := `
struct Player {
	name string
}

fn consume(p: Player) {}

fn main() {
	mut p = Player{ name: "hero" }
	consume(p)
	p = Player{ name: "new" }
	let name = p.name // should be fine now!
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

	tc := types.NewTypeChecker(mods, res.Resolutions)
	tc.TypeCheckAll()

	bc := NewBorrowChecker(mods, res.Resolutions, tc.ExprTypes)
	success := bc.BorrowCheckAll()
	if !success {
		t.Fatalf("expected re-assignment to resolve move, but got errors: %v", bc.Errors())
	}
}
