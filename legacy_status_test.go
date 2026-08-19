package aronline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

func TestStatusEmailPreservaAsDuasConvencoesDeAusencia(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"dateSend":       "18/07/2026 01:01:32",
		"dateDelivery":   "",
		"dateReading":    nil,
		"dateAcceptance": nil,
		"error":          false,
		"description":    "Enviado",
		"failureReason":  nil,
		"customID":       "pedido-4471",
		"idEmail":        "c62582cc-fc79-4ef5-a20d-27a8476b651d",
	})

	status, err := fake.client().Legacy.Status.Email(context.Background(), "c62582cc")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// "" e null são convenções DIFERENTES na mesma resposta, e ficam como
	// vieram: quem testar só `== nil` nos quatro perde metade.
	if status.DateSend != "18/07/2026 01:01:32" || status.DateDelivery != "" {
		t.Errorf("datas por string vazia = %q / %q", status.DateSend, status.DateDelivery)
	}

	if status.DateReading != nil || status.DateAcceptance != nil {
		t.Errorf("datas por null = %v / %v", status.DateReading, status.DateAcceptance)
	}

	// A data do legado fica string: "18/07/2026 01:01:32" não traz fuso, e
	// nenhum instante inequívoco sai daí.
	if _, ok := any(status.DateSend).(string); !ok {
		t.Error("a data do legado tem de ser string")
	}

	if status.CustomID == nil || *status.CustomID != "pedido-4471" {
		t.Errorf("customID = %v", status.CustomID)
	}

	if fake.received.Method != http.MethodGet || fake.received.Path != "/gw/email/c62582cc" {
		t.Errorf("chamou %s %q", fake.received.Method, fake.received.Path)
	}
}

func TestStatusEmailRemarshalaCadaConvencaoDeVolta(t *testing.T) {
	// A prova de que as duas convenções são distinguíveis no tipo: o que entrou
	// como "" volta como "", o que entrou como null volta como null, e a chave
	// que não veio continua sem vir.
	wire := `{"dateSend":"","dateDelivery":"","dateReading":null,"dateAcceptance":null,` +
		`"error":false,"description":"Processado","failureReason":null,"customID":null,` +
		`"idEmail":"c62582cc"}`

	var status aronline.StatusEmail
	if err := json.Unmarshal([]byte(wire), &status); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if string(encoded) != wire {
		t.Errorf("remarshalou %s,\nqueria      %s", encoded, wire)
	}
}

func TestStatusEmailDeOutraPessoaChegaComoO404DoGateway(t *testing.T) {
	fake := newFakeGateway(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{"message": "E-mail não encontrado"})

	_, err := fake.client().Legacy.Status.Email(context.Background(), "sumido")
	failure := legacyError(t, err)

	if failure.Status != http.StatusNotFound || failure.HTTPStatus != http.StatusNotFound {
		t.Errorf("status = %d / http = %d", failure.Status, failure.HTTPStatus)
	}

	if failure.Message != "E-mail não encontrado" {
		t.Errorf("mensagem = %q", failure.Message)
	}
}

func TestStatusEscapaOID(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{})

	if _, err := fake.client().Legacy.Status.Email(context.Background(), "../full/x"); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// O que viajou no fio: as barras escapadas. É esta a asserção que prova o
	// escape — `Path` já vem decodificado pelo servidor.
	if fake.received.EscapedPath != "/gw/email/..%2Ffull%2Fx" {
		t.Errorf("caminho no fio = %q", fake.received.EscapedPath)
	}
}

