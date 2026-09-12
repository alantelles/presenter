package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestApp() *App {
	return NewApp(Config{
		Location:      "http://192.168.0.10:8080",
		BasicAuthUser: "admin",
		BasicAuthPass: "secret",
	})
}

func TestAppInsertAddressOnContent(t *testing.T) {
	app := newTestApp()
	got := string(app.insertAddressOnContent([]byte(`fetch('{{APP_LOCATION}}/api/songs')`)))
	want := `fetch('http://192.168.0.10:8080/api/songs')`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAppInsertAuthTokenOnContent(t *testing.T) {
	app := newTestApp()
	got := string(app.insertAuthTokenOnContent([]byte(`token: '{{BASIC_AUTH_TOKEN}}'`)))
	want := "token: '" + app.getAuthAsB64() + "'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if app.getAuthAsB64() == "" {
		t.Error("expected a non-empty base64 auth token")
	}
}

func TestAppAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newTestApp()

	router := gin.New()
	router.GET("/protected", app.AuthMiddleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("valid credentials", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.SetBasicAuth("admin", "secret")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("wrong credentials", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.SetBasicAuth("admin", "wrong-password")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("no credentials", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestAppCopyIncomingProviderToExistent(t *testing.T) {
	app := newTestApp()

	t.Run("known provider gets updated", func(t *testing.T) {
		newContent := ProviderData{Content: "ola", Type: "TEXT"}
		if err := app.CopyIncomingProviderToExistent("main", newContent); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if app.Providers["main"] != newContent {
			t.Errorf("providers[main] = %+v, want %+v", app.Providers["main"], newContent)
		}
	})

	t.Run("unknown provider is rejected", func(t *testing.T) {
		err := app.CopyIncomingProviderToExistent("nao-existe", ProviderData{})
		if err == nil {
			t.Fatal("expected an error for an unknown provider")
		}
	})
}
