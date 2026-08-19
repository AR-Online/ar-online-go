# AR Online — SDK Go

Cliente oficial da API do AR Online para Go.

> **Estado: em construção.** O que está pronto é o repositório — licença,
> empacotamento e publicação. O cliente HTTP ainda não: hoje o módulo exporta
> só o endereço padrão e a versão. Enquanto isso, a API responde a qualquer
> cliente HTTP — o contrato está abaixo e em
> [docs.ar-online.com.br](https://docs.ar-online.com.br).

## Instalação

```bash
go get github.com/AR-Online/ar-online-go
```

Go 1.23 ou mais novo.

```go
import "github.com/AR-Online/ar-online-go"
```

O caminho do módulo termina em `ar-online-go`, mas o **pacote** se chama
`aronline` — é assim que ele entra no seu código.

## A API que este SDK fala

Só a **`/v3`**. As rotas `/v1` e `/v2` existem e continuam de pé, mas elas
respondem **byte a byte** o que as APIs antigas respondiam, idiossincrasias
incluídas — inclusive erro com status `200`. São espelhos para ninguém precisar
migrar no mesmo dia, não contrato novo, e um cliente tipado que as
"melhorasse" quebraria exatamente quem elas existem para não quebrar.

### Endereço

```
https://v3.ar-online.com.br/v3/<recurso>
```

### Autenticação

Token **JWT RS256** no cabeçalho:

```
Authorization: Bearer <token>
```

O token é emitido pelo emissor do AR Online — **a API não emite token**, ela
só tem a chave pública e verifica. Dois tipos de identidade circulam:

- **pessoa** — o token traz `sub`. Rotas pessoais (etiquetas, lista de
  permitidos) respondem a este;
- **integração** — sem `sub`, ligado à entidade. Serve para servidor a
  servidor; nas rotas pessoais recebe `403` dizendo isso, e não uma lista
  vazia (que leria como "você não tem nada").

Cada rota exige uma permissão nominal, que vem na claim `permissions` do
token — a tabela abaixo diz qual.

### Formato das respostas

Sucesso vem envelopado em `data`:

```json
{ "data": [{ "id": "…", "name": "…" }] }
```

Falha vem envelopada em `error`, com status HTTP de verdade:

```json
{
  "error": {
    "code": "not_found",
    "message": "Modelo não encontrado.",
    "request_id": "0f3a…"
  }
}
```

O catálogo de códigos:

| status | `code` | quando |
|---|---|---|
| 400 | `invalid_request` | a requisição não pôde ser lida (inclui filtro desconhecido) |
| 401 | `unauthenticated` | credencial ausente ou inválida |
| 403 | `forbidden` | autenticado, sem permissão para a ação |
| 404 | `not_found` | não existe — **ou** não é seu (responder 403 contaria que existe) |
| 409 | `conflict` | conflito com o estado atual |
| 422 | `business_rule` | recusado pela regra de negócio |
| 429 | `rate_limited` | limite excedido — veja o cabeçalho `Retry-After` |
| 503 | `unavailable` | indisponível no momento — veja `Retry-After` |
| 500 | `internal_error` | falha nossa |

Toda resposta, com erro ou sem, traz `X-Request-Id`. É o `request_id` do corpo
e o primeiro dado que o suporte pede.

### O que a /v3 responde hoje

| rota | permissão | responde |
|---|---|---|
| `GET /v3/templates` | `templates:read` | os modelos que a sua identidade alcança |
| `GET /v3/templates/{id}` | `templates:read` | um modelo pelo uuid público |
| `GET /v3/tags` | `tags:read` | as suas etiquetas (token de pessoa) |
| `GET /v3/tags/{id}` | `tags:read` | uma etiqueta sua |
| `GET /v3/allowlist` | `allowlist:read` | os destinatários permitidos (token de pessoa) |
| `GET /v3/freshness` | `freshness:read` | há quanto tempo a cópia dos dados foi atualizada |
| `GET /v3/version` | — | versão da API, migration mínima e ambiente (rota aberta) |

A superfície está crescendo. O documento OpenAPI em
`https://v3.ar-online.com.br/docs/openapi.json` é sempre a lista completa do
que está no ar.

## Desenvolvimento

```bash
go build ./...
go vet ./...
go test ./...
```

## Licença

[Apache 2.0](LICENSE) — © 2026 AR ONLINE TECNOLOGIA LTDA.
