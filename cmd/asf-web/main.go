// Command asf-web serves the unified Agentic Software Factory HTTP & SSE API
// consumed by both the Next.js Control Plane (web-ui/) and the embedded single-binary UI.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/PratikDhanave/agentic-software-factory-go/cmd/asf-web/handlers"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/usecases"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/bootstrap"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/stateregistry"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/mcp"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webguard"
)

var launchTaskTypes = map[string]string{
	"review":   "review",
	"bugfix":   "generate",
	"feature":  "generate",
	"generate": "generate",
	"verify":   "verify",
	"eval":     "evaluate",
	"evaluate": "evaluate",
	"refactor": "refactor",
	"migrate":  "refactor",
}

type webServer struct {
	container *bootstrap.Container
	mcpServer *mcp.MCPServer
}

func newWebServer(workingDir string) (*webServer, error) {
	c, err := bootstrap.NewContainer(workingDir)
	if err != nil {
		return nil, err
	}
	handlers.SetContainer(c)
	return &webServer{
		container: c,
		mcpServer: mcp.NewMCPServer(c.Factory, c.FactoryDef).WithStateMachines(stateregistry.All()),
	}, nil
}

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

	// Public health & webhook endpoints
	mux.HandleFunc("/api/github/token", s.handleGitHubToken)
	mux.HandleFunc("/api/mcp/info", handlers.HandleMCPInfo)
	mux.HandleFunc("/api/mcp/rpc", handlers.HandleMCPServer)
	mux.HandleFunc("/health", handlers.HandleHealth)
	mux.HandleFunc("/webhooks/github", handlers.HandleGitHubWebhookIntegration)
	mux.HandleFunc("/", s.handleRoot)

	// Guarded Control Plane API
	apiMux := http.NewServeMux()

	// System Overview & Phase Control
	apiMux.HandleFunc("/api/system/overview", s.handleSystemOverview)
	apiMux.HandleFunc("/api/system/phase", s.handleSystemPhase)
	apiMux.HandleFunc("/api/system/metrics", handlers.HandleSystemMetrics)
	apiMux.HandleFunc("/api/dashboard", handlers.HandleDashboard)
	apiMux.HandleFunc("/api/state-machines", s.handleStateMachines)
	apiMux.HandleFunc("/api/state-machines/validate", s.handleStateMachineValidate)

	// Autonomous Software Factory Assembly Line (6-Stage End-to-End Pipeline)
	apiMux.HandleFunc("/api/v1/factory/runs", s.handleFactoryRuns)
	apiMux.HandleFunc("/api/v1/factory/runs/", s.handleFactoryRunByID)
	apiMux.HandleFunc("/api/v1/gateway/config", s.handleGatewayConfig)
	apiMux.HandleFunc("/api/v1/agents/consult", s.handleAgentConsult)

	// Multi-Agent Coordination, DAG Execution & Approvals Gate
	apiMux.HandleFunc("/api/v1/workflows/generate", s.handleWorkflowGenerate)
	apiMux.HandleFunc("/api/v1/workflows/coordinate", s.handleWorkflowGenerate)
	apiMux.HandleFunc("/api/v1/workflows/execute", s.handleWorkflowExecute)
	apiMux.HandleFunc("/api/v1/workflows/executions", s.handleListWorkflowExecutions)
	apiMux.HandleFunc("/api/v1/sessions", s.handleListSessions)
	apiMux.HandleFunc("/api/v1/sessions/status", s.handleSessionStatus)
	apiMux.HandleFunc("/api/v1/approvals/pending", s.handlePendingApprovals)
	apiMux.HandleFunc("/api/v1/approvals/submit", s.handleApprovalSubmit)

	// Self-Improvement & Learning Simulator
	apiMux.HandleFunc("/api/v1/metrics/self-improvement", s.handleSelfImprovementMetrics)
	apiMux.HandleFunc("/api/metrics/self-improvement", s.handleSelfImprovementMetrics)
	apiMux.HandleFunc("/api/v1/learning/simulate", s.handleLearningSimulate)
	apiMux.HandleFunc("/api/v1/learning/learn", s.handleLearningTrigger)

	// Factory Worker Pool, Isolated Workspace Tasks & Specs
	apiMux.HandleFunc("/api/factory/status", s.handleFactoryStatus)
	apiMux.HandleFunc("/api/factory/launch/", s.handleFactoryLaunch)
	apiMux.HandleFunc("/api/factory/verify", s.handleFactoryVerify)
	apiMux.HandleFunc("/api/factory/spec-to-code", handlers.HandleSpecToCode)
	apiMux.HandleFunc("/api/factory/code-review", handlers.HandleCodeReview)
	apiMux.HandleFunc("/api/factory/generate-code", handlers.HandleGenerateCode)
	apiMux.HandleFunc("/api/factory/execute-task", handlers.HandleExecuteTask)
	apiMux.HandleFunc("/api/factory/stream-task", handlers.HandleStreamTask)
	apiMux.HandleFunc("/api/harness/status", s.handleHarnessStatus)
	apiMux.HandleFunc("/api/specs", s.handleSpecs)
	apiMux.HandleFunc("/api/specs/", s.handleSpecByID)
	apiMux.HandleFunc("/api/agents", s.handleListAgents)
	apiMux.HandleFunc("/api/tasks", s.handleTask)
	apiMux.HandleFunc("/api/tasks/", s.handleTask)
	apiMux.HandleFunc("/api/workflows", s.handleWorkflows)
	apiMux.HandleFunc("/api/workflows/", s.handleWorkflows)

	// Legacy & Auxiliary Endpoints
	apiMux.HandleFunc("/api/agents/run", handlers.HandleRunAgent)
	apiMux.HandleFunc("/api/agents/status", handlers.HandleAgentStatus)
	apiMux.HandleFunc("/api/agents/metrics", handlers.HandleMetrics)
	apiMux.HandleFunc("/api/agents/llm", handlers.HandleLLMAgent)
	apiMux.HandleFunc("/api/github/review", handlers.HandleGitHubReview)
	apiMux.HandleFunc("/api/github/review/status", handlers.HandleReviewStatus)
	apiMux.HandleFunc("/api/github/triage", handlers.HandleGitHubTriage)
	apiMux.HandleFunc("/api/github/triage/status", handlers.HandleTriageStatus)
	apiMux.HandleFunc("/api/mcp/task", handlers.HandleMCPTask)
	apiMux.HandleFunc("/api/mcp/tools", handlers.HandleMCPTools)
	apiMux.Handle("/api/mcp/factory", s.mcpServer)
	apiMux.HandleFunc("/api/db/stats", handlers.HandleDatabaseStats)
	apiMux.HandleFunc("/api/db/health", handlers.HandleDatabaseHealth)
	apiMux.HandleFunc("/api/db/workflows", handlers.HandleWorkflowHistory)
	apiMux.HandleFunc("/api/db/workflow/details", handlers.HandleWorkflowDetails)
	apiMux.HandleFunc("/api/db/search", handlers.HandleSearchWorkflows)
	apiMux.HandleFunc("/api/stream", handlers.HandleRealtimeStream)
	apiMux.HandleFunc("/api/metrics/detailed", handlers.HandleMetricsDetailed)
	apiMux.HandleFunc("/api/jobs/status", handlers.HandleJobStatus)
	apiMux.HandleFunc("/api/jobs/list", handlers.HandleJobsList)
	apiMux.HandleFunc("/api/audit/log", handlers.HandleAuditLog)

	// Code Conversion Endpoints
	apiMux.HandleFunc("/api/conversions", handlers.HandleConversions)
	apiMux.HandleFunc("/api/conversions/", handlers.HandleConversionByID)

	mux.Handle("/api/", webguard.Guard(apiMux, "/api/"))
	return corsMiddleware(mux)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// handleSystemOverview serves a unified real-time telemetry snapshot for the Control Plane UI.
