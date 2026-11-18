package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
)

// LearningRepository implements domain.LearningRepository (domain.WorkflowRepository) in SQLite.
type LearningRepository struct {
	db *sql.DB
}

var _ domain.LearningRepository = (*LearningRepository)(nil)

func NewLearningRepository(store *Store) *LearningRepository {
	return &LearningRepository{db: store.DB()}
}

func (r *LearningRepository) SaveDecision(ctx context.Context, d *domain.AgentDecision) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	if d.IssueType == "" {
		d.IssueType = "general"
	}
	if d.ApprovalStatus == "" {
		d.ApprovalStatus = "pending"
	}
	altsJSON, _ := json.Marshal(d.Alternatives)
	reqApproval := 0
	if d.RequiresApproval {
		reqApproval = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_decisions (id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			confidence = excluded.confidence,
			requires_approval = excluded.requires_approval,
			approval_status = excluded.approval_status,
			updated_at = excluded.updated_at
	`, d.ID.String(), d.ExecutionID.String(), d.AgentRole, d.IssueType, d.IssueNumber, d.Decision, d.Reasoning, d.Confidence, string(altsJSON), reqApproval, d.ApprovalStatus, d.CreatedAt, d.UpdatedAt)
	return err
}

func (r *LearningRepository) SaveApproval(ctx context.Context, a *domain.Approval) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.ApprovalTimestamp.IsZero() {
		a.ApprovalTimestamp = now
	}
	a.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO approvals (id, execution_id, decision_id, agent_role, status, approver_id, comments, approved_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			approver_id = excluded.approver_id,
			comments = excluded.comments,
			updated_at = excluded.updated_at
	`, a.ID.String(), a.ExecutionID.String(), a.DecisionID.String(), a.AgentRole, a.Status, a.ApproverID, a.ApprovalComments, a.ApprovalTimestamp, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *LearningRepository) SaveMetric(ctx context.Context, m *domain.AgentMetric) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	correct := 0
	if m.DecisionCorrect {
		correct = 1
	}
	reqApp := 0
	if m.ApprovalRequired {
		reqApp = 1
	}
	human := 0
	if m.ApprovedByHuman {
		human = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO agent_metrics (id, agent_role, execution_id, decision_correct, confidence, approval_required, approved_by_human, issue_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID.String(), m.AgentRole, m.ExecutionID.String(), correct, m.Confidence, reqApp, human, m.IssueType, m.CreatedAt)
	return err
}

func (r *LearningRepository) RecordMetric(ctx context.Context, m *domain.AgentMetric) error {
	return r.SaveMetric(ctx, m)
}

func (r *LearningRepository) SavePattern(ctx context.Context, p *domain.LearnedPattern) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO learned_patterns (id, agent_role, pattern_type, issue_type, description, success_rate, times_applied, confidence_base, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_role, pattern_type, issue_type) DO UPDATE SET
			success_rate = excluded.success_rate,
			times_applied = excluded.times_applied,
			confidence_base = excluded.confidence_base,
			updated_at = excluded.updated_at
	`, p.ID.String(), p.AgentRole, p.PatternType, p.IssueType, p.PatternDescription, p.SuccessRate, p.TimesApplied, p.ConfidenceBase, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *LearningRepository) SaveCalibration(ctx context.Context, cal *domain.ConfidenceCalibration) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO confidence_calibrations (agent_role, confidence_bin, calibration, sample_count, actual_rate, predicted_rate, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_role, confidence_bin) DO UPDATE SET
			calibration = excluded.calibration,
			sample_count = excluded.sample_count,
			actual_rate = excluded.actual_rate,
			predicted_rate = excluded.predicted_rate,
			updated_at = excluded.updated_at
	`, cal.AgentRole, cal.ConfidenceBin, cal.Calibration, cal.SampleCount, cal.ActualRate, cal.PredictedRate, time.Now())
	return err
}

func (r *LearningRepository) SaveSpecialization(ctx context.Context, spec *domain.AgentSpecialization) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_specializations (agent, specialty_type, success_rate, avg_confidence, decision_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent, specialty_type) DO UPDATE SET
			success_rate = excluded.success_rate,
			avg_confidence = excluded.avg_confidence,
			decision_count = excluded.decision_count,
			updated_at = excluded.updated_at
	`, spec.Agent, spec.SpecialtyType, spec.SuccessRate, spec.AverageConfidence, spec.DecisionCount, time.Now())
	return err
}

func (r *LearningRepository) GetRecentDecisions(ctx context.Context, agentRole string, limit int) ([]*domain.AgentDecision, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at
		FROM agent_decisions WHERE LOWER(agent_role) = LOWER(?) ORDER BY created_at DESC LIMIT ?
	`, agentRole, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentDecisions(rows)
}

func (r *LearningRepository) GetDecision(ctx context.Context, id uuid.UUID) (*domain.AgentDecision, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at
		FROM agent_decisions WHERE id = ?
	`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanAgentDecisions(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

func (r *LearningRepository) GetExecution(ctx context.Context, id string) (*domain.WorkflowExecution, error) {
	parsedID, _ := uuid.Parse(id)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at
		FROM agent_decisions WHERE execution_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	decisions, _ := scanAgentDecisions(rows)
	if len(decisions) == 0 {
		return nil, nil
	}
	return &domain.WorkflowExecution{
		ID:        parsedID,
		Decisions: decisions,
	}, nil
}

func (r *LearningRepository) GetDecisionsByAgentAndType(ctx context.Context, agent, issueType string, limit int) ([]*domain.AgentDecision, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at
		FROM agent_decisions WHERE LOWER(agent_role) = LOWER(?) AND LOWER(issue_type) = LOWER(?) ORDER BY created_at DESC LIMIT ?
	`, agent, issueType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentDecisions(rows)
}

