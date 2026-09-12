package providers

import "testing"

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

func TestStoreGetUnknownChannelReturnsZeroValue(t *testing.T) {
	store := NewStore()
	got := store.Get("nao-existe")
	if got != (Data{}) {
		t.Errorf("got %+v, want zero value", got)
	}
}

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

func TestStoreSetUnknownChannel(t *testing.T) {
	store := NewStore()
	if err := store.Set("nao-existe", Data{}); err == nil {
		t.Fatal("expected an error for an unknown channel")
	}
}
