package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Message represents a real-time update message
type Message struct {
	Type      string        `json:"type"`      // workflow-started, workflow-completed, etc
	WorkflowID string        `json:"workflow_id"`
	Status    string        `json:"status"`
	Progress  int           `json:"progress"`
	Data      interface{}   `json:"data,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// Hub manages real-time connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client represents a connected client
type Client struct {
	id    string
	hub   *Hub
	send  chan *Message
	close chan bool
}

// NewHub creates a new real-time hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Start begins the hub's event loop
func (h *Hub) Start() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.mu.Lock()
				h.clients[client] = true
				h.mu.Unlock()
				log.Printf("[Hub] Client registered: %s (total: %d)", client.id, len(h.clients))

			case client := <-h.unregister:
				h.mu.Lock()
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
				}
				h.mu.Unlock()
				log.Printf("[Hub] Client unregistered: %s (total: %d)", client.id, len(h.clients))

			case msg := <-h.broadcast:
				h.mu.RLock()
				for client := range h.clients {
					select {
					case client.send <- msg:
					default:
						// Client's send channel is full, skip
						log.Printf("[Hub] Dropping message for slow client: %s", client.id)
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// RegisterClient adds a new client
func (h *Hub) RegisterClient(id string) *Client {
	client := &Client{
		id:    id,
		hub:   h,
		send:  make(chan *Message, 64),
		close: make(chan bool),
	}
	h.register <- client
	return client
}

// UnregisterClient removes a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg *Message) {
	msg.Timestamp = time.Now()
	h.broadcast <- msg
}

// BroadcastJSON broadcasts a message from JSON data
func (h *Hub) BroadcastJSON(msgType, workflowID, status string, progress int, data interface{}) {
	msg := &Message{
		Type:       msgType,
		WorkflowID: workflowID,
		Status:     status,
		Progress:   progress,
		Data:       data,
	}
	h.Broadcast(msg)
}

// BroadcastWorkflowUpdate broadcasts a workflow status update
func (h *Hub) BroadcastWorkflowUpdate(workflowID, status string, progress int) {
	msg := &Message{
		Type:       "workflow-update",
		WorkflowID: workflowID,
		Status:     status,
		Progress:   progress,
	}
	h.Broadcast(msg)
}

// BroadcastWorkflowStarted broadcasts a workflow start event
func (h *Hub) BroadcastWorkflowStarted(workflowID, wfType string) {
	msg := &Message{
		Type:       "workflow-started",
		WorkflowID: workflowID,
		Status:     "running",
		Progress:   0,
		Data: map[string]interface{}{
			"type": wfType,
		},
	}
	h.Broadcast(msg)
}

// BroadcastWorkflowCompleted broadcasts a workflow completion event
func (h *Hub) BroadcastWorkflowCompleted(workflowID, status string, result interface{}) {
	progress := 100
	if status != "success" {
		progress = 0
	}

	msg := &Message{
		Type:       "workflow-completed",
		WorkflowID: workflowID,
		Status:     status,
		Progress:   progress,
		Data:       result,
	}
	h.Broadcast(msg)
}

// BroadcastMetricsUpdate broadcasts a metrics update
func (h *Hub) BroadcastMetricsUpdate(stats map[string]interface{}) {
	msg := &Message{
		Type:     "metrics-update",
		Status:   "updated",
		Progress: 100,
		Data:     stats,
	}
	h.Broadcast(msg)
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendMessage sends a message to a specific client
func (c *Client) SendMessage(msg *Message) {
	select {
	case c.send <- msg:
	case <-c.close:
		// Client is closed
	default:
		log.Printf("[Client] Send buffer full: %s", c.id)
	}
}

// ReceiveMessage receives the next message for the client
func (c *Client) ReceiveMessage() (*Message, bool) {
	select {
	case msg := <-c.send:
		return msg, true
	case <-c.close:
		return nil, false
	}
}

// Close closes the client connection
func (c *Client) Close() {
	select {
	case <-c.close:
	default:
		close(c.close)
		c.hub.UnregisterClient(c)
	}
}

// MessageToJSON converts a message to JSON
func (m *Message) ToJSON() []byte {
	data, _ := json.Marshal(m)
	return data
}
