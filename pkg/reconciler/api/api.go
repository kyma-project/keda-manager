package api

import (
	"fmt"
)

type MatchStringer interface {
	fmt.Stringer
	Match(*string) bool
}
