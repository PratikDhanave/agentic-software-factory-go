package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/codegen"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
)

var factoryEngine *factory.WorkflowEngine

func init() {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "mistral"
	}
	factoryEngine = factory.NewWorkflowEngine(ollamaURL, ollamaModel)
}

// HandleSpecToCode: spec → code, tests, review, docs
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

	log.Printf("[Factory] SpecToCode: %s", spec.Title)

	result, err := factoryEngine.SpecToShip(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Workflow failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// HandleCodeReview: analyze code for issues
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

	log.Printf("[Factory] CodeReview: %s", req.FileName)

	result, err := factoryEngine.ReviewAndFix(req.Code, req.Language)
	if err != nil {
		http.Error(w, fmt.Sprintf("Review failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// HandleGenerateCode: spec → code only
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

	cg := codegen.NewCodeGenerator(os.Getenv("OLLAMA_URL"), os.Getenv("OLLAMA_MODEL"))
	generated, err := cg.Generate(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(generated)
}

// HandleExecuteTask: complex multi-step tasks
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

	log.Printf("[Factory] ExecuteTask: %s", req.Task)

	result, err := factoryEngine.ExecuteTask(req.Task)
	if err != nil {
		http.Error(w, fmt.Sprintf("Task failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// HandleStreamTask: same as ExecuteTask but with SSE streaming
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
	json.Unmarshal(body, &req)

	flusher := w.(http.Flusher)

	// Send initial status
	data := map[string]interface{}{
		"status":   "starting",
		"progress": 10,
		"message":  "Initializing task execution...",
	}
	bytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", string(bytes))
	flusher.Flush()

	// Execute task
	result, err := factoryEngine.ExecuteTask(req.Task)

	if err != nil {
		data := map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Task failed: %v", err),
		}
		bytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", string(bytes))
		flusher.Flush()
		return
	}

	// Send final result
	data = map[string]interface{}{
		"status":   "complete",
		"progress": 100,
		"result":   result,
	}
	bytes, _ = json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", string(bytes))
	flusher.Flush()
}
