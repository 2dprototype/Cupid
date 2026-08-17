package parser

import (
	"fmt"
	"strconv"
	"cupid/compiler/ast"
	"cupid/compiler/diagnostics"
	"cupid/compiler/lexer"
)

type Parser struct {
	l            *lexer.Lexer
	curToken     lexer.Token
	peekToken    lexer.Token
	errors       []error
	source       string
	noStructInit bool
}

func New(l *lexer.Lexer, source string) *Parser {
	p := &Parser{
		l:      l,
		source: source,
	}
	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []error {
	return p.errors
}

func (p *Parser) error(diag diagnostics.Diagnostic) {
	p.errors = append(p.errors, diag)
}

func (p *Parser) peekError(t lexer.TokenType) {
	diag := diagnostics.Diagnostic{
		Code:    "E101",
		Message: fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type),
		File:    p.peekToken.File,
		Line:    p.peekToken.Line,
		Column:  p.peekToken.Col,
		SpanLen: len(p.peekToken.Literal),
	}
	p.error(diag)
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{
		Position: ast.Position{
			File: p.curToken.File,
			Line: p.curToken.Line,
			Col:  p.curToken.Col,
		},
	}

	for !p.curTokenIs(lexer.EOF) {
		decl := p.parseDecl()
		if decl != nil {
			prog.Decls = append(prog.Decls, decl)
		}
		p.nextToken()
	}

	return prog
}

func (p *Parser) parseDecl() ast.Decl {
	exported := false
	if p.curTokenIs(lexer.EXPORT) {
		exported = true
		p.nextToken()
	}

	switch p.curToken.Type {
	case lexer.IMPORT:
		return p.parseImportDecl()
	case lexer.FN:
		return p.parseFuncDecl(exported)
	case lexer.STRUCT:
		return p.parseStructDecl(exported)
	case lexer.IMPL:
		return p.parseImplDecl()
	case lexer.TRAIT:
		return p.parseTraitDecl(exported)
	case lexer.CONST:
		return p.parseGlobalConstDecl(exported)
	default:
		diag := diagnostics.Diagnostic{
			Code:    "E102",
			Message: fmt.Sprintf("expected declaration keyword, got %s", p.curToken.Type),
			File:    p.curToken.File,
			Line:    p.curToken.Line,
			Column:  p.curToken.Col,
			SpanLen: len(p.curToken.Literal),
		}
		p.error(diag)
		return nil
	}
}

func (p *Parser) parseImportDecl() *ast.ImportDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	// import "math"
	// import "graphics" as gfx
	// import { Player, Enemy } from "entities"
	if p.peekTokenIs(lexer.STRING) {
		p.nextToken()
		path := p.curToken.Literal
		alias := ""
		if p.peekTokenIs(lexer.AS) {
			p.nextToken() // consume 'as'
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			alias = p.curToken.Literal
		}
		return &ast.ImportDecl{Position: pos, Path: path, Alias: alias}
	}

	if p.expectPeek(lexer.LBRACE) {
		// import { A, B } from "module"
		symbols := []ast.ImportSymbol{}
		for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
			p.nextToken()
			if !p.curTokenIs(lexer.IDENT) {
				p.error(diagnostics.Diagnostic{
					Code:    "E101",
					Message: fmt.Sprintf("expected identifier in import list, got %v", p.curToken.Type),
					File:    p.curToken.File,
					Line:    p.curToken.Line,
					Column:  p.curToken.Col,
				})
				return nil
			}
			name := p.curToken.Literal
			alias := ""
			if p.peekTokenIs(lexer.AS) {
				p.nextToken() // consume 'as'
				if !p.expectPeek(lexer.IDENT) {
					return nil
				}
				alias = p.curToken.Literal
			}
			symbols = append(symbols, ast.ImportSymbol{Name: name, Alias: alias})

			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
		}
		if !p.expectPeek(lexer.RBRACE) {
			return nil
		}
		if !p.expectPeek(lexer.FROM) {
			return nil
		}
		if !p.expectPeek(lexer.STRING) {
			return nil
		}
		fromModule := p.curToken.Literal
		return &ast.ImportDecl{Position: pos, Symbols: symbols, FromModule: fromModule}
	}

	return nil
}

func (p *Parser) parseFuncDecl(exported bool) *ast.FuncDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var lifetimes []string
	var generics []ast.GenericParam

	// Generics e.g. fn max<T: Comparable>(a: T)
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume '<'
		p.nextToken()
		for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.CHAR) { // wait, 'a is represented as 'a which the lexer reads as CHAR if single quote
				lifetimes = append(lifetimes, p.curToken.Literal)
			} else if p.curTokenIs(lexer.IDENT) {
				gname := p.curToken.Literal
				var bounds []string
				if p.peekTokenIs(lexer.COLON) {
					p.nextToken() // consume name, curToken becomes ':'
					for {
						if !p.expectPeek(lexer.IDENT) {
							break
						}
						bounds = append(bounds, p.curToken.Literal)
						if p.peekTokenIs(lexer.ADD) {
							p.nextToken() // consume '+'
						} else {
							break
						}
					}
				}
				generics = append(generics, ast.GenericParam{Name: gname, Bounds: bounds})
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		if !p.curTokenIs(lexer.GT) {
			return nil
		}
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	params := []ast.Param{}
	for !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		var pname string
		var ptype ast.Type
		if p.curTokenIs(lexer.MUT) && p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "self" {
			pname = "self"
			p.nextToken() // consume 'self'
			ptype = &ast.PointerType{Position: pos, To: &ast.PrimitiveType{Position: pos, Name: "self"}, Mutable: true}
		} else if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "self" {
			pname = "self"
			ptype = &ast.PrimitiveType{Position: pos, Name: "self"}
		} else if p.curTokenIs(lexer.IDENT) {
			pname = p.curToken.Literal
			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()
			ptype = p.parseType()
		} else {
			return nil
		}
		params = append(params, ast.Param{Name: pname, Type: ptype})

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	var retType ast.Type
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // consume '->'
		p.nextToken()
		retType = p.parseType()
	}

	var whereClauses []ast.WhereClause
	if p.peekTokenIs(lexer.WHERE) { // wait, WHERE is not a lexer keyword explicitly but an identifier
		p.nextToken() // consume 'where'
		for {
			if !p.expectPeek(lexer.IDENT) {
				break
			}
			pname := p.curToken.Literal
			if !p.expectPeek(lexer.COLON) {
				break
			}
			var bounds []string
			for {
				if !p.expectPeek(lexer.IDENT) {
					break
				}
				bounds = append(bounds, p.curToken.Literal)
				if p.peekTokenIs(lexer.ADD) {
					p.nextToken()
				} else {
					break
				}
			}
			whereClauses = append(whereClauses, ast.WhereClause{ParamName: pname, Bounds: bounds})
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			} else {
				break
			}
		}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	body := p.parseBlockStmt()

	return &ast.FuncDecl{
		Position:     pos,
		Exported:     exported,
		Name:         name,
		Lifetimes:    lifetimes,
		Generics:     generics,
		Params:       params,
		ReturnType:   retType,
		WhereClauses: whereClauses,
		Body:         body,
	}
}

