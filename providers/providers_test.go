package providers

import (
	"errors"
	"os"
	"testing"
)

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

func TestNewStoreDefaultsToFourProtectedProviders(t *testing.T) {
	withTempDir(t)
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

func TestStoreGetUnknownChannelReturnsZeroValue(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	got := store.Get("nao-existe")
	if got != (Data{}) {
		t.Errorf("got %+v, want zero value", got)
	}
}

func TestStoreSetKnownChannel(t *testing.T) {
	withTempDir(t)
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

func TestStoreSetUnknownChannel(t *testing.T) {
	withTempDir(t)
	store := NewStore()
	if err := store.Set("nao-existe", Data{}); err == nil {
		t.Fatal("expected an error for an unknown channel")
	}
}

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
