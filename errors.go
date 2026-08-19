package aronline

import "fmt"

// ErrorDetail is one field-level complaint inside a validation error.
type ErrorDetail struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIError is a refusal from the API.
//
// Every method returns it as its error, so a failed call cannot be mistaken
// for a good one. RequestID is a first-class field on purpose: it is the first
// thing support asks for, and an SDK that swallowed it would force whoever hit
// the failure to reproduce the call with curl just to find the number.
type APIError struct {
	// Status is the HTTP status. Zero when the API was never reached.
	Status int
	// Code is the catalog code -- not_found, forbidden, rate_limited, ...
	Code string
	// Message is what the API said, in pt-BR.
	Message string
	// RequestID identifies the call in our logs.
	RequestID string
	// Field is the one at fault, when the refusal is about one.
	Field string
	// Details carries one entry per rejected field, on validation errors.
	Details []ErrorDetail
	// RetryAfter is the Retry-After header in seconds, on 429 and 503. Zero
	// when the header was absent or not a number.
	RetryAfter float64
	// Cause is the transport failure underneath, when there was one.
	Cause error
}

func (e *APIError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("aronline: %s (%d): %s", e.Code, e.Status, e.Message)
	}

	return fmt.Sprintf(
		"aronline: %s (%d): %s [request_id=%s]",
		e.Code, e.Status, e.Message, e.RequestID,
	)
}

// Unwrap exposes the transport failure to errors.Is and errors.As, so a caller
// can still reach for context.DeadlineExceeded when a call timed out.
func (e *APIError) Unwrap() error { return e.Cause }

// Retryable reports whether the API said this is worth trying again.
func (e *APIError) Retryable() bool {
	return e.Status == 429 || e.Status == 503
}