func (s *webServer) handleSystemOverview(w http.ResponseWriter, r *http.Request) {
	c := s.container
	factoryStatus := c.Factory.GetFactoryStatus()
	dbStats, _ := c.Store.Stats(r.Context())
	specsMap, _ := c.SpecEngine.LoadAllSpecs()
	pendingSessions, _ := c.SessionRepo.ListPendingApprovals()

	pendingCount := 0
	for _, sess := range pendingSessions {
		for _, d := range sess.Decisions {
			if d != nil && d.NeedsApproval {
				pendingCount++
			}
		}
	}

	agents := make([]map[string]interface{}, 0)
	for _, cap := range c.AgentRegistry.GetAll() {
		agents = append(agents, map[string]interface{}{
			"role":               cap.Role,
			"title":              cap.Title,
			"description":        cap.Description,
			"capabilities":       cap.Capabilities,
			"decision_authority": cap.DecisionAuthority,
			"interaction_mode":   cap.InteractionMode,
			"domains":            cap.KnowledgeDomains,
		})
	}
	sort.Slice(agents, func(i, j int) bool {
		return fmt.Sprint(agents[i]["role"]) < fmt.Sprint(agents[j]["role"])
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            "healthy",
		"uptime":            time.Since(c.StartedAt).Round(time.Second).String(),
		"phase":             c.Phase(),
		"llm_provider":      c.LLMGateway.ProviderName(),
		"gateway_mode":      c.LLMGateway.PreferredMode(),
		"gateway_providers": c.LLMGateway.ProviderStatuses(),
		"persistence":       "sqlite3-wal",
		"db_path":           c.Store.Path(),
		"specs_count":       len(specsMap),
		"agents_count":      len(agents),
		"agents":            agents,
		"pending_approvals": pendingCount,
		"factory":           factoryStatus,
		"database":          dbStats,
	})
}

