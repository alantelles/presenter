package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
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
