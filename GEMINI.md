# GEMINI.md — Guia de Arquitetura, Achados e Utilidades do Presenter

Documento de referência para agentes e desenvolvedores sobre a arquitetura do Presenter e a funcionalidade de exibição e controle de vídeos.

---

## 1. Visão Geral da Aplicação

O **Presenter** é um servidor de apresentação de conteúdo em tempo real voltado para transmissões multi-stream e multi-output (letras de música, versículos bíblicos, imagens e vídeos).

- **Backend**: Go 1.24, framework Gin, arquitetura multi-módulo (`replace` em `go.mod`).
- **Frontend**: HTML5 puro + Vanilla ES Modules (`templates/` e `static/js/`).
- **Resolução de Rotas e Base Path**:
  - Em vez de interpolar `{{APP_LOCATION}}` (que engessa o hostname configurado nas flags), o frontend utiliza detecção dinâmica do prefixo do caminho:
    ```javascript
    const isPrefixed = window.location.pathname.startsWith('/presenter');
    const APP_BASE_PATH = isPrefixed ? '/presenter' : '';
    function withBasePath(path) { return APP_BASE_PATH + path; }
    ```
    Isso permite que a aplicação rode perfeitamente tanto na raiz `/` quanto sob proxies reversos com subcaminho `/presenter`.
- **Comunicação Telão / Controller**:
  - Operadores publicam conteúdo em canais via `POST /api/content/set/:providerId`.
  - Telões (`/live`) consultam via polling HTTP a cada 1 segundo em `GET /api/content?providerId=...`.
  - Comandos em tempo real (ex.: cronômetro, controle de vídeo) trafegam pelo provider protegido `command`.

---

## 2. Exibição de Vídeos no Telão (`/live`)

### Tela Cheia e Renderização
- O elemento `<video id="live-media-video" playsinline>` ocupa toda a tela (`position: fixed; width: 100vw; height: 100vh; object-fit: contain; background: black; z-index: 99995;`).
- Ao receber `data.type === 'VIDEO'`, o `/live` oculta o vídeo de fundo decorativo (`#bgvideo`), limpa textos e capítulos e inicia a reprodução do vídeo apontando para `/api/videos/content?name=...`.
- Ao alternar para texto, imagem ou limpar tela (`content: ''`), o vídeo é pausado e descarregado automaticamente.

### Controle de Som via QueryString
Para evitar múltiplos painéis tocando áudio simultaneamente em configurações multi-output:
- **Telão principal (com som)**: `http://localhost:8080/live`
- **Telões auxiliares / streaming (sem som)**: `http://localhost:8080/live?muted=true`
  - Parâmetros aceitos para silenciar: `muted=true`, `mute=true`, `audio=false`, `som=false`, `semSom=true`.
- **Política de Autoplay de Navegadores**: Caso o navegador bloqueie o autoplay com áudio desmutado sem interação prévia do usuário, o `/live` automaticamente inicia o vídeo mutado e exibe uma notificação discreta (`#audio-unlock-banner`), ativando o som no primeiro clique em qualquer lugar da tela.

### Comandos de Controle Remoto
O `/live` processa comandos enviados pelo provider `command` (`data.contentId` no formato `<comando>-<timestamp>`):
- `videoRestart` / `restartVideo` / `videoInicio`: Reinicia o vídeo para o início (0:00) e retoma a reprodução.
- `videoPlay` / `playVideo`: Inicia ou retoma a reprodução (reinicia do zero se o vídeo tiver terminado).
- `videoPause` / `pauseVideo`: Pausa a reprodução.
- `videoForward` / `videoAvancar`: Avança N segundos (padrão 10s se omitido).
- `videoRewind` / `videoVoltar`: Retrocede N segundos (padrão 10s se omitido).
- `videoSeek`: Posiciona em segundo específico via `data.content`.

---

## 3. Submódulo de Vídeos (`presenter/videos`)

Localizado em `videos/` com seu próprio `go.mod`:
- **Diretório em disco**: `<pastaMedia>/videos/` (default: `media/videos/`).
- **Limite de upload**: 500 MB (`videos.MaxUploadSize`).
- **Validação de formato**: Content-sniffing dos primeiros bytes (MP4 box `ftyp`/`moov`, WebM `\x1a\x45\xdf\xa3`, OGG `OggS`) com suporte a extensões `.mp4`, `.webm`, `.mov`, `.ogg`, `.ogv`, `.m4v`.
- **Gravação em disco eficiente**: Gravação via streaming temporário (`.tmp`) evitando carregar o arquivo inteiro na memória RAM.
- **Suporte a Range Requests**: O endpoint `GET /api/videos/content?name=...` utiliza `c.File()`, delegando para `http.ServeFile` com suporte nativo a HTTP 206 Partial Content (indispensável para seeking/scrubbing de vídeo em HTML5).

