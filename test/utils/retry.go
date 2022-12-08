package utils

import (
	"time"

	"github.com/avast/retry-go/v4"
)

func WithRetry(utils *TestUtils, f func(utils *TestUtils) error) error {
	return retry.Do(
		func() error {
			return f(utils)
		},
		retry.Delay(1*time.Second),
		retry.Context(utils.Ctx),
		retry.LastErrorOnly(true),
	)
}
