// Package providers holds the in-memory provider content channels
// (main, preview, aux, command, etc.) that controllers publish to and the
// live panel/other consumers read from.
package providers

import "fmt"

// Data is the content held by a single provider channel.
type Data struct {
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	ContentID string `json:"contentId,omitempty"`
}

// Store is the set of fixed provider channels and their current content.
type Store struct {
	channels map[string]Data
}

// NewStore creates a Store with the default set of provider channels, all
// starting empty.
func NewStore() *Store {
	return &Store{
		channels: map[string]Data{
			"main":           {},
			"preview":        {},
			"aux":            {},
			"internal":       {},
			"command":        {},
			"operator":       {},
			"sound-engineer": {},
			"alerts":         {},
		},
	}
}

// Get returns the current content of a provider channel. It returns the zero
// value if the channel doesn't exist.
func (s *Store) Get(providerId string) Data {
	return s.channels[providerId]
}

// Set replaces the content of an existing provider channel. It returns an
// error if providerId isn't one of the known channels.
func (s *Store) Set(providerId string, content Data) error {
	if _, ok := s.channels[providerId]; !ok {
		return fmt.Errorf("provider with id %s not found", providerId)
	}
	s.channels[providerId] = content
	return nil
}
