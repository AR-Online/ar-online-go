package aronline

import "context"

// AllowlistEntry is a recipient allowed to receive messages.
type AllowlistEntry struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	// CreatedAt is ISO 8601 with the real offset.
	CreatedAt string `json:"created_at"`
}

// AllowlistService reaches the recipients allowed to receive messages.
//
// The legacy called this a whitelist and answered it under the key "leads", a
// copy-paste that became contract. Here it is an allowlist, and the name says
// what the list holds. Like labels, it is personal -- an integration token
// gets 403.
type AllowlistService struct {
	transport *transport
}

// List answers your allowed recipients, ordered by recipient.
func (s *AllowlistService) List(ctx context.Context) ([]AllowlistEntry, error) {
	var entries []AllowlistEntry
	if err := s.transport.envelope(ctx, "/v3/allowlist", nil, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}
