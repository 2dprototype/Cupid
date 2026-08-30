package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	EOF TokenType = iota
	ILLEGAL

	// Identifiers & Literals
	IDENT
	INT
	FLOAT
	STRING
	CHAR
	LIFETIME

	// Keywords
	FN
	STRUCT
	IMPL
	TRAIT
	LET
	MUT
	CONST
	GO
	CHANNEL
	SELECT
	CASE
	DEFAULT
	IN
	DYN
	TRUE
	FALSE
	UNSAFE
	ASM
	MATCH
	IMPORT
	EXPORT
	AS
	FROM
	IF
	ELSE
	FOR
	RETURN
	WHERE
	BREAK
	CONTINUE

	// Operators & Symbols
	ADD          // +
	SUB          // -
	MUL          // *
	DIV          // /
	MOD          // %
	EQ           // ==
	NEQ          // !=
	LT           // <
	LTE          // <=
	GT           // >
	GTE          // >=
	ASSIGN       // =
	ADD_ASSIGN   // +=
	SUB_ASSIGN   // -=
	MUL_ASSIGN   // *=
	DIV_ASSIGN   // /=
	MOD_ASSIGN   // %=
	AND_ASSIGN   // &=
	OR_ASSIGN    // |=
	XOR_ASSIGN   // ^=
	SHL_ASSIGN   // <<=
	SHR_ASSIGN   // >>=
	AND          // &&
	OR           // ||
	NOT          // !
	BITAND       // &
	BITOR        // |
	BITXOR       // ^
	SHL          // <<
	SHR          // >>
	ARROW        // ->
	FAT_ARROW    // =>
	QUESTION     // ?
	DOT          // .
	DOTDOT       // ..
	COLON        // :
	DOUBLE_COLON // ::
	COMMA        // ,
	AT           // @

	// Delimiters
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]
)

var tokenNames = map[TokenType]string{
	EOF:        "EOF",
	ILLEGAL:    "ILLEGAL",
	IDENT:      "IDENT",
	INT:        "INT",
	FLOAT:      "FLOAT",
	STRING:     "STRING",
	CHAR:       "CHAR",
	LIFETIME:   "LIFETIME",
	FN:         "fn",
	STRUCT:     "struct",
	IMPL:       "impl",
	TRAIT:      "trait",
	LET:        "let",
	MUT:        "mut",
	CONST:      "const",
	GO:         "go",
	CHANNEL:    "channel",
	SELECT:     "select",
	CASE:       "case",
	DEFAULT:    "default",
	IN:         "in",
	DYN:        "dyn",
	TRUE:       "true",
	FALSE:      "false",
	UNSAFE:     "unsafe",
	ASM:        "asm",
	MATCH:      "match",
	IMPORT:     "import",
	EXPORT:     "export",
	AS:         "as",
	FROM:       "from",
	IF:         "if",
	ELSE:       "else",
	FOR:        "for",
	RETURN:     "return",
	WHERE:      "where",
	BREAK:      "break",
	CONTINUE:   "continue",

	ADD:        "+",
	SUB:        "-",
	MUL:        "*",
	DIV:        "/",
	MOD:        "%",
	EQ:         "==",
	NEQ:        "!=",
	LT:         "<",
	LTE:        "<=",
	GT:         ">",
	GTE:        ">=",
	ASSIGN:     "=",
	ADD_ASSIGN: "+=",
	SUB_ASSIGN: "-=",
	MUL_ASSIGN: "*=",
	DIV_ASSIGN: "/=",
	MOD_ASSIGN: "%=",
	AND_ASSIGN: "&=",
	OR_ASSIGN:  "|=",
	XOR_ASSIGN: "^=",
	SHL_ASSIGN: "<<=",
	SHR_ASSIGN: ">>=",
	AND:        "&&",
	OR:         "||",
	NOT:        "!",
	BITAND:     "&",
	BITOR:      "|",
	BITXOR:     "^",
	SHL:        "<<",
	SHR:        ">>",
	ARROW:      "->",
	FAT_ARROW:  "=>",
	QUESTION:   "?",
	DOT:        ".",
	DOTDOT:     "..",
	COLON:      ":",
	DOUBLE_COLON: "::",
	COMMA:      ",",
	AT:         "@",
	LPAREN:     "(",
	RPAREN:     ")",
	LBRACE:     "{",
	RBRACE:     "}",
	LBRACKET:   "[",
	RBRACKET:   "]",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Token(%d)", t)
}