func TestStatusSMSAnsweredEhListaDeObjetos(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"description":  "Entregue",
		"dateSend":     "18/07/2026 01:01:32",
		"dateReading":  nil,
		"dateAnswered": nil,
		"answered": []map[string]any{
			{"resposta": "SIM", "em": "18/07/2026 02:00:00"},
		},
	})

	status, err := fake.client().Legacy.Status.SMS(context.Background(), "c62582cc")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// A documentação antiga diz lista de texto e está errada: o fio traz
	// OBJETO, e quem integrou leu o objeto.
	if len(status.Answered) != 1 || status.Answered[0]["resposta"] != "SIM" {
		t.Errorf("answered = %+v", status.Answered)
	}

	if fake.received.Path != "/gw/sms/c62582cc" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestStatusWhatsAppDataQueNaoAconteceuSomeDaResposta(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"description":   "Enviado",
		"dateSent":      "18/07/2026 01:01:32",
		"error":         false,
		"failureReason": nil,
		"customID":      nil,
		"idEmail":       "c62582cc-fc79-4ef5-a20d-27a8476b651d",
	})

	status, err := fake.client().Legacy.Status.WhatsApp(context.Background(), "c62582cc")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if status.DateSent == nil || *status.DateSent != "18/07/2026 01:01:32" {
		t.Errorf("dateSent = %v", status.DateSent)
	}

	if status.DateDelivery != nil {
		t.Errorf("dateDelivery = %v, a chave nem veio", *status.DateDelivery)
	}

	// customID vem SEMPRE nulo nesta rota, mesmo quando a mensagem tem um.
	if status.CustomID != nil {
		t.Errorf("customID = %v, o WhatsApp responde sempre nulo", *status.CustomID)
	}

	// E a chave ausente volta ausente: `omitempty` do lado certo é o que
	// distingue "sumiu" de "veio null".
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if stringsContains(string(encoded), "dateDelivery") {
		t.Errorf("remarshalou com a chave que tinha sumido: %s", encoded)
	}

	if !stringsContains(string(encoded), `"customID":null`) {
		t.Errorf("remarshalou sem o null que veio: %s", encoded)
	}

	if fake.received.Path != "/gw/whatsapp/c62582cc" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestStatusVozSemRegistroResponde200ENaoEhErro(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"description": "Não há registro de voz para este envio"})

	status, err := fake.client().Legacy.Status.Voz(context.Background(), "qualquer")
	// A voz é a única rota que NUNCA responde 404. Uuid sem registro é 200 com
	// uma frase, e isso não é falha.
	if err != nil {
		t.Fatalf("uuid sem registro de voz não é erro: %v", err)
	}

	if status.Description != "Não há registro de voz para este envio" {
		t.Errorf("description = %q", status.Description)
	}

	if status.DateSuccessCall != nil || status.DateFailureCall != nil {
		t.Errorf("sem registro não tem data: %+v", status)
	}

	if fake.received.Path != "/gw/voz/qualquer" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestStatusCartaTrazAsEtapasRenomeadasDoContrato(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"description":     "Entregue",
		"error":           false,
		"dateProcessing":  "05/08/2025 11:00:15",
		"datePreparation": "05/08/2025 11:02:40",
		"dateSent":        "27/08/2025 15:23:57",
		"dateDelivery":    "02/09/2025 10:11:00",
		"sro":             "YQ694562879BR",
		"linkRastreio":    "https://rastreamento.correios.com.br/YQ694562879BR",
	})

	status, err := fake.client().Legacy.Status.Carta(context.Background(), "c62582cc")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// O provedor produz datePrepared/dateDelivered; a resposta traz
	// datePreparation/dateDelivery. Os nomes do provedor não chegam ao cliente.
	if status.DatePreparation == nil || *status.DatePreparation != "05/08/2025 11:02:40" {
		t.Errorf("datePreparation = %v", status.DatePreparation)
	}

	if status.DateDelivery == nil || *status.DateDelivery != "02/09/2025 10:11:00" {
		t.Errorf("dateDelivery = %v", status.DateDelivery)
	}

	if status.SRO == nil || *status.SRO != "YQ694562879BR" {
		t.Errorf("sro = %v", status.SRO)
	}

	if status.LinkArCartaComprovante != nil {
		t.Errorf("linkArCartaComprovante = %v, a chave nem veio", *status.LinkArCartaComprovante)
	}

	if fake.received.Path != "/gw/carta/c62582cc" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestStatusFullTrazHistoricoUltimoStatusEOsBlocosPorCanal(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{
		"codEmail": 12345,
		"statusFull": map[string]any{
			"email": []map[string]any{{"label": "Enviado", "dateTime": "14/05/2025 17:04:44"}},
		},
		"lastStatus": map[string]any{
			"email": map[string]any{"label": "Enviado", "dateTime": "14/05/2025 17:04:44"},
		},
		"email":    []map[string]any{{"subject": "Documento", "remetente": "noreply@empresa.com"}},
		"sms":      []any{},
		"whatsapp": []any{},
		"voz":      []any{},
		"carta":    []any{},
	})

	full, err := fake.client().Legacy.Status.Full(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if full.CodEmail != 12345 {
		t.Errorf("codEmail = %d", full.CodEmail)
	}

	if len(full.StatusFull.Email) != 1 || full.StatusFull.Email[0].Label != "Enviado" {
		t.Errorf("statusFull.email = %+v", full.StatusFull.Email)
	}

	if full.LastStatus.Email == nil || full.LastStatus.Email.DateTime != "14/05/2025 17:04:44" {
		t.Errorf("lastStatus.email = %+v", full.LastStatus.Email)
	}

	// Os blocos periciais passam crus: a documentação pública os mostra por
	// exemplo, e tipo inventado sobre exemplo prometeria campo nunca provado.
	if len(full.Email) != 1 || full.Email[0]["subject"] != "Documento" {
		t.Errorf("email = %+v", full.Email)
	}

	if fake.received.Path != "/gw/full/f6cb58f2" {
		t.Errorf("caminho = %q", fake.received.Path)
	}
}

