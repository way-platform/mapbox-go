package mapbox

import (
	"context"
	"time"
)

// WithRetrySleepForTest exports the unexported withRetrySleep option for tests.
func WithRetrySleepForTest(fn func(ctx context.Context, d time.Duration) bool) Option {
	return withRetrySleep(fn)
}
