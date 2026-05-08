package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Sriniketh24/rollout/internal/streaming"
	"github.com/nats-io/nats.go"
)

// subject builds a NATS subject for flag change events in an environment.
// Format: rollout.{projectID}.{envID}.flags
func subject(projectID, envID string) string {
	return fmt.Sprintf("rollout.%s.%s.flags", projectID, envID)
}

// Bus bridges NATS pub/sub with the SSE Hub, allowing flag change events
// published by any server instance to be forwarded to connected clients.
type Bus struct {
	conn *nats.Conn
	hub  *streaming.Hub

	mu            sync.Mutex
	subscriptions map[string]*nats.Subscription // subject -> subscription
}

// NewBus creates a Bus that publishes and subscribes to the given NATS
// connection and forwards received events to hub.
func NewBus(conn *nats.Conn, hub *streaming.Hub) *Bus {
	return &Bus{
		conn:          conn,
		hub:           hub,
		subscriptions: make(map[string]*nats.Subscription),
	}
}

// PublishFlagChange serialises the event as JSON and publishes it to the
// appropriate NATS subject for the given environment.
func (b *Bus) PublishFlagChange(ctx context.Context, projectID, envID string, event streaming.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("events: marshal: %w", err)
	}

	subj := subject(projectID, envID)
	if err := b.conn.Publish(subj, data); err != nil {
		return fmt.Errorf("events: publish to %s: %w", subj, err)
	}

	return nil
}

// Subscribe starts listening on the NATS subject for the given environment
// and forwards every received message to the SSE Hub. Calling Subscribe
// multiple times for the same environment is a no-op.
func (b *Bus) Subscribe(projectID, envID string) error {
	subj := subject(projectID, envID)

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscriptions[subj]; exists {
		return nil // already subscribed
	}

	sub, err := b.conn.Subscribe(subj, func(msg *nats.Msg) {
		var event streaming.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("events: unmarshal from %s: %v", subj, err)
			return
		}
		b.hub.Publish(projectID, envID, event)
	})
	if err != nil {
		return fmt.Errorf("events: subscribe to %s: %w", subj, err)
	}

	b.subscriptions[subj] = sub
	return nil
}

// Unsubscribe stops listening on the NATS subject for the given environment.
func (b *Bus) Unsubscribe(projectID, envID string) error {
	subj := subject(projectID, envID)

	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subscriptions[subj]
	if !exists {
		return nil
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("events: unsubscribe from %s: %w", subj, err)
	}

	delete(b.subscriptions, subj)
	return nil
}

// Close drains all active subscriptions and releases resources. The
// underlying NATS connection is NOT closed — the caller owns that.
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var firstErr error
	for subj, sub := range b.subscriptions {
		if err := sub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("events: close unsubscribe %s: %w", subj, err)
		}
		delete(b.subscriptions, subj)
	}

	return firstErr
}
