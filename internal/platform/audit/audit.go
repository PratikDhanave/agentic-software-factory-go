package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/platform/logging"
	"github.com/google/uuid"
)

type EventType string

const (
	AgentCreated       EventType = "agent.created"
	AgentTransitioned  EventType = "agent.transitioned"
	ToolInvoked        EventType = "tool.invoked"
	WorkflowStarted    EventType = "workflow.started"
	WorkflowCompleted  EventType = "workflow.completed"
	CredentialIssued   EventType = "credential.issued"
	CredentialRevoked  EventType = "credential.revoked"
	PolicyEvaluated    EventType = "policy.evaluated"
	ErrorOccurred      EventType = "error.occurred"
)

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Subject   string                 `json:"subject"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func Log(ctx context.Context, typ EventType, subject string, agentID string, details map[string]interface{}) {
	event := Event{
		ID:        uuid.New().String(),
		Type:      typ,
		Timestamp: time.Now().UTC(),
		AgentID:   agentID,
		Subject:   subject,
		Details:   details,
	}

	if userID := ctx.Value("user_id"); userID != nil {
		event.UserID = fmt.Sprintf("%v", userID)
	}

	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		event.TenantID = fmt.Sprintf("%v", tenantID)
	}

	logging.Info(ctx, "audit event",
		logging.NewField("event_id", event.ID),
		logging.NewField("event_type", string(event.Type)),
		logging.NewField("agent_id", event.AgentID),
		logging.NewField("subject", event.Subject))
}

func LogTransition(ctx context.Context, agentID string, fromState string, toState string, reason string) {
	Log(ctx, AgentTransitioned, fmt.Sprintf("%s -> %s", fromState, toState), agentID, map[string]interface{}{
		"from_state": fromState,
		"to_state":   toState,
		"reason":     reason,
	})
}

func LogToolInvocation(ctx context.Context, agentID string, toolName string, success bool, duration time.Duration) {
	Log(ctx, ToolInvoked, fmt.Sprintf("agent invoked %s", toolName), agentID, map[string]interface{}{
		"tool_name": toolName,
		"success":   success,
		"duration":  duration.String(),
	})
}

func LogError(ctx context.Context, agentID string, msg string, err error) {
	Log(ctx, ErrorOccurred, msg, agentID, map[string]interface{}{
		"error": err.Error(),
	})
}
