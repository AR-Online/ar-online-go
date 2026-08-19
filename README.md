# AR Online SDK para Go

[![CI](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml/badge.svg)](https://github.com/AR-Online/ar-online-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/AR-Online/ar-online-go.svg)](https://pkg.go.dev/github.com/AR-Online/ar-online-go)
[![Go](https://img.shields.io/badge/go-1.23%2B-00add8.svg)](https://go.dev/)
[![Licença](https://img.shields.io/badge/licen%C3%A7a-Apache--2.0-green.svg)](LICENSE)

Cliente oficial da API da AR Online para Go.

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

A plataforma tem duas superfícies de API, e cada uma usa uma credencial
diferente. O SDK aceita as duas no mesmo cliente e envia cada uma no formato que
a sua superfície espera.

### Token do gateway (API legada)

É a credencial que você usa para enviar notificações e consultar status hoje.
Solicite em <suporte@ar-online.com.br>. No SDK, ela vai em `LegacyToken`.

### Token da API /v3

Solicite em <suporte@ar-online.com.br>. O token fica preso a uma entidade da sua
conta, e é ela que define quais dados ele enxerga — se você precisa consultar
mais de uma, peça um token para cada. O padrão é somente leitura.

O token tem prazo de validade. Token ausente, expirado ou revogado responde
`401`; se um token vazar, peça a revogação e ele deixa de ser aceito na chamada
seguinte.

> **A /v3 ainda não está publicada.** O endereço `v3.ar-online.com.br`, que é o
> padrão do SDK para essa superfície, entra no ar junto com ela — assim como a
> emissão de token por conta própria, na tela *Gerar Token* da documentação, com
> o mesmo usuário e senha do portal. Até lá, a parte da /v3 deste SDK serve para
> desenvolver contra um ambiente de teste, e é o `client.Legacy` que fala com a
> API em produção.

## Primeiros passos

O envio de notificações é feito hoje pela API legada, exposta no SDK em
`client.Legacy`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	aronline "github.com/AR-Online/ar-online-go"
)

func main() {
	ctx := context.Background()
	client := aronline.New(aronline.Options{LegacyToken: os.Getenv("AR_GW_TOKEN")})

	sent, err := client.Legacy.Send(ctx, aronline.EnvioRequest{
		NameTo:  "João da Silva",
		To:      "joao@exemplo.com",
		Subject: "Notificação de vencimento",
		Content: "<p>Prezado João, identificamos uma pendência em seu contrato.</p>",
		SMS:     &aronline.CanalSMS{Number: "11999998888"},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("notificação aceita:", sent.IDEmail)
}
```

Guarde o `IDEmail`: é com ele que você consulta o status de qualquer canal e
baixa os comprovantes.

```go
status, err := client.Legacy.Status.Email(ctx, sent.IDEmail)

fmt.Println(status.Description) // "Processado", "Enviado", "Entregue", "Lido"
```

Todo método recebe um `context.Context`, então cancelamento e prazo continuam
sob o seu controle.

## Referência

### Envio e acompanhamento (`client.Legacy`)

| método | o que faz |
|---|---|
| `Legacy.Send(ctx, envio)` | envia a notificação em um ou mais canais |
| `Legacy.Status.Email(ctx, id)` | status do AR-Email |
| `Legacy.Status.SMS(ctx, id)` | status do AR-SMS |
| `Legacy.Status.WhatsApp(ctx, id)` | status do AR-WhatsApp |
| `Legacy.Status.Voz(ctx, id)` | status do AR-Voz |
| `Legacy.Status.Carta(ctx, id)` | status do AR-Cartas, com o rastreio dos Correios |
| `Legacy.Status.Full(ctx, id)` | dados de perícia de todos os canais numa chamada |
| `Legacy.SendingProof(ctx, id)` | comprovante de envio em PDF |
| `Legacy.Laudo(ctx, id)` | laudo pericial em PDF |
| `Legacy.FinalizarRegua(ctx, id)` | encerra a régua de notificação do envio |
| `Legacy.Templates.List(ctx, filtro)` | lista os modelos da sua entidade |
| `Legacy.Templates.Get(ctx, id)` | busca um modelo |
| `Legacy.Templates.Update(ctx, id, campos)` | edita nome e compartilhamento |
| `Legacy.Templates.Deactivate(ctx, id)` | desativa um modelo |
| `Legacy.Templates.SetStatus(ctx, id, ativo)` | ativa ou desativa um modelo |

O `id` das rotas de status é sempre o `IDEmail` da notificação — o mesmo para
todos os canais. Não existe identificador por canal.

Envio multicanal: cada canal é um bloco opcional no corpo.

```go
_, err := client.Legacy.Send(ctx, aronline.EnvioRequest{
	NameTo:      "João da Silva",
	To:          "joao@exemplo.com",
	Subject:     "Notificação de vencimento",
	Content:     "<p>Conteúdo em HTML.</p>",
	CustomID:    "contrato-4471", // sua referência, devolvida na consulta de status
	Attachments: []aronline.Anexo{{Name: "contrato.pdf", Base64: "…"}},
	SMS: &aronline.CanalSMS{
		Number:        "11999998888",
		TypeSend:      aronline.SMSTypeSendFallback, // "1" só se o e-mail não for entregue
		CustomMessage: "Você recebeu um AR-Email. Acesse: {SHORT_LINK}",
	},
	WhatsApp: &aronline.CanalWhatsApp{
		Number:    "11999998888",
		Variables: map[string]any{"template": "aviso_01"},
	},
	Voz:   &aronline.CanalVoz{Number: "1133334444", Template: "aviso_voz"},
	Carta: &aronline.CanalCarta{Name: "João da Silva", Modelo: "padrao"},
})
```

Comprovantes: o comprovante de envio chega em base64 dentro de um JSON e o SDK
já o decodifica; o laudo pericial chega como PDF binário.

```go
comprovante, err := client.Legacy.SendingProof(ctx, id)

if comprovante.PDF != nil {
	err = os.WriteFile("comprovante.pdf", comprovante.PDF, 0o600)
} else {
	fmt.Println(comprovante.Message) // ainda sem status de entrega
}

laudo, err := client.Legacy.Laudo(ctx, id)
err = os.WriteFile("laudo.pdf", laudo, 0o600)
```

Duas coisas do contrato antigo aparecem nas structs, e são de propósito:

- **A data do legado é `string`** (`"18/07/2026 01:01:32"`). O formato não traz
  fuso, então nenhum `time.Time` sai dele sem chute.
- **Ausência tem quatro convenções**, às vezes duas na mesma resposta: `""`
  (`DateSend` do e-mail), `null` (`DateReading`), a chave que some (as datas de
  WhatsApp, voz e carta) e `{}`. Onde o contrato responde `null`, o campo é
  ponteiro sem `omitempty`; onde a chave some, é ponteiro **com** `omitempty` —
  assim uma struct que volta para JSON sai com a mesma convenção com que entrou.

Modelos do gateway: a família `/gw/templates` responde `200` mesmo em erro, com
o código de verdade dentro do corpo. O SDK lê o de dentro e devolve
`*LegacyAPIError`, então você não precisa saber que existe envelope.

```go
modelos, err := client.Legacy.Templates.List(ctx, aronline.GwTemplateFilter{
	Type: aronline.GwTemplateTypeWhatsApp,
})
```

Os códigos são os do legado: `GwTemplateTypeWhatsApp` (`"1"`),
`GwTemplateTypeEmail` (`"2"`), `GwTemplateTypeSMS` (`"3"`) e
`GwTemplateTypeCarta` (`"4"`). `aronline.GwTemplateTypes` traz a lista inteira.
Código fora dela responde **lista vazia, não erro**.

### Consultas da API /v3 (`client.*`)

A /v3 é a API nova, com contrato limpo e validação estrita. Hoje ela é somente
de leitura.

| método | o que faz | precisa de token |
|---|---|---|
| `Templates.List(ctx, filtro)` | lista os modelos, com filtro por canal | sim |
| `Templates.Get(ctx, id)` | busca um modelo pelo UUID | sim |
| `Tags.List(ctx)` · `Tags.Get(ctx, id)` | suas etiquetas | sim |
| `Allowlist.List(ctx)` | seus destinatários permitidos | sim |
| `Freshness.Get(ctx)` | o atraso da carga de dados | sim |
| `Version.Get(ctx)` | qual versão da API está no ar | não |

```go
todos, err := client.Templates.List(ctx, aronline.TemplateFilter{})
doWhatsApp, err := client.Templates.List(ctx, aronline.TemplateFilter{
	Channel: aronline.ChannelWhatsApp,
})
um, err := client.Templates.Get(ctx, "9b2f-uuid")
```

Os canais são constantes: `ChannelEmail`, `ChannelSMS`, `ChannelWhatsApp`,
`ChannelVoice` e `ChannelLetter`. `aronline.Channels` traz a lista inteira.

```go
etiquetas, err := client.Tags.List(ctx)
uma, err := client.Tags.Get(ctx, "12")
permitidos, err := client.Allowlist.List(ctx)
```

Etiquetas e lista de permitidos são recursos **pessoais**: respondem o que
pertence a quem está no token. Um token de integração, que não representa uma
pessoa, recebe `403` nessas rotas.

```go
frescor, err := client.Freshness.Get(ctx)

if frescor.SourcesBehind > 0 {
	log.Printf("%d de %d atrasadas", frescor.SourcesBehind, frescor.SourcesTracked)
}
```

O frescor serve para responder uma pergunta prática: quando uma consulta
devolve menos do que você esperava, o problema é a API ou a carga de dados está
atrasada?

```go
info, err := client.Version.Get(ctx)
fmt.Println(info.Version, info.Environment)
```

`Version.Get` é a única chamada que funciona sem token, útil para conferir a
instalação antes de ter uma credencial.

## Tratamento de erros

Chamada que devolveu `err == nil` deu certo. Você não precisa ler status HTTP
nem procurar campo de erro no corpo da resposta.

A /v3 devolve `*aronline.APIError`, alcançável por `errors.As`:

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

A API legada devolve `*aronline.LegacyAPIError`, com os campos do contrato
antigo:

```go
_, err := client.Legacy.Templates.Get(ctx, "nao-existe")

var recusa *aronline.LegacyAPIError
if errors.As(err, &recusa) {
	fmt.Println(recusa.Status)     // 404 — o código que vale
	fmt.Println(recusa.HTTPStatus) // 200 — o que o fio disse
}
```

| campo | conteúdo |
|---|---|
| `Status` | o código que vale, mesmo quando o HTTP respondeu `200` |
| `HTTPStatus` | o status que veio no protocolo (`0` se a chamada nem saiu) |
| `Message` | a mensagem do gateway, em português |
| `Body` | o corpo da resposta em bytes, exatamente como chegou |

Duas respostas do legado **não** são erro, e por isso chegam como valor:
`Status.Voz` de um uuid sem registro (o gateway responde `200` com uma frase) e
o comprovante que ainda não tem status de entrega (`PDF` vem `nil`, `Message`
preenchida).

Erro de rede e resposta que não é JSON também chegam como erro do SDK nas duas
superfícies, e a causa original continua embaixo:
`errors.Is(err, context.DeadlineExceeded)` funciona como funcionaria sem o SDK.

O SDK não repete chamadas automaticamente, porque só quem chamou sabe se a
operação pode acontecer duas vezes.

## Configuração do cliente

```go
aronline.New(aronline.Options{
	Token:         os.Getenv("AR_TOKEN"),          // credencial da /v3
	LegacyToken:   os.Getenv("AR_GW_TOKEN"),       // credencial do gateway
	BaseURL:       "https://v3.ar-online.com.br",  // padrão
	LegacyBaseURL: "https://api.ar-online.com.br", // padrão
	Timeout:       30 * time.Second,               // padrão, vale para as duas
	HTTPClient:    meuClient,                      // opcional: seu pool, seu proxy
})
```

Cada credencial é opcional: informe só a da superfície que você vai usar. Nenhum
dos dois tokens viaja para o endereço do outro, e a área de legado sem
`LegacyToken` falha **antes do socket**, dizendo qual opção falta.

`Options{}` zerado já é utilizável: aponta para produção sem credencial, o
suficiente para `Version.Get`.

Os campos das structs usam os nomes que a API escreve (`customID`,
`provider_identifier`, `created_at`), para que o que você lê no SDK seja o mesmo
que você vê na documentação e nos nossos registros de suporte. Campo que a API
responde `null` é ponteiro, para que ausência e zero não se confundam: "nenhuma
fonte tem marca de leitura" não é "está tudo em dia".

## Webhooks

Em vez de consultar o status repetidamente, você pode receber uma chamada `POST`
a cada mudança. A configuração é feita com o suporte, que cadastra o seu endpoint
e os parâmetros de autenticação. O SDK não recebe a requisição por você, mas
exporta as structs do payload — `aronline.WebhookPayloadV1` e
`aronline.WebhookPayloadV2`.

No v2, `Payload` é `json.RawMessage`: leia `Channel` primeiro e decodifique na
struct daquele canal.

Veja <https://docs.ar-online.com.br/webhooks/visao-geral> para o fluxo completo,
incluindo a política de retentativas.

## As duas superfícies, e o caminho entre elas

A **API legada** é a que está em produção hoje e concentra envio, status e
comprovantes. A **/v3** é a API nova, para onde as funcionalidades estão sendo
migradas aos poucos.

Quando uma rota ganha equivalente na /v3, a função correspondente de
`client.Legacy` passa a falar com a /v3 internamente, **sem mudar de
assinatura**. Na prática, você migra atualizando o módulo, não reescrevendo a
sua integração. Cada troca dessas é registrada no [CHANGELOG](CHANGELOG.md).

Hoje: a leitura de modelos tem equivalente (`client.Templates`); envio, status e
comprovantes ainda não têm.

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
| Testes | 101, subtestes incluídos |
| Cobertura | 99,1% |
| Dependências | 0 |

Como o módulo não tem dependência nenhuma, o que o `govulncheck` cobra é a
biblioteca padrão — que é justamente onde apareceria uma CVE de HTTP ou de TLS.
Por isso ele reprova quando a sua toolchain está atrasada: o aviso é sobre o Go
da sua máquina, não sobre o SDK. O CI usa `stable`.

Os testes ficam em `package aronline_test`, a separação caixa-preta que a
linguagem oferece: enxergam apenas a API pública, como qualquer pessoa que
instala o módulo. Eles sobem um `httptest.Server` real em uma porta livre — um
para a /v3 e outro para o gateway, porque o que um SDK precisa acertar é
justamente o fio.

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
