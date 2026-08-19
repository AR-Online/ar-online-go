package aronline_test

import (
	"context"
	"net/http"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

func templatePayload() map[string]any {
	return map[string]any{
		"id":                  "9b2f-uuid",
		"name":                "Aviso de boleto",
		"channel":             "whatsapp",
		"subject":             nil,
		"body":                "Olá {{1}}, seu boleto vence em {{2}}.",
		"active":              true,
		"provider_identifier": "hx_boleto_01",
		"variables": []map[string]any{
			{"name": "1", "component": "body", "type": "text"},
		},
		"created_at": "2024-10-12T14:11:13-03:00",
		"updated_at": nil,
	}
}

func TestTemplatesListDesembrulhaEDecodifica(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{templatePayload()}})

	templates, err := fake.client().Templates.List(context.Background(), aronline.TemplateFilter{})
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(templates) != 1 {
		t.Fatalf("veio %d modelo(s), queria 1", len(templates))
	}

	template := templates[0]

	if template.ID != "9b2f-uuid" || template.Name != "Aviso de boleto" {
		t.Errorf("modelo decodificado errado: %+v", template)
	}

	// subject é null na API — tem de virar ponteiro nil, não string vazia:
	// "sem assunto" e "assunto em branco" são coisas diferentes.
	if template.Subject != nil {
		t.Errorf("subject = %v, queria nil", *template.Subject)
	}

	if template.ProviderIdentifier == nil || *template.ProviderIdentifier != "hx_boleto_01" {
		t.Errorf("provider_identifier = %v", template.ProviderIdentifier)
	}

	if len(template.Variables) != 1 || template.Variables[0].Name != "1" {
		t.Errorf("variables = %+v", template.Variables)
	}

	if fake.received.Path != "/v3/templates" || fake.received.RawQuery != "" {
		t.Errorf("caminho = %q?%q", fake.received.Path, fake.received.RawQuery)
	}
}

func TestTemplatesListLevaOCanal(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{}})

	filter := aronline.TemplateFilter{Channel: aronline.ChannelWhatsApp}
	if _, err := fake.client().Templates.List(context.Background(), filter); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if fake.received.RawQuery != "channel=whatsapp" {
		t.Errorf("query = %q, queria channel=whatsapp", fake.received.RawQuery)
	}
}

func TestTemplatesGetEscapaOID(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": templatePayload()})

	if _, err := fake.client().Templates.Get(context.Background(), "../../ops/health"); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// O que viajou no fio: as barras escapadas. É esta a asserção que prova o
	// escape — `Path` já vem decodificado pelo servidor, e passaria mesmo se
	// o id tivesse virado outra rota.
	if fake.received.EscapedPath != "/v3/templates/..%2F..%2Fops%2Fhealth" {
		t.Errorf("caminho no fio = %q", fake.received.EscapedPath)
	}
}

func TestTemplatesGetQuatrocentosEQuatro(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{
		"code": "not_found", "message": "Modelo não encontrado.",
	}, nil)

	_, err := fake.client().Templates.Get(context.Background(), "sumido")

	if got := apiError(t, err).Status; got != http.StatusNotFound {
		t.Errorf("status = %d, queria 404", got)
	}
}

func TestTagsListDesembrulha(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{
		map[string]any{
			"id": "12", "name": "urgente", "color": "#ff0000",
			"created_at": "2024-10-12T14:11:13-03:00",
		},
	}})

	tags, err := fake.client().Tags.List(context.Background())
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(tags) != 1 || tags[0].Name != "urgente" {
		t.Errorf("tags = %+v", tags)
	}

	if fake.received.Path != "/v3/tags" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestTagsTokenDeIntegracaoRecebe403(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusForbidden, map[string]any{
		"code":    "forbidden",
		"message": "Esta rota responde a token de pessoa: etiquetas são pessoais.",
	}, nil)

	_, err := fake.client().Tags.List(context.Background())
	failure := apiError(t, err)

	// 403, e não lista vazia: vazio leria como "você não tem nenhuma".
	if failure.Status != http.StatusForbidden || failure.Code != "forbidden" {
		t.Errorf("recusa = %+v", failure)
	}
}

func TestAllowlistList(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{
		map[string]any{
			"id": "7", "recipient": "alguem@exemplo.com.br",
			"created_at": "2024-10-12T14:11:13-03:00",
		},
	}})

	entries, err := fake.client().Allowlist.List(context.Background())
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(entries) != 1 || entries[0].Recipient != "alguem@exemplo.com.br" {
		t.Errorf("entries = %+v", entries)
	}

	if fake.received.Path != "/v3/allowlist" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestFreshnessVemDiretoSemEnvelope(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{
		"refreshed_at":        "2026-08-18T11:42:03-03:00",
		"last_load_at":        "2026-08-18T11:40:00-03:00",
		"worst_lag_seconds":   34904,
		"tables_tracked":      46,
		"tables_never_loaded": 2,
		"behind": []map[string]any{
			{"legacy": "geral.ger_voz", "lag_seconds": 34904},
		},
	})

	freshness, err := fake.client().Freshness.Get(context.Background())
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if freshness.TablesTracked != 46 || freshness.TablesNeverLoaded != 2 {
		t.Errorf("freshness = %+v", freshness)
	}

	if freshness.WorstLagSeconds == nil || *freshness.WorstLagSeconds != 34904 {
		t.Errorf("worst_lag_seconds = %v", freshness.WorstLagSeconds)
	}

	if len(freshness.Behind) != 1 || freshness.Behind[0].Legacy != "geral.ger_voz" {
		t.Errorf("behind = %+v", freshness.Behind)
	}

	if fake.received.Path != "/v3/freshness" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestFreshnessSemMarcaDeLeituraVemNulo(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{
		"refreshed_at": nil, "last_load_at": nil, "worst_lag_seconds": nil,
		"tables_tracked": 46, "tables_never_loaded": 46, "behind": []any{},
	})

	freshness, err := fake.client().Freshness.Get(context.Background())
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// nil, e não 0: "nenhuma tabela tem marca" não é "está em dia".
	if freshness.WorstLagSeconds != nil {
		t.Errorf("worst_lag_seconds = %v, queria nil", *freshness.WorstLagSeconds)
	}
}

func TestVersionRespondeSemToken(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{
		"version": "0.1.0", "min_migration": "0025_credentials.sql", "environment": "local",
	})

	info, err := fake.anonymous().Version.Get(context.Background())
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if info.Version != "0.1.0" || info.Environment != "local" {
		t.Errorf("version = %+v", info)
	}

	if fake.received.Authorization != "" {
		t.Errorf("mandou credencial: %q", fake.received.Authorization)
	}
}

func TestChannelsEhOQueAV3Aceita(t *testing.T) {
	// A lista é duplicada do servidor. Este teste é o que torna a duplicação
	// segura: canal novo lá e não aqui faria o SDK recusar valor que a API
	// aceita, e quem chamou culparia o SDK, com razão.
	want := []aronline.Channel{"email", "sms", "whatsapp", "voice", "letter"}

	if len(aronline.Channels) != len(want) {
		t.Fatalf("Channels = %v, queria %v", aronline.Channels, want)
	}

	for i, channel := range want {
		if aronline.Channels[i] != channel {
			t.Errorf("Channels[%d] = %q, queria %q", i, aronline.Channels[i], channel)
		}
	}
}