func (p *Parser) parseStructDecl(exported bool) *ast.StructDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var lifetimes []string
	var generics []ast.GenericParam

	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // '<'
		p.nextToken()
		for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.CHAR) {
				lifetimes = append(lifetimes, p.curToken.Literal)
			} else if p.curTokenIs(lexer.IDENT) {
				gname := p.curToken.Literal
				var bounds []string
				if p.peekTokenIs(lexer.COLON) {
					p.nextToken()
					for {
						if !p.expectPeek(lexer.IDENT) {
							break
						}
						bounds = append(bounds, p.curToken.Literal)
						if p.peekTokenIs(lexer.ADD) {
							p.nextToken()
						} else {
							break
						}
					}
				}
				generics = append(generics, ast.GenericParam{Name: gname, Bounds: bounds})
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		if !p.curTokenIs(lexer.GT) {
			return nil
		}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	fields := []ast.StructField{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		// Struct fields in spec: "name string \n hp i32" (Go style: name then type)
		if !p.curTokenIs(lexer.IDENT) {
			return nil
		}
		fname := p.curToken.Literal
		p.nextToken()
		ftype := p.parseType()
		fields = append(fields, ast.StructField{Name: fname, Type: ftype})
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return &ast.StructDecl{
		Position:  pos,
		Exported:  exported,
		Name:      name,
		Lifetimes: lifetimes,
		Generics:  generics,
		Fields:    fields,
	}
}

func (p *Parser) parseImplDecl() *ast.ImplDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	var lifetimes []string
	var generics []ast.GenericParam

	if p.peekTokenIs(lexer.LT) {
		p.nextToken()
		p.nextToken()
		for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.CHAR) {
				lifetimes = append(lifetimes, p.curToken.Literal)
			} else if p.curTokenIs(lexer.IDENT) {
				gname := p.curToken.Literal
				var bounds []string
				if p.peekTokenIs(lexer.COLON) {
					p.nextToken()
					for {
						if !p.expectPeek(lexer.IDENT) {
							break
						}
						bounds = append(bounds, p.curToken.Literal)
						if p.peekTokenIs(lexer.ADD) {
							p.nextToken()
						} else {
							break
						}
					}
				}
				generics = append(generics, ast.GenericParam{Name: gname, Bounds: bounds})
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		if !p.curTokenIs(lexer.GT) {
			return nil
		}
	}

	p.nextToken()
	// Trait implementation format: "impl Drawable for Player" or "impl Player"
	firstType := p.parseType()

	var traitName string
	var targetType ast.Type

	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "for" {
		p.nextToken() // consume 'for'
		p.nextToken()
		targetType = p.parseType()
		traitName = firstType.String()
	} else {
		targetType = firstType
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	methods := []*ast.FuncDecl{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		if p.curTokenIs(lexer.FN) {
			method := p.parseFuncDecl(false)
			if method != nil {
				methods = append(methods, method)
			}
		} else {
			p.nextToken()
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return &ast.ImplDecl{
		Position:  pos,
		TraitName: traitName,
		Target:    targetType,
		Lifetimes: lifetimes,
		Generics:  generics,
		Methods:   methods,
	}
}

func (p *Parser) parseTraitDecl(exported bool) *ast.TraitDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var lifetimes []string
	var generics []ast.GenericParam

	if p.peekTokenIs(lexer.LT) {
		p.nextToken()
		for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
			p.nextToken()
			if p.curTokenIs(lexer.CHAR) {
				lifetimes = append(lifetimes, p.curToken.Literal)
			} else if p.curTokenIs(lexer.IDENT) {
				gname := p.curToken.Literal
				var bounds []string
				if p.peekTokenIs(lexer.COLON) {
					p.nextToken()
					for {
						if !p.expectPeek(lexer.IDENT) {
							break
						}
						bounds = append(bounds, p.curToken.Literal)
						if p.peekTokenIs(lexer.ADD) {
							p.nextToken()
						} else {
							break
						}
					}
				}
				generics = append(generics, ast.GenericParam{Name: gname, Bounds: bounds})
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
		}
		p.expectPeek(lexer.GT)
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	methods := []ast.TraitMethodSignature{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		if p.curTokenIs(lexer.FN) {
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			mname := p.curToken.Literal
			if !p.expectPeek(lexer.LPAREN) {
				return nil
			}
			params := []ast.Param{}
			for !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
				p.nextToken()
				pname := p.curToken.Literal
				var ptype ast.Type
				if pname == "self" {
					ptype = &ast.PrimitiveType{Position: pos, Name: "self"}
				} else {
					if !p.expectPeek(lexer.COLON) {
						return nil
					}
					p.nextToken()
					ptype = p.parseType()
				}
				params = append(params, ast.Param{Name: pname, Type: ptype})
				if p.peekTokenIs(lexer.COMMA) {
					p.nextToken()
				}
			}
			p.expectPeek(lexer.RPAREN)
			var retType ast.Type
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken()
				p.nextToken()
				retType = p.parseType()
			}
			methods = append(methods, ast.TraitMethodSignature{
				Name:       mname,
				Params:     params,
				ReturnType: retType,
			})
		}
	}

	p.expectPeek(lexer.RBRACE)

	return &ast.TraitDecl{
		Position:  pos,
		Exported:  exported,
		Name:      name,
		Lifetimes: lifetimes,
		Generics:  generics,
		Methods:   methods,
	}
}

func (p *Parser) parseGlobalConstDecl(exported bool) *ast.GlobalConstDecl {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var ftype ast.Type
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		ftype = p.parseType()
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()
	val := p.parseExpr(LOWEST)

	return &ast.GlobalConstDecl{
		Position: pos,
		Exported: exported,
		Name:     name,
		Type:     ftype,
		Value:    val,
	}
}

// ---------------- Type Parsing ----------------

func (p *Parser) parseType() ast.Type {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	// Pointer types: &T or &mut T
	if p.curTokenIs(lexer.BITAND) {
		mutable := false
		if p.peekTokenIs(lexer.MUT) {
			p.nextToken() // consume 'mut'
			mutable = true
		}
		p.nextToken()
		toType := p.parseType()
		return &ast.PointerType{Position: pos, To: toType, Mutable: mutable}
	}

	// Array and slice types: [10]T or []T
	if p.curTokenIs(lexer.LBRACKET) {
		size := -1
		if p.peekTokenIs(lexer.INT) {
			p.nextToken()
			val, _ := strconv.Atoi(p.curToken.Literal)
			size = val
		}
		p.expectPeek(lexer.RBRACKET)
		p.nextToken()
		elemType := p.parseType()
		return &ast.ArrayType{Position: pos, Element: elemType, Size: size}
	}

	// channel<T>
	if p.curTokenIs(lexer.CHANNEL) {
		if p.expectPeek(lexer.LT) {
			p.nextToken()
			valType := p.parseType()
			p.expectPeek(lexer.GT)
			return &ast.ChannelType{Position: pos, Value: valType}
		}
	}

	// dyn Trait
	if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "dyn" {
		p.nextToken()
		return &ast.DynTraitType{Position: pos, Trait: p.curToken.Literal}
	}

	// Primitive or Generic e.g. Option<T>
	typeName := p.curToken.Literal
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume '<'
		p.nextToken()
		var lifetimes []string
		var params []ast.Type
		for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.CHAR) {
				lifetimes = append(lifetimes, p.curToken.Literal)
			} else {
				params = append(params, p.parseType())
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		if !p.curTokenIs(lexer.GT) {
			return nil
		}
		return &ast.GenericType{
			Position:  pos,
			BaseName:  typeName,
			Params:    params,
			Lifetimes: lifetimes,
		}
	}

	return &ast.PrimitiveType{Position: pos, Name: typeName}
}

