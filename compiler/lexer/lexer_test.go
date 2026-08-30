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

func TestNewKeywordsAndTokens(t *testing.T) {
	input := `select { case x = ch.recv(): default: } for item in items { let d: []dyn Drawable = true && false }`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{SELECT, "select"},
		{LBRACE, "{"},
		{CASE, "case"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{IDENT, "ch"},
		{DOT, "."},
		{IDENT, "recv"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{COLON, ":"},
		{DEFAULT, "default"},
		{COLON, ":"},
		{RBRACE, "}"},
		{FOR, "for"},
		{IDENT, "item"},
		{IN, "in"},
		{IDENT, "items"},
		{LBRACE, "{"},
		{LET, "let"},
		{IDENT, "d"},
		{COLON, ":"},
		{LBRACKET, "["},
		{RBRACKET, "]"},
		{DYN, "dyn"},
		{IDENT, "Drawable"},
		{ASSIGN, "="},
		{TRUE, "true"},
		{AND, "&&"},
		{FALSE, "false"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := New(input, "test_keywords.cu")

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

func TestLifetimesAndCharLiterals(t *testing.T) {
	input := `fn longer<'a, 'static, 'ctx>(x: &'a string, y: &'static string) -> char { let c = 'z'; let esc = '\n'; let quote = '\''; return c }`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{FN, "fn"},
		{IDENT, "longer"},
		{LT, "<"},
		{LIFETIME, "a"},
		{COMMA, ","},
		{LIFETIME, "static"},
		{COMMA, ","},
		{LIFETIME, "ctx"},
		{GT, ">"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COLON, ":"},
		{BITAND, "&"},
		{LIFETIME, "a"},
		{IDENT, "string"},
		{COMMA, ","},
		{IDENT, "y"},
		{COLON, ":"},
		{BITAND, "&"},
		{LIFETIME, "static"},
		{IDENT, "string"},
		{RPAREN, ")"},
		{ARROW, "->"},
		{IDENT, "char"},
		{LBRACE, "{"},
		{LET, "let"},
		{IDENT, "c"},
		{ASSIGN, "="},
		{CHAR, "z"},
		{ILLEGAL, ";"},
		{LET, "let"},
		{IDENT, "esc"},
		{ASSIGN, "="},
		{CHAR, "\\n"},
		{ILLEGAL, ";"},
		{LET, "let"},
		{IDENT, "quote"},
		{ASSIGN, "="},
		{CHAR, "\\'"},
		{ILLEGAL, ";"},
		{RETURN, "return"},
		{IDENT, "c"},
		{RBRACE, "}"},
	}

	l := New(input, "test_lifetimes.cu")

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q (%s), got=%q (%s) (literal: %q)",
				i, tt.expectedType, tt.expectedType.String(), tok.Type, tok.Type.String(), tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestCompoundBitwiseAssignments(t *testing.T) {
	input := `a &= b; c |= d; e ^= f; g <<= 2; h >>= 3`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IDENT, "a"},
		{AND_ASSIGN, "&="},
		{IDENT, "b"},
		{ILLEGAL, ";"},
		{IDENT, "c"},
		{OR_ASSIGN, "|="},
		{IDENT, "d"},
		{ILLEGAL, ";"},
		{IDENT, "e"},
		{XOR_ASSIGN, "^="},
		{IDENT, "f"},
		{ILLEGAL, ";"},
		{IDENT, "g"},
		{SHL_ASSIGN, "<<="},
		{INT, "2"},
		{ILLEGAL, ";"},
		{IDENT, "h"},
		{SHR_ASSIGN, ">>="},
		{INT, "3"},
		{EOF, ""},
	}

	l := New(input, "test_bitwise.cu")

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q (%s), got=%q (%s)",
				i, tt.expectedType, tt.expectedType.String(), tok.Type, tok.Type.String())
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNestedCommentsAndMultilineStrings(t *testing.T) {
	input := `/* Outer comment
    /* Inner nested comment */
   still outer comment */
let multiline = "Line 1
Line 2
Line 3"
let next_token = 42
`
	l := New(input, "test_nested.cu")

	tok1 := l.NextToken() // let
	if tok1.Type != LET || tok1.Line != 4 {
		t.Fatalf("expected LET on line 4, got %v on line %d", tok1.Type, tok1.Line)
	}

	tok2 := l.NextToken() // multiline
	if tok2.Type != IDENT {
		t.Fatalf("expected IDENT, got %v", tok2.Type)
	}

	tok3 := l.NextToken() // =
	if tok3.Type != ASSIGN {
		t.Fatalf("expected ASSIGN, got %v", tok3.Type)
	}

	tok4 := l.NextToken() // string
	if tok4.Type != STRING {
		t.Fatalf("expected STRING, got %v", tok4.Type)
	}

	tok5 := l.NextToken() // let
	if tok5.Type != LET || tok5.Line != 7 {
		t.Fatalf("expected LET on line 7, got %v on line %d", tok5.Type, tok5.Line)
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
