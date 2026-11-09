package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	chatProtocol     = "presenter-chat"
	chatHistoryLimit = 100
	chatMessageLimit = 500
	chatNameLimit    = 40
	chatClientQueue  = 128
	chatWriteWait    = 10 * time.Second
	chatPongWait     = 60 * time.Second
	chatPingPeriod   = 50 * time.Second
)

type chatMessage struct {
	Type     string    `json:"type"`
	Action   string    `json:"action,omitempty"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sentAt"`
	History  bool      `json:"history,omitempty"`
}

type chatHub struct {
	mu      sync.Mutex
	clients map[*chatClient]struct{}
	history []chatMessage
}

type ChatHub struct {
	hub chatHub
}

type chatClient struct {
	conn     *websocket.Conn
	send     chan chatMessage
	username string
	hub      *chatHub
}

func NewChatHub() *ChatHub {
	return &ChatHub{hub: chatHub{clients: make(map[*chatClient]struct{})}}
}

func validChatUsername(username string) bool {
	return username != "" && utf8.RuneCountInString(username) <= chatNameLimit && !strings.ContainsAny(username, "\r\n")
}

func (h *ChatHub) publishTextAdded(username, title string) {
	h.hub.publish(chatMessage{
		Type:     "system",
		Action:   "refreshSongs",
		Username: "Sistema",
		Content:  username + " adicionou o texto " + title,
		SentAt:   time.Now().UTC(),
	})
}

func (h *ChatHub) publishTextRemoved(username, title string) {
	h.hub.publish(chatMessage{
		Type:     "system",
		Action:   "refreshSongs",
		Username: "Sistema",
		Content:  username + " removeu o texto " + title,
		SentAt:   time.Now().UTC(),
	})
}

func (a *App) authenticateChatRequest(r *http.Request) (string, bool) {
	if username, password, ok := r.BasicAuth(); ok && a.validChatCredentials(username, password) {
		return username, true
	}

	for _, protocol := range websocket.Subprotocols(r) {
		if !strings.HasPrefix(protocol, "auth.") {
			continue
		}
		credentialBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(protocol, "auth."))
		if err != nil {
			continue
		}
		username, password, ok := strings.Cut(string(credentialBytes), ":")
		if ok && a.validChatCredentials(username, password) {
			return username, true
		}
	}
	return "", false
}

func (a *App) validChatCredentials(username, password string) bool {
	return username == a.Config.BasicAuthUser && password == a.Config.BasicAuthPass
}

func (a *App) chatWebSocket(c *gin.Context) {
	username, authenticated := a.authenticateChatRequest(c.Request)
	if !authenticated {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	upgrader := websocket.Upgrader{Subprotocols: []string{chatProtocol}}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &chatClient{
		conn:     conn,
		send:     make(chan chatMessage, chatClientQueue),
		username: username,
		hub:      &a.Chat.hub,
	}
	client.hub.register(client)
	go client.writePump()
	client.readPump()
}

func (h *chatHub) register(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, message := range h.history {
		message.History = true
		client.send <- message
	}
	h.clients[client] = struct{}{}
}

func (h *chatHub) unregister(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[client]; exists {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *chatHub) publish(message chatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = append(h.history, message)
	if len(h.history) > chatHistoryLimit {
		h.history = h.history[len(h.history)-chatHistoryLimit:]
	}
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
}

func (client *chatClient) readPump() {
	defer func() {
		client.hub.unregister(client)
		client.conn.Close()
	}()
	client.conn.SetReadLimit(2048)
	client.conn.SetReadDeadline(time.Now().Add(chatPongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(chatPongWait))
	})
	for {
		_, payload, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
		var incoming struct {
			Type     string `json:"type"`
			Username string `json:"username"`
			Content  string `json:"content"`
		}
		if json.Unmarshal(payload, &incoming) != nil {
			continue
		}
		if incoming.Type == "identify" {
			username := strings.TrimSpace(incoming.Username)
			if validChatUsername(username) {
				client.username = username
			}
			continue
		}
		if incoming.Type != "message" || client.username == "" {
			continue
		}
		content := strings.TrimSpace(incoming.Content)
		if content == "" || utf8.RuneCountInString(content) > chatMessageLimit {
			continue
		}
		client.hub.publish(chatMessage{
			Type:     "chat",
			Username: client.username,
			Content:  content,
			SentAt:   time.Now().UTC(),
		})
	}
}

func (client *chatClient) writePump() {
	ticker := time.NewTicker(chatPingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case message, open := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(chatWriteWait))
			if !open {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(chatWriteWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
