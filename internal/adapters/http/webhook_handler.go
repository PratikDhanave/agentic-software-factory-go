package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type WebhookHandler struct {
	submitJobUC *jobs.SubmitJobUseCase
	logger      services.Logger
}

func NewWebhookHandler(submitJobUC *jobs.SubmitJobUseCase, logger services.Logger) *WebhookHandler {
	return &WebhookHandler{
		submitJobUC: submitJobUC,
		logger:      logger,
	}
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "missing X-GitHub-Event header", http.StatusBadRequest)
		return
	}

	event, err := h.parseGitHubEvent(eventType, payload)
	if err != nil {
		h.logger.Error("failed to parse github event", err, "type", eventType)
		http.Error(w, "failed to parse event", http.StatusBadRequest)
		return
	}

	if event == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	job := entities.NewJob(
		fmt.Sprintf("gh-webhook-%d", time.Now().UnixNano()),
		event.WorkflowType,
		event.ToInput(),
	)
	job.Priority = 5

	if err := h.submitJobUC.Execute(ctx, job); err != nil {
		h.logger.Error("failed to submit job", err)
		http.Error(w, "failed to submit job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "accepted",
		"workflow_id": job.ID,
		"type":        job.Type,
	})
}

type GitHubEventData struct {
	Action       string
	WorkflowType string
	Repo         string
	Issue        int
	PR           int
	Title        string
	Body         string
}

func (e *GitHubEventData) ToInput() map[string]interface{} {
	return map[string]interface{}{
		"action":   e.Action,
		"repo":     e.Repo,
		"issue":    e.Issue,
		"pr":       e.PR,
		"title":    e.Title,
		"body":     e.Body,
	}
}

func (h *WebhookHandler) parseGitHubEvent(eventType string, payload []byte) (*GitHubEventData, error) {
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(payload, &rawEvent); err != nil {
		return nil, err
	}

	action, _ := rawEvent["action"].(string)

	var event *GitHubEventData

	switch eventType {
	case "issues":
		event = h.parseIssueEvent(rawEvent, action)
	case "pull_request":
		event = h.parsePullRequestEvent(rawEvent, action)
	case "push":
		event = h.parsePushEvent(rawEvent)
	default:
		return nil, nil
	}

	return event, nil
}

func (h *WebhookHandler) parseIssueEvent(raw map[string]interface{}, action string) *GitHubEventData {
	if action != "opened" && action != "reopened" {
		return nil
	}

	issue, _ := raw["issue"].(map[string]interface{})
	repo, _ := raw["repository"].(map[string]interface{})

	issueNum := 0
	if number, ok := issue["number"].(float64); ok {
		issueNum = int(number)
	}

	title := ""
	if t, ok := issue["title"].(string); ok {
		title = t
	}

	body := ""
	if b, ok := issue["body"].(string); ok {
		body = b
	}

	repoName := ""
	if name, ok := repo["full_name"].(string); ok {
		repoName = name
	}

	return &GitHubEventData{
		Action:       action,
		WorkflowType: "issue-triage",
		Repo:         repoName,
		Issue:        issueNum,
		Title:        title,
		Body:         body,
	}
}

func (h *WebhookHandler) parsePullRequestEvent(raw map[string]interface{}, action string) *GitHubEventData {
	if action != "opened" && action != "synchronize" {
		return nil
	}

	pr, _ := raw["pull_request"].(map[string]interface{})
	repo, _ := raw["repository"].(map[string]interface{})

	prNum := 0
	if number, ok := pr["number"].(float64); ok {
		prNum = int(number)
	}

	title := ""
	if t, ok := pr["title"].(string); ok {
		title = t
	}

	body := ""
	if b, ok := pr["body"].(string); ok {
		body = b
	}

	repoName := ""
	if name, ok := repo["full_name"].(string); ok {
		repoName = name
	}

	return &GitHubEventData{
		Action:       action,
		WorkflowType: "pr-review",
		Repo:         repoName,
		PR:           prNum,
		Title:        title,
		Body:         body,
	}
}

func (h *WebhookHandler) parsePushEvent(raw map[string]interface{}) *GitHubEventData {
	repo, _ := raw["repository"].(map[string]interface{})

	repoName := ""
	if name, ok := repo["full_name"].(string); ok {
		repoName = name
	}

	return &GitHubEventData{
		Action:       "push",
		WorkflowType: "push-analyze",
		Repo:         repoName,
	}
}
