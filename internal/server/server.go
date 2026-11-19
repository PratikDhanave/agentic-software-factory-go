// Package server hosts the unified HTTP Control Plane API, SSE stream,
// GitHub webhook receiver, JSON-RPC 2.0 MCP endpoint, and embedded Next.js UI.
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/engine"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/gateway"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/mcp"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/sandbox"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

//go:embed all:dist
var embeddedDist embed.FS

type Server struct {
	rootDir string
	store   *store.Store
	gateway *gateway.Gateway
	engine  *engine.Engine
	mcp     *mcp.Handler
	started time.Time
	mux     *http.ServeMux
}

func New(rootDir string, st *store.Store, gw *gateway.Gateway, eng *engine.Engine) *Server {
	s := &Server{
		rootDir: rootDir,
		store:   st,
		gateway: gw,
		engine:  eng,
		mcp:     mcp.New(st, eng),
		started: time.Now(),
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withCORS(s.mux)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/webhooks/github", s.handleGitHubWebhook)

	// Control Plane API (loopback or ASF_API_KEY protected)
	s.mux.HandleFunc("/api/overview", s.guard(s.handleOverview))
	s.mux.HandleFunc("/api/agents", s.guard(s.handleAgents))
	s.mux.HandleFunc("/api/agents/consult", s.guard(s.handleConsult))
	s.mux.HandleFunc("/api/specs", s.guard(s.handleSpecs))
	s.mux.HandleFunc("/api/runs", s.guard(s.handleRuns))
	s.mux.HandleFunc("/api/runs/", s.guard(s.handleRunDetail))
	s.mux.HandleFunc("/api/autonomy", s.guard(s.handleAutonomy))
	s.mux.HandleFunc("/api/gateway", s.guard(s.handleGateway))
	s.mux.HandleFunc("/api/logs", s.guard(s.handleLogs))
	s.mux.HandleFunc("/api/usage", s.guard(s.handleUsage))
	s.mux.HandleFunc("/api/triggers", s.guard(s.handleTriggers))
	s.mux.HandleFunc("/api/environments", s.guard(s.handleEnvironments))
	s.mux.HandleFunc("/api/state-machines", s.guard(s.handleStateMachines))
	s.mux.HandleFunc("/api/state-machines/validate", s.guard(s.handleValidateTransition))
	s.mux.HandleFunc("/api/mcp/tools", s.guard(s.handleMCPTools))
	s.mux.HandleFunc("/api/mcp", s.guard(s.handleMCP))

	// Embedded Next.js Control Plane SPA
	subFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return
	}
	fileServer := http.FileServer(http.FS(subFS))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if f, err := subFS.Open(clean); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		if f, err := subFS.Open(clean + ".html"); err == nil {
			_ = f.Close()
			r2 := *r
			u2 := *r.URL
			u2.Path = "/" + clean + ".html"
			r2.URL = &u2
			fileServer.ServeHTTP(w, &r2)
			return
		}
		// SPA fallback to index.html
		b, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-ASF-API-Key")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := strings.TrimSpace(os.Getenv("ASF_API_KEY"))
		if apiKey != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got == "" {
				got = r.Header.Get("X-ASF-API-Key")
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) == 1 {
				next(w, r)
				return
			}
			writeErr(w, http.StatusUnauthorized, "valid ASF_API_KEY required")
			return
		}
		if !isLoopback(r.RemoteAddr) {
			writeErr(w, http.StatusForbidden, "control plane API restricted to loopback unless ASF_API_KEY is set")
			return
		}
		next(w, r)
	}
}

