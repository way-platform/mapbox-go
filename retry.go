package mapbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	retryBaseDelay = 500 * time.Millisecond
	retryMaxDelay  = 10 * time.Second
)

// retryTransport retries requests on 429 and 5xx responses using exponential
// backoff with full jitter. It does not retry on context cancellation or
// deadline exceeded, nor on 4xx errors other than 429.
type retryTransport struct {
	next       http.RoundTripper
	maxRetries int
	// sleep overrides the wait function for testing. Nil uses the real timer.
	sleep func(ctx context.Context, d time.Duration) bool
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer the request body so it can be replayed on retry.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
	}

	var (
		resp *http.Response
		err  error
	)
	for attempt := range t.maxRetries + 1 {
		// Restore the body for each attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err = t.next.RoundTrip(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			if attempt < t.maxRetries {
				sleepFn := t.sleep
				if sleepFn == nil {
					sleepFn = waitFor
				}
				sleepFn(req.Context(), retryDelay(attempt, 0))
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < t.maxRetries {
				resp.Body.Close()
				delay := retryDelay(attempt, retryAfterDelay(resp))
				sleepFn := t.sleep
				if sleepFn == nil {
					sleepFn = waitFor
				}
				if !sleepFn(req.Context(), delay) {
					return nil, req.Context().Err()
				}
				continue
			}
		}

		return resp, nil
	}
	return resp, err
}

// retryDelay returns the backoff duration for the given attempt.
// If a non-zero Retry-After delay is provided, it takes precedence over the
// computed backoff when it is larger.
func retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	exp := math.Pow(2, float64(attempt))
	backoff := time.Duration(float64(retryBaseDelay)*exp) + time.Duration(rand.Float64()*float64(retryBaseDelay))
	if backoff > retryMaxDelay {
		backoff = retryMaxDelay
	}
	if retryAfter > backoff {
		return retryAfter
	}
	return backoff
}

// retryAfterDelay parses the Retry-After header from a response.
// Returns 0 if the header is absent or unparseable.
func retryAfterDelay(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	secs, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

// waitFor sleeps for d, or until ctx is cancelled. Returns false if ctx was cancelled.
func waitFor(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
