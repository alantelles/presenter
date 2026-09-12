# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## O que é

Presenter é um servidor de apresentação de conteúdo para cultos/eventos ao vivo (letras de música,
versículos bíblicos, imagens/vídeos), escrito em Go com o framework Gin. Um operador controla o
conteúdo através de páginas de "controller" e o conteúdo é exibido em uma página de "painel" (`/live`),
que outras telas/dispositivos abrem no navegador.

Apesar do uso atual ser voltado a música/bíblia, a arquitetura é pensada para ser genérica: qualquer
pessoa pode criar um controller próprio (ver "Descoberta dinâmica de templates" abaixo) sem alterar
código Go. O ponto ainda não implementado é tornar os `providers` (canais de conteúdo) também
dinâmicos/configuráveis, para suportar múltiplas telas/casos de uso além dos fixos atuais.

## Comandos

Este é um módulo Go raiz (`module main`, Go 1.24) com dois submódulos locais referenciados via
`replace` no `go.mod`: `presenter/bible` (`./bible`) e `presenter/flags` (`./flags`).

```bash
go build -v ./...          # build (usado no CI)
go run .                   # rodar localmente (porta 8080 fixa)
go run . -usuario admin -senha admin -endereco http://localhost:8080   # com flags
go vet ./...
```

Não há testes automatizados no repositório no momento — não presuma um comando `go test` funcional
sem antes checar se algum arquivo `_test.go` foi adicionado.

Flags de execução (definidas em `flags/flags.go`):
- `-endereco`: endereço público usado para substituir `{{APP_LOCATION}}` nos templates HTML.
- `-usuario` / `-senha`: credenciais de Basic Auth (default `admin`/`admin`) usadas pelo `AuthMiddleware`.

## Arquitetura

**Módulo raiz (`main.go`, `handlers*.go`, `manager.go`, `providers.go`, `letras.go`)** — toda a
aplicação principal está no pacote `main`, sem subpastas internas além dos módulos Go separados.

- **Providers (`providers.go`)**: o estado central da aplicação é o mapa `providers` — canais fixos
  como `main`, `preview`, `aux`, `command`, `operator`, `sound-engineer`, `alerts`, hardcoded no código
  (não configurável em runtime). Um "controller" envia conteúdo via `POST /api/content/set/:providerId`,
  e o painel (ou outros consumidores) lê via `GET /api/content?providerId=...`. Não há push/websocket —
  o cliente precisa fazer polling. **Pendente**: permitir múltiplos providers dinâmicos/configuráveis,
  para alimentar telas e casos de uso além dos fixos atuais.
- **Descoberta dinâmica de templates / controllers (`handlers.go`)**: `viewController` recebe `:page`
  via rota (`/controller/:page`) e simplesmente serve `templates/controllers/<page>.html` do disco —
  não há lista de páginas hardcoded no Go. Ou seja, criar um novo controller (ex.: para um novo tipo
  de conteúdo) é só adicionar um `.html` em `templates/controllers/`, sem alterar código. É esse
  mecanismo que torna a aplicação extensível além de músicas/bíblia.
- **Templates (`templates/`)**: HTML servido diretamente do disco (não é `html/template` do Go).
  `getHtmlPage` (`handlers.go`) lê o arquivo e faz substituição de tokens de texto:
  `{{APP_LOCATION}}` → endereço configurado, `{{BASIC_AUTH_TOKEN}}` → credenciais em base64. Isso
  injeta o endereço/token diretamente no HTML servido para as páginas de controller/painel/index
  usarem nas chamadas de API.
- **Mídia (`manager.go`)**: músicas/letras são arquivos `.txt` simples em `media/songs/` (criado em
  runtime por `createDefaultFolders`), organizados livremente em subpastas ("archive"/"folder").
  Imagens ficam em `media/images/` com thumbnails gerados em `media/images/thumbs/`
  (`createThumbnail` em `handlers.go`, usando `golang.org/x/image/draw`).
- **Letras externas (`letras.go`)**: faz scraping por regex do site letras.mus.br (busca por artista →
  lista de músicas → id da música → HTML da letra), sem client HTTP dedicado ou parsing de DOM. É
  frágil a mudanças de layout do site.

**Submódulo `bible/`** (`presenter/bible`): busca e cacheia capítulos/versões bíblicas.
`bible.go` expõe os handlers Gin (`GetBooksList`, `GetChapter`); `bible_client.go` busca de uma API
externa; `bible_cache.go` persiste o resultado em disco sob `bible/content/<versão>/<livro>/...json`
para evitar rebuscar. O cache é checado antes de qualquer chamada externa.

**Submódulo `flags/`** (`presenter/flags`): parsing de flags de linha de comando via pacote `flag`
padrão, expostas por getters (`GetLocation`, `GetUsername`, etc.) para o resto da aplicação.

**Autenticação**: Basic Auth simples e global (`AuthMiddleware` em `main.go`), aplicada apenas às
rotas de escrita/administração (`/api/content/set`, `/api/media`, `/controller`, `/api/images`).
Rotas de leitura pública (painel, descoberta, letras, bíblia) não exigem autenticação.

**CORS**: liberado para qualquer origem (`Access-Control-Allow-Origin: *`) em praticamente todos os
handlers, via função `CORS` duplicada em `main.go` e em `bible/bible.go`.

**Distribuição (`.github/workflows/release.yml`)**: builds standalone para Linux e Windows
(`GOOS=linux`/`windows`, `GOARCH=amd64`), empacotados junto com `bible/content/`, `static/`,
`templates/` e o respectivo script `organiza.sh`/`organiza.bat`. Versão e changelog são gerados
automaticamente por tag semver a cada push em `main`.
