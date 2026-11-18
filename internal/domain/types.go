package domain

import (
	"time"

	"github.com/google/uuid"
)

// AgentMetric tracks agent performance for learning
type AgentMetric struct {
	ID               uuid.UUID
	AgentRole        string
	ExecutionID      uuid.UUID
	DecisionCorrect  bool
	Confidence       float64
	ApprovalRequired bool
	ApprovedByHuman  bool
	IssueType        string
	IssueComplexity  string
	TimeToDecision   int64
	PatternMatches   []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// LearnedPattern represents discovered patterns for self-improvement
type LearnedPattern struct {
	ID               uuid.UUID
	AgentRole        string
	PatternType      string
	IssueType        string
	PatternDescription string
	SuccessRate      float64
	TimesApplied     int
	ConfidenceBase   float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ConfidenceCalibration tracks how well agent confidence scores match reality
type ConfidenceCalibration struct {
	ID             uuid.UUID
	AgentRole      string
	ConfidenceBin  string
	Calibration    float64
	SampleCount    int
	ActualRate     float64
	PredictedRate  float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentSpecialization tracks what each agent is best at
type AgentSpecialization struct {
	ID                uuid.UUID
	Agent             string
	SpecialtyType     string
	SuccessRate       float64
	AverageConfidence float64
	DecisionCount     int
	Priority          float64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// AgentDecision represents a single agent decision
type AgentDecision struct {
	ID               uuid.UUID
	ExecutionID      uuid.UUID
	AgentRole        string
	Decision         string
	Reasoning        string
	Confidence       float64
	Alternatives     []string
	RequiresApproval bool
	ApprovalStatus   string // pending, approved, rejected
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Approval represents a human approval decision
type Approval struct {
	ID                 uuid.UUID
	ExecutionID        uuid.UUID
	DecisionID         uuid.UUID
	AgentRole          string
	Status             string
	ApproverID         string
	ApprovalComments   string
	ApprovalTimestamp  time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// WorkflowExecution represents a workflow run
type WorkflowExecution struct {
	ID           uuid.UUID
	DefinitionID uuid.UUID
	IssueTitle   string
	IssueBody    string
	TriggerType  string
	TriggerID    string
	Status       string
	Decisions    []*AgentDecision
	Approvals    []*Approval
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WorkflowRepository interface for persistence
type WorkflowRepository interface {
	SaveMetric(ctx interface{}, metric *AgentMetric) error
	SavePattern(ctx interface{}, pattern *LearnedPattern) error
	SaveCalibration(ctx interface{}, cal *ConfidenceCalibration) error
	SaveSpecialization(ctx interface{}, spec *AgentSpecialization) error

	GetRecentDecisions(ctx interface{}, agentRole string, limit int) ([]*AgentDecision, error)
	GetDecision(ctx interface{}, id uuid.UUID) (*AgentDecision, error)
	GetExecution(ctx interface{}, id string) (*WorkflowExecution, error)
	GetDecisionsByAgentAndType(ctx interface{}, agent, issueType string, limit int) ([]*AgentDecision, error)

	FindPatterns(ctx interface{}, agent, issueType string) ([]*LearnedPattern, error)
	GetCalibration(ctx interface{}, agent string, confidence float64) (*ConfidenceCalibration, error)
	GetTopPatterns(ctx interface{}, limit int) ([]*LearnedPattern, error)

	CountRecentDecisions(ctx interface{}, agent string, limit int) (int, error)
	MarkDecisionCorrect(ctx interface{}, id uuid.UUID, correct bool) error
	FindExecutionByPRURL(ctx interface{}, url string) (*WorkflowExecution, error)
	RecordMetric(ctx interface{}, metric *AgentMetric) error

	FindDecisionsForIssue(ctx interface{}, issueNumber int) ([]*AgentDecision, error)
	MarkExecutionSuccessful(ctx interface{}, id uuid.UUID, successful bool) error
}