func (s *webServer) handleGatewayConfig(w http.ResponseWriter, r *http.Request) {
	gw := s.container.LLMGateway
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"preferred_provider": gw.PreferredMode(),
			"active_provider":    gw.ProviderName(),
			"providers":          gw.ProviderStatuses(),
		})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		PreferredProvider string `json:"preferred_provider"`
		GeminiAPIKey      string `json:"gemini_api_key"`
		GeminiModel       string `json:"gemini_model"`
		AnthropicAPIKey   string `json:"anthropic_api_key"`
		AnthropicModel    string `json:"anthropic_model"`
		OpenAIAPIKey      string `json:"openai_api_key"`
		OpenAIModel       string `json:"openai_model"`
		OllamaURL         string `json:"ollama_url"`
		OllamaModel       string `json:"ollama_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	next := gw.ConfigSnapshot()
	if strings.TrimSpace(req.PreferredProvider) != "" {
		next.PreferredProvider = strings.ToLower(strings.TrimSpace(req.PreferredProvider))
	}
	if strings.TrimSpace(req.GeminiAPIKey) != "" {
		next.GeminiAPIKey = strings.TrimSpace(req.GeminiAPIKey)
	}
	if strings.TrimSpace(req.GeminiModel) != "" {
		next.GeminiModel = strings.TrimSpace(req.GeminiModel)
	}
	if strings.TrimSpace(req.AnthropicAPIKey) != "" {
		next.AnthropicAPIKey = strings.TrimSpace(req.AnthropicAPIKey)
	}
	if strings.TrimSpace(req.AnthropicModel) != "" {
		next.AnthropicModel = strings.TrimSpace(req.AnthropicModel)
	}
	if strings.TrimSpace(req.OpenAIAPIKey) != "" {
		next.OpenAIAPIKey = strings.TrimSpace(req.OpenAIAPIKey)
	}
	if strings.TrimSpace(req.OpenAIModel) != "" {
		next.OpenAIModel = strings.TrimSpace(req.OpenAIModel)
	}
	if strings.TrimSpace(req.OllamaURL) != "" {
		next.OllamaURL = strings.TrimSpace(req.OllamaURL)
	}
	if strings.TrimSpace(req.OllamaModel) != "" {
		next.OllamaModel = strings.TrimSpace(req.OllamaModel)
	}

	gw.Reconfigure(next)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"preferred_provider": gw.PreferredMode(),
		"active_provider":    gw.ProviderName(),
		"providers":          gw.ProviderStatuses(),
		"message":            "Model Gateway updated — active provider is now " + gw.ProviderName(),
	})
}

func (s *webServer) handleAgentConsult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Role   string `json:"role"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}

	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "architect"
	}

	sysPrompt := fmt.Sprintf("You are the Factory %s Agent inside an Agentic Go Software Factory. Provide concise, actionable engineering guidance.", role)
	for _, cap := range s.container.AgentRegistry.GetAll() {
		if strings.EqualFold(string(cap.Role), role) {
			sysPrompt = fmt.Sprintf(
				"You are %s (%s). Authority: %s. Capabilities: %s. Respond with crisp, concrete Go engineering recommendations.",
				cap.Title,
				cap.Description,
				cap.DecisionAuthority,
				strings.Join(cap.Capabilities, ", "),
			)
			break
		}
	}

	callStart := time.Now()
	resp, err := s.container.LLMGateway.Complete(r.Context(), &services.LLMRequest{
		SystemPrompt: sysPrompt,
		Prompt:       req.Prompt,
		Temperature:  0.2,
		MaxTokens:    1200,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"role":        role,
		"provider":    resp.Provider,
		"model":       resp.Model,
		"content":     resp.Content,
		"tokens":      resp.Tokens,
		"duration_ms": time.Since(callStart).Milliseconds(),
	})
}

