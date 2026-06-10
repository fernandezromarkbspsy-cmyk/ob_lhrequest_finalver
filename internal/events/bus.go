package events

import (
	"sync"
	"time"
)

const (
	RequestCreated      = "REQUEST_CREATED"
	RequestApproved     = "REQUEST_APPROVED"
	RequestEdited       = "REQUEST_EDITED"
	RequestCancelled    = "REQUEST_CANCELLED"
	RequestRejectedByMM = "REQUEST_REJECTED_BY_MM"
	TruckAssigned       = "TRUCK_ASSIGNED"
	TruckForDocking     = "TRUCK_FOR_DOCKING"
	TruckDocked         = "TRUCK_DOCKED"
	RequestConfirmed    = "REQUEST_CONFIRMED"
	UserCreated         = "USER_CREATED"
	UserUpdated         = "USER_UPDATED"
	UserDisabled        = "USER_DISABLED"
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

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
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
		if subscriber, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(subscriber)
		}
	}

	return ch, cancel
}

var DefaultBus = NewBus()
