# AR Online SDK para Go

[![CI](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml/badge.svg)](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/AR-Online/ar-online-go.svg)](https://pkg.go.dev/github.com/AR-Online/ar-online-go)
[![Go](https://img.shields.io/badge/go-1.23%2B-00add8.svg)](https://go.dev/)
[![Licença](https://img.shields.io/badge/licen%C3%A7a-Apache--2.0-green.svg)](LICENSE)

Cliente oficial da API da AR Online para Go.

> **Status:** este SDK cobre as consultas da API /v3, que ainda não está
> publicada — o endereço `v3.ar-online.com.br` entra no ar junto com ela. O
> envio de notificações em produção é feito hoje pela API legada, que ainda não
> está neste SDK. Fale com o suporte antes de planejar uma integração em cima
> dele.

## Sobre a AR Online

A AR Online é uma plataforma brasileira de notificação eletrônica com validade
jurídica. Uma única requisição dispara a notificação em até cinco canais, e cada
etapa do percurso — envio, entrega e leitura — é registrada com carimbo do tempo
emitido por uma Autoridade de Carimbo do Tempo da ICP-Brasil. Esse registro é o
que dá à comunicação o valor de prova documental previsto na MP 2.200-2/2001, e
é o que diferencia a plataforma de um serviço comum de disparo de mensagens.

Os canais disponíveis são:

| canal | o que é |
|---|---|
| AR-Email | e-mail com comprovação de entrega e de leitura |
| AR-SMS | mensagem de texto para o celular do destinatário |
| AR-WhatsApp | notificação por WhatsApp |
| AR-Voz | chamada telefônica automatizada |
| AR-Cartas | carta física registrada, enviada pelos Correios |

Você escolhe quais canais usar em cada envio. O processamento é assíncrono: a
API confirma o recebimento na hora e devolve um identificador, que você usa
depois para consultar o status de cada canal e baixar os comprovantes.

| | |
|---|---|
| Site | <https://www.ar-online.com.br> |
| Documentação da API | <https://docs.ar-online.com.br> |
| Suporte | <suporte@ar-online.com.br> · +55 (11) 4200-7766 |

## Requisitos

- Go 1.23 ou mais novo
- Nenhuma dependência: o SDK usa apenas a biblioteca padrão

## Instalação

```bash
go get github.com/AR-Online/ar-online-go
```

O caminho do módulo termina em `ar-online-go`, mas o pacote se chama `aronline`.

## Autenticação

### Token da API /v3

Solicite em <suporte@ar-online.com.br>. O token fica preso a uma entidade da sua
conta, e é ela que define quais dados ele enxerga — se você precisa consultar
mais de uma, peça um token para cada. O padrão é somente leitura.

O token tem prazo de validade. Token ausente, expirado ou revogado responde
`401`; se um token vazar, peça a revogação e ele deixa de ser aceito na chamada
seguinte.

Quando a /v3 for publicada, a emissão passa a ser por conta própria, na tela
*Gerar Token* da documentação, com o mesmo usuário e senha do portal.

## Primeiros passos

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

## Referência

Este SDK cobre hoje as consultas da API /v3. Todo método recebe um
`context.Context`, então cancelamento e prazo continuam sob o seu controle.

| método | o que faz | precisa de token |
|---|---|---|
| `Templates.List(ctx, filtro)` | lista os modelos, com filtro por canal | sim |
| `Templates.Get(ctx, id)` | busca um modelo pelo UUID | sim |
| `Tags.List(ctx)` · `Tags.Get(ctx, id)` | suas etiquetas | sim |
| `Allowlist.List(ctx)` | seus destinatários permitidos | sim |
| `Freshness.Get(ctx)` | o atraso da carga de dados | sim |
| `Version.Get(ctx)` | qual versão da API está no ar | não |

### Modelos

```go
todos, err := client.Templates.List(ctx, aronline.TemplateFilter{})
doWhatsApp, err := client.Templates.List(ctx, aronline.TemplateFilter{
	Channel: aronline.ChannelWhatsApp,
})
um, err := client.Templates.Get(ctx, "9b2f-uuid")
```

Os canais são constantes: `ChannelEmail`, `ChannelSMS`, `ChannelWhatsApp`,
`ChannelVoice` e `ChannelLetter`. `aronline.Channels` traz a lista inteira.

### Etiquetas e lista de permitidos

```go
etiquetas, err := client.Tags.List(ctx)
uma, err := client.Tags.Get(ctx, "12")
permitidos, err := client.Allowlist.List(ctx)
```

São recursos **pessoais**: respondem o que pertence a quem está no token. Um
token de integração, que não representa uma pessoa, recebe `403` nessas rotas.

### Atraso da carga

```go
frescor, err := client.Freshness.Get(ctx)

if frescor.SourcesBehind > 0 {
	log.Printf("%d de %d atrasadas", frescor.SourcesBehind, frescor.SourcesTracked)
}
```

Serve para responder uma pergunta prática: quando uma consulta devolve menos do
que você esperava, o problema é a API ou a carga de dados está atrasada?

### Versão

```go
info, err := client.Version.Get(ctx)
fmt.Println(info.Version, info.Environment)
```

É a única chamada que funciona sem token, útil para conferir a instalação antes
de ter uma credencial.

## Envio de notificações

O envio, a consulta de status por canal e os comprovantes estão na API legada do
gateway, que **ainda não está neste SDK** — hoje ela está disponível no
[SDK TypeScript](https://github.com/AR-Online/ar-online-typescript) e chega aqui
nas próximas versões.

Enquanto isso, o contrato HTTP está documentado em
<https://docs.ar-online.com.br>, e a credencial do gateway é emitida pelo
suporte.

## Tratamento de erros

Toda recusa vira `*aronline.APIError`, alcançável por `errors.As`:

```go
_, err := client.Templates.Get(ctx, "nao-existe")

var failure *aronline.APIError
if errors.As(err, &failure) {
	fmt.Println(failure.Code)      // "not_found"
	fmt.Println(failure.Status)    // 404
	fmt.Println(failure.RequestID) // informe este número ao abrir um chamado
}
```

| campo | conteúdo |
|---|---|
| `Status` | o status HTTP (`0` quando a API não foi alcançada) |
| `Code` | o código do catálogo: `not_found`, `forbidden`, `rate_limited`, … |
| `Message` | a mensagem da API, em português |
| `RequestID` | identifica a chamada nos nossos registros |
| `Field` | o campo recusado, quando a recusa é sobre um campo |
| `Details` | uma entrada por campo, em erro de validação |
| `RetryAfter` | quantos segundos esperar, em `429` e `503` |
| `Retryable()` | `true` em `429` e `503` |

Erro de rede e resposta que não é JSON também chegam como `*APIError`, e a causa
original continua embaixo: `errors.Is(err, context.DeadlineExceeded)` funciona
como funcionaria sem o SDK.

O SDK não repete chamadas automaticamente, porque só quem chamou sabe se a
operação pode acontecer duas vezes.

## Configuração do cliente

```go
aronline.New(aronline.Options{
	Token:      "…",                           // opcional: sem ele, só Version funciona
	BaseURL:    "https://v3.ar-online.com.br", // padrão
	Timeout:    30 * time.Second,              // padrão
	HTTPClient: meuClient,                     // opcional: seu pool, seu proxy
})
```

`Options{}` zerado já é utilizável: aponta para produção sem credencial, o
suficiente para `Version.Get`.

Campo que a API responde `null` é ponteiro nas structs, para que ausência e zero
não se confundam: "nenhuma fonte tem marca de leitura" não é "está tudo em dia".

## Desenvolvimento

| comando | o que cobra |
|---|---|
| `gofmt -l .` | formato |
| `go vet ./...` | o vet da linguagem |
| `golangci-lint run` | `bodyclose`, `errorlint`, `gosec`, `noctx`, `revive` |
| `codespell` | ortografia |
| `go test ./... -race -coverprofile=coverage.out` | testes, com detector de corrida |
| `go tool cover -func=coverage.out` | cobertura, com mínimo de 95% |
| `govulncheck ./...` | vulnerabilidade conhecida |

| métrica | valor |
|---|---|
| Testes | 37, subtestes incluídos |
| Cobertura | 97,9% |
| Dependências | 0 |

Como o módulo não tem dependência nenhuma, o que o `govulncheck` cobra é a
biblioteca padrão — que é justamente onde apareceria uma CVE de HTTP ou de TLS.
Por isso ele reprova quando a sua toolchain está atrasada: o aviso é sobre o Go
da sua máquina, não sobre o SDK. O CI usa `stable`.

Os testes ficam em `package aronline_test`, a separação caixa-preta que a
linguagem oferece: enxergam apenas a API pública, como qualquer pessoa que
instala o módulo. Eles sobem um `httptest.Server` real em uma porta livre.

Para publicar uma versão, veja [PUBLICANDO.md](PUBLICANDO.md).

## Suporte

- Dúvidas de integração e emissão de credenciais: <suporte@ar-online.com.br>
- Telefone: +55 (11) 4200-7766
- Defeitos neste SDK: [issues do repositório](https://github.com/AR-Online/ar-online-go/issues)

Ao abrir um chamado sobre uma chamada que falhou, informe o `RequestID` do erro:
é com ele que localizamos a requisição nos nossos registros.

## Licença

Apache License 2.0 — veja [LICENSE](LICENSE).

© 2026 AR ONLINE TECNOLOGIA LTDA.
