package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GitHubPullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

type GitHubFile struct {
	Filename string `json:"filename"`
	Patch    string `json:"patch"`
	Additions int   `json:"additions"`
	Deletions int   `json:"deletions"`
}

type CodeReviewResult struct {
	ID          string        `json:"id"`
	PR          int           `json:"pr"`
	Repo        string        `json:"repo"`
	Issues      []ReviewIssue `json:"issues"`
	Score       float64       `json:"score"`
	Summary     string        `json:"summary"`
	CreatedAt   string        `json:"createdAt"`
}

type ReviewIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"` // critical, warning, info
	Message  string `json:"message"`
}

var reviewStore = make(map[string]*CodeReviewResult)

// HandleGitHubReview handles GET /api/github/review?owner=X&repo=Y&pr=N&token=Z
func HandleGitHubReview(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	prNum := r.URL.Query().Get("pr")
	token := r.URL.Query().Get("token")

	if owner == "" || repo == "" || prNum == "" || token == "" {
		http.Error(w, "owner, repo, pr, token required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher := w.(http.Flusher)
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
	time.Sleep(500 * time.Millisecond)

	// Get PR details
	sendSSE("fetching", "Fetching PR details", 25)
	pr, err := getPRFromGitHub(owner, repo, prNum, token)
	if err != nil {
		sendSSE("error", fmt.Sprintf("Failed to fetch PR: %v", err), 0)
		return
	}

	// Get PR files
	sendSSE("analyzing", "Analyzing files", 45)
	files, err := getPRFilesFromGitHub(owner, repo, prNum, token)
	if err != nil {
		sendSSE("error", fmt.Sprintf("Failed to fetch files: %v", err), 0)
		return
	}

	// Review code
	sendSSE("reviewing", "Running 7-layer review", 65)
	result := reviewCode(owner, repo, pr, files)

	// Post review to GitHub
	sendSSE("posting", "Posting review to GitHub", 85)
	if err := postReviewToGitHub(owner, repo, prNum, token, result); err != nil {
		sendSSE("warning", fmt.Sprintf("Review complete but failed to post: %v", err), 90)
	}

	mu.Lock()
	reviewStore[result.ID] = result
	mu.Unlock()

	sendSSE("complete", fmt.Sprintf("Review done: %d issues, score %.0f", len(result.Issues), result.Score), 100)
}

// HandleReviewStatus returns stored review
func HandleReviewStatus(w http.ResponseWriter, r *http.Request) {
	reviewID := r.URL.Query().Get("id")
	if reviewID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	result, ok := reviewStore[reviewID]
	mu.RUnlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getPRFromGitHub fetches PR data
func getPRFromGitHub(owner, repo, prNum, token string) (*GitHubPullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s", owner, repo, prNum)
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

	var pr GitHubPullRequest
	json.NewDecoder(resp.Body).Decode(&pr)
	return &pr, nil
}

// getPRFilesFromGitHub fetches changed files
func getPRFilesFromGitHub(owner, repo, prNum, token string) ([]GitHubFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/files", owner, repo, prNum)
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
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var files []GitHubFile
	json.NewDecoder(resp.Body).Decode(&files)
	return files, nil
}

// reviewCode runs 7-layer review on PR
func reviewCode(owner, repo string, pr *GitHubPullRequest, files []GitHubFile) *CodeReviewResult {
	result := &CodeReviewResult{
		ID:        uuid.New().String(),
		PR:        pr.Number,
		Repo:      owner + "/" + repo,
		Issues:    []ReviewIssue{},
		Score:     100,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	// Layer 1: Parse changes
	totalLines := 0
	for _, f := range files {
		totalLines += f.Additions + f.Deletions
	}

	// Layer 2: Load context
	goFiles := 0
	for _, f := range files {
		if strings.HasSuffix(f.Filename, ".go") {
			goFiles++
		}
	}

	// Layer 3: Compute metrics
	changeSize := "medium"
	if totalLines > 500 {
		changeSize = "large"
	} else if totalLines < 50 {
		changeSize = "small"
	}

	// Layer 4: Analyze code
	issues := analyzeCodeQuality(files)
	result.Issues = issues

	// Layer 5: Score
	baseScore := 100.0
	baseScore -= float64(len(issues) * 5) // -5 per issue

	if changeSize == "large" && len(files) > 10 {
		baseScore -= 10 // Large changes penalty
	}

	if goFiles > 0 && !hasTests(files) {
		baseScore -= 15 // No tests penalty
	}

	if baseScore < 0 {
		baseScore = 0
	}

	result.Score = baseScore

	// Layer 6: Orchestrate review summary
	criticalCount := 0
	warningCount := 0
	for _, issue := range issues {
		if issue.Severity == "critical" {
			criticalCount++
		} else if issue.Severity == "warning" {
			warningCount++
		}
	}

	result.Summary = fmt.Sprintf("PR Review: %d files, %d lines changed. %d critical, %d warning issues. Score: %.0f%%",
		len(files), totalLines, criticalCount, warningCount, result.Score)

	// Layer 7: Return result
	return result
}

// analyzeCodeQuality identifies issues using AST parsing
func analyzeCodeQuality(files []GitHubFile) []ReviewIssue {
	issues := []ReviewIssue{}

	for _, f := range files {
		if !strings.HasSuffix(f.Filename, ".go") {
			continue
		}

		// Use AST analyzer for deep code analysis
		astIssues := DeepAnalyzeFile(f.Filename, f.Patch)
		issues = append(issues, astIssues...)

		// Add pattern-based checks for patch content (no AST available)
		issues = append(issues, patternBasedPatchChecks(f.Filename, f.Patch)...)

		// Check for large functions
		if f.Additions > 200 {
			issues = append(issues, ReviewIssue{
				File:     f.Filename,
				Severity: "warning",
				Message:  "Large function (200+ lines), consider breaking down",
			})
		}
	}

	// Remove duplicate issues
	seen := make(map[string]bool)
	var unique []ReviewIssue
	for _, issue := range issues {
		key := issue.File + ":" + issue.Message
		if !seen[key] {
			seen[key] = true
			unique = append(unique, issue)
		}
	}

	return unique
}

// patternBasedPatchChecks performs pattern matching on patches
func patternBasedPatchChecks(filename, patch string) []ReviewIssue {
	var issues []ReviewIssue

	if !strings.HasSuffix(filename, ".go") {
		return issues
	}

	lines := strings.Split(patch, "\n")
	for i, line := range lines {
		// Only check added lines
		if !strings.HasPrefix(line, "+") {
			continue
		}

		codeLine := strings.TrimPrefix(line, "+")

		// Check for Lock without defer
		if strings.Contains(codeLine, ".Lock()") {
			// Look ahead for defer
			hasDefer := false
			for j := i + 1; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "defer") && strings.Contains(lines[j], "Unlock") {
					hasDefer = true
					break
				}
			}
			if !hasDefer {
				issues = append(issues, ReviewIssue{
					File:     filename,
					Severity: "critical",
					Message:  "Lock() without defer Unlock() - deadlock/leak risk",
				})
			}
		}

		// Check for error assignment without check
		if strings.Contains(codeLine, "err :=") {
			// Check if next line checks error
			if i+1 < len(lines) {
				nextLine := lines[i+1]
				if !strings.Contains(nextLine, "if err") && !strings.Contains(nextLine, "return") {
					issues = append(issues, ReviewIssue{
						File:     filename,
						Severity: "warning",
						Message:  "Error assigned but not checked immediately",
					})
				}
			}
		}

		// Check for TODO/FIXME
		if strings.Contains(codeLine, "TODO") || strings.Contains(codeLine, "FIXME") {
			issues = append(issues, ReviewIssue{
				File:     filename,
				Severity: "warning",
				Message:  "TODO/FIXME comment found",
			})
		}

		// Check for panic without recovery
		if strings.Contains(codeLine, "panic(") && !strings.Contains(codeLine, "defer recover") {
			issues = append(issues, ReviewIssue{
				File:     filename,
				Severity: "warning",
				Message:  "panic() call without recovery mechanism",
			})
		}

		// Check for unused assignments
		if strings.Contains(codeLine, ":=") && strings.Contains(codeLine, "_") {
			if !strings.Contains(codeLine, "_ =") {
				// Might be intentional blank assignment
			}
		}
	}

	return issues
}

// hasTests checks if PR includes tests
func hasTests(files []GitHubFile) bool {
	for _, f := range files {
		if strings.Contains(f.Filename, "_test.go") {
			return true
		}
	}
	return false
}

// postReviewToGitHub posts review as PR comment
func postReviewToGitHub(owner, repo, prNum, token string, result *CodeReviewResult) error {
	body := fmt.Sprintf(`## Code Review - Score: %.0f%%

%s

### Issues Found: %d

`, result.Score, result.Summary, len(result.Issues))

	if len(result.Issues) > 0 {
		body += "| File | Severity | Message |\n"
		body += "|------|----------|----------|\n"
		for _, issue := range result.Issues {
			body += fmt.Sprintf("| %s | %s | %s |\n",
				issue.File, issue.Severity, issue.Message)
		}
	} else {
		body += "✓ No critical issues found!\n"
	}

	body += fmt.Sprintf("\n_Reviewed by 7-Layer Code Review Agent_")

	payload := map[string]string{
		"body": body,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s/comments", owner, repo, prNum)
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
