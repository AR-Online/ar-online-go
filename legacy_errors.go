package aronline

import "fmt"

// LegacyAPIError is a refusal from the legacy gateway.
//
// The gateway has two ways of saying no: an HTTP status with a
// {"statusCode": …, "message": …} body, and -- in the templates family -- an
// HTTP 200 whose real code hides inside the {"data": …, "statusCode": …}
// envelope. Both arrive here, so a caller has one type to reach for with
// errors.As, exactly like [*APIError] on the /v3 side.
//
// It is a separate type from [*APIError] on purpose: the old contract has no
// catalog code and no request id, and a shared struct would carry empty fields
// that look like data loss.
type LegacyAPIError struct {
	// Status is the code that MATTERS: the envelope's inner statusCode when the
	// refusal came wrapped, the HTTP status otherwise. Zero when the gateway
	// was never reached.
	Status int
	// HTTPStatus is what the wire said -- 200 when the envelope hid a 404. Kept
	// apart from Status because "200 carrying a 404" is exactly the kind of
	// thing worth logging while debugging the old contract. Zero when the call
	// never left the machine.
	HTTPStatus int
	// Message is what the gateway said, in pt-BR, or what the SDK says when the
	// gateway said nothing usable.
	Message string
	// Body is the response body exactly as it came, byte for byte. Not parsed:
	// this area exists to be faithful to a contract full of idiosyncrasies, and
	// a caller who needs to see one reads it here. Nil when there was no
	// response.
	Body []byte
	// Cause is the transport or decoding failure underneath, when there was one.
	Cause error
}

func (e *LegacyAPIError) Error() string {
	if e.HTTPStatus != e.Status {
		return fmt.Sprintf(
			"aronline: legado (%d): %s [http=%d]", e.Status, e.Message, e.HTTPStatus,
		)
	}

	return fmt.Sprintf("aronline: legado (%d): %s", e.Status, e.Message)
}

// Unwrap exposes the failure underneath to errors.Is and errors.As, so a caller
// can still reach for context.DeadlineExceeded when a call timed out.
func (e *LegacyAPIError) Unwrap() error { return e.Cause }