func (s *webServer) handleSystemPhase(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]string{"phase": s.container.Phase()})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Phase string `json:"phase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := s.container.SetPhase(req.Phase); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"phase":   s.container.Phase(),
		"message": "Factory maturity phase updated to " + s.container.Phase(),
	})
}

// handleWorkflowGenerate runs multi-agent coordination, persists session/workflow/decisions
// to SQLite, evaluates phase-aware HITL policy, and optionally executes the DAG.
func (s *webServer) handleWorkflowGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IssueTitle       string                 `json:"issue_title"`
		IssueDescription string                 `json:"issue_description"`
		TriggerType      string                 `json:"trigger_type"`
		Classification   string                 `json:"classification"`
		TriggerEvent     map[string]interface{} `json:"trigger_event"`
		ApprovalRequired *bool                  `json:"approval_required"`
		AutoExecute      *bool                  `json:"auto_execute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}

	if req.TriggerEvent == nil {
		req.TriggerEvent = map[string]interface{}{}
	}
	if req.IssueTitle != "" {
		req.TriggerEvent["title"] = req.IssueTitle
	}
	if req.IssueDescription != "" {
		req.TriggerEvent["description"] = req.IssueDescription
	}
	title, _ := req.TriggerEvent["title"].(string)
	desc, _ := req.TriggerEvent["description"].(string)
	if strings.TrimSpace(title) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "issue_title is required"})
		return
	}
	if req.TriggerType == "" {
		req.TriggerType = "issue"
	}
	if req.Classification == "" {
		req.Classification = classifyIssueType(title + " " + desc)
	}

	c := s.container
	phase := c.Phase()
	forceApproval := phase == "crawl" || (req.ApprovalRequired != nil && *req.ApprovalRequired)

	out, err := c.CoordinateUC.Execute(r.Context(), &usecases.CoordinateWorkflowInput{
		TriggerEvent:     req.TriggerEvent,
		TriggerType:      req.TriggerType,
		Classification:   req.Classification,
		ApprovalRequired: forceApproval,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	execUUID := uuid.New()
	pendingIDs := make([]string, 0)
	decisionsView := make([]map[string]interface{}, 0, len(out.Decisions))
	anyNeedsApproval := false

	roles := make([]string, 0, len(out.Decisions))
	for role := range out.Decisions {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	for _, role := range roles {
		d := out.Decisions[role]
		if d == nil {
			continue
		}
		skip := !forceApproval && c.SelfImproving.ShouldSkipApproval(r.Context(), role, req.Classification, d.Confidence, phase)
		d.NeedsApproval = !skip
		if d.NeedsApproval {
			anyNeedsApproval = true
		}

		decID := uuid.New()
		approvalStatus := "approved"
		if d.NeedsApproval {
			approvalStatus = "pending"
			pendingIDs = append(pendingIDs, out.Session.ID+":"+role)
		}

		agentDec := &domain.AgentDecision{
			ID:               decID,
			ExecutionID:      execUUID,
			AgentRole:        role,
			IssueType:        req.Classification,
			Decision:         d.Choice,
			Reasoning:        d.Reasoning,
			Confidence:       d.Confidence,
			Alternatives:     d.Alternatives,
			RequiresApproval: d.NeedsApproval,
			ApprovalStatus:   approvalStatus,
			CreatedAt:        time.Now(),
		}
		_ = c.LearningRepo.SaveDecision(r.Context(), agentDec)
		_ = c.SelfImproving.RecordDecision(r.Context(), agentDec, req.Classification)

		decisionsView = append(decisionsView, map[string]interface{}{
			"id":                out.Session.ID + ":" + role,
			"decision_uuid":     decID.String(),
			"agent_role":        role,
			"decision":          d.Choice,
			"choice":            d.Choice,
			"reasoning":         d.Reasoning,
			"confidence":        d.Confidence,
			"alternatives":      d.Alternatives,
			"needs_approval":    d.NeedsApproval,
			"approval_status":   approvalStatus,
		})
	}

	if anyNeedsApproval {
		out.Session.RequestApproval()
	} else {
		out.Session.Status = entities.SessionExecuting
	}
	_ = c.SessionRepo.SaveSession(out.Session)

	yamlBytes, _ := yaml.Marshal(out.Workflow)

	var executionView interface{}
	autoExec := req.AutoExecute == nil || *req.AutoExecute
	if !anyNeedsApproval && autoExec && out.Workflow != nil {
		if execRes, execErr := c.ExecuteWorkflowUC.Execute(r.Context(), out.Workflow.ID, req.TriggerEvent); execErr == nil {
			out.Session.Complete()
			_ = c.SessionRepo.SaveSession(out.Session)
			executionView = execRes
		}
	}

	// Build DAG preview levels for UI visualization
	var dagLevels [][]string
	if dag, dagErr := c.DAGBuilder.Build(out.Workflow); dagErr == nil {
		dagLevels = dag.GetExecutionLevels()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":        out.Session.ID,
		"workflow_id":       out.Workflow.ID,
		"workflow_name":     out.Workflow.Name,
		"status":            out.Session.Status,
		"phase":             phase,
		"classification":    req.Classification,
		"requires_approval": anyNeedsApproval,
		"decisions":         decisionsView,
		"pending_approvals": pendingIDs,
		"dag_levels":        dagLevels,
		"jobs":              out.Workflow.JobsMap,
		"workflow_yaml":     string(yamlBytes),
		"execution":         executionView,
	})
}

func (s *webServer) handleWorkflowExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		WorkflowID string                 `json:"workflow_id"`
		SessionID  string                 `json:"session_id"`
		Input      map[string]interface{} `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	c := s.container
	workflowID := req.WorkflowID
	trigger := req.Input
	if req.SessionID != "" {
		if sess, err := c.SessionRepo.GetSession(req.SessionID); err == nil {
			if workflowID == "" {
				workflowID = sess.WorkflowDefID
			}
			if trigger == nil {
				trigger = sess.TriggerEvent
			}
		}
	}
	if workflowID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workflow_id or session_id is required"})
		return
	}

	execRes, err := c.ExecuteWorkflowUC.Execute(r.Context(), workflowID, trigger)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if req.SessionID != "" {
		_ = c.SessionRepo.UpdateSessionStatus(req.SessionID, entities.SessionCompleted)
	}
	writeJSON(w, http.StatusOK, execRes)
}

func (s *webServer) handleListWorkflowExecutions(w http.ResponseWriter, r *http.Request) {
	execs, err := s.container.WorkflowRepo.ListExecutions(r.Context(), "", 25, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"executions": execs})
}

func (s *webServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.container.SessionRepo.ListRecentSessions(25)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

func (s *webServer) handlePendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	sessions, err := s.container.SessionRepo.ListPendingApprovals()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	approvals := make([]map[string]interface{}, 0)
	for _, sess := range sessions {
		title, _ := sess.TriggerEvent["title"].(string)
		roles := make([]string, 0, len(sess.Decisions))
		for role := range sess.Decisions {
			roles = append(roles, role)
		}
		sort.Strings(roles)

		for _, role := range roles {
			dec := sess.Decisions[role]
			if dec == nil || !dec.NeedsApproval {
				continue
			}
			approvals = append(approvals, map[string]interface{}{
				"id":           sess.ID + ":" + role,
				"session_id":   sess.ID,
				"workflow_id":  sess.WorkflowDefID,
				"issue_title":  title,
				"agent_role":   role,
				"decision":     dec.Choice,
				"reasoning":    dec.Reasoning,
				"confidence":   dec.Confidence,
				"alternatives": dec.Alternatives,
				"status":       "pending",
				"created_at":   sess.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals":         approvals,
		"pending_approvals": approvals,
		"total_pending":     len(approvals),
	})
}

func (s *webServer) handleApprovalSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		ApprovalID string `json:"approval_id"`
		SessionID  string `json:"session_id"`
		Approver   string `json:"approver"`
		Approved   bool   `json:"approved"`
		Comments   string `json:"comments"`
		Override   string `json:"override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Approver == "" {
		req.Approver = "human-operator"
	}

	sessionID := req.SessionID
	targetRole := ""
	if strings.Contains(req.ApprovalID, ":") {
		parts := strings.SplitN(req.ApprovalID, ":", 2)
		sessionID = parts[0]
		targetRole = parts[1]
	} else if sessionID == "" {
		sessionID = req.ApprovalID
	}

	c := s.container
	sess, err := c.SessionRepo.GetSession(sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "approval session not found"})
		return
	}

	if !req.Approved {
		_ = c.SessionRepo.UpdateSessionStatus(sessionID, entities.SessionFailed)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"approval_id": req.ApprovalID,
			"session_id":  sessionID,
			"status":      "rejected",
			"message":     "Decision rejected; workflow halted.",
		})
		return
	}

	stillPending := 0
	for role, dec := range sess.Decisions {
		if dec == nil {
			continue
		}
		if targetRole == "" || role == targetRole {
			dec.NeedsApproval = false
			dec.Approver = req.Approver
			if req.Override != "" {
				dec.Choice = req.Override
				dec.Reasoning = "Human override: " + req.Override
			} else if req.Comments != "" {
				dec.Reasoning = dec.Reasoning + " (Approved: " + req.Comments + ")"
			}
		}
		if dec.NeedsApproval {
			stillPending++
		}
	}

	var execView interface{}
	status := "approved_partial"
	msg := fmt.Sprintf("Approved %s (%d remaining in session)", targetRole, stillPending)

	if stillPending == 0 {
		sess.Status = entities.SessionExecuting
		_ = c.SessionRepo.SaveSession(sess)
		// Also resume any paused Autonomous Factory Run tied to this session!
		if resumedRun, resumeErr := c.Pipeline.ResumeBySessionID(r.Context(), sessionID, req.Approver, false); resumeErr == nil && resumedRun != nil {
			execView = resumedRun
			status = "approved"
			msg = "All session gates approved — Factory Assembly Line resumed (" + resumedRun.ID + ")."
		} else {
			if sess.WorkflowDefID != "" {
				if execRes, execErr := c.ExecuteWorkflowUC.Execute(r.Context(), sess.WorkflowDefID, sess.TriggerEvent); execErr == nil {
					execView = execRes
					sess.Complete()
				}
			} else {
				sess.Complete()
			}
			status = "approved"
			msg = "All session gates approved — workflow executed."
		}
	}
	_ = c.SessionRepo.SaveSession(sess)

	// Record positive learning signal
	if recent, _ := c.LearningRepo.GetRecentDecisions(r.Context(), targetRole, 1); len(recent) > 0 {
		_ = c.SelfImproving.RecordApproval(r.Context(), &domain.Approval{
			ID:                uuid.New(),
			DecisionID:        recent[0].ID,
			AgentRole:         targetRole,
			Status:            "approved",
			ApproverID:        req.Approver,
			ApprovalComments:  req.Comments,
			ApprovalTimestamp: time.Now(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approval_id":       req.ApprovalID,
		"session_id":        sessionID,
		"status":            status,
		"remaining_pending": stillPending,
		"message":           msg,
		"execution":         execView,
	})
}

func (s *webServer) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id required"})
		return
	}
	sess, err := s.container.SessionRepo.GetSession(sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	pending := 0
	for _, d := range sess.Decisions {
		if d != nil && d.NeedsApproval {
			pending++
		}
	}
	title, _ := sess.TriggerEvent["title"].(string)
	desc, _ := sess.TriggerEvent["description"].(string)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":                   sess.ID,
		"workflow_id":                  sess.WorkflowDefID,
		"status":                       sess.Status,
		"issue_title":                  title,
		"issue_description":            desc,
		"total_decisions":              len(sess.Decisions),
		"pending_approvals":            pending,
		"decisions_requiring_approval": pending,
		"created_at":                   sess.CreatedAt.Format(time.RFC3339),
		"decisions":                    sess.Decisions,
	})
}

