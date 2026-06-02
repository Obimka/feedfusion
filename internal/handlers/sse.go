package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"rss-aggregator/internal/auth"
	"gorm.io/gorm"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	UserID   uint
	Response http.ResponseWriter
	Done     chan struct{}
}

// SSEBroker manages SSE connections
type SSEBroker struct {
	Clients    map[uint][]SSEClient
	Register   chan SSEClient
	Unregister chan SSEClient
	Broadcast  chan SSEEvent
}

// SSEEvent represents an event to send to clients
type SSEEvent struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    uint        `json:"user_id,omitempty"`
}

// Global SSE broker
var sseBroker = NewSSEBroker()

// NewSSEBroker creates a new SSE broker
func NewSSEBroker() *SSEBroker {
	broker := &SSEBroker{
		Clients:    make(map[uint][]SSEClient),
		Register:   make(chan SSEClient),
		Unregister: make(chan SSEClient),
		Broadcast:  make(chan SSEEvent),
	}
	go broker.Run()
	return broker
}

// Run manages the SSE broker lifecycle
func (b *SSEBroker) Run() {
	for {
		select {
		case client := <-b.Register:
			b.Clients[client.UserID] = append(b.Clients[client.UserID], client)
			log.Printf("SSE: Client connected for user %d", client.UserID)

		case client := <-b.Unregister:
			for i, c := range b.Clients[client.UserID] {
				if c.Response == client.Response {
					b.Clients[client.UserID] = append(b.Clients[client.UserID][:i], b.Clients[client.UserID][i+1:]...)
					close(c.Done)
					log.Printf("SSE: Client disconnected for user %d", client.UserID)
					break
				}
			}

		case event := <-b.Broadcast:
			// Send to specific user or all users
			if event.UserID > 0 {
				// Send to specific user
				for _, client := range b.Clients[event.UserID] {
					b.sendEvent(client, event)
				}
			} else {
				// Broadcast to all users
				for userID, clients := range b.Clients {
					event.UserID = userID
					for _, client := range clients {
						b.sendEvent(client, event)
					}
				}
			}
		}
	}
}

// sendEvent sends an event to a client
func (b *SSEBroker) sendEvent(client SSEClient, event SSEEvent) {
	// Format as SSE
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("SSE: Error marshaling event: %v", err)
		return
	}

	fmt.Fprintf(client.Response, "data: %s\n\n", string(data))
	// Flush immediately
	if f, ok := client.Response.(http.Flusher); ok {
		f.Flush()
	}
}

// SSEHandler handles SSE connections
func SSEHandler(db *gorm.DB, jwtConfig auth.JWTConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context (set by auth middleware)
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			// Try to get from cookie as fallback
			cookie, err := r.Cookie("token")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err = auth.ValidateJWT(cookie.Value, jwtConfig)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		userID := claims.UserID

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

		// Create client
		client := SSEClient{
			UserID:   userID,
			Response: w,
			Done:     make(chan struct{}),
		}

		// Register client
		sseBroker.Register <- client
		defer func() {
			sseBroker.Unregister <- client
			close(client.Done)
		}()

		// Send initial connection message
		initialEvent := SSEEvent{
			Type:      "connected",
			Data:      map[string]string{"message": "SSE connection established"},
			Timestamp: time.Now(),
			UserID:    userID,
		}
		data, _ := json.Marshal(initialEvent)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Keep connection alive with periodic pings
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send ping to keep connection alive
				fmt.Fprintf(w, "data: {\"type\":\"ping\"}\n\n")
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}

// NotifyNewItems sends a notification about new items to a user
func NotifyNewItems(userID uint, feedID uint, feedTitle string, itemCount int) {
	event := SSEEvent{
		Type: "new_items",
		Data: map[string]interface{}{
			"feed_id":    feedID,
			"feed_title": feedTitle,
			"count":      itemCount,
			"message":    fmt.Sprintf("%d new items in %s", itemCount, feedTitle),
		},
		Timestamp: time.Now(),
		UserID:    userID,
	}
	sseBroker.Broadcast <- event
}

// NotifyFeedUpdated sends a notification about feed updates
func NotifyFeedUpdated(userID uint, feedID uint, feedTitle string) {
	event := SSEEvent{
		Type: "feed_updated",
		Data: map[string]interface{}{
			"feed_id":    feedID,
			"feed_title": feedTitle,
			"message":    fmt.Sprintf("Feed '%s' updated", feedTitle),
		},
		Timestamp: time.Now(),
		UserID:    userID,
	}
	sseBroker.Broadcast <- event
}

// NotifyGeneral sends a general notification to a user
func NotifyGeneral(userID uint, messageType string, message string) {
	event := SSEEvent{
		Type:      messageType,
		Data:      map[string]string{"message": message},
		Timestamp: time.Now(),
		UserID:    userID,
	}
	sseBroker.Broadcast <- event
}
