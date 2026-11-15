package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webguard"
)

// MCPServer exposes the factory as an MCP server
type MCPServer struct {
	factory    *factory.FactoryOrchestrator
	factoryDef *factorydef.FactoryDefinition
	mu         sync.RWMutex
	server     *http.Server
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewMCPServer creates a new MCP server
func NewMCPServer(factory *factory.FactoryOrchestrator, factoryDef *factorydef.FactoryDefinition) *MCPServer {
	return &MCPServer{
		factory:    factory,
		factoryDef: factoryDef,
	}
}

// Start serves MCP over HTTP. The endpoint can submit credential-spending
// tasks, so it carries the same cross-site/remote guard as the web API.
func (s *MCPServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)

	s.server = &http.Server{
		Addr:    addr,
		Handler: webguard.Guard(mux, "/mcp"),
	}

	return s.server.ListenAndServe()
}

// Stop stops the MCP server
func (s *MCPServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleMCP handles incoming MCP requests
func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, req.ID, -32700, "Parse error", err.Error())
		return
	}

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result, err = s.handleInitialize(req.Params)
	case "tools/list":
		result, err = s.handleToolsList(req.Params)
	case "tools/call":
		result, err = s.handleToolsCall(req.Params)
	case "resources/list":
		result, err = s.handleResourcesList(req.Params)
	case "resources/read":
		result, err = s.handleResourcesRead(req.Params)
	default:
		err = fmt.Errorf("method not found: %s", req.Method)
	}

	if err != nil {
		s.writeError(w, req.ID, -32603, "Internal error", err.Error())
		return
	}

	s.writeResult(w, req.ID, result)
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "asf-factory",
			"version": "1.0.0",
		},
	}, nil
}

