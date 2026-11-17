package entities

import "time"

type AuditEvent struct {
	ID        string
	EventType string
	Actor     string
	Resource  string
	Action    string
	Result    string
	Details   map[string]interface{}
	CreatedAt time.Time
}

func NewAuditEvent(eventType, actor, resource, action string) *AuditEvent {
	return &AuditEvent{
		EventType: eventType,
		Actor:     actor,
		Resource:  resource,
		Action:    action,
		CreatedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

func (e *AuditEvent) Success() {
	e.Result = "success"
}

func (e *AuditEvent) Failure(reason string) {
	e.Result = "failure"
	e.Details["error"] = reason
}
