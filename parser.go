package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/repr"
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
				Name: Identifier(name),
				Kind: p.parseType(),
			})
		case FUNC:
			_, name := p.l.LexExpecting(IDENT)
			var arguments []struct {
				Name Identifier
				Kind Type
			}

			p.l.LexExpecting(LPAREN)
			if !p.l.PeekIs(RPAREN) {
				for {
					_, name := p.l.LexExpecting(IDENT)
					p.l.LexExpecting(COLON)
					kind := p.parseType()

					arguments = append(arguments, struct {
						Name Identifier
						Kind Type
					}{
						Name: Identifier(name),
						Kind: kind,
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
					From:     tok.From,
					To:       tok.To,
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
				Name:      Identifier(name),
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
	tok, path := p.l.Lex()
	if tok.Kind != STRING {
		panic(ExpectedKindGotKind{STRING, tok.Kind, tok.From, tok.To})
	}

	p.ast.Toplevels = append(p.ast.Toplevels, Import(path))

	tok, path = p.l.Lex()
	if tok.Kind != EOS {
		panic(ExpectedKindGotKind{EOS, tok.Kind, tok.From, tok.To})
	}
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

func (p *Parser) parseExpression() Expression {
	tok, lit := p.l.LexExpecting(IDENT, IF, LBRACKET, INT)

	switch tok.Kind {
	case INT:
		parsed, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			panic(err)
		}
		return Lit{Integer(parsed)}
	case IDENT:
		if !p.l.PeekIs(LPAREN) {
			return Var(lit)
		}

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
			Function:  Identifier(lit),
			Arguments: args,
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
	}

	panic("unhandled")
}

// expected to be called after reading type keyword and name token.
func (p *Parser) parseType() Type {
	tok, lit := p.l.LexExpecting(IDENT, FUNC, STRUCT)

	switch tok.Kind {
	case IDENT:
		return Ident(lit)
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
				_, name := p.l.LexExpecting(IDENT)
				p.l.LexExpecting(COLON)
				kind := p.parseType()

				s = append(s, struct {
					Name string
					Kind Type
				}{
					Name: name,
					Kind: kind,
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

const wholeProgram = `import ` + "`hi`" + `

func eep() int64 {
	50
}

func main() {
	eep()
	if 0 then 50 else 30
}
`

func main() {
	l := NewLexer(strings.NewReader(wholeProgram))
	p := NewParser(l)
	err := p.Parse()
	if err != nil {
		tracerr.PrintSourceColor(err)
		os.Exit(1)
	}
	repr.Println(p.ast)
	codegen(p.ast.Toplevels)
}
