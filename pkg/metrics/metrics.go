package metrics

import (
	"log"
	"sync"
	"time"
)

// Metrics tracks factory performance metrics
type Metrics struct {
	// Workflow metrics
	WorkflowsStarted   int64
	WorkflowsCompleted int64
	WorkflowsFailed    int64
	TotalDuration      int64 // milliseconds

	// Code metrics
	BugsDetected int64
	BugsFixed    int64
	CodeScoreSum int64

	// GitHub metrics
	IssuesAnalyzed int64
	PRsReviewed    int64

	// API metrics
	RequestsTotal      int64
	RequestsError      int64
	TotalResponseTime  int64 // milliseconds

	// Database metrics
	QueriesExecuted int64
	QueryErrors     int64
	TotalQueryTime  int64 // milliseconds

	// System metrics
	ActiveConnections int64
	MemoryUsage       int64
	CPUUsage          float64

	mu sync.RWMutex
}

// WorkflowMetric represents a single workflow's metrics
type WorkflowMetric struct {
	WorkflowID    string
	Type          string
	Status        string
	StartedAt     time.Time
	CompletedAt   *time.Time
	Duration      time.Duration
	BugsFound     int
	CodeScore     int
	ReviewScore   int
	Success       bool
}

// RequestMetric represents a single request's metrics
type RequestMetric struct {
	Method       string
	Endpoint     string
	Status       int
	ResponseTime time.Duration
	Error        string
	Timestamp    time.Time
}

// QueryMetric represents a single query's metrics
type QueryMetric struct {
	Query       string
	ExecutedAt  time.Time
	Duration    time.Duration
	RowsAffected int64
	Error       string
}

