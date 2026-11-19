package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/platform/discovery"
)

// MCPCapability describes what a service can do
type MCPCapability struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`  // JSON Schema
	OutputSchema interface{} `json:"output_schema"` // JSON Schema
	Idempotent  bool        `json:"idempotent"`
}

// MCPRequest is a request to invoke a capability
type MCPRequest struct {
	Capability string      `json:"capability"`
	Input      interface{} `json:"input"`
}

// MCPResponse is a response from invoking a capability
type MCPResponse struct {
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Server exposes agent capabilities via MCP protocol
type Server struct {
	agentID      string
	capabilities map[string]*MCPCapability
	handlers     map[string]func(ctx context.Context, input interface{}) (interface{}, error)
}

// NewServer creates a new MCP server
func NewServer(agentID string) *Server {
	return &Server{
		agentID:      agentID,
		capabilities: make(map[string]*MCPCapability),
		handlers:     make(map[string]func(ctx context.Context, input interface{}) (interface{}, error)),
	}
}

// RegisterCapability adds a new capability
func (s *Server) RegisterCapability(cap *MCPCapability, handler func(ctx context.Context, input interface{}) (interface{}, error)) error {
	if cap.Name == "" {
		return fmt.Errorf("capability name required")
	}

	if handler == nil {
		return fmt.Errorf("handler required for capability %s", cap.Name)
	}

	s.capabilities[cap.Name] = cap
	s.handlers[cap.Name] = handler

	return nil
}

// ServeHTTP handles HTTP MCP requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/mcp/capabilities":
		s.handleCapabilities(w, r)
	case "/mcp/call":
		s.handleCall(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleCapabilities returns the list of capabilities
func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	capabilities := make([]*MCPCapability, 0, len(s.capabilities))
	for _, cap := range s.capabilities {
		capabilities = append(capabilities, cap)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent_id":      s.agentID,
		"capabilities": capabilities,
	})
}

// handleCall invokes a capability
func (s *Server) handleCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MCPResponse{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if req.Capability == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MCPResponse{Error: "capability name required"})
		return
	}

	handler, ok := s.handlers[req.Capability]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(MCPResponse{Error: fmt.Sprintf("capability not found: %s", req.Capability)})
		return
	}

	output, err := handler(r.Context(), req.Input)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MCPResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MCPResponse{Output: output})
}

// Client discovers and calls services via MCP
type Client struct {
	registry *discovery.Registry
	tenantID string
}

// NewClient creates a new MCP client
func NewClient(registry *discovery.Registry, tenantID string) *Client {
	return &Client{
		registry: registry,
		tenantID: tenantID,
	}
}

// ListCapabilities discovers service capabilities
func (c *Client) ListCapabilities(ctx context.Context, serviceName string) ([]*MCPCapability, error) {
	instances, err := c.registry.Discover(ctx, serviceName, c.tenantID)
	if err != nil {
		return nil, fmt.Errorf("discover service: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Use first healthy instance
	instance := instances[0]
	url := fmt.Sprintf("http://%s:%d/mcp/capabilities", instance.Address, instance.Port)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get capabilities: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	caps, ok := result["capabilities"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	var capabilities []*MCPCapability
	for _, c := range caps {
		b, _ := json.Marshal(c)
		var cap MCPCapability
		json.Unmarshal(b, &cap)
		capabilities = append(capabilities, &cap)
	}

	return capabilities, nil
}

// CallCapability invokes a service capability
func (c *Client) CallCapability(ctx context.Context, serviceName string, capabilityName string, input interface{}) (interface{}, error) {
	instances, err := c.registry.Discover(ctx, serviceName, c.tenantID)
	if err != nil {
		return nil, fmt.Errorf("discover service: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Use first healthy instance
	instance := instances[0]
	url := fmt.Sprintf("http://%s:%d/mcp/call", instance.Address, instance.Port)

	req := MCPRequest{
		Capability: capabilityName,
		Input:      input,
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("call capability: %w", err)
	}
	defer resp.Body.Close()

	var result MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("service error: %s", result.Error)
	}

	return result.Output, nil
}
