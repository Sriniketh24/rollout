package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// EventType represents the type of SSE event being broadcast.
type EventType string

const (
	EventFlagUpdated        EventType = "flag.updated"
	EventFlagCreated        EventType = "flag.created"
	EventFlagDeleted        EventType = "flag.deleted"
	EventFlagToggled        EventType = "flag.toggled"
	EventEnvironmentUpdated EventType = "environment.updated"
	EventExperimentStarted  EventType = "experiment.started"
	EventExperimentStopped  EventType = "experiment.stopped"
)

const (
	// clientBufferSize is the number of events buffered per client before
	// overflow protection kicks in and the oldest event is dropped.
	clientBufferSize = 64

	// heartbeatInterval controls how often a keep-alive comment is sent.
	heartbeatInterval = 30 * time.Second
)

// Event is a single SSE payload sent to connected clients.
type Event struct {
	Type      EventType       `json:"type"`
	FlagKey   string          `json:"flag_key,omitempty"`
	Data      json.RawMessage `json:"data"`
	Version   int64           `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
}

// envKey builds a map key from a project ID and environment ID.
func envKey(projectID, envID string) string {
	return projectID + ":" + envID
}

// client is an individual SSE subscriber.
type client struct {
	events chan Event
}

// Hub manages SSE client connections grouped by environment.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{} // envKey -> set of clients
}

// NewHub creates a ready-to-use Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
	}
}

// Subscribe streams SSE events for the given environment to the HTTP client.
// It blocks until the client disconnects or the request context is cancelled.
func (h *Hub) Subscribe(projectID, envID string, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	c := &client{
		events: make(chan Event, clientBufferSize),
	}

	key := envKey(projectID, envID)

	h.mu.Lock()
	if h.clients[key] == nil {
		h.clients[key] = make(map[*client]struct{})
	}
	h.clients[key][c] = struct{}{}
	h.mu.Unlock()

	// Ensure the client is removed when we return.
	defer func() {
		h.mu.Lock()
		delete(h.clients[key], c)
		if len(h.clients[key]) == 0 {
			delete(h.clients, key)
		}
		h.mu.Unlock()
	}()

	ctx := r.Context()
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	// Send an initial comment so the client knows the connection is live.
	fmt.Fprintf(w, ": connected to %s/%s\n\n", projectID, envID)
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-c.events:
			if err := writeEvent(w, ev); err != nil {
				log.Printf("sse: write error for %s: %v", key, err)
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}

// Publish broadcasts an event to every subscriber for the given environment.
func (h *Hub) Publish(projectID, envID string, event Event) {
	key := envKey(projectID, envID)

	h.mu.RLock()
	subscribers := h.clients[key]
	h.mu.RUnlock()

	for c := range subscribers {
		select {
		case c.events <- event:
		default:
			// Buffer is full — drop the oldest event to make room.
			select {
			case <-c.events:
			default:
			}
			// Try sending again; if it still fails the event is dropped.
			select {
			case c.events <- event:
			default:
			}
		}
	}
}

// Stats returns the number of connected clients per environment key
// (formatted as "projectID:envID").
func (h *Hub) Stats() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[string]int, len(h.clients))
	for key, clients := range h.clients {
		stats[key] = len(clients)
	}
	return stats
}

// ConnectedClients returns the subscriber count for a single environment.
func (h *Hub) ConnectedClients(projectID, envID string) int {
	key := envKey(projectID, envID)

	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[key])
}

// writeEvent formats and writes one SSE message to w.
func writeEvent(w http.ResponseWriter, ev Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// event: <type>\ndata: <json>\n\n
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, payload); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// ServeHTTP is a convenience handler that extracts projectID and envID from
// query parameters and subscribes the caller. Useful for quick wiring:
//
//	mux.Handle("/v1/stream", hub)
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	envID := r.URL.Query().Get("env_id")
	if projectID == "" || envID == "" {
		http.Error(w, `{"error":"project_id and env_id query params required"}`, http.StatusBadRequest)
		return
	}

	h.Subscribe(projectID, envID, w, r)
}

// NewEvent is a helper to construct an Event with the timestamp set to now.
func NewEvent(eventType EventType, flagKey string, data json.RawMessage, version int64) Event {
	return Event{
		Type:      eventType,
		FlagKey:   flagKey,
		Data:      data,
		Version:   version,
		Timestamp: time.Now().UTC(),
	}
}

// PublishWithContext is like Publish but respects context cancellation. It is
// useful when the caller wants to abandon broadcasting if a deadline expires.
func (h *Hub) PublishWithContext(ctx context.Context, projectID, envID string, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		h.Publish(projectID, envID, event)
		return nil
	}
}
