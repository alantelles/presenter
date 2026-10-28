# Providers Dinâmicos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let providers (content channels) be created and deleted at runtime via HTTP API, persisted to disk, while keeping `main`/`preview`/`aux`/`command` as the only providers protected from deletion.

**Architecture:** Extend `presenter/providers` (`Store`) with a `Label` field on `Data`, a fixed protected-ID set, and JSON persistence to a single file (`providers.json`) rewritten on every mutation. Add four new HTTP endpoints in the root module (`main` package) that call the new `Store` methods (`List`/`Create`/`Delete`/`DeleteAll`), reusing the existing `AuthMiddleware` for the write endpoints.

**Tech Stack:** Go 1.23/1.24, Gin, standard library (`encoding/json`, `sync`, `os`). No new dependencies.

**Spec:** [docs/superpowers/specs/2026-09-12-providers-dinamicos-design.md](../specs/2026-09-12-providers-dinamicos-design.md)

## Global Constraints

- No database — persistence is a single JSON file.
- Persistence path is fixed: `./providers.json`, relative to the process's working directory, independent of the `-pastaMedia` flag.
- Exactly 4 protected providers, never deletable: `main` ("Conteúdo principal"), `preview` ("Prévia conteúdo"), `aux` ("Visão auxiliar"), `command` ("Linha de comandos"). All other current defaults (`internal`, `operator`, `sound-engineer`, `alerts`) are removed from the default boot set.
- `POST` (create/overwrite) is allowed on any id, including protected ones (only overwrites the label, preserves existing content). Only `DELETE` is blocked for protected ids.
- Creating a provider with an existing id overwrites its label and preserves its current content; creating a new id starts with empty content.
- All write endpoints (`POST /api/providers`, `DELETE /api/providers/:id`, `DELETE /api/providers`) require Basic Auth via the existing `AuthMiddleware`. `GET /api/providers` stays public, like `GET /api/content`.
- If `providers.json` exists but fails to parse at boot, the process must fail fast (fatal), never silently discard it.

---

## File Structure

- **Modify `providers/providers.go`** — add `Label` to `Data`, protected-ID set, `sync.Mutex`, JSON persistence (`save`/`loadChannels`), and new methods `List`, `Create`, `Delete`, `DeleteAll`.
- **Modify `providers/providers_test.go`** — update existing tests for the new default set/label-preserving `Set`, add tests for persistence, `List`, `Create`, `Delete`, `DeleteAll`.
- **Modify `app_test.go`** — add a `withTempWorkDir` helper and use it wherever tests build an `App` (which now touches disk via `providers.NewStore()`).
- **Modify `routes_test.go`** — use `withTempWorkDir` in the existing tests that build an `App`; add `TestRegisterProviderRoutes`.
- **Modify `routes.go`** — add `registerProviderRoutes`.
- **Modify `main.go`** — call `registerProviderRoutes` in `main()`.
- **Create `handlers_providers.go`** — the 4 new Gin handlers (`listProviders`, `createProvider`, `deleteProvider`, `deleteAllProviders`).
- **Create `handlers_providers_test.go`** — HTTP-level tests for the 4 endpoints, including auth checks.

---

### Task 1: `Data.Label` field and the 4 protected providers

**Files:**
- Modify: `providers/providers.go`
- Test: `providers/providers_test.go`

**Interfaces:**
- Produces: `Data.Label string`; `NewStore()` now creates exactly `main`, `preview`, `aux`, `command` (each with its label), and no longer creates `internal`, `operator`, `sound-engineer`, `alerts`. `Set` preserves the existing provider's `Label` when replacing its content.

- [ ] **Step 1: Write the failing test**

Add to `providers/providers_test.go`:

```go
func TestNewStoreDefaultsToFourProtectedProviders(t *testing.T) {
	store := NewStore()

	want := map[string]string{
		"main":    "Conteúdo principal",
		"preview": "Prévia conteúdo",
		"aux":     "Visão auxiliar",
		"command": "Linha de comandos",
	}
	for id, label := range want {
		if got := store.Get(id); got.Label != label {
			t.Errorf("Get(%q).Label = %q, want %q", id, got.Label, label)
		}
	}

	for _, id := range []string{"internal", "operator", "sound-engineer", "alerts"} {
		if got := store.Get(id); got != (Data{}) {
			t.Errorf("Get(%q) = %+v, want zero value (should not exist by default)", id, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd providers && go test ./... -run TestNewStoreDefaultsToFourProtectedProviders -v`