### Rotas HTTP de Vídeo
- `POST /api/videos`: Upload multipart (campo `files`). Requer autenticação Basic Auth.
- `GET /api/videos`: Lista de nomes de arquivos ordenados em JSON `{"videos": [...]}`.
- `GET /api/videos/content?name=...`: Streaming do vídeo.
- `DELETE /api/videos/:name`: Exclui arquivo de vídeo. Requer autenticação Basic Auth.
- `GET /api/videos/downloader/status`: Verifica se `yt-dlp` e `ffmpeg` estão instalados no servidor.
- `POST /api/videos/download`: Baixa vídeo via URL usando `yt-dlp -f "bv*+ba/b" --merge-output-format mp4`. Requer autenticação Basic Auth.

### Script de Instalação de Ferramentas de Vídeo
Localizado na raiz do projeto: `install-video-tools.sh` (executável via `sudo ./install-video-tools.sh`):
- Instala o `ffmpeg` via apt.
- Baixa o binário standalone compilado oficial `yt-dlp_linux` (com runtime Python 3.11 embutido) diretamente para `/usr/local/bin/yt-dlp`.
- Garante total compatibilidade com Ubuntu 20.04, 22.04 e 24.04 sem conflitos de versões de Python.

---

## 4. Gerenciador de Mídia Redesenhado (`/controller/images`)

A tela de gerenciamento de imagens foi modernizada para atuar como o **Gerenciador de Mídia** (Imagens e Vídeos):
- **Painel de Seleção e Preparação**:
  - Clicar em um card **não coloca mais o conteúdo direto no ar**.
  - O card selecionado é destacado com borda azul e badge `✓ SELECIONADO`.
  - Exibe prévia do item selecionado e botão dedicado para colocar no ar (`▶ Dar o Play no Vídeo` ou `🖼️ Exibir Imagem no Telão`).
- **Barra de Controle de Reprodução (No Ar)**:
  - Exibe status em tempo real (`NO AR: <arquivo>` com indicador pulsante ou `Parado`).
  - Botões dedicados na ordem ergonômica padrão:
    `⏮ Do Início`, `⏪ Voltar -10s`, `▶ Reproduzir`, `⏸ Pausar`, `⏩ Avançar +10s`, `⏹ Parar Telão`.
- **Área de Drag & Drop e Upload**:
  - Aceita tanto imagens (até 10MB) quanto vídeos (até 500MB).
  - Identifica o tipo de arquivo automaticamente e encaminha para `/api/images` ou `/api/videos`.
- **Abas de Filtragem e Busca**:
  - Abas: `Todas as Mídias`, `Imagens (N)`, `Vídeos (N)`.
  - Campo de busca instantânea por nome de arquivo.
- **Galeria de Cards Interativos**:
  - Identificação visual clara entre imagens e vídeos (badges coloridos).
  - Pré-visualização do primeiro frame do vídeo usando HTML5 metadata (`#t=0.1`).
  - Destaque com borda verde brilhante no card atualmente em exibição no telão.
  - Ação rápida de exibição ao clicar no card e botão de exclusão de vídeo.
- **Sincronização Contínua**:
  - Polling periódico a cada 2 segundos verifica se outro operador alterou o telão, mantendo a interface sempre atualizada.

---

## 5. Comandos Úteis de Desenvolvimento

Como este ambiente possui o binário do Go em caminho específico:
```bash
# Executar testes do módulo raiz
/home/alantelles/Documents/go/bin/go test -v ./...

# Executar testes de todos os módulos
/home/alantelles/Documents/go/bin/go test ./... && \
(cd videos && /home/alantelles/Documents/go/bin/go test ./...) && \
(cd images && /home/alantelles/Documents/go/bin/go test ./...) && \
(cd storage && /home/alantelles/Documents/go/bin/go test ./...) && \
(cd providers && /home/alantelles/Documents/go/bin/go test ./...)

# Build completo
/home/alantelles/Documents/go/bin/go build -v ./...

# Rodar a aplicação localmente
/home/alantelles/Documents/go/bin/go run .
```
