package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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
