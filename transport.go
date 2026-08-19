package aronline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout is how long a call waits before giving up.
const DefaultTimeout = 30 * time.Second

// transport is the only part of the SDK that knows HTTP exists. Everything
// above it -- the services, the client -- deals in methods and structs. That
// is the whole point of the package.
type transport struct {
	baseURL string
	token   string
	http    *http.Client
}

// envelope decodes the routes that answer {"data": ...} -- templates, tags and
// allowlist -- into out.
//
// Not every route wraps its answer, which is the one trap of this API:
// freshness and version answer the object itself. Unwrapping all of them, or
// none, breaks half the calls, so the choice is made per route by whichever
// method the service calls.
func (t *transport) envelope(ctx context.Context, path string, query url.Values, out any) error {
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}

	if err := t.do(ctx, path, query, true, &wrapper); err != nil {
		return err
	}

	if wrapper.Data == nil {
		return &APIError{
			Status:  http.StatusOK,
			Code:    "invalid_response",
			Message: fmt.Sprintf("%s respondeu sem o envelope 'data' que a rota promete", path),
		}
	}

	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return &APIError{
			Status:  http.StatusOK,
			Code:    "invalid_response",
			Message: fmt.Sprintf("%s respondeu um 'data' que não casa com o tipo esperado", path),
			Cause:   err,
		}
	}

	return nil
}

// bare decodes the routes that answer the object directly -- freshness and
// version -- into out.
func (t *transport) bare(ctx context.Context, path string, authenticated bool, out any) error {
	return t.do(ctx, path, nil, authenticated, out)
}

func (t *transport) do(
	ctx context.Context,
	path string,
	query url.Values,
	authenticated bool,
	out any,
) error {
	// Refused here, before the socket: a 401 round trip teaches nothing that
	// the missing token does not already say.
	if authenticated && t.token == "" {
		return &APIError{
			Status:  http.StatusUnauthorized,
			Code:    "unauthenticated",
			Message: fmt.Sprintf("%s exige um token; construa o cliente com Options{Token: ...}", path),
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url(path, query), http.NoBody)
	if err != nil {
		return &APIError{
			Status:  0,
			Code:    "invalid_request",
			Message: fmt.Sprintf("não foi possível montar a requisição para %s", path),
			Cause:   err,
		}
	}

	request.Header.Set("Accept", "application/json")

	if authenticated {
		request.Header.Set("Authorization", "Bearer "+t.token)
	}

	response, err := t.http.Do(request)
	if err != nil {
		// Timeout, cancellation and connection refused arrive as opaque
		// transport errors. They become the SDK's own error so a caller has
		// one type to inspect -- and Unwrap keeps errors.Is working on the
		// original.
		return &APIError{
			Status:  0,
			Code:    "unreachable",
			Message: fmt.Sprintf("não foi possível falar com %s: %v", t.baseURL, err),
			Cause:   err,
		}
	}

	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return &APIError{
			Status:  response.StatusCode,
			Code:    "unreachable",
			Message: fmt.Sprintf("a resposta de %s foi interrompida no meio", path),
			Cause:   err,
		}
	}

	if response.StatusCode >= 400 {
		return toAPIError(response, body)
	}

	// A 2xx that is not JSON is something other than the API answering. It
	// fails like any other refusal -- a raw json.SyntaxError leaking out here
	// would send whoever hit it looking for a bug in their own code.
	if err := json.Unmarshal(body, out); err != nil {
		return &APIError{
			Status:    response.StatusCode,
			Code:      "invalid_response",
			Message:   "a resposta não é o JSON que a rota promete — algo respondeu no lugar da API",
			RequestID: response.Header.Get("X-Request-Id"),
			Cause:     err,
		}
	}

	return nil
}

func (t *transport) url(path string, query url.Values) string {
	address := t.baseURL + path

	if len(query) > 0 {
		address += "?" + query.Encode()
	}

	return address
}

// toAPIError turns a refused response into an *APIError.
//
// A refusal that does not carry the envelope is still a refusal: a proxy
// answering 502 in HTML has to fail the same way, or whoever hit it goes
// looking for a bug in their own parsing code.
func toAPIError(response *http.Response, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Code      string        `json:"code"`
			Message   string        `json:"message"`
			RequestID string        `json:"request_id"`
			Field     string        `json:"field"`
			Details   []ErrorDetail `json:"details"`
		} `json:"error"`
	}

	_ = json.Unmarshal(body, &envelope)

	failure := &APIError{
		Status:     response.StatusCode,
		Code:       envelope.Error.Code,
		Message:    envelope.Error.Message,
		RequestID:  envelope.Error.RequestID,
		Field:      envelope.Error.Field,
		Details:    envelope.Error.Details,
		RetryAfter: retryAfter(response.Header.Get("Retry-After")),
	}

	if failure.Code == "" {
		failure.Code = "invalid_response"
		failure.Message = fmt.Sprintf(
			"a API respondeu %d sem o corpo de erro esperado", response.StatusCode,
		)
	}

	if failure.RequestID == "" {
		failure.RequestID = response.Header.Get("X-Request-Id")
	}

	return failure
}

// retryAfter reads the header as seconds.
//
// Retry-After also has an HTTP-date form. The API only ever sends seconds, so
// anything else is dropped rather than guessed at: a wrong delay is worse than
// no delay.
func retryAfter(header string) float64 {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(header), 64)
	if err != nil || seconds < 0 {
		return 0
	}

	return seconds
}
