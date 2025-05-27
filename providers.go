package main

import "fmt"

var providers = map[string]ProviderData{
	"main":           {},
	"preview":        {},
	"internal":       {},
	"command":        {},
	"operator":       {},
	"sound-engineer": {},
	"alerts":         {},
}

func CopyIncomingProviderToExistent(providerId string, newContent ProviderData) error {
	_, ok := providers[providerId]
	if !ok {
		return fmt.Errorf("provider with id %s not found", providerId)
	}
	providers[providerId] = newContent
	fmt.Printf("Provider %s of type %s updated with new content\n", providerId, newContent.Type)
	return nil
}
