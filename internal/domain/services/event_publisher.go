package services

type Event interface {
	EventType() string
}

type EventPublisher interface {
	Publish(event Event) error
	Subscribe(eventType string, handler func(Event) error) error
}
