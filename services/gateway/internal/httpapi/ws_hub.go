package httpapi

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsSendBufferSize = 64
	wsWriteWait      = 5 * time.Second
)

type wsClient struct {
	conn  *websocket.Conn
	send  chan []byte
	close sync.Once
}

type wsHub struct {
	mu    sync.RWMutex
	conns map[string]map[*wsClient]struct{}
}

func newHub() *wsHub {
	return &wsHub{conns: make(map[string]map[*wsClient]struct{})}
}

func (h *wsHub) register(tenantID string, conn *websocket.Conn) *wsClient {
	client := &wsClient{
		conn: conn,
		send: make(chan []byte, wsSendBufferSize),
	}
	go client.writePump()

	h.mu.Lock()
	if h.conns[tenantID] == nil {
		h.conns[tenantID] = make(map[*wsClient]struct{})
	}
	h.conns[tenantID][client] = struct{}{}
	h.mu.Unlock()
	return client
}

func (h *wsHub) unregister(tenantID string, client *wsClient) {
	h.mu.Lock()
	if m := h.conns[tenantID]; m != nil {
		delete(m, client)
	}
	h.mu.Unlock()
	client.close.Do(func() { close(client.send) })
}

func (h *wsHub) broadcast(tenantID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.conns[tenantID] {
		select {
		case client.send <- payload:
		default:
			// Slow consumer: drop and disconnect so the feed stays healthy.
			go h.dropClient(tenantID, client)
		}
	}
}

func (h *wsHub) dropClient(tenantID string, client *wsClient) {
	h.unregister(tenantID, client)
	_ = client.conn.Close()
}

func (c *wsClient) writePump() {
	for payload := range c.send {
		_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
		if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}
