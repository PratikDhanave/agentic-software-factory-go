package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/integration"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/queue"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webguard"
)

// MCPServer exposes the factory as an MCP server
type MCPServer struct {
	factory       *factory.FactoryOrchestrator
	factoryDef    *factorydef.FactoryDefinition
	stateMachines []*state.Machine
	mu            sync.RWMutex
	server        *http.Server
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
func NewMCPServer(f *factory.FactoryOrchestrator, factoryDef *factorydef.FactoryDefinition) *MCPServer {
	defaults := append([]*state.Machine{}, factory.Machines()...)
	defaults = append(defaults,
		queue.QueueJobMachine,
		harness.HarnessMachine,
		verify.PhaseMachine,
		integration.ConnectionMachine,
	)
	return &MCPServer{
		factory:       f,
		factoryDef:    factoryDef,
		stateMachines: defaults,
	}
}

// WithStateMachines overrides the catalog of explicit state machines exposed via MCP tools and resources.
func (s *MCPServer) WithStateMachines(machines []*state.Machine) *MCPServer {
	if len(machines) > 0 {
		s.stateMachines = machines
	}
	return s
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

// ServeHTTP allows mounting MCPServer directly onto an existing HTTP mux (e.g. asf-web).
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handleMCP(w, r)
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
						"description": "Task type: generate, verify, review, refactor, evaluate (judge the latest generate output)",
						"enum":        []string{"generate", "verify", "review", "refactor", "evaluate"},
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
		{
			"name":        "list_state_machines",
			"description": "List all explicit state machines governing the software factory (states, transitions, terminal states, descriptions, and Mermaid diagrams)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category filter: assembly, governance, orchestration, execution, verification, infrastructure",
					},
				},
			},
		},
		{
			"name":        "inspect_state_machine",
			"description": "Inspect a state machine and optionally validate whether a state transition is legal",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"machine": map[string]interface{}{
						"type":        "string",
						"description": "State machine name (e.g. factory-assembly-line, coordination-session, task)",
					},
					"state": map[string]interface{}{
						"type":        "string",
						"description": "Current state to inspect",
					},
					"next_state": map[string]interface{}{
						"type":        "string",
						"description": "Optional target state to validate transition from state -> next_state",
					},
				},
				"required": []string{"machine", "state"},
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
	case "list_state_machines":
		return s.listStateMachines(arguments)
	case "inspect_state_machine":
		return s.inspectStateMachine(arguments)
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

// listSpecs lists all specifications from SpecEngine (plus factory.yaml workflows if defined).
func (s *MCPServer) listSpecs() (interface{}, error) {
	specs := make([]map[string]interface{}, 0)

	if s.factory != nil && s.factory.SpecEngine() != nil {
		if loaded, err := s.factory.SpecEngine().LoadAllSpecs(); err == nil {
			for id, sp := range loaded {
				specs = append(specs, map[string]interface{}{
					"id":          id,
					"title":       sp.Title,
					"version":     sp.Version,
					"description": sp.Description,
					"steps":       len(sp.Steps),
				})
			}
		}
	}

	if len(specs) == 0 && s.factoryDef != nil {
		for _, w := range s.factoryDef.Factory.Workflows {
			specs = append(specs, map[string]interface{}{
				"id":    w.ID,
				"steps": len(w.Steps),
			})
		}
	}

	return map[string]interface{}{
		"specs": specs,
	}, nil
}

// getSpec gets a specific specification from SpecEngine (falling back to factory.yaml workflows).
func (s *MCPServer) getSpec(args map[string]interface{}) (interface{}, error) {
	specID, _ := args["spec_id"].(string)
	if specID == "" {
		return nil, fmt.Errorf("spec_id is required")
	}

	if s.factory != nil && s.factory.SpecEngine() != nil {
		if loaded, err := s.factory.SpecEngine().LoadAllSpecs(); err == nil {
			if sp, ok := loaded[specID]; ok {
				return sp, nil
			}
		}
	}

	if s.factoryDef == nil {
		return nil, fmt.Errorf("spec %s not found", specID)
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

// runVerification runs the real VerificationEngine on the workspace.
func (s *MCPServer) runVerification() (interface{}, error) {
	if s.factory == nil || s.factory.Verifier() == nil {
		return nil, fmt.Errorf("verification engine not initialized")
	}
	res, err := s.factory.Verifier().RunVerification(context.Background(), "")
	return map[string]interface{}{
		"success":      res != nil && res.Success,
		"verification": res,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	}, nil
}

func (s *MCPServer) listStateMachines(args map[string]interface{}) (interface{}, error) {
	categoryFilter, _ := args["category"].(string)
	exports := make([]state.Export, 0, len(s.stateMachines))
	for _, m := range s.stateMachines {
		if categoryFilter != "" && m.Category() != categoryFilter {
			continue
		}
		exports = append(exports, m.Export())
	}
	return map[string]interface{}{
		"count":    len(exports),
		"machines": exports,
	}, nil
}

func (s *MCPServer) inspectStateMachine(args map[string]interface{}) (interface{}, error) {
	machineName, _ := args["machine"].(string)
	curState, _ := args["state"].(string)
	nextState, _ := args["next_state"].(string)
	if machineName == "" || curState == "" {
		return nil, fmt.Errorf("machine and state are required")
	}
	var target *state.Machine
	for _, m := range s.stateMachines {
		if m.Name() == machineName {
			target = m
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("state machine %q not found", machineName)
	}

	insp := target.Inspect(state.State(curState))
	out := map[string]interface{}{
		"inspection": insp,
	}
	if nextState != "" {
		err := target.Validate(state.State(curState), state.State(nextState))
		out["next_state"] = nextState
		out["can_transition"] = err == nil
		if err != nil {
			out["validation_error"] = err.Error()
		}
	}
	return out, nil
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
			{
				"uri":         "factory://state-machines",
				"name":        "Explicit State Machines",
				"description": "Formal state transition tables and Mermaid diagrams for every factory lifecycle",
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
	case "factory://state-machines":
		result, _ := s.listStateMachines(nil)
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
