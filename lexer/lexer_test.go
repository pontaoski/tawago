package lexer

import (
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
	l := NewLexer(strings.NewReader("aaa if else then ;"), "stdin")
	tokens := l.lexToEOF()
	t.Fatalf("%#v", tokens)
}
