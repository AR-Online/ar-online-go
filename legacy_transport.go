package aronline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// DefaultLegacyBaseURL is where the legacy gateway lives. It is a different
// address from [DefaultBaseURL], and overriding one does not touch the other.
const DefaultLegacyBaseURL = "https://api.ar-online.com.br"

// legacyTransport is the one place that knows how the legacy gateway talks.
//
// Two things make it a different transport from the /v3 one rather than a flag
// on the same one: the credential goes RAW in the authorization header -- no
// "Bearer" -- and success is not the HTTP status. The templates family answers
// 200 with the real code inside the body, and the voice status answers 200 with
// a sentence where the other channels answer 404. Each method here decides by
// the contract of the route that called it.
type legacyTransport struct {
	baseURL string
	token   string
	http    *http.Client
}

// decode calls a route that answers JSON and refuses with an HTTP status, and
// unmarshals the answer into out.
//
// No query parameter here: outside the templates family, which goes through
// [legacyTransport.envelope], no legacy route takes one.
func (t *legacyTransport) decode(
	ctx context.Context,
	method, path string,
	body, out any,
) error {
	status, raw, err := t.exchange(ctx, method, path, nil, body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, out); err != nil {
		// A 2xx that is not JSON is something other than the gateway answering
		// -- a proxy, a captive portal, a login page. A raw json.SyntaxError
		// leaking out here would send whoever hit it looking for a bug in their
		// own code.
		return &LegacyAPIError{
			Status:     status,
			HTTPStatus: status,
			Message:    "a resposta não é JSON — algo respondeu no lugar do gateway",
			Body:       raw,
			Cause:      err,
		}
	}

	return nil
}

// envelope calls a route of the /gw/templates family, which wraps everything in
// {"data": …, "statusCode": …} and answers HTTP 200 EVEN ON ERROR.
//
// The code that matters is the inner one; reading only the HTTP status is the
// single most common integration bug against this family, and unwrapping it
// here is exactly what the legacy area abstracts.
func (t *legacyTransport) envelope(
	ctx context.Context,
	method, path string,
	query url.Values,
	body, out any,
) error {
	status, raw, err := t.exchange(ctx, method, path, query, body)
	if err != nil {
		return err
	}

	var wrapper struct {
		Data       json.RawMessage `json:"data"`
		StatusCode *int            `json:"statusCode"`
	}

	if err := json.Unmarshal(raw, &wrapper); err != nil || wrapper.Data == nil {
		return &LegacyAPIError{
			Status:     status,
			HTTPStatus: status,
			Message: fmt.Sprintf(
				"%s respondeu sem o envelope { data, statusCode } que a família promete", path,
			),
			Body:  raw,
			Cause: err,
		}
	}

	inner := status
	if wrapper.StatusCode != nil {
		inner = *wrapper.StatusCode
	}

	if inner >= 400 {
		message := envelopeError(wrapper.Data)
		if message == "" {
			message = fmt.Sprintf("o gateway recusou com %d", inner)
		}

		return &LegacyAPIError{Status: inner, HTTPStatus: status, Message: message, Body: raw}
	}

	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return &LegacyAPIError{
			Status:     inner,
			HTTPStatus: status,
			Message:    fmt.Sprintf("%s respondeu um 'data' que não casa com o tipo esperado", path),
			Body:       raw,
			Cause:      err,
		}
	}

	return nil
}

