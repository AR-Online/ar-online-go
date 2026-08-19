package aronline

import (
	"net/http"
	"strings"
	"time"
)

// Options configures a [Client]. The zero value is usable: it points at
// production with no credential, which is enough for [VersionService.Get].
//
// There are two credentials because there are two surfaces, and they are not
// interchangeable: Token is the /v3 one, LegacyToken is the gateway's. Fill in
// only the one you use; neither ever travels to the other's address.
type Options struct {
	// Token is the RS256 token issued by AR Online. Every /v3 route except
	// /v3/version refuses with 401 without it.
	Token string
	// LegacyToken is the gateway's JWT -- a DIFFERENT credential, used by
	// [Client.Legacy] and sent raw, without "Bearer". Every legacy route needs
	// it; the gateway has no open route.
	LegacyToken string
	// BaseURL defaults to [DefaultBaseURL].
	BaseURL string
	// LegacyBaseURL defaults to [DefaultLegacyBaseURL]. It is independent of
	// BaseURL: pointing one at a test environment leaves the other alone.
	LegacyBaseURL string
	// Timeout defaults to [DefaultTimeout] and covers both surfaces. Ignored
	// when HTTPClient is set -- that client carries its own.
	Timeout time.Duration
	// HTTPClient replaces the one the SDK would build. Set it to reuse your
	// own pool, your own proxy, or a stub in tests. Both surfaces use it.
	HTTPClient *http.Client
}

// Client is the AR Online API client -- the one thing you construct.
//
// It owns the transport and hands it to each service; the services are the
// public surface. Nothing above this line knows that HTTP is involved.
type Client struct {
	// Templates holds the message templates your identity reaches.
	Templates *TemplatesService
	// Tags holds your labels.
	Tags *TagsService
	// Allowlist holds your allowed recipients.
	Allowlist *AllowlistService
	// Freshness tells how far behind the copy of the data is.
	Freshness *FreshnessService
	// Version tells which version is running. The one route that needs no token.
	Version *VersionService
	// Legacy is the gateway's surface -- sending, per-channel status, proofs
	// and the gateway's own templates. It talks to another address with another
	// credential; see [LegacyArea].
	Legacy *LegacyArea
}

// New builds a client.
//
// Options travels by value, big as it grew with the second surface's address
// and credential: the call site reads New(aronline.Options{...}), and a pointer
// parameter would make every caller write an ampersand for nothing -- the SDK
// reads the struct once and never keeps it.
//
//nolint:gocritic // hugeParam: by value on purpose, see above.
func New(options Options) *Client {
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	legacyBaseURL := options.LegacyBaseURL
	if legacyBaseURL == "" {
		legacyBaseURL = DefaultLegacyBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		timeout := options.Timeout
		if timeout == 0 {
			timeout = DefaultTimeout
		}

		httpClient = &http.Client{Timeout: timeout}
	}

	shared := &transport{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   options.Token,
		http:    httpClient,
	}

	// A transport of its own, not a flag on the shared one: the legacy gateway
	// wants another address, another credential and another way of saying no.
	legacy := &legacyTransport{
		baseURL: strings.TrimRight(legacyBaseURL, "/"),
		token:   options.LegacyToken,
		http:    httpClient,
	}

	return &Client{
		Templates: &TemplatesService{transport: shared},
		Tags:      &TagsService{transport: shared},
		Allowlist: &AllowlistService{transport: shared},
		Freshness: &FreshnessService{transport: shared},
		Version:   &VersionService{transport: shared},
		Legacy: &LegacyArea{
			Status:    &LegacyStatusService{transport: legacy},
			Templates: &LegacyTemplatesService{transport: legacy},
			transport: legacy,
		},
	}
}
