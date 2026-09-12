package main

import "fmt"

func (a *App) CopyIncomingProviderToExistent(providerId string, newContent ProviderData) error {
	_, ok := a.Providers[providerId]
	if !ok {
		return fmt.Errorf("provider with id %s not found", providerId)
	}
	a.Providers[providerId] = newContent
	fmt.Printf("Provider %s of type %s updated with new content\n", providerId, newContent.Type)
	return nil
}
