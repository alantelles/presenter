package main

import (
	"errors"
	"net/http"
	"presenter/providers"

	"github.com/gin-gonic/gin"
)

type providerCreateRequest struct {
	ID    string `json:"id" binding:"required"`
	Label string `json:"label"`
}

// listProviders returns every registered provider's id and label.
func (a *App) listProviders(c *gin.Context) {
	c.JSON(http.StatusOK, a.Providers.List())
}

// createProvider registers a new provider, or overwrites the label of an
// existing one while preserving its current content.
func (a *App) createProvider(c *gin.Context) {
	var req providerCreateRequest
	if err := c.BindJSON(&req); err != nil {
		return
	}
	if err := a.Providers.Create(req.ID, req.Label); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "label": req.Label})
}

// deleteProvider removes a single provider. Protected providers get a 400,
// unknown ids get a 404.
func (a *App) deleteProvider(c *gin.Context) {
	id := c.Param("id")
	err := a.Providers.Delete(id)
	switch {
	case err == nil:
		c.Status(http.StatusOK)
	case errors.Is(err, providers.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// deleteAllProviders removes every custom provider, keeping the protected
// ones intact.
func (a *App) deleteAllProviders(c *gin.Context) {
	if err := a.Providers.DeleteAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
