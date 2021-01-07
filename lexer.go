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
	FATARROW

	VAR
	LET

	EOS

	INT

	IDENT
	STRING

	TYPE
	IF
	THEN
	ELSE
	FUNC
	STRUCT
	IMPORT
)

func (t TokenKind) String() string {
	data := map[TokenKind]string{
		EOF:      "EOF",
		ILLEGAL:  "ILLEGAL",
		COLON:    "COLON",
		LPAREN:   "LPAREN",
		RPAREN:   "RPAREN",
		LBRACKET: "LBRACKET",
		RBRACKET: "RBRACKET",
		COMMA:    "COMMA",
		EQUALS:   "EQUALS",
		EOS:      "EOS",
		IDENT:    "IDENT",
		STRING:   "STRING",
		TYPE:     "TYPE",
		IF:       "IF",
		THEN:     "THEN",
		ELSE:     "ELSE",
		FUNC:     "FUNC",
		STRUCT:   "STRUCT",
		IMPORT:   "IMPORT",
	}
	return data[t]
}

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

func (l *Lexer) PeekIs(k ...TokenKind) bool {
	token, _ := l.Peek()
	for _, kind := range k {
		if token.Kind == kind {
			return true
		}
	}

	return false
}

func (l *Lexer) PeekIsWithRet(k ...TokenKind) (bool, Token, string) {
	token, lit := l.Peek()
	for _, kind := range k {
		if token.Kind == kind {
			return true, token, lit
		}
	}

	return false, Token{}, ""
}

func (l *Lexer) LexExpecting(k ...TokenKind) (Token, string) {
	token, lit := l.Lex()
	for _, kind := range k {
		if token.Kind == kind {
			return token, lit
		}
	}

	panic(ExpectedOneOfKindGotKind{
		Expected: k,
		Got:      token.Kind,
		From:     token.From,
		To:       token.To,
	})
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

		switch {
		case r == '=':
			byt, err := l.reader.Peek(1)
			if err != nil && err != io.EOF {
				panic(err)
			}
			if byt[0] == '>' {

				if _, _, err := l.reader.ReadRune(); err != nil {
					panic(err)
				}
				return l.kinded(FATARROW), "=>"
			}
			return l.kinded(EQUALS), "="
		}

		data := map[rune]TokenKind{
			':': COLON,
			'(': LPAREN,
			')': RPAREN,
			'{': LBRACKET,
			'}': RBRACKET,
			',': COMMA,
			';': EOS,
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
			"struct": STRUCT,
			"var":    VAR,
			"let":    LET,
		}

		switch {
		case unicode.IsDigit(r):
			var runes string
			runes += string(r)
			for {
				r, _, err := l.reader.ReadRune()
				if err != nil {
					if err == io.EOF {
						return l.kinded(INT), runes
					}
					panic(err)
				}

				if !unicode.IsDigit(r) {
					l.backup()
					return l.kinded(INT), runes
				}

				runes += string(r)
			}
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
