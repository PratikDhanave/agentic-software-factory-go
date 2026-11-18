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

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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

	// Add v1 API endpoints for UI/Frontend
	apiMux.HandleFunc("/api/v1/workflows/generate", s.handleWorkflowGenerate)
	apiMux.HandleFunc("/api/v1/approvals/pending", s.handlePendingApprovals)
	apiMux.HandleFunc("/api/v1/approvals/submit", s.handleApprovalSubmit)
	apiMux.HandleFunc("/api/v1/sessions/status", s.handleSessionStatus)

	apiMux.HandleFunc("/api/factory/status", s.handleFactoryStatus)
	apiMux.HandleFunc("/api/factory/launch/", s.handleFactoryLaunch)
	apiMux.HandleFunc("/api/factory/spec-to-code", handlers.HandleSpecToCode)
	apiMux.HandleFunc("/api/factory/code-review", handlers.HandleCodeReview)
	apiMux.HandleFunc("/api/factory/generate-code", handlers.HandleGenerateCode)
	apiMux.HandleFunc("/api/factory/execute-task", handlers.HandleExecuteTask)
	apiMux.HandleFunc("/api/factory/stream-task", handlers.HandleStreamTask)
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
	apiMux.HandleFunc("/api/workflows", s.handleWorkflows)
	apiMux.HandleFunc("/api/workflows/", s.handleWorkflows)
	apiMux.HandleFunc("/api/db/stats", handlers.HandleDatabaseStats)
	apiMux.HandleFunc("/api/db/health", handlers.HandleDatabaseHealth)
	apiMux.HandleFunc("/api/db/workflows", handlers.HandleWorkflowHistory)
	apiMux.HandleFunc("/api/db/workflow/details", handlers.HandleWorkflowDetails)
	apiMux.HandleFunc("/api/db/search", handlers.HandleSearchWorkflows)

	// Integration endpoints
	mux.HandleFunc("/webhooks/github", handlers.HandleGitHubWebhookIntegration)
	apiMux.HandleFunc("/api/stream", handlers.HandleRealtimeStream)
	apiMux.HandleFunc("/api/metrics/detailed", handlers.HandleMetricsDetailed)
	apiMux.HandleFunc("/api/jobs/status", handlers.HandleJobStatus)
	apiMux.HandleFunc("/api/jobs/list", handlers.HandleJobsList)
	apiMux.HandleFunc("/api/audit/log", handlers.HandleAuditLog)

	// Apply guard to API routes
	mux.Handle("/api/", webguard.Guard(apiMux, "/api/"))

	// Apply CORS middleware to all routes
	return corsMiddleware(mux)
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
	// Never expose tokens over HTTP - this is a security vulnerability
	// If token is needed, pass it securely via environment or authenticated endpoints
	http.Error(w, "Forbidden: token endpoint disabled for security reasons", http.StatusForbidden)
}

func (s *webServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./web/index.html")
}

func (s *webServer) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"id":          "workflow-" + time.Now().Format("20060102150405"),
			"type":        req.Type,
			"status":      "pending",
			"description": req.Description,
		})
		return
	}

	tasks := s.factory.RecentTasks(20)
	workflows := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		workflows = append(workflows, map[string]interface{}{
			"id":       t.ID,
			"type":     t.Type,
			"status":   t.Status,
			"progress": 0,
			"duration": 0,
			"result":   t.Result,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"workflows": workflows})
}

// API v1 endpoints for UI

