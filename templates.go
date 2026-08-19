package aronline

import (
	"context"
	"net/url"
)

// Channel is a channel a template can belong to.
//
// The list is closed on the server (decision A11): a value outside it answers
// 400 invalid_value with the accepted ones in the message, never a silent
// empty list.
type Channel string

// The channels the template filter accepts.
const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelVoice    Channel = "voice"
	ChannelLetter   Channel = "letter"
)

// Channels is the same list, for anyone validating before calling.
var Channels = []Channel{
	ChannelEmail, ChannelSMS, ChannelWhatsApp, ChannelVoice, ChannelLetter,
}

// TemplateVariable is a placeholder a template expects to be filled in.
type TemplateVariable struct {
	Name string `json:"name"`
	// Component is where the provider puts it. Empty when the channel has no
	// components.
	Component *string `json:"component"`
	Type      *string `json:"type"`
}

// Template is a message template.
type Template struct {
	// ID is the public UUID. The internal id never leaves the API.
	ID      string `json:"id"`
	Name    string `json:"name"`
	Channel string `json:"channel"`
	// Subject is filled in only by channels that carry one.
	Subject            *string `json:"subject"`
	Body               string  `json:"body"`
	Active             bool    `json:"active"`
	ProviderIdentifier *string `json:"provider_identifier"`
	// Variables is a flat list -- the provider's component nesting stays in
	// the mirror.
	Variables []TemplateVariable `json:"variables"`
	// CreatedAt is ISO 8601 with the real offset, e.g. 2024-10-12T14:11:13-03:00.
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

// TemplateFilter narrows [TemplatesService.List]. The zero value asks for
// every channel.
type TemplateFilter struct {
	Channel Channel
}

// TemplatesService reaches the message templates.
type TemplatesService struct {
	transport *transport
}

// List answers the templates your identity reaches, newest first.
//
// Unlike the /gw/templates mirror, it does not filter by lineage: a template
// created in /v3 has no legacy row and still shows up.
func (s *TemplatesService) List(ctx context.Context, filter TemplateFilter) ([]Template, error) {
	query := url.Values{}
	if filter.Channel != "" {
		query.Set("channel", string(filter.Channel))
	}

	var templates []Template
	if err := s.transport.envelope(ctx, "/v3/templates", query, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

// Get answers one template by its public UUID.
//
// A template that does not exist and one that is not yours fail the same way
// -- 404. Answering 403 would confirm the id exists.
func (s *TemplatesService) Get(ctx context.Context, id string) (*Template, error) {
	var template Template

	path := "/v3/templates/" + url.PathEscape(id)
	if err := s.transport.envelope(ctx, path, nil, &template); err != nil {
		return nil, err
	}

	return &template, nil
}
