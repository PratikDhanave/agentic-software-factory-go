package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	State string `json:"state"`
}

type TriageResult struct {
	ID          string   `json:"id"`
	Issue       int      `json:"issue"`
	Repo        string   `json:"repo"`
	Title       string   `json:"title"`
	Category    string   `json:"category"` // feature, bug, documentation, refactor
	Priority    string   `json:"priority"` // critical, high, medium, low
	ReadyToImpl  bool     `json:"ready_to_implement"`
	Reason      string   `json:"reason"`
	Labels      []string `json:"labels_to_apply"`
	GeneratedSpec string `json:"generated_spec,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

var triageStore = make(map[string]*TriageResult)

// HandleGitHubTriage analyzes an issue and determines if it's ready to implement
func HandleGitHubTriage(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	issueNum := r.URL.Query().Get("issue")
	token := r.URL.Query().Get("token")

	log.Printf("[Triage] Request: owner=%s repo=%s issue=%s token=%s", owner, repo, issueNum, token[:min(len(token), 15)]+"...")

	if owner == "" || repo == "" || issueNum == "" || token == "" {
		log.Printf("[Triage] Missing params: owner=%v repo=%v issue=%v token=%v", owner != "", repo != "", issueNum != "", token != "")
		http.Error(w, "owner, repo, issue, token required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[Triage] Response writer doesn't support flushing")
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sendSSE := func(status, msg string, progress int) {
		data := map[string]interface{}{
			"status":   status,
			"progress": progress,
			"message":  msg,
		}
		bytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", string(bytes))
		flusher.Flush()
	}

	sendSSE("connecting", "Connecting to GitHub", 10)
	time.Sleep(300 * time.Millisecond)

	// Fetch issue
	sendSSE("fetching", "Fetching issue details", 25)
	issue, err := getIssueFromGitHub(owner, repo, issueNum, token)
	if err != nil {
		log.Printf("[Triage] Failed to fetch issue: %v", err)
		sendSSE("error", fmt.Sprintf("Failed to fetch issue: %v", err), 0)
		return
	}
	log.Printf("[Triage] Issue fetched: %s", issue.Title)

	// Analyze issue
	sendSSE("analyzing", "Analyzing issue content", 45)
	time.Sleep(500 * time.Millisecond)
	triageResult := analyzeIssue(owner, repo, issue)

	// Generate spec if ready
	if triageResult.ReadyToImpl {
		sendSSE("spec-generation", "Generating implementation spec", 65)
		time.Sleep(300 * time.Millisecond)
		triageResult.GeneratedSpec = generateSpecFromIssue(issue)
	}

	// Apply labels
	sendSSE("labeling", "Applying labels to issue", 80)
	if err := applyLabelsToIssue(owner, repo, issueNum, token, triageResult.Labels); err != nil {
		sendSSE("warning", fmt.Sprintf("Triage complete but failed to apply labels: %v", err), 85)
	}

	// Post comment
	sendSSE("commenting", "Posting triage comment to issue", 90)
	if err := postTriageCommentToIssue(owner, repo, issueNum, token, triageResult); err != nil {
		sendSSE("warning", fmt.Sprintf("Triage complete but failed to comment: %v", err), 95)
	}

	mu.Lock()
	triageStore[triageResult.ID] = triageResult
	mu.Unlock()

	// Send final result with complete status
	finalData := map[string]interface{}{
		"status":                 "complete",
		"progress":               100,
		"message":                fmt.Sprintf("Triage complete: %s (%s)", triageResult.Category, triageResult.Priority),
		"id":                     triageResult.ID,
		"issue":                  triageResult.Issue,
		"repo":                   triageResult.Repo,
		"title":                  triageResult.Title,
		"category":               triageResult.Category,
		"priority":               triageResult.Priority,
		"ready_to_implement":     triageResult.ReadyToImpl,
		"reason":                 triageResult.Reason,
		"labels_to_apply":        triageResult.Labels,
		"generated_spec":         triageResult.GeneratedSpec,
		"created_at":             triageResult.CreatedAt,
	}
	finalBytes, _ := json.Marshal(finalData)
	fmt.Fprintf(w, "data: %s\n\n", string(finalBytes))
	flusher.Flush()
}

// getIssueFromGitHub fetches issue details
func getIssueFromGitHub(owner, repo, issueNum, token string) (*GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s", owner, repo, issueNum)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var issue GitHubIssue
	json.NewDecoder(resp.Body).Decode(&issue)
	return &issue, nil
}

// analyzeIssue determines category, priority, and readiness
func analyzeIssue(owner, repo string, issue *GitHubIssue) *TriageResult {
	result := &TriageResult{
		ID:        uuid.New().String(),
		Issue:     issue.Number,
		Repo:      owner + "/" + repo,
		Title:     issue.Title,
		CreatedAt: time.Now().Format(time.RFC3339),
		Labels:    []string{},
	}

	// Determine category from title and body
	titleLower := strings.ToLower(issue.Title)
	bodyLower := strings.ToLower(issue.Body)

	if strings.Contains(titleLower, "bug") || strings.Contains(bodyLower, "error") {
		result.Category = "bug"
	} else if strings.Contains(titleLower, "doc") || strings.Contains(bodyLower, "documentation") {
		result.Category = "documentation"
	} else if strings.Contains(titleLower, "refactor") || strings.Contains(bodyLower, "refactor") {
		result.Category = "refactor"
	} else {
		result.Category = "feature"
	}

	// Determine priority from keywords
	if strings.Contains(bodyLower, "critical") || strings.Contains(bodyLower, "urgent") ||
		strings.Contains(titleLower, "critical") {
		result.Priority = "critical"
	} else if strings.Contains(bodyLower, "high priority") || strings.Contains(titleLower, "high") {
		result.Priority = "high"
	} else if strings.Contains(bodyLower, "low priority") || strings.Contains(titleLower, "low") {
		result.Priority = "low"
	} else {
		result.Priority = "medium"
	}

	// Determine readiness
	hasAcceptanceCriteria := strings.Contains(bodyLower, "acceptance") ||
		strings.Contains(bodyLower, "steps") ||
		strings.Contains(bodyLower, "expected")

	hasContext := len(issue.Body) > 100

	hasNoBlockers := !strings.Contains(bodyLower, "blocked") &&
		!strings.Contains(bodyLower, "needs clarification") &&
		!strings.Contains(bodyLower, "?")

	result.ReadyToImpl = hasAcceptanceCriteria && hasContext && hasNoBlockers

	if result.ReadyToImpl {
		result.Reason = "Has clear acceptance criteria, sufficient context, and no blockers"
		result.Labels = append(result.Labels, "ready-to-implement")
		result.Labels = append(result.Labels, fmt.Sprintf("priority-%s", result.Priority))
	} else {
		if !hasAcceptanceCriteria {
			result.Reason = "Needs acceptance criteria or clear expected behavior"
			result.Labels = append(result.Labels, "needs-info")
		} else if !hasContext {
			result.Reason = "Needs more context and details"
			result.Labels = append(result.Labels, "needs-info")
		} else {
			result.Reason = "Blocked by dependencies or unclear requirements"
			result.Labels = append(result.Labels, "wait-to-implement")
		}
	}

	result.Labels = append(result.Labels, fmt.Sprintf("category-%s", result.Category))

	return result
}

// generateSpecFromIssue creates a YAML spec from issue
func generateSpecFromIssue(issue *GitHubIssue) string {
	spec := fmt.Sprintf(`id: "issue-%d"
version: "1.0"
title: "%s"
description: "%s"

intent:
  goal: "Implement the requirement described in the issue"
  context: "GitHub issue: %s"
  success_criteria:
    - "Code compiles without errors"
    - "Implementation matches issue requirements"
    - "Tests pass if applicable"
    - "No new warnings or errors introduced"

constraints:
  - id: "c1"
    type: "must"
    description: "Follow Go best practices and conventions"
  - id: "c2"
    type: "must"
    description: "Include appropriate error handling"
  - id: "c3"
    type: "should"
    description: "Add unit tests for new functionality"

steps:
  - id: "s1"
    name: "Analyze requirements"
    description: "Understand the issue and requirements"
  - id: "s2"
    name: "Implement solution"
    description: "Write code to address the issue"
  - id: "s3"
    name: "Test thoroughly"
    description: "Verify the implementation works correctly"
  - id: "s4"
    name: "Review code quality"
    description: "Ensure code meets standards"
`,
		issue.Number,
		issue.Title,
		sanitizeForYAML(issue.Body),
		issue.Title,
	)
	return spec
}

// sanitizeForYAML escapes special characters
func sanitizeForYAML(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// applyLabelsToIssue adds labels to the issue
func applyLabelsToIssue(owner, repo, issueNum, token string, labels []string) error {
	payload := map[string][]string{
		"labels": labels,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s/labels", owner, repo, issueNum)
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// postTriageCommentToIssue posts a triage summary comment
func postTriageCommentToIssue(owner, repo, issueNum, token string, result *TriageResult) error {
	body := fmt.Sprintf(`## Automatic Triage Report

**Category:** %s
**Priority:** %s
**Status:** %s

**Analysis:**
%s

**Labels Applied:** %s
`,
		result.Category,
		result.Priority,
		map[bool]string{true: "✅ Ready to Implement", false: "⏳ Needs Review"}[result.ReadyToImpl],
		result.Reason,
		strings.Join(result.Labels, ", "),
	)

	if result.ReadyToImpl && result.GeneratedSpec != "" {
		body += "\n**Implementation Spec Generated:** Ready for code generation\n"
	}

	body += "\n_Analyzed by 7-Layer Automatic Triage Agent_"

	payload := map[string]string{
		"body": body,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s/comments", owner, repo, issueNum)
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HandleTriageStatus returns stored triage result
func HandleTriageStatus(w http.ResponseWriter, r *http.Request) {
	triageID := r.URL.Query().Get("id")
	if triageID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	result, ok := triageStore[triageID]
	mu.RUnlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
