# TODO — Refatorações de organização e estilo (backend)

Lista levantada numa sessão de revisão do backend Go. Itens concluídos ficam marcados;
o restante é para revisitar quando houver tempo.

## Feito

- [x] Nil pointer em `saveMedia`/`moveMedia` (`handlers_api.go`) — categoria inválida
      agora retorna 400 em vez de panic.
- [x] Nil pointer em `FetchChapter` (`bible/bible_client.go`) — não acessa mais
      `resp.StatusCode` quando `resp` é nil.
- [x] CORS duplicado em cada handler (e copiado em `bible/bible.go`) — virou um
      `CORSMiddleware` único registrado no router.
- [x] **Helpers de arquivo duplicados** — `pathExists`/`createFolder`/`saveTextFile`
      (`manager.go`) e `PathExists`/`CreateFolder`/`SaveTextFile`/`LoadTextFile`
      (`bible/bible_cache.go`) agora delegam pro módulo novo `presenter/fsutil`
      (`PathExists`, `CreateFolder`, `WriteTextFile`, `ReadTextFile`). De quebra,
      `LoadTextFile` deixou de correr risco de nil pointer se `os.Open` falhasse
      (usa `os.ReadFile` por baixo agora).
- [x] **`loadMediaList`/`loadMediaListFromFolder`/`loadSongFolders`** (`manager.go`)
      — consolidadas num único `listDirEntries(path, wantDirs)`.
- [x] **Regexp recompilado a cada chamada** em `letras.go`
      (`findArtistPathInIndex`, `findSongLyricsId`) — como os patterns interpolavam
      artista/música vindos do usuário direto no texto do regex (risco de injeção
      de regex e recompilação a cada request), viraram padrões genéricos
      compilados uma vez (`artistLinkPattern`, `songLinkPattern`) com filtro feito
      em Go (`strings.EqualFold`) em vez de embutir o valor no pattern.
      `getSongNameAndArtistName` também passou a usar patterns estáticos
      (`trackNamePattern`, `artistNamePattern`). Adicionado `letras_test.go` com
      casos pra essas três funções — o que exigiu também renomear o módulo raiz
      de `main` pra `presenter` no `go.mod`, já que `go test` se recusa a rodar
      num módulo cujo caminho é literalmente `"main"` (`cannot import "main"`).
      Validado com testes sintéticos e também contra o site real
      (`/api/lyrics/letras`).

- [x] **`main()` com ~20 rotas inline** — agrupadas em `routes.go`
      (`registerMediaRoutes`, `registerViewRoutes`, `registerMiscRoutes`,
      `registerLyricsRoutes`, `registerBibleRoutes`). `main()` ficou só com o
      bootstrap. `routes_test.go` verifica via `router.Routes()` que cada
      grupo registra exatamente os endpoints esperados, sem invocar handlers.
- [x] **`categories` como array + busca linear para 1 elemento só** — virou
      `map[string]Category` em `manager.go` (depois movido pra
      `presenter/storage`, ver abaixo). `TestFindCategoryByName` cobre
      categoria conhecida/desconhecida.
- [x] **Content-Type inconsistente** — `handlers_api.go`/`handlers_letras.go`
      passaram a usar a constante `ContentTypeText` já existente; `bible/bible.go`
      ganhou sua própria `ContentTypeJSON` (módulo separado). Teste HTTP
      (`handlers_api_test.go`) valida o header em `getSongContent`.
- [x] **Estado global mutável** (`port`, `location`, `usePort`,
      `basicAuthUser`/`basicAuthPass`, `providers`) — encapsulado em `app.go`:
      `Config` (resolvida uma vez via `NewConfig`) + `App` (Config + Providers).
      Handlers que dependiam de globals (`AuthMiddleware`,
      `viewPanel`/`viewController`/`viewHome`,
      `setMediaProviderContent`/`getMediaProviderContent`,
      `CopyIncomingProviderToExistent`) viraram métodos de `*App`.
      `app_test.go` cobre `AuthMiddleware` (credenciais válidas/inválidas/
      ausentes), substituição de tokens e providers.
- [x] **Pacote raiz monolítico (`main`)** — extraídos `presenter/storage`
      (categorias, listagem/gravação de texto — era `manager.go`) e
      `presenter/providers` (`Store` com `Get`/`Set` no lugar do
      `map[string]ProviderData` solto). Seguiu a convenção já existente no
      repo (módulos via `replace` no `go.mod`, como `bible`/`flags`/`fsutil`)
      em vez de introduzir `internal/`. `storage_test.go` e
      `providers_test.go` cobrem os dois. **Cuidado**: o módulo chama-se
      `storage`, não `media` — colide com a pasta `media/` de dados em
      runtime do app (ver `.gitignore`) se usar esse nome.

- [x] **Providers dinâmicos** — `providers.Store` ganhou `Data.Label`, 4
      providers protegidos (main/preview/aux/command), persistência em
      `providers.json` e os métodos `List`/`Create`/`Delete`/`DeleteAll`
      (`providers/providers.go`). Endpoints HTTP em `handlers_providers.go`
      (`GET/POST/DELETE /api/providers`, `DELETE /api/providers/:id`),
      registrados via `registerProviderRoutes` (`routes.go`, chamado em
      `main.go`). Ver plano/spec em
      `.superpowers/sdd/2026-09-12-providers-dinamicos/`.

- [x] **Upload e exibição de imagens** — novo submódulo `presenter/images`
      (`images.go`, `thumbnail.go`) valida formato por content-sniffing
      (JPEG/PNG/GIF/BMP/WEBP), limite de 10MB, armazenamento achatado sob
      `storage.BasePath()+"images/"` e miniaturas sempre em PNG. Endpoints
      em `handlers_images.go` (`GET/POST /api/images`, `GET
      /api/images/content`, `GET /api/images/thumb`, registrados via
      `registerImageRoutes` em `routes.go`), painel (`panel.html`) passou a
      renderizar conteúdo `type=IMAGE` como `<img>`, e novo controller
      (`templates/controllers/images.html`) para upload/galeria, linkado a
      partir de `templates/index.html`. Ver spec/plano em
      `docs/superpowers/specs/2026-09-12-upload-exibicao-imagens-design.md`
      e `docs/superpowers/plans/2026-09-12-upload-exibicao-imagens.md`.

## Pendente

(nenhum item pendente no momento)
