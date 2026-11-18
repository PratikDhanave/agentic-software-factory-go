package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/bootstrap"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/codegen"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

var activeContainer *bootstrap.Container

// SetContainer wires the unified Clean Architecture DI container into HTTP handlers.
func SetContainer(c *bootstrap.Container) {
	activeContainer = c
	if c != nil {
		SetSQLiteStore(c.Store)
	}
}

func workflowEngine() *factory.WorkflowEngine {
	if activeContainer != nil && activeContainer.PromptClient != nil {
		return factory.NewWorkflowEngineWithClient(activeContainer.PromptClient)
	}
	return factory.NewWorkflowEngineWithClient(llm.NewDeterministicClient())
}

// HandleSpecToCode executes spec -> code, tests, bug scan, review, and docs via the unified LLM gateway.
func HandleSpecToCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var spec codegen.Specification
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, "Invalid spec", http.StatusBadRequest)
		return
	}

	result, err := workflowEngine().SpecToShip(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Workflow failed: %v", err), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// HandleCodeReview analyzes code for bugs, security, and quality via the unified LLM gateway.
func HandleCodeReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code     string `json:"code"`
		Language string `json:"language"`
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Language == "" {
		req.Language = "go"
	}

	result, err := workflowEngine().ReviewAndFix(req.Code, req.Language)
	if err != nil {
		http.Error(w, fmt.Sprintf("Review failed: %v", err), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// HandleGenerateCode generates code from a specification via the unified LLM gateway.
func HandleGenerateCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var spec codegen.Specification
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, "Invalid spec", http.StatusBadRequest)
		return
	}

	var client llm.Client = llm.NewDeterministicClient()
	if activeContainer != nil && activeContainer.PromptClient != nil {
		client = activeContainer.PromptClient
	}
	cg := codegen.NewCodeGeneratorWithClient(client)
	generated, err := cg.Generate(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Generation failed: %v", err), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(generated)
}

// HandleExecuteTask executes multi-step tasks via the unified LLM gateway.
func HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Task string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	result, err := workflowEngine().ExecuteTask(req.Task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Task failed: %v", err), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// HandleStreamTask streams multi-step task execution via SSE.
func HandleStreamTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Task string `json:"task"`
	}
	_ = json.Unmarshal(body, &req)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	send := func(payload map[string]interface{}) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", string(b))
		flusher.Flush()
	}

	send(map[string]interface{}{"status": "starting", "progress": 15, "message": "Dispatching task to factory gateway..."})
	result, err := workflowEngine().ExecuteTask(req.Task)
	if err != nil {
		send(map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	send(map[string]interface{}{"status": "complete", "progress": 100, "result": result})
}
