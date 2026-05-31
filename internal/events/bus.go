package events

import (
	"strconv"
	"sync"
	"time"
)

const (
	RequestCreated       = "request.created"
	RequestUpdated       = "request.updated"
	RequestStatusChanged = "request.status_changed"

	replayBufferSize = 100
)

type Event struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	OccurredAt     time.Time              `json:"occurred_at"`
	Aggregate      string                 `json:"aggregate"`
	AggregateID    uint                   `json:"aggregate_id"`
	Action         string                 `json:"action,omitempty"`
	Status         string                 `json:"status,omitempty"`
	PreviousStatus string                 `json:"previous_status,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
}

type Bus struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]chan Event

	replayMu sync.RWMutex
	replay   [replayBufferSize]Event
	replayHead int
	replayCount int
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[uint64]chan Event),
	}
}

func (b *Bus) Publish(event Event) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	if event.ID == "" {
		event.ID = strconv.FormatInt(event.OccurredAt.UnixNano(), 10)
	}

	b.replayMu.Lock()
	b.replay[b.replayHead] = event
	b.replayHead = (b.replayHead + 1) % replayBufferSize
	if b.replayCount < replayBufferSize {
		b.replayCount++
	}
	b.replayMu.Unlock()

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *Bus) Subscribe(buffer int) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	ch := make(chan Event, buffer)
	b.subscribers[id] = ch

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if sub, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(sub)
		}
	}

	return ch, cancel
}

func (b *Bus) EventsSince(lastID string) []Event {
	if lastID == "" {
		return nil
	}
	b.replayMu.RLock()
	defer b.replayMu.RUnlock()

	if b.replayCount == 0 {
		return nil
	}

	start := b.replayHead - b.replayCount
	if start < 0 {
		start += replayBufferSize
	}

	var result []Event
	for i := 0; i < b.replayCount; i++ {
		idx := (start + i) % replayBufferSize
		ev := b.replay[idx]
		if ev.ID > lastID {
			result = append(result, ev)
		}
	}
	return result
}

var DefaultBus = NewBus()
