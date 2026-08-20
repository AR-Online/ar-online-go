# Changelog

Todas as mudanças notáveis deste SDK são documentadas aqui.

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e versionamento [SemVer](https://semver.org/lang/pt-BR/).

O SDK acompanha as duas superfícies da API: rota nova vira função nova aqui, na
mesma leva. Quando uma rota da API legada ganha equivalente na `/v3`, a função
correspondente troca o transporte **sem mudar de assinatura**, e a troca é
registrada aqui, rota a rota.

---

## [Unreleased]

Nada ainda.

## [0.3.0] — 2026-08-20

### Changed

- **As cinco linguagens passam a andar na mesma versão.** Até aqui cada SDK
  numerava por conta própria — o TypeScript na 0.2.1, os outros na 0.1.0 — e
  perguntar "qual versão tem a área de legado?" dava quatro respostas
  diferentes. A partir da 0.3.0 o número é o mesmo nas cinco, e a mesma
  superfície sai no mesmo dia.

### Added

- **A área de legado, em `client.Legacy`**, com tudo o que a documentação
  pública do gateway descreve: envio multicanal (`Send`), status por canal
  (`Status.Email`, `.SMS`, `.WhatsApp`, `.Voz`, `.Carta`), o consolidado
  pericial (`Status.Full`), comprovante (`SendingProof`), laudo (`Laudo`),
  finalização da régua (`FinalizarRegua`) e os modelos do gateway, leitura e
  escrita. É o que fala com a API que está em produção hoje.
- **Duas credenciais no mesmo cliente, sem uma vazar na área da outra.**
  `Options.LegacyToken` vai **cru** no cabeçalho `authorization`, sem `Bearer`
  — o oposto da `/v3`. `Options.LegacyBaseURL` tem padrão próprio
  (`api.ar-online.com.br`) e é independente do endereço da `/v3`. Área de
  legado sem credencial falha **antes do socket**, dizendo qual opção falta.
- **O envelope `{ data, statusCode }` dos modelos, resolvido pelo SDK.** A
  família `/gw/templates` responde HTTP 200 até em erro, e o código que vale é
  o de dentro do corpo. Ler o status HTTP é o defeito nº 1 de quem integra na
  mão; aqui o 403/404/500 de dentro vira `*LegacyAPIError`, com `HTTPStatus`
  registrando o 200 que o fio disse.
- **`*LegacyAPIError`, um tipo só para o contrato antigo**, alcançável por
  `errors.As`. Ele é separado do `*APIError` porque o legado não tem código de
  catálogo nem `request_id`, e struct compartilhada carregaria campos vazios
  que parecem perda de dado.
- **As quatro convenções de ausência preservadas no tipo.** `""`, `null`, a
  chave que some e `{}` chegam como vieram: ponteiro sem `omitempty` onde o
  contrato responde `null`, ponteiro **com** `omitempty` onde a chave some.
  Uma struct que volta para JSON sai com a mesma convenção com que entrou.
  Normalizar quebraria a fidelidade que a área existe para dar.
- **As esquisitices do contrato, mantidas de propósito**: a voz nunca responde
  404 (uuid sem registro é 200 com uma frase, e não é erro); `CustomID` do
  WhatsApp vem sempre nulo; `Answered` do SMS é lista de **objetos**, não de
  texto; a carta expõe `DatePreparation`/`DateDelivery`, renomeadas do
  provedor; `FinalizarRegua` é GET com efeito colateral, e o SDK não
  "conserta" para POST.
- **Data do legado é `string`.** `"18/07/2026 01:01:32"` não traz fuso, e
  nenhum `time.Time` sai daí sem chute.
- **Structs dos payloads de webhook** (`WebhookPayloadV1`, `WebhookPayloadV2`)
  exportadas. O SDK não recebe HTTP por você, mas quem recebe não deveria ter
  de digitar o contrato à mão.
- **Vocabulário do legado nos nomes** (`Laudo`, `FinalizarRegua`,
  `SendingProof`, `EnvioRequest`, `Voz`, `Carta`): exceção deliberada à regra
  do inglês, porque traduzir criaria nomes que não existem em documentação
  nenhuma.
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
  responde. Dublê provaria só que o código chama o dublê. São dois servidores,
  um por superfície, porque o gateway erra de outro jeito.
- CI em três sistemas operacionais × Go 1.23 e `stable` — a 1.23 porque é a
  mínima que o `go.mod` promete, e promessa que ninguém confere quebra na
  máquina do parceiro, não na nossa.
- Publicação **pela tag**: em Go não há registro nem token — o
  `proxy.golang.org` busca o módulo direto do GitHub. O workflow avisa o proxy
  para a versão aparecer no `pkg.go.dev` sem esperar o primeiro `go get`.

Hoje o portão mede: **101 testes (subtestes incluídos), 99,1% de cobertura**.

[Unreleased]: https://github.com/AR-Online/ar-online-go/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/AR-Online/ar-online-go/releases/tag/v0.3.0
