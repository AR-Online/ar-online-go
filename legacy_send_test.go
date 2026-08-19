package aronline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

func TestSendPostaOCorpoComoJSONEDevolveOIDEmail(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"idEmail": "8c4813f5-8430-4ad4-ab72-19d7eed39731"})

	request := aronline.EnvioRequest{
		NameTo:  "João da Silva",
		To:      "joao@exemplo.com",
		Subject: "Documento importante",
		Content: "<p>Você recebeu um documento.</p>",
		SMS:     &aronline.CanalSMS{Number: "11999998888", TypeSend: aronline.SMSTypeSendFallback},
	}

	sent, err := fake.client().Legacy.Send(context.Background(), request)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if sent.IDEmail != "8c4813f5-8430-4ad4-ab72-19d7eed39731" {
		t.Errorf("idEmail = %q", sent.IDEmail)
	}

	// Apesar do caminho dizer e-mail, é a rota MULTICANAL.
	if fake.received.Method != http.MethodPost || fake.received.Path != "/gw/email" {
		t.Errorf("chamou %s %q", fake.received.Method, fake.received.Path)
	}

	if fake.received.ContentType != "application/json" {
		t.Errorf("Content-Type = %q", fake.received.ContentType)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(fake.received.Body), &body); err != nil {
		t.Fatalf("o corpo não era JSON: %v", err)
	}

	// Os campos vazios não viajam: um `to` em branco no corpo é recusa do
	// servidor por um campo que quem chamou nem informou.
	if _, ok := body["customID"]; ok {
		t.Errorf("mandou campo que ninguém preencheu: %s", fake.received.Body)
	}

	sms, ok := body["sms"].(map[string]any)
	if !ok || sms["typeSend"] != "1" {
		t.Errorf("bloco sms = %v", body["sms"])
	}
}

func TestSendLevaOsCincoCanaisNoMesmoCorpo(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"idEmail": "qualquer"})

	_, err := fake.client().Legacy.Send(context.Background(), aronline.EnvioRequest{
		NameTo:      "João da Silva",
		To:          "joao@exemplo.com",
		Subject:     "Documento",
		Content:     "<p>oi</p>",
		CustomID:    "contrato-4471",
		Attachments: []aronline.Anexo{{Name: "contrato.pdf", Base64: "JVBERi0x"}},
		Validation:  &aronline.EnvioValidation{Question: "Seu CPF?", Reply: "12345678909"},
		SMS: &aronline.CanalSMS{
			Number:        "11999998888",
			TypeSend:      aronline.SMSTypeSendAlways,
			CustomMessage: "Você recebeu um AR-Email. Acesse: {SHORT_LINK}",
		},
		WhatsApp: &aronline.CanalWhatsApp{
			Number:    "11999998888",
			Variables: map[string]any{"template": "aviso_01"},
		},
		Voz:   &aronline.CanalVoz{Number: "1133334444", Template: "aviso_voz"},
		Carta: &aronline.CanalCarta{Name: "João da Silva", Modelo: "padrao"},
	})
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(fake.received.Body), &body); err != nil {
		t.Fatalf("o corpo não era JSON: %v", err)
	}

	for _, canal := range []string{"sms", "whatsapp", "voz", "carta", "validation", "attachments"} {
		if _, ok := body[canal]; !ok {
			t.Errorf("o bloco %q não viajou: %s", canal, fake.received.Body)
		}
	}
}

func TestSendRecusaDoGatewayViraLegacyAPIErrorComOCorpoCru(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusBadRequest, map[string]any{
		"statusCode": 400,
		"message":    "O número do destinatário informado é inválido, Verifique o número inserido.",
	})

	_, err := fake.client().Legacy.Send(context.Background(), aronline.EnvioRequest{
		NameTo: "A", Subject: "B", Content: "C",
	})
	failure := legacyError(t, err)

	if failure.Status != http.StatusBadRequest || failure.HTTPStatus != http.StatusBadRequest {
		t.Errorf("status = %d / http = %d", failure.Status, failure.HTTPStatus)
	}

	if !stringsContains(failure.Message, "número do destinatário") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if !stringsContains(string(failure.Body), `"statusCode":400`) {
		t.Errorf("Body = %q, queria o corpo cru", failure.Body)
	}
}

func TestSendRecusaComListaDeMensagensNaoPerdeAsQueixas(t *testing.T) {
	// A camada de validação do gateway responde `message` como LISTA quando
	// recusa vários campos. Perder as três e mostrar a frase genérica deixaria
	// quem chamou sem saber o que consertar.
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusBadRequest, map[string]any{
		"statusCode": 400,
		"message":    []string{"to must be an email", "subject should not be empty"},
	})

	_, err := fake.client().Legacy.Send(context.Background(), aronline.EnvioRequest{NameTo: "A"})
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "must be an email") {
		t.Errorf("mensagem = %q, queria carregar as queixas", failure.Message)
	}
}

func TestSendRecusaSemCorpoDeErroAindaDizOStatus(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answersRaw(http.StatusInternalServerError, "text/plain", "")

	_, err := fake.client().Legacy.Send(context.Background(), aronline.EnvioRequest{NameTo: "A"})
	failure := legacyError(t, err)

	if failure.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, queria 500", failure.Status)
	}

	if !stringsContains(failure.Message, "sem o corpo de erro esperado") {
		t.Errorf("mensagem = %q", failure.Message)
	}
}

func TestOsCodigosDeTipoSaoOsQuatroDoLegado(t *testing.T) {
	want := []aronline.GwTemplateType{"1", "2", "3", "4"}

	if len(aronline.GwTemplateTypes) != len(want) {
		t.Fatalf("GwTemplateTypes = %v", aronline.GwTemplateTypes)
	}

	for i, code := range want {
		if aronline.GwTemplateTypes[i] != code {
			t.Errorf("GwTemplateTypes[%d] = %q, queria %q", i, aronline.GwTemplateTypes[i], code)
		}
	}
}

func TestOsTiposDeWebhookExistemParaQuemRecebe(t *testing.T) {
	// O SDK não recebe HTTP por você; o que ele evita é você digitar o contrato
	// à mão do outro lado.
	v1 := `{"notificationID":"c62582cc","channel":"email","description":"Entregue",
		"dateSent":"18/07/2026 01:01:32","dateDelivery":null,"dateRead":null,
		"logDate":"18/07/2026 01:02:00"}`

	var evento aronline.WebhookPayloadV1
	if err := json.Unmarshal([]byte(v1), &evento); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if evento.NotificationID != "c62582cc" || evento.DateDelivery != nil {
		t.Errorf("v1 = %+v", evento)
	}

	v2 := `{"eventVersion":"2.0","occurredAt":"2026-07-18T01:02:00-03:00",
		"notificationID":"c62582cc","channel":"voz","status":"Atendida",
		"payload":{"description":"Atendida"},"metadata":{"webhookVersion":"v2","attempt":1}}`

	var enriquecido aronline.WebhookPayloadV2
	if err := json.Unmarshal([]byte(v2), &enriquecido); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if enriquecido.Channel != aronline.WebhookChannelVoz || enriquecido.Metadata.Attempt != 1 {
		t.Errorf("v2 = %+v", enriquecido)
	}

	// O payload fica cru porque a forma depende do canal: leia Channel, depois
	// decodifique na struct daquele canal.
	var voz aronline.StatusVoz
	if err := json.Unmarshal(enriquecido.Payload, &voz); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if voz.Description != "Atendida" {
		t.Errorf("payload da voz = %+v", voz)
	}
}
