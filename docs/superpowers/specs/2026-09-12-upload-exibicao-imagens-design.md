# Upload e exibição de imagens — design

## Contexto

O projeto já tem um caminho de upload de imagem (`POST /api/images` →
`uploadImage`, `handlers_api.go`) e geração de thumbnail (`createThumbnail`,
`handlers.go`), mas ambos estão incompletos/quebrados:

- `uploadImage` não valida formato, tamanho, nem sanitiza o nome do arquivo;
  descarta todos os erros (`c.MultipartForm()`, `c.SaveUploadedFile`).
- `createThumbnail` está hardcoded pra decodificar só JPEG (`jpeg.Decode`) e
  tem um bug de rename case-sensitive (`.JPG` → `.png`, não cobre `.jpg`,
  `.png`, `.gif`, etc.) — qualquer upload que não seja `.JPG` maiúsculo
  quebra silenciosamente ou gera uma thumbnail com extensão errada.
- Não existe nenhum endpoint para servir a imagem ou a thumbnail de volta.
- Não existe página de controller para upload/seleção de imagem
  (`templates/controllers/` só tem `songs.html` e `bible.html`).
- O painel (`templates/panels/default.html`) ignora completamente
  `data.type` — sempre trata `data.content` como texto, mesmo a constante
  `TypeImage` já existindo em `main.go` (definida, mas nunca usada).

O objetivo desta rodada é completar essa funcionalidade: aceitar upload nos
formatos combinados, corrigir a geração de thumbnail, servir as imagens de
volta, criar uma página de controller dedicada (separada de `songs.html`,
que já está muito cheio) e fazer o painel exibir a imagem na mesma área
onde hoje exibe texto (`main`).

## Escopo

- Upload validado por conteúdo (magic bytes, não extensão) para os formatos
  JPEG, PNG, WEBP, GIF e BMP.
- Limite de 10MB por upload.
- Geração de thumbnail corrigida, sempre em `.png` (independente do formato
  original — necessário porque não há encoder de WEBP disponível na
  biblioteca padrão nem em `golang.org/x/image` sem depender de cgo).
- Novos endpoints HTTP para listar e servir imagens/thumbnails.
- Nova página `templates/controllers/images.html` + módulo JS
  `static/js/images/` (seguindo o padrão de `static/js/songs/`): grade de
  miniaturas de todas as imagens já enviadas + botão para publicar uma no
  provider `main`.
- Painel (`templates/panels/default.html`): `processMainProviderData`
  passa a checar `data.type === 'IMAGE'` e renderizar um `<img>` em vez de
  tratar o conteúdo como texto.
- Nome de arquivo duplicado: sobrescreve o arquivo (e a thumbnail)
  existente, mesmo comportamento de hoje.

**Fora de escopo:**
- Organização por categoria/subpasta (como `storage.Songs` tem para
  músicas) — imagens ficam numa lista simples (flat) em `media/images/`.
- Provider `preview` — não é consumido em nenhum template hoje (nem no
  painel, nem em nenhum controller); a ideia de usá-lo para uma prévia no
  próprio controller de imagens fica para uma rodada futura, quando
  houver um lugar definido na UI pra isso.
- Provider `aux` — só `main` recebe suporte a `Type=IMAGE` nesta rodada.
- Vídeo/áudio (`TypeVideo`/`TypeAudio` já existem como constantes, mas não
  fazem parte desta rodada).
- Reprodução especial de GIF animado — o navegador já anima um `<img>`
  apontando para um GIF automaticamente, então servir os bytes originais
  já resolve isso sem lógica adicional.

## Novo submódulo `presenter/images`

Segue a convenção já estabelecida por `storage`/`providers`/`fsutil`
(módulo próprio via `replace` no `go.mod` raiz).

