package aronline_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	aronline "github.com/AR-Online/ar-online-go"
)

func TestCabecalhoMontaOBearerSozinho(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{}})

	if _, err := fake.client().Tags.List(context.Background()); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if got := fake.received.Authorization; got != "Bearer tok" {
		t.Errorf("Authorization = %q, queria %q", got, "Bearer tok")
	}

	if got := fake.received.Accept; got != "application/json" {
		t.Errorf("Accept = %q, queria application/json", got)
	}

	if got := fake.received.Method; got != http.MethodGet {
		t.Errorf("método = %q, queria GET", got)
	}
}

func TestRotaAbertaNaoMandaCredencial(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"version": "0.1.0"})

	if _, err := fake.client().Version.Get(context.Background()); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if got := fake.received.Authorization; got != "" {
		t.Errorf("mandou credencial em rota aberta: %q", got)
	}
}

func TestEnvelopeAusenteEhRecusado(t *testing.T) {
	fake := newFakeAPI(t)
	// A rota promete {"data": ...} e respondeu outra coisa.
	fake.answers(t, map[string]any{"tags": []any{}})

	_, err := fake.client().Tags.List(context.Background())

	if got := apiError(t, err).Code; got != "invalid_response" {
		t.Errorf("code = %q, queria invalid_response", got)
	}
}

func TestRecusaCarregaTudoQueOCorpoTraz(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusNotFound, map[string]any{
		"code":       "not_found",
		"message":    "Etiqueta não encontrada.",
		"request_id": "req-do-corpo",
	}, nil)

	_, err := fake.client().Tags.Get(context.Background(), "1")
	failure := apiError(t, err)

	if failure.Status != http.StatusNotFound {
		t.Errorf("status = %d, queria 404", failure.Status)
	}

	if failure.Code != "not_found" {
		t.Errorf("code = %q, queria not_found", failure.Code)
	}

	if failure.Message != "Etiqueta não encontrada." {
		t.Errorf("message = %q", failure.Message)
	}

	if failure.RequestID != "req-do-corpo" {
		t.Errorf("request_id = %q, queria o do corpo", failure.RequestID)
	}

	if failure.Retryable() {
		t.Error("404 não é repetível")
	}
}

func TestRequestIDCaiNoCabecalhoQuandoOCorpoNaoTraz(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusForbidden, map[string]any{
		"code":    "forbidden",
		"message": "sem permissão",
	}, nil)

	_, err := fake.client().Tags.List(context.Background())

	if got := apiError(t, err).RequestID; got != "req-do-cabecalho" {
		t.Errorf("request_id = %q, queria o do cabeçalho", got)
	}
}

func TestRetryAfterEhLidoEMarcaRepetivel(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusTooManyRequests,
		map[string]any{"code": "rate_limited", "message": "aguarde"},
		map[string]string{"Retry-After": "30"},
	)

	_, err := fake.client().Tags.List(context.Background())
	failure := apiError(t, err)

	if failure.RetryAfter != 30 {
		t.Errorf("RetryAfter = %v, queria 30", failure.RetryAfter)
	}

	if !failure.Retryable() {
		t.Error("429 é repetível")
	}
}

func TestRetryAfterQueNaoEhNumeroEhDescartado(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusServiceUnavailable,
		map[string]any{"code": "unavailable", "message": "volte depois"},
		map[string]string{"Retry-After": "Wed, 21 Oct 2026 07:28:00 GMT"},
	)

	_, err := fake.client().Freshness.Get(context.Background())
	failure := apiError(t, err)

	// Zero, e não um atraso chutado: atraso errado é pior que nenhum.
	if failure.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, queria 0", failure.RetryAfter)
	}

	if !failure.Retryable() {
		t.Error("503 é repetível")
	}
}

func TestValidacaoCarregaFieldEDetails(t *testing.T) {
	fake := newFakeAPI(t)
	fake.refuses(t, http.StatusBadRequest, map[string]any{
		"code":    "invalid_value",
		"message": "channel aceita: email, sms, whatsapp, voice, letter.",
		"field":   "channel",
		"details": []map[string]string{
			{"field": "channel", "code": "invalid_value", "message": "fora da lista"},
		},
	}, nil)

	_, err := fake.client().Templates.List(context.Background(), aronline.TemplateFilter{})
	failure := apiError(t, err)

	if failure.Field != "channel" {
		t.Errorf("field = %q, queria channel", failure.Field)
	}

	if len(failure.Details) != 1 || failure.Details[0].Code != "invalid_value" {
		t.Errorf("details = %+v", failure.Details)
	}
}

func TestRespostaQueNaoEhJSONViraAPIError(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answersRaw(http.StatusBadGateway, "<html>bad gateway</html>")

	_, err := fake.client().Tags.List(context.Background())
	failure := apiError(t, err)

	if failure.Status != http.StatusBadGateway {
		t.Errorf("status = %d, queria 502", failure.Status)
	}

	if failure.Code != "invalid_response" {
		t.Errorf("code = %q, queria invalid_response", failure.Code)
	}
}

func TestDuzentosQueNaoEhJSONTambemViraAPIError(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answersRaw(http.StatusOK, "nem json")

	_, err := fake.client().Freshness.Get(context.Background())

	if got := apiError(t, err).Code; got != "invalid_response" {
		t.Errorf("code = %q, queria invalid_response", got)
	}
}

func TestEnderecoForaDoArViraUnreachable(t *testing.T) {
	client := aronline.New(aronline.Options{
		Token:   "tok",
		BaseURL: "http://127.0.0.1:1",
		Timeout: 2 * time.Second,
	})

	_, err := client.Freshness.Get(context.Background())
	failure := apiError(t, err)

	if failure.Status != 0 {
		t.Errorf("status = %d, queria 0", failure.Status)
	}

	if failure.Code != "unreachable" {
		t.Errorf("code = %q, queria unreachable", failure.Code)
	}

	if failure.Unwrap() == nil {
		t.Error("a falha de transporte tem de continuar alcançável por errors.Is")
	}
}

func TestContextoCanceladoContinuaAlcancavelPorErrorsIs(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fake.client().Tags.List(ctx)

	// O SDK embrulha em APIError, mas não esconde a causa: quem trata
	// cancelamento continua usando errors.Is como usa no resto do programa.
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false; err = %v", err)
	}
}

func TestRotaAutenticadaSemTokenFalhaAntesDeSair(t *testing.T) {
	fake := newFakeAPI(t)

	_, err := fake.anonymous().Tags.List(context.Background())
	failure := apiError(t, err)

	if failure.Status != http.StatusUnauthorized {
		t.Errorf("status = %d, queria 401", failure.Status)
	}

	if failure.Code != "unauthenticated" {
		t.Errorf("code = %q, queria unauthenticated", failure.Code)
	}

	if fake.received.Path != "" {
		t.Errorf("a requisição saiu da máquina: %q", fake.received.Path)
	}
}

func TestErrorMostraOQueOSuportePede(t *testing.T) {
	failure := &aronline.APIError{
		Status: 404, Code: "not_found", Message: "sumiu", RequestID: "r-9",
	}

	want := "aronline: not_found (404): sumiu [request_id=r-9]"
	if got := failure.Error(); got != want {
		t.Errorf("Error() = %q, queria %q", got, want)
	}

	sem := &aronline.APIError{Status: 401, Code: "unauthenticated", Message: "sem token"}
	if got := sem.Error(); got != "aronline: unauthenticated (401): sem token" {
		t.Errorf("Error() sem request_id = %q", got)
	}
}
