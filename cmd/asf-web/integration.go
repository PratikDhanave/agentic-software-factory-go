package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/http/github_webhook_handler"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// IntegrationManager coordinates GitHub webhooks, skill execution, and self-improvement
type IntegrationManager struct {
	webhookHandler       *github_webhook_handler.GitHubWebhookHandler
	selfImprovingService *application.SelfImprovingService
	skillRegistry        *services.SkillRegistry
	llmClient            services.ClaudeClient
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager(repo interface{}, llmClient services.ClaudeClient, skillRegistry *services.SkillRegistry) *IntegrationManager {
	webhookSecret := "your-github-secret" // Load from env in production

	return &IntegrationManager{
		webhookHandler:       github_webhook_handler.NewGitHubWebhookHandler(webhookSecret),
		selfImprovingService: application.NewSelfImprovingService(nil, llmClient), // repo would be injected
		skillRegistry:        skillRegistry,
		llmClient:            llmClient,
	}
}

// RegisterRoutes registers webhook and metrics endpoints
func (im *IntegrationManager) RegisterRoutes(mux *http.ServeMux) {
	// GitHub webhook receiver
	mux.HandleFunc("POST /webhooks/github", im.handleGitHubWebhook)

	// Self-improvement metrics
	mux.HandleFunc("GET /metrics/self-improvement", im.handleSelfImprovementMetrics)

	// Workflow outcome reporting (from CI/CD)
	mux.HandleFunc("POST /webhooks/outcome", im.handleWorkflowOutcome)
}

// handleGitHubWebhook processes incoming GitHub events
func (im *IntegrationManager) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Read and verify signature
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	if !im.webhookHandler.VerifySignature(body, signature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse event and generate workflow trigger
	var event interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "failed to parse event", http.StatusBadRequest)
		return
	}

	// Process based on GitHub event type
	eventType := r.Header.Get("X-GitHub-Event")
	im.processGitHubEvent(r.Context(), eventType, body)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

// processGitHubEvent handles different GitHub event types
func (im *IntegrationManager) processGitHubEvent(ctx context.Context, eventType string, payload []byte) error {
	switch eventType {
	case "pull_request":
		return im.handlePullRequestEvent(ctx, payload)

	case "issues":
		return im.handleIssueEvent(ctx, payload)

	case "pull_request_review":
		return im.handlePRReviewEvent(ctx, payload)

	case "issue_comment":
		return im.handleIssueCommentEvent(ctx, payload)

	default:
		return nil // Ignore unhandled event types
	}
}

// handlePullRequestEvent triggers review workflow on PR events
func (im *IntegrationManager) handlePullRequestEvent(ctx context.Context, payload []byte) error {
	type PREvent struct {
		Action      string `json:"action"`
		PullRequest struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			Body   string `json:"body"`
			State  string `json:"state"`
		} `json:"pull_request"`
	}

	var event PREvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Trigger code review skill
	switch event.Action {
	case "opened", "ready_for_review":
		return im.executeCodeReviewSkill(ctx, event.PullRequest.Title, event.PullRequest.Body)
	}

	return nil
}

// handleIssueEvent triggers triage workflow on issue events
func (im *IntegrationManager) handleIssueEvent(ctx context.Context, payload []byte) error {
	type IssueEvent struct {
		Action string `json:"action"`
		Issue  struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		} `json:"issue"`
	}

	var event IssueEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Trigger triage skill on issue creation
	if event.Action == "opened" {
		return im.executeTriageSkill(ctx, event.Issue.Title, event.Issue.Body)
	}

	return nil
}

// handlePRReviewEvent triggers merge decision on review submission
func (im *IntegrationManager) handlePRReviewEvent(ctx context.Context, payload []byte) error {
	type ReviewEvent struct {
		Action string `json:"action"`
	}

	var event ReviewEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	if event.Action == "submitted" {
		// Trigger merge decision skill
		// In production: analyze review comments and decide merge readiness
		return nil
	}

	return nil
}

// handleIssueCommentEvent handles comment-based triggers
func (im *IntegrationManager) handleIssueCommentEvent(ctx context.Context, payload []byte) error {
	type CommentEvent struct {
		Action  string `json:"action"`
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
	}

	var event CommentEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	// Listen for "@factory" mentions to trigger workflows
	if event.Action == "created" && strings.Contains(event.Comment.Body, "@factory") {
		// Parse command and trigger appropriate skill
		return nil
	}

	return nil
}

