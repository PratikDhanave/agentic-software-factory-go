package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
)

// PostgresRepository implements domain.WorkflowRepository
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveMetric stores an agent metric
func (r *PostgresRepository) SaveMetric(ctx interface{}, metric *domain.AgentMetric) error {
	query := `
		INSERT INTO agent_metrics (id, agent_role, metric_type, metric_value, execution_id, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	_, err := r.db.ExecContext(c, query,
		uuid.New(),
		metric.AgentRole,
		"accuracy",
		metric.Confidence,
		metric.ExecutionID,
		time.Now(),
		time.Now(),
		time.Now(),
	)

	return err
}

// SaveDecision stores an agent decision
func (r *PostgresRepository) SaveDecision(ctx interface{}, decision *domain.AgentDecision) error {
	query := `
		INSERT INTO agent_decisions (id, execution_id, agent_role, decision, reasoning, confidence, approval_status, requires_approval, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	status := decision.ApprovalStatus
	if status == "" {
		status = "pending"
	}

	_, err := r.db.ExecContext(c, query,
		decision.ID,
		decision.ExecutionID,
		decision.AgentRole,
		decision.Decision,
		decision.Reasoning,
		decision.Confidence,
		status,
		decision.RequiresApproval,
		decision.CreatedAt,
	)

	return err
}

// SavePattern stores a learned pattern
func (r *PostgresRepository) SavePattern(ctx interface{}, pattern *domain.LearnedPattern) error {
	query := `
		INSERT INTO learned_patterns (id, agent_role, pattern_type, pattern_description, pattern_data, confidence, times_applied, success_rate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (agent_role, pattern_type) DO UPDATE
		SET times_applied = times_applied + 1, success_rate = $8, updated_at = $10
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	patternData := fmt.Sprintf(`{"issue_type": "%s", "confidence_base": %f}`, pattern.IssueType, pattern.ConfidenceBase)

	_, err := r.db.ExecContext(c, query,
		uuid.New(),
		pattern.AgentRole,
		pattern.PatternType,
		fmt.Sprintf("Pattern for %s issues", pattern.IssueType),
		patternData,
		pattern.SuccessRate,
		pattern.TimesApplied,
		pattern.SuccessRate,
		time.Now(),
		time.Now(),
	)

	return err
}

// SaveCalibration stores confidence calibration data
func (r *PostgresRepository) SaveCalibration(ctx interface{}, cal *domain.ConfidenceCalibration) error {
	query := `
		INSERT INTO agent_metrics (id, agent_role, metric_type, metric_value, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	_, err := r.db.ExecContext(c, query,
		uuid.New(),
		cal.AgentRole,
		"calibration_"+cal.ConfidenceBin,
		cal.Calibration,
		time.Now(),
	)

	return err
}

// SaveSpecialization stores agent specialization data
func (r *PostgresRepository) SaveSpecialization(ctx interface{}, spec *domain.AgentSpecialization) error {
	query := `
		INSERT INTO agent_metrics (id, agent_role, metric_type, metric_value, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	_, err := r.db.ExecContext(c, query,
		uuid.New(),
		spec.Agent,
		"specialization_"+spec.SpecialtyType,
		spec.SuccessRate,
		time.Now(),
	)

	return err
}

// GetRecentDecisions fetches recent decisions for an agent
func (r *PostgresRepository) GetRecentDecisions(ctx interface{}, agentRole string, limit int) ([]*domain.AgentDecision, error) {
	query := `
		SELECT id, execution_id, agent_role, decision, reasoning, confidence, approval_status, created_at
		FROM agent_decisions
		WHERE agent_role = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	rows, err := r.db.QueryContext(c, query, agentRole, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*domain.AgentDecision
	for rows.Next() {
		var d domain.AgentDecision
		var status sql.NullString

		err := rows.Scan(&d.ID, &d.ExecutionID, &d.AgentRole, &d.Decision, &d.Reasoning, &d.Confidence, &status, &d.CreatedAt)
		if err != nil {
			return nil, err
		}

		if status.Valid {
			d.ApprovalStatus = status.String
		}

		decisions = append(decisions, &d)
	}

	return decisions, rows.Err()
}

// GetDecision fetches a single decision
func (r *PostgresRepository) GetDecision(ctx interface{}, id uuid.UUID) (*domain.AgentDecision, error) {
	query := `
		SELECT id, execution_id, agent_role, decision, reasoning, confidence, approval_status, created_at
		FROM agent_decisions
		WHERE id = $1
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	var d domain.AgentDecision
	var status sql.NullString

	err := r.db.QueryRowContext(c, query, id).Scan(&d.ID, &d.ExecutionID, &d.AgentRole, &d.Decision, &d.Reasoning, &d.Confidence, &status, &d.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if status.Valid {
		d.ApprovalStatus = status.String
	}

	return &d, nil
}

// GetExecution fetches a workflow execution
func (r *PostgresRepository) GetExecution(ctx interface{}, id string) (*domain.WorkflowExecution, error) {
	query := `
		SELECT id, definition_id, issue_title, issue_description, trigger_type, trigger_id, status, created_at, updated_at
		FROM workflow_executions
		WHERE id::text = $1
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	var e domain.WorkflowExecution
	err := r.db.QueryRowContext(c, query, id).Scan(&e.ID, &e.DefinitionID, &e.IssueTitle, &e.IssueBody, &e.TriggerType, &e.TriggerID, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &e, nil
}

// GetDecisionsByAgentAndType fetches decisions by agent and issue type
func (r *PostgresRepository) GetDecisionsByAgentAndType(ctx interface{}, agent, issueType string, limit int) ([]*domain.AgentDecision, error) {
	// For now, return recent decisions for the agent
	// In production, would extract issue type from decision reasoning
	return r.GetRecentDecisions(ctx, agent, limit)
}

// FindPatterns finds learned patterns
func (r *PostgresRepository) FindPatterns(ctx interface{}, agent, issueType string) ([]*domain.LearnedPattern, error) {
	query := `
		SELECT id, agent_role, pattern_type, pattern_description, confidence, times_applied, success_rate, created_at, updated_at
		FROM learned_patterns
		WHERE agent_role = $1 AND pattern_description LIKE $2
		ORDER BY success_rate DESC
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	rows, err := r.db.QueryContext(c, query, agent, "%"+issueType+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []*domain.LearnedPattern
	for rows.Next() {
		var p domain.LearnedPattern
		err := rows.Scan(&p.ID, &p.AgentRole, &p.PatternType, &p.PatternDescription, &p.SuccessRate, &p.TimesApplied, &p.SuccessRate, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, &p)
	}

	return patterns, rows.Err()
}

// GetCalibration fetches confidence calibration
func (r *PostgresRepository) GetCalibration(ctx interface{}, agent string, confidence float64) (*domain.ConfidenceCalibration, error) {
	// Simplified: return calibration based on confidence bin
	return &domain.ConfidenceCalibration{
		AgentRole:     agent,
		ConfidenceBin: fmt.Sprintf("%.1f-%.1f", confidence-0.05, confidence+0.05),
		Calibration:   1.0,
	}, nil
}

// GetTopPatterns fetches the most successful patterns
func (r *PostgresRepository) GetTopPatterns(ctx interface{}, limit int) ([]*domain.LearnedPattern, error) {
	query := `
		SELECT id, agent_role, pattern_type, pattern_description, confidence, times_applied, success_rate, created_at, updated_at
		FROM learned_patterns
		ORDER BY success_rate DESC
		LIMIT $1
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	rows, err := r.db.QueryContext(c, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []*domain.LearnedPattern
	for rows.Next() {
		var p domain.LearnedPattern
		err := rows.Scan(&p.ID, &p.AgentRole, &p.PatternType, &p.PatternDescription, &p.SuccessRate, &p.TimesApplied, &p.SuccessRate, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, &p)
	}

	return patterns, rows.Err()
}

// CountRecentDecisions counts recent decisions
func (r *PostgresRepository) CountRecentDecisions(ctx interface{}, agent string, limit int) (int, error) {
	query := `
		SELECT COUNT(*) FROM agent_decisions
		WHERE agent_role = $1 AND created_at > NOW() - INTERVAL '24 hours'
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	var count int
	err := r.db.QueryRowContext(c, query, agent).Scan(&count)
	return count, err
}

// MarkDecisionCorrect marks a decision as correct or incorrect
func (r *PostgresRepository) MarkDecisionCorrect(ctx interface{}, id uuid.UUID, correct bool) error {
	query := `
		UPDATE agent_decisions
		SET approval_status = $1, updated_at = $2
		WHERE id = $3
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	status := "rejected"
	if correct {
		status = "approved"
	}

	_, err := r.db.ExecContext(c, query, status, time.Now(), id)
	return err
}

// FindExecutionByPRURL finds execution by PR URL
func (r *PostgresRepository) FindExecutionByPRURL(ctx interface{}, url string) (*domain.WorkflowExecution, error) {
	query := `
		SELECT id, definition_id, issue_title, issue_description, trigger_type, trigger_id, status, created_at, updated_at
		FROM workflow_executions
		WHERE trigger_id LIKE $1
		LIMIT 1
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	var e domain.WorkflowExecution
	err := r.db.QueryRowContext(c, query, "%"+url+"%").Scan(&e.ID, &e.DefinitionID, &e.IssueTitle, &e.IssueBody, &e.TriggerType, &e.TriggerID, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &e, nil
}

// RecordMetric records a metric (alias for SaveMetric)
func (r *PostgresRepository) RecordMetric(ctx interface{}, metric *domain.AgentMetric) error {
	return r.SaveMetric(ctx, metric)
}

// FindDecisionsForIssue finds decisions for a specific issue
func (r *PostgresRepository) FindDecisionsForIssue(ctx interface{}, issueNumber int) ([]*domain.AgentDecision, error) {
	query := `
		SELECT d.id, d.execution_id, d.agent_role, d.decision, d.reasoning, d.confidence, d.approval_status, d.created_at
		FROM agent_decisions d
		JOIN workflow_executions e ON d.execution_id = e.id
		WHERE e.trigger_id LIKE $1
		ORDER BY d.created_at DESC
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	rows, err := r.db.QueryContext(c, query, fmt.Sprintf("%%%d%%", issueNumber))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*domain.AgentDecision
	for rows.Next() {
		var d domain.AgentDecision
		var status sql.NullString

		err := rows.Scan(&d.ID, &d.ExecutionID, &d.AgentRole, &d.Decision, &d.Reasoning, &d.Confidence, &status, &d.CreatedAt)
		if err != nil {
			return nil, err
		}

		if status.Valid {
			d.ApprovalStatus = status.String
		}

		decisions = append(decisions, &d)
	}

	return decisions, rows.Err()
}

// MarkExecutionSuccessful marks an execution as successful or failed
func (r *PostgresRepository) MarkExecutionSuccessful(ctx interface{}, id uuid.UUID, successful bool) error {
	query := `
		UPDATE workflow_executions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	status := "failed"
	if successful {
		status = "completed"
	}

	_, err := r.db.ExecContext(c, query, status, time.Now(), id)
	return err
}