// ---------------- Statement Parsing ----------------

func (p *Parser) parseStmt() ast.Stmt {
	switch p.curToken.Type {
	case lexer.LET:
		return p.parseLetStmt(false)
	case lexer.MUT:
		return p.parseLetStmt(true)
	case lexer.CONST:
		return p.parseConstStmt()
	case lexer.RETURN:
		return p.parseReturnStmt()
	case lexer.GO:
		return p.parseGoStmt()
	case lexer.SELECT:
		return p.parseSelectStmt()
	case lexer.UNSAFE:
		return p.parseUnsafeOrBlock()
	case lexer.ASM:
		return p.parseAsmBlock()
	case lexer.IF:
		return p.parseIfStmt()
	case lexer.FOR:
		return p.parseForStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseLetStmt(mutable bool) *ast.LetStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var ftype ast.Type
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		ftype = p.parseType()
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()
	val := p.parseExpr(LOWEST)

	return &ast.LetStmt{
		Position: pos,
		Name:     name,
		Mutable:  mutable,
		Type:     ftype,
		Value:    val,
	}
}

func (p *Parser) parseConstStmt() *ast.ConstStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var ftype ast.Type
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		ftype = p.parseType()
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()
	val := p.parseExpr(LOWEST)

	return &ast.ConstStmt{
		Position: pos,
		Name:     name,
		Type:     ftype,
		Value:    val,
	}
}

