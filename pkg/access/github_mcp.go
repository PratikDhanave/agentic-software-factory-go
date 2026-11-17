package access

import (
	"encoding/json"
	"fmt"
)

type GitHubMCPTool struct {
	token string
}

func NewGitHubMCPTool(token string) *GitHubMCPTool {
	return &GitHubMCPTool{token: token}
}

// Tool definitions that agents can call
type GitHubToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

type GitHubToolResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (g *GitHubMCPTool) Call(toolName string, params map[string]interface{}) GitHubToolResult {
	switch toolName {
	case "get_issue":
		return g.getIssue(params)
	case "create_issue":
		return g.createIssue(params)
	case "update_issue":
		return g.updateIssue(params)
	case "add_labels":
		return g.addLabels(params)
	case "post_comment":
		return g.postComment(params)
	case "create_pr":
		return g.createPR(params)
	default:
		return GitHubToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", toolName),
		}
	}
}

func (g *GitHubMCPTool) getIssue(params map[string]interface{}) GitHubToolResult {
	owner, ok := params["owner"].(string)
	if !ok {
		return GitHubToolResult{Success: false, Error: "owner required"}
	}
	repo, ok := params["repo"].(string)
	if !ok {
		return GitHubToolResult{Success: false, Error: "repo required"}
	}
	issue, ok := params["issue"].(float64)
	if !ok {
		return GitHubToolResult{Success: false, Error: "issue required"}
	}

	// Would call actual GitHub API here
	return GitHubToolResult{
		Success: true,
		Result: map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"issue": int(issue),
			"title": "Sample Issue",
		},
	}
}

func (g *GitHubMCPTool) createIssue(params map[string]interface{}) GitHubToolResult {
	title, ok := params["title"].(string)
	if !ok {
		return GitHubToolResult{Success: false, Error: "title required"}
	}
	return GitHubToolResult{
		Success: true,
		Result: map[string]interface{}{
			"issue_number": 42,
			"title":        title,
			"url":          "https://github.com/owner/repo/issues/42",
		},
	}
}

func (g *GitHubMCPTool) updateIssue(params map[string]interface{}) GitHubToolResult {
	return GitHubToolResult{Success: true, Result: "Issue updated"}
}

func (g *GitHubMCPTool) addLabels(params map[string]interface{}) GitHubToolResult {
	labels, ok := params["labels"].([]interface{})
	if !ok {
		return GitHubToolResult{Success: false, Error: "labels required"}
	}
	return GitHubToolResult{
		Success: true,
		Result: map[string]interface{}{
			"labels_added": len(labels),
		},
	}
}

func (g *GitHubMCPTool) postComment(params map[string]interface{}) GitHubToolResult {
	body, ok := params["body"].(string)
	if !ok {
		return GitHubToolResult{Success: false, Error: "body required"}
	}
	return GitHubToolResult{
		Success: true,
		Result: map[string]interface{}{
			"comment": body[:min(len(body), 50)] + "...",
		},
	}
}

func (g *GitHubMCPTool) createPR(params map[string]interface{}) GitHubToolResult {
	title, ok := params["title"].(string)
	if !ok {
		return GitHubToolResult{Success: false, Error: "title required"}
	}
	return GitHubToolResult{
		Success: true,
		Result: map[string]interface{}{
			"pr_number": 123,
			"title":     title,
			"url":       "https://github.com/owner/repo/pull/123",
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ToJSON serializes the tool call
func (g GitHubToolCall) ToJSON() string {
	data, _ := json.MarshalIndent(g, "", "  ")
	return string(data)
}
