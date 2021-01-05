package main

import (
	"bufio"
	"io"
	"unicode"
)

type TokenKind int

const (
	EOF TokenKind = iota
	ILLEGAL

	COLON
	LPAREN
	RPAREN
	LBRACKET
	RBRACKET
	COMMA
	EQUALS

	EOS

	IDENT
	STRING

	TYPE
	IF
	THEN
	ELSE
	FUNC
	IMPORT
)

type Position struct {
	Line   int
	Column int
}

type Lexer struct {
	pos          Position
	reader       *bufio.Reader
	peeked       *Token
	peekedString string
}

type Token struct {
	Kind TokenKind
	From Position
	To   Position
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{
		pos:    Position{Line: 1, Column: 0},
		reader: bufio.NewReader(reader),
	}
}

func (l *Lexer) newline() {
	l.pos.Line++
	l.pos.Column = 0
}

func (l *Lexer) backup() {
	if err := l.reader.UnreadRune(); err != nil {
		panic(err)
	}

	l.pos.Column--
}

func (l *Lexer) backupN(n int) {
	for i := 0; i < n; i++ {
		l.backup()
	}
}

func (l *Lexer) kinded(t TokenKind) Token {
	return Token{
		From: l.pos,
		To:   l.pos,
		Kind: t,
	}
}

func firstChar(r rune) bool {
	return r == '_' || r == '\'' || unicode.IsLetter(r)
}

func otherChar(r rune) bool {
	return firstChar(r) || unicode.IsDigit(r)
}

func (l *Lexer) lexIdent() (Position, Position, string) {
	var lit string
	var from Position
	var to Position

	r, _, err := l.reader.ReadRune()
	l.pos.Column++
	from = l.pos

	for {
		if err != nil {
			if err == io.EOF {
				return from, to, lit
			}
			panic(err)
		}

		if otherChar(r) {
			lit += string(r)
		} else {
			l.backup()
			to = l.pos
			return from, to, lit
		}

		r, _, err = l.reader.ReadRune()
		l.pos.Column++
		to = l.pos
	}
}

func (l *Lexer) lexString() (Position, Position, string) {
	var lit string
	var from Position
	var to Position
	seenOpen := false

	r, _, err := l.reader.ReadRune()
	l.pos.Column++
	from = l.pos

	for {
		if err != nil {
			if err == io.EOF {
				return from, to, lit
			}
			panic(err)
		}

		switch r {
		case '`':
			if seenOpen {
				to = l.pos
				return from, to, lit
			}
			seenOpen = true
		default:
			lit += string(r)
		}

		r, _, err = l.reader.ReadRune()
		l.pos.Column++
		to = l.pos
	}
}

func (l *Lexer) Peek() (Token, string) {
	if l.peeked != nil {
		return *l.peeked, l.peekedString
	}

	tok, str := l.Lex()
	l.peeked = &tok
	l.peekedString = str

	return tok, str
}

func (l *Lexer) Lex() (Token, string) {
	if l.peeked != nil {
		defer func() { l.peeked = nil }()
		return *l.peeked, l.peekedString
	}

	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return l.kinded(EOF), ""
			}
			panic(err)
		}

		l.pos.Column++

		data := map[rune]TokenKind{
			':': COLON,
			'(': LPAREN,
			')': RPAREN,
			'{': LBRACKET,
			'}': RBRACKET,
			',': COMMA,
			';': EOS,
			'=': EQUALS,
		}

		if kind, ok := data[r]; ok {
			return l.kinded(kind), string(r)
		}

		switch r {
		case '\n':
			l.newline()
			return l.kinded(EOS), "\n"
		case '`':
			l.backup()
			from, to, lit := l.lexString()

			return Token{STRING, from, to}, lit
		}

		keywords := map[string]TokenKind{
			"type":   TYPE,
			"if":     IF,
			"then":   THEN,
			"else":   ELSE,
			"func":   FUNC,
			"import": IMPORT,
		}

		switch {
		case unicode.IsSpace(r):
			continue
		case otherChar(r):
			l.backup()
			from, to, lit := l.lexIdent()

			if kind, ok := keywords[lit]; ok {
				return Token{kind, from, to}, lit
			}

			return Token{IDENT, from, to}, lit
		}

		panic("unhandled")
	}
}

type testToken struct {
	t Token
	s string
}

func (l *Lexer) lexToEOF() (ret []testToken) {
	t, s := l.Lex()
	for t.Kind != EOF {
		ret = append(ret, testToken{
			t: t,
			s: s,
		})
		t, s = l.Lex()
	}
	return
}