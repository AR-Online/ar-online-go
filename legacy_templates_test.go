package aronline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

func gwTemplatePayload() map[string]any {
	return map[string]any{
		"id":         "9b2f-uuid",
		"templateId": "hx_boleto_01",
		"nome":       "Aviso de boleto",
		"tipo":       "whatsapp",
		"conteudo":   "Olá {{1}}, …",
		"variaveis": []map[string]any{
			{"type": "body", "parameters": []map[string]any{{"name": "1"}}},
		},
		"metadata":     nil,
		"ativo":        true,
		"versao":       1,
		"criadoEm":     "2024-10-12T17:11:13.000Z",
		"atualizadoEm": nil,
		"criadoPor":    nil,
	}
}

func TestGwTemplatesListDesembrulhaOEnvelope(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": []any{gwTemplatePayload()}, "statusCode": 200})

	templates, err := fake.client().Legacy.Templates.List(
		context.Background(), aronline.GwTemplateFilter{},
	)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(templates) != 1 || templates[0].Nome != "Aviso de boleto" {
		t.Fatalf("templates = %+v", templates)
	}

	if templates[0].TemplateID == nil || *templates[0].TemplateID != "hx_boleto_01" {
		t.Errorf("templateId = %v", templates[0].TemplateID)
	}

	// Colunas que o legado nunca preencheu: nulas, e nada de lógica em cima.
	if templates[0].Metadata != nil || templates[0].CriadoPor != nil {
		t.Errorf("metadata/criadoPor = %v / %v", templates[0].Metadata, templates[0].CriadoPor)
	}

	if templates[0].Versao != 1 {
		t.Errorf("versao = %d, o versionamento nunca saiu do papel", templates[0].Versao)
	}

	if fake.received.Path != "/gw/templates" || fake.received.RawQuery != "" {
		t.Errorf("chamou %q?%q", fake.received.Path, fake.received.RawQuery)
	}
}

func TestGwTemplatesListLevaOCodigoLegadoComoFiltro(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": []any{}, "statusCode": 200})

	filter := aronline.GwTemplateFilter{Type: aronline.GwTemplateTypeWhatsApp}
	if _, err := fake.client().Legacy.Templates.List(context.Background(), filter); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if fake.received.RawQuery != "type=1" {
		t.Errorf("query = %q, queria type=1", fake.received.RawQuery)
	}
}

func TestGwTemplatesRespostaSemOEnvelopeViraErro(t *testing.T) {
	fake := newFakeGateway(t)
	// A família promete { data, statusCode } e respondeu a lista pelada.
	fake.answers(t, []any{gwTemplatePayload()})

	_, err := fake.client().Legacy.Templates.List(
		context.Background(), aronline.GwTemplateFilter{},
	)
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "envelope") {
		t.Errorf("mensagem = %q", failure.Message)
	}
}

func TestGwTemplatesDataDeOutroTipoViraErro(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": "não é lista", "statusCode": 200})

	_, err := fake.client().Legacy.Templates.List(
		context.Background(), aronline.GwTemplateFilter{},
	)
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "não casa com o tipo esperado") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if failure.Unwrap() == nil {
		t.Error("a falha de decodificação tem de continuar alcançável")
	}
}

func TestGwTemplatesDuzentosComErroDentroViraErroTipado(t *testing.T) {
	// A pegadinha nº 1 de quem integra na mão: a família responde HTTP 200 até
	// em erro, e o código que vale é o de dentro do envelope.
	cases := []struct {
		name    string
		inner   int
		erro    string
		message string
	}{
		{"não encontrado", 404, "Template não encontrado", "Template não encontrado"},
		{"de outra entidade", 403, "Acesso negado ao template", "Acesso negado ao template"},
		{"id que não é uuid", 500, "Erro ao buscar template(s)", "Erro ao buscar template(s)"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fake := newFakeGateway(t)
			fake.answers(t, map[string]any{
				"data":       map[string]any{"error": test.erro},
				"statusCode": test.inner,
			})

			_, err := fake.client().Legacy.Templates.Get(context.Background(), "9b2f-uuid")
			failure := legacyError(t, err)

			if failure.Status != test.inner {
				t.Errorf("Status = %d, queria o %d de dentro", failure.Status, test.inner)
			}

			// E o que o fio disse continua registrado: "200 carregando um 404"
			// é exatamente o que vale depurar no contrato antigo.
			if failure.HTTPStatus != http.StatusOK {
				t.Errorf("HTTPStatus = %d, queria 200", failure.HTTPStatus)
			}

			if failure.Message != test.message {
				t.Errorf("mensagem = %q", failure.Message)
			}

			if !stringsContains(string(failure.Body), test.erro) {
				t.Errorf("Body = %q, queria o corpo cru", failure.Body)
			}
		})
	}
}

func TestGwTemplatesRecusaSemErroNoEnvelopeAindaDizOCodigo(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": map[string]any{}, "statusCode": 503})

	_, err := fake.client().Legacy.Templates.Get(context.Background(), "9b2f-uuid")
	failure := legacyError(t, err)

	if failure.Status != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, queria 503", failure.Status)
	}

	if !stringsContains(failure.Message, "503") {
		t.Errorf("mensagem = %q", failure.Message)
	}
}

