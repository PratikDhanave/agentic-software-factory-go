package storage

import "time"

type WorkflowRun struct {
	ID        string    `json:"id" db:"id"`
	Type      string    `json:"type" db:"type"` // spec-to-code, code-review, etc
	Status    string    `json:"status" db:"status"`
	Input     string    `json:"input" db:"input"`
	Output    string    `json:"output" db:"output"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Duration  int       `json:"duration_ms" db:"duration_ms"`
}

type CodeMetrics struct {
	ID           string    `json:"id" db:"id"`
	WorkflowID   string    `json:"workflow_id" db:"workflow_id"`
	CodeScore    int       `json:"code_score" db:"code_score"`
	BugsFound    int       `json:"bugs_found" db:"bugs_found"`
	BugsFixed    int       `json:"bugs_fixed" db:"bugs_fixed"`
	TestCoverage int       `json:"test_coverage" db:"test_coverage"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
