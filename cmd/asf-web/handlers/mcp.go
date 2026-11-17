package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/access"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/agents"
)

type MCPTaskRequest struct {
	AgentName string                 `json:"agent_name"`
	Task      string                 `json:"task"`
	Tool      string                 `json:"tool,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

type MCPTaskResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// HandleMCPTask executes a task using MCP agent
func HandleMCPTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(MCPTaskResponse{
			Success: false,
			Error:   "POST required",
		})
		return
	}

	var req MCPTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(MCPTaskResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Create agent
	agent := agents.NewMCPAgent(req.AgentName)

	// If tool call requested, execute it
	if req.Tool != "" {
		result, err := agent.CallTool(req.Tool, req.Params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(MCPTaskResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MCPTaskResponse{
			Success: result.Success,
			Result:  result.Result,
			Error:   result.Error,
		})
		return
	}

	// Otherwise execute general task
	taskResult, err := agent.ExecuteTask(req.Task)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MCPTaskResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MCPTaskResponse{
		Success: true,
		Result:  taskResult,
	})
}

// HandleMCPTools returns available MCP tools
func HandleMCPTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	agent := agents.NewMCPAgent("info-agent")
	tools := agent.GetTools()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}

// HandleMCPServer implements MCP protocol (JSON-RPC 2.0)
func HandleMCPServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read body"})
		return
	}
	defer r.Body.Close()

	// Parse MCP request
	var req access.MCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[MCP] Parse error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	// Create MCP server
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Printf("[MCP] Warning: GITHUB_TOKEN not set")
	}

	mcpServer := access.NewMCPServer(token)

	// Process request
	response := mcpServer.ProcessRequest(req)

	// Send response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleMCPInfo returns MCP server info
func HandleMCPInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":       "agentic-software-factory",
		"version":    "1.0.0",
		"protocols":  []string{"mcp"},
		"endpoints": map[string]string{
			"rpc":    "/api/mcp/rpc",
			"tools":  "/api/mcp/tools",
			"info":   "/api/mcp/info",
		},
	})
}
