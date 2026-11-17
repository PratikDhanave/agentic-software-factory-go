// Command asf-web serves the factory HTTP API consumed by the Next.js
// dashboard (web-ui/). Task lifecycle semantics come from factory.TaskMachine;
// this server never hardcodes state names or terminality.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/cmd/asf-web/handlers"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webguard"
)

// launchTaskTypes maps a dashboard launch action to the factory task type
// that implements it today.
var launchTaskTypes = map[string]string{
	"review":   "review",
	"bugfix":   "generate",
	"feature":  "generate",
	"eval":     "evaluate",
	"refactor": "refactor",
	"migrate":  "refactor",
}

type webServer struct {
	factory    *factory.FactoryOrchestrator
	factoryDef *factorydef.FactoryDefinition
	harness    *harness.CompleteHarness
	specs      *spec.SpecEngine
}

func newWebServer(workingDir string) (*webServer, error) {
	fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
	if err != nil {
		return nil, fmt.Errorf("creating factory: %w", err)
	}
	if err := fo.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("starting factory: %w", err)
	}

	var factoryDef *factorydef.FactoryDefinition
	if path, err := factorydef.FindFactoryDefinition(workingDir); err == nil {
		factoryDef, err = factorydef.LoadFactoryDefinition(path)
		if err != nil {
			log.Printf("Warning: failed to load factory definition: %v", err)
		}
	}

	return &webServer{
		factory:    fo,
		factoryDef: factoryDef,
		harness:    harness.NewCompleteHarness(workingDir),
		specs:      spec.NewSpecEngine(filepath.Join(workingDir, "specs")),
	}, nil
}

func (s *webServer) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	mux.Handle("/templates/", http.StripPrefix("/templates/", http.FileServer(http.Dir("./web/templates"))))

	// Public endpoints (not guarded)
	mux.HandleFunc("/api/github/token", s.handleGitHubToken)
	mux.HandleFunc("/api/mcp/info", handlers.HandleMCPInfo)
	mux.HandleFunc("/api/mcp/rpc", handlers.HandleMCPServer)
	mux.HandleFunc("/health", handlers.HandleHealth)
	mux.HandleFunc("/", s.handleRoot)

	// Guarded API endpoints
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/factory/status", s.handleFactoryStatus)
	apiMux.HandleFunc("/api/factory/launch/", s.handleFactoryLaunch)
	apiMux.HandleFunc("/api/harness/status", s.handleHarnessStatus)
	apiMux.HandleFunc("/api/specs", s.handleListSpecs)
	apiMux.HandleFunc("/api/agents", s.handleListAgents)
	apiMux.HandleFunc("/api/agents/run", handlers.HandleRunAgent)
	apiMux.HandleFunc("/api/agents/status", handlers.HandleAgentStatus)
	apiMux.HandleFunc("/api/agents/metrics", handlers.HandleMetrics)
	apiMux.HandleFunc("/api/github/review", handlers.HandleGitHubReview)
	apiMux.HandleFunc("/api/github/review/status", handlers.HandleReviewStatus)
	apiMux.HandleFunc("/api/github/triage", handlers.HandleGitHubTriage)
	apiMux.HandleFunc("/api/github/triage/status", handlers.HandleTriageStatus)
	apiMux.HandleFunc("/api/mcp/task", handlers.HandleMCPTask)
	apiMux.HandleFunc("/api/mcp/tools", handlers.HandleMCPTools)
	apiMux.HandleFunc("/api/agents/llm", handlers.HandleLLMAgent)
	apiMux.HandleFunc("/api/dashboard", handlers.HandleDashboard)
	apiMux.HandleFunc("/api/system/metrics", handlers.HandleSystemMetrics)
	apiMux.HandleFunc("/api/state-machines", s.handleStateMachines)
	apiMux.HandleFunc("/api/tasks", s.handleTask)
	apiMux.HandleFunc("/api/tasks/", s.handleTask)

	// Apply guard to API routes
	mux.Handle("/api/", webguard.Guard(apiMux, "/api/"))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *webServer) handleFactoryStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.factory.GetFactoryStatus())
}

