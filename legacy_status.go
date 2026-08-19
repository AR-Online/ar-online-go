package aronline

import (
	"context"
	"net/http"
	"net/url"
)

// The per-channel answers of GET /gw/<canal>/{idEmail}.
//
// This family says "not yet" four different ways -- "", null, a key that
// vanishes, and {} -- sometimes two of them in the same response. The structs
// keep each convention where the wire has it, because normalizing them would
// break the fidelity the legacy area exists to give.
//
// How the four look in Go, and why:
//
//   - "" (empty string): a plain string field, no omitempty. The wire always
//     sends the key, and "" is the value.
//   - null: a *string with NO omitempty. Nil is the null, and re-encoding the
//     struct puts the null back.
//   - the key vanishes: a *string WITH omitempty. Nil is the absence, and
//     re-encoding leaves the key out instead of writing null.
//   - {}: an empty struct or a nil slice, on the consolidated view.
//
// Reading a nil pointer does not tell you which of null and absent it was --
// Go's decoder cannot, both leave nil -- but the field's own tag does, it is
// fixed per field by the contract, and a struct round-tripped through
// encoding/json comes out with the same convention it went in with.
//
// In every route the id is the notification's uuid -- the e-mail's. There is no
// per-channel id.

// StatusEmail is the status do AR-Email.
//
// DateSend and DateDelivery come as "" until they happen; DateReading and
// DateAcceptance come as null. Empty string and nil mean the same thing here --
// checking only for nil misses half of them.
type StatusEmail struct {
	DateSend       string  `json:"dateSend"`
	DateDelivery   string  `json:"dateDelivery"`
	DateReading    *string `json:"dateReading"`
	DateAcceptance *string `json:"dateAcceptance"`
	Error          bool    `json:"error"`
	// Description climbs with the stage reached: Processado, Enviado, Entregue,
	// Lido.
	Description   string  `json:"description"`
	FailureReason *string `json:"failureReason"`
	// FailureReasonDescription is the full description of the failure, filled
	// in together with FailureReason. The key is absent on a healthy send.
	FailureReasonDescription *string `json:"failureReasonDescription,omitempty"`
	CustomID                 *string `json:"customID"`
	IDEmail                  string  `json:"idEmail"`
}

// SMSAnswer is one entry of [StatusSMS].Answered. The old documentation says a
// list of strings; the wire carries OBJECTS, and whoever integrated read the
// objects -- so the object is what stays.
type SMSAnswer map[string]any

// StatusSMS is the status do AR-SMS.
type StatusSMS struct {
	// Description: "Lido (acessou o link)" beats every other label when the
	// link was opened.
	Description string `json:"description"`
	// DateSend comes as "" until it happens.
	DateSend     string      `json:"dateSend"`
	DateReading  *string     `json:"dateReading"`
	DateAnswered *string     `json:"dateAnswered"`
	Answered     []SMSAnswer `json:"answered"`
}

// StatusWhatsApp is the status do AR-WhatsApp.
//
// The dates that have not happened VANISH from the response instead of coming
// null -- hence the omitempty on the four of them.
type StatusWhatsApp struct {
	Description    string  `json:"description"`
	DateSent       *string `json:"dateSent,omitempty"`
	DateDelivery   *string `json:"dateDelivery,omitempty"`
	DateResponse   *string `json:"dateResponse,omitempty"`
	DateAccessLink *string `json:"dateAccessLink,omitempty"`
	Error          bool    `json:"error"`
	FailureReason  *string `json:"failureReason"`
	// CustomID is always null on this route, even when the message has one --
	// read it on the e-mail route.
	CustomID *string `json:"customID"`
	IDEmail  string  `json:"idEmail"`
}

// StatusVoz is the status do AR-Voz.
//
// The one route that never answers 404: an unknown uuid gets a 200 with only
// Description -- "Não há registro de voz para este envio". And when a call
// failed before succeeding, the answer tells only the failure: DateSuccessCall
// never travels together with DateFailureCall.
type StatusVoz struct {
	Description     string  `json:"description"`
	DateSent        *string `json:"dateSent,omitempty"`
	DateSuccessCall *string `json:"dateSuccessCall,omitempty"`
	DateFailureCall *string `json:"dateFailureCall,omitempty"`
	// LinkCall is the recording's link -- it depends on a data load that may
	// lag behind.
	LinkCall *string `json:"linkCall,omitempty"`
}

// StatusCarta is the status do AR-Cartas.
//
// Two stages change name on the way out: the provider produces datePrepared and
// dateDelivered, the response carries datePreparation and dateDelivery. The
// provider's names never reach the client.
type StatusCarta struct {
	Description     string  `json:"description"`
	Error           bool    `json:"error"`
	DateProcessing  *string `json:"dateProcessing,omitempty"`
	DatePreparation *string `json:"datePreparation,omitempty"`
	DateSent        *string `json:"dateSent,omitempty"`
	DateDelivery    *string `json:"dateDelivery,omitempty"`
	// SRO is the Correios tracking code.
	SRO                    *string `json:"sro,omitempty"`
	LinkArCartaComprovante *string `json:"linkArCartaComprovante,omitempty"`
	LinkRastreio           *string `json:"linkRastreio,omitempty"`
}

