package modules

import (
	"cupid/compiler/diagnostics"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleResolver_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_modules_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create root file
	rootContent := `
import "math"
import { Player } from "entities"

fn main() {
	mut p = Player{ name: "hero", hp: 100 }
	math.sin(10)
}
`
	// Create math.cu (simulated stdlib module)
	mathContent := `
export fn sin(x: f64) -> f64 {
	return x // dummy implementation
}
`
	// Create entities.cu
	entitiesContent := `
export struct Player {
	name string
	hp i32
}
`

	rootPath := filepath.Join(tmpDir, "main.cu")
	mathPath := filepath.Join(tmpDir, "math.cu")
	entitiesPath := filepath.Join(tmpDir, "entities.cu")

	if err := os.WriteFile(rootPath, []byte(rootContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := os.WriteFile(mathPath, []byte(mathContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := os.WriteFile(entitiesPath, []byte(entitiesContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := NewModuleResolver(tmpDir) // Use tmpDir as stdlibDir for test
	rootMod, err := mr.ResolveGraph(rootPath)
	if err != nil {
		t.Logf("Errors in resolver:")
		for _, e := range mr.errors {
			t.Logf("  %v", e)
		}
		t.Fatalf("unexpected error resolving graph: %v", err)
	}

	if rootMod == nil {
		t.Fatal("rootMod should not be nil")
	}

	if len(mr.order) != 3 {
		t.Errorf("expected 3 modules resolved, got %d", len(mr.order))
	}

	// Verify order: math and entities should be compiled before main
	mainCanonical, _ := mr.canonicalizePath(rootPath)
	if mr.order[2] != mainCanonical {
		t.Errorf("expected main to be last, got %v", mr.order)
	}
}

func TestModuleResolver_CircularDependency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_modules_circular")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	aContent := `import "b"`
	bContent := `import "a"`

	aPath := filepath.Join(tmpDir, "a.cu")
	bPath := filepath.Join(tmpDir, "b.cu")

	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := NewModuleResolver(tmpDir)
	_, err = mr.ResolveGraph(aPath)
	if err == nil {
		t.Fatal("expected cyclic dependency error, got nil")
	}

	foundE602 := false
	for _, e := range mr.errors {
		if diag, ok := e.(diagnostics.Diagnostic); ok && diag.Code == "E602" {
			foundE602 = true
			if !strings.Contains(diag.Message, "circular import") {
				t.Errorf("expected circular import error message, got %q", diag.Message)
			}
		}
	}

	if !foundE602 {
		t.Fatalf("expected E602 error code in diagnostics, got errors: %v", mr.errors)
	}
}

func TestModuleResolver_PrivateSymbol(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_modules_private")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	aContent := `import { secret } from "b"`
	bContent := `fn secret() {}`

	aPath := filepath.Join(tmpDir, "a.cu")
	bPath := filepath.Join(tmpDir, "b.cu")

	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := NewModuleResolver(tmpDir)
	_, err = mr.ResolveGraph(aPath)
	if err == nil {
		t.Fatal("expected private symbol error, got nil")
	}

	foundE601 := false
	for _, e := range mr.errors {
		if diag, ok := e.(diagnostics.Diagnostic); ok && diag.Code == "E601" {
			foundE601 = true
			if !strings.Contains(diag.Message, "is private to module") {
				t.Errorf("expected private visibility message, got %q", diag.Message)
			}
		}
	}

	if !foundE601 {
		t.Fatalf("expected E601 error code in diagnostics, got errors: %v", mr.errors)
	}
}

func TestModuleResolver_MissingModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cupid_modules_missing")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	aContent := `import "non_existent"`
	aPath := filepath.Join(tmpDir, "a.cu")

	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	mr := NewModuleResolver(tmpDir)
	_, err = mr.ResolveGraph(aPath)
	if err == nil {
		t.Fatal("expected missing module error, got nil")
	}

	foundE603 := false
	for _, e := range mr.errors {
		if diag, ok := e.(diagnostics.Diagnostic); ok && diag.Code == "E603" {
			foundE603 = true
		}
	}

	if !foundE603 {
		t.Fatalf("expected E603 error code in diagnostics, got errors: %v", mr.errors)
	}
}
