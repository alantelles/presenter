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
