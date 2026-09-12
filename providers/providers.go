// Package providers holds the in-memory provider content channels
// (main, preview, aux, command, etc.) that controllers publish to and the
// live panel/other consumers read from.
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
