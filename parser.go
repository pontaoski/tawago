package main

import (
	"strings"

	"github.com/alecthomas/repr"
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
				err = rerr
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

func main() {
	l := NewLexer(strings.NewReader("import `ok`; type Yeet int; type Yeet func(int) int; type Yeet struct { hi: func(int, struct { nested: int }) int };"))
	p := NewParser(l)
	err := p.Parse()
	if err != nil {
		panic(err)
	}
	repr.Println(p.ast)
}
