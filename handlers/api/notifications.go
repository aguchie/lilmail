package api

import (
	"bufio"
	"encoding/json"
	"lilmail/utils"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

// Notification represents a real-time notification
type Notification struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"` // "new_email", "deleted", "status_change"
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
	Time    time.Time              `json:"time"`
}

// NotificationHandler handles real-time notifications using SSE
type NotificationHandler struct {
	store       *session.Store
	subscribers map[string]chan Notification
	mu          sync.RWMutex
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(store *session.Store) *NotificationHandler {
	return &NotificationHandler{
		store:       store,
		subscribers: make(map[string]chan Notification),
	}
}

// HandleSSE handles Server-Sent Events for real-time notifications
func (h *NotificationHandler) HandleSSE(c *fiber.Ctx) error {
	// Set headers for SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	
	// Get session token to identify subscriber
	token, err := GetSessionToken(c, h.store)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}
	
	// Create channel for this subscriber
	subscriberID := uuid.New().String()
	messageChan := make(chan Notification, 10)
	
	h.mu.Lock()
	h.subscribers[subscriberID] = messageChan
	h.mu.Unlock()
	
	// Cleanup on disconnect
	defer func() {
		h.mu.Lock()
		delete(h.subscribers, subscriberID)
		close(messageChan)
		h.mu.Unlock()
		
		utils.Log.Info("SSE subscriber disconnected: %s (token: %s)", subscriberID, token[:8])
	}()
	
	utils.Log.Info("SSE subscriber connected: %s", subscriberID)
	
	// Send initial connection message  
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Keep-alive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case notification := <-messageChan:
				// Send notification
				data, _ := json.Marshal(notification)
				w.WriteString("data: " + string(data) + "\n\n")
				w.Flush()
				
			case <-ticker.C:
				// Send keep-alive comment
				w.WriteString(": keepalive\n\n")
				w.Flush()
				
			case <-c.Context().Done():
				return
			}
		}
	}))
	
	return nil
}

// HandleWebSocket handles WebSocket connections for real-time notifications
func (h *NotificationHandler) HandleWebSocket(c *websocket.Conn) {
	subscriberID := uuid.New().String()
	messageChan := make(chan Notification, 10)
	
	h.mu.Lock()
	h.subscribers[subscriberID] = messageChan
	h.mu.Unlock()
	
	defer func() {
		h.mu.Lock()
		delete(h.subscribers, subscriberID)
		close(messageChan)
		h.mu.Unlock()
		
		c.Close()
		utils.Log.Info("WebSocket subscriber disconnected: %s", subscriberID)
	}()
	
	utils.Log.Info("WebSocket subscriber connected: %s", subscriberID)
	
	// Send messages
	for notification := range messageChan {
		if err := c.WriteJSON(notification); err != nil {
			utils.Log.Error("Failed to send WebSocket notification: %v", err)
			break
		}
	}
}

// BroadcastNotification sends a notification to all subscribers
func (h *NotificationHandler) BroadcastNotification(notification Notification) {
	notification.ID = uuid.New().String()
	notification.Time = time.Now()
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	utils.Log.Info("Broadcasting notification: type=%s to %d subscribers", notification.Type, len(h.subscribers))
	
	for subscriberID, ch := range h.subscribers {
		select {
		case ch <- notification:
			// Sent successfully
		default:
			// Channel full, skip this subscriber
			utils.Log.Warn("Notification channel full for subscriber %s", subscriberID)
		}
	}
}

// NotifyNewEmail sends a notification for a new email
func (h *NotificationHandler) NotifyNewEmail(from, subject string) {
	h.BroadcastNotification(Notification{
		Type:    "new_email",
		Message: "New email received",
		Data: map[string]interface{}{
			"from":    from,
			"subject": subject,
		},
	})
}

// NotifyEmailDeleted sends a notification for a deleted email
func (h *NotificationHandler) NotifyEmailDeleted(emailID string) {
	h.BroadcastNotification(Notification{
		Type:    "deleted",
		Message: "Email deleted",
		Data: map[string]interface{}{
			"email_id": emailID,
		},
	})
}

// NotifyStatusChange sends a notification for an email status change
func (h *NotificationHandler) NotifyStatusChange(emailID, status string) {
	h.BroadcastNotification(Notification{
		Type:    "status_change",
		Message: "Email status changed",
		Data: map[string]interface{}{
			"email_id": emailID,
			"status":   status,
		},
	})
}
