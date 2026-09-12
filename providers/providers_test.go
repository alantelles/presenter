package providers

import "testing"

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
	if got := store.Get("main"); got != content {
		t.Errorf("got %+v, want %+v", got, content)
	}
}

func TestStoreSetUnknownChannel(t *testing.T) {
	store := NewStore()
	if err := store.Set("nao-existe", Data{}); err == nil {
		t.Fatal("expected an error for an unknown channel")
	}
}