Expected: FAIL (current `NewStore` still creates the old 8-channel set with no labels)

- [ ] **Step 3: Implement**

Replace the top of `providers/providers.go` (keep the package doc comment) with:

```go
package providers

import "fmt"

// Data is the content held by a single provider channel.
type Data struct {
	Label     string `json:"label,omitempty"`
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	ContentID string `json:"contentId,omitempty"`
}

// protectedIDs are the providers that always exist by default and can never
// be deleted, keyed by id with their display label.
var protectedIDs = map[string]string{
	"main":    "Conteúdo principal",
	"preview": "Prévia conteúdo",
	"aux":     "Visão auxiliar",
	"command": "Linha de comandos",
}

// Store is the set of provider channels and their current content.
type Store struct {
	channels map[string]Data
}

func defaultChannels() map[string]Data {
	channels := make(map[string]Data, len(protectedIDs))
	for id, label := range protectedIDs {
		channels[id] = Data{Label: label}
	}
	return channels
}

// NewStore creates a Store with the default set of protected provider
// channels (main/preview/aux/command), all starting empty.
func NewStore() *Store {
	return &Store{channels: defaultChannels()}
}

// Get returns the current content of a provider channel. It returns the zero
// value if the channel doesn't exist.
func (s *Store) Get(providerId string) Data {
	return s.channels[providerId]
}

// Set replaces the content of an existing provider channel, preserving its
// label. It returns an error if providerId isn't one of the known channels.
func (s *Store) Set(providerId string, content Data) error {
	existing, ok := s.channels[providerId]
	if !ok {
		return fmt.Errorf("provider with id %s not found", providerId)
	}
	content.Label = existing.Label
	s.channels[providerId] = content
	return nil
}
```

- [ ] **Step 4: Update the existing test that now breaks**

`TestStoreSetKnownChannel` in `providers/providers_test.go` currently expects
`Get("main")` to equal the content it just `Set`, but `Set` now preserves the
existing label (`"Conteúdo principal"`). Update it to:

```go
func TestStoreSetKnownChannel(t *testing.T) {
	store := NewStore()
	content := Data{Content: "ola", Type: "TEXT"}

	if err := store.Set("main", content); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := content
	want.Label = "Conteúdo principal"
	if got := store.Get("main"); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 5: Run all provider tests to verify they pass**

Run: `cd providers && go test ./... -v`
Expected: PASS (all tests, including `TestStoreGetUnknownChannelReturnsZeroValue` and `TestStoreSetUnknownChannel`, which are unaffected)

- [ ] **Step 6: Commit**

```bash
git add providers/providers.go providers/providers_test.go
git commit -m "feat(providers): reduce default set to 4 protected providers with labels"
```

---

### Task 2: JSON persistence and concurrency safety

**Files:**
- Modify: `providers/providers.go`
- Test: `providers/providers_test.go`
- Modify: `app_test.go`
- Modify: `routes_test.go`

**Interfaces:**
- Consumes: `Data`, `protectedIDs`, `defaultChannels()` from Task 1.
- Produces: `NewStore()` now loads `./providers.json` if present, otherwise creates the defaults and persists them; `Store` gains a `sync.Mutex`; `Set` now persists on every call. Any test that builds an `App` (and therefore a `Store`) must run inside an isolated working directory from now on.

- [ ] **Step 1: Write the failing test**

Add to `providers/providers_test.go` (needs `"os"` imported):

```go
func withTempDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
}

