package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// EventProcessor handles GitHub webhook events
type EventProcessor struct {
	secret string
}

// GitHubEvent represents a GitHub webhook event
type GitHubEvent struct {
	Action      string          `json:"action"`
	Issue       *Issue          `json:"issue"`
	PullRequest *PullRequest    `json:"pull_request"`
	Repository  Repository      `json:"repository"`
	Sender      Sender          `json:"sender"`
}

type Issue struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

type PullRequest struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

type Repository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	HTMLURL string `json:"html_url"`
}

type Sender struct {
	Login string `json:"login"`
}

// WorkflowTrigger represents a workflow to execute
type WorkflowTrigger struct {
	Type       string                 `json:"type"` // issue-triage, pr-review, etc
	EventType  string                 `json:"event_type"`
	Repo       string                 `json:"repo"`
	Issue      int                    `json:"issue,omitempty"`
	PR         int                    `json:"pr,omitempty"`
	SHA        string                 `json:"sha,omitempty"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	Params     map[string]interface{} `json:"params"`
}

// NewEventProcessor creates a new webhook processor
func NewEventProcessor(secret string) *EventProcessor {
	if secret == "" {
		secret = os.Getenv("GITHUB_WEBHOOK_SECRET")
	}
	return &EventProcessor{secret: secret}
}

// VerifySignature validates the GitHub webhook signature
func (p *EventProcessor) VerifySignature(payload []byte, signature string) bool {
	if p.secret == "" {
		log.Println("[Webhook] Warning: No webhook secret configured")
		return true // Allow if no secret configured
	}

	expectedSig := "sha256=" + p.computeSignature(payload)
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

func (p *EventProcessor) computeSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(p.secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// ProcessEvent parses and processes a GitHub webhook event
func (p *EventProcessor) ProcessEvent(eventType string, payload []byte) (*WorkflowTrigger, error) {
	var event GitHubEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	log.Printf("[Webhook] Processing %s event from %s", eventType, event.Repository.FullName)

	switch eventType {
	case "issues":
		return p.processIssueEvent(&event)
	case "pull_request":
		return p.processPullRequestEvent(&event)
	case "pull_request_review":
		return p.processPRReviewEvent(&event)
	case "push":
		return p.processPushEvent(&event)
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func (p *EventProcessor) processIssueEvent(event *GitHubEvent) (*WorkflowTrigger, error) {
	if event.Issue == nil {
		return nil, fmt.Errorf("missing issue in payload")
	}

	// Only trigger on opened or reopened
	if event.Action != "opened" && event.Action != "reopened" {
		log.Printf("[Webhook] Skipping issue action: %s", event.Action)
		return nil, nil
	}

	return &WorkflowTrigger{
		Type:      "issue-triage",
		EventType: "issues",
		Repo:      event.Repository.FullName,
		Issue:     event.Issue.Number,
		Title:     event.Issue.Title,
		Body:      event.Issue.Body,
		Params: map[string]interface{}{
			"issue_number": event.Issue.Number,
			"repository":   event.Repository.FullName,
			"owner":        event.Repository.Owner.Login,
			"action":       event.Action,
		},
	}, nil
}

func (p *EventProcessor) processPullRequestEvent(event *GitHubEvent) (*WorkflowTrigger, error) {
	if event.PullRequest == nil {
		return nil, fmt.Errorf("missing pull_request in payload")
	}

	// Only trigger on opened or synchronize
	if event.Action != "opened" && event.Action != "synchronize" {
		log.Printf("[Webhook] Skipping PR action: %s", event.Action)
		return nil, nil
	}

	return &WorkflowTrigger{
		Type:      "pr-review",
		EventType: "pull_request",
		Repo:      event.Repository.FullName,
		PR:        event.PullRequest.Number,
		SHA:       event.PullRequest.Head.SHA,
		Title:     event.PullRequest.Title,
		Body:      event.PullRequest.Body,
		Params: map[string]interface{}{
			"pr_number":  event.PullRequest.Number,
			"repository": event.Repository.FullName,
			"owner":      event.Repository.Owner.Login,
			"sha":        event.PullRequest.Head.SHA,
			"action":     event.Action,
		},
	}, nil
}

func (p *EventProcessor) processPRReviewEvent(event *GitHubEvent) (*WorkflowTrigger, error) {
	if event.PullRequest == nil {
		return nil, fmt.Errorf("missing pull_request in payload")
	}

	// Only trigger on submitted reviews
	if event.Action != "submitted" {
		log.Printf("[Webhook] Skipping review action: %s", event.Action)
		return nil, nil
	}

	return &WorkflowTrigger{
		Type:      "pr-feedback",
		EventType: "pull_request_review",
		Repo:      event.Repository.FullName,
		PR:        event.PullRequest.Number,
		SHA:       event.PullRequest.Head.SHA,
		Title:     event.PullRequest.Title,
		Params: map[string]interface{}{
			"pr_number":  event.PullRequest.Number,
			"repository": event.Repository.FullName,
			"owner":      event.Repository.Owner.Login,
			"sha":        event.PullRequest.Head.SHA,
		},
	}, nil
}

func (p *EventProcessor) processPushEvent(event *GitHubEvent) (*WorkflowTrigger, error) {
	// Extract SHA from ref (refs/heads/main -> commit SHA would be in payload but simplified)
	refParts := strings.Split(event.Repository.HTMLURL, "/")
	if len(refParts) == 0 {
		return nil, fmt.Errorf("invalid repository URL")
	}

	return &WorkflowTrigger{
		Type:      "push-analyze",
		EventType: "push",
		Repo:      event.Repository.FullName,
		Title:     "Push to " + event.Repository.Name,
		Params: map[string]interface{}{
			"repository": event.Repository.FullName,
			"owner":      event.Repository.Owner.Login,
		},
	}, nil
}

// ShouldTrigger determines if a workflow should be triggered
func (t *WorkflowTrigger) ShouldTrigger() bool {
	return t.Type != "" && t.Repo != ""
}

// GetWorkflowInput returns the input for the workflow
func (t *WorkflowTrigger) GetWorkflowInput() map[string]interface{} {
	return map[string]interface{}{
		"type":       t.Type,
		"event_type": t.EventType,
		"repo":       t.Repo,
		"issue":      t.Issue,
		"pr":         t.PR,
		"sha":        t.SHA,
		"title":      t.Title,
		"body":       t.Body,
		"params":     t.Params,
	}
}