// StatusEvent is one status event: the label and when it happened, in BRT.
type StatusEvent struct {
	Label string `json:"label"`
	// DateTime is dd/mm/aaaa hh:mm:ss, Brasília time. It stays a string: the
	// format carries no offset, so no unambiguous instant can be built from it.
	DateTime string `json:"dateTime"`
}

// FullChannelDetail is a channel's detail block inside the consolidated status,
// passed through as it came.
type FullChannelDetail map[string]any

// StatusFullHistory is the full status history of each channel. A channel with
// nothing to tell has its key absent, and the slice stays nil.
type StatusFullHistory struct {
	Email    []StatusEvent `json:"email,omitempty"`
	SMS      []StatusEvent `json:"sms,omitempty"`
	WhatsApp []StatusEvent `json:"whatsapp,omitempty"`
	Voz      []StatusEvent `json:"voz,omitempty"`
	Carta    []StatusEvent `json:"carta,omitempty"`
}

// StatusFullLast is each channel's latest status. An unused channel has its key
// absent, and the pointer stays nil -- this is the {} convention: a send with
// only the e-mail leg answers lastStatus with just one key, or none at all.
type StatusFullLast struct {
	Email    *StatusEvent `json:"email,omitempty"`
	SMS      *StatusEvent `json:"sms,omitempty"`
	WhatsApp *StatusEvent `json:"whatsapp,omitempty"`
	Voz      *StatusEvent `json:"voz,omitempty"`
	Carta    *StatusEvent `json:"carta,omitempty"`
}

// StatusFull is the status completo -- the forensic view.
//
// StatusFull and LastStatus are typed precisely; the per-channel detail slices
// are passed through as they come. They carry the expert-evidence material --
// signed timestamps, reading trails, geolocation -- in provider-shaped nests
// that the public documentation shows by example rather than by schema, and a
// type invented on top of an example would promise fields this SDK has never
// proven.
type StatusFull struct {
	// CodEmail is the e-mail's internal numeric code on the platform.
	CodEmail   int                 `json:"codEmail"`
	StatusFull StatusFullHistory   `json:"statusFull"`
	LastStatus StatusFullLast      `json:"lastStatus"`
	Email      []FullChannelDetail `json:"email"`
	SMS        []FullChannelDetail `json:"sms"`
	WhatsApp   []FullChannelDetail `json:"whatsapp"`
	Voz        []FullChannelDetail `json:"voz"`
	Carta      []FullChannelDetail `json:"carta"`
}

// LegacyStatusService answers "what happened to this notification", one route
// per channel -- the most used surface of the old API.
//
// Every method takes the same id: the notification's uuid, the one Send
// answered. Asking for the SMS is asking for the SMS OF THAT NOTIFICATION;
// there is no per-channel id to keep.
//
// An unknown id answers 404 and arrives as a [*LegacyAPIError] -- except on
// [LegacyStatusService.Voz], where the old API answers 200 with a sentence, and
// so does this.
//
// No /v3 equivalent yet.
type LegacyStatusService struct {
	transport *legacyTransport
}

// Email answers when it went out, when the recipient's server accepted it, and
// when it was read.
func (s *LegacyStatusService) Email(ctx context.Context, id string) (*StatusEmail, error) {
	return getStatus[StatusEmail](ctx, s.transport, "/gw/email/", id)
}

// SMS answers what happened to the SMS, and the recipient's answers when there
// were any.
func (s *LegacyStatusService) SMS(ctx context.Context, id string) (*StatusSMS, error) {
	return getStatus[StatusSMS](ctx, s.transport, "/gw/sms/", id)
}

// WhatsApp answers the WhatsApp leg's stages.
func (s *LegacyStatusService) WhatsApp(ctx context.Context, id string) (*StatusWhatsApp, error) {
	return getStatus[StatusWhatsApp](ctx, s.transport, "/gw/whatsapp/", id)
}

// Voz answers the voice call's stages. Never a 404: no record answers 200 with
// only a Description, and that is not an error.
func (s *LegacyStatusService) Voz(ctx context.Context, id string) (*StatusVoz, error) {
	return getStatus[StatusVoz](ctx, s.transport, "/gw/voz/", id)
}

// Carta answers the letter's stages, preparation to delivery, with the Correios
// tracking.
func (s *LegacyStatusService) Carta(ctx context.Context, id string) (*StatusCarta, error) {
	return getStatus[StatusCarta](ctx, s.transport, "/gw/carta/", id)
}

// Full answers every channel's forensic data in one call. For following a
// single channel's current stage, the per-channel routes are the lighter ask.
func (s *LegacyStatusService) Full(ctx context.Context, id string) (*StatusFull, error) {
	return getStatus[StatusFull](ctx, s.transport, "/gw/full/", id)
}

// getStatus is the shape all six routes share: GET <prefix><id>, decoded into
// T. Generic rather than six copies, so an escaping mistake cannot happen in
// only one of them.
func getStatus[T any](
	ctx context.Context,
	transport *legacyTransport,
	prefix, id string,
) (*T, error) {
	var status T

	path := prefix + url.PathEscape(id)
	if err := transport.decode(ctx, http.MethodGet, path, nil, &status); err != nil {
		return nil, err
	}

	return &status, nil
}
