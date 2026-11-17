package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/metrics"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/queue"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/realtime"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/security"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webhooks"
)

// Global instances
var (
	jobQueue          *queue.Queue
	jobScheduler      *queue.Scheduler
	metricsCollector  *metrics.MetricsCollector
	realtimeHub       *realtime.Hub
	securityValidator *security.Validator
	rateLimiter       *security.RateLimiter
	auditLog          *security.AuditLog
	webhookProcessor  *webhooks.EventProcessor
)

// InitializeIntegration initializes all systems
func InitializeIntegration() error {
	log.Println("[Integration] Initializing all systems...")

	// Initialize queue
	jobQueue = queue.NewQueue(5, 3)
	jobQueue.Start()

	// Initialize scheduler
	jobScheduler = queue.NewScheduler(jobQueue)
	jobScheduler.Start()

	// Initialize metrics
	metricsCollector = metrics.NewMetricsCollector(1000)

	// Initialize realtime hub
	realtimeHub = realtime.NewHub()
	realtimeHub.Start()

	// Initialize security
	securityValidator = &security.Validator{}
	rateLimiter = security.NewRateLimiter(100)
	auditLog = security.NewAuditLog(500)

	// Initialize webhook processor
	webhookProcessor = webhooks.NewEventProcessor(os.Getenv("GITHUB_WEBHOOK_SECRET"))

	log.Println("[Integration] ✅ All systems initialized")
	return nil
}

// HandleGitHubWebhook processes GitHub webhook events
func HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Rate limit check
	clientIP := r.RemoteAddr
	if !rateLimiter.Allow(clientIP) {
		auditLog.Log("RATE_LIMIT", clientIP, "webhook", "post", "rejected", "Rate limit exceeded")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Verify signature
	signature := r.Header.Get("X-Hub-Signature-256")
	payload, _ := io.ReadAll(r.Body)

	if !webhookProcessor.VerifySignature(payload, signature) {
		auditLog.Log("SECURITY", "unknown", "webhook", "verify", "failed", "Invalid signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse event
	eventType := r.Header.Get("X-GitHub-Event")
	trigger, err := webhookProcessor.ProcessEvent(eventType, payload)

	if err != nil {
		auditLog.Log("ERROR", clientIP, "webhook", "parse", "failed", err.Error())
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	if trigger == nil || !trigger.ShouldTrigger() {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	// Submit to queue
	job := &queue.Job{
		ID:       fmt.Sprintf("gh-%s-%d-%d", trigger.Type, trigger.Issue+trigger.PR, time.Now().UnixNano()),
		Type:     trigger.Type,
		Status:   "pending",
		Input:    trigger.GetWorkflowInput(),
		Priority: 5,
		ExecutionFn: func(j *queue.Job) error {
			return executeWorkflow(j, trigger)
		},
	}

	if err := jobQueue.Submit(job); err != nil {
		auditLog.Log("ERROR", clientIP, "webhook", "submit", "failed", err.Error())
		http.Error(w, "Failed to submit workflow", http.StatusInternalServerError)
		return
	}

	// Broadcast to realtime clients
	realtimeHub.BroadcastWorkflowStarted(job.ID, trigger.Type)
	metricsCollector.RecordWorkflowStart(job.ID, trigger.Type)

	// Audit log
	auditLog.Log("WEBHOOK", clientIP, trigger.Repo, eventType, "success", fmt.Sprintf("Workflow %s submitted", job.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "accepted",
		"workflow_id": job.ID,
		"type":        trigger.Type,
	})
}

// executeWorkflow executes a workflow from a webhook trigger
func executeWorkflow(job *queue.Job, trigger *webhooks.WorkflowTrigger) error {
	startTime := time.Now()
	log.Printf("[Workflow] Executing %s for %s", trigger.Type, trigger.Repo)

	// Broadcast update
	realtimeHub.BroadcastWorkflowUpdate(job.ID, "running", 25)

	// Execute based on type
	var output map[string]interface{}
	var err error

	switch trigger.Type {
	case "issue-triage":
		output, err = triageIssue(trigger)
	case "pr-review":
		output, err = reviewPullRequest(trigger)
	case "pr-feedback":
		output, err = handlePRFeedback(trigger)
	case "push-analyze":
		output, err = analyzePush(trigger)
	default:
		err = fmt.Errorf("unknown workflow type: %s", trigger.Type)
	}

	duration := time.Since(startTime)
	job.Output = output

	// Record metrics
	success := err == nil
	metricsCollector.RecordWorkflowCompletion(job.ID, success, duration, 0, 85, 90)

	// Broadcast completion
	if success {
		realtimeHub.BroadcastWorkflowCompleted(job.ID, "success", output)
		metricsCollector.RecordIssueAnalyzed()
	} else {
		realtimeHub.BroadcastWorkflowCompleted(job.ID, "failed", map[string]string{"error": err.Error()})
	}

	return err
}

// triageIssue performs issue triage
func triageIssue(trigger *webhooks.WorkflowTrigger) (map[string]interface{}, error) {
	// Simulate analysis
	time.Sleep(100 * time.Millisecond)

	result := map[string]interface{}{
		"issue_number": trigger.Issue,
		"repository":   trigger.Repo,
		"priority":     "high",
		"category":     "feature",
		"action":       "review",
		"suggested_labels": []string{"feature", "review-needed"},
	}

	// Store in database if available
	if database != nil {
		database.RecordTriageResult(
			trigger.Issue,
			trigger.Repo,
			"high",
			"feature",
			trigger.Title,
			trigger.Body,
			"review",
		)
	}

	return result, nil
}

// reviewPullRequest performs PR review
func reviewPullRequest(trigger *webhooks.WorkflowTrigger) (map[string]interface{}, error) {
	// Simulate review
	time.Sleep(150 * time.Millisecond)

	result := map[string]interface{}{
		"pr_number":    trigger.PR,
		"repository":   trigger.Repo,
		"score":        85,
		"approved":     true,
		"findings":     []string{},
	}

	metricsCollector.RecordPRReviewed()
	return result, nil
}

// handlePRFeedback handles PR review feedback
func handlePRFeedback(trigger *webhooks.WorkflowTrigger) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"pr_number":  trigger.PR,
		"repository": trigger.Repo,
		"status":     "feedback_received",
	}
	return result, nil
}

// analyzePush analyzes a push event
func analyzePush(trigger *webhooks.WorkflowTrigger) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"repository": trigger.Repo,
		"status":     "analyzed",
	}
	return result, nil
}