func TestStatusFullComObjetoVazioEhAQuartaConvencaoDeAusencia(t *testing.T) {
	fake := newFakeGateway(t)
	// Envio só de e-mail: os canais que não existiram vêm como {} — a quarta
	// convenção, ao lado de "", null e a chave que some.
	fake.answers(t, map[string]any{
		"codEmail":   999,
		"statusFull": map[string]any{},
		"lastStatus": map[string]any{},
		"email":      []any{},
		"sms":        []any{},
		"whatsapp":   []any{},
		"voz":        []any{},
		"carta":      []any{},
	})

	full, err := fake.client().Legacy.Status.Full(context.Background(), "f6cb58f2")
	if err != nil {
		t.Fatalf("{} não é erro: %v", err)
	}

	if full.LastStatus.SMS != nil || full.StatusFull.Carta != nil {
		t.Errorf("{} tinha de deixar tudo zerado: %+v", full)
	}
}

func TestCadaRotaDeStatusPropagaARecusa(t *testing.T) {
	// Uma tabela e não seis testes: o que se prova é que NENHUMA rota engole a
	// falha pelo caminho.
	cases := []struct {
		name string
		call func(*aronline.Client) error
	}{
		{"status.Email", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.Email(context.Background(), "x")
			return err
		}},
		{"status.SMS", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.SMS(context.Background(), "x")
			return err
		}},
		{"status.WhatsApp", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.WhatsApp(context.Background(), "x")
			return err
		}},
		{"status.Voz", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.Voz(context.Background(), "x")
			return err
		}},
		{"status.Carta", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.Carta(context.Background(), "x")
			return err
		}},
		{"status.Full", func(c *aronline.Client) error {
			_, err := c.Legacy.Status.Full(context.Background(), "x")
			return err
		}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fake := newFakeGateway(t)
			fake.refuses(t, http.StatusInternalServerError, map[string]any{
				"statusCode": 500, "message": "Erro interno",
			})

			failure := legacyError(t, test.call(fake.client()))

			if failure.Status != http.StatusInternalServerError {
				t.Errorf("status = %d, queria 500", failure.Status)
			}
		})
	}
}
