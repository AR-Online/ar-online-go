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
// # Two surfaces
//
// The SDK speaks two APIs. The top-level services are /v3, the new surface,
// read-only today. [Client.Legacy] is the gateway that is in production now --
// sending, per-channel status and the proofs -- and it keeps the old contract
// exactly as it is, idiosyncrasies included, because the callers it exists for
// depend on them.
//
//	client := aronline.New(aronline.Options{
//		Token:       os.Getenv("AR_TOKEN"),    // /v3
//		LegacyToken: os.Getenv("AR_GW_TOKEN"), // gateway
//	})
//
//	sent, err := client.Legacy.Send(ctx, aronline.EnvioRequest{ ... })
//	status, err := client.Legacy.Status.Email(ctx, sent.IDEmail)
//
// As /v3 grows an equivalent for a legacy route, the legacy function swaps its
// transport without changing signature: you migrate by updating the module, not
// by rewriting. Failures on the /v3 side arrive as [*APIError], on the legacy
// side as [*LegacyAPIError].
//
// The /v1 and /v2 mirrors are not covered: they answer the old contracts byte
// for byte, and a typed client that "improved" them would break the callers
// they exist to keep working.
package aronline

// DefaultBaseURL is where /v3 lives. Override it for staging or for a local
// process.
const DefaultBaseURL = "https://v3.ar-online.com.br"

// Version is this module's version.
const Version = "0.1.0"
