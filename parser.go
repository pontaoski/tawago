package main

import (
	"go/parser"
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
		}
	}
}

type AST struct {
	Toplevels []TopLevel
}

func (p *Parser) parseImport() {
	tok, path := p.l.Lex()
	if tok.Kind != STRING {
		panic(ExpectedKindGotKind{STRING, tok.Kind})
	}

	p.ast.Toplevels = append(p.ast.Toplevels, Import(path))

	tok, path = p.l.Lex()
	if tok.Kind != EOS {
		panic(ExpectedKindGotKind{EOS, tok.Kind})
	}
}

func main() {
	data, err := parser.ParseExpr("Yeet")
	if err != nil {
		panic(err)
	}
	repr.Println(data)

	l := NewLexer(strings.NewReader("import `ok`;"))
	p := NewParser(l)
	err = p.Parse()
	if err != nil {
		panic(err)
	}
	repr.Println(p.ast)
}
