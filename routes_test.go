package main

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func assertRouteRegistered(t *testing.T, router *gin.Engine, method, path string) {
	t.Helper()
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Errorf("route %s %s was not registered", method, path)
}

func TestRegisterMediaRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerMediaRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"POST", "/api/content/set/:providerId"},
		{"GET", "/api/content"},
		{"POST", "/api/media"},
		{"PUT", "/api/media/move"},
		{"GET", "/api/songs"},
		{"GET", "/api/songs/content"},
		{"GET", "/api/songs/folders"},
		{"GET", "/api/songs/folder"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}

func TestRegisterImageRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerImageRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"POST", "/api/images"},
		{"GET", "/api/images"},
		{"GET", "/api/images/content"},
		{"GET", "/api/images/thumb"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}

func TestRegisterProviderRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerProviderRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"GET", "/api/providers"},
		{"POST", "/api/providers"},
		{"DELETE", "/api/providers/:id"},
		{"DELETE", "/api/providers"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}

func TestRegisterViewRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerViewRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"GET", "/controller"},
		{"GET", "/controller/:page"},
		{"GET", "/"},
		{"GET", "/live"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}

func TestRegisterMiscRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerMiscRoutes(router)

	assertRouteRegistered(t, router, "GET", "/api/discover")
	if got := len(router.Routes()); got != 1 {
		t.Errorf("got %d routes registered, want 1", got)
	}
}

func TestRegisterLyricsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerLyricsRoutes(router)

	want := []struct{ method, path string }{
		{"GET", "/api/lyrics/letras"},
		{"GET", "/api/lyrics/letras/song"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}

func TestRegisterBibleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerBibleRoutes(router)

	want := []struct{ method, path string }{
		{"GET", "/api/bible/books"},
		{"GET", "/api/bible/chapter/:version/:book/:chapter"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}
