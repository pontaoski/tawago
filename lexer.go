package main

import (
	"bufio"
	"fmt"
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
	PERIOD

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
	NEW
	DELETE
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
		FATARROW: "FATARROW",
		PERIOD:   "PERIOD",
		VAR:      "VAR",
		LET:      "LET",
		EOS:      "EOS",
		INT:      "INT",
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
	Line     int
	Column   int
	Filename string
}

type Span struct {
	From Position
	To   Position
}

func (p Position) String() string {
	if p.Filename == "" {
		p.Filename = "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

func (s Span) String() string {
	return fmt.Sprintf("%s-%d:%d", s.From, s.To.Line, s.To.Column)
}

func SingleCharSpan(p Position) Span {
	return Span{p, p}
}

type Lexer struct {
	pos           Position
	reader        *bufio.Reader
	peeked        *Token
	peekedString  string
	insertNewline bool
}

type Token struct {
	Kind     TokenKind
	Location Span
}

func NewLexer(reader io.Reader, filename string) *Lexer {
	return &Lexer{
		pos:    Position{Line: 1, Column: 0, Filename: filename},
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
		Location: SingleCharSpan(l.pos),
		Kind:     t,
	}
}

func firstChar(r rune) bool {
	return r == '_' || r == '\'' || unicode.IsLetter(r)
}

func otherChar(r rune) bool {
	return r == '/' || firstChar(r) || unicode.IsDigit(r)
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

func (l *Lexer) LexWithI(i int, kinds ...TokenKind) (t Token, s string) {
	for idx, kind := range kinds {
		tok, lit := l.LexExpecting(kind)
		if idx == i {
			t = tok
			s = lit
		}
	}

	return
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
		Location: token.Location,
	})
}

func (l *Lexer) Lex() (r Token, s string) {
	if l.peeked != nil {
		defer func() { l.peeked = nil }()
		return *l.peeked, l.peekedString
	}
	if l.insertNewline {
		l.insertNewline = false
		return Token{
			Kind:     EOS,
			Location: SingleCharSpan(l.pos),
		}, "\n"
	}

	defer func() {
		byt, err := l.reader.Peek(1)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if err == io.EOF {
			byt = append(byt, '\n')
		}

		if byt[0] == '\n' {
			switch r.Kind {
			case IDENT, RBRACKET, RPAREN, INT, STRING:
				_, err = l.reader.ReadByte()
				if err != nil {
					panic(err)
				}

				l.pos.Column = 0
				l.pos.Line++
				l.insertNewline = true
			}
		}
	}()

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
			'.': PERIOD,
		}

		if kind, ok := data[r]; ok {
			return l.kinded(kind), string(r)
		}

		switch r {
		case '\n':
			l.newline()
			continue
		case '`':
			l.backup()
			from, to, lit := l.lexString()

			return Token{STRING, Span{from, to}}, lit
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
			"new":    NEW,
			"delete": DELETE,
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
				return Token{kind, Span{from, to}}, lit
			}

			return Token{IDENT, Span{from, to}}, lit
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
