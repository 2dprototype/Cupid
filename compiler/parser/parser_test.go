package parser

import (
	"testing"
	"cupid/compiler/lexer"
)

func TestParseProgram(t *testing.T) {
	input := `import "math"

export struct Player {
	name string
	hp i32
}

fn add(a: i32, b: i32) -> i32 {
	mut x: i32 = a + b
	return x
}

impl Player {
	fn damage(mut self, amount: i32) {
		self.hp -= amount
	}
}

fn main() {
	asm {
		mov rax, 10
	}
	mut score = 10
	match score {
		0 => {
			return
		}
	}
}
`
	l := lexer.New(input, "test.cu")
	p := New(l, input)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(prog.Decls) != 5 {
		t.Fatalf("expected 5 declarations, got %d", len(prog.Decls))
	}
}
