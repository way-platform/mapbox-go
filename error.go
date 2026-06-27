package mapbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Code is a Mapbox API response code. It appears in two places:
//   - HTTP 200 Map Matching response bodies (e.g. [CodeOK], [CodeNoMatch])
//   - HTTP 4xx error response bodies (e.g. [CodeInvalidInput], [CodeTooManyCoordinates])
//
// Use [CodeOf] to extract the code from any error.
type Code string

const (
	// CodeUnknown is the zero value, returned by [CodeOf] when the error has no code.
	CodeUnknown Code = ""
	// CodeOK indicates a successful Map Matching result.
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

// IsSuccess reports whether the code represents a successful Map Matching result.
func (c Code) IsSuccess() bool { return c == CodeOK }

// Error represents an HTTP-level error response from the Mapbox API.
// Use [errors.As] to unwrap, or [CodeOf] to extract the response code.
type Error struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Status is the HTTP status text (e.g. "422 Unprocessable Entity").
	Status string
	// Message is the human-readable error message parsed from the Mapbox JSON
	// response body (the "message" field). Empty if the body was not valid JSON
	// or did not contain a message field.
	Message string
	// Body is the raw response body, kept for debugging.
	Body string

	code Code
}

// Code returns the Mapbox API response code parsed from the error body, or
// [CodeUnknown] if the body did not contain a code field.
func (e *Error) Code() Code { return e.code }

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("mapbox: http %d: %s", e.StatusCode, e.Message)
	}
	if e.Body != "" {
		return fmt.Sprintf("mapbox: http %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("mapbox: http %d", e.StatusCode)
}

// CodeOf returns the Mapbox API response code from err, or [CodeUnknown] if err
// is nil, is not a Mapbox error, or the error body contained no code field.
//
// Use this to switch on specific error codes:
//
//	switch mapbox.CodeOf(err) {
//	case mapbox.CodeTooManyCoordinates:
//	    // split and retry
//	case mapbox.CodeInvalidInput:
//	    // log and discard
//	}
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.code
	}
	return CodeUnknown
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
	e := &Error{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(body),
	}
	var parsed struct {
		Code    Code   `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		e.code = parsed.Code
		e.Message = parsed.Message
	}
	return e
}