// handleToolsList returns the list of available tools
func (s *MCPServer) handleToolsList(params map[string]interface{}) (interface{}, error) {
	tools := []map[string]interface{}{
		{
			"name":        "submit_task",
			"description": "Submit a task to the agent factory",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Task type: generate, verify, review, refactor",
						"enum":        []string{"generate", "verify", "review", "refactor"},
					},
					"spec_id": map[string]interface{}{
						"type":        "string",
						"description": "Specification ID to work on",
					},
					"priority": map[string]interface{}{
						"type":        "integer",
						"description": "Task priority (higher = more urgent)",
						"default":     1,
					},
				},
				"required": []string{"type", "spec_id"},
			},
		},
		{
			"name":        "get_task_result",
			"description": "Get the result of a submitted task",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to retrieve",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "get_factory_status",
			"description": "Get the current factory status",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "list_specs",
			"description": "List all available specifications",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_spec",
			"description": "Get a specific specification",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_id": map[string]interface{}{
						"type":        "string",
						"description": "Specification ID",
					},
				},
				"required": []string{"spec_id"},
			},
		},
		{
			"name":        "list_agents",
			"description": "List all agents in the factory",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "run_verification",
			"description": "Run the verification harness on current code",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall handles a tool call
func (s *MCPServer) handleToolsCall(params map[string]interface{}) (interface{}, error) {
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	switch name {
	case "submit_task":
		return s.submitTask(arguments)
	case "get_task_result":
		return s.getTaskResult(arguments)
	case "get_factory_status":
		return s.getFactoryStatus()
	case "list_specs":
		return s.listSpecs()
	case "get_spec":
		return s.getSpec(arguments)
	case "list_agents":
		return s.listAgents()
	case "run_verification":
		return s.runVerification()
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// submitTask submits a task to the factory
func (s *MCPServer) submitTask(args map[string]interface{}) (interface{}, error) {
	taskType, _ := args["type"].(string)
	specID, _ := args["spec_id"].(string)
	priority := 1
	if p, ok := args["priority"].(float64); ok {
		priority = int(p)
	}

	if taskType == "" || specID == "" {
		return nil, fmt.Errorf("type and spec_id are required")
	}

	task := &factory.Task{
		Type:     taskType,
		SpecID:   specID,
		Priority: priority,
	}

	if err := s.factory.SubmitTask(task); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"task_id": task.ID,
		"status":  "submitted",
	}, nil
}

// getTaskResult gets a task result
func (s *MCPServer) getTaskResult(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	task, ok := s.factory.GetTaskResult(taskID)
	if !ok {
		return map[string]interface{}{
			"error": "task not found",
		}, nil
	}

	return map[string]interface{}{
		"task_id":      task.ID,
		"type":         task.Type,
		"spec_id":      task.SpecID,
		"status":       task.Status,
		"result":       task.Result,
		"error":        task.Error,
		"retries":      task.Retries,
		"created_at":   task.CreatedAt,
		"started_at":   task.StartedAt,
		"completed_at": task.CompletedAt,
	}, nil
}

// getFactoryStatus gets the factory status
func (s *MCPServer) getFactoryStatus() (interface{}, error) {
	status := s.factory.GetFactoryStatus()
	return status, nil
}

// listSpecs lists all specifications
func (s *MCPServer) listSpecs() (interface{}, error) {
	if s.factoryDef == nil {
		return map[string]interface{}{
			"specs": []string{},
		}, nil
	}

	specs := make([]map[string]interface{}, 0)
	for _, w := range s.factoryDef.Factory.Workflows {
		specs = append(specs, map[string]interface{}{
			"id":    w.ID,
			"steps": len(w.Steps),
		})
	}

	return map[string]interface{}{
		"specs": specs,
	}, nil
}

// getSpec gets a specific specification
func (s *MCPServer) getSpec(args map[string]interface{}) (interface{}, error) {
	specID, _ := args["spec_id"].(string)
	if specID == "" {
		return nil, fmt.Errorf("spec_id is required")
	}

	if s.factoryDef == nil {
		return nil, fmt.Errorf("factory definition not loaded")
	}

	w, err := s.factoryDef.GetWorkflow(specID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":    w.ID,
		"steps": w.Steps,
	}, nil
}

// listAgents lists all agents
func (s *MCPServer) listAgents() (interface{}, error) {
	if s.factoryDef == nil {
		return map[string]interface{}{
			"agents": []string{},
		}, nil
	}

	agents := make([]map[string]interface{}, 0)
	for _, a := range s.factoryDef.Factory.Agents {
		agents = append(agents, map[string]interface{}{
			"id":           a.ID,
			"role":         a.Role,
			"capabilities": a.Capabilities,
			"model":        a.Model,
		})
	}

	return map[string]interface{}{
		"agents": agents,
	}, nil
}

// runVerification runs the verification harness
func (s *MCPServer) runVerification() (interface{}, error) {
	// In a real implementation, this would trigger the verification
	return map[string]interface{}{
		"status":  "started",
		"message": "Verification triggered",
	}, nil
}

// handleResourcesList lists available resources
func (s *MCPServer) handleResourcesList(params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"resources": []map[string]interface{}{
			{
				"uri":         "factory://status",
				"name":        "Factory Status",
				"description": "Current factory status and metrics",
				"mimeType":    "application/json",
			},
			{
				"uri":         "factory://specs",
				"name":        "Specifications",
				"description": "All available specifications",
				"mimeType":    "application/json",
			},
		},
	}, nil
}

// handleResourcesRead reads a resource
func (s *MCPServer) handleResourcesRead(params map[string]interface{}) (interface{}, error) {
	uri, _ := params["uri"].(string)

	switch uri {
	case "factory://status":
		status := s.factory.GetFactoryStatus()
		data, _ := json.Marshal(status)
		return map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     string(data),
				},
			},
		}, nil
	case "factory://specs":
		result, _ := s.listSpecs()
		data, _ := json.Marshal(result)
		return map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     string(data),
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
}

// writeResult writes a successful response
func (s *MCPServer) writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// writeError writes an error response
func (s *MCPServer) writeError(w http.ResponseWriter, id interface{}, code int, message, data string) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still 200 OK
	json.NewEncoder(w).Encode(resp)
}
