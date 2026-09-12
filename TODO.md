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

## Pendente

- [ ] **Regexp recompilado a cada chamada** em `letras.go`
      (`findArtistPathInIndex`, `findSongLyricsId`) — extrair como `var` de pacote
      (`regexp.MustCompile`) uma vez só.
- [ ] **`main()` com ~20 rotas inline** — agrupar em funções tipo
      `registerMediaRoutes(r)`, `registerViewRoutes(r)`, `registerLyricsRoutes(r)`,
      `registerBibleRoutes(r)`.
- [ ] **Estado global mutável** (`port`, `location`, `usePort`,
      `basicAuthUser`/`basicAuthPass`, `providers`) — encapsular num `struct
      Config`/`struct App` passado explicitamente, em vez de `var` de pacote
      setados em `main()`/`varSetup()`. Melhora testabilidade.
- [ ] **`categories` como array + busca linear para 1 elemento só**
      (`manager.go:19-32`) — se a ideia é crescer, virar `map[string]Category`;
      se não, simplificar direto.
- [ ] **Pacote raiz monolítico (`main`)** — considerar separar em
      `internal/media`, `internal/providers`, `internal/httpapi`, etc., se o
      projeto crescer.
- [ ] **Content-Type inconsistente** — existem constantes como
      `ContentTypeText = "text/plain; charset=utf-8"` em `main.go`, mas alguns
      handlers hardcodam `"text/plain; charset=UTF-8"` (casing diferente) em vez
      de usar a constante.
- [ ] **Providers dinâmicos** (já registrado no `CLAUDE.md`) — hoje o mapa
      `providers` é fixo/hardcoded; a ideia é permitir múltiplos providers
      configuráveis para alimentar outras telas além dos canais atuais.
