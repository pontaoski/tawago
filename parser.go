package main

import (
	"strconv"

	"github.com/ztrue/tracerr"
)

type Parser struct {
	l   *Lexer
	ast AST
}

func NewParser(l *Lexer) Parser {
	a := AST{}
	return Parser{l, a}
}

func (p *Parser) Parse() (err error) {
	defer func() {
		if r := recover(); r != nil {
			rerr, ok := r.(error)
			if ok {
				err = tracerr.Wrap(rerr)
			} else {
				panic(r)
			}
		}
	}()
	for {
		tok, _ := p.l.Lex()

		if tok.Kind == EOF {
			return
		}

		switch tok.Kind {
		case IMPORT:
			p.parseImport()
		case TYPE:
			_, name := p.l.LexExpecting(IDENT)
			p.ast.Toplevels = append(p.ast.Toplevels, TypeDeclaration{
				Ident: NewID(name),
				Kind:  p.parseType(),
			})
		case FUNC:
			_, name := p.l.LexExpecting(IDENT)
			var arguments []struct {
				Ident Identifier
				Kind  Type
			}

			p.l.LexExpecting(LPAREN)
			if !p.l.PeekIs(RPAREN) {
				for {
					_, name := p.l.LexExpecting(IDENT)
					p.l.LexExpecting(COLON)
					kind := p.parseType()

					arguments = append(arguments, struct {
						Ident Identifier
						Kind  Type
					}{
						Ident: NewID(name),
						Kind:  kind,
					})

					if p.l.PeekIs(COMMA, RPAREN) {
						if p.l.PeekIs(RPAREN) {
							break
						}
						continue
					}

					p.l.LexExpecting(COMMA, RPAREN)
				}
			}
			p.l.LexExpecting(RPAREN)

			var ret *Type
			if p.l.PeekIs(IDENT, FUNC, STRUCT) {
				t := p.parseType()
				ret = &t
			}
			var expr Expression
			if !p.l.PeekIs(FATARROW, LBRACKET) {
				tok, _ := p.l.Peek()
				panic(ExpectedOneOfKindGotKind{
					Expected: []TokenKind{FATARROW, LBRACKET},
					Got:      tok.Kind,
					Location: tok.Location,
				})
			}
			if p.l.PeekIs(FATARROW) {
				p.l.LexExpecting(FATARROW)
				expr = p.parseExpression()
			} else {
				p.l.LexExpecting(LBRACKET)
				expr = p.parseBlock()
			}
			p.ast.Toplevels = append(p.ast.Toplevels, Func{
				Ident:     NewID(name),
				Arguments: arguments,
				Returns:   ret,
				Expr:      expr,
			})
			p.l.LexExpecting(EOS)
		}
	}
}

type AST struct {
	Toplevels []TopLevel
}

func (p *Parser) parseImport() {
	_, path := p.l.LexExpecting(STRING)
	p.ast.Toplevels = append(p.ast.Toplevels, Import(path))
	p.l.LexExpecting(EOS)
}

// parseBlock should be called with the parser is past the opening brace
func (p *Parser) parseBlock() Expression {
	var statements []Expression

	if !p.l.PeekIs(RBRACKET) {
		for {
			if p.l.PeekIs(EOS) {
				p.l.LexExpecting(EOS)
				continue
			}

			statements = append(statements, p.parseExpression())

			if p.l.PeekIs(EOS, RBRACKET) {
				if p.l.PeekIs(RBRACKET) {
					break
				}
				p.l.LexExpecting(EOS)
				if p.l.PeekIs(RBRACKET) {
					break
				}
				continue
			}

			p.l.LexExpecting(EOS, RBRACKET)
		}
	}
	p.l.LexExpecting(RBRACKET)

	return Block(statements)
}

func (p *Parser) parseStructLiteral() (r map[string]Expression) {
	r = map[string]Expression{}

	p.l.LexExpecting(LBRACKET)

	if !p.l.PeekIs(RBRACKET) {
		for {
			if p.l.PeekIs(COMMA, EOS, RBRACKET) {
				if p.l.PeekIs(RBRACKET) {
					break
				}
				p.l.LexExpecting(COMMA, EOS)
				continue
			}

			tok, name := p.l.LexExpecting(IDENT)
			p.l.LexExpecting(COLON)
			expr := p.parseExpression()

			if _, ok := r[name]; ok {
				panic(DuplicateField{
					Name:     name,
					Location: tok.Location,
				})
			}

			r[name] = expr

			if p.l.PeekIs(COMMA, EOS, RBRACKET) {
				if p.l.PeekIs(RBRACKET) {
					break
				}
				continue
			}

			p.l.LexExpecting(COMMA, RBRACKET)
		}
	}
	p.l.LexExpecting(RBRACKET)

	return
}