// executeCodeReviewSkill runs the code review skill
func (im *IntegrationManager) executeCodeReviewSkill(ctx context.Context, title, body string) error {
	// Get code review skill from registry
	skill, err := im.skillRegistry.GetSkill("code-review")
	if err != nil {
		return err
	}

	// Prepare input
	input := map[string]interface{}{
		"title": title,
		"body":  body,
	}

	// Execute skill
	output, err := skill.Execute(ctx, input)
	if err != nil {
		return err
	}

	// Record the decision for learning
	_ = output // Use output to generate decision

	return nil
}

// executeTriageSkill runs the triage skill
func (im *IntegrationManager) executeTriageSkill(ctx context.Context, title, body string) error {
	// Get triage skill from registry
	skill, err := im.skillRegistry.GetSkill("triage-issue")
	if err != nil {
		return err
	}

	// Prepare input
	input := map[string]interface{}{
		"issue": title + "\n" + body,
	}

	// Execute skill
	output, err := skill.Execute(ctx, input)
	if err != nil {
		return err
	}

	// Record the decision for learning
	_ = output // Use output to categorize issue

	return nil
}

// handleWorkflowOutcome records whether a workflow succeeded or failed
func (im *IntegrationManager) handleWorkflowOutcome(w http.ResponseWriter, r *http.Request) {
	type OutcomeRequest struct {
		ExecutionID string `json:"execution_id"`
		Success     bool   `json:"success"`
		PRNumber    int    `json:"pr_number"`
		Merged      bool   `json:"merged"`
		BuildPassed bool   `json:"build_passed"`
	}

	var req OutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Record outcome for self-improvement
	success := req.Success && req.Merged && req.BuildPassed
	if err := im.selfImprovingService.RecordOutcome(r.Context(), req.ExecutionID, success); err != nil {
		http.Error(w, "failed to record outcome", http.StatusInternalServerError)
		return
	}

	// Trigger learning if threshold reached
	go im.selfImprovingService.LearnPatterns(context.Background(), "all_agents")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

// handleSelfImprovementMetrics returns current agent metrics
func (im *IntegrationManager) handleSelfImprovementMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := im.selfImprovingService.GetMetrics(r.Context())
	if err != nil {
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Crawl-Walk-Run Implementation Phases

// Phase 1: Crawl - Basic workflow generation (CURRENT)
// - Receive GitHub events
// - Generate agent decisions
// - Store decisions for learning
// - Manual approvals required

// Phase 2: Walk - Self-improvement enabled
// - Collect metrics from approvals
// - Discover patterns
// - Apply patterns to future decisions
// - Confidence calibration

// Phase 3: Run - Fully autonomous
// - High-confidence decisions skip approval
// - LLM as judge validates decisions
// - Multi-skill composition
// - Full GitHub automation

type CrawlWalkRunPhase string

const (
	PhaseCrawl CrawlWalkRunPhase = "crawl" // Manual approval required
	PhaseWalk  CrawlWalkRunPhase = "walk"  // Learning enabled
	PhaseRun   CrawlWalkRunPhase = "run"   // Fully autonomous
)

// GetPhaseConfig returns configuration for current phase
func GetPhaseConfig(phase CrawlWalkRunPhase) map[string]interface{} {
	configs := map[CrawlWalkRunPhase]map[string]interface{}{
		PhaseCrawl: {
			"approval_required_threshold": 0.95,     // All decisions need approval
			"pattern_learning_enabled":    false,
			"confidence_calibration":      false,
			"autonomous_execution":        false,
			"skills_enabled":              []string{"code-review", "triage-issue"},
		},
		PhaseWalk: {
			"approval_required_threshold": 0.90,     // 90%+ confidence skip approval
			"pattern_learning_enabled":    true,
			"confidence_calibration":      true,
			"autonomous_execution":        false,
			"skills_enabled":              []string{"code-review", "triage-issue", "security-review", "review-spec"},
			"learning_window":             100, // Trigger learning every 100 decisions
		},
		PhaseRun: {
			"approval_required_threshold": 0.95,     // Only 95%+ confidence execute
			"pattern_learning_enabled":    true,
			"confidence_calibration":      true,
			"autonomous_execution":        true,
			"skills_enabled":              []string{"code-review", "triage-issue", "security-review", "review-spec", "merge-decision"},
			"learning_window":             50,  // Faster learning cycles
			"llm_as_judge_enabled":        true,
		},
	}

	return configs[phase]
}
