package messaging

import (
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type EventBus struct {
	subscribers map[string][]func(services.Event) error
	mu          sync.RWMutex
}

func NewEventBus() services.EventPublisher {
	return &EventBus{
		subscribers: make(map[string][]func(services.Event) error),
	}
}

func (eb *EventBus) Publish(event services.Event) error {
	eb.mu.RLock()
	handlers := eb.subscribers[event.EventType()]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			return err
		}
	}
	return nil
}

func (eb *EventBus) Subscribe(eventType string, handler func(services.Event) error) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
	return nil
}
