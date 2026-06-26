package mapbox

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Error represents an HTTP-level error response from the Mapbox API.
// Use [errors.As] to unwrap this type from returned errors.
type Error struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Status is the HTTP status text (e.g. "422 Unprocessable Entity").
	Status string
	// Body is the raw response body from Mapbox, may contain a JSON error object.
	Body string
}

func (e *Error) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("mapbox: http %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("mapbox: http %d", e.StatusCode)
}

// IsNotFound reports whether err is a 404 Mapbox API error.
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err is a 401 Mapbox API error.
func IsUnauthorized(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == http.StatusUnauthorized
}

// IsForbidden reports whether err is a 403 Mapbox API error.
func IsForbidden(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == http.StatusForbidden
}

// IsRateLimited reports whether err is a 429 Mapbox API error.
func IsRateLimited(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == http.StatusTooManyRequests
}

// IsInvalidInput reports whether err is a 422 Mapbox API error.
func IsInvalidInput(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == http.StatusUnprocessableEntity
}

// IsServerError reports whether err is a 5xx Mapbox API error.
func IsServerError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode >= 500
}

func newResponseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		body = fmt.Appendf(nil, "failed to read response body: %s", err)
	}
	return &Error{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(body),
	}
}

// Code is a Mapbox API response code returned in a 200 OK response body.
// Map Matching returns HTTP 200 for both successful and semantic-failure cases;
// callers must inspect Code to distinguish them.
type Code string

const (
	// CodeOK indicates a successful match.
	CodeOK Code = "Ok"
	// CodeNoMatch indicates no route matched the input trace.
	CodeNoMatch Code = "NoMatch"
	// CodeNoSegment indicates no road was found within the provided radiuses.
	CodeNoSegment Code = "NoSegment"
	// CodeTooManyCoordinates indicates more than 100 coordinates were provided.
	CodeTooManyCoordinates Code = "TooManyCoordinates"
	// CodeInvalidInput indicates a malformed request.
	CodeInvalidInput Code = "InvalidInput"
	// CodeProfileNotFound indicates an invalid routing profile was specified.
	CodeProfileNotFound Code = "ProfileNotFound"
)

// IsSuccess reports whether the code represents a successful match.
func (c Code) IsSuccess() bool { return c == CodeOK }