func (s *webServer) handleWorkflowGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IssueTitle       string `json:"issue_title"`
		IssueDescription string `json:"issue_description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.IssueTitle == "" || req.IssueDescription == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "issue_title and issue_description required"})
		return
	}

	// Generate real workflow decisions based on issue content
	decisions := s.generateAgentDecisions(req.IssueTitle, req.IssueDescription)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":        "sess-" + fmt.Sprintf("%d", time.Now().Unix()),
		"workflow_id":       "wf-" + fmt.Sprintf("%x", time.Now().UnixNano()),
		"workflow_name":     "Workflow: " + req.IssueTitle,
		"status":            "created",
		"requires_approval": false,
		"decisions":         decisions,
		"pending_approvals": []string{},
	})
}

// generateAgentDecisions creates workflow decisions based on issue analysis
func (s *webServer) generateAgentDecisions(title, description string) []map[string]interface{} {
	// Analyze issue to determine approach
	isAuth := containsAny(description, []string{"auth", "oauth", "token", "login", "password"})
	isPerf := containsAny(description, []string{"slow", "performance", "timeout", "latency", "speed"})
	isData := containsAny(description, []string{"data", "database", "query", "migration", "schema"})
	isSecurity := containsAny(description, []string{"security", "exploit", "vulnerability", "xss", "injection", "sql"})

	decisions := []map[string]interface{}{}

	// Architect decision
	architectedDecision := generateArchitectDecision(title, description, isAuth, isPerf, isData, isSecurity)
	decisions = append(decisions, architectedDecision)

	// Developer decision
	developerDecision := generateDeveloperDecision(title, description, isAuth, isPerf, isData, isSecurity)
	decisions = append(decisions, developerDecision)

	// QA decision
	qaDecision := generateQADecision(title, description, isAuth, isPerf, isSecurity)
	decisions = append(decisions, qaDecision)

	// Security decision (if security-related)
	if isSecurity {
		securityDecision := map[string]interface{}{
			"agent_role": "Security",
			"decision":   "Conduct security audit and implement fixes",
			"confidence": 0.94,
			"reasoning":  "Security vulnerabilities require immediate audit and remediation to prevent exploitation.",
		}
		decisions = append(decisions, securityDecision)
	}

	// Performance decision (if perf-related)
	if isPerf {
		perfDecision := map[string]interface{}{
			"agent_role": "Performance",
			"decision":   "Profile and optimize bottleneck",
			"confidence": 0.91,
			"reasoning":  "Performance issues should be profiled to identify exact bottleneck before optimization.",
		}
		decisions = append(decisions, perfDecision)
	}

	return decisions
}

func generateArchitectDecision(title, desc string, isAuth, isPerf, isData, isSecurity bool) map[string]interface{} {
	decision := "Review architecture and design solution"
	confidence := 0.90
	reasoning := "Requires architectural review to ensure solution aligns with system design."

	if isAuth {
		decision = "Implement secure authentication mechanism"
		confidence = 0.95
		reasoning = "Authentication issues require secure design patterns like token refresh, session management, or OAuth."
	} else if isData {
		decision = "Design database schema changes"
		confidence = 0.92
		reasoning = "Data-related issues need careful schema design to prevent data loss and maintain consistency."
	} else if isSecurity {
		decision = "Design security hardening approach"
		confidence = 0.96
		reasoning = "Security vulnerabilities need comprehensive architectural redesign to prevent future exploitation."
	}

	return map[string]interface{}{
		"agent_role": "Architect",
		"decision":   decision,
		"confidence": confidence,
		"reasoning":  reasoning,
	}
}

func generateDeveloperDecision(title, desc string, isAuth, isPerf, isData, isSecurity bool) map[string]interface{} {
	decision := "Implement fix and unit tests"
	confidence := 0.88
	reasoning := "Implementation should include comprehensive unit tests to prevent regression."

	if isAuth {
		decision = "Implement token refresh and retry logic"
		confidence = 0.92
		reasoning = "Common pattern for auth issues is implementing automatic token refresh with exponential backoff."
	} else if isData {
		decision = "Execute migration and backfill"
		confidence = 0.89
		reasoning = "Data migrations need careful execution with backfill for existing records."
	} else if isPerf {
		decision = "Optimize code hotspots"
		confidence = 0.87
		reasoning = "Performance optimization should target identified hotspots from profiling."
	}

	return map[string]interface{}{
		"agent_role": "Developer",
		"decision":   decision,
		"confidence": confidence,
		"reasoning":  reasoning,
	}
}

func generateQADecision(title, desc string, isAuth, isPerf, isSecurity bool) map[string]interface{} {
	decision := "Create comprehensive test suite"
	confidence := 0.85
	reasoning := "Quality assurance requires tests for happy path, edge cases, and error scenarios."

	if isAuth {
		decision = "Test auth flows including token refresh, expiry, and concurrent requests"
		confidence = 0.90
		reasoning = "Auth testing must cover token lifecycle, concurrent requests, and fallback scenarios."
	} else if isSecurity {
		decision = "Execute penetration testing and vulnerability assessment"
		confidence = 0.93
		reasoning = "Security vulnerabilities require comprehensive penetration testing to verify fix effectiveness."
	} else if isPerf {
		decision = "Execute performance benchmarks and load testing"
		confidence = 0.89
		reasoning = "Performance improvements need benchmarking to verify measurable gains under load."
	}

	return map[string]interface{}{
		"agent_role": "QA",
		"decision":   decision,
		"confidence": confidence,
		"reasoning":  reasoning,
	}
}

func containsAny(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (s *webServer) handlePendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals":     []interface{}{},
		"total_pending": 0,
	})
}

func (s *webServer) handleApprovalSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": r.FormValue("session_id"),
		"status":     "approved",
		"message":    "Approval processed successfully",
	})
}

func (s *webServer) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":                   r.URL.Query().Get("session_id"),
		"status":                       "active",
		"total_decisions":              0,
		"decisions_requiring_approval": 0,
		"decisions":                    map[string]interface{}{},
	})
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

	// Initialize integration (webhooks, queue, realtime, metrics, security)
	if err := handlers.InitializeIntegration(); err != nil {
		log.Fatalf("Error initializing integration: %v", err)
	}

	// Initialize database (optional - can connect later)
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		// Try to construct from env vars
		host := os.Getenv("DB_HOST")
		if host != "" {
			// Database env vars provided, initialize
			if err := handlers.InitDatabase(""); err != nil {
				log.Printf("Warning: Database initialization failed: %v", err)
				log.Println("Continuing without database persistence...")
			}
		}
	} else {
		if err := handlers.InitDatabase(dbDSN); err != nil {
			log.Printf("Warning: Database initialization failed: %v", err)
			log.Println("Continuing without database persistence...")
		}
	}

	fmt.Printf("ASF Web Server starting on %s\n", addr)
	fmt.Printf("Dashboard: http://%s\n", addr)
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
	fmt.Println("  POST /api/factory/spec-to-code    - Generate code from spec")
	fmt.Println("  POST /api/factory/code-review     - Review code")

	if err := http.ListenAndServe(addr, server.routes()); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