func isLoopback(remoteAddr string) bool {
	if remoteAddr == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	prov, model := s.gateway.ActiveProvider()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "healthy",
		"uptime_seconds":  int(time.Since(s.started).Seconds()),
		"autonomy":        s.engine.Autonomy(),
		"active_provider": prov,
		"active_model":    model,
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agents, _ := s.store.ListAgents(ctx)
	runs, _ := s.store.ListRuns(ctx, 25)
	specs, _ := sandbox.LoadSpecs(s.engine.SpecsDir())
	triggers, _ := s.store.ListTriggers(ctx, 10)
	usage := s.store.UsageSummary(ctx)
	gw := s.gateway.Status()

	var completed, awaiting, active, promoted, scoreSum int
	for _, run := range runs {
		switch state.State(run.Status) {
		case state.RunCompleted:
			completed++
			scoreSum += run.Score
		case state.RunAwaitingApproval:
			awaiting++
		case state.RunQueued, state.RunPlanning, state.RunSynthesizing, state.RunVerifying:
			active++
		}
		if run.Promotion == string(state.PromoteCompleted) {
			promoted++
		}
	}
	avgScore := 0
	if completed > 0 {
		avgScore = scoreSum / completed
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "healthy",
		"uptime_seconds":   int(time.Since(s.started).Seconds()),
		"autonomy":         s.engine.Autonomy(),
		"gateway":          gw,
		"usage":            usage,
		"agents_count":     len(agents),
		"specs_count":      len(specs),
		"machines_count":   len(state.All()),
		"mcp_tools_count":  len(mcp.ToolsCatalog()),
		"runs_total":       len(runs),
		"runs_completed":   completed,
		"runs_awaiting":    awaiting,
		"runs_active":      active,
		"runs_promoted":    promoted,
		"avg_quality":      avgScore,
		"recent_runs":      runs,
		"recent_triggers":  triggers,
		"agents":           agents,
		"specs":            specs,
	})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

func (s *Server) handleConsult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req struct {
		AgentSlug string `json:"agent_slug"`
		Prompt    string `json:"prompt"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeErr(w, http.StatusBadRequest, "prompt is required")
		return
	}
	ag, err := s.store.GetAgent(r.Context(), req.AgentSlug)
	if err != nil {
		writeErr(w, http.StatusNotFound, "agent not found")
		return
	}
	resp := s.gateway.Consult(r.Context(), ag, req.Prompt)
	s.store.AddLog(r.Context(), store.LogEntry{
		AgentSlug:  ag.Slug,
		Provider:   resp.Provider,
		Model:      resp.Model,
		Tokens:     resp.Tokens,
		DurationMs: resp.DurationMs,
		Level:      "info",
		Message:    fmt.Sprintf("Consultation with %s", ag.Name),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"agent":       ag,
		"reply":       resp.Content,
		"provider":    resp.Provider,
		"model":       resp.Model,
		"tokens":      resp.Tokens,
		"duration_ms": resp.DurationMs,
	})
}

func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		specs, err := sandbox.LoadSpecs(s.engine.SpecsDir())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"specs": specs})
	case http.MethodPost:
		var sp sandbox.Spec
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&sp); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid spec JSON")
			return
		}
		if strings.TrimSpace(sp.Title) == "" {
			writeErr(w, http.StatusBadRequest, "title is required")
			return
		}
		saved, err := sandbox.SaveSpec(s.engine.SpecsDir(), sp)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		runs, err := s.store.ListRuns(r.Context(), 50)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
	case http.MethodPost:
		var req engine.StartRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		rec, err := s.engine.StartRun(r.Context(), req)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeErr(w, http.StatusNotFound, "missing run id")
		return
	}
	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "":
		rec, err := s.store.GetRun(r.Context(), runID)
		if err != nil {
			writeErr(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "stream":
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeErr(w, http.StatusInternalServerError, "streaming unsupported")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, cancel := s.engine.Subscribe(runID)
		defer cancel()

		if cur, err := s.store.GetRun(r.Context(), runID); err == nil {
			b, _ := json.Marshal(cur)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
			if state.FactoryRunMachine.IsTerminal(state.State(cur.Status)) {
				return
			}
		}

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case rec := <-ch:
				b, _ := json.Marshal(rec)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
				if state.FactoryRunMachine.IsTerminal(state.State(rec.Status)) {
					return
				}
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			}
		}

	case "approve":
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		rec, err := s.engine.ApproveRun(r.Context(), runID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "reject":
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body)
		rec, err := s.engine.RejectRun(r.Context(), runID, body.Reason)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case "promote":
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var body struct {
			Target string `json:"target"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body)
		rec, err := s.engine.PromoteRun(r.Context(), runID, body.Target)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rec)

	default:
		writeErr(w, http.StatusNotFound, "unknown run action")
	}
}

func (s *Server) handleAutonomy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cur := s.engine.Autonomy()
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":    cur,
			"allowed": state.AutonomyMachine.AllowedFrom(cur),
			"machine": state.AutonomyMachine.Export(),
		})
	case http.MethodPost:
		var body struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		next, err := s.engine.SetAutonomy(r.Context(), body.Mode)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":    next,
			"allowed": state.AutonomyMachine.AllowedFrom(next),
		})
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

