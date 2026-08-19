package aronline_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	aronline "github.com/AR-Online/ar-online-go"
)

// Os caminhos que só o mundo real percorre. Eles são a razão de o SDK existir
// — quem chama não deveria descobrir na produção dele que a API respondeu algo
// que o cliente não sabia ler.

func TestCadaRecursoPropagaARecusa(t *testing.T) {
	// Uma tabela e não seis testes: o que se prova é que NENHUM recurso
	// engole a falha pelo caminho, e uma lista deixa isso óbvio de manter
	// quando entrar o sétimo.
	cases := []struct {
		name string
		call func(*aronline.Client) error
	}{
		{"templates.List", func(c *aronline.Client) error {
			_, err := c.Templates.List(context.Background(), aronline.TemplateFilter{})
			return err
		}},
		{"templates.Get", func(c *aronline.Client) error {
			_, err := c.Templates.Get(context.Background(), "x")
			return err
		}},
		{"tags.List", func(c *aronline.Client) error {
			_, err := c.Tags.List(context.Background())
			return err
		}},
		{"tags.Get", func(c *aronline.Client) error {
			_, err := c.Tags.Get(context.Background(), "x")
			return err
		}},
		{"allowlist.List", func(c *aronline.Client) error {
			_, err := c.Allowlist.List(context.Background())
			return err
		}},
		{"freshness.Get", func(c *aronline.Client) error {
			_, err := c.Freshness.Get(context.Background())
			return err
		}},
		{"version.Get", func(c *aronline.Client) error {
			_, err := c.Version.Get(context.Background())
			return err
		}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fake := newFakeAPI(t)
			fake.refuses(t, http.StatusServiceUnavailable, map[string]any{
				"code": "unavailable", "message": "indisponível",
			}, nil)

			failure := apiError(t, test.call(fake.client()))

			if failure.Status != http.StatusServiceUnavailable {
				t.Errorf("status = %d, queria 503", failure.Status)
			}

			if !failure.Retryable() {
				t.Error("503 é repetível")
			}
		})
	}
}

func TestEnvelopeComDataDeOutroTipo(t *testing.T) {
	fake := newFakeAPI(t)
	// A rota promete uma lista dentro de `data` e mandou um texto.
	fake.answers(t, map[string]any{"data": "não é lista"})

	_, err := fake.client().Tags.List(context.Background())
	failure := apiError(t, err)

	if failure.Code != "invalid_response" {
		t.Errorf("code = %q, queria invalid_response", failure.Code)
	}

	// A causa do decodificador continua embaixo, para quem for depurar.
	if failure.Unwrap() == nil {
		t.Error("a falha de decodificação tem de continuar alcançável")
	}
}

func TestEnderecoImpossivelNaoViraPanico(t *testing.T) {
	// Um caractere de controle no endereço: o http.NewRequest recusa antes de
	// abrir socket. Tem de virar erro do SDK, não pânico dentro da aplicação
	// de quem chamou.
	client := aronline.New(aronline.Options{Token: "tok", BaseURL: "http://exemplo\x7f.test"})

	_, err := client.Freshness.Get(context.Background())
	failure := apiError(t, err)

	if failure.Code != "invalid_request" {
		t.Errorf("code = %q, queria invalid_request", failure.Code)
	}

	if failure.Unwrap() == nil {
		t.Error("a causa tem de continuar alcançável")
	}
}

func TestOptionsAceitaClienteHttpProprio(t *testing.T) {
	fake := newFakeAPI(t)
	fake.answers(t, map[string]any{"data": []any{}})

	// Quem já tem pool, proxy ou instrumentação passa o próprio cliente — e o
	// Timeout das Options é ignorado, porque esse cliente carrega o dele.
	mine := &http.Client{Timeout: 5 * time.Second}
	client := aronline.New(aronline.Options{
		Token:      "tok",
		BaseURL:    fake.server.URL,
		Timeout:    time.Millisecond, // ignorado: o cliente de fora manda
		HTTPClient: mine,
	})

	if _, err := client.Tags.List(context.Background()); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
}

func TestOptionsZeradoApontaParaProducao(t *testing.T) {
	// Options{} vazio tem de ser utilizável — sem credencial, apontando para
	// produção. É o que faz `Version.Get` funcionar sem configurar nada.
	client := aronline.New(aronline.Options{Timeout: time.Millisecond})

	_, err := client.Version.Get(context.Background())
	failure := apiError(t, err)

	// Um milissegundo não fala com produção; o que se prova é que ele tentou,
	// e tentou no endereço certo.
	if failure.Code != "unreachable" {
		t.Errorf("code = %q, queria unreachable", failure.Code)
	}

	if !contains(failure.Message, aronline.DefaultBaseURL) {
		t.Errorf("mensagem = %q, queria citar %q", failure.Message, aronline.DefaultBaseURL)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && stringsContains(haystack, needle)
}

func stringsContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