type Token struct {
	Type    TokenType
	Literal string
	File    string
	Line    int
	Col     int
}

var keywords = map[string]TokenType{
	"fn":       FN,
	"struct":   STRUCT,
	"impl":     IMPL,
	"trait":    TRAIT,
	"let":      LET,
	"mut":      MUT,
	"const":    CONST,
	"go":       GO,
	"channel":  CHANNEL,
	"select":   SELECT,
	"case":     CASE,
	"default":  DEFAULT,
	"in":       IN,
	"dyn":      DYN,
	"true":     TRUE,
	"false":    FALSE,
	"unsafe":   UNSAFE,
	"asm":      ASM,
	"match":    MATCH,
	"import":   IMPORT,
	"export":   EXPORT,
	"as":       AS,
	"from":     FROM,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"return":   RETURN,
	"where":    WHERE,
	"break":    BREAK,
	"continue": CONTINUE,
}

type Lexer struct {
	input    []rune
	inputStr string
	file     string
	pos      int  // current character position
	readPos  int  // current reading position (after current char)
	ch       rune // current char
	line     int
	col      int
}

func New(input string, file string) *Lexer {
	l := &Lexer{
		input:    []rune(input),
		inputStr: input,
		file:     file,
		line:     1,
		col:      0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekNChar(n int) rune {
	idx := l.readPos + n - 1
	if idx >= len(l.input) || idx < 0 {
		return 0
	}
	return l.input[idx]
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	line := l.line
	col := l.col

	var tok Token
	tok.File = l.file
	tok.Line = line
	tok.Col = col

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	case '+':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = ADD_ASSIGN
			tok.Literal = "+="
		} else {
			tok.Type = ADD
			tok.Literal = "+"
		}
	case '-':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = SUB_ASSIGN
			tok.Literal = "-="
		} else if l.peekChar() == '>' {
			l.readChar()
			tok.Type = ARROW
			tok.Literal = "->"
		} else {
			tok.Type = SUB
			tok.Literal = "-"
		}
	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = MUL_ASSIGN
			tok.Literal = "*="
		} else {
			tok.Type = MUL
			tok.Literal = "*"
		}
	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = DIV_ASSIGN
			tok.Literal = "/="
		} else {
			tok.Type = DIV
			tok.Literal = "/"
		}
	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = MOD_ASSIGN
			tok.Literal = "%="
		} else {
			tok.Type = MOD
			tok.Literal = "%"
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = EQ
			tok.Literal = "=="
		} else if l.peekChar() == '>' {
			l.readChar()
			tok.Type = FAT_ARROW
			tok.Literal = "=>"
		} else {
			tok.Type = ASSIGN
			tok.Literal = "="
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = NEQ
			tok.Literal = "!="
		} else {
			tok.Type = NOT
			tok.Literal = "!"
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = LTE
			tok.Literal = "<="
		} else if l.peekChar() == '<' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok.Type = SHL_ASSIGN
				tok.Literal = "<<="
			} else {
				tok.Type = SHL
				tok.Literal = "<<"
			}
		} else {
			tok.Type = LT
			tok.Literal = "<"
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = GTE
			tok.Literal = ">="
		} else if l.peekChar() == '>' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok.Type = SHR_ASSIGN
				tok.Literal = ">>="
			} else {
				tok.Type = SHR
				tok.Literal = ">>"
			}
		} else {
			tok.Type = GT
			tok.Literal = ">"
		}
	case '&':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = AND_ASSIGN
			tok.Literal = "&="
		} else if l.peekChar() == '&' {
			l.readChar()
			tok.Type = AND
			tok.Literal = "&&"
		} else {
			tok.Type = BITAND
			tok.Literal = "&"
		}
	case '|':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = OR_ASSIGN
			tok.Literal = "|="
		} else if l.peekChar() == '|' {
			l.readChar()
			tok.Type = OR
			tok.Literal = "||"
		} else {
			tok.Type = BITOR
			tok.Literal = "|"
		}
	case '^':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = XOR_ASSIGN
			tok.Literal = "^="
		} else {
			tok.Type = BITXOR
			tok.Literal = "^"
		}
	case '?':
		tok.Type = QUESTION
		tok.Literal = "?"
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok.Type = DOTDOT
			tok.Literal = ".."
		} else {
			tok.Type = DOT
			tok.Literal = "."
		}
	case ':':
		if l.peekChar() == ':' {
			l.readChar()
			tok.Type = DOUBLE_COLON
			tok.Literal = "::"
		} else {
			tok.Type = COLON
			tok.Literal = ":"
		}
	case ',':
		tok.Type = COMMA
		tok.Literal = ","
	case '@':
		tok.Type = AT
		tok.Literal = "@"
	case '(':
		tok.Type = LPAREN
		tok.Literal = "("
	case ')':
		tok.Type = RPAREN
		tok.Literal = ")"
	case '{':
		tok.Type = LBRACE
		tok.Literal = "{"
	case '}':
		tok.Type = RBRACE
		tok.Literal = "}"
	case '[':
		tok.Type = LBRACKET
		tok.Literal = "["
	case ']':
		tok.Type = RBRACKET
		tok.Literal = "]"
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		return tok
	case '\'':
		if l.peekChar() == '\\' {
			// Escaped char literal e.g. '\n', '\'', '\\'
			tok.Type = CHAR
			tok.Literal = l.readCharLiteral()
			return tok
		} else if isLetter(l.peekChar()) {
			// Single char literal like 'a' vs Lifetime like 'a or 'static
			if l.peekNChar(2) == '\'' {
				tok.Type = CHAR
				tok.Literal = l.readCharLiteral()
				return tok
			}
			// Multi-char or unclosed single quote followed by ident -> LIFETIME
			tok.Type = LIFETIME
			tok.Literal = l.readLifetime()
			return tok
		} else {
			// Other char literal e.g. ';', ' ', or empty
			tok.Type = CHAR
			tok.Literal = l.readCharLiteral()
			return tok
		}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			if tokType, ok := keywords[tok.Literal]; ok {
				tok.Type = tokType
				if tokType == ASM {
					l.skipWhitespaceAndComments()
					if l.ch == '{' {
						l.readChar()
						tok.Literal = l.ReadRawUntilMatchingBrace()
						if l.ch == '}' {
							l.readChar()
						}
					}
				}
			} else {
				tok.Type = IDENT
			}
			return tok
		} else if isDigit(l.ch) {
			literal, isFloat := l.readNumber()
			if isFloat {
				tok.Type = FLOAT
			} else {
				tok.Type = INT
			}
			tok.Literal = literal
			return tok
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
			l.readChar()
		} else if l.ch == '/' && l.peekChar() == '/' {
			// Single-line comment
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
		} else if l.ch == '/' && l.peekChar() == '*' {
			// Multi-line comment (supports nested comments)
			l.readChar() // consume '/'
			l.readChar() // consume '*'
			nesting := 1
			for nesting > 0 && l.ch != 0 {
				if l.ch == '/' && l.peekChar() == '*' {
					nesting++
					l.readChar() // consume '/'
					l.readChar() // consume '*'
				} else if l.ch == '*' && l.peekChar() == '/' {
					nesting--
					l.readChar() // consume '*'
					l.readChar() // consume '/'
				} else {
					if l.ch == '\n' {
						l.line++
						l.col = 0
					}
					l.readChar()
				}
			}
		} else {
			break
		}
	}
}

