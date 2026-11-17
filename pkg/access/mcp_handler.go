package access

import (
	"encoding/json"
	"fmt"
	"log"
)

type MCPToolHandler struct {
	github *GitHubMCPTool
}

func NewMCPToolHandler(githubToken string) *MCPToolHandler {
	return &MCPToolHandler{
		github: NewGitHubMCPTool(githubToken),
	}
}

// ExecuteTool runs an MCP tool call from an agent
func (h *MCPToolHandler) ExecuteTool(toolCall GitHubToolCall) GitHubToolResult {
	log.Printf("[MCP] Executing tool: %s with params: %v", toolCall.Tool, toolCall.Params)

	result := h.github.Call(toolCall.Tool, toolCall.Params)

	if result.Success {
		log.Printf("[MCP] Tool succeeded: %s", toolCall.Tool)
	} else {
		log.Printf("[MCP] Tool failed: %s - %s", toolCall.Tool, result.Error)
	}

	return result
}

// ExecuteToolFromJSON parses and executes a tool call from JSON
func (h *MCPToolHandler) ExecuteToolFromJSON(jsonStr string) (GitHubToolResult, error) {
	var toolCall GitHubToolCall
	if err := json.Unmarshal([]byte(jsonStr), &toolCall); err != nil {
		return GitHubToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse tool call: %v", err),
		}, err
	}

	return h.ExecuteTool(toolCall), nil
}

// GetToolDefinitions returns available tool definitions for agents
func (h *MCPToolHandler) GetToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "get_issue",
			"description": "Fetch GitHub issue details",
			"params": map[string]interface{}{
				"owner": "string (required)",
				"repo":  "string (required)",
				"issue": "number (required)",
			},
		},
		{
			"name":        "create_issue",
			"description": "Create a new GitHub issue",
			"params": map[string]interface{}{
				"owner": "string (required)",
				"repo":  "string (required)",
				"title": "string (required)",
				"body":  "string (optional)",
			},
		},
		{
			"name":        "add_labels",
			"description": "Add labels to an issue",
			"params": map[string]interface{}{
				"owner":  "string (required)",
				"repo":   "string (required)",
				"issue":  "number (required)",
				"labels": "array of strings (required)",
			},
		},
		{
			"name":        "post_comment",
			"description": "Post a comment on an issue",
			"params": map[string]interface{}{
				"owner": "string (required)",
				"repo":  "string (required)",
				"issue": "number (required)",
				"body":  "string (required)",
			},
		},
		{
			"name":        "create_pr",
			"description": "Create a pull request",
			"params": map[string]interface{}{
				"owner":  "string (required)",
				"repo":   "string (required)",
				"title":  "string (required)",
				"body":   "string (optional)",
				"head":   "string (required) - branch name",
				"base":   "string (optional, default: main)",
			},
		},
	}
}
