package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestChatWebSocketRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := NewApp(Config{BasicAuthUser: "operator", BasicAuthPass: "secret"})
	router := gin.New()
	registerChatRoutes(router, app)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestChatWebSocketBroadcastsAndKeepsRecentHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := NewApp(Config{BasicAuthUser: "operator", BasicAuthPass: "secret"})
	router := gin.New()
	registerChatRoutes(router, app)
	server := httptest.NewServer(router)
	defer server.Close()
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/chat"
	credential := base64.RawURLEncoding.EncodeToString([]byte("operator:secret"))
	protocols := []string{chatProtocol, "auth." + credential}
	dialer := websocket.Dialer{Subprotocols: protocols}

	first, response, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		if response != nil {
			t.Fatalf("first websocket connection failed: %v (status %d)", err, response.StatusCode)
		}
		t.Fatalf("first websocket connection failed: %v", err)
	}
	defer first.Close()
	second, _, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("second websocket connection failed: %v", err)
	}
	defer second.Close()
	for _, client := range []*websocket.Conn{first, second} {
		if err := client.WriteJSON(map[string]string{"type": "identify", "username": "Equipe do culto"}); err != nil {
			t.Fatalf("sending chat identity failed: %v", err)
		}
	}

	if err := first.WriteJSON(map[string]string{"type": "message", "content": "Mensagem de teste"}); err != nil {
		t.Fatalf("sending chat message failed: %v", err)
	}
	for _, client := range []*websocket.Conn{first, second} {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		var message chatMessage
		if err := client.ReadJSON(&message); err != nil {
			t.Fatalf("receiving chat message failed: %v", err)
		}
		if message.Username != "Equipe do culto" || message.Content != "Mensagem de teste" || message.SentAt.IsZero() {
			t.Errorf("unexpected chat message: %+v", message)
		}
	}

	third, _, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("third websocket connection failed: %v", err)
	}
	defer third.Close()
	if err := third.WriteJSON(map[string]string{"type": "identify", "username": "Terceiro operador"}); err != nil {
		t.Fatalf("sending history reader identity failed: %v", err)
	}
	third.SetReadDeadline(time.Now().Add(2 * time.Second))
	var historyMessage chatMessage
	if err := third.ReadJSON(&historyMessage); err != nil {
		t.Fatalf("receiving chat history failed: %v", err)
	}
	if historyMessage.Content != "Mensagem de teste" || !historyMessage.History {
		t.Errorf("history message = %+v, want marked historical message", historyMessage)
	}
}

func TestSavingSongBroadcastsSystemRefreshCommand(t *testing.T) {
	withTempWorkDir(t)
	if err := os.MkdirAll(filepath.Join("media", "songs"), 0755); err != nil {
		t.Fatalf("creating songs directory failed: %v", err)
	}
	gin.SetMode(gin.TestMode)
	app := NewApp(Config{BasicAuthUser: "operator", BasicAuthPass: "secret"})
	router := gin.New()
	registerChatRoutes(router, app)
	registerMediaRoutes(router, app)
	server := httptest.NewServer(router)
	defer server.Close()
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/chat"
	credential := base64.RawURLEncoding.EncodeToString([]byte("operator:secret"))
	dialer := websocket.Dialer{Subprotocols: []string{chatProtocol, "auth." + credential}}
	clients := make([]*websocket.Conn, 0, 2)
	for _, username := range []string{"Ana", "Bruno"} {
		client, _, err := dialer.Dial(websocketURL, nil)
		if err != nil {
			t.Fatalf("websocket connection failed: %v", err)
		}
		clients = append(clients, client)
		t.Cleanup(func() { client.Close() })
		if err := client.WriteJSON(map[string]string{"type": "identify", "username": username}); err != nil {
			t.Fatalf("sending chat identity failed: %v", err)
		}
	}

	payload, err := json.Marshal(Media{
		Category: "songs",
		Title:    "Canção Nova",
		Author:   "Equipe",
		Content:  "Letra salva",
		Username: "Ana",
	})
	if err != nil {
		t.Fatalf("encoding media payload failed: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/media", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("creating media request failed: %v", err)
	}
	request.SetBasicAuth("operator", "secret")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("saving media failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if _, err := os.Stat(filepath.Join("media", "songs", "Canção Nova - Equipe.txt")); err != nil {
		t.Fatalf("saved song was not found: %v", err)
	}

	for _, client := range clients {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		var message chatMessage
		if err := client.ReadJSON(&message); err != nil {
			t.Fatalf("receiving refresh command failed: %v", err)
		}
		if message.Type != "system" || message.Action != "refreshSongs" || message.Username != "Sistema" || message.Content != "Ana adicionou o texto Canção Nova - Equipe" {
			t.Errorf("unexpected system message: %+v", message)
		}
	}
}

func TestRemovingSongBroadcastsSystemRefreshCommand(t *testing.T) {
	withTempWorkDir(t)
	songPath := filepath.Join("media", "songs", "Cancao Teste.txt")
	if err := os.MkdirAll(filepath.Dir(songPath), 0755); err != nil {
		t.Fatalf("creating songs directory failed: %v", err)
	}
	if err := os.WriteFile(songPath, []byte("letra"), 0644); err != nil {
		t.Fatalf("creating song file failed: %v", err)
	}
	gin.SetMode(gin.TestMode)
	app := NewApp(Config{BasicAuthUser: "operator", BasicAuthPass: "secret"})
	router := gin.New()
	registerChatRoutes(router, app)
	registerMediaRoutes(router, app)
	server := httptest.NewServer(router)
	defer server.Close()
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/chat"
	credential := base64.RawURLEncoding.EncodeToString([]byte("operator:secret"))
	dialer := websocket.Dialer{Subprotocols: []string{chatProtocol, "auth." + credential}}
	client, _, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("websocket connection failed: %v", err)
	}
	defer client.Close()
	if err := client.WriteJSON(map[string]string{"type": "identify", "username": "Ana"}); err != nil {
		t.Fatalf("sending chat identity failed: %v", err)
	}

	payload, err := json.Marshal(MoveMediaCommand{
		MediaID:     "Cancao Teste.txt",
		Destination: "musicas/_removidas",
		Category:    "songs",
		Username:    "Ana",
	})
	if err != nil {
		t.Fatalf("encoding move payload failed: %v", err)
	}
	request, err := http.NewRequest(http.MethodPut, server.URL+"/api/media/move", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("creating move request failed: %v", err)
	}
	request.SetBasicAuth("operator", "secret")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("removing song failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("move status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if _, err := os.Stat(filepath.Join("media", "songs", "musicas", "_removidas", "Cancao Teste.txt")); err != nil {
		t.Fatalf("removed song was not moved to archive: %v", err)
	}

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	var message chatMessage
	if err := client.ReadJSON(&message); err != nil {
		t.Fatalf("receiving refresh command failed: %v", err)
	}
	if message.Type != "system" || message.Action != "refreshSongs" || message.Username != "Sistema" || message.Content != "Ana removeu o texto Cancao Teste" {
		t.Errorf("unexpected system message: %+v", message)
	}
}
