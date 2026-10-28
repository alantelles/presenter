package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetSongContentUsesTextPlainContentType(t *testing.T) {
	dir := t.TempDir()
	songsDir := filepath.Join(dir, "media", "songs")
	if err := os.MkdirAll(songsDir, 0755); err != nil {
		t.Fatalf("failed to create test songs dir: %v", err)
	}
	const songBody = "Título\nAutor\n\nLetra da música"
	if err := os.WriteFile(filepath.Join(songsDir, "musica-teste.txt"), []byte(songBody), 0644); err != nil {
		t.Fatalf("failed to write test song file: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	defer os.Chdir(cwd)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/songs/content?song=musica-teste.txt", nil)

	getSongContent(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Type"); got != ContentTypeText {
		t.Errorf("Content-Type = %q, want %q", got, ContentTypeText)
	}
	if got := w.Body.String(); got != songBody {
		t.Errorf("body = %q, want %q", got, songBody)
	}
}
