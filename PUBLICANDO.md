# Publicando o módulo Go

⚠️ **Não há registro, não há conta, não há token** — e portanto não há Trusted
Publishing a configurar. Em Go a **tag é a publicação**: o
`proxy.golang.org` busca o módulo direto do GitHub na primeira vez que alguém
pede aquela versão.

Isso significa que o caminho já é o mais seguro possível: nenhuma credencial
existe para vazar.

## O que configurar

Nada, nos dois lados. O [`release.yml`](.github/workflows/release.yml) só
precisa de `contents: write` para criar a release, que já está declarado.

## Como publicar

```bash
# 1. a tag, prefixada com v e em SemVer
git tag v0.1.0
git push origin v0.1.0
```

O workflow roda build, vet e testes, avisa o proxy e cria a release.

O aviso ao proxy é o passo que faz a versão aparecer em
<https://pkg.go.dev/github.com/AR-Online/ar-online-go> sem esperar o primeiro
`go get` de alguém — é só um `go list -m` contra `proxy.golang.org`.

## O que não dá para desfazer

O proxy é **imutável por desenho**: uma vez que ele serviu `v0.1.0`, aquele
conteúdo fica em cache para sempre, mesmo que a tag seja apagada ou movida.
Apagar a tag no GitHub não tira a versão de circulação.

Se uma versão saiu errada, o caminho é publicar a seguinte — `v0.1.1` — e,
se ela for perigosa, registrar um `retract` no `go.mod`:

```
retract v0.1.0 // publicada com o caminho de módulo errado
```

Não existe `yank` em Go. Mover tag é o erro clássico aqui, e ele produz duas
máquinas com código diferente sob o mesmo número de versão.