func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	var val ast.Expr
	if !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		val = p.parseExpr(LOWEST)
	}

	return &ast.ReturnStmt{Position: pos, Value: val}
}

func (p *Parser) parseGoStmt() *ast.GoStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	p.nextToken()
	expr := p.parseExpr(LOWEST)
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		diag := diagnostics.Diagnostic{
			Code:    "E501",
			Message: "go statement must invoke a function call",
			File:    p.curToken.File,
			Line:    p.curToken.Line,
			Column:  p.curToken.Col,
		}
		p.error(diag)
		return nil
	}

	return &ast.GoStmt{Position: pos, Call: call}
}

func (p *Parser) parseSelectStmt() *ast.SelectStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	cases := []ast.SelectCase{}
	var defaultBlock *ast.BlockStmt

	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "case" {
			casePos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
			p.nextToken()
			var varName string
			var channelOp ast.Expr
			if p.peekTokenIs(lexer.ASSIGN) {
				varName = p.curToken.Literal
				p.nextToken() // consume name
				p.nextToken() // consume '='
				channelOp = p.parseExpr(LOWEST)
			} else {
				channelOp = p.parseExpr(LOWEST)
			}
			p.expectPeek(lexer.COLON)
			
			// Parse statements until next case, default, or }
			stmts := []ast.Stmt{}
			for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
				if p.peekTokenIs(lexer.IDENT) && (p.peekToken.Literal == "case" || p.peekToken.Literal == "default") {
					break
				}
				p.nextToken()
				stmt := p.parseStmt()
				if stmt != nil {
					stmts = append(stmts, stmt)
				}
			}
			body := &ast.BlockStmt{Position: casePos, Stmts: stmts}
			cases = append(cases, ast.SelectCase{VarName: varName, ChannelOp: channelOp, Body: body})
		} else if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "default" {
			defaultPos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
			p.expectPeek(lexer.COLON)
			
			stmts := []ast.Stmt{}
			for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
				if p.peekTokenIs(lexer.IDENT) && (p.peekToken.Literal == "case" || p.peekToken.Literal == "default") {
					break
				}
				p.nextToken()
				stmt := p.parseStmt()
				if stmt != nil {
					stmts = append(stmts, stmt)
				}
			}
			defaultBlock = &ast.BlockStmt{Position: defaultPos, Stmts: stmts}
		}
	}

	p.expectPeek(lexer.RBRACE)

	return &ast.SelectStmt{Position: pos, Cases: cases, Default: defaultBlock}
}

func (p *Parser) parseUnsafeOrBlock() ast.Stmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	if p.peekTokenIs(lexer.LBRACE) {
		p.nextToken()
		block := p.parseBlockStmt()
		return &ast.UnsafeBlock{Position: pos, Block: block}
	}

	// Just a normal expression statement starting with unsafe variable/call
	return p.parseExprStmt()
}

func (p *Parser) parseAsmBlock() *ast.AsmBlock {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	rawText := p.curToken.Literal
	return &ast.AsmBlock{Position: pos, RawText: rawText}
}

