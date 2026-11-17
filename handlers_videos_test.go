package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newVideoTestRouter(t *testing.T) (*gin.Engine, *App) {
	t.Helper()
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	app := newTestApp()
	router := gin.New()
	registerVideoRoutes(router, app)
	return router, app
}

func mp4UploadBody(t *testing.T, filename string) (*bytes.Buffer, string) {
	t.Helper()
	mp4Data := []byte{0x00, 0x00, 0x00, 0x14, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0x00, 0x00, 0x02, 0x00}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := part.Write(mp4Data); err != nil {
		t.Fatalf("writing form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestUploadVideoRequiresAuth(t *testing.T) {
	router, _ := newVideoTestRouter(t)
	body, contentType := mp4UploadBody(t, "video.mp4")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/videos", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUploadVideoThenListFetchAndDelete(t *testing.T) {
	router, app := newVideoTestRouter(t)
	body, contentType := mp4UploadBody(t, "clipe.mp4")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/videos", body)
	req.Header.Set("Content-Type", contentType)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// List videos
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/videos", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("clipe.mp4")) {
		t.Errorf("list body %q does not contain %q", w.Body.String(), "clipe.mp4")
	}

	// Fetch video content
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/videos/content?name=clipe.mp4", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("content status = %d, want %d", w.Code, http.StatusOK)
	}

	// Delete video
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/videos/clipe.mp4", nil)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", w.Code, http.StatusOK)
	}

	// Confirm 404 after deletion
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/videos/content?name=clipe.mp4", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status after deletion = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUploadVideoRejectsUnsupportedFormat(t *testing.T) {
	router, app := newVideoTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "arquivo.txt")
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	part.Write([]byte("just plain text"))
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/videos", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetVideoContentNotFound(t *testing.T) {
	router, _ := newVideoTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/videos/content?name=nao-existe.mp4", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetDownloaderStatus(t *testing.T) {
	router, _ := newVideoTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/videos/downloader/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("installScript")) {
		t.Errorf("expected response to contain installScript, got %s", w.Body.String())
	}
}

func TestDownloadVideoRequiresAuth(t *testing.T) {
	router, _ := newVideoTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/videos/download", bytes.NewBufferString(`{"url":"https://youtube.com/watch?v=123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDownloadVideoRejectsEmptyURL(t *testing.T) {
	router, app := newVideoTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/videos/download", bytes.NewBufferString(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
