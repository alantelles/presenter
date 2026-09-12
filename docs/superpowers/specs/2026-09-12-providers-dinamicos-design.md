# Providers dinâmicos — design

## Contexto

O `providers.Store` (`presenter/providers`) hoje nasce com um conjunto fixo e
hardcoded de canais de conteúdo (`main`, `preview`, `aux`, `internal`,
`command`, `operator`, `sound-engineer`, `alerts`), todos sem nome de
exibição, todos igualmente "fixos" — não há como criar, listar ou apagar
providers. Esse item já estava registrado como pendência conhecida em
`TODO.md` e `CLAUDE.md`, fora do escopo da rodada de refatoração anterior por
ser uma feature nova.

O objetivo desta rodada é permitir múltiplos providers configuráveis em
runtime, para alimentar outras telas além dos canais fixos atuais, sem
introduzir um banco de dados.

## Escopo

- Criar/apagar providers via API HTTP, em runtime, sem reiniciar o servidor.
- Persistir em disco (JSON simples) para sobreviver a um restart.
- Reduzir o conjunto de providers "fixos" (protegidos contra delete, com
  nome de exibição) a 4: `main`, `preview`, `aux`, `command`. Os demais
  atuais (`internal`, `operator`, `sound-engineer`, `alerts`) saem do boot
  padrão — só voltam a existir se alguém os recriar via API.
- Fora de escopo: banco de dados, UI de controller para gerenciar
  providers (só a API), autenticação diferente da já existente
  (`AuthMiddleware`/Basic Auth), concorrência distribuída (múltiplas
  instâncias do servidor compartilhando o mesmo arquivo).

## Modelo de dados

`providers.Data` ganha um campo `Label string` (nome de exibição), além dos
já existentes `Content`, `Type`, `ContentID`:

```go
type Data struct {
    Label     string `json:"label,omitempty"`
    Content   string `json:"content"`
    Type      string `json:"type,omitempty"`
    ContentID string `json:"contentId,omitempty"`
}
```

Os 4 providers protegidos nascem com label pré-definida:

| ID | Label |
|---|---|
| `main` | Conteúdo principal |
| `preview` | Prévia conteúdo |
| `aux` | Visão auxiliar |
| `command` | Linha de comandos |

## Store

`Store` (`providers/providers.go`) ganha:

- Um `sync.Mutex` para proteger leitura/escrita concorrente do mapa e do
  arquivo em disco (hoje inexistente; passa a ser necessário porque agora
  há escrita em disco disparada por requests HTTP concorrentes).
- Um conjunto fixo de IDs protegidos (`main`, `preview`, `aux`, `command`)
  usado só para decidir o que `Delete`/`DeleteAll` recusam apagar — não
  limita `Create`/`Set`.
- Persistência em `./providers.json` (caminho fixo, relativo ao diretório
  de trabalho do processo — independente da flag `-pastaMedia`, que é só
  para mídia de música/imagem).

### API do Store

```go
func NewStore() *Store              // carrega ./providers.json se existir; senão cria só os 4 protegidos e persiste
func (s *Store) Get(id string) Data
func (s *Store) Set(id string, content Data) error   // já existe; content (Content/Type/ContentID) de um provider existente; agora também persiste
func (s *Store) List() []ProviderInfo                // novo: []{ID, Label} de todos os providers, ordenado por ID
func (s *Store) Create(id, label string) error       // novo: cria ou sobrescreve (label); zera Content/Type/ContentID se for novo, preserva se já existir
func (s *Store) Delete(id string) error              // novo: erro se protegido ou inexistente; senão remove e persiste
func (s *Store) DeleteAll()                          // novo: remove todos exceto os protegidos; persiste
```

`Set` mantém a assinatura atual (erro se `id` não existe) — `Create` é o
novo caminho para registrar um ID que ainda não existe (ou redefinir o
label de um existente).

### Persistência

- Formato: JSON do mapa completo `map[string]Data` (chave = ID do
  provider), salvo em `./providers.json`.
- Toda mutação (`Create`, `Delete`, `DeleteAll`, `Set`) reescreve o arquivo
  inteiro (volume baixo, sem necessidade de writes incrementais).
- Falha ao escrever (disco cheio, permissão): loga o erro e retorna erro
  para a operação HTTP que disparou a mutação (500); o estado em memória já
  foi alterado e fica à frente do disco até a próxima escrita bem-sucedida
  — aceitável para o escopo atual (single instância, sem HA).
- Se `./providers.json` existir mas estiver corrompido/ilegível no boot: o
  processo falha o startup com um erro claro (não tenta "adivinhar" ou
  descartar o arquivo silenciosamente).

## Endpoints HTTP

Novo grupo `registerProviderRoutes` (`routes.go`), registrado a partir de um
novo arquivo `handlers_providers.go`:

| Método | Rota | Auth | Body/Query | Resposta |
|---|---|---|---|---|
| GET | `/api/providers` | não | — | `200 [{"id", "label"}]` |
| POST | `/api/providers` | sim | `{"id", "label"}` | `201` (cria ou sobrescreve label; 400 se `id` vazio) |
| DELETE | `/api/providers/:id` | sim | — | `200` / `400` (protegido) / `404` (inexistente) |
| DELETE | `/api/providers` | sim | — | `200` (sempre sucesso, mesmo sem customizados a apagar) |

`GET /api/content` e `POST /api/content/set/:providerId` (conteúdo)
continuam como estão — não fazem parte deste escopo além de agora também
persistirem.

## Erros e edge cases

- `POST /api/providers` com `id` de um provider protegido: sobrescreve o
  `label` normalmente (overwrite permitido para todos, só o `Delete` tem a
  proteção).
- `DELETE /api/providers/:id` com `id` protegido → `400` com mensagem
  (`"provider %s é fixo e não pode ser apagado"`).
- `DELETE /api/providers/:id` com `id` inexistente → `404`.
- Todos os endpoints de escrita (`POST`/`DELETE`) passam por
  `app.AuthMiddleware`, igual aos demais endpoints de administração
  (`/api/content/set`, `/api/media`, etc.). `GET /api/providers` fica
  público, igual a `GET /api/content`.

## Testes

- `providers/providers_test.go`: `Create` (novo, overwrite de custom,
  overwrite de protegido preserva o conteúdo), `Delete` (protegido → erro,
  inexistente → erro, custom → remove), `DeleteAll` (preserva os 4
  protegidos), `List` (ordenado, inclui label), round-trip de persistência
  (criar Store, mutar, criar um `NewStore` novo apontando pro mesmo
  arquivo de teste, conferir que o estado bate).
- Novo `handlers_providers_test.go` (raiz): os 4 endpoints via
  `httptest`, incluindo `401` nos de escrita sem credenciais e os cenários
  de erro (`400`/`404`) acima.

## Migração

Sem migração de dados existente — não há arquivo `providers.json` hoje.
No primeiro boot após o deploy, o Store detecta a ausência do arquivo e
cria só os 4 protegidos, como um boot limpo.
