package aronline

import (
	"context"
	"net/http"
	"net/url"
)

// GwTemplateType is a template type code the legacy gateway understands.
//
// An unknown code answers an EMPTY LIST, not an error -- if you expect results
// and get none, check the code first.
type GwTemplateType string

// The four codes the gateway knows.
const (
	GwTemplateTypeWhatsApp GwTemplateType = "1"
	GwTemplateTypeEmail    GwTemplateType = "2"
	GwTemplateTypeSMS      GwTemplateType = "3"
	GwTemplateTypeCarta    GwTemplateType = "4"
)

// GwTemplateTypes is the same list, for anyone validating before calling.
var GwTemplateTypes = []GwTemplateType{
	GwTemplateTypeWhatsApp, GwTemplateTypeEmail, GwTemplateTypeSMS, GwTemplateTypeCarta,
}

// GwTemplate is a template as the legacy gateway shapes it.
type GwTemplate struct {
	// ID is the public UUID.
	ID string `json:"id"`
	// TemplateID is the provider's identifier, e.g. hx_boleto_01.
	TemplateID *string `json:"templateId"`
	Nome       string  `json:"nome"`
	// Tipo is the channel as a word: whatsapp, email, sms, carta. It is not the
	// numeric code the filter takes.
	Tipo      string           `json:"tipo"`
	Conteudo  string           `json:"conteudo"`
	Variaveis []map[string]any `json:"variaveis"`
	// Metadata is always null -- the legacy column was 100% null. Do not build
	// logic on it.
	Metadata any  `json:"metadata"`
	Ativo    bool `json:"ativo"`
	// Versao is always 1 -- template versioning never shipped. Do not build
	// logic on it.
	Versao int `json:"versao"`
	// CriadoEm looks ISO with a Z, but the Z is not really UTC -- see the API
	// docs' concepts page. It stays a string for that reason.
	CriadoEm     string  `json:"criadoEm"`
	AtualizadoEm *string `json:"atualizadoEm"`
	// CriadoPor is always null -- the legacy column was 100% null.
	CriadoPor any `json:"criadoPor"`
}

// GwTemplateWriteResult is what the write routes answer, passed through
// untyped.
//
// Production has not been fixtured for the writes yet, so the SDK hands the
// object over as it came rather than promise fields it has never seen. The type
// tightens when the mirror proves the shape.
type GwTemplateWriteResult = map[string]any

// GwTemplateFilter narrows [LegacyTemplatesService.List]. The zero value asks
// for every type.
type GwTemplateFilter struct {
	Type GwTemplateType
}

// GwTemplateUpdate is what [LegacyTemplatesService.Update] changes -- the two
// fields the gateway lets you edit.
//
// Both are pointers so that "leave it alone" and "set it to empty/false" are
// different requests: a nil field is left out of the body entirely.
type GwTemplateUpdate struct {
	Nome                     *string `json:"nome,omitempty"`
	CompartilhadoComEntidade *bool   `json:"compartilhadoComEntidade,omitempty"`
}

// LegacyTemplatesService reaches the gateway's template routes.
//
// The whole family answers through the {"data": …, "statusCode": …} envelope
// with HTTP 200 EVEN ON ERROR; the transport unwraps it and turns the inner
// 403/404/500 into a [*LegacyAPIError], so none of that reaches you.
//
// The /v3 equivalent for the reads is [Client.Templates] -- same database row,
// clean contract. The writes have no /v3 equivalent yet.
//
// The version routes (GET /{id}/versions and /{id}/versions/{v}) are
// deliberately absent: production answers empty or 404 for every template, and
// a function that never finds anything only invites integration against a dead
// resource.
type LegacyTemplatesService struct {
	transport *legacyTransport
}

// List answers your entity's templates and the ones shared with it, newest
// first.
func (s *LegacyTemplatesService) List(
	ctx context.Context,
	filter GwTemplateFilter,
) ([]GwTemplate, error) {
	query := url.Values{}
	if filter.Type != "" {
		query.Set("type", string(filter.Type))
	}

	var templates []GwTemplate
	if err := s.transport.envelope(
		ctx, http.MethodGet, "/gw/templates", query, nil, &templates,
	); err != nil {
		return nil, err
	}

	return templates, nil
}

// Get answers one template by its public UUID. Someone else's answers the
// family's 403 -- inside the envelope, and out here as a [*LegacyAPIError].
func (s *LegacyTemplatesService) Get(ctx context.Context, id string) (*GwTemplate, error) {
	var template GwTemplate

	path := "/gw/templates/" + url.PathEscape(id)
	if err := s.transport.envelope(ctx, http.MethodGet, path, nil, nil, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

// Update edits name and entity-wide sharing -- the two things the gateway lets
// you touch.
func (s *LegacyTemplatesService) Update(
	ctx context.Context,
	id string,
	changes GwTemplateUpdate,
) (GwTemplateWriteResult, error) {
	path := "/gw/templates/" + url.PathEscape(id)

	return writeTemplate(ctx, s.transport, http.MethodPut, path, changes)
}

// Deactivate is a soft delete: the template deactivates, the row stays.
func (s *LegacyTemplatesService) Deactivate(
	ctx context.Context,
	id string,
) (GwTemplateWriteResult, error) {
	path := "/gw/templates/" + url.PathEscape(id)

	return writeTemplate(ctx, s.transport, http.MethodDelete, path, nil)
}

// SetStatus turns a template on or off without deleting anything.
func (s *LegacyTemplatesService) SetStatus(
	ctx context.Context,
	id string,
	ativo bool,
) (GwTemplateWriteResult, error) {
	path := "/gw/templates/" + url.PathEscape(id) + "/status"

	return writeTemplate(ctx, s.transport, http.MethodPatch, path, map[string]bool{"ativo": ativo})
}

// writeTemplate is the shape the three write routes share.
func writeTemplate(
	ctx context.Context,
	transport *legacyTransport,
	method, path string,
	body any,
) (GwTemplateWriteResult, error) {
	result := GwTemplateWriteResult{}
	if err := transport.envelope(ctx, method, path, nil, body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