func (s *Server) handleGateway(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.gateway.Status())
	case http.MethodPost:
		var cfg gateway.UpdateConfig
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		status := s.gateway.Update(r.Context(), cfg)
		s.store.AddLog(r.Context(), store.LogEntry{
			Level:   "info",
			Message: fmt.Sprintf("Model Gateway updated -> active provider: %v (%v)", status["active_provider"], status["active_model"]),
		})
		writeJSON(w, http.StatusOK, status)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.ListLogs(r.Context(), 120)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.UsageSummary(r.Context()))
}

func (s *Server) handleTriggers(w http.ResponseWriter, r *http.Request) {
	triggers, err := s.store.ListTriggers(r.Context(), 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"triggers": triggers})
}

func (s *Server) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"go_version":     runtime.Version(),
		"os_arch":        fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		"root_dir":       s.rootDir,
		"sqlite_db":      s.store.Path(),
		"workspaces_dir": s.engine.WorkspacesDir(),
		"generated_dir":  s.engine.GeneratedDir(),
		"specs_dir":      s.engine.SpecsDir(),
		"gates": []string{
			"go build -buildvcs=false ./...",
			"go vet -buildvcs=false ./...",
			"go test -v -race -buildvcs=false ./...",
			"YAML specification must_not constraints",
		},
	})
}

func (s *Server) handleStateMachines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(state.All()),
		"machines": state.Exports(),
	})
}

func (s *Server) handleValidateTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var body struct {
		Machine string `json:"machine"`
		From    string `json:"from"`
		To      string `json:"to"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	m, ok := state.ByName(body.Machine)
	if !ok {
		writeErr(w, http.StatusNotFound, "unknown state machine")
		return
	}
	from := state.State(body.From)
	to := state.State(body.To)
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
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"endpoint": "/api/mcp",
		"protocol": "JSON-RPC 2.0 (MCP 2024-11-05)",
		"tools":    mcp.ToolsCatalog(),
	})
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req mcp.RPCRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, mcp.RPCResponse{
			JSONRPC: "2.0",
			Error:   &mcp.RPCError{Code: -32700, Message: "parse error"},
		})
		return
	}
	resp := s.mcp.Handle(r.Context(), req)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "failed reading body")
		return
	}

	curState := state.TriggerReceived
	secret := strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	if secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(sig)) {
			_ = state.TriggerMachine.Transition(&curState, state.TriggerFailed)
			_ = s.store.SaveTrigger(r.Context(), store.TriggerRecord{
				ID:        "trg-" + uuid.NewString()[:8],
				Source:    "github",
				EventType: r.Header.Get("X-GitHub-Event"),
				Status:    string(curState),
				Summary:   "Rejected: invalid X-Hub-Signature-256",
			})
			writeErr(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		eventType = "manual_webhook"
	}
	var payload struct {
		Action  string `json:"action"`
		Prompt  string `json:"prompt"`
		SpecID  string `json:"spec_id"`
		Issue   struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"issue"`
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
	}
	_ = json.Unmarshal(body, &payload)

	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" && strings.Contains(payload.Comment.Body, "@factory") {
		prompt = strings.TrimSpace(strings.ReplaceAll(payload.Comment.Body, "@factory", ""))
	}
	if prompt == "" && payload.Issue.Title != "" {
		prompt = payload.Issue.Title + ": " + payload.Issue.Body
	}

	trigID := "trg-" + uuid.NewString()[:8]
	if prompt == "" && payload.SpecID == "" {
		_ = state.TriggerMachine.Transition(&curState, state.TriggerIgnored)
		tr := store.TriggerRecord{
			ID:        trigID,
			Source:    "github",
			EventType: eventType,
			Status:    string(curState),
			Summary:   "Ignored event without @factory directive or prompt",
		}
		_ = s.store.SaveTrigger(r.Context(), tr)
		writeJSON(w, http.StatusOK, tr)
		return
	}

	runRec, err := s.engine.StartRun(r.Context(), engine.StartRequest{
		Prompt: prompt,
		SpecID: payload.SpecID,
	})
	if err != nil {
		_ = state.TriggerMachine.Transition(&curState, state.TriggerFailed)
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = state.TriggerMachine.Transition(&curState, state.TriggerDispatched)
	tr := store.TriggerRecord{
		ID:        trigID,
		Source:    "github",
		EventType: eventType,
		Status:    string(curState),
		RunID:     runRec.ID,
		Summary:   fmt.Sprintf("Dispatched %s for %s", runRec.ID, runRec.Title),
	}
	_ = s.store.SaveTrigger(r.Context(), tr)
	writeJSON(w, http.StatusAccepted, tr)
}