func (p *Parser) parseIfStmt() *ast.IfStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	p.nextToken()
	oldNoStructInit := p.noStructInit
	p.noStructInit = true
	cond := p.parseExpr(LOWEST)
	p.noStructInit = oldNoStructInit

	p.expectPeek(lexer.LBRACE)
	thenBlock := p.parseBlockStmt()

	var elseBlock ast.Stmt
	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()
		if p.peekTokenIs(lexer.IF) {
			p.nextToken()
			elseBlock = p.parseIfStmt()
		} else if p.peekTokenIs(lexer.LBRACE) {
			p.nextToken()
			elseBlock = p.parseBlockStmt()
		}
	}

	return &ast.IfStmt{
		Position:  pos,
		Condition: cond,
		ThenBlock: thenBlock,
		ElseBlock: elseBlock,
	}
}

func (p *Parser) parseForStmt() *ast.ForStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	// for item in items {
	// for cond {
	p.nextToken()
	var varName string
	var rangeExpr ast.Expr

	oldNoStructInit := p.noStructInit
	p.noStructInit = true

	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "in" {
		varName = p.curToken.Literal
		p.nextToken() // consume var
		p.nextToken() // consume 'in'
		rangeExpr = p.parseExpr(LOWEST)
	} else if !p.curTokenIs(lexer.LBRACE) {
		rangeExpr = p.parseExpr(LOWEST)
	}

	p.noStructInit = oldNoStructInit

	p.expectPeek(lexer.LBRACE)
	body := p.parseBlockStmt()

	return &ast.ForStmt{
		Position:  pos,
		VarName:   varName,
		RangeExpr: rangeExpr,
		Body:      body,
	}
}

func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	stmts := []ast.Stmt{}

	if p.curTokenIs(lexer.LBRACE) {
		p.nextToken()
	}

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		p.nextToken()
	}

	return &ast.BlockStmt{Position: pos, Stmts: stmts}
}

func (p *Parser) parseExprStmt() *ast.ExprStmt {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	expr := p.parseExpr(LOWEST)
	return &ast.ExprStmt{Position: pos, Expression: expr}
}

// ---------------- Expression Parsing (Precedence Pratt Parsing) ----------------

const (
	_ int = iota
	LOWEST
	ASSIGNMENT // = or +=
	OR         // ||
	AND        // &&
	COMPARE    // == != < <= > >=
	SUM        // + -
	PRODUCT    // * / %
	PREFIX     // -x or !x or &x or *x
	CALL       // fn() or obj.method() or arr[index] or expr?
)

