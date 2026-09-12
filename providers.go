package main

import "fmt"

func (a *App) CopyIncomingProviderToExistent(providerId string, newContent ProviderData) error {
	if err := a.Providers.Set(providerId, newContent); err != nil {
		return err
	}
	fmt.Printf("Provider %s of type %s updated with new content\n", providerId, newContent.Type)
	return nil
}
