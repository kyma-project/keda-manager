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
	// AppendMissingArgs returns args that should be appended if they don't exist
	AppendMissingArgs(existingArgs []string) []string
}