// HandleRealtimeStream streams real-time updates via Server-Sent Events
func HandleRealtimeStream(w http.ResponseWriter, r *http.Request) {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Register client with hub
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := realtimeHub.RegisterClient(clientID)
	defer realtimeHub.UnregisterClient(client)

	log.Printf("[Stream] Client connected: %s", clientID)

	// Stream messages
	for {
		select {
		case <-r.Context().Done():
			log.Printf("[Stream] Client disconnected: %s", clientID)
			return

		case msg, ok := <-client.send:
			if !ok {
				return
			}

			// Send as SSE
			fmt.Fprintf(w, "data: %s\n\n", string(msg.ToJSON()))
			flusher.Flush()
		}
	}
}

// HandleMetrics returns current metrics
func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if metricsCollector == nil {
		http.Error(w, "Metrics not available", http.StatusServiceUnavailable)
		return
	}

	metrics := metricsCollector.GetMetrics()
	queueStats := jobQueue.Stats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metrics,
		"queue":   queueStats,
		"clients": realtimeHub.ClientCount(),
	})
}

// HandleJobStatus returns job status
func HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	job, ok := jobQueue.Get(jobID)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

// HandleJobsList returns recent jobs
func HandleJobsList(w http.ResponseWriter, r *http.Request) {
	pending := jobQueue.GetByStatus("pending")
	running := jobQueue.GetByStatus("running")
	success := jobQueue.GetByStatus("success")
	failed := jobQueue.GetByStatus("failed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending": pending,
		"running": running,
		"success": success,
		"failed":  failed,
	})
}

// HandleAuditLog returns audit events
func HandleAuditLog(w http.ResponseWriter, r *http.Request) {
	if auditLog == nil {
		http.Error(w, "Audit log not available", http.StatusServiceUnavailable)
		return
	}

	limit := 50
	events := auditLog.GetEvents(limit)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