func TestGwTemplatesGetBuscaPeloUUIDEscapado(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": gwTemplatePayload(), "statusCode": 200})

	template, err := fake.client().Legacy.Templates.Get(context.Background(), "9b2f/uuid")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if template.ID != "9b2f-uuid" {
		t.Errorf("id = %q", template.ID)
	}

	if fake.received.EscapedPath != "/gw/templates/9b2f%2Fuuid" {
		t.Errorf("caminho no fio = %q", fake.received.EscapedPath)
	}
}

func TestGwTemplatesUpdateMandaPUTComSoOQueOGatewayDeixaEditar(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": map[string]any{"ok": true}, "statusCode": 200})

	nome := "Novo nome"
	compartilhado := true

	result, err := fake.client().Legacy.Templates.Update(
		context.Background(), "9b2f-uuid",
		aronline.GwTemplateUpdate{Nome: &nome, CompartilhadoComEntidade: &compartilhado},
	)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// O retorno das escritas passa cru: produção ainda não foi fixturada, e
	// tipo inventado sobre exemplo mentiria.
	if result["ok"] != true {
		t.Errorf("result = %+v", result)
	}

	if fake.received.Method != http.MethodPut || fake.received.Path != "/gw/templates/9b2f-uuid" {
		t.Errorf("chamou %s %q", fake.received.Method, fake.received.Path)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(fake.received.Body), &body); err != nil {
		t.Fatalf("o corpo não era JSON: %v", err)
	}

	if body["nome"] != "Novo nome" || body["compartilhadoComEntidade"] != true {
		t.Errorf("corpo = %s", fake.received.Body)
	}
}

func TestGwTemplatesUpdateSemCampoNaoMandaCampoNenhum(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": map[string]any{}, "statusCode": 200})

	_, err := fake.client().Legacy.Templates.Update(
		context.Background(), "9b2f-uuid", aronline.GwTemplateUpdate{},
	)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// Ponteiro em vez de valor: "não mexa no nome" e "apague o nome" são
	// pedidos diferentes, e só o ponteiro sabe dizer qual é qual.
	if fake.received.Body != "{}" {
		t.Errorf("corpo = %q, queria {}", fake.received.Body)
	}
}

func TestGwTemplatesDeactivateEhDELETESemCorpo(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": map[string]any{"ok": true}, "statusCode": 200})

	if _, err := fake.client().Legacy.Templates.Deactivate(
		context.Background(), "9b2f-uuid",
	); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if fake.received.Method != http.MethodDelete {
		t.Errorf("método = %q, queria DELETE", fake.received.Method)
	}

	if fake.received.Body != "" {
		t.Errorf("corpo = %q, queria vazio", fake.received.Body)
	}

	if fake.received.ContentType != "" {
		t.Errorf("Content-Type = %q, sem corpo não há tipo", fake.received.ContentType)
	}
}

func TestGwTemplatesSetStatusEhPATCHEmStatus(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"data": map[string]any{"ok": true}, "statusCode": 200})

	if _, err := fake.client().Legacy.Templates.SetStatus(
		context.Background(), "9b2f-uuid", false,
	); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if fake.received.Method != http.MethodPatch {
		t.Errorf("método = %q, queria PATCH", fake.received.Method)
	}

	if fake.received.Path != "/gw/templates/9b2f-uuid/status" {
		t.Errorf("caminho = %q", fake.received.Path)
	}

	if fake.received.Body != `{"ativo":false}` {
		t.Errorf("corpo = %q", fake.received.Body)
	}
}

func TestGwTemplatesEscritaPropagaARecusaDoEnvelope(t *testing.T) {
	cases := []struct {
		name string
		call func(*aronline.Client) error
	}{
		{"Update", func(c *aronline.Client) error {
			_, err := c.Legacy.Templates.Update(
				context.Background(), "x", aronline.GwTemplateUpdate{},
			)
			return err
		}},
		{"Deactivate", func(c *aronline.Client) error {
			_, err := c.Legacy.Templates.Deactivate(context.Background(), "x")
			return err
		}},
		{"SetStatus", func(c *aronline.Client) error {
			_, err := c.Legacy.Templates.SetStatus(context.Background(), "x", true)
			return err
		}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fake := newFakeGateway(t)
			fake.answers(t, map[string]any{
				"data":       map[string]any{"error": "Template não encontrado"},
				"statusCode": 404,
			})

			failure := legacyError(t, test.call(fake.client()))

			if failure.Status != http.StatusNotFound || failure.HTTPStatus != http.StatusOK {
				t.Errorf("status = %d / http = %d", failure.Status, failure.HTTPStatus)
			}
		})
	}
}

func TestGwTemplates401CruDoGateway(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusUnauthorized, map[string]any{
		"message": "Unauthorized", "statusCode": 401,
	})

	_, err := fake.client().Legacy.Templates.List(
		context.Background(), aronline.GwTemplateFilter{},
	)
	failure := legacyError(t, err)

	if failure.Status != http.StatusUnauthorized || failure.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("status = %d / http = %d", failure.Status, failure.HTTPStatus)
	}

	if failure.Message != "Unauthorized" {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if !stringsContains(string(failure.Body), `"statusCode":401`) {
		t.Errorf("Body = %q, queria o corpo cru", failure.Body)
	}
}
