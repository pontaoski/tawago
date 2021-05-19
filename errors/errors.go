package errors

import (
	"fmt"

	"github.com/pontaoski/tawago/types"
)

type ExpectedKindGotKind struct {
	Expected types.TokenKind
	Got      types.TokenKind
	Location types.Span
}

func (e ExpectedKindGotKind) Error() string {
	return fmt.Sprintf("got a %d, expected a %d. %s", e.Got, e.Expected, e.Location)
}

type ExpectedOneOfKindGotKind struct {
	Expected []types.TokenKind
	Got      types.TokenKind
	Location types.Span
}

func (e ExpectedOneOfKindGotKind) Error() string {
	return fmt.Sprintf("got a %s, expected one of %s. %s", e.Got, e.Expected, e.Location)
}

type DuplicateField struct {
	Name     string
	Location types.Span
}

func (e DuplicateField) Error() string {
	return fmt.Sprintf("field %s specified more than once. %s", e.Name, e.Location)
}
