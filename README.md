# AR Online — SDK Go

[![CI](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml/badge.svg)](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.23%2B-00add8.svg)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/AR-Online/ar-online-go.svg)](https://pkg.go.dev/github.com/AR-Online/ar-online-go)
[![Cobertura](https://img.shields.io/badge/cobertura-97.9%25-success.svg)](#-desenvolvimento)
[![Dependências](https://img.shields.io/badge/depend%C3%AAncias-0-success.svg)](#-o-que-ele-resolve)
[![Licença](https://img.shields.io/badge/licen%C3%A7a-Apache--2.0-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-n%C3%A3o%20publicado-orange.svg)](#-escopo)

Cliente oficial da API do AR Online para Go. Você não monta URL, não escreve cabeçalho, não desembrulha envelope e não lê status para saber se deu certo: chama método, recebe struct tipada, e a falha chega como `*APIError`.

## ✨ O que ele resolve

- **O envelope não é uniforme** — `templates`, `tags` e `allowlist` respondem `{"data": …}`; `freshness` e `version` respondem o objeto direto. Desembrulhar tudo, ou nada, quebra metade das chamadas. O SDK sabe por rota.
- **Um erro só, e `errors.Is` continua funcionando** — recusa do catálogo, proxy respondendo HTML e rede fora do ar chegam todos como `*APIError`, com a causa original ainda alcançável por `errors.Is(err, context.DeadlineExceeded)`.
- **`RequestID` de primeira classe** — é o primeiro dado que o suporte pede. Um SDK que o engolisse obrigaria você a reproduzir a falha no `curl` para achar o número.
- **Rota aberta funciona sem token** — `Version` é pública. Cliente construído sem credencial chama ela, o que serve para conferir a instalação antes de ter token.
- **`context.Context` em todo método** — cancelamento e prazo são seus, como no resto do seu programa.
- **Zero dependência** — só a biblioteca padrão. Nada entra no `go.sum` da sua aplicação por causa deste SDK.
- **`nil` e `0` não se confundem** — campo que a API responde `null` é ponteiro aqui. "Nenhuma fonte tem marca de leitura" não é "está tudo em dia".

## 🚀 Começando

### Instalação

```bash
go get github.com/AR-Online/ar-online-go
```

Go 1.23 ou mais novo. O caminho do módulo termina em `ar-online-go`, mas o **pacote** se chama `aronline`.

### Primeira chamada

```go
package main

import (
	"context"
	"fmt"
	"os"

	aronline "github.com/AR-Online/ar-online-go"
)

func main() {
	client := aronline.New(aronline.Options{Token: os.Getenv("AR_TOKEN")})

	templates, err := client.Templates.List(context.Background(), aronline.TemplateFilter{
		Channel: aronline.ChannelWhatsApp,
	})
	if err != nil {
		panic(err)
	}

	for _, template := range templates {
		fmt.Println(template.Name, len(template.Variables))
	}
}
```

O token é emitido pelo AR Online — a API só verifica, ela não emite. Se você ainda não tem o seu, fale com o suporte.

## 🧰 O que dá para fazer

| recurso | métodos | precisa de token |
|---|---|---|
| Modelos | `Templates.List(ctx, filtro)` · `Templates.Get(ctx, id)` | sim |
| Etiquetas | `Tags.List(ctx)` · `Tags.Get(ctx, id)` | sim |
| Lista de permitidos | `Allowlist.List(ctx)` | sim |
| Frescor dos dados | `Freshness.Get(ctx)` | sim |
| Versão | `Version.Get(ctx)` | **não** |

### Modelos

```go
todos, err := client.Templates.List(ctx, aronline.TemplateFilter{})
doWhatsApp, err := client.Templates.List(ctx, aronline.TemplateFilter{
	Channel: aronline.ChannelWhatsApp,
})
um, err := client.Templates.Get(ctx, "9b2f-uuid")
```

Os canais são constantes: `ChannelEmail`, `ChannelSMS`, `ChannelWhatsApp`, `ChannelVoice` e `ChannelLetter`. `aronline.Channels` traz a lista inteira.

### Etiquetas e lista de permitidos

```go
etiquetas, err := client.Tags.List(ctx)
uma, err := client.Tags.Get(ctx, "12")
permitidos, err := client.Allowlist.List(ctx)
```

Ambas são **pessoais**: respondem ao que pertence a quem está no token. Token de integração recebe `403` dizendo isso — e não uma lista vazia, que leria como "você não tem nenhuma".

### Frescor dos dados

```go
frescor, err := client.Freshness.Get(ctx)

if frescor.SourcesBehind > 0 {
	log.Printf("%d de %d atrasadas", frescor.SourcesBehind, frescor.SourcesTracked)
}
```

Responde a pergunta prática de quando uma consulta devolve menos do que você esperava: o defeito é da API, ou a carga está atrasada? Sem esse número as duas hipóteses parecem a mesma coisa.

Ela responde em **contagens**, não em lista de tabelas: "46 acompanhadas, 3 atrasadas" responde "está fresco?"; quarenta e seis nomes de tabela é relatório que ninguém lê na hora.

### Versão

```go
info, err := client.Version.Get(ctx)
fmt.Println(info.Version, info.Environment)
```

O único método que funciona **sem token**. É o primeiro dado que o suporte pede.

## ⚠️ Quando dá errado

Toda recusa vira `*aronline.APIError`, alcançável por `errors.As`:

```go
_, err := client.Templates.Get(ctx, "nao-existe")

var failure *aronline.APIError
if errors.As(err, &failure) {
	fmt.Println(failure.Code)      // "not_found"
	fmt.Println(failure.Status)    // 404
	fmt.Println(failure.RequestID) // o número que o suporte pede
}
```

| campo | o que é |
|---|---|
| `Status` | o status HTTP (`0` quando a API nem foi alcançada) |
| `Code` | o código do catálogo: `not_found`, `forbidden`, `rate_limited`, … |
| `Message` | a mensagem da API, em pt-BR |
| `RequestID` | identifica a chamada nos nossos registros — **sempre informe num chamado** |
| `Field` | o campo recusado, quando a recusa é sobre um |
| `Details` | uma entrada por campo, em erro de validação |
| `RetryAfter` | quantos segundos esperar, em `429` e `503`; `0` quando o cabeçalho não veio |
| `Retryable()` | `true` em `429` e `503` |

Repetir é decisão sua — o SDK não repete sozinho, porque só quem chamou sabe se a operação pode acontecer duas vezes.

A causa continua embaixo: `errors.Is(err, context.DeadlineExceeded)` funciona como funcionaria sem o SDK.

## ⚙️ Configuração

```go
aronline.New(aronline.Options{
	Token:      "…",                          // opcional: sem ele, só Version funciona
	BaseURL:    "https://v3.ar-online.com.br", // padrão; troque para homologação
	Timeout:    30 * time.Second,             // padrão
	HTTPClient: meuClient,                    // opcional: seu pool, seu proxy
})
```

`Options{}` zerado já é utilizável: aponta para produção sem credencial, que é o suficiente para `Version.Get`.

## 🎯 Escopo

Este SDK fala **só a `/v3`**. As rotas `/v1` e `/v2` continuam de pé, mas respondem byte a byte o que as APIs antigas respondiam, idiossincrasias incluídas — inclusive erro com status `200`. São espelhos para ninguém precisar migrar no mesmo dia, e um cliente tipado que as "melhorasse" quebraria exatamente quem elas protegem.

A superfície `/v3` é **só de leitura** hoje. Escrita entra nos cinco SDKs na mesma leva em que entrar na API.

## 🧪 Desenvolvimento

| comando | o que cobra |
|---|---|
| `gofmt -l .` | formato |
| `go vet ./...` | o vet da linguagem |
| `golangci-lint run` | `bodyclose`, `errorlint`, `gosec`, `noctx`, `revive` |
| `codespell` | ortografia |
| `go test ./... -race -coverprofile=coverage.out` | testes, com detector de corrida |
| `go tool cover -func=coverage.out` | cobertura — o mínimo é **95%** |
| `govulncheck ./...` | vulnerabilidade conhecida |

| métrica | valor |
|---|---|
| Testes | 37 (subtestes incluídos) |
| Cobertura | 97,9% |
| Dependências | 0 |

⚠️ Este módulo **não tem dependência nenhuma**, então o que o `govulncheck` cobra é a **biblioteca padrão** — que é exatamente onde uma CVE de HTTP ou de TLS apareceria. Por isso ele reprova quando a sua toolchain está atrás: não é o SDK, é o Go da sua máquina. O CI usa `stable`.

Os testes ficam ao lado do código, em `package aronline_test` — a separação caixa-preta que a linguagem oferece: enxergam só a API pública, como qualquer pessoa que instala o módulo. Um diretório `test/` à parte seria outro pacote, e `go test ./...` deixaria de medir cobertura deste aqui.

Eles sobem um `httptest.Server` **de verdade numa porta livre**. Não há dublê: o que este SDK precisa acertar é justamente o fio.

## 📚 Documentação

- [Documentação da API](https://docs.ar-online.com.br) — o contrato HTTP cru
- [Referência do pacote](https://pkg.go.dev/github.com/AR-Online/ar-online-go) — godoc
- `https://v3.ar-online.com.br/docs/openapi.json` — sempre a lista completa do que está no ar

## 📄 Licença

Apache License 2.0 — veja [LICENSE](LICENSE). © 2026 AR ONLINE TECNOLOGIA LTDA.
