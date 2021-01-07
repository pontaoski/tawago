package main

import "fmt"

type ExpectedKindGotKind struct {
	Expected TokenKind
	Got      TokenKind
	Location Span
}

func (e ExpectedKindGotKind) Error() string {
	return fmt.Sprintf("got a %d, expected a %d. %s", e.Got, e.Expected, e.Location)
}

type ExpectedOneOfKindGotKind struct {
	Expected []TokenKind
	Got      TokenKind
	Location Span
}

func (e ExpectedOneOfKindGotKind) Error() string {
	return fmt.Sprintf("got a %s, expected one of %s. %s", e.Got, e.Expected, e.Location)
}

type DuplicateField struct {
	Name     string
	Location Span
}

func (e DuplicateField) Error() string {
	return fmt.Sprintf("field %s specified more than once. %s", e.Name, e.Location)
}
