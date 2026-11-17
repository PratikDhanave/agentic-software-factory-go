package db

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// RecordWorkflow saves a workflow execution to the database
func (db *Database) RecordWorkflow(wfType, status string, input, output interface{}) (string, error) {
	repo := NewWorkflowRepository(db)
	run := &WorkflowRun{
		ID:        uuid.New().String(),
		Type:      wfType,
		Status:    status,
		StartedAt: time.Now(),
	}

	if input != nil {
		data, _ := json.Marshal(input)
		run.Input = data
	}

	if output != nil {
		data, _ := json.Marshal(output)
		run.Output = data
	}

	if err := repo.CreateRun(run); err != nil {
		return "", fmt.Errorf("failed to record workflow: %w", err)
	}

	log.Printf("[DB] Recorded workflow: %s (%s)", run.ID, wfType)
	return run.ID, nil
}

// UpdateWorkflow updates a workflow execution status
func (db *Database) UpdateWorkflow(id, status string, output interface{}, errorMsg string) error {
	repo := NewWorkflowRepository(db)
	run, err := repo.GetRun(id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	run.Status = status
	now := time.Now()
	run.CompletedAt = &now

	if run.StartedAt != (time.Time{}) {
		duration := now.Sub(run.StartedAt)
		run.DurationMs = int(duration.Milliseconds())
	}

	if output != nil {
		data, _ := json.Marshal(output)
		run.Output = data
	}

	if errorMsg != "" {
		run.ErrorMessage = errorMsg
	}

	if err := repo.UpdateRun(run); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	log.Printf("[DB] Updated workflow: %s -> %s", id, status)
	return nil
}

// RecordCodeMetrics saves code quality metrics
func (db *Database) RecordCodeMetrics(workflowID, language string, metrics map[string]interface{}) error {
	repo := NewCodeMetricsRepository(db)
	m := &CodeMetrics{
		ID:            uuid.New().String(),
		WorkflowRunID: workflowID,
		Language:      language,
	}

	// Extract metrics from map
	if loc, ok := metrics["lines_of_code"].(int); ok {
		m.LinesOfCode = loc
	}
	if score, ok := metrics["code_score"].(int); ok {
		m.CodeScore = score
	}
	if bugs, ok := metrics["bugs_found"].(int); ok {
		m.BugsFound = bugs
	}
	if fixed, ok := metrics["bugs_fixed"].(int); ok {
		m.BugsFixed = fixed
	}
	if coverage, ok := metrics["test_coverage"].(float64); ok {
		m.TestCoverage = coverage
	}

	if err := repo.Create(m); err != nil {
		return fmt.Errorf("failed to record code metrics: %w", err)
	}

	log.Printf("[DB] Recorded code metrics for workflow: %s", workflowID)
	return nil
}

// RecordBug saves a detected bug
func (db *Database) RecordBug(workflowID string, bug map[string]interface{}) error {
	repo := NewBugRepository(db)
	b := &BugDetection{
		ID:            uuid.New().String(),
		WorkflowRunID: workflowID,
	}

	// Extract bug details
	if severity, ok := bug["severity"].(string); ok {
		b.Severity = severity
	}
	if bugType, ok := bug["type"].(string); ok {
		b.BugType = bugType
	}
	if location, ok := bug["location"].(string); ok {
		b.Location = location
	}
	if description, ok := bug["description"].(string); ok {
		b.Description = description
	}
	if impact, ok := bug["impact"].(string); ok {
		b.Impact = impact
	}
	if fix, ok := bug["fix_suggestion"].(string); ok {
		b.FixSuggestion = fix
	}
	if confidence, ok := bug["confidence"].(int); ok {
		b.Confidence = confidence
	}

	if err := repo.Create(b); err != nil {
		return fmt.Errorf("failed to record bug: %w", err)
	}

	return nil
}

// RecordTriageResult saves a triage analysis result
func (db *Database) RecordTriageResult(issue int, repo, priority, category, title, description, action string) error {
	triageRepo := NewTriageRepository(db)
	result := &TriageResult{
		ID:              uuid.New().String(),
		IssueNumber:     issue,
		Repository:      repo,
		Priority:        priority,
		Category:        category,
		Title:           title,
		Description:     description,
		SuggestedAction: action,
		AnalyzedAt:      time.Now(),
	}

	if err := triageRepo.Create(result); err != nil {
		return fmt.Errorf("failed to record triage result: %w", err)
	}

	log.Printf("[DB] Recorded triage result: %s#%d", repo, issue)
	return nil
}

// GetDashboardStats retrieves comprehensive dashboard statistics
func (db *Database) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Workflow stats
	wfRepo := NewWorkflowRepository(db)
	wfStats, err := wfRepo.GetStats()
	if err == nil {
		stats["workflow"] = wfStats
	}

	// Bug stats
	bugRepo := NewBugRepository(db)
	bugStats, err := bugRepo.GetStats()
	if err == nil {
		stats["bugs"] = bugStats
	}

	// Triage stats
	triageRepo := NewTriageRepository(db)
	triageStats, err := triageRepo.GetStats()
	if err == nil {
		stats["triage"] = triageStats
	}

	// Code metrics
	metricsRepo := NewCodeMetricsRepository(db)
	avgScore, err := metricsRepo.AverageScore()
	if err == nil {
		stats["avg_code_score"] = avgScore
	}

	// Recent workflows
	runs, err := wfRepo.ListRuns(5, 0)
	if err == nil {
		recentWorkflows := make([]map[string]interface{}, 0, len(runs))
		for _, run := range runs {
			recentWorkflows = append(recentWorkflows, map[string]interface{}{
				"id":       run.ID,
				"type":     run.Type,
				"status":   run.Status,
				"duration": run.DurationMs,
			})
		}
		stats["recent_workflows"] = recentWorkflows
	}

	log.Println("[DB] Dashboard stats retrieved")
	return stats, nil
}

// GetWorkflowHistory retrieves historical workflow data
func (db *Database) GetWorkflowHistory(limit, offset int) ([]*WorkflowRun, error) {
	repo := NewWorkflowRepository(db)
	return repo.ListRuns(limit, offset)
}

// GetWorkflowByType retrieves workflows of a specific type
func (db *Database) GetWorkflowByType(wfType string, limit, offset int) ([]*WorkflowRun, error) {
	repo := NewWorkflowRepository(db)
	return repo.ListByType(wfType, limit, offset)
}

// GetWorkflowDetails retrieves full details of a workflow including associated data
func (db *Database) GetWorkflowDetails(workflowID string) (map[string]interface{}, error) {
	wfRepo := NewWorkflowRepository(db)
	run, err := wfRepo.GetRun(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	details := map[string]interface{}{
		"workflow": run,
	}

	// Get associated metrics
	metricsRepo := NewCodeMetricsRepository(db)
	if metrics, err := metricsRepo.GetByWorkflow(workflowID); err == nil {
		details["metrics"] = metrics
	}

	// Get associated bugs
	bugRepo := NewBugRepository(db)
	if bugs, err := bugRepo.ListByWorkflow(workflowID); err == nil {
		details["bugs"] = bugs
	}

	return details, nil
}
