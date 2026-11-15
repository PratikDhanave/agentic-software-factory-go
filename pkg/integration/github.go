package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// GitHubIssue represents a GitHub issue payload
type GitHubIssue struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      *GitHubUser `json:"user"`
	Labels    []GitHubLabel `json:"labels"`
	Milestone *GitHubMilestone `json:"milestone"`
}

// GitHubPullRequest represents a GitHub pull request payload
type GitHubPullRequest struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ClosedAt  string `json:"closed_at"`
	MergedAt  string `json:"merged_at"`
	User      *GitHubUser `json:"user"`
	Head      *GitHubRef  `json:"head"`
	Base      *GitHubRef  `json:"base"`
}

// GitHubRef represents a git ref in a PR or push event
type GitHubRef struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
	SHA   string `json:"sha"`
}

// GitHubComment represents a comment on a PR or issue
type GitHubComment struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      *GitHubUser `json:"user"`
	Issue     *GitHubIssue `json:"issue"`
	PullRequest *GitHubPullRequest `json:"pull_request"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	Login   string `json:"login"`
	ID      int    `json:"id"`
	URL     string `json:"url"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	Private     bool   `json:"private"`
	Owner       *GitHubUser `json:"owner"`
}

// GitHubLabel represents a repository label
type GitHubLabel struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// GitHubMilestone represents a milestone
type GitHubMilestone struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// GitHubMember represents a GitHub member
type GitHubMember struct {
	Login   string `json:"login"`
	URL     string `json:"url"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubOrganization represents a GitHub organization
type GitHubOrganization struct {
	Login    string `json:"login"`
	ID       int    `json:"id"`
	URL      string `json:"url"`
	AvatarURL string `json:"avatar_url"`
}

// ProviderGitHub is the provider identifier for GitHub
const ProviderGitHub = "github"

// githubAdapter implements integration.Adapter for GitHub webhooks
type githubAdapter struct {
	secret string
}

// NewGitHubAdapter creates a new GitHub webhook adapter
func NewGitHubAdapter(secret string) *githubAdapter {
	return &githubAdapter{
		secret: secret,
	}
}

// Platform returns the platform name
func (a *githubAdapter) Platform() string { return ProviderGitHub }

// VerifyWebhook verifies the webhook signature using HMAC SHA256
func (a *githubAdapter) VerifyWebhook(r *http.Request) bool {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	// Parse the signature format: "sha256=hexhash"
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signatureHash := signature[7:]

	// Read the raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		return false
	}

	// Compute HMAC SHA256
	mac := hmac.New(sha256.New, []byte(a.secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(expectedSig), []byte(signatureHash))
}

// ParseWebhook parses the webhook payload and returns the event type and data
func (a *githubAdapter) ParseWebhook(r *http.Request) (string, json.RawMessage, error) {
	// Read the raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Try to unmarshal into a generic map to get the event type
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Get the X-GitHub-Event header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return "", nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	return eventType, body, nil
}

// HandlePREvent handles pull request webhook events
func (a *githubAdapter) HandlePREvent(payload []byte) error {
	var prEvent struct {
		Action      string         `json:"action"`
		PullRequest *GitHubPullRequest `json:"pull_request"`
		Repository  *GitHubRepository `json:"repository"`
		Sender      *GitHubUser      `json:"sender"`
	}

	if err := json.Unmarshal(payload, &prEvent); err != nil {
		return fmt.Errorf("failed to parse PR event: %w", err)
	}

	log.Printf("GitHub PR event: action=%s, PR #%d, repo=%s", prEvent.Action, prEvent.PullRequest.Number, prEvent.Repository.FullName)

	switch prEvent.Action {
	case "opened":
		log.Printf("PR #%d opened in %s by %s", prEvent.PullRequest.Number, prEvent.Repository.FullName, prEvent.Sender.Login)
	case "reopened":
		log.Printf("PR #%d reopened in %s by %s", prEvent.PullRequest.Number, prEvent.Repository.FullName, prEvent.Sender.Login)
	case "synchronize":
		log.Printf("PR #%d updated (synchronize) in %s by %s", prEvent.PullRequest.Number, prEvent.Repository.FullName, prEvent.Sender.Login)
	case "closed":
		log.Printf("PR #%d closed in %s by %s", prEvent.PullRequest.Number, prEvent.Repository.FullName, prEvent.Sender.Login)
		// Handle PR closure - check if merged or closed
		if prEvent.PullRequest.MergedAt != "" {
			log.Printf("PR #%d was merged", prEvent.PullRequest.Number)
		} else {
			log.Printf("PR #%d was closed without merging", prEvent.PullRequest.Number)
		}
	case "edited":
		log.Printf("PR #%d edited in %s by %s", prEvent.PullRequest.Number, prEvent.Repository.FullName, prEvent.Sender.Login)
	default:
		log.Printf("Unhandled PR action: %s", prEvent.Action)
	}

	return nil
}

// HandleIssueEvent handles issue webhook events
func (a *githubAdapter) HandleIssueEvent(payload []byte) error {
	var issueEvent struct {
		Action      string      `json:"action"`
		Issue       *GitHubIssue  `json:"issue"`
		Repository  *GitHubRepository `json:"repository"`
		Sender      *GitHubUser   `json:"sender"`
		Label       *GitHubLabel  `json:"label"`
		Assignee    *GitHubUser   `json:"assignee"`
	}

	if err := json.Unmarshal(payload, &issueEvent); err != nil {
		return fmt.Errorf("failed to parse issue event: %w", err)
	}

	log.Printf("GitHub Issue event: action=%s, issue #%d, repo=%s", issueEvent.Action, issueEvent.Issue.Number, issueEvent.Repository.FullName)

	switch issueEvent.Action {
	case "opened":
		log.Printf("Issue #%d opened in %s by %s", issueEvent.Issue.Number, issueEvent.Repository.FullName, issueEvent.Sender.Login)
	case "labeled":
		if issueEvent.Label != nil {
			log.Printf("Issue #%d labeled with %s in %s by %s", issueEvent.Issue.Number, issueEvent.Label.Name, issueEvent.Repository.FullName, issueEvent.Sender.Login)
		}
	case "unlabeled":
		if issueEvent.Label != nil {
			log.Printf("Issue #%d unlabeled %s in %s by %s", issueEvent.Issue.Number, issueEvent.Label.Name, issueEvent.Repository.FullName, issueEvent.Sender.Login)
		}
	case "closed":
		log.Printf("Issue #%d closed in %s by %s", issueEvent.Issue.Number, issueEvent.Repository.FullName, issueEvent.Sender.Login)
	case "reopened":
		log.Printf("Issue #%d reopened in %s by %s", issueEvent.Issue.Number, issueEvent.Repository.FullName, issueEvent.Sender.Login)
	case "assigned":
		if issueEvent.Assignee != nil {
			log.Printf("Issue #%d assigned to %s in %s by %s", issueEvent.Issue.Number, issueEvent.Assignee.Login, issueEvent.Repository.FullName, issueEvent.Sender.Login)
		}
	case "unassigned":
		log.Printf("Issue #%d unassigned in %s by %s", issueEvent.Issue.Number, issueEvent.Repository.FullName, issueEvent.Sender.Login)
	default:
		log.Printf("Unhandled issue action: %s", issueEvent.Action)
	}

	return nil
}

// HandleCommentEvent handles comment webhook events
func (a *githubAdapter) HandleCommentEvent(payload []byte) error {
	var commentEvent struct {
		Action      string      `json:"action"`
		Comment     *GitHubComment `json:"comment"`
		Issue       *GitHubIssue   `json:"issue"`
		PullRequest *GitHubPullRequest `json:"pull_request"`
		Repository  *GitHubRepository `json:"repository"`
		Sender      *GitHubUser    `json:"sender"`
	}

	if err := json.Unmarshal(payload, &commentEvent); err != nil {
		return fmt.Errorf("failed to parse comment event: %w", err)
	}

	log.Printf("GitHub Comment event: action=%s, comment on #%d, repo=%s", commentEvent.Action, commentEvent.Comment.ID, commentEvent.Repository.FullName)

	switch commentEvent.Action {
	case "created":
		log.Printf("Comment created on issue/PR #%d in %s by %s", commentEvent.Comment.ID, commentEvent.Repository.FullName, commentEvent.Sender.Login)
	case "edited":
		log.Printf("Comment edited on issue/PR #%d in %s by %s", commentEvent.Comment.ID, commentEvent.Repository.FullName, commentEvent.Sender.Login)
	case "deleted":
		log.Printf("Comment deleted from issue/PR #%d in %s", commentEvent.Comment.ID, commentEvent.Repository.FullName)
	default:
		log.Printf("Unhandled comment action: %s", commentEvent.Action)
	}

	return nil
}

// HandleMemberEvent handles member webhook events
func (a *githubAdapter) HandleMemberEvent(payload []byte) error {
	var memberEvent struct {
		Action      string      `json:"action"`
		Member      *GitHubMember `json:"member"`
		Organization *GitHubOrganization `json:"organization"`
	}

	if err := json.Unmarshal(payload, &memberEvent); err != nil {
		return fmt.Errorf("failed to parse member event: %w", err)
	}

	log.Printf("GitHub Member event: action=%s, member=%s, org=%s", memberEvent.Action, memberEvent.Member.Login, memberEvent.Organization.Login)

	switch memberEvent.Action {
	case "added":
		log.Printf("Member %s added to organization %s", memberEvent.Member.Login, memberEvent.Organization.Login)
	case "removed":
		log.Printf("Member %s removed from organization %s", memberEvent.Member.Login, memberEvent.Organization.Login)
	default:
		log.Printf("Unhandled member action: %s", memberEvent.Action)
	}

	return nil
}

// RegisterGitHubRoutes registers the GitHub webhook endpoint on an http.ServeMux
func RegisterGitHubRoutes(mux *http.ServeMux, adapter *githubAdapter) {
	mux.HandleFunc("/github/webhook", func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify webhook signature
		if !adapter.VerifyWebhook(r) {
			log.Printf("Webhook signature verification failed")
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}

		// Parse the webhook
		eventType, payload, err := adapter.ParseWebhook(r)
		if err != nil {
			log.Printf("Failed to parse webhook: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		log.Printf("Received GitHub webhook: event=%s", eventType)

		// Handle based on event type
		switch eventType {
		case "pull_request":
			if err := adapter.HandlePREvent(payload); err != nil {
				log.Printf("Failed to handle PR event: %v", err)
			}
		case "issue":
			if err := adapter.HandleIssueEvent(payload); err != nil {
				log.Printf("Failed to handle issue event: %v", err)
			}
		case "comment":
			if err := adapter.HandleCommentEvent(payload); err != nil {
				log.Printf("Failed to handle comment event: %v", err)
			}
		case "member":
			if err := adapter.HandleMemberEvent(payload); err != nil {
				log.Printf("Failed to handle member event: %v", err)
			}
		default:
			log.Printf("Unhandled GitHub event type: %s", eventType)
		}

		// Always respond with 200 OK to acknowledge receipt
		w.WriteHeader(http.StatusOK)
	})
}