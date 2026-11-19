// Package mcp exposes the Factory's agents, specs, runs, and explicit finite
// state machines over Model Context Protocol (JSON-RPC 2.0) for external IDEs.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/engine"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/sandbox"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Handler struct {
	store  *store.Store
	engine *engine.Engine
}

func New(st *store.Store, eng *engine.Engine) *Handler {
	return &Handler{store: st, engine: eng}
}

func ToolsCatalog() []map[string]any {
	return []map[string]any{
		{
			"name":        "list_agents",
			"description": "List registered Factory specialist agents and their operational states.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "list_specs",
			"description": "List verified Go blueprint YAML specifications in specs/.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "start_run",
			"description": "Start an autonomous Go package synthesis & go test -race verification run.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt":  map[string]any{"type": "string", "description": "Plain-English Go package specification"},
					"spec_id": map[string]any{"type": "string", "description": "Optional blueprint ID (e.g. spec-token-bucket)"},
				},
			},
		},
		{
			"name":        "get_run",
			"description": "Inspect the current state, files, and test output of a Factory Run.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"run_id"},
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "list_state_machines",
			"description": "Introspect all explicit finite state machines, transitions, and Mermaid graphs.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "validate_transition",
			"description": "Check whether a state transition (from -> to) is legal on a named state machine.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"machine", "from", "to"},
				"properties": map[string]any{
					"machine": map[string]any{"type": "string"},
					"from":    map[string]any{"type": "string"},
					"to":      map[string]any{"type": "string"},
				},
			},
		},
	}
}

func (h *Handler) Handle(ctx context.Context, req RPCRequest) RPCResponse {
	ok := func(res any) RPCResponse {
		return RPCResponse{JSONRPC: "2.0", ID: req.ID, Result: res}
	}
	fail := func(code int, msg string) RPCResponse {
		return RPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: code, Message: msg}}
	}

	switch req.Method {
	case "initialize":
		return ok(map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "agentic-software-factory-go",
				"version": "2.0.0",
			},
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
		})

	case "tools/list":
		return ok(map[string]any{"tools": ToolsCatalog()})

	case "resources/list":
		return ok(map[string]any{
			"resources": []map[string]string{
				{
					"uri":         "factory://state-machines",
					"name":        "Explicit State Machine Registry",
					"mimeType":    "application/json",
					"description": "All 8 core Factory finite state machines with transitions and Mermaid diagrams.",
				},
			},
		})

	case "resources/read":
		b, _ := json.MarshalIndent(state.Exports(), "", "  ")
		return ok(map[string]any{
			"contents": []map[string]string{
				{
					"uri":      "factory://state-machines",
					"mimeType": "application/json",
					"text":     string(b),
				},
			},
		})

	case "tools/call":
		var call struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &call); err != nil {
			return fail(-32602, "invalid tool call params")
		}
		text, err := h.execTool(ctx, call.Name, call.Arguments)
		if err != nil {
			return fail(-32000, err.Error())
		}
		return ok(map[string]any{
			"content": []map[string]string{{"type": "text", "text": text}},
		})

	default:
		return fail(-32601, fmt.Sprintf("unknown MCP method %q", req.Method))
	}
}

func (h *Handler) execTool(ctx context.Context, name string, args map[string]any) (string, error) {
	strArg := func(k string) string {
		if v, ok := args[k].(string); ok {
			return v
		}
		return ""
	}

	switch name {
	case "list_agents":
		agents, err := h.store.ListAgents(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(agents, "", "  ")
		return string(b), nil

	case "list_specs":
		specs, err := sandbox.LoadSpecs(h.engine.SpecsDir())
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(specs, "", "  ")
		return string(b), nil

	case "start_run":
		r, err := h.engine.StartRun(ctx, engine.StartRequest{
			Prompt: strArg("prompt"),
			SpecID: strArg("spec_id"),
		})
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		return string(b), nil

	case "get_run":
		r, err := h.store.GetRun(ctx, strArg("run_id"))
		if err != nil {
			return "", err
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		return string(b), nil

	case "list_state_machines":
		b, _ := json.MarshalIndent(state.Exports(), "", "  ")
		return string(b), nil

	case "validate_transition":
		m, ok := state.ByName(strArg("machine"))
		if !ok {
			return "", fmt.Errorf("unknown machine %q", strArg("machine"))
		}
		from := state.State(strArg("from"))
		to := state.State(strArg("to"))
		err := m.Validate(from, to)
		res := map[string]any{
			"machine": m.Name(),
			"from":    from,
			"to":      to,
			"valid":   err == nil,
			"allowed": m.AllowedFrom(from),
		}
		if err != nil {
			res["reason"] = err.Error()
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return string(b), nil

	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}