var precedences = map[lexer.TokenType]int{
	lexer.ASSIGN:     ASSIGNMENT,
	lexer.ADD_ASSIGN: ASSIGNMENT,
	lexer.SUB_ASSIGN: ASSIGNMENT,
	lexer.MUL_ASSIGN: ASSIGNMENT,
	lexer.DIV_ASSIGN: ASSIGNMENT,
	lexer.MOD_ASSIGN: ASSIGNMENT,
	lexer.OR:         OR,
	lexer.AND:        AND,
	lexer.EQ:         COMPARE,
	lexer.NEQ:        COMPARE,
	lexer.LT:         COMPARE,
	lexer.LTE:        COMPARE,
	lexer.GT:         COMPARE,
	lexer.GTE:        COMPARE,
	lexer.ADD:        SUM,
	lexer.SUB:        SUM,
	lexer.MUL:        PRODUCT,
	lexer.DIV:        PRODUCT,
	lexer.MOD:        PRODUCT,
	lexer.LPAREN:     CALL,
	lexer.LBRACKET:   CALL,
	lexer.DOT:        CALL,
	lexer.QUESTION:   CALL,
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() int {
	if p.peekToken.Type == lexer.LT && p.isGenericSpecifier() {
		return CALL
	}
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) parseExpr(precedence int) ast.Expr {
	prefix := p.parsePrefixFn(p.curToken.Type)
	if prefix == nil {
		diag := diagnostics.Diagnostic{
			Code:    "E103",
			Message: fmt.Sprintf("no prefix parsing function for %s", p.curToken.Type),
			File:    p.curToken.File,
			Line:    p.curToken.Line,
			Column:  p.curToken.Col,
			SpanLen: len(p.curToken.Literal),
		}
		p.error(diag)
		return nil
	}

	leftExpr := prefix()

	for !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
		infix := p.parseInfixFn(p.peekToken.Type)
		if infix == nil {
			return leftExpr
		}
		p.nextToken()
		leftExpr = infix(leftExpr)
	}

	return leftExpr
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

func (p *Parser) parsePrefixFn(t lexer.TokenType) prefixParseFn {
	switch t {
	case lexer.IDENT:
		return p.parseIdentExpr
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.CHAR:
		return p.parseLiteralExpr
	case lexer.SUB, lexer.NOT, lexer.MUL:
		return p.parsePrefixExpr
	case lexer.BITAND:
		return p.parseRefExpr
	case lexer.LPAREN:
		return p.parseGroupedExpr
	case lexer.MATCH:
		return p.parseMatchExpr
	default:
		return nil
	}
}

func (p *Parser) parseInfixFn(t lexer.TokenType) infixParseFn {
	if t == lexer.LT && p.isGenericSpecifier() {
		return p.parseGenericCallOrInit
	}
	switch t {
	case lexer.ADD, lexer.SUB, lexer.MUL, lexer.DIV, lexer.MOD,
		lexer.EQ, lexer.NEQ, lexer.LT, lexer.LTE, lexer.GT, lexer.GTE,
		lexer.AND, lexer.OR, lexer.ASSIGN, lexer.ADD_ASSIGN, lexer.SUB_ASSIGN,
		lexer.MUL_ASSIGN, lexer.DIV_ASSIGN, lexer.MOD_ASSIGN:
		return p.parseBinaryExpr
	case lexer.LPAREN:
		return p.parseCallExpr
	case lexer.LBRACKET:
		return p.parseIndexExpr
	case lexer.DOT:
		return p.parseSelectorExpr
	case lexer.QUESTION:
		return p.parseQuestionExpr
	default:
		return nil
	}
}

func (p *Parser) parseIdentExpr() ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}

	// Struct Initialization: Name { fields }
	// We can check if peek is LBRACE.
	if !p.noStructInit && p.peekTokenIs(lexer.LBRACE) {
		typeName := p.curToken.Literal
		p.nextToken() // consume name
		p.nextToken() // consume '{'
		fields := []ast.StructInitField{}
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			if !p.curTokenIs(lexer.IDENT) {
				break
			}
			fname := p.curToken.Literal
			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()
			fval := p.parseExpr(LOWEST)
			fields = append(fields, ast.StructInitField{Name: fname, Value: fval})
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		return &ast.StructInitExpr{
			Position: pos,
			Struct:   &ast.PrimitiveType{Position: pos, Name: typeName},
			Fields:   fields,
		}
	}

	return &ast.IdentExpr{Position: pos, Name: p.curToken.Literal}
}

func (p *Parser) parseLiteralExpr() ast.Expr {
	return &ast.LiteralExpr{
		Position: ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col},
		Type:     p.curToken.Type,
		Value:    p.curToken.Literal,
	}
}

func (p *Parser) parsePrefixExpr() ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	op := p.curToken.Type
	p.nextToken()
	right := p.parseExpr(PREFIX)

	if op == lexer.MUL { // *x is dereference
		return &ast.DerefExpr{Position: pos, Target: right}
	}
	return &ast.UnaryExpr{Position: pos, Op: op, Right: right}
}

func (p *Parser) parseRefExpr() ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	mutable := false
	if p.peekTokenIs(lexer.MUT) {
		p.nextToken()
		mutable = true
	}
	p.nextToken()
	target := p.parseExpr(PREFIX)
	return &ast.RefExpr{Position: pos, Mutable: mutable, Target: target}
}

func (p *Parser) parseGroupedExpr() ast.Expr {
	p.nextToken()
	expr := p.parseExpr(LOWEST)
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return expr
}

func (p *Parser) parseBinaryExpr(left ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	op := p.curToken.Type
	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpr(precedence)
	return &ast.BinaryExpr{Position: pos, Op: op, Left: left, Right: right}
}

