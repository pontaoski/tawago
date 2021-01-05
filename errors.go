package main

import "fmt"

type ExpectedKindGotKind struct {
	Expected TokenKind
	Got      TokenKind
}

func (e ExpectedKindGotKind) Error() string {
	return fmt.Sprintf("got a %d, expected a %d", e.Got, e.Expected)
}
