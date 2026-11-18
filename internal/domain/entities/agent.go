package entities

import (
	"time"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

type AgentRole string

const (
	ProductAgent     AgentRole = "product"
	ArchitectAgent   AgentRole = "architect"
	DesignAgent      AgentRole = "design"
	DeveloperAgent   AgentRole = "developer"
	SecurityAgent    AgentRole = "security"
	QAAgent          AgentRole = "qa"
	PerformanceAgent AgentRole = "performance"
	DevOpsAgent      AgentRole = "devops"
	ComplianceAgent  AgentRole = "compliance"
	CoordinatorAgent AgentRole = "coordinator"
)

type LLMModel string

const (
	Claude35Sonnet LLMModel = "claude-3-5-sonnet-20241022"
	Claude3Opus    LLMModel = "claude-3-opus-20250219"
	Claude3Haiku   LLMModel = "claude-3-haiku-20250307"
)

type InteractionMode string

const (
	Autonomous  InteractionMode = "autonomous"
	Interactive InteractionMode = "interactive"
	Advisory    InteractionMode = "advisory"
)

type DecisionAuthority string

const (
	Authoritative DecisionAuthority = "authoritative"
	Suggestive    DecisionAuthority = "suggestive"
)

type AgentCapabilities struct {
	Role              AgentRole
	Title             string
	Description       string
	Capabilities      []string
	LLMModel          LLMModel
	ContextWindow     int
	ResponseFormat    string
	KnowledgeDomains  []string
	DecisionAuthority DecisionAuthority
	InteractionMode   InteractionMode
}

type Decision struct {
	AgentRole     AgentRole
	Choice        string
	Reasoning     string
	Confidence    float64
	Alternatives  []string
	Impact        map[string]interface{}
	NeedsApproval bool
	Approver      string
	CreatedAt     time.Time
}

type CoordinationSession struct {
	ID            string
	TriggerEvent  map[string]interface{}
	WorkflowDefID string
	Agents        []*AgentExecution
	Decisions     map[string]*Decision
	Status        SessionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
}

type SessionStatus string

const (
	SessionPending         SessionStatus = "pending"
	SessionCoordinating    SessionStatus = "coordinating"
	SessionWaitingApproval SessionStatus = "waiting_approval"
	SessionExecuting       SessionStatus = "executing"
	SessionCompleted       SessionStatus = "completed"
	SessionFailed          SessionStatus = "failed"
)

// CoordinationSessionMachine explicitly governs the 10-agent coordination session persisted in SQLite.
var CoordinationSessionMachine = state.NewMachine("coordination-session", state.State(SessionPending), map[state.State][]state.State{
	state.State(SessionPending):         {state.State(SessionCoordinating), state.State(SessionWaitingApproval), state.State(SessionExecuting), state.State(SessionFailed)},
	state.State(SessionCoordinating):    {state.State(SessionWaitingApproval), state.State(SessionExecuting), state.State(SessionCompleted), state.State(SessionFailed)},
	state.State(SessionWaitingApproval): {state.State(SessionExecuting), state.State(SessionCompleted), state.State(SessionFailed)},
	state.State(SessionExecuting):       {state.State(SessionCompleted), state.State(SessionFailed)},
}).WithCategory("governance").
	WithDescription("Persisted 10-agent coordination session lifecycle from intake to human gate, DAG execution, and completion.").
	WithStateDescriptions(map[state.State]string{
		state.State(SessionPending):         "Session initialized for incoming requirement.",
		state.State(SessionCoordinating):    "10 specialized agents evaluating architecture, security, and test strategy.",
		state.State(SessionWaitingApproval): "Paused waiting for human operator approval.",
		state.State(SessionExecuting):       "Approved and executing downstream DAG / sandbox build.",
		state.State(SessionCompleted):       "Coordination and execution finished cleanly.",
		state.State(SessionFailed):          "Session rejected or failed during execution.",
	})

type AgentExecution struct {
	ID          string
	SessionID   string
	AgentRole   AgentRole
	Status      string
	Decision    *Decision
	Error       error
	StartedAt   time.Time
	CompletedAt *time.Time
}

func NewCoordinationSession(triggerEvent map[string]interface{}) *CoordinationSession {
	return &CoordinationSession{
		ID:           uuid.New().String(),
		TriggerEvent: triggerEvent,
		Agents:       make([]*AgentExecution, 0),
		Decisions:    make(map[string]*Decision),
		Status:       SessionPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// TransitionTo validates and applies a status change via CoordinationSessionMachine.
func (s *CoordinationSession) TransitionTo(next SessionStatus) error {
	if s.Status == next {
		return nil
	}
	cur := state.State(s.Status)
	if err := CoordinationSessionMachine.Transition(&cur, state.State(next)); err != nil {
		return err
	}
	s.Status = SessionStatus(cur)
	now := time.Now()
	s.UpdatedAt = now
	if CoordinationSessionMachine.IsTerminal(cur) {
		s.CompletedAt = &now
	}
	return nil
}

func (s *CoordinationSession) Start() {
	_ = s.TransitionTo(SessionCoordinating)
}

func (s *CoordinationSession) RequestApproval() {
	_ = s.TransitionTo(SessionWaitingApproval)
}

func (s *CoordinationSession) BeginExecution() {
	_ = s.TransitionTo(SessionExecuting)
}

func (s *CoordinationSession) Complete() {
	_ = s.TransitionTo(SessionCompleted)
}

func (s *CoordinationSession) Fail() {
	_ = s.TransitionTo(SessionFailed)
}

func (d *Decision) HasHighConfidence() bool {
	return d.Confidence >= 0.85
}

func (d *Decision) NeedsHumanReview() bool {
	return d.Confidence < 0.7
}
