# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## O que é

Presenter é um servidor de apresentação de conteúdo para cultos/eventos ao vivo (letras de música,
versículos bíblicos, imagens/vídeos), escrito em Go com o framework Gin. Um operador controla o
conteúdo através de páginas de "controller" e o conteúdo é exibido em uma página de "painel" (`/live`),
que outras telas/dispositivos abrem no navegador.

Apesar do uso atual ser voltado a música/bíblia, a arquitetura é pensada para ser genérica: qualquer
pessoa pode criar um controller próprio (ver "Descoberta dinâmica de templates" abaixo) sem alterar
código Go. Os `providers` (canais de conteúdo) também são configuráveis em runtime — ver "Providers"
abaixo.

## Comandos

Este é um monorepo multi-módulo: o módulo raiz (`module presenter`, Go 1.24) referencia via `replace`
no `go.mod` os submódulos locais `presenter/bible` (`./bible`), `presenter/flags` (`./flags`),
`presenter/fsutil` (`./fsutil`), `presenter/storage` (`./storage`) e `presenter/providers`
(`./providers`) — cada um com seu próprio `go.mod`.

```bash
go build -v ./...          # build (usado no CI)
go run .                   # rodar localmente (porta 8080 fixa)
go run . -usuario admin -senha admin -endereco http://localhost:8080   # com flags
go vet ./...
```

Há testes automatizados em vários pacotes (raiz, `storage/`, `providers/`). Como é um monorepo
multi-módulo (veja "Arquitetura"), `go test ./...` na raiz só cobre o módulo raiz — rode
`go test ./...` dentro de `storage/` e `providers/` também para cobrir esses módulos.

Flags de execução (definidas em `flags/flags.go`):
- `-endereco`: endereço público usado para substituir `{{APP_LOCATION}}` nos templates HTML.
- `-usuario` / `-senha`: credenciais de Basic Auth (default `admin`/`admin`) usadas pelo `AuthMiddleware`.
- `-pastaMedia`: caminho (relativo ou absoluto) onde músicas/imagens são armazenadas (default `media`).
  Configura `storage.SetBasePath` uma vez no bootstrap (`main()`).

## Arquitetura

**Módulo raiz** (`main.go`, `app.go`, `routes.go`, `handlers*.go`, `providers.go`, `letras.go`) — pacote
`main`. `main()` só faz bootstrap (flags, `Config`/`App`, registro de rotas); tudo que dependia de
estado global mutável foi encapsulado:

- **`app.go`**: `Config` é resolvida uma vez no startup (`NewConfig`, a partir das flags) e `App`
  agrupa `Config` + `Providers` (`*providers.Store`). Handlers que precisam desse estado
  (`AuthMiddleware`, `viewPanel`/`viewController`/`viewHome`, `setMediaProviderContent`/
  `getMediaProviderContent`, `CopyIncomingProviderToExistent`) são métodos de `*App` em vez de
  funções soltas lendo `var` de pacote.
- **`routes.go`**: registro de rotas agrupado por domínio (`registerMediaRoutes`,
  `registerViewRoutes`, `registerMiscRoutes`, `registerLyricsRoutes`, `registerBibleRoutes`),
  cada uma recebendo `*gin.Engine` (e `*App` quando precisa de estado).
- **Providers (`providers.go`, `handlers_providers.go` + submódulo `presenter/providers`)**: canais de
  conteúdo vivem num `providers.Store`, persistido em `providers.json` (raiz do processo, no
  `.gitignore` — arquivo de runtime, não faz parte do repo). Quatro providers são protegidos e sempre
  existem (`main`, `preview`, `aux`, `command`); além deles, providers customizados podem ser criados e
  removidos em runtime via HTTP. O `Store` expõe `Get`/`Set` (conteúdo de um provider existente),
  `List()` (id + label de todos), `Create(id, label string) error` (cria ou atualiza o label de um
  provider, preservando conteúdo existente) e `Delete(id string) error` / `DeleteAll() error`
  (`Delete` de um protegido retorna `providers.ErrProtected`; de um inexistente,
  `providers.ErrNotFound` — ambos com `errors.Is`; `DeleteAll` remove só os customizados). Endpoints
  HTTP (`handlers_providers.go`, registrados via `registerProviderRoutes` em `routes.go`):
  `GET /api/providers` (lista, sem auth), `POST /api/providers` (cria/atualiza label, requer
  `AuthMiddleware`), `DELETE /api/providers/:id` (remove um; 400 se protegido, 404 se inexistente,
  requer auth) e `DELETE /api/providers` (remove todos os customizados, requer auth). Um "controller"
  envia conteúdo via `POST /api/content/set/:providerId`, e o painel (ou outros consumidores) lê via
  `GET /api/content?providerId=...`. Não há push/websocket — o cliente precisa fazer polling.
