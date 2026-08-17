package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `let mut score = 100
const PI = 3.14
fn add(a: i32, b: i32) -> i32 {
	return a + b
}
unsafe {
	asm {
		mov rax, 10
	}
}
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{MUT, "mut"},
		{IDENT, "score"},
		{ASSIGN, "="},
		{INT, "100"},
		{CONST, "const"},
		{IDENT, "PI"},
		{ASSIGN, "="},
		{FLOAT, "3.14"},
		{FN, "fn"},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "a"},
		{COLON, ":"},
		{IDENT, "i32"},
		{COMMA, ","},
		{IDENT, "b"},
		{COLON, ":"},
		{IDENT, "i32"},
		{RPAREN, ")"},
		{ARROW, "->"},
		{IDENT, "i32"},
		{LBRACE, "{"},
		{RETURN, "return"},
		{IDENT, "a"},
		{ADD, "+"},
		{IDENT, "b"},
		{RBRACE, "}"},
		{UNSAFE, "unsafe"},
		{LBRACE, "{"},
		{ASM, "\n\t\tmov rax, 10\n\t"},
		{RBRACE, "}"},
	}

	l := New(input, "test.cu")

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal: %q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestAsmRawScanning(t *testing.T) {
	input := `asm {
	mov rax, 10
	add rax, 5
}`
	l := New(input, "test_asm.cu")
	tok := l.NextToken() // asm
	if tok.Type != ASM {
		t.Fatalf("expected ASM, got %v", tok.Type)
	}
	expectedRaw := "\n\tmov rax, 10\n\tadd rax, 5\n"
	if tok.Literal != expectedRaw {
		t.Fatalf("expected raw %q, got %q", expectedRaw, tok.Literal)
	}
}
