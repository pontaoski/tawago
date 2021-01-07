package main

import "fmt"

type ExpectedKindGotKind struct {
	Expected TokenKind
	Got      TokenKind
	From     Position
	To       Position
}

func (e ExpectedKindGotKind) Error() string {
	return fmt.Sprintf("got a %d, expected a %d", e.Got, e.Expected)
}

type ExpectedOneOfKindGotKind struct {
	Expected []TokenKind
	Got      TokenKind
	From     Position
	To       Position
}

func (e ExpectedOneOfKindGotKind) Error() string {
	return fmt.Sprintf("got a %s, expected one of %s. %d:%d - %d:%d", e.Got, e.Expected, e.From.Line, e.From.Column, e.To.Line, e.To.Column)
}

type DuplicateField struct {
	Name string
	From Position
	To   Position
}

func (e DuplicateField) Error() string {
	return fmt.Sprintf("field %s specified more than once %d:%d - %d:%d", e.Name, e.From.Line, e.From.Column, e.To.Line, e.To.Column)
}
