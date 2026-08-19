package aronline_test

import "errors"

// errorsAs exists so the assertions read as one line. It is errors.As, and
// nothing else: the point is that *APIError is reachable through the standard
// unwrapping, not through a type switch the SDK would have to document.
func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
