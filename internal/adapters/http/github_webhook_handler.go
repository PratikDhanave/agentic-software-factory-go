package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitHubWebhookEvent represents a GitHub webhook event
type GitHubWebhookEvent struct {
	Action     string                 `json:"action"`
	Number     int                    `json:"number"`
	PullRequest *GitHubPullRequest     `json:"pull_request"`
	Issue      *GitHubIssue           `json:"issue"`
	Repository *GitHubRepository      `json:"repository"`
	Sender     *GitHubUser            `json:"sender"`
}

type GitHubPullRequest struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Draft  bool   `json:"draft"`
	User   *GitHubUser `json:"user"`
	Head   *GitHubRef `json:"head"`
	Base   *GitHubRef `json:"base"`
}

type GitHubIssue struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	User   *GitHubUser `json:"user"`
	Labels []GitHubLabel `json:"labels"`
}

type GitHubRef struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	Repo *GitHubRepository `json:"repo"`
}

type GitHubRepository struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Owner *GitHubUser `json:"owner"`
}

type GitHubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

type GitHubLabel struct {
	Name string `json:"name"`
}

// GitHubWorkflowTrigger represents a workflow trigger from GitHub
type GitHubWorkflowTrigger struct {
	Type       string // "pr-opened", "pr-review-requested", "issue-created", etc.
	EventID    string
	Number     int
	Title      string
	Body       string
	Repository string
	Author     string
	Agent      string // Which agent should handle this
	Skill      string // Which skill to use
	Metadata   map[string]interface{}
}

// GitHubWebhookHandler handles GitHub webhook events
type GitHubWebhookHandler struct {
	secret       string
	triggerChan  chan *GitHubWorkflowTrigger
	logger       interface{} // Logger interface
}

// NewGitHubWebhookHandler creates a new webhook handler
func NewGitHubWebhookHandler(secret string) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		secret:      secret,
		triggerChan: make(chan *GitHubWorkflowTrigger, 100),
	}
}

// VerifySignature verifies the GitHub webhook signature
func (h *GitHubWebhookHandler) VerifySignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return true // No verification if no secret set
	}

	hash := hmac.New(sha256.New, []byte(h.secret))
	hash.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// HandleWebhook processes GitHub webhook events