func (r *LearningRepository) FindPatterns(ctx context.Context, agent, issueType string) ([]*domain.LearnedPattern, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, agent_role, pattern_type, issue_type, description, success_rate, times_applied, confidence_base, created_at, updated_at
		FROM learned_patterns WHERE LOWER(agent_role) = LOWER(?) AND LOWER(issue_type) = LOWER(?) ORDER BY success_rate DESC
	`, agent, issueType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPatterns(rows)
}

func (r *LearningRepository) GetTopPatterns(ctx context.Context, limit int) ([]*domain.LearnedPattern, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, agent_role, pattern_type, issue_type, description, success_rate, times_applied, confidence_base, created_at, updated_at
		FROM learned_patterns ORDER BY success_rate DESC, times_applied DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPatterns(rows)
}

func (r *LearningRepository) GetCalibration(ctx context.Context, agent string, confidence float64) (*domain.ConfidenceCalibration, error) {
	bin := confidenceBinFor(confidence)
	row := r.db.QueryRowContext(ctx, `
		SELECT agent_role, confidence_bin, calibration, sample_count, actual_rate, predicted_rate, updated_at
		FROM confidence_calibrations WHERE LOWER(agent_role) = LOWER(?) AND confidence_bin = ?
	`, agent, bin)
	var cal domain.ConfidenceCalibration
	if err := row.Scan(&cal.AgentRole, &cal.ConfidenceBin, &cal.Calibration, &cal.SampleCount, &cal.ActualRate, &cal.PredictedRate, &cal.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cal, nil
}

func (r *LearningRepository) CountRecentDecisions(ctx context.Context, agent string, limit int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (SELECT 1 FROM agent_decisions WHERE LOWER(agent_role) = LOWER(?) ORDER BY created_at DESC LIMIT ?)
	`, agent, limit).Scan(&count)
	return count, err
}

func (r *LearningRepository) MarkDecisionCorrect(ctx context.Context, id uuid.UUID, correct bool) error {
	status := "rejected"
	correctInt := 0
	if correct {
		status = "approved"
		correctInt = 1
	}
	_, _ = r.db.ExecContext(ctx, `UPDATE agent_metrics SET decision_correct = ? WHERE execution_id = (SELECT execution_id FROM agent_decisions WHERE id = ?)`, correctInt, id.String())
	_, err := r.db.ExecContext(ctx, `UPDATE agent_decisions SET approval_status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), id.String())
	return err
}

func (r *LearningRepository) FindExecutionByPRURL(ctx context.Context, url string) (*domain.WorkflowExecution, error) {
	var idStr string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM workflow_executions WHERE pr_url = ? LIMIT 1`, url).Scan(&idStr)
	if err != nil {
		return nil, nil
	}
	return r.GetExecution(ctx, idStr)
}

func (r *LearningRepository) FindDecisionsForIssue(ctx context.Context, issueNumber int) ([]*domain.AgentDecision, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, agent_role, issue_type, issue_number, decision, reasoning, confidence, alternatives_json, requires_approval, approval_status, created_at, updated_at
		FROM agent_decisions WHERE issue_number = ?
	`, issueNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentDecisions(rows)
}

func (r *LearningRepository) MarkExecutionSuccessful(ctx context.Context, id uuid.UUID, successful bool) error {
	status := "failed"
	if successful {
		status = "completed"
	}
	_, err := r.db.ExecContext(ctx, `UPDATE workflow_executions SET status = ? WHERE id = ?`, status, id.String())
	return err
}

func scanAgentDecisions(rows *sql.Rows) ([]*domain.AgentDecision, error) {
	var out []*domain.AgentDecision
	for rows.Next() {
		var d domain.AgentDecision
		var idStr, execStr, altsJSON string
		var reqApp int
		if err := rows.Scan(&idStr, &execStr, &d.AgentRole, &d.IssueType, &d.IssueNumber, &d.Decision, &d.Reasoning, &d.Confidence, &altsJSON, &reqApp, &d.ApprovalStatus, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.ID, _ = uuid.Parse(idStr)
		d.ExecutionID, _ = uuid.Parse(execStr)
		d.RequiresApproval = reqApp == 1
		_ = json.Unmarshal([]byte(altsJSON), &d.Alternatives)
		out = append(out, &d)
	}
	return out, rows.Err()
}

func scanPatterns(rows *sql.Rows) ([]*domain.LearnedPattern, error) {
	var out []*domain.LearnedPattern
	for rows.Next() {
		var p domain.LearnedPattern
		var idStr string
		if err := rows.Scan(&idStr, &p.AgentRole, &p.PatternType, &p.IssueType, &p.PatternDescription, &p.SuccessRate, &p.TimesApplied, &p.ConfidenceBase, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ID, _ = uuid.Parse(idStr)
		out = append(out, &p)
	}
	return out, rows.Err()
}

func confidenceBinFor(confidence float64) string {
	switch {
	case confidence < 0.70:
		return "0.60-0.70"
	case confidence < 0.75:
		return "0.70-0.75"
	case confidence < 0.80:
		return "0.75-0.80"
	case confidence < 0.85:
		return "0.80-0.85"
	case confidence < 0.90:
		return "0.85-0.90"
	case confidence < 0.95:
		return "0.90-0.95"
	default:
		return "0.95-1.00"
	}
}