- **Descoberta dinâmica de templates / controllers (`handlers.go`)**: `viewController` recebe `:page`
  via rota (`/controller/:page`) e simplesmente serve `templates/controllers/<page>.html` do disco —
  não há lista de páginas hardcoded no Go. Ou seja, criar um novo controller (ex.: para um novo tipo
  de conteúdo) é só adicionar um `.html` em `templates/controllers/`, sem alterar código. É esse
  mecanismo que torna a aplicação extensível além de músicas/bíblia.
- **Templates (`templates/`)**: HTML servido diretamente do disco (não é `html/template` do Go).
  `(*App).getHtmlPage` (`handlers.go`) lê o arquivo e faz substituição de tokens de texto:
  `{{APP_LOCATION}}` → endereço configurado, `{{BASIC_AUTH_TOKEN}}` → credenciais em base64. Isso
  injeta o endereço/token diretamente no HTML servido para as páginas de controller/painel/index
  usarem nas chamadas de API.
- **Letras externas (`letras.go`)**: faz scraping por regex do site letras.mus.br (busca por artista →
  lista de músicas → id da música → HTML da letra), sem client HTTP dedicado ou parsing de DOM. É
  frágil a mudanças de layout do site.

**Submódulo `storage/`** (`presenter/storage`): armazenamento em disco de músicas/letras — categorias,
listar pastas/arquivos, salvar/carregar texto. O diretório base é configurável em runtime via
`SetBasePath`/`BasePath` (ver flag `-pastaMedia` abaixo); o default é `media/`. **Cuidado**: não reuse
o nome "media" para módulos/pacotes novos — é o nome da pasta de dados em runtime (gitignored).

**Submódulo `bible/`** (`presenter/bible`): busca e cacheia capítulos/versões bíblicas.
`bible.go` expõe os handlers Gin (`GetBooksList`, `GetChapter`); `bible_client.go` busca de uma API
externa; `bible_cache.go` persiste o resultado em disco sob `bible/content/<versão>/<livro>/...json`
para evitar rebuscar. O cache é checado antes de qualquer chamada externa.

**Submódulo `flags/`** (`presenter/flags`): parsing de flags de linha de comando via pacote `flag`
padrão, expostas por getters (`GetLocation`, `GetUsername`, `GetMediaPath`, etc.) para o resto da
aplicação.

**Submódulo `fsutil/`** (`presenter/fsutil`): helpers de I/O compartilhados (`PathExists`,
`CreateFolder`, `WriteTextFile`, `ReadTextFile`), usados por `storage/` e `bible/`.

**Submódulo `images/`** (`presenter/images`): upload, armazenamento e recuperação de imagens de
apresentação. `Save` valida formato por content-sniffing dos primeiros bytes (JPEG/PNG/GIF/BMP/WEBP,
não pela extensão do nome) e tamanho (limite de 10MB, `MaxUploadSize`), e grava tudo achatado (sem
subpastas por categoria) sob `storage.BasePath()+"images/"`. Toda imagem aceita ganha uma miniatura
gerada como PNG (`thumbnail.go`), independente do formato original, guardada em
`storage.BasePath()+"images/thumbs/"`. Endpoints HTTP (`handlers_images.go`, registrados via
`registerImageRoutes` em `routes.go`): `POST /api/images` (upload multipart, campo `files`, requer
`AuthMiddleware`), `GET /api/images` (lista nomes, público), `GET /api/images/content?name=...`
(serve o arquivo original, público) e `GET /api/images/thumb?name=...` (serve a miniatura, público).

**Autenticação**: Basic Auth simples e global (`(*App).AuthMiddleware` em `app.go`), aplicada apenas
às rotas de escrita/administração (`/api/content/set`, `/api/media`, `/controller`, `/api/images`).
Rotas de leitura pública (painel, descoberta, letras, bíblia) não exigem autenticação.

**CORS**: liberado para qualquer origem (`Access-Control-Allow-Origin: *`) via um único
`CORSMiddleware` (`main.go`) registrado globalmente no router — cobre rotas de qualquer pacote,
inclusive `bible/`.

**Distribuição (`.github/workflows/release.yml`)**: builds standalone para Linux e Windows
(`GOOS=linux`/`windows`, `GOARCH=amd64`), empacotados junto com `bible/content/`, `static/`,
`templates/` e o respectivo script `organiza.sh`/`organiza.bat`. Versão e changelog são gerados
automaticamente por tag semver a cada push em `main`.
