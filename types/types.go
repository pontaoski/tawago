package types

import (
	"fmt"
)

type Position struct {
	Line     int
	Column   int
	Filename string
}

type Span struct {
	From Position
	To   Position
}

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

type Token struct {
	Kind     TokenKind
	Location Span
}
