package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// WorkflowRun represents a workflow execution
type WorkflowRun struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output"`
	ErrorMessage string          `json:"error_message"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at"`
	DurationMs   int             `json:"duration_ms"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CodeMetrics represents code quality metrics
type CodeMetrics struct {
	ID                   string    `json:"id"`
	WorkflowRunID        string    `json:"workflow_run_id"`
	Language             string    `json:"language"`
	LinesOfCode          int       `json:"lines_of_code"`
	CodeScore            int       `json:"code_score"`
	BugsFound            int       `json:"bugs_found"`
	BugsFixed            int       `json:"bugs_fixed"`
	TestCoverage         float64   `json:"test_coverage"`
	ComplexityScore      int       `json:"complexity_score"`
	MaintainabilityScore int       `json:"maintainability_score"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ReviewFinding represents a code review finding
type ReviewFinding struct {
	ID             string          `json:"id"`
	WorkflowRunID  string          `json:"workflow_run_id"`
	FileName       string          `json:"file_name"`
	Score          int             `json:"score"`
	Summary        string          `json:"summary"`
	Approved       bool            `json:"approved"`
	Findings       json.RawMessage `json:"findings"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// BugDetection represents a detected bug
type BugDetection struct {
	ID            string    `json:"id"`
	WorkflowRunID string    `json:"workflow_run_id"`
	Severity      string    `json:"severity"`
	BugType       string    `json:"bug_type"`
	Location      string    `json:"location"`
	Description   string    `json:"description"`
	Impact        string    `json:"impact"`
	FixSuggestion string    `json:"fix_suggestion"`
	Confidence    int       `json:"confidence"`
	Fixed         bool      `json:"fixed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TriageResult represents a triage analysis result
type TriageResult struct {
	ID               string    `json:"id"`
	IssueNumber      int       `json:"issue_number"`
	Repository       string    `json:"repository"`
	Priority         string    `json:"priority"`
	Category         string    `json:"category"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	SuggestedAction  string    `json:"suggested_action"`
	AnalyzedAt       time.Time `json:"analyzed_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WorkflowRepository handles workflow run queries
type WorkflowRepository struct {
	db *Database
}

func NewWorkflowRepository(db *Database) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) CreateRun(run *WorkflowRun) error {
	query := `
		INSERT INTO workflow_runs (id, type, status, input, output, error_message, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.db.Exec(query, run.ID, run.Type, run.Status, run.Input, run.Output, run.ErrorMessage, run.StartedAt)
	return err
}

func (r *WorkflowRepository) UpdateRun(run *WorkflowRun) error {
	query := `
		UPDATE workflow_runs
		SET status = $1, output = $2, error_message = $3, completed_at = $4, duration_ms = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Exec(query, run.Status, run.Output, run.ErrorMessage, run.CompletedAt, run.DurationMs, run.ID)
	return err
}

func (r *WorkflowRepository) GetRun(id string) (*WorkflowRun, error) {
	query := `SELECT id, type, status, input, output, error_message, started_at, completed_at, duration_ms, created_at, updated_at FROM workflow_runs WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanWorkflowRun(row)
}

func (r *WorkflowRepository) ListRuns(limit, offset int) ([]*WorkflowRun, error) {
	query := `
		SELECT id, type, status, input, output, error_message, started_at, completed_at, duration_ms, created_at, updated_at
		FROM workflow_runs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*WorkflowRun
	for rows.Next() {
		run, err := scanWorkflowRun(rows)
		if err != nil {
			log.Printf("[DB] Error scanning workflow run: %v", err)
			continue
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *WorkflowRepository) ListByType(wfType string, limit, offset int) ([]*WorkflowRun, error) {
	query := `
		SELECT id, type, status, input, output, error_message, started_at, completed_at, duration_ms, created_at, updated_at
		FROM workflow_runs
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(query, wfType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*WorkflowRun
	for rows.Next() {
		run, err := scanWorkflowRun(rows)
		if err != nil {
			continue
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *WorkflowRepository) GetStats() (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status = 'running' THEN 1 END) as running,
			ROUND(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)))) as avg_duration
		FROM workflow_runs
	`
	var total, success, failed, running sql.NullInt64
	var avgDuration sql.NullFloat64

	err := r.db.QueryRow(query).Scan(&total, &success, &failed, &running, &avgDuration)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":       total.Int64,
		"success":     success.Int64,
		"failed":      failed.Int64,
		"running":     running.Int64,
		"avg_duration": avgDuration.Float64,
	}, nil
}

// CodeMetricsRepository handles code metrics queries
type CodeMetricsRepository struct {
	db *Database
}

func NewCodeMetricsRepository(db *Database) *CodeMetricsRepository {
	return &CodeMetricsRepository{db: db}
}

func (r *CodeMetricsRepository) Create(metrics *CodeMetrics) error {
	query := `
		INSERT INTO code_metrics (id, workflow_run_id, language, lines_of_code, code_score, bugs_found, bugs_fixed, test_coverage, complexity_score, maintainability_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`
	_, err := r.db.Exec(query,
		metrics.ID, metrics.WorkflowRunID, metrics.Language, metrics.LinesOfCode,
		metrics.CodeScore, metrics.BugsFound, metrics.BugsFixed, metrics.TestCoverage,
		metrics.ComplexityScore, metrics.MaintainabilityScore)
	return err
}

func (r *CodeMetricsRepository) GetByWorkflow(workflowID string) (*CodeMetrics, error) {
	query := `
		SELECT id, workflow_run_id, language, lines_of_code, code_score, bugs_found, bugs_fixed, test_coverage, complexity_score, maintainability_score, created_at, updated_at
		FROM code_metrics
		WHERE workflow_run_id = $1
	`
	row := r.db.QueryRow(query, workflowID)
	return scanCodeMetrics(row)
}

func (r *CodeMetricsRepository) AverageScore() (float64, error) {
	query := `SELECT COALESCE(AVG(code_score), 0) FROM code_metrics WHERE code_score > 0`
	var avg float64
	err := r.db.QueryRow(query).Scan(&avg)
	return avg, err
}

// BugRepository handles bug detection queries
type BugRepository struct {
	db *Database
}

func NewBugRepository(db *Database) *BugRepository {
	return &BugRepository{db: db}
}

func (r *BugRepository) Create(bug *BugDetection) error {
	query := `
		INSERT INTO bug_detections (id, workflow_run_id, severity, bug_type, location, description, impact, fix_suggestion, confidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := r.db.Exec(query,
		bug.ID, bug.WorkflowRunID, bug.Severity, bug.BugType, bug.Location,
		bug.Description, bug.Impact, bug.FixSuggestion, bug.Confidence)
	return err
}

func (r *BugRepository) ListByWorkflow(workflowID string) ([]*BugDetection, error) {
	query := `
		SELECT id, workflow_run_id, severity, bug_type, location, description, impact, fix_suggestion, confidence, fixed, created_at, updated_at
		FROM bug_detections
		WHERE workflow_run_id = $1
		ORDER BY severity DESC, created_at DESC
	`
	rows, err := r.db.Query(query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bugs []*BugDetection
	for rows.Next() {
		bug, err := scanBugDetection(rows)
		if err != nil {
			continue
		}
		bugs = append(bugs, bug)
	}
	return bugs, rows.Err()
}

func (r *BugRepository) GetStats() (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN fixed = true THEN 1 END) as fixed,
			COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical
		FROM bug_detections
	`
	var total, fixed, critical sql.NullInt64

	err := r.db.QueryRow(query).Scan(&total, &fixed, &critical)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":    total.Int64,
		"fixed":    fixed.Int64,
		"critical": critical.Int64,
	}, nil
}

// TriageRepository handles triage result queries
type TriageRepository struct {
	db *Database
}

func NewTriageRepository(db *Database) *TriageRepository {
	return &TriageRepository{db: db}
}

func (r *TriageRepository) Create(result *TriageResult) error {
	query := `
		INSERT INTO triage_results (id, issue_number, repository, priority, category, title, description, suggested_action, analyzed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NOW())
	`
	_, err := r.db.Exec(query,
		result.ID, result.IssueNumber, result.Repository, result.Priority, result.Category,
		result.Title, result.Description, result.SuggestedAction)
	return err
}

func (r *TriageRepository) GetStats() (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN priority = 'critical' THEN 1 END) as critical,
			COUNT(CASE WHEN priority = 'high' THEN 1 END) as high,
			COUNT(CASE WHEN priority = 'medium' THEN 1 END) as medium,
			COUNT(CASE WHEN priority = 'low' THEN 1 END) as low
		FROM triage_results
		WHERE analyzed_at > NOW() - INTERVAL '7 days'
	`
	stats := make(map[string]interface{})
	var total, critical, high, medium, low sql.NullInt64

	err := r.db.QueryRow(query).Scan(&total, &critical, &high, &medium, &low)
	if err != nil {
		return nil, err
	}

	stats["total"] = total.Int64
	stats["by_priority"] = map[string]interface{}{
		"critical": critical.Int64,
		"high":     high.Int64,
		"medium":   medium.Int64,
		"low":      low.Int64,
	}
	return stats, nil
}

// Scan helpers
func scanWorkflowRun(scanner interface {
	Scan(...interface{}) error
}) (*WorkflowRun, error) {
	run := &WorkflowRun{}
	err := scanner.Scan(
		&run.ID, &run.Type, &run.Status, &run.Input, &run.Output,
		&run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.DurationMs,
		&run.CreatedAt, &run.UpdatedAt)
	return run, err
}

func scanCodeMetrics(scanner interface {
	Scan(...interface{}) error
}) (*CodeMetrics, error) {
	m := &CodeMetrics{}
	err := scanner.Scan(
		&m.ID, &m.WorkflowRunID, &m.Language, &m.LinesOfCode, &m.CodeScore,
		&m.BugsFound, &m.BugsFixed, &m.TestCoverage, &m.ComplexityScore,
		&m.MaintainabilityScore, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func scanBugDetection(scanner interface {
	Scan(...interface{}) error
}) (*BugDetection, error) {
	bug := &BugDetection{}
	err := scanner.Scan(
		&bug.ID, &bug.WorkflowRunID, &bug.Severity, &bug.BugType, &bug.Location,
		&bug.Description, &bug.Impact, &bug.FixSuggestion, &bug.Confidence,
		&bug.Fixed, &bug.CreatedAt, &bug.UpdatedAt)
	return bug, err
}
