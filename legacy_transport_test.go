package aronline_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	aronline "github.com/AR-Online/ar-online-go"
)

func TestLegadoTemEnderecoProprioIndependenteDaV3(t *testing.T) {
	if aronline.DefaultLegacyBaseURL != "https://api.ar-online.com.br" {
		t.Errorf("DefaultLegacyBaseURL = %q", aronline.DefaultLegacyBaseURL)
	}

	if aronline.DefaultLegacyBaseURL == aronline.DefaultBaseURL {
		t.Error("o endereço do legado não pode ser o mesmo da /v3")
	}
}

func TestLegadoMandaOTokenCruSemBearer(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"description": "ok"})

	if _, err := fake.client().Legacy.Status.Voz(context.Background(), "x"); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	// Cru: o gateway quer o JWT SEM "Bearer" — o oposto da /v3. Prefixar aqui
	// transformaria toda chamada no 401 dele.
	if got := fake.received.Authorization; got != "tok-gw" {
		t.Errorf("Authorization = %q, queria o token cru", got)
	}

	if got := fake.received.Accept; got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
}

func TestAsDuasCredenciaisConvivemSemVazarUmaNaAreaDaOutra(t *testing.T) {
	fake := newFakeGateway(t)

	ambos := aronline.New(aronline.Options{
		Token:         "tok-v3",
		BaseURL:       fake.server.URL,
		LegacyToken:   "tok-gw",
		LegacyBaseURL: fake.server.URL,
	})

	fake.answers(t, map[string]any{"description": "ok"})

	if _, err := ambos.Legacy.Status.Voz(context.Background(), "x"); err != nil {
		t.Fatalf("não esperava erro no legado: %v", err)
	}

	if got := fake.received.Authorization; got != "tok-gw" {
		t.Errorf("na área de legado o Authorization = %q, queria tok-gw", got)
	}

	fake.answers(t, map[string]any{"data": []any{}})

	if _, err := ambos.Tags.List(context.Background()); err != nil {
		t.Fatalf("não esperava erro na /v3: %v", err)
	}

	if got := fake.received.Authorization; got != "Bearer tok-v3" {
		t.Errorf("na /v3 o Authorization = %q, queria Bearer tok-v3", got)
	}
}

func TestLegadoSemTokenFalhaAntesDoSocketDizendoQualFalta(t *testing.T) {
	fake := newFakeGateway(t)

	_, err := fake.anonymous().Legacy.Send(context.Background(), aronline.EnvioRequest{
		NameTo: "A", Subject: "B", Content: "C",
	})
	failure := legacyError(t, err)

	if failure.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, queria 401", failure.Status)
	}

	// httpStatus 0: a chamada não chegou a sair da máquina.
	if failure.HTTPStatus != 0 {
		t.Errorf("HTTPStatus = %d, queria 0", failure.HTTPStatus)
	}

	if !stringsContains(failure.Message, "LegacyToken") {
		t.Errorf("mensagem = %q, queria nomear a opção que falta", failure.Message)
	}

	if fake.received.Path != "" {
		t.Errorf("a requisição saiu da máquina: %q", fake.received.Path)
	}
}

func TestLegadoDuzentosQueNaoEhJSONNaoVazaErroDeParser(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answersRaw(http.StatusOK, "text/html", "<html>proxy</html>")

	_, err := fake.client().Legacy.Status.Email(context.Background(), "x")
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "não é JSON") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if string(failure.Body) != "<html>proxy</html>" {
		t.Errorf("Body = %q, queria o corpo cru", failure.Body)
	}

	// A causa do decodificador continua embaixo, para quem for depurar.
	if failure.Unwrap() == nil {
		t.Error("a falha de decodificação tem de continuar alcançável")
	}
}

func TestLegado502DeHTMLFalhaComOStatusSemErroDeParser(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answersRaw(http.StatusBadGateway, "text/html", "<html>bad gateway</html>")

	_, err := fake.client().Legacy.Status.Email(context.Background(), "x")
	failure := legacyError(t, err)

	if failure.Status != http.StatusBadGateway || failure.HTTPStatus != http.StatusBadGateway {
		t.Errorf("status = %d / http = %d, queria 502 nos dois", failure.Status, failure.HTTPStatus)
	}

	if !stringsContains(failure.Message, "502") {
		t.Errorf("mensagem = %q, queria citar o 502", failure.Message)
	}
}