func (h *GitHubWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !h.VerifySignature(body, signature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse event
	var event GitHubWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "failed to parse event", http.StatusBadRequest)
		return
	}

	// Route based on event type
	eventType := r.Header.Get("X-GitHub-Event")
	trigger := h.routeEvent(eventType, &event)

	if trigger != nil {
		// Send to trigger channel (non-blocking)
		select {
		case h.triggerChan <- trigger:
		default:
			// Channel full, log warning
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// routeEvent determines how to handle an event
func (h *GitHubWebhookHandler) routeEvent(eventType string, event *GitHubWebhookEvent) *GitHubWorkflowTrigger {
	switch eventType {
	case "pull_request":
		return h.handlePREvent(event)
	case "issues":
		return h.handleIssueEvent(event)
	case "pull_request_review":
		return h.handlePRReviewEvent(event)
	case "issue_comment":
		return h.handleIssueCommentEvent(event)
	default:
		return nil
	}
}

// handlePREvent routes PR events
func (h *GitHubWebhookHandler) handlePREvent(event *GitHubWebhookEvent) *GitHubWorkflowTrigger {
	if event.PullRequest == nil {
		return nil
	}

	switch event.Action {
	case "opened":
		if !event.PullRequest.Draft {
			return &GitHubWorkflowTrigger{
				Type:       "pr-opened",
				EventID:    fmt.Sprintf("pr-%d", event.PullRequest.ID),
				Number:     event.PullRequest.Number,
				Title:      event.PullRequest.Title,
				Body:       event.PullRequest.Body,
				Repository: event.Repository.Name,
				Author:     event.PullRequest.User.Login,
				Agent:      "Review",
				Skill:      "code-review",
				Metadata: map[string]interface{}{
					"url":    event.PullRequest.ID,
					"branch": event.PullRequest.Head.Ref,
				},
			}
		}

	case "ready_for_review":
		return &GitHubWorkflowTrigger{
			Type:       "pr-ready-for-review",
			EventID:    fmt.Sprintf("pr-%d-ready", event.PullRequest.ID),
			Number:     event.PullRequest.Number,
			Title:      event.PullRequest.Title,
			Body:       event.PullRequest.Body,
			Repository: event.Repository.Name,
			Author:     event.PullRequest.User.Login,
			Agent:      "Review",
			Skill:      "code-review",
		}

	case "review_requested":
		return &GitHubWorkflowTrigger{
			Type:       "pr-review-requested",
			EventID:    fmt.Sprintf("pr-%d-review-req", event.PullRequest.ID),
			Number:     event.PullRequest.Number,
			Title:      event.PullRequest.Title,
			Body:       event.PullRequest.Body,
			Repository: event.Repository.Name,
			Author:     event.PullRequest.User.Login,
			Agent:      "Review",
			Skill:      "code-review",
		}
	}

	return nil
}

// handleIssueEvent routes issue events
func (h *GitHubWebhookHandler) handleIssueEvent(event *GitHubWebhookEvent) *GitHubWorkflowTrigger {
	if event.Issue == nil {
		return nil
	}

	switch event.Action {
	case "opened":
		return &GitHubWorkflowTrigger{
			Type:       "issue-created",
			EventID:    fmt.Sprintf("issue-%d", event.Issue.ID),
			Number:     event.Issue.Number,
			Title:      event.Issue.Title,
			Body:       event.Issue.Body,
			Repository: event.Repository.Name,
			Author:     event.Issue.User.Login,
			Agent:      "Triage",
			Skill:      "triage-issue",
			Metadata: map[string]interface{}{
				"labels": labelsToStrings(event.Issue.Labels),
			},
		}

	case "labeled":
		// Check if labeled with specific gate labels
		labels := labelsToStrings(event.Issue.Labels)
		if contains(labels, "needs-review") {
			return &GitHubWorkflowTrigger{
				Type:       "issue-needs-review",
				EventID:    fmt.Sprintf("issue-%d-review", event.Issue.ID),
				Number:     event.Issue.Number,
				Title:      event.Issue.Title,
				Body:       event.Issue.Body,
				Repository: event.Repository.Name,
				Author:     event.Issue.User.Login,
				Agent:      "Review",
				Skill:      "review-spec",
			}
		}
	}

	return nil
}

// handlePRReviewEvent routes PR review events
func (h *GitHubWebhookHandler) handlePRReviewEvent(event *GitHubWebhookEvent) *GitHubWorkflowTrigger {
	if event.Action == "submitted" && event.PullRequest != nil {
		return &GitHubWorkflowTrigger{
			Type:       "pr-review-submitted",
			EventID:    fmt.Sprintf("pr-%d-reviewed", event.PullRequest.ID),
			Number:     event.PullRequest.Number,
			Title:      event.PullRequest.Title,
			Repository: event.Repository.Name,
			Author:     event.Sender.Login,
			Agent:      "Foreman",
			Skill:      "merge-decision",
		}
	}
	return nil
}

// handleIssueCommentEvent routes issue comment events
func (h *GitHubWebhookHandler) handleIssueCommentEvent(event *GitHubWebhookEvent) *GitHubWorkflowTrigger {
	// Could trigger workflows based on comment mentions
	// E.g., "@factory review" triggers review workflow
	return nil
}

// GetTriggerChannel returns the trigger channel
func (h *GitHubWebhookHandler) GetTriggerChannel() <-chan *GitHubWorkflowTrigger {
	return h.triggerChan
}

// Helper functions
func labelsToStrings(labels []GitHubLabel) []string {
	result := make([]string, len(labels))
	for i, l := range labels {
		result[i] = l.Name
	}
	return result
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