func (s *webServer) handleSelfImprovementMetrics(w http.ResponseWriter, r *http.Request) {
	c := s.container
	m, err := c.SelfImproving.GetMetrics(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	dbStats, _ := c.Store.Stats(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Agents":    m.Agents,
		"Patterns":  m.Patterns,
		"Timestamp": m.Timestamp,
		"Phase":     c.Phase(),
		"Database":  dbStats,
	})
}

func (s *webServer) handleLearningSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Count <= 0 {
		req.Count = 25
	}
	if req.Count > 200 {
		req.Count = 200
	}

	c := s.container
	agents := []string{"Architect", "Developer", "QA", "Security", "Performance"}
	issueTypes := []struct {
		kind      string
		reasoning string
	}{
		{"auth", "OAuth2 PKCE login session and JWT token validation"},
		{"security", "SQL injection and XSS vulnerability remediation"},
		{"performance", "Query latency and worker queue throughput optimization"},
		{"data", "SQLite WAL schema migration and transactional integrity"},
	}

	for i := 0; i < req.Count; i++ {
		ag := agents[i%len(agents)]
		it := issueTypes[(i+rand.Intn(len(issueTypes)))%len(issueTypes)]
		conf := 0.82 + float64(i%17)*0.01
		dec := &domain.AgentDecision{
			ID:               uuid.New(),
			ExecutionID:      uuid.New(),
			AgentRole:        ag,
			IssueType:        it.kind,
			Decision:         fmt.Sprintf("Apply verified %s architecture pattern", it.kind),
			Reasoning:        it.reasoning,
			Confidence:       conf,
			RequiresApproval: conf < 0.90,
			ApprovalStatus:   "approved",
			CreatedAt:        time.Now(),
		}
		_ = c.LearningRepo.SaveDecision(r.Context(), dec)
		_ = c.SelfImproving.RecordDecision(r.Context(), dec, it.kind)
		_ = c.SelfImproving.RecordApproval(r.Context(), &domain.Approval{
			ID:                uuid.New(),
			DecisionID:        dec.ID,
			AgentRole:         ag,
			Status:            "approved",
			ApproverID:        "simulator",
			ApprovalTimestamp: time.Now(),
		})
	}

	for _, ag := range agents {
		_ = c.SelfImproving.LearnPatterns(r.Context(), ag)
	}
	_ = c.SelfImproving.UpdateAgentSpecialization(r.Context())

	metrics, _ := c.SelfImproving.GetMetrics(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"generated":      req.Count,
		"patterns_found": len(metrics.Patterns),
		"metrics":        metrics,
	})
}