// MetricsCollector collects and aggregates metrics
type MetricsCollector struct {
	metrics       *Metrics
	workflows     []*WorkflowMetric
	requests      []*RequestMetric
	queries       []*QueryMetric
	maxHistory    int
	mu            sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(maxHistory int) *MetricsCollector {
	if maxHistory < 100 {
		maxHistory = 100
	}
	if maxHistory > 10000 {
		maxHistory = 10000
	}

	return &MetricsCollector{
		metrics:    &Metrics{},
		workflows:  make([]*WorkflowMetric, 0, maxHistory),
		requests:   make([]*RequestMetric, 0, maxHistory),
		queries:    make([]*QueryMetric, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// RecordWorkflowStart records the start of a workflow
func (mc *MetricsCollector) RecordWorkflowStart(workflowID, wfType string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.WorkflowsStarted++
	mc.workflows = append(mc.workflows, &WorkflowMetric{
		WorkflowID: workflowID,
		Type:       wfType,
		Status:     "running",
		StartedAt:  time.Now(),
	})

	mc.trimHistory()
}

// RecordWorkflowCompletion records the completion of a workflow
func (mc *MetricsCollector) RecordWorkflowCompletion(workflowID string, success bool, duration time.Duration, bugsFound, codeScore, reviewScore int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	mc.metrics.TotalDuration += duration.Milliseconds()

	if success {
		mc.metrics.WorkflowsCompleted++
		mc.metrics.CodeScoreSum += int64(codeScore)
	} else {
		mc.metrics.WorkflowsFailed++
	}

	mc.metrics.BugsDetected += int64(bugsFound)

	// Find and update workflow metric
	for _, wm := range mc.workflows {
		if wm.WorkflowID == workflowID {
			wm.CompletedAt = &now
			wm.Duration = duration
			wm.BugsFound = bugsFound
			wm.CodeScore = codeScore
			wm.ReviewScore = reviewScore
			wm.Success = success
			if success {
				wm.Status = "success"
			} else {
				wm.Status = "failed"
			}
			break
		}
	}

	log.Printf("[Metrics] Workflow completed: %s (duration: %v, bugs: %d, score: %d)", workflowID, duration, bugsFound, codeScore)
}

// RecordRequest records an HTTP request
func (mc *MetricsCollector) RecordRequest(method, endpoint string, status int, duration time.Duration, err string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.RequestsTotal++
	mc.metrics.TotalResponseTime += duration.Milliseconds()

	if status >= 400 {
		mc.metrics.RequestsError++
	}

	mc.requests = append(mc.requests, &RequestMetric{
		Method:       method,
		Endpoint:     endpoint,
		Status:       status,
		ResponseTime: duration,
		Error:        err,
		Timestamp:    time.Now(),
	})

	mc.trimHistory()
}

// RecordQuery records a database query
func (mc *MetricsCollector) RecordQuery(query string, duration time.Duration, rowsAffected int64, err string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.QueriesExecuted++
	mc.metrics.TotalQueryTime += duration.Milliseconds()

	if err != "" {
		mc.metrics.QueryErrors++
	}

	mc.queries = append(mc.queries, &QueryMetric{
		Query:        query,
		ExecutedAt:   time.Now(),
		Duration:     duration,
		RowsAffected: rowsAffected,
		Error:        err,
	})

	mc.trimHistory()
}

// RecordIssueAnalyzed records a GitHub issue analysis
func (mc *MetricsCollector) RecordIssueAnalyzed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.IssuesAnalyzed++
}

// RecordPRReviewed records a PR review
func (mc *MetricsCollector) RecordPRReviewed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.PRsReviewed++
}

// RecordBugFixed records a bug fix
func (mc *MetricsCollector) RecordBugFixed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.BugsFixed++
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var avgDuration, avgCodeScore, avgResponseTime, avgQueryTime float64

	if mc.metrics.WorkflowsCompleted > 0 {
		avgDuration = float64(mc.metrics.TotalDuration) / float64(mc.metrics.WorkflowsCompleted)
		avgCodeScore = float64(mc.metrics.CodeScoreSum) / float64(mc.metrics.WorkflowsCompleted)
	}

	if mc.metrics.RequestsTotal > 0 {
		avgResponseTime = float64(mc.metrics.TotalResponseTime) / float64(mc.metrics.RequestsTotal)
	}

	if mc.metrics.QueriesExecuted > 0 {
		avgQueryTime = float64(mc.metrics.TotalQueryTime) / float64(mc.metrics.QueriesExecuted)
	}

	successRate := float64(0)
	if mc.metrics.WorkflowsStarted > 0 {
		successRate = float64(mc.metrics.WorkflowsCompleted) / float64(mc.metrics.WorkflowsStarted) * 100
	}

	return map[string]interface{}{
		"workflows": map[string]interface{}{
			"started":           mc.metrics.WorkflowsStarted,
			"completed":         mc.metrics.WorkflowsCompleted,
			"failed":            mc.metrics.WorkflowsFailed,
			"success_rate":      successRate,
			"avg_duration_ms":   avgDuration,
			"avg_code_score":    avgCodeScore,
		},
		"bugs": map[string]interface{}{
			"detected": mc.metrics.BugsDetected,
			"fixed":    mc.metrics.BugsFixed,
		},
		"github": map[string]interface{}{
			"issues_analyzed": mc.metrics.IssuesAnalyzed,
			"prs_reviewed":    mc.metrics.PRsReviewed,
		},
		"api": map[string]interface{}{
			"requests_total":      mc.metrics.RequestsTotal,
			"requests_error":      mc.metrics.RequestsError,
			"error_rate":          float64(mc.metrics.RequestsError) / float64(mc.metrics.RequestsTotal) * 100,
			"avg_response_time_ms": avgResponseTime,
		},
		"database": map[string]interface{}{
			"queries_executed":  mc.metrics.QueriesExecuted,
			"query_errors":      mc.metrics.QueryErrors,
			"avg_query_time_ms": avgQueryTime,
		},
	}
}

// GetRecentWorkflows returns recent workflow metrics
func (mc *MetricsCollector) GetRecentWorkflows(limit int) []*WorkflowMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit > len(mc.workflows) {
		limit = len(mc.workflows)
	}

	start := len(mc.workflows) - limit
	if start < 0 {
		start = 0
	}

	return mc.workflows[start:]
}

// GetRecentRequests returns recent request metrics
func (mc *MetricsCollector) GetRecentRequests(limit int) []*RequestMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit > len(mc.requests) {
		limit = len(mc.requests)
	}

	start := len(mc.requests) - limit
	if start < 0 {
		start = 0
	}

	return mc.requests[start:]
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = &Metrics{}
	mc.workflows = mc.workflows[:0]
	mc.requests = mc.requests[:0]
	mc.queries = mc.queries[:0]
}

// trimHistory removes old entries if exceeding max history
func (mc *MetricsCollector) trimHistory() {
	if len(mc.workflows) > mc.maxHistory {
		mc.workflows = mc.workflows[len(mc.workflows)-mc.maxHistory:]
	}
	if len(mc.requests) > mc.maxHistory {
		mc.requests = mc.requests[len(mc.requests)-mc.maxHistory:]
	}
	if len(mc.queries) > mc.maxHistory {
		mc.queries = mc.queries[len(mc.queries)-mc.maxHistory:]
	}
}
