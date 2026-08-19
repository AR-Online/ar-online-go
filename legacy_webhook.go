package aronline

import "encoding/json"

// The SDK does not receive HTTP for you -- webhooks arrive at YOUR endpoint.
// These are the payload types, exported so whoever receives them does not have
// to type the contract by hand.

// WebhookChannel is a channel a webhook event names.
type WebhookChannel string

// The channels a v2 event can carry.
const (
	WebhookChannelEmail    WebhookChannel = "email"
	WebhookChannelSMS      WebhookChannel = "sms"
	WebhookChannelWhatsApp WebhookChannel = "whatsapp"
	WebhookChannelVoz      WebhookChannel = "voz"
	WebhookChannelCarta    WebhookChannel = "carta"
)

// WebhookPayloadV1 is the default payload, delivered unless v2 was enabled with
// support.
//
// On failure events the three dates come null together -- hence the pointers,
// without omitempty: the keys are always there.
type WebhookPayloadV1 struct {
	// NotificationID is the notification's uuid -- the same idEmail the API
	// answers on send.
	NotificationID string  `json:"notificationID"`
	Channel        string  `json:"channel"`
	Description    string  `json:"description"`
	DateSent       *string `json:"dateSent"`
	DateDelivery   *string `json:"dateDelivery"`
	DateRead       *string `json:"dateRead"`
	LogDate        string  `json:"logDate"`
}

// WebhookMetadata is the delivery bookkeeping of a v2 event.
type WebhookMetadata struct {
	WebhookVersion string `json:"webhookVersion"`
	// Attempt is the delivery attempt -- up to 4 with the retry schedule.
	Attempt int `json:"attempt"`
}

// WebhookPayloadV2 is the enriched payload, enabled by asking support.
type WebhookPayloadV2 struct {
	EventVersion string `json:"eventVersion"`
	// OccurredAt is an ISO 8601 timestamp of the event itself. Unlike the
	// legacy dates, this one does carry an offset.
	OccurredAt string `json:"occurredAt"`
	// NotificationID is the notification's uuid -- the same idEmail the API
	// answers on send.
	NotificationID  string         `json:"notificationID"`
	Channel         WebhookChannel `json:"channel"`
	Status          string         `json:"status"`
	StatusTimestamp *string        `json:"statusTimestamp,omitempty"`
	// Payload mirrors the answer of the channel's own status route. It stays
	// raw because the shape depends on Channel: read Channel first, then
	// unmarshal this into [StatusEmail], [StatusSMS], [StatusWhatsApp],
	// [StatusVoz] or [StatusCarta]. A union would cost every reader the fields
	// of four channels they did not get.
	Payload  json.RawMessage `json:"payload"`
	Metadata WebhookMetadata `json:"metadata"`
}