func (s *webServer) handleLearningTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	c := s.container
	for _, ag := range []string{"Architect", "Developer", "QA", "Security", "Performance"} {
		_ = c.SelfImproving.LearnPatterns(r.Context(), ag)
	}
	_ = c.SelfImproving.UpdateAgentSpecialization(r.Context())
	metrics, _ := c.SelfImproving.GetMetrics(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"patterns_found": len(metrics.Patterns),
	})
}

func (s *webServer) handleFactoryStatus(w http.ResponseWriter, r *http.Request) {
	status := s.container.Factory.GetFactoryStatus()
	status["llm_provider"] = s.container.LLMGateway.ProviderName()
	status["phase"] = s.container.Phase()
	writeJSON(w, http.StatusOK, status)
}

func (s *webServer) handleFactoryVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	res, err := s.container.Factory.Verifier().RunVerification(r.Context(), "")
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      res != nil && res.Success,
		"verification": res,
		"error":        errMsg,
	})
}

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
	if err := s.container.Factory.SubmitTask(task); err != nil {
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

func (s *webServer) handleTask(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/tasks"), "/")
	taskID := strings.TrimSuffix(rest, "/stream")
	streaming := strings.HasSuffix(rest, "/stream")

	if taskID == "" {
		tasks := s.container.Factory.RecentTasks(30)
		views := make([]map[string]interface{}, 0, len(tasks))
		for _, t := range tasks {
			views = append(views, taskView(t))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": views})
		return
	}

	if !streaming {
		task, ok := s.container.Factory.GetTaskResult(taskID)
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

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastStatus state.State
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			task, ok := s.container.Factory.GetTaskResult(taskID)
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

func taskView(task *factory.Task) map[string]interface{} {
	return map[string]interface{}{
		"id":           task.ID,
		"type":         task.Type,
		"spec_id":      task.SpecID,
		"assigned_to":  task.AssignedTo,
		"status":       task.Status,
		"terminal":     factory.TaskMachine.IsTerminal(task.Status),
		"retries":      task.Retries,
		"result":       task.Result,
		"error":        task.Error,
		"created_at":   task.CreatedAt,
		"started_at":   task.StartedAt,
		"completed_at": task.CompletedAt,
	}
}

func (s *webServer) handleStateMachines(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	all := stateregistry.All()
	exports := make([]state.Export, 0, len(all))
	for _, m := range all {
		if category != "" && m.Category() != category {
			continue
		}
		exports = append(exports, m.Export())
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(exports),
		"machines": exports,
	})
}

func (s *webServer) handleStateMachineValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Machine string `json:"machine"`
		From    string `json:"from"`
		To      string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	m, ok := stateregistry.ByName(req.Machine)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("state machine %q not found", req.Machine)})
		return
	}
	insp := m.Inspect(state.State(req.From))
	resp := map[string]interface{}{
		"machine":    req.Machine,
		"from":       req.From,
		"inspection": insp,
	}
	if req.To != "" {
		valErr := m.Validate(state.State(req.From), state.State(req.To))
		resp["to"] = req.To
		resp["allowed"] = valErr == nil
		if valErr != nil {
			resp["error"] = valErr.Error()
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *webServer) handleFactoryRuns(w http.ResponseWriter, r *http.Request) {
	c := s.container
	switch r.Method {
	case http.MethodGet:
		runs, err := c.FactoryRunRepo.List(r.Context(), 30)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs})
	case http.MethodPost:
		var req struct {
			Title         string   `json:"title"`
			Description   string   `json:"description"`
			SpecID        string   `json:"spec_id"`
			Constraints   []string `json:"constraints"`
			ForceApproval bool     `json:"force_approval"`
			Sync          bool     `json:"sync"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
			return
		}
		run, err := c.Pipeline.StartRun(r.Context(), application.FactoryRunInput{
			Title:         req.Title,
			Description:   req.Description,
			SpecID:        req.SpecID,
			Constraints:   req.Constraints,
			ForceApproval: req.ForceApproval,
			Phase:         c.Phase(),
		}, req.Sync)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		code := http.StatusAccepted
		if req.Sync {
			code = http.StatusOK
		}
		writeJSON(w, code, run)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *webServer) handleFactoryRunByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/factory/runs/")
	parts := strings.SplitN(rest, "/", 2)
	runID := parts[0]
	sub := ""
	if len(parts) == 2 {
		sub = parts[1]
	}

	c := s.container
	switch sub {
	case "":
		rec, err := c.FactoryRunRepo.Get(r.Context(), runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "factory run not found"})
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "approve":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var req struct {
			Approver string `json:"approver"`
			Sync     bool   `json:"sync"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		rec, err := c.Pipeline.ResumeRun(r.Context(), runID, req.Approver, req.Sync)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "promote":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var req struct {
			TargetSubdir string `json:"target_subdir"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		rec, err := c.Pipeline.PromoteRun(r.Context(), runID, req.TargetSubdir)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "stream":
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, unsub := c.Pipeline.Subscribe(runID)
		defer unsub()

		if current, err := c.FactoryRunRepo.Get(r.Context(), runID); err == nil {
			b, _ := json.Marshal(current)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
			if current.Status == "completed" || current.Status == "failed" {
				return
			}
		}

		for {
			select {
			case <-r.Context().Done():
				return
			case update := <-ch:
				b, _ := json.Marshal(update)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
				if update.Status == "completed" || update.Status == "failed" || update.Status == "awaiting_approval" {
					return
				}
			}
		}
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) handleHarnessStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ready",
		"llm_provider": s.container.LLMGateway.ProviderName(),
		"phase":        s.container.Phase(),
	})
}

func (s *webServer) handleSpecs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req spec.Specification
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid spec payload"})
			return
		}
		if req.Version == "" {
			req.Version = "1.0"
		}
		if errs := s.container.SpecEngine.ValidateSpec(&req); len(errs) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": strings.Join(errs, "; "), "validation_errors": errs})
			return
		}
		specsDir := filepath.Join(s.container.WorkingDir, "specs")
		_ = os.MkdirAll(specsDir, 0755)
		yamlBytes, err := yaml.Marshal(&req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		filename := sanitizeFilename(req.ID) + ".yaml"
		if err := os.WriteFile(filepath.Join(specsDir, filename), yamlBytes, 0644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"spec": req, "file": "specs/" + filename})
		return
	}

	all, err := s.container.SpecEngine.LoadAllSpecs()
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
		sp := all[id]
		specs = append(specs, map[string]interface{}{
			"id":          id,
			"title":       sp.Title,
			"version":     sp.Version,
			"description": sp.Description,
			"goal":        sp.Intent.Goal,
			"steps_count": len(sp.Steps),
			"constraints": sp.Constraints,
			"steps":       sp.Steps,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"specs": specs})
}

func (s *webServer) handleSpecByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/specs/")
	all, err := s.container.SpecEngine.LoadAllSpecs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	sp, ok := all[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "spec not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"spec":   sp,
		"prompt": s.container.SpecEngine.GeneratePrompt(sp),
	})
}

func (s *webServer) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]map[string]interface{}, 0)
	for _, cap := range s.container.AgentRegistry.GetAll() {
		agents = append(agents, map[string]interface{}{
			"id":           string(cap.Role),
			"role":         cap.Title,
			"description":  cap.Description,
			"capabilities": cap.Capabilities,
			"authority":    cap.DecisionAuthority,
			"mode":         cap.InteractionMode,
			"model":        cap.LLMModel,
		})
	}
	sort.Slice(agents, func(i, j int) bool {
		return fmt.Sprint(agents[i]["id"]) < fmt.Sprint(agents[j]["id"])
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
}

func (s *webServer) handleGitHubToken(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Forbidden: token endpoint disabled for security reasons", http.StatusForbidden)
}

func (s *webServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	outDir := filepath.Join(s.container.WorkingDir, "web-ui", "out")
	if info, err := os.Stat(filepath.Join(outDir, "index.html")); err == nil && !info.IsDir() {
		cleanReq := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanReq == "." || cleanReq == "" {
			http.ServeFile(w, r, filepath.Join(outDir, "index.html"))
			return
		}
		candidate := filepath.Join(outDir, cleanReq)
		if rel, relErr := filepath.Rel(outDir, candidate); relErr == nil && !strings.HasPrefix(rel, "..") {
			if st, stErr := os.Stat(candidate); stErr == nil && !st.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}
			if st, stErr := os.Stat(candidate + ".html"); stErr == nil && !st.IsDir() {
				http.ServeFile(w, r, candidate+".html")
				return
			}
		}
		http.NotFound(w, r)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.container.WorkingDir, "web", "index.html"))
}

func (s *webServer) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	tasks := s.container.Factory.RecentTasks(25)
	workflows := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		workflows = append(workflows, taskView(t))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"workflows": workflows})
}

func classifyIssueType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "security") || strings.Contains(lower, "vulnerability") || strings.Contains(lower, "xss") || strings.Contains(lower, "injection"):
		return "security-vulnerability"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "oauth") || strings.Contains(lower, "jwt") || strings.Contains(lower, "login"):
		return "auth"
	case strings.Contains(lower, "slow") || strings.Contains(lower, "latency") || strings.Contains(lower, "performance") || strings.Contains(lower, "memory"):
		return "performance-issue"
	case strings.Contains(lower, "bug") || strings.Contains(lower, "panic") || strings.Contains(lower, "crash") || strings.Contains(lower, "fix"):
		return "bug"
	default:
		return "feature"
	}
}

func sanitizeFilename(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "spec"
	}
	return b.String()
}

func main() {
	addr := "localhost:8080"
	if envAddr := os.Getenv("ASF_WEB_ADDR"); envAddr != "" {
		addr = envAddr
	}
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	workingDir, _ := os.Getwd()
	server, err := newWebServer(workingDir)
	if err != nil {
		log.Fatalf("Error initializing Agentic Software Factory server: %v", err)
	}
	defer server.container.Close()

	if err := handlers.InitializeIntegration(); err != nil {
		log.Printf("Warning: webhook queue init: %v", err)
	}

	fmt.Printf("Agentic Software Factory Control Plane listening on http://%s\n", addr)
	fmt.Printf("  SQLite Store : %s\n", server.container.Store.Path())
	fmt.Printf("  LLM Provider : %s\n", server.container.LLMGateway.ProviderName())
	fmt.Printf("  Active Phase : %s\n", server.container.Phase())
	fmt.Printf("  Next.js UI   : cd web-ui && npm run dev (http://127.0.0.1:3000)\n")

	if err := http.ListenAndServe(addr, server.routes()); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
