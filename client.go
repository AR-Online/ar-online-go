package aronline

import (
	"net/http"
	"strings"
	"time"
)

// Options configures a [Client]. The zero value is usable: it points at
// production with no credential, which is enough for [VersionService.Get].
type Options struct {
	// Token is the RS256 token issued by AR Online. Every route except
	// /v3/version refuses with 401 without it.
	Token string
	// BaseURL defaults to [DefaultBaseURL].
	BaseURL string
	// Timeout defaults to [DefaultTimeout]. Ignored when HTTPClient is set --
	// that client carries its own.
	Timeout time.Duration
	// HTTPClient replaces the one the SDK would build. Set it to reuse your
	// own pool, your own proxy, or a stub in tests.
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
}

// New builds a client.
func New(options Options) *Client {
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
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

	return &Client{
		Templates: &TemplatesService{transport: shared},
		Tags:      &TagsService{transport: shared},
		Allowlist: &AllowlistService{transport: shared},
		Freshness: &FreshnessService{transport: shared},
		Version:   &VersionService{transport: shared},
	}
}