func (l *Lexer) readIdentifier() string {
	startPos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[startPos:l.pos])
}

func (l *Lexer) readLifetime() string {
	l.readChar() // consume initial single quote '\''
	startPos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[startPos:l.pos])
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isBinaryDigit(ch rune) bool {
	return ch == '0' || ch == '1'
}

func isOctalDigit(ch rune) bool {
	return ch >= '0' && ch <= '7'
}

func (l *Lexer) readNumber() (string, bool) {
	startPos := l.pos
	isFloat := false

	if l.ch == '0' {
		peek := l.peekChar()
		if peek == 'x' || peek == 'X' {
			l.readChar() // consume '0'
			l.readChar() // consume 'x'/'X'
			for isHexDigit(l.ch) || l.ch == '_' {
				l.readChar()
			}
			return string(l.input[startPos:l.pos]), false
		} else if peek == 'b' || peek == 'B' {
			l.readChar() // consume '0'
			l.readChar() // consume 'b'/'B'
			for isBinaryDigit(l.ch) || l.ch == '_' {
				l.readChar()
			}
			return string(l.input[startPos:l.pos]), false
		} else if peek == 'o' || peek == 'O' {
			l.readChar() // consume '0'
			l.readChar() // consume 'o'/'O'
			for isOctalDigit(l.ch) || l.ch == '_' {
				l.readChar()
			}
			return string(l.input[startPos:l.pos]), false
		}
	}

	for isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume '.'
		for isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		peek := l.peekChar()
		if isDigit(peek) || ((peek == '+' || peek == '-') && isDigit(l.peekNChar(2))) {
			isFloat = true
			l.readChar() // consume 'e'/'E'
			if l.ch == '+' || l.ch == '-' {
				l.readChar()
			}
			for isDigit(l.ch) || l.ch == '_' {
				l.readChar()
			}
		}
	}
	return string(l.input[startPos:l.pos]), isFloat
}