```go
package images

// MaxUploadSize é o limite de tamanho por upload (10MB).
const MaxUploadSize = 10 << 20

// Save valida o conteúdo de content (magic bytes, tamanho), salva o
// arquivo original em <BasePath>/images/<name> (sobrescrevendo se já
// existir) e gera uma thumbnail em <BasePath>/images/thumbs/<name sem
// extensão>.png. Retorna o nome efetivamente salvo (sanitizado) ou um
// erro se o formato não for suportado, o arquivo exceder MaxUploadSize,
// ou o nome for inválido (ex.: contém separadores de path).
func Save(name string, content io.Reader) (savedName string, err error)

// List retorna os nomes de arquivo em <BasePath>/images (não desce em
// thumbs/), ordenados.
func List() ([]string, error)

// ContentPath retorna o caminho absoluto/relativo do arquivo original de
// name, ou erro se name for inválido ou o arquivo não existir.
func ContentPath(name string) (string, error)

// ThumbPath retorna o caminho da thumbnail de name (sempre .png), ou erro
// se name for inválido ou a thumbnail não existir.
func ThumbPath(name string) (string, error)
```

### Validação de formato

Por conteúdo, não por extensão do nome enviado. Assinaturas (magic bytes)
verificadas nos primeiros bytes do arquivo:

| Formato | Assinatura |
|---|---|
| JPEG | `FF D8 FF` |
| PNG | `89 50 4E 47 0D 0A 1A 0A` |
| GIF | `GIF87a` ou `GIF89a` |
| BMP | `BM` |
| WEBP | `RIFF????WEBP` (bytes 0-3 `RIFF`, bytes 8-11 `WEBP`) |

Um arquivo cujos bytes não batem com nenhuma dessas assinaturas é
rejeitado com `400`, independente da extensão no nome enviado.

### Geração de thumbnail

- Decodifica o formato correto conforme detectado (usa `image/jpeg`,
  `image/png`, `image/gif`, `golang.org/x/image/bmp`,
  `golang.org/x/image/webp` — os dois últimos já cobertos pela dependência
  `golang.org/x/image` já presente no `go.mod`; `webp` só decodifica, o que
  é suficiente pois só precisamos ler pra gerar a miniatura).
- Sempre codifica a saída como PNG (`image/png`), corrigindo o
  case-sensitivity bug atual e o problema de não ter encoder de WEBP.
- Mesma lógica de redimensionamento (`ratio = 8`, `draw.NearestNeighbor`)
  já existente em `getThumbnailDimensions`/`createThumbnail`, migrada pra
  cá.
- Falha ao decodificar (arquivo corrompido que passou no sniff de
  assinatura, mas não é uma imagem válida de fato) não derruba o upload:
  loga o erro, mantém o arquivo original salvo, thumbnail fica ausente
  (o controller mostra um placeholder pra imagem sem thumbnail).

### Sanitização de nome

