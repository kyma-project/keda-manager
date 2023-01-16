package api

import (
	"fmt"
)

type MatchStringer interface {
	fmt.Stringer
	Match(*string) bool
}

type ArgUpdater interface {
	UpdateArg(arg *string)
}
