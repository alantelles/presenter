package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newProviderTestRouter(t *testing.T) (*gin.Engine, *App) {
	t.Helper()
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	app := newTestApp()
	router := gin.New()
	registerProviderRoutes(router, app)
	return router, app
}

func TestListProviders(t *testing.T) {
	router, _ := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	for _, want := range []string{`"id":"main"`, `"label":"Conteúdo principal"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("body %q does not contain %q", w.Body.String(), want)
		}
	}
}

func TestCreateProviderRequiresAuth(t *testing.T) {
	router, _ := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/providers", strings.NewReader(`{"id":"telao-2","label":"Telão"}`))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateProvider(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/providers", strings.NewReader(`{"id":"telao-2","label":"Telão da entrada"}`))
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if got := app.Providers.Get("telao-2").Label; got != "Telão da entrada" {
		t.Errorf("provider label = %q, want %q", got, "Telão da entrada")
	}
}

func TestDeleteProtectedProviderReturnsBadRequest(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/main", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteUnknownProviderReturnsNotFound(t *testing.T) {
	router, app := newProviderTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/nao-existe", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteCustomProvider(t *testing.T) {
	router, app := newProviderTestRouter(t)
	if err := app.Providers.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/telao-2", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteAllProvidersKeepsProtected(t *testing.T) {
	router, app := newProviderTestRouter(t)
	if err := app.Providers.Create("telao-2", "Telão"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/providers", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := app.Providers.Get("telao-2"); got != (ProviderData{}) {
		t.Errorf("custom provider should be gone, got %+v", got)
	}
	if got := app.Providers.Get("main").Label; got == "" {
		t.Error("protected provider should survive DeleteAll")
	}
}