`name` nunca pode conter `/`, `\`, ou `..` — rejeitado com erro antes de
tocar o filesystem, em `Save`, `ContentPath` e `ThumbPath` igualmente
(mesma preocupação que motivou builds anteriores de sanitização no resto
do projeto, mas ainda inexistente para imagens).

## Rotas HTTP

Novo grupo `registerImageRoutes` (`routes.go`), handlers em
`handlers_images.go` (root module):

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| POST | `/api/images` | sim | Upload — substitui a implementação atual de `uploadImage`, mesma rota, delega pra `images.Save` |
| GET | `/api/images` | não | Lista nomes de arquivo (`images.List`), pro controller montar a grade |
| GET | `/api/images/content` | não | Serve o arquivo original (`?name=`), via `images.ContentPath` |
| GET | `/api/images/thumb` | não | Serve a thumbnail (`?name=`), via `images.ThumbPath` |

`POST /api/images` mantém a mesma rota e método já registrados hoje em
`registerMediaRoutes` — só a implementação do handler muda (passa a
delegar pro pacote novo). As 3 rotas de leitura são novas.

## Frontend

### `templates/controllers/images.html` (novo)

Página de controller dedicada (chega em `/controller/images`, via o
mecanismo já existente de servir `templates/controllers/<page>.html` sem
lista hardcoded no Go). Estrutura:

- Formulário de upload (`<input type="file">` + botão), enviando
  `multipart/form-data` pra `POST /api/images`.
- Grade de miniaturas: ao carregar a página (e após cada upload), busca
  `GET /api/images` e renderiza um `<img>` por nome apontando pra
  `GET /api/images/thumb?name=...`, com um placeholder quando a thumbnail
  não existir (upload cuja geração de thumbnail falhou).
- Clicar numa miniatura publica a imagem: `POST /api/content/set/main`
  com `{content: <nome>, type: 'IMAGE'}`.
- Reaproveita o botão/token de Basic Auth já injetado nos templates
  (`{{BASIC_AUTH_TOKEN}}`) do mesmo jeito que `songs.html` faz.

### `static/js/images/` (novo)

Mesmo padrão de `static/js/songs/client.js` + `songs.js` (módulos ES,
`type="module"`):

- `client.js`: funções de API — `uploadImage(file)` (usa `FormData`, não
  o helper JSON existente `authorizedRequest`, que não serve pra
  multipart), `listImages()`, `emitImageContent(name)` (equivalente ao
  `emitMainContent` de songs, mas com `type: 'IMAGE'`).
- `images.js`: lógica de UI — renderiza a grade, liga os event listeners
  de upload/seleção.

### `templates/panels/default.html`

`processMainProviderData` (função existente) passa a checar
`data.type === 'IMAGE'` antes de tratar `data.content` como texto:

```js
function processMainProviderData(data) {
    const liveBlock = document.getElementById('liveblock');
    const metaBlock = document.getElementById('metablock');
    // ... lógica de clock mode existente, inalterada ...
    if (lastMainText === data.content && lastMainType === data.type) {
        return;
    }
    lastMainText = data.content;
    lastMainType = data.type;

    if (data.type === 'IMAGE') {
        metaBlock.innerHTML = '';
        liveBlock.innerHTML = data.content
            ? `<img src="/api/images/content?name=${encodeURIComponent(data.content)}">`
            : '';
    } else {
        // ... lógica de texto existente (meta/live split, processarConteudo) ...
    }
    // ... lógica de bible chapter / vídeo existente, inalterada ...
}
```

`lastMainType` é uma nova variável de estado (ao lado da já existente
`lastMainText`), necessária porque hoje a função só decide re-renderizar
comparando `data.content` — trocar de uma imagem pra um texto com o mesmo
conteúdo textual (raro, mas possível) precisa dessa checagem adicional
pra não ficar preso no branch errado.

## Erros e edge cases

- Upload com formato não suportado → `400`, nada é salvo.
- Upload acima de 10MB → `400`, leitura interrompida antes de gravar em
  disco (`http.MaxBytesReader` no handler, ou checagem de tamanho em
  `images.Save` via `io.LimitReader(content, MaxUploadSize+1)` e erro se
  os bytes lidos excederem o limite).
- Nome de arquivo inválido (contém `/`, `\`, `..`) → `400` em qualquer uma
  das 4 rotas.
- `GET /api/images/content|thumb` com `name` inexistente → `404`.
- Falha ao gerar thumbnail (decode falha após passar no sniff de
  assinatura) → upload não falha; thumbnail fica ausente, `GET
  /api/images/thumb` retorna `404` pra esse nome até um novo upload
  corrigir.
- Sobrescrita: novo upload com nome já existente substitui arquivo e
  thumbnail antigos sem aviso (comportamento atual mantido).

## Testes

- `images/images_test.go`: `Save` (cada um dos 5 formatos aceito com
  bytes reais de assinatura válida, formato rejeitado, arquivo acima do
  limite, sobrescrita, geração de thumbnail em `.png` pra cada formato
  incluindo WEBP, falha de decode não derruba o save), `List` (ordenado,
  não inclui `thumbs/`), sanitização de nome em `Save`/`ContentPath`/
  `ThumbPath` (rejeita `/`, `\`, `..`).
- `handlers_images_test.go` (root module): os 4 endpoints via `httptest`
  — upload sem auth → `401`, upload válido → `200`/`201` e aparece em
  `GET /api/images`, `GET /api/images/content|thumb` com nome inexistente
  → `404`, upload de formato inválido → `400`.
- Painel (`templates/panels/default.html`): sem framework de teste
  automatizado hoje — validado via smoke test manual no browser (upload
  de cada formato, exibição no painel, troca pra texto depois, tela
  limpa).

## Migração

Nenhuma migração de dados necessária — `media/images/` e
`media/images/thumbs/` já são criadas no boot (`createDefaultFolders`,
inalterado) e estão vazias hoje em qualquer ambiente existente.
