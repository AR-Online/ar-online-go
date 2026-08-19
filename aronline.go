// Package aronline is the official SDK for the AR Online API.
//
// The SDK speaks the /v3 surface only. The /v1 and /v2 mirrors answer the old
// contracts byte for byte -- idiosyncrasies included -- and a typed client
// that "improved" them would break the callers they exist to keep working.
package aronline

// DefaultBaseURL is where /v3 lives. Override it to point at staging or at a
// local process.
const DefaultBaseURL = "https://api.aronline.com.br"

// Version is this module's version.
const Version = "0.1.0"
