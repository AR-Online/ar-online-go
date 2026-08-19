// Package aronline is the official SDK for the AR Online API.
//
// You do not build URLs, set headers, unwrap envelopes or read status codes.
// You call a method, get a typed struct, and a refusal comes back as an
// [*APIError].
//
//	client := aronline.New(aronline.Options{Token: os.Getenv("AR_TOKEN")})
//
//	templates, err := client.Templates.List(ctx, aronline.TemplateFilter{
//		Channel: aronline.ChannelWhatsApp,
//	})
//
// The SDK speaks the /v3 surface only. The /v1 and /v2 mirrors answer the old
// contracts byte for byte -- idiosyncrasies included -- and a typed client
// that "improved" them would break the callers they exist to keep working.
package aronline

// DefaultBaseURL is where /v3 lives. Override it for staging or for a local
// process.
const DefaultBaseURL = "https://v3.ar-online.com.br"

// Version is this module's version.
const Version = "0.1.0"