func (p *Parser) parseExpressionLeaf() Expression {
	tok, lit := p.l.LexExpecting(IDENT, IF, STRING, LBRACKET, INT, LET, VAR, NEW, DELETE)

	switch tok.Kind {
	case LET:
		_, ident := p.l.LexExpecting(IDENT)
		p.l.LexExpecting(EQUALS)
		return Declaration{
			To:    Identifier(NewID(ident)),
			Value: p.parseExpression(),
		}
	case VAR:
		_, ident := p.l.LexExpecting(IDENT)
		p.l.LexExpecting(EQUALS)
		return MutDeclaration{
			To:    Identifier(NewID(ident)),
			Value: p.parseExpression(),
		}
	case STRING:
		return Lit{StringLiteral(lit)}
	case INT:
		parsed, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			panic(err)
		}
		return Lit{Integer(parsed)}
	case IDENT:
		if !p.l.PeekIs(LPAREN, EQUALS, LBRACKET) {
			return Var(NewID(lit))
		}

		if p.l.PeekIs(LPAREN) {
			p.l.LexExpecting(LPAREN)
			var args []Expression

			if !p.l.PeekIs(RPAREN) {
				for {
					args = append(args, p.parseExpression())

					if p.l.PeekIs(COMMA, RPAREN) {
						if p.l.PeekIs(RPAREN) {
							break
						}
						continue
					}

					p.l.LexExpecting(COMMA, RPAREN)
				}
			}
			p.l.LexExpecting(RPAREN)

			return Call{
				Function:  NewID(lit),
				Arguments: args,
			}
		} else if p.l.PeekIs(EQUALS) {
			p.l.LexExpecting(EQUALS)

			return Assignment{
				To:    NewID(lit),
				Value: p.parseExpression(),
				Pos:   Span{tok.Location.From, p.l.pos},
			}
		} else if p.l.PeekIs(LBRACKET) {
			return Lit{StructLiteral{
				Ident:  NewID(lit),
				Fields: p.parseStructLiteral(),
			}}
		}
	case IF:
		cond := p.parseExpression()
		p.l.LexExpecting(THEN)
		then := p.parseExpression()
		p.l.LexExpecting(ELSE)
		elseExpr := p.parseExpression()

		return If{
			Condition: cond,
			Then:      then,
			Else:      elseExpr,
		}
	case LBRACKET:
		return p.parseBlock()
	case NEW:
		return Allocation{
			PutOnHeap: p.parseExpression(),
		}
	case DELETE:
		return Freeing{
			RemoveFromHeap: p.parseExpression(),
		}
	}

	panic("unhandled")
}

func (p *Parser) parseExpression() Expression {
	from := p.l.pos
	expr := p.parseExpressionLeaf()

	if p.l.PeekIs(PERIOD) {
		tok, lit := p.l.LexWithI(1, PERIOD, IDENT)

		if p.l.PeekIs(EQUALS) {
			p.l.LexExpecting(EQUALS)

			return FieldAssignment{
				Struct: expr,
				Field:  NewID(lit),
				Value:  p.parseExpression(),
				Pos:    Span{from, p.l.pos},
			}
		}

		return Field{
			Of:    expr,
			Ident: Identifier{lit, Span{from, tok.Location.To}},
		}
	}

	return expr
}

// expected to be called after reading type keyword and name token.
func (p *Parser) parseType() Type {
	tok, lit := p.l.LexExpecting(IDENT, FUNC, STRUCT)

	switch tok.Kind {
	case IDENT:
		return Ident(NewID(lit))
	case FUNC:
		p.l.LexExpecting(LPAREN)
		f := FunctionPointer{}
		if !p.l.PeekIs(RPAREN) {
			for {
				f.Arguments = append(f.Arguments, p.parseType())
				if p.l.PeekIs(COMMA) {
					p.l.LexExpecting(COMMA)
					continue
				}
				break
			}
		}
		p.l.LexExpecting(RPAREN)
		if p.l.PeekIs(IDENT, FUNC, STRUCT) {
			t := p.parseType()
			f.Returns = &t
		}
		return f
	case STRUCT:
		s := Struct{}
		p.l.LexExpecting(LBRACKET)
		if !p.l.PeekIs(RBRACKET) {
			for {
				if p.l.PeekIs(EOS, RBRACKET) {
					if p.l.PeekIs(RBRACKET) {
						break
					}
					p.l.LexExpecting(EOS)
					continue
				}

				_, name := p.l.LexExpecting(IDENT)
				p.l.LexExpecting(COLON)
				kind := p.parseType()

				s = append(s, struct {
					Ident string
					Kind  Type
				}{
					Ident: name,
					Kind:  kind,
				})

				if p.l.PeekIs(EOS, RBRACKET) {
					if p.l.PeekIs(RBRACKET) {
						break
					}
					continue
				}

				p.l.LexExpecting(EOS, RBRACKET)
			}
		}
		p.l.LexExpecting(RBRACKET)
		return s
	}

	panic("Unexpected")
}
