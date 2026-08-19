package aronline

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
)

// LegacyArea is the legacy gateway, as functions -- everything
// docs.ar-online.com.br documents, spoken exactly as the old API speaks it.
//
// This area exists so an integration written against the old contract gets
// typed calls today. As /v3 grows an equivalent for a route, the function here
// swaps its transport without changing shape -- the migration happens under
// your feet, not in your code, and each swap is recorded in the CHANGELOG. Each
// function's documentation names its /v3 equivalent when one exists.
//
// The credential is not the /v3 one: it is the gateway's JWT, in
// Options.LegacyToken. A call without it fails before the socket, naming the
// option that is missing.
type LegacyArea struct {
	// Status holds the per-channel status routes and the consolidated view.
	Status *LegacyStatusService
	// Templates holds the gateway's template routes. The /v3 equivalent for
	// reads is [Client.Templates].
	Templates *LegacyTemplatesService

	transport *legacyTransport
}

// Send sends a notification -- POST /gw/email, the MULTICHANNEL route despite
// the name. Processing is asynchronous: keep the returned IDEmail, it is the
// handle for every status and proof question later.
//
// No /v3 equivalent yet.
//
// The request travels by value, big as it is: the call site reads
// Send(ctx, aronline.EnvioRequest{...}), and a pointer parameter would make
// every caller write an ampersand for nothing -- the SDK never mutates it.
//
//nolint:gocritic // hugeParam: by value on purpose, see above.
func (a *LegacyArea) Send(ctx context.Context, request EnvioRequest) (*EnvioResponse, error) {
	var sent EnvioResponse

	if err := a.transport.decode(
		ctx, http.MethodPost, "/gw/email", request, &sent,
	); err != nil {
		return nil, err
	}

	return &sent, nil
}

// SendingProof answers the sending proof as a PDF.
//
// The wire carries it in base64 inside JSON; this decodes it for you and keeps
// the raw string reachable. While the e-mail has no delivery status the gateway
// answers a message instead, and PDF comes back nil -- that is not an error,
// ask again later.
//
// No /v3 equivalent yet.
func (a *LegacyArea) SendingProof(ctx context.Context, id string) (*SendingProof, error) {
	var wire struct {
		Content string `json:"content"`
		Message string `json:"message"`
	}

	path := "/gw/sending-proof/" + url.PathEscape(id)
	if err := a.transport.decode(ctx, http.MethodGet, path, nil, &wire); err != nil {
		return nil, err
	}

	if wire.Content == "" {
		return &SendingProof{Message: wire.Message}, nil
	}

	pdf, err := base64.StdEncoding.DecodeString(wire.Content)
	if err != nil {
		return nil, &LegacyAPIError{
			Status:     http.StatusOK,
			HTTPStatus: http.StatusOK,
			Message:    "o comprovante veio com base64 ilegível",
			Body:       []byte(wire.Content),
			Cause:      err,
		}
	}

	return &SendingProof{PDF: pdf, ContentBase64: wire.Content}, nil
}

// Laudo answers the expert-evidence report -- the one route that hands back the
// PDF binary directly, no base64, no JSON. A missing record still refuses in
// JSON, and still arrives as a [*LegacyAPIError].
//
// No /v3 equivalent yet.
func (a *LegacyArea) Laudo(ctx context.Context, id string) ([]byte, error) {
	return a.transport.binary(ctx, "/gw/email/laudo/"+url.PathEscape(id))
}

// FinalizarRegua stops the notification ladder for this send.
//
// A GET with a side effect -- that is the old contract, and the SDK does not
// "fix" it to POST. Fixing it would mean a call that works in curl and not
// here.
//
// No /v3 equivalent yet.
func (a *LegacyArea) FinalizarRegua(
	ctx context.Context,
	id string,
) (*FinalizarReguaResult, error) {
	var result FinalizarReguaResult

	path := "/regua-notificacao/finalizar/" + url.PathEscape(id)
	if err := a.transport.decode(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