func TestLegadoEnderecoForaDoArViraOMesmoErroComStatusZero(t *testing.T) {
	client := aronline.New(aronline.Options{
		LegacyToken:   "tok-gw",
		LegacyBaseURL: "http://127.0.0.1:1",
		Timeout:       2 * time.Second,
	})

	_, err := client.Legacy.Status.Email(context.Background(), "x")
	failure := legacyError(t, err)

	if failure.Status != 0 || failure.HTTPStatus != 0 {
		t.Errorf("status = %d / http = %d, queria 0 nos dois", failure.Status, failure.HTTPStatus)
	}

	if failure.Unwrap() == nil {
		t.Error("a falha de transporte tem de continuar alcançável por errors.Is")
	}
}

func TestLegadoEnderecoImpossivelNaoViraPanico(t *testing.T) {
	// Um caractere de controle no endereço: o http.NewRequest recusa antes de
	// abrir socket. Tem de virar erro do SDK, não pânico dentro da aplicação de
	// quem chamou.
	client := aronline.New(aronline.Options{
		LegacyToken:   "tok-gw",
		LegacyBaseURL: "http://exemplo\x7f.test",
	})

	_, err := client.Legacy.Status.Email(context.Background(), "x")
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "montar a requisição") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if failure.Unwrap() == nil {
		t.Error("a causa tem de continuar alcançável")
	}
}

func TestLegadoCorpoQueNaoDaParaSerializarFalhaAntesDoSocket(t *testing.T) {
	fake := newFakeGateway(t)

	// Um canal dentro do payload livre da voz: o encoding/json recusa, e isso
	// tem de virar erro do SDK e não pânico.
	_, err := fake.client().Legacy.Send(context.Background(), aronline.EnvioRequest{
		NameTo:  "A",
		Subject: "B",
		Content: "C",
		Voz:     &aronline.CanalVoz{Payload: map[string]any{"impossivel": make(chan int)}},
	})
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "montar o corpo") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if fake.received.Path != "" {
		t.Errorf("a requisição saiu da máquina: %q", fake.received.Path)
	}
}

func TestLegadoRespostaInterrompidaNoMeioViraErro(t *testing.T) {
	// O cabeçalho promete 200 bytes e a conexão morre no meio da frase. Tem de
	// virar erro do SDK, com a causa embaixo — não um corpo pela metade
	// decodificado como se estivesse inteiro.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("o servidor de teste tinha de deixar sequestrar a conexão")

			return
		}

		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("não consegui sequestrar a conexão: %v", err)

			return
		}

		_, _ = conn.Write([]byte(
			"HTTP/1.1 200 OK\r\nContent-Length: 200\r\n\r\n{\"description\":",
		))
		_ = conn.Close()
	}))
	defer server.Close()

	client := aronline.New(aronline.Options{LegacyToken: "tok-gw", LegacyBaseURL: server.URL})

	_, err := client.Legacy.Status.Voz(context.Background(), "x")
	failure := legacyError(t, err)

	if !stringsContains(failure.Message, "interrompida no meio") {
		t.Errorf("mensagem = %q", failure.Message)
	}

	if failure.Unwrap() == nil {
		t.Error("a causa tem de continuar alcançável")
	}
}

func TestLegadoContextoCanceladoContinuaAlcancavelPorErrorsIs(t *testing.T) {
	fake := newFakeGateway(t)
	fake.answers(t, map[string]any{"description": "ok"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fake.client().Legacy.Status.Voz(ctx, "x")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false; err = %v", err)
	}
}

func TestLegadoErrorMostraOsDoisStatusQuandoDiferem(t *testing.T) {
	// O caso que vale registrar: um 200 carregando um 404 lá dentro.
	escondido := &aronline.LegacyAPIError{
		Status: 404, HTTPStatus: 200, Message: "Template não encontrado",
	}

	want := "aronline: legado (404): Template não encontrado [http=200]"
	if got := escondido.Error(); got != want {
		t.Errorf("Error() = %q, queria %q", got, want)
	}

	direto := &aronline.LegacyAPIError{Status: 404, HTTPStatus: 404, Message: "sumiu"}
	if got := direto.Error(); got != "aronline: legado (404): sumiu" {
		t.Errorf("Error() = %q", got)
	}
}