func TestNewStorePersistsAcrossRestarts(t *testing.T) {
	withTempDir(t)

	store := NewStore()
	if err := store.Set("main", Data{Content: "ola"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	restarted := NewStore()
	got := restarted.Get("main")
	want := Data{Label: "Conteúdo principal", Content: "ola"}
	if got != want {
		t.Errorf("after restart, Get(\"main\") = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd providers && go test ./... -run TestNewStorePersistsAcrossRestarts -v`
Expected: FAIL (nothing is persisted yet, so `restarted.Get("main")` has no content)

- [ ] **Step 3: Implement persistence and locking**

Replace `providers/providers.go` with:

```go
package providers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// Data is the content held by a single provider channel.
type Data struct {
	Label     string `json:"label,omitempty"`
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	ContentID string `json:"contentId,omitempty"`
}

// protectedIDs are the providers that always exist by default and can never
// be deleted, keyed by id with their display label.
var protectedIDs = map[string]string{
	"main":    "Conteúdo principal",
	"preview": "Prévia conteúdo",
	"aux":     "Visão auxiliar",
	"command": "Linha de comandos",
}

// storeFile is the fixed path where the Store persists its state, relative
// to the process's working directory. It's independent of the -pastaMedia
// flag, which only configures where song/image media lives.
const storeFile = "providers.json"

// Store is the set of provider channels and their current content,
// persisted to storeFile on every mutation.
type Store struct {
	mu       sync.Mutex
	channels map[string]Data
}

func defaultChannels() map[string]Data {
	channels := make(map[string]Data, len(protectedIDs))
	for id, label := range protectedIDs {
		channels[id] = Data{Label: label}
	}
	return channels
}

// loadChannels reads storeFile from disk. ok is false if the file doesn't
// exist (a fresh boot). A file that exists but isn't valid JSON is fatal —
// silently discarding it would lose data.
func loadChannels() (channels map[string]Data, ok bool) {
	raw, err := os.ReadFile(storeFile)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		log.Fatalf("providers: failed to read %s: %v", storeFile, err)
	}
	if err := json.Unmarshal(raw, &channels); err != nil {
		log.Fatalf("providers: %s exists but is not valid JSON: %v", storeFile, err)
	}
	return channels, true
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.channels, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storeFile, raw, 0644)
}

// NewStore loads providers from storeFile if it exists; otherwise it creates
// the protected set (main/preview/aux/command) and persists it.
func NewStore() *Store {
	if channels, ok := loadChannels(); ok {
		return &Store{channels: channels}
	}
	s := &Store{channels: defaultChannels()}
	if err := s.save(); err != nil {
		log.Printf("providers: failed to persist initial state: %v", err)
	}
	return s
}

// Get returns the current content of a provider channel. It returns the zero
// value if the channel doesn't exist.
func (s *Store) Get(providerId string) Data {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.channels[providerId]
}

// Set replaces the content of an existing provider channel, preserving its
// label, and persists the change. It returns an error if providerId isn't
// one of the known channels.
func (s *Store) Set(providerId string, content Data) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.channels[providerId]
	if !ok {
		return fmt.Errorf("provider with id %s not found", providerId)
	}
	content.Label = existing.Label
	s.channels[providerId] = content
	return s.save()
}
```

- [ ] **Step 4: Add `withTempDir(t)` to every existing provider test**

Every test in `providers/providers_test.go` now triggers disk I/O through
`NewStore()`. Add `withTempDir(t)` as the first line of
`TestStoreGetUnknownChannelReturnsZeroValue`, `TestStoreSetKnownChannel`,
`TestStoreSetUnknownChannel`, and `TestNewStoreDefaultsToFourProtectedProviders`
(from Task 1) — each should isolate itself into its own temp directory before
calling `NewStore()`.

- [ ] **Step 5: Run all provider tests to verify they pass**

Run: `cd providers && go test ./... -v`
Expected: PASS

- [ ] **Step 6: Isolate root-module tests that build an `App`**

`NewApp` calls `providers.NewStore()`, which now writes `providers.json` to
the process's working directory. Root-package tests that build an `App` must
no longer run with the repo directory as their cwd. Add to `app_test.go`
(needs `"os"` imported):

```go
func withTempWorkDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
}
```

Add `withTempWorkDir(t)` as the first line of `TestAppAuthMiddleware` and
`TestAppCopyIncomingProviderToExistent` (both call `newTestApp()`).

In `routes_test.go`, add `withTempWorkDir(t)` as the first line of
`TestRegisterMediaRoutes` and `TestRegisterViewRoutes` (both call
`NewApp(Config{})`).

- [ ] **Step 7: Run the full root-module test suite to verify it passes**

Run: `go test ./... -v`
Expected: PASS, and no `providers.json` left behind in the repo root (check
with `git status --short` — should be clean)

- [ ] **Step 8: Commit**

```bash
git add providers/providers.go providers/providers_test.go app_test.go routes_test.go
git commit -m "feat(providers): persist Store to providers.json, guard with a mutex"
```

---

### Task 3: `List` and `Create`

**Files:**
- Modify: `providers/providers.go`
- Test: `providers/providers_test.go`

**Interfaces:**
- Consumes: `Store`, `Data`, `protectedIDs`, `save()` from Task 2.
- Produces: `type ProviderInfo struct { ID, Label string }`; `func (s *Store) List() []ProviderInfo` (sorted by ID); `func (s *Store) Create(id, label string) error` (creates or overwrites the label, preserves existing content, persists, errors on empty id).

- [ ] **Step 1: Write the failing tests**

Add to `providers/providers_test.go` (needs `"sort"` not required here, only in implementation):

```go
func TestStoreList(t *testing.T) {
	withTempDir(t)
	store := NewStore()

	got := store.List()
	want := []ProviderInfo{
		{ID: "aux", Label: "Visão auxiliar"},
		{ID: "command", Label: "Linha de comandos"},
		{ID: "main", Label: "Conteúdo principal"},
		{ID: "preview", Label: "Prévia conteúdo"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d providers, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStoreCreateNewProvider(t *testing.T) {
	withTempDir(t)
	store := NewStore()

	if err := store.Create("telao-2", "Telão da entrada"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Data{Label: "Telão da entrada"}
	if got := store.Get("telao-2"); got != want {
		t.Errorf("Get(\"telao-2\") = %+v, want %+v", got, want)
	}
}

func TestStoreCreateOverwritesLabelPreservingContent(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	if err := store.Set("main", Data{Content: "ola"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Create("main", "Novo nome"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Data{Label: "Novo nome", Content: "ola"}
	if got := store.Get("main"); got != want {
		t.Errorf("Get(\"main\") = %+v, want %+v", got, want)
	}
}

func TestStoreCreateRejectsEmptyID(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	if err := store.Create("", "label"); err == nil {
		t.Fatal("expected an error for an empty id")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd providers && go test ./... -run 'TestStoreList|TestStoreCreate' -v`
Expected: FAIL with "undefined: ProviderInfo" / "s.Create undefined"

- [ ] **Step 3: Implement**

Add to `providers/providers.go` (add `"sort"` to the import block):

```go
// ProviderInfo is the public listing shape for a provider: its id plus
// display label, without the current content.
type ProviderInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// List returns every registered provider's id and label, sorted by id.
func (s *Store) List() []ProviderInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos := make([]ProviderInfo, 0, len(s.channels))
	for id, data := range s.channels {
		infos = append(infos, ProviderInfo{ID: id, Label: data.Label})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })
	return infos
}

// Create registers a provider under id with the given label, persisting the
// change. If id already exists (protected or custom), only its label is
// overwritten and its current content is preserved; otherwise a new
// provider is created with empty content.
func (s *Store) Create(id, label string) error {
	if id == "" {
		return fmt.Errorf("provider id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.channels[id]
	existing.Label = label
	s.channels[id] = existing
	return s.save()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd providers && go test ./... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add providers/providers.go providers/providers_test.go
git commit -m "feat(providers): add List and Create"
```

---

### Task 4: `Delete` and `DeleteAll`

**Files:**
- Modify: `providers/providers.go`
- Test: `providers/providers_test.go`

**Interfaces:**
- Consumes: `Store`, `protectedIDs`, `save()` from earlier tasks.
- Produces: `var ErrProtected, ErrNotFound error` (exported, for callers to distinguish via `errors.Is`); `func (s *Store) Delete(id string) error`; `func (s *Store) DeleteAll() error`.

- [ ] **Step 1: Write the failing tests**

Add to `providers/providers_test.go` (needs `"errors"` imported):

```go
func TestStoreDeleteProtectedProviderFails(t *testing.T) {
	withTempDir(t)
	store := NewStore()

	err := store.Delete("main")
	if !errors.Is(err, ErrProtected) {
		t.Fatalf("Delete(\"main\") error = %v, want ErrProtected", err)
	}
	if got := store.Get("main"); got.Label == "" {
		t.Error("protected provider should still exist after failed delete")
	}
}

func TestStoreDeleteUnknownProviderFails(t *testing.T) {
	withTempDir(t)
	store := NewStore()

	err := store.Delete("nao-existe")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete(\"nao-existe\") error = %v, want ErrNotFound", err)
	}
}

func TestStoreDeleteCustomProvider(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	if err := store.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete("telao-2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.Get("telao-2"); got != (Data{}) {
		t.Errorf("Get(\"telao-2\") = %+v, want zero value after delete", got)
	}
}

func TestStoreDeleteAllKeepsProtected(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	if err := store.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.DeleteAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.Get("telao-2"); got != (Data{}) {
		t.Errorf("custom provider should be gone, got %+v", got)
	}
	if got := store.Get("main"); got.Label == "" {
		t.Error("protected provider should survive DeleteAll")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd providers && go test ./... -run 'TestStoreDelete' -v`
Expected: FAIL with "undefined: ErrProtected" / "s.Delete undefined"

- [ ] **Step 3: Implement**

Add `"errors"` to the import block in `providers/providers.go` and add:

```go
var (
	// ErrProtected is returned by Delete when id is one of the providers
	// that can never be removed.
	ErrProtected = errors.New("provider is protected and cannot be deleted")
	// ErrNotFound is returned by Delete when id doesn't exist.
	ErrNotFound = errors.New("provider not found")
)

// Delete removes a provider and persists the change. It returns
// ErrProtected if id is one of the protected providers, or ErrNotFound if
// id doesn't exist.
func (s *Store) Delete(id string) error {
	if _, protected := protectedIDs[id]; protected {
		return fmt.Errorf("%s: %w", id, ErrProtected)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[id]; !ok {
		return fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	delete(s.channels, id)
	return s.save()
}

// DeleteAll removes every custom provider, keeping the protected ones, and
// persists the change.
func (s *Store) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.channels {
		if _, protected := protectedIDs[id]; !protected {
			delete(s.channels, id)
		}
	}
	return s.save()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd providers && go test ./... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add providers/providers.go providers/providers_test.go
git commit -m "feat(providers): add Delete and DeleteAll"
```

---

### Task 5: HTTP endpoints and route registration

**Files:**
- Create: `handlers_providers.go`
- Create: `handlers_providers_test.go`
- Modify: `routes.go`
- Modify: `routes_test.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `a.Providers.List() []providers.ProviderInfo`, `a.Providers.Create(id, label string) error`, `a.Providers.Delete(id string) error`, `a.Providers.DeleteAll() error`, `providers.ErrNotFound` (all from Tasks 3–4); `app.AuthMiddleware` (existing, `app.go`); `withTempWorkDir(t)` (existing, `app_test.go`, Task 2); `newTestApp()` (existing, `app_test.go`).
- Produces: `GET /api/providers`, `POST /api/providers`, `DELETE /api/providers/:id`, `DELETE /api/providers` — wired via `registerProviderRoutes(router *gin.Engine, app *App)`.

- [ ] **Step 1: Write the failing HTTP tests**

Create `handlers_providers_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newProviderTestRouter(t *testing.T) (*gin.Engine, *App) {
	t.Helper()
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	app := newTestApp()
	router := gin.New()
	registerProviderRoutes(router, app)
	return router, app
}

func TestListProviders(t *testing.T) {
	router, _ := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	for _, want := range []string{`"id":"main"`, `"label":"Conteúdo principal"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("body %q does not contain %q", w.Body.String(), want)
		}
	}
}

func TestCreateProviderRequiresAuth(t *testing.T) {
	router, _ := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/providers", strings.NewReader(`{"id":"telao-2","label":"Telão"}`))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateProvider(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/providers", strings.NewReader(`{"id":"telao-2","label":"Telão da entrada"}`))
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if got := app.Providers.Get("telao-2").Label; got != "Telão da entrada" {
		t.Errorf("provider label = %q, want %q", got, "Telão da entrada")
	}
}

func TestDeleteProtectedProviderReturnsBadRequest(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/main", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteUnknownProviderReturnsNotFound(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/nao-existe", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteCustomProvider(t *testing.T) {
	router, app := newProviderTestRouter(t)
	if err := app.Providers.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/telao-2", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteAllProvidersKeepsProtected(t *testing.T) {
	router, app := newProviderTestRouter(t)
	if err := app.Providers.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := app.Providers.Get("telao-2"); got != (ProviderData{}) {
		t.Errorf("custom provider should be gone, got %+v", got)
	}
	if got := app.Providers.Get("main").Label; got == "" {
		t.Error("protected provider should survive DeleteAll")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run 'TestListProviders|TestCreateProvider|TestDeleteProtectedProvider|TestDeleteUnknownProvider|TestDeleteCustomProvider|TestDeleteAllProviders' -v`
Expected: FAIL to compile — "undefined: registerProviderRoutes"

- [ ] **Step 3: Implement the handlers**

Create `handlers_providers.go`:

```go
package main

import (
	"errors"
	"net/http"
	"presenter/providers"

	"github.com/gin-gonic/gin"
)

type providerCreateRequest struct {
	ID    string `json:"id" binding:"required"`
	Label string `json:"label"`
}

// listProviders returns every registered provider's id and label.
func (a *App) listProviders(c *gin.Context) {
	c.JSON(http.StatusOK, a.Providers.List())
}

// createProvider registers a new provider, or overwrites the label of an
// existing one while preserving its current content.
func (a *App) createProvider(c *gin.Context) {
	var req providerCreateRequest
	if err := c.BindJSON(&req); err != nil {
		return
	}
	if err := a.Providers.Create(req.ID, req.Label); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "label": req.Label})
}

// deleteProvider removes a single provider. Protected providers get a 400,
// unknown ids get a 404.
func (a *App) deleteProvider(c *gin.Context) {
	id := c.Param("id")
	err := a.Providers.Delete(id)
	switch {
	case err == nil:
		c.Status(http.StatusOK)
	case errors.Is(err, providers.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// deleteAllProviders removes every custom provider, keeping the protected
// ones intact.
func (a *App) deleteAllProviders(c *gin.Context) {
	if err := a.Providers.DeleteAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
```

Add to `routes.go`:

```go
// registerProviderRoutes wires up provider management: creating, listing
// and deleting the channels controllers publish content to.
func registerProviderRoutes(router *gin.Engine, app *App) {
	router.GET("/api/providers", app.listProviders)
	router.POST("/api/providers", app.AuthMiddleware, app.createProvider)
	router.DELETE("/api/providers/:id", app.AuthMiddleware, app.deleteProvider)
	router.DELETE("/api/providers", app.AuthMiddleware, app.deleteAllProviders)
}
```

In `main.go`, add the call alongside the other `register*Routes` calls in
`main()`:

```go
	registerMediaRoutes(router, app)
	registerProviderRoutes(router, app)
	registerViewRoutes(router, app)
```

- [ ] **Step 4: Run the new tests to verify they pass**

Run: `go test ./... -run 'TestListProviders|TestCreateProvider|TestDeleteProtectedProvider|TestDeleteUnknownProvider|TestDeleteCustomProvider|TestDeleteAllProviders' -v`
Expected: PASS

- [ ] **Step 5: Add the route-registration test**

Add to `routes_test.go`:

```go
func TestRegisterProviderRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerProviderRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"GET", "/api/providers"},
		{"POST", "/api/providers"},
		{"DELETE", "/api/providers/:id"},
		{"DELETE", "/api/providers"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}
```

- [ ] **Step 6: Run the full test suite**

Run: `go test ./... -v && go vet ./...`
Expected: everything PASS, no vet warnings

Also run the submodule's own tests (per this repo's multi-module setup):

Run: `cd providers && go test ./... -v`
Expected: PASS

- [ ] **Step 7: Manual smoke check**

Run: `go build -v ./... && ./presenter -usuario admin -senha admin`

In another terminal:

```bash
curl http://localhost:8080/api/providers
curl -u admin:admin -X POST http://localhost:8080/api/providers -d '{"id":"telao-2","label":"Telão da entrada"}'
curl http://localhost:8080/api/providers
curl -u admin:admin -X DELETE http://localhost:8080/api/providers/main
curl -u admin:admin -X DELETE http://localhost:8080/api/providers/telao-2
cat providers.json
```

Confirm: the first `curl` lists the 4 protected providers; after the `POST`,
the second `curl` also lists `telao-2`; the `DELETE` on `main` returns an
error (400); the `DELETE` on `telao-2` succeeds; `providers.json` reflects
the final state. Stop the server (Ctrl+C) and delete the local
`providers.json` this smoke test created before committing (it's a runtime
artifact, not part of the repo — check `git status --short`).

- [ ] **Step 8: Commit**

```bash
git add handlers_providers.go handlers_providers_test.go routes.go routes_test.go main.go
git commit -m "feat(providers): add HTTP endpoints to create/list/delete providers"
```

---

## Post-implementation

- Update `TODO.md`: move "Providers dinâmicos" from "Pendente" to "Feito", with a short note pointing at this plan/spec.
- Update `CLAUDE.md`'s "Providers" section: it currently says providers are "hoje hardcoded — não configurável em runtime" — update to describe the new `List`/`Create`/`Delete`/`DeleteAll` API, the 4 protected providers, and the `providers.json` persistence file.
- Add `providers.json` to `.gitignore` (it's a runtime data file, like the `media/` folder).