// binary calls the one route that answers a file instead of JSON -- the laudo.
// A refusal still arrives as JSON and still becomes a [*LegacyAPIError].
func (t *legacyTransport) binary(ctx context.Context, path string) ([]byte, error) {
	_, raw, err := t.exchange(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// exchange runs one request and hands back the HTTP status and the raw body,
// having already turned every refusal into a [*LegacyAPIError].
func (t *legacyTransport) exchange(
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
) (status int, raw []byte, err error) {
	// Refused here, before the socket: a 401 round trip teaches nothing that
	// the missing token does not already say. Every legacy route needs the
	// credential -- the gateway has no open route.
	if t.token == "" {
		return 0, nil, &LegacyAPIError{
			Status:     http.StatusUnauthorized,
			HTTPStatus: 0,
			Message: fmt.Sprintf(
				"%s exige o token do gateway; construa o cliente com Options{LegacyToken: ...}", path,
			),
		}
	}

	payload := io.Reader(http.NoBody)

	if body != nil {
		encoded, failure := json.Marshal(body)
		if failure != nil {
			return 0, nil, &LegacyAPIError{
				Status:     0,
				HTTPStatus: 0,
				Message:    fmt.Sprintf("não foi possível montar o corpo da requisição para %s", path),
				Cause:      failure,
			}
		}

		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, t.url(path, query), payload)
	if err != nil {
		return 0, nil, &LegacyAPIError{
			Status:     0,
			HTTPStatus: 0,
			Message:    fmt.Sprintf("não foi possível montar a requisição para %s", path),
			Cause:      err,
		}
	}

	request.Header.Set("Accept", "application/json")
	// Raw on purpose: the gateway wants the JWT WITHOUT "Bearer", the exact
	// opposite of /v3. Prefixing it here would turn every call into the
	// gateway's own 401.
	request.Header.Set("Authorization", t.token)

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := t.http.Do(request)
	if err != nil {
		return 0, nil, &LegacyAPIError{
			Status:     0,
			HTTPStatus: 0,
			Message:    fmt.Sprintf("não foi possível falar com %s: %v", t.baseURL, err),
			Cause:      err,
		}
	}

	defer func() { _ = response.Body.Close() }()

	raw, err = io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, &LegacyAPIError{
			Status:     response.StatusCode,
			HTTPStatus: response.StatusCode,
			Message:    fmt.Sprintf("a resposta de %s foi interrompida no meio", path),
			Cause:      err,
		}
	}

	if response.StatusCode >= 400 {
		return 0, nil, refusal(response.StatusCode, raw)
	}

	return response.StatusCode, raw, nil
}

func (t *legacyTransport) url(path string, query url.Values) string {
	address := t.baseURL + path

	if len(query) > 0 {
		address += "?" + query.Encode()
	}

	return address
}

// refusal turns an HTTP-level refusal into a [*LegacyAPIError].
//
// The gateway's error body is {"statusCode": …, "message": …}; a proxy
// answering 502 in HTML carries neither, and has to fail the same way, or
// whoever hit it goes looking for a bug in their own parsing code.
func refusal(httpStatus int, raw []byte) *LegacyAPIError {
	var shape struct {
		StatusCode *int            `json:"statusCode"`
		Message    json.RawMessage `json:"message"`
	}

	_ = json.Unmarshal(raw, &shape)

	failure := &LegacyAPIError{
		Status:     httpStatus,
		HTTPStatus: httpStatus,
		Message:    fmt.Sprintf("o gateway respondeu %d sem o corpo de erro esperado", httpStatus),
		Body:       raw,
	}

	if shape.StatusCode != nil {
		failure.Status = *shape.StatusCode
	}

	if message := readMessage(shape.Message); message != "" {
		failure.Message = message
	}

	return failure
}

// readMessage reads the gateway's `message`, which is a string on most routes
// and a LIST of strings when the validation layer refuses several fields at
// once. The list is kept as its JSON text rather than dropped: a caller who
// sees three complaints is better served than one who sees the generic
// sentence.
func readMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	return string(raw)
}

// envelopeError reads the templates family's error, which sits inside `data` as
// {"error": "Template não encontrado"}. Anything else stays raw on the error.
func envelopeError(data json.RawMessage) string {
	var shape struct {
		Error string `json:"error"`
	}

	_ = json.Unmarshal(data, &shape)

	return shape.Error
}
