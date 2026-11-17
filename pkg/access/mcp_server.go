package access

import (
	"encoding/json"
	"fmt"
	"log"
)

// MCP protocol structures (JSON-RPC 2.0 compatible)
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPServer struct {
	github *GitHubMCPTool
	name   string
	version string
}

func NewMCPServer(token string) *MCPServer {
	return &MCPServer{
		github:  NewGitHubMCPTool(token),
		name:    "agentic-software-factory",
		version: "1.0.0",
	}
}

// ProcessRequest handles incoming MCP requests
func (s *MCPServer) ProcessRequest(req MCPRequest) MCPResponse {
	log.Printf("[MCP] Request: method=%s id=%v", req.Method, req.ID)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *MCPServer) handleInitialize(req MCPRequest) MCPResponse {
	result := map[string]interface{}{
		"name":       s.name,
		"version":    s.version,
		"capabilities": map[string]interface{}{
			"tools": true,
		},
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *MCPServer) handleToolsList(req MCPRequest) MCPResponse {
	tools := s.getToolDefinitions()

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleToolCall(req MCPRequest) MCPResponse {
	var params struct {
		Name   string                 `json:"name"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32700,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	log.Printf("[MCP] Calling tool: %s", params.Name)

	result := s.github.Call(params.Name, params.Params)

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *MCPServer) getToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "get_issue",
			"description": "Fetch GitHub issue details",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]string{"type": "string", "description": "GitHub owner"},
					"repo":  map[string]string{"type": "string", "description": "Repository name"},
					"issue": map[string]string{"type": "string", "description": "Issue number"},
				},
				"required": []string{"owner", "repo", "issue"},
			},
		},
		{
			"name":        "create_issue",
			"description": "Create a new GitHub issue",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]string{"type": "string"},
					"repo":  map[string]string{"type": "string"},
					"title": map[string]string{"type": "string"},
					"body":  map[string]string{"type": "string"},
				},
				"required": []string{"owner", "repo", "title"},
			},
		},
		{
			"name":        "add_labels",
			"description": "Add labels to a GitHub issue",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner":  map[string]string{"type": "string"},
					"repo":   map[string]string{"type": "string"},
					"issue":  map[string]string{"type": "string"},
					"labels": map[string]string{"type": "array"},
				},
				"required": []string{"owner", "repo", "issue", "labels"},
			},
		},
		{
			"name":        "post_comment",
			"description": "Post a comment on a GitHub issue",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]string{"type": "string"},
					"repo":  map[string]string{"type": "string"},
					"issue": map[string]string{"type": "string"},
					"body":  map[string]string{"type": "string"},
				},
				"required": []string{"owner", "repo", "issue", "body"},
			},
		},
		{
			"name":        "create_pr",
			"description": "Create a pull request",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]string{"type": "string"},
					"repo":  map[string]string{"type": "string"},
					"title": map[string]string{"type": "string"},
					"head":  map[string]string{"type": "string"},
					"base":  map[string]string{"type": "string"},
				},
				"required": []string{"owner", "repo", "title", "head"},
			},
		},
	}
}
