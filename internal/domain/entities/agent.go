package entities

import (
	"time"

	"github.com/google/uuid"
)

type AgentRole string

const (
	ProductAgent    AgentRole = "product"
	ArchitectAgent  AgentRole = "architect"
	DesignAgent     AgentRole = "design"
	DeveloperAgent  AgentRole = "developer"
	SecurityAgent   AgentRole = "security"
	QAAgent         AgentRole = "qa"
	PerformanceAgent AgentRole = "performance"
	DevOpsAgent     AgentRole = "devops"
	ComplianceAgent AgentRole = "compliance"
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
	Autonomous   InteractionMode = "autonomous"
	Interactive  InteractionMode = "interactive"
	Advisory     InteractionMode = "advisory"
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
	AgentRole      AgentRole
	Choice         string
	Reasoning      string
	Confidence     float64
	Alternatives   []string
	Impact         map[string]interface{}
	NeedsApproval  bool
	Approver       string
	CreatedAt      time.Time
}

type CoordinationSession struct {
	ID                string
	TriggerEvent      map[string]interface{}
	WorkflowDefID     string
	Agents            []*AgentExecution
	Decisions         map[string]*Decision
	Status            SessionStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
}

type SessionStatus string

const (
	SessionPending     SessionStatus = "pending"
	SessionCoordinating SessionStatus = "coordinating"
	SessionWaitingApproval SessionStatus = "waiting_approval"
	SessionExecuting   SessionStatus = "executing"
	SessionCompleted   SessionStatus = "completed"
	SessionFailed      SessionStatus = "failed"
)

type AgentExecution struct {
	ID            string
	SessionID     string
	AgentRole     AgentRole
	Status        string
	Decision      *Decision
	Error         error
	StartedAt     time.Time
	CompletedAt   *time.Time
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

func (s *CoordinationSession) Start() {
	s.Status = SessionCoordinating
	s.UpdatedAt = time.Now()
}

func (s *CoordinationSession) RequestApproval() {
	s.Status = SessionWaitingApproval
	s.UpdatedAt = time.Now()
}

func (s *CoordinationSession) Complete() {
	s.Status = SessionCompleted
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

func (s *CoordinationSession) Fail() {
	s.Status = SessionFailed
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

func (d *Decision) HasHighConfidence() bool {
	return d.Confidence >= 0.85
}

func (d *Decision) NeedsHumanReview() bool {
	return d.Confidence < 0.7
}
