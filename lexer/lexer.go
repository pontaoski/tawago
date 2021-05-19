package lexer

import (
	"bufio"
	"io"
	"unicode"

	"github.com/pontaoski/tawago/errors"
	"github.com/pontaoski/tawago/types"
)

type Lexer struct {
	pos           types.Position
	reader        *bufio.Reader
	peeked        *types.Token
	peekedString  string
	insertNewline bool
}

func NewLexer(reader io.Reader, filename string) *Lexer {
	return &Lexer{
		pos:    types.Position{Line: 1, Column: 0, Filename: filename},
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

func (l *Lexer) kinded(t types.TokenKind) types.Token {
	return types.Token{
		Location: types.SingleCharSpan(l.pos),
		Kind:     t,
	}
}

func firstChar(r rune) bool {
	return r == '_' || r == '\'' || unicode.IsLetter(r)
}

func otherChar(r rune) bool {
	return r == '/' || firstChar(r) || unicode.IsDigit(r)
}

func (l *Lexer) lexIdent() (types.Position, types.Position, string) {
	var lit string
	var from types.Position
	var to types.Position

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

func (l *Lexer) lexString() (types.Position, types.Position, string) {
	var lit string
	var from types.Position
	var to types.Position
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

func (l *Lexer) Peek() (types.Token, string) {
	if l.peeked != nil {
		return *l.peeked, l.peekedString
	}

	tok, str := l.Lex()
	l.peeked = &tok
	l.peekedString = str

	return tok, str
}

func (l *Lexer) PeekIs(k ...types.TokenKind) bool {
	token, _ := l.Peek()
	for _, kind := range k {
		if token.Kind == kind {
			return true
		}
	}

	return false
}

func (l *Lexer) PeekIsWithRet(k ...types.TokenKind) (bool, types.Token, string) {
	token, lit := l.Peek()
	for _, kind := range k {
		if token.Kind == kind {
			return true, token, lit
		}
	}

	return false, types.Token{}, ""
}

func (l *Lexer) LexWithI(i int, kinds ...types.TokenKind) (t types.Token, s string) {
	for idx, kind := range kinds {
		tok, lit := l.LexExpecting(kind)
		if idx == i {
			t = tok
			s = lit
		}
	}

	return
}

func (l *Lexer) LexExpecting(k ...types.TokenKind) (types.Token, string) {
	token, lit := l.Lex()
	for _, kind := range k {
		if token.Kind == kind {
			return token, lit
		}
	}

	panic(errors.ExpectedOneOfKindGotKind{
		Expected: k,
		Got:      token.Kind,
		Location: token.Location,
	})
}

func (l *Lexer) Lex() (r types.Token, s string) {
	if l.peeked != nil {
		defer func() { l.peeked = nil }()
		return *l.peeked, l.peekedString
	}
	if l.insertNewline {
		l.insertNewline = false
		return types.Token{
			Kind:     types.EOS,
			Location: types.SingleCharSpan(l.pos),
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
			case types.IDENT, types.RBRACKET, types.RPAREN, types.INT, types.STRING:
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
				return l.kinded(types.EOF), ""
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
				return l.kinded(types.FATARROW), "=>"
			}
			return l.kinded(types.EQUALS), "="
		}

		data := map[rune]types.TokenKind{
			':': types.COLON,
			'(': types.LPAREN,
			')': types.RPAREN,
			'{': types.LBRACKET,
			'}': types.RBRACKET,
			',': types.COMMA,
			';': types.EOS,
			'.': types.PERIOD,
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

			return types.Token{types.STRING, types.Span{from, to}}, lit
		}

		keywords := map[string]types.TokenKind{
			"type":   types.TYPE,
			"if":     types.IF,
			"then":   types.THEN,
			"else":   types.ELSE,
			"func":   types.FUNC,
			"import": types.IMPORT,
			"struct": types.STRUCT,
			"var":    types.VAR,
			"let":    types.LET,
			"new":    types.NEW,
			"delete": types.DELETE,
		}

		switch {
		case unicode.IsDigit(r):
			var runes string
			runes += string(r)
			for {
				r, _, err := l.reader.ReadRune()
				if err != nil {
					if err == io.EOF {
						return l.kinded(types.INT), runes
					}
					panic(err)
				}

				if !unicode.IsDigit(r) {
					l.backup()
					return l.kinded(types.INT), runes
				}

				runes += string(r)
			}
		case unicode.IsSpace(r):
			continue
		case otherChar(r):
			l.backup()
			from, to, lit := l.lexIdent()

			if kind, ok := keywords[lit]; ok {
				return types.Token{kind, types.Span{from, to}}, lit
			}

			return types.Token{types.IDENT, types.Span{from, to}}, lit
		}

		panic("unhandled")
	}
}

type testToken struct {
	t types.Token
	s string
}

func (l *Lexer) lexToEOF() (ret []testToken) {
	t, s := l.Lex()
	for t.Kind != types.EOF {
		ret = append(ret, testToken{
			t: t,
			s: s,
		})
		t, s = l.Lex()
	}
	return
}
