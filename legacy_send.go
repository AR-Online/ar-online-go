package aronline

// The send contract of POST /gw/email -- the field names are the gateway's,
// Portuguese and all. The legacy area keeps the old vocabulary on purpose:
// inventing an English rendition here would create names that exist in no
// documentation anywhere.

// Anexo is an attachment, carried inline as base64.
type Anexo struct {
	Name   string `json:"name"`
	Base64 string `json:"base64"`
}

// EnvioValidation is a question/answer gate on the notification's page. It
// needs prior enablement on the account.
type EnvioValidation struct {
	Question string `json:"question,omitempty"`
	Reply    string `json:"reply,omitempty"`
}

// SMSTypeSend says when the SMS goes out.
type SMSTypeSend string

// The two moments the gateway understands.
const (
	// SMSTypeSendFallback sends the SMS only if the e-mail is not sent or
	// delivered. It is the gateway's default.
	SMSTypeSendFallback SMSTypeSend = "1"
	// SMSTypeSendAlways sends the SMS regardless of what the e-mail did.
	SMSTypeSendAlways SMSTypeSend = "2"
)

// CanalSMS is the SMS leg of a send.
type CanalSMS struct {
	// Number is the mobile number, digits only.
	Number   string      `json:"number,omitempty"`
	TypeSend SMSTypeSend `json:"typeSend,omitempty"`
	// CustomMessage takes up to 140 characters; {SHORT_LINK} is expanded by the
	// gateway.
	CustomMessage string `json:"customMessage,omitempty"`
}

// CanalWhatsApp is the WhatsApp leg of a send.
type CanalWhatsApp struct {
	Number string `json:"number,omitempty"`
	// Variables carries the custom template's variables, including the
	// "template" identifier itself.
	Variables map[string]any `json:"variables,omitempty"`
}

// CanalVoz is the voice-call leg of a send.
type CanalVoz struct {
	Number   string         `json:"number,omitempty"`
	Template string         `json:"template,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// CanalCarta is the physical-letter leg of a send.
type CanalCarta struct {
	Name      string         `json:"name,omitempty"`
	Modelo    string         `json:"modelo,omitempty"`
	Template  string         `json:"template,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
}

// EnvioRequest is what [LegacyArea.Send] takes. Despite the path saying e-mail,
// this is the MULTICHANNEL request: each optional block adds a channel to the
// same notification.
//
// Only the enumerable is typed -- see [SMSTypeSend]. Business rules (To
// required when the send is e-mail only, number formats, template existence)
// stay on the server, where they already live. A duplicated rule drifts, and
// the client's copy is the one that lies.
type EnvioRequest struct {
	// NameTo is the recipient's name.
	NameTo string `json:"nameTo"`
	// To is the recipient's e-mail. The server requires it when the send is
	// e-mail only.
	To      string `json:"to,omitempty"`
	Subject string `json:"subject"`
	// Content is the HTML body.
	Content string `json:"content"`
	// CustomID is your own reference, echoed back by the status routes.
	CustomID    string           `json:"customID,omitempty"`
	Attachments []Anexo          `json:"attachments,omitempty"`
	Validation  *EnvioValidation `json:"validation,omitempty"`
	SMS         *CanalSMS        `json:"sms,omitempty"`
	WhatsApp    *CanalWhatsApp   `json:"whatsapp,omitempty"`
	Voz         *CanalVoz        `json:"voz,omitempty"`
	Carta       *CanalCarta      `json:"carta,omitempty"`
}

// EnvioResponse is what a send answers. Processing is asynchronous: the IDEmail
// is the one handle for every later question -- status of any channel, proofs,
// the works.
type EnvioResponse struct {
	IDEmail string `json:"idEmail"`
}

// SendingProof is what [LegacyArea.SendingProof] gives back.
//
// The wire answers one of two bodies: {"content": …} with the PDF in base64, or
// {"message": …} when the e-mail has no delivery status yet. The SDK decodes
// the PDF for you and keeps the raw base64 reachable -- at most one of PDF and
// Message is filled in.
type SendingProof struct {
	// PDF is the proof, decoded. Nil while the gateway only has a message.
	PDF []byte
	// ContentBase64 is the base64 exactly as the gateway sent it. Empty when
	// there was none.
	ContentBase64 string
	// Message is the gateway's sentence when the proof is not available yet --
	// try again later. Empty when the proof came.
	Message string
}

// FinalizarReguaResult is what [LegacyArea.FinalizarRegua] answers.
type FinalizarReguaResult struct {
	Message string `json:"message"`
}