func (l *Lexer) readString() string {
	l.readChar() // consume initial double quote
	startPos := l.pos
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // consume escape prefix '\'
			if l.ch != 0 {
				if l.ch == '\n' {
					l.line++
					l.col = 0
				}
				l.readChar() // consume escaped char
			}
		} else {
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
			l.readChar()
		}
	}
	val := string(l.input[startPos:l.pos])
	if l.ch == '"' {
		l.readChar() // consume closing quote
	}
	return val
}

func (l *Lexer) readCharLiteral() string {
	l.readChar() // consume initial single quote
	startPos := l.pos
	if l.ch == '\\' {
		l.readChar() // consume escape prefix '\'
		if l.ch != 0 {
			l.readChar() // consume escaped character
		}
	} else if l.ch != 0 && l.ch != '\'' {
		l.readChar() // consume character
	}
	val := string(l.input[startPos:l.pos])
	if l.ch == '\'' {
		l.readChar() // consume closing quote
	}
	return val
}

// ReadRawUntilMatchingBrace parses raw assembly contents.
// It is called immediately after consuming the 'asm {' sequence.
func (l *Lexer) ReadRawUntilMatchingBrace() string {
	braceCount := 1
	startPos := l.pos

	for braceCount > 0 && l.ch != 0 {
		if l.ch == '{' {
			braceCount++
		} else if l.ch == '}' {
			braceCount--
			if braceCount == 0 {
				break
			}
		} else if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.readChar()
	}

	raw := string(l.input[startPos:l.pos])
	return raw
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

// UTF8 validation check
func IsValidUTF8(str string) bool {
	return utf8.ValidString(str)
}
