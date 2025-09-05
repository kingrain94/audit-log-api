package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/service"
	"github.com/kingrain94/audit-log-api/internal/service/pubsub"
	"github.com/kingrain94/audit-log-api/internal/utils"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

const (
	websocketReadBufferSize        = 1024
	websocketWriteBufferSize       = 1024
	websocketSendChannelBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  websocketReadBufferSize,
	WriteBufferSize: websocketWriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	tenantID string
	send     chan []byte
}

type WebSocketHandler struct {
	auditLogService *service.AuditLogService
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client
	mutex           sync.RWMutex
	logger          *logger.Logger
	pubsub          *pubsub.RedisPubSub
	ctx             context.Context
	cancel          context.CancelFunc
	tenantClients   map[string]int // Count of clients per tenant
}

func NewWebSocketHandler(auditLogService *service.AuditLogService, logger *logger.Logger, pubsub *pubsub.RedisPubSub) *WebSocketHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHandler{
		auditLogService: auditLogService,
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		logger:          logger,
		pubsub:          pubsub,
		ctx:             ctx,
		cancel:          cancel,
		tenantClients:   make(map[string]int),
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware). tenant scope is required
	tenantID, exists := c.Get(string(utils.TenantIDKey))
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No tenant ID found"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// Create and register new client
	client := &Client{
		conn:     conn,
		tenantID: tenantID.(string),
		send:     make(chan []byte, websocketSendChannelBufferSize),
	}
	h.register <- client

	go h.writePump(client)
	go h.readPump(client)
}

func (h *WebSocketHandler) Start() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.tenantClients[client.tenantID]++

			// Subscribe to tenant's channel if this is the first client
			if h.tenantClients[client.tenantID] == 1 {
				if err := h.pubsub.Subscribe(h.ctx, client.tenantID, h.handlePubSubMessage); err != nil {
					h.logger.Errorf("Failed to subscribe to tenant %s: %v", client.tenantID, err)
				}
			}
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Decrement tenant client count
				h.tenantClients[client.tenantID]--

				// Unsubscribe if no more clients for this tenant
				if h.tenantClients[client.tenantID] == 0 {
					h.pubsub.Unsubscribe(client.tenantID)
					delete(h.tenantClients, client.tenantID)
				}
			}
			h.mutex.Unlock()

		case <-h.ctx.Done():
			return
		}
	}
}

func (h *WebSocketHandler) Stop() {
	h.cancel()
	h.pubsub.Close()
}

// handlePubSubMessage handles messages received from Redis pub/sub
func (h *WebSocketHandler) handlePubSubMessage(log *dto.AuditLogResponse) {
	message, err := json.Marshal(log)
	if err != nil {
		h.logger.Errorf("Error marshaling log: %v", err)
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.tenantID == log.TenantID {
			select {
			case client.send <- message:
			default: // If the channel is full, close the channel and remove the client
				close(client.send)
				delete(h.clients, client)
				h.tenantClients[client.tenantID]--

				// Unsubscribe if no more clients for this tenant
				if h.tenantClients[client.tenantID] == 0 {
					h.pubsub.Unsubscribe(client.tenantID)
					delete(h.tenantClients, client.tenantID)
				}
			}
		}
	}
}

func (h *WebSocketHandler) writePump(client *Client) {
	defer func() {
		client.conn.Close()
	}()

	for message := range client.send {
		w, err := client.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		if err := w.Close(); err != nil {
			return
		}
	}

	// Channel was closed, send close message
	client.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

func (h *WebSocketHandler) readPump(client *Client) {
	defer func() {
		h.unregister <- client
		client.conn.Close()
	}()

	for {
		messageType, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warnf("Unexpected close error for client %s: %v", client.tenantID, err)
			} else {
				h.logger.Warnf("Read error for client %s: %v", client.tenantID, err)
			}
			break
		}

		// Handle any actual messages from client (though we don't expect any)
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			h.logger.Infof("Received message from client %s: %s", client.tenantID, string(message))
		}
	}
}

// BroadcastLog sends a log to all connected clients of the same tenant
func (h *WebSocketHandler) BroadcastLog(log *dto.AuditLogResponse) {
	if err := h.pubsub.Publish(h.ctx, log); err != nil {
		h.logger.Errorf("Failed to publish log: %v", err)
	}
}