// handleFactoryLaunch handles POST /api/factory/launch/{action}: it submits
// a task of the mapped type and returns its ID so the client can follow the
// task's state machine via /api/tasks/{id}.
func (s *webServer) handleFactoryLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	action := strings.TrimPrefix(r.URL.Path, "/api/factory/launch/")
	taskType, ok := launchTaskTypes[action]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown launch action: " + action})
		return
	}

	var req struct {
		SpecID     string                 `json:"spec_id"`
		Priority   int                    `json:"priority"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.SpecID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "spec_id is required"})
		return
	}
	if req.Priority == 0 {
		req.Priority = 1
	}

	task := &factory.Task{
		Type:       taskType,
		SpecID:     req.SpecID,
		Priority:   req.Priority,
		Parameters: req.Parameters,
	}
	if err := s.factory.SubmitTask(task); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"task_id": task.ID,
		"status":  task.Status,
		"action":  action,
		"type":    taskType,
	})
}

// handleTask serves GET /api/tasks/{id} (snapshot) and
// GET /api/tasks/{id}/stream (Server-Sent Events until a terminal state).
func (s *webServer) handleTask(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/tasks"), "/")
	taskID := strings.TrimSuffix(rest, "/stream")
	streaming := strings.HasSuffix(rest, "/stream")

	// GET /api/tasks with no id lists recent tasks, newest first.
	if taskID == "" {
		tasks := s.factory.RecentTasks(20)
		views := make([]map[string]interface{}, 0, len(tasks))
		for _, t := range tasks {
			views = append(views, taskView(t))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": views})
		return
	}

	if !streaming {
		task, ok := s.factory.GetTaskResult(taskID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusOK, taskView(task))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastStatus state.State
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			task, ok := s.factory.GetTaskResult(taskID)
			if !ok {
				continue
			}
			if task.Status != lastStatus {
				lastStatus = task.Status
				payload, _ := json.Marshal(taskView(task))
				fmt.Fprintf(w, "data: %s\n\n", payload)
				flusher.Flush()
			}
			if factory.TaskMachine.IsTerminal(task.Status) {
				return
			}
		}
	}
}

// taskView is the JSON shape served for a task.
func taskView(task *factory.Task) map[string]interface{} {
	return map[string]interface{}{
		"id":           task.ID,
		"type":         task.Type,
		"spec_id":      task.SpecID,
		"status":       task.Status,
		"terminal":     factory.TaskMachine.IsTerminal(task.Status),
		"retries":      task.Retries,
		"result":       task.Result,
		"error":        task.Error,
		"created_at":   task.CreatedAt,
		"completed_at": task.CompletedAt,
	}
}

// handleStateMachines exposes every state machine declaration so clients and
// agents can introspect lifecycles instead of hardcoding them.
func (s *webServer) handleStateMachines(w http.ResponseWriter, r *http.Request) {
	machines := append(factory.Machines(), verify.PhaseMachine)
	exports := make([]state.Export, 0, len(machines))
	for _, m := range machines {
		exports = append(exports, m.Export())
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"machines": exports})
}

func (s *webServer) handleHarnessStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ready"})
}

// handleListSpecs lists the specifications that can actually be launched —
// the spec files on disk, not the factory-definition workflows.
func (s *webServer) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	all, err := s.specs.LoadAllSpecs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ids := make([]string, 0, len(all))
	for id := range all {
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	specs := make([]map[string]interface{}, 0, len(ids))
	for _, id := range ids {
		specs = append(specs, map[string]interface{}{
			"id":      id,
			"title":   all[id].Title,
			"version": all[id].Version,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"specs": specs})
}

func (s *webServer) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]map[string]interface{}, 0)
	if s.factoryDef != nil {
		for _, a := range s.factoryDef.Factory.Agents {
			agents = append(agents, map[string]interface{}{
				"id":           a.ID,
				"role":         a.Role,
				"capabilities": a.Capabilities,
				"model":        a.Model,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
}

func (s *webServer) handleGitHubToken(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("GITHUB_TOKEN")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"loaded": token != "",
	})
}

func (s *webServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./web/index.html")
}

func main() {
	addr := "localhost:8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	workingDir, _ := os.Getwd()
	server, err := newWebServer(workingDir)
	if err != nil {
		log.Fatalf("Error creating web server: %v", err)
	}

	fmt.Printf("ASF Web Server starting on %s\n", addr)
	fmt.Println("7-Layer Agent Endpoints:")
	fmt.Println("  POST /api/agents/run              - Execute 7-layer agent (SSE)")
	fmt.Println("  GET  /api/agents/status           - Get result")
	fmt.Println("  GET  /api/agents/metrics          - Get metrics")
	fmt.Println("GitHub Integration Endpoints:")
	fmt.Println("  GET  /api/github/review           - Review PR (SSE streaming)")
	fmt.Println("       ?owner=X&repo=Y&pr=N&token=Z")
	fmt.Println("  GET  /api/github/review/status    - Get review result")
	fmt.Println("GitHub Automatic Triage Endpoints:")
	fmt.Println("  GET  /api/github/triage           - Triage issue (SSE streaming)")
	fmt.Println("       ?owner=X&repo=Y&issue=N&token=Z")
	fmt.Println("  GET  /api/github/triage/status    - Get triage result")
	fmt.Println("Factory Endpoints:")
	fmt.Println("  GET  /api/factory/status          - Factory status")
	fmt.Println("  POST /api/factory/launch          - Launch task")

	if err := http.ListenAndServe(addr, server.routes()); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
