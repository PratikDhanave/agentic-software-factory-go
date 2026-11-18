package domain

import (
	"context"
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
	ID                 uuid.UUID
	AgentRole          string
	PatternType        string
	IssueType          string
	PatternDescription string
	SuccessRate        float64
	TimesApplied       int
	ConfidenceBase     float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ConfidenceCalibration tracks how well agent confidence scores match reality
type ConfidenceCalibration struct {
	ID            uuid.UUID
	AgentRole     string
	ConfidenceBin string
	Calibration   float64
	SampleCount   int
	ActualRate    float64
	PredictedRate float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
	IssueType        string
	IssueNumber      int
	Decision         string
	Reasoning        string
	Confidence       float64
	Alternatives     []string
	RequiresApproval bool
	ApprovalStatus   string // pending, approved, rejected, correct, incorrect
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Approval represents a human approval decision
type Approval struct {
	ID                uuid.UUID
	ExecutionID       uuid.UUID
	DecisionID        uuid.UUID
	AgentRole         string
	Status            string
	ApproverID        string
	ApprovalComments  string
	ApprovalTimestamp time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// WorkflowExecution represents a learning-tracked workflow run
type WorkflowExecution struct {
	ID           uuid.UUID
	DefinitionID uuid.UUID
	IssueTitle   string
	IssueBody    string
	TriggerType  string
	TriggerID    string
	PRURL        string
	Status       string
	Decisions    []*AgentDecision
	Approvals    []*Approval
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// LearningRepository persists agent decisions, human approvals, confidence
// calibrations, and discovered patterns in SQLite.
type LearningRepository interface {
	SaveDecision(ctx context.Context, decision *AgentDecision) error
	SaveApproval(ctx context.Context, approval *Approval) error
	SaveMetric(ctx context.Context, metric *AgentMetric) error
	SavePattern(ctx context.Context, pattern *LearnedPattern) error
	SaveCalibration(ctx context.Context, cal *ConfidenceCalibration) error
	SaveSpecialization(ctx context.Context, spec *AgentSpecialization) error

	GetRecentDecisions(ctx context.Context, agentRole string, limit int) ([]*AgentDecision, error)
	GetDecision(ctx context.Context, id uuid.UUID) (*AgentDecision, error)
	GetExecution(ctx context.Context, id string) (*WorkflowExecution, error)
	GetDecisionsByAgentAndType(ctx context.Context, agent, issueType string, limit int) ([]*AgentDecision, error)

	FindPatterns(ctx context.Context, agent, issueType string) ([]*LearnedPattern, error)
	GetCalibration(ctx context.Context, agent string, confidence float64) (*ConfidenceCalibration, error)
	GetTopPatterns(ctx context.Context, limit int) ([]*LearnedPattern, error)

	CountRecentDecisions(ctx context.Context, agent string, limit int) (int, error)
	MarkDecisionCorrect(ctx context.Context, id uuid.UUID, correct bool) error
	FindExecutionByPRURL(ctx context.Context, url string) (*WorkflowExecution, error)
	RecordMetric(ctx context.Context, metric *AgentMetric) error

	FindDecisionsForIssue(ctx context.Context, issueNumber int) ([]*AgentDecision, error)
	MarkExecutionSuccessful(ctx context.Context, id uuid.UUID, successful bool) error
}

// WorkflowRepository aliases LearningRepository for backward compatibility with SelfImprovingService.
type WorkflowRepository = LearningRepository
