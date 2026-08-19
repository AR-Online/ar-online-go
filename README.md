# AR Online — SDK Go

Cliente oficial da API do AR Online para Go.

Você não monta URL, não escreve cabeçalho, não desembrulha envelope e não lê
status para saber se deu certo. Chama método, recebe struct tipada, e a falha
chega como `*aronline.APIError`.

## Instalação

```bash
go get github.com/AR-Online/ar-online-go
```

Go 1.23 ou mais novo. **Zero dependência** — só a biblioteca padrão.

O caminho do módulo termina em `ar-online-go`, mas o **pacote** se chama
`aronline`:

```go
import aronline "github.com/AR-Online/ar-online-go"
```

## Começando

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

Todo método recebe `context.Context` — cancelamento e prazo são seus, como no
resto do seu programa.

O token é emitido pelo AR Online. Se você ainda não tem o seu, fale com o
suporte — a API só verifica token, ela não emite.

## O que dá para fazer

### Modelos

```go
todos, err := client.Templates.List(ctx, aronline.TemplateFilter{})
doWhatsApp, err := client.Templates.List(ctx, aronline.TemplateFilter{
	Channel: aronline.ChannelWhatsApp,
})
um, err := client.Templates.Get(ctx, "9b2f-uuid")
```

Os canais são constantes: `ChannelEmail`, `ChannelSMS`, `ChannelWhatsApp`,
`ChannelVoice` e `ChannelLetter`. `aronline.Channels` tem a lista inteira.

### Etiquetas

```go
etiquetas, err := client.Tags.List(ctx)
uma, err := client.Tags.Get(ctx, "12")
```

Etiqueta é **pessoal**: esses métodos respondem às etiquetas de quem está no
token. Token de integração recebe `403` dizendo isso, em vez de uma lista
vazia — que leria como "você não tem nenhuma".

### Lista de permitidos

```go
permitidos, err := client.Allowlist.List(ctx)
```

Também pessoal, pelo mesmo motivo.

### Frescor dos dados

```go
frescor, err := client.Freshness.Get(ctx)

if frescor.SourcesBehind > 0 {
	log.Printf("%d de %d atrasadas", frescor.SourcesBehind, frescor.SourcesTracked)
}
```

Responde a pergunta prática de quando uma consulta devolve menos do que você
esperava: o defeito é da API, ou a carga está atrasada? Sem esse número as
duas hipóteses parecem a mesma coisa.

Ela responde em **contagens**, não numa lista de tabelas: "46 acompanhadas, 3
atrasadas" responde "está fresco?"; quarenta e seis nomes de tabela é um
relatório que ninguém lê na hora em que a pergunta é feita.

Campo que a API responde `null` é ponteiro aqui — `*int`, `*string`. É de
propósito: `nil` e `0` são situações diferentes, e "nenhuma tabela tem marca
de leitura" não é "está tudo em dia".

### Versão

```go
info, err := client.Version.Get(ctx)
fmt.Println(info.Version, info.Environment)
```

O único método que funciona **sem token** — é rota aberta. É o primeiro dado
que o suporte pede.

## Quando dá errado

Toda recusa da API vira `*aronline.APIError`, alcançável por `errors.As`:

```go
_, err := client.Templates.Get(ctx, "nao-existe")

var failure *aronline.APIError
if errors.As(err, &failure) {
	fmt.Println(failure.Code)      // "not_found"
	fmt.Println(failure.Status)    // 404
	fmt.Println(failure.RequestID) // o número que o suporte pede
}
```

O que vem em `APIError`:

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

Repetir a chamada é decisão sua — o SDK não repete sozinho.

Rede fora do ar e resposta que não é JSON (um proxy respondendo no lugar da
API) também chegam como `*APIError`, com `Code` `unreachable` e
`invalid_response`. E a causa continua embaixo: `errors.Is(err,
context.DeadlineExceeded)` funciona como funcionaria sem o SDK.

## Configuração

```go
aronline.New(aronline.Options{
	Token:      "…",                          // opcional: sem ele, só Version funciona
	BaseURL:    "https://v3.ar-online.com.br", // padrão; troque para homologação
	Timeout:    30 * time.Second,             // padrão
	HTTPClient: meuClient,                    // opcional: seu pool, seu proxy
})
```

`Options{}` zerado já é utilizável: aponta para produção sem credencial, que é
o suficiente para `Version.Get`.

## Escopo

Este SDK fala **só a `/v3`**. As rotas `/v1` e `/v2` continuam de pé, mas elas
respondem byte a byte o que as APIs antigas respondiam, idiossincrasias
incluídas — inclusive erro com status `200`. São espelhos para ninguém
precisar migrar no mesmo dia, e um cliente tipado que as "melhorasse"
quebraria exatamente quem elas protegem.

A superfície `/v3` é só de leitura hoje. Escrita entra nos cinco SDKs na mesma
leva em que entrar na API.

Quem precisa do contrato HTTP cru — porque está escrevendo um cliente em outra
linguagem, ou depurando o que passou no fio — encontra em
[docs.ar-online.com.br](https://docs.ar-online.com.br).

## Desenvolvimento

| comando | o que cobra |
|---|---|
| `gofmt -l .` | formato |
| `go vet ./...` | o vet da linguagem |
| `golangci-lint run` | o conjunto do `.golangci.yml` — `bodyclose`, `errorlint`, `gosec`, `noctx`, `revive` |
| `codespell` | ortografia |
| `go test ./... -race -coverprofile=coverage.out` | testes, com detector de corrida |
| `go tool cover -func=coverage.out` | cobertura — o mínimo é **95%** |
| `govulncheck ./...` | vulnerabilidade conhecida |

Este módulo não tem dependência nenhuma, então o que o `govulncheck` cobra é
a **biblioteca padrão** — que é exatamente onde uma CVE de HTTP ou de TLS
apareceria. Por isso ele reprova quando a sua toolchain está atrás: não é o
SDK, é o Go da sua máquina. Atualize o Go e roda limpo; o CI usa `stable`
justamente para nunca ficar para trás.

Os testes ficam ao lado do código, em `package aronline_test` — é a separação
caixa-preta que a linguagem oferece: eles enxergam só a API pública, como
qualquer pessoa que instala o módulo. Um diretório `test/` à parte seria outro
pacote, e `go test ./...` deixaria de medir cobertura deste aqui.

## Licença

[Apache 2.0](LICENSE) — © 2026 AR ONLINE TECNOLOGIA LTDA.
