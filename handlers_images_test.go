package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newImageTestRouter(t *testing.T) (*gin.Engine, *App) {
	t.Helper()
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	app := newTestApp()
	router := gin.New()
	registerImageRoutes(router, app)
	return router, app
}

func pngUploadBody(t *testing.T, filename string) (*bytes.Buffer, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encoding png fixture: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := part.Write(pngBuf.Bytes()); err != nil {
		t.Fatalf("writing form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestUploadImageRequiresAuth(t *testing.T) {
	router, _ := newImageTestRouter(t)
	body, contentType := pngUploadBody(t, "foto.png")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUploadImageThenListAndFetch(t *testing.T) {
	router, app := newImageTestRouter(t)
	body, contentType := pngUploadBody(t, "foto.png")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", contentType)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("foto.png")) {
		t.Errorf("list body %q does not contain %q", w.Body.String(), "foto.png")
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images/content?name=foto.png", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("content status = %d, want %d", w.Code, http.StatusOK)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images/thumb?name=foto.png", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("thumb status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUploadImageRejectsUnsupportedFormat(t *testing.T) {
	router, app := newImageTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "nota.txt")
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	part.Write([]byte("just some text, not an image"))
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetImageContentNotFound(t *testing.T) {
	router, _ := newImageTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/images/content?name=nao-existe.png", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetImageThumbNotFound(t *testing.T) {
	router, _ := newImageTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/images/thumb?name=nao-existe.png", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