func (p *Parser) parseCallExpr(function ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	args := []ast.Expr{}

	for !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		args = append(args, p.parseExpr(LOWEST))
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}
	p.expectPeek(lexer.RPAREN)

	return &ast.CallExpr{
		Position: pos,
		Function: function,
		Args:     args,
	}
}

func (p *Parser) parseIndexExpr(left ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	p.nextToken() // consume '['
	index := p.parseExpr(LOWEST)
	p.expectPeek(lexer.RBRACKET)
	return &ast.IndexExpr{Position: pos, Target: left, Index: index}
}

func (p *Parser) parseSelectorExpr(left ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	p.expectPeek(lexer.IDENT)
	field := p.curToken.Literal
	return &ast.SelectorExpr{Position: pos, Target: left, Field: field}
}

func (p *Parser) parseQuestionExpr(left ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	return &ast.QuestionExpr{Position: pos, Target: left}
}

func (p *Parser) parseMatchExpr() ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	p.nextToken() // consume 'match'
	oldNoStructInit := p.noStructInit
	p.noStructInit = true
	target := p.parseExpr(LOWEST)
	p.noStructInit = oldNoStructInit

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	cases := []ast.MatchCase{}
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		pattern := p.parseExpr(LOWEST)
		if !p.expectPeek(lexer.FAT_ARROW) {
			return nil
		}
		p.expectPeek(lexer.LBRACE)
		body := p.parseBlockStmt()
		cases = append(cases, ast.MatchCase{Pattern: pattern, Body: body})
	}
	p.expectPeek(lexer.RBRACE)

	return &ast.MatchExpr{Position: pos, Target: target, Cases: cases}
}

func (p *Parser) isGenericSpecifier() bool {
	if p.peekToken.Type != lexer.LT {
		return false
	}
	lexerClone := *p.l
	depth := 1
	for {
		tok := lexerClone.NextToken()
		if tok.Type == lexer.EOF {
			return false
		}
		if tok.Type == lexer.LT {
			depth++
		} else if tok.Type == lexer.GT {
			depth--
			if depth == 0 {
				nextTok := lexerClone.NextToken()
				return nextTok.Type == lexer.LPAREN || nextTok.Type == lexer.LBRACE
			}
		}
	}
}

func (p *Parser) parseGenericCallOrInit(left ast.Expr) ast.Expr {
	pos := ast.Position{File: p.curToken.File, Line: p.curToken.Line, Col: p.curToken.Col}
	// curToken is '<'
	
	generics := []ast.Type{}
	for !p.peekTokenIs(lexer.GT) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		generics = append(generics, p.parseType())
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}
	
	if !p.expectPeek(lexer.GT) {
		return nil
	}
	
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume '('
		args := []ast.Expr{}
		for !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.EOF) {
			p.nextToken()
			args = append(args, p.parseExpr(LOWEST))
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
		}
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
		return &ast.CallExpr{
			Position:  pos,
			Function:  left,
			Generics:  generics,
			Args:      args,
		}
	}
	
	if p.peekTokenIs(lexer.LBRACE) {
		p.nextToken() // consume '{'
		fields := []ast.StructInitField{}
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			p.nextToken()
			if !p.curTokenIs(lexer.IDENT) {
				break
			}
			fname := p.curToken.Literal
			if !p.expectPeek(lexer.COLON) {
				return nil
			}
			p.nextToken()
			fval := p.parseExpr(LOWEST)
			fields = append(fields, ast.StructInitField{Name: fname, Value: fval})
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		
		ident, ok := left.(*ast.IdentExpr)
		var baseName string
		if ok {
			baseName = ident.Name
		} else {
			baseName = left.String()
		}
		
		return &ast.StructInitExpr{
			Position: pos,
			Struct: &ast.GenericType{
				Position: pos,
				BaseName: baseName,
				Params:   generics,
			},
			Fields: fields,
		}
	}
	
	return left
}
