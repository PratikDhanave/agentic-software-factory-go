package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/messaging"
)

func TestWebhookHandler_HandleGitHubWebhook_IssueOpened(t *testing.T) {
	logger := logging.NewStdLogger()
	jobRepo := persistence.NewInMemoryJobRepository()
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	handler := NewWebhookHandler(submitJobUC, logger)

	payload := []byte(`{
		"action": "opened",
		"issue": {
			"number": 1,
			"title": "Test Issue",
			"body": "Issue body"
		},
		"repository": {
			"full_name": "owner/repo",
			"html_url": "https://github.com/owner/repo"
		}
	}`)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !bytes.Contains(body, []byte("accepted")) {
		t.Errorf("expected 'accepted' in response body")
	}
}

func TestWebhookHandler_HandleGitHubWebhook_MissingEvent(t *testing.T) {
	logger := logging.NewStdLogger()
	jobRepo := persistence.NewInMemoryJobRepository()
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	handler := NewWebhookHandler(submitJobUC, logger)

	req := httptest.NewRequest("POST", "/webhook", nil)

	w := httptest.NewRecorder()
	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestWebhookHandler_HandleGitHubWebhook_IgnoredAction(t *testing.T) {
	logger := logging.NewStdLogger()
	jobRepo := persistence.NewInMemoryJobRepository()
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	handler := NewWebhookHandler(submitJobUC, logger)

	payload := []byte(`{
		"action": "closed",
		"issue": {"number": 1, "title": "Test", "body": ""},
		"repository": {"full_name": "owner/repo"}
	}`)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")

	w := httptest.NewRecorder()
	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for ignored action, got %d", w.Code)
	}
}
