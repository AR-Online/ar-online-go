# Changelog

Todas as mudanças notáveis deste SDK são documentadas aqui.

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e versionamento [SemVer](https://semver.org/lang/pt-BR/).

O SDK acompanha a superfície `/v3` da API: rota nova na API vira função nova
aqui, na mesma leva.

---

## [Unreleased]

Tudo abaixo entra na **0.1.0**, a primeira publicação. Enquanto a tag não sai,
o conteúdo fica aqui — ver [PUBLICANDO.md](PUBLICANDO.md).

### Added

- **O cliente da `/v3`**, com os cinco recursos que a API responde hoje:
  modelos (listar com filtro de canal, buscar por id), etiquetas (listar,
  buscar), lista de permitidos (listar), frescor da carga e versão. Quem
  instala não escreve HTTP: não monta URL, não põe cabeçalho, não desembrulha
  envelope e não lê status para saber se deu certo.
- **O envelope resolvido por rota.** `templates`, `tags` e `allowlist`
  respondem `{"data": …}`; `freshness` e `version` respondem o objeto direto.
  Desembrulhar tudo, ou nada, quebra metade das chamadas — a escolha é do SDK,
  e quem chama nem sabe que existe envelope.
- **Um tipo de falha só.** Recusa do catálogo, proxy respondendo HTML no lugar
  da API e rede fora do ar chegam todos como `*APIError`. Não há erro de parser
  cru vazando para quem chamou.
- **`RequestID` como campo de primeira classe**, e não detalhe enterrado: é
  o primeiro dado que o suporte pede, e um SDK que o engolisse obrigaria quem
  bateu na falha a reproduzir tudo no `curl` só para achar o número.
- **A rota aberta funciona sem credencial.** `Version.Get` é pública; um
  cliente construído sem token chama ela, o que serve para conferir a
  instalação antes de ter credencial. Exigir token no construtor tornaria
  inalcançável justamente a rota que o suporte pede primeiro.
- **`Retry-After` já lido em segundos**, com `Retryable()` dizendo se vale
  repetir. **Repetir é decisão de quem chama** — o SDK não repete sozinho,
  porque só quem chamou sabe se a operação pode acontecer duas vezes.
- **`context.Context` em todo método**: cancelamento e prazo continuam sendo
  de quem chama, como no resto do programa dele.
- **A causa continua alcançável.** O erro embrulha, mas não esconde:
  `errors.Is(err, context.DeadlineExceeded)` funciona como funcionaria sem o
  SDK.
- **Campo que a API responde `null` é ponteiro**, não zero — `nil` e `0` são
  situações diferentes, e "nenhuma fonte tem marca de leitura" não é "está
  tudo em dia".
- **Zero dependência.** Só a biblioteca padrão: nada entra no `go.sum` de quem
  instala por causa deste SDK.

### Quality

- Portão com lint, formato, ortografia (codespell), **cobertura mínima de
  95%** e auditoria de dependência. Nada com `allow_failure`, que é a mesma
  regra do portão da API.
- Os testes falam com um **servidor de verdade numa porta livre**, não com um
  dublê de HTTP: o que um SDK precisa acertar é justamente o fio — qual rota
  embrulha, como a recusa volta, o que acontece quando algo que não é a API
  responde. Dublê provaria só que o código chama o dublê.
- CI em três sistemas operacionais × Go 1.23 e `stable` — a 1.23 porque é a
  mínima que o `go.mod` promete, e promessa que ninguém confere quebra na
  máquina do parceiro, não na nossa.
- Publicação **pela tag**: em Go não há registro nem token — o
  `proxy.golang.org` busca o módulo direto do GitHub. O workflow avisa o proxy
  para a versão aparecer no `pkg.go.dev` sem esperar o primeiro `go get`.

Hoje o portão mede: **37 testes (subtestes incluídos), 97,9% de cobertura**.

[Unreleased]: https://github.com/AR-Online/ar-online-go/commits/main
