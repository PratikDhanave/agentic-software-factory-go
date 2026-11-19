// Package engine orchestrates autonomous Go software runs across explicit
// finite state machines, SSE streaming, human approval gates, and promotion.
package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/gateway"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/sandbox"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

type StartRequest struct {
	Prompt        string `json:"prompt"`
	SpecID        string `json:"spec_id"`
	AgentSlug     string `json:"agent_slug"`
	ForceApproval bool   `json:"force_approval"`
	Sync          bool   `json:"-"`
}

type Engine struct {
	rootDir string
	store   *store.Store
	gateway *gateway.Gateway

	mu       sync.RWMutex
	autonomy state.State
	subs     map[string]map[chan store.RunRecord]struct{}
}

func New(rootDir string, st *store.Store, gw *gateway.Gateway) *Engine {
	saved := state.State(st.GetSetting(context.Background(), "autonomy_mode", string(state.AutonomyWalk)))
	if !state.AutonomyMachine.IsValid(saved) {
		saved = state.AutonomyWalk
	}
	return &Engine{
		rootDir:  rootDir,
		store:    st,
		gateway:  gw,
		autonomy: saved,
		subs:     make(map[string]map[chan store.RunRecord]struct{}),
	}
}

func (e *Engine) SpecsDir() string {
	return filepath.Join(e.rootDir, "specs")
}

func (e *Engine) WorkspacesDir() string {
	return filepath.Join(e.rootDir, ".asf", "workspaces")
}

func (e *Engine) GeneratedDir() string {
	return filepath.Join(e.rootDir, "generated")
}

func (e *Engine) Autonomy() state.State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.autonomy
}

func (e *Engine) SetAutonomy(ctx context.Context, target string) (state.State, error) {
	next := state.State(strings.ToLower(strings.TrimSpace(target)))
	e.mu.Lock()
	defer e.mu.Unlock()
	if next == e.autonomy {
		return e.autonomy, nil
	}
	if err := state.AutonomyMachine.Transition(&e.autonomy, next); err != nil {
		return e.autonomy, err
	}
	_ = e.store.SetSetting(ctx, "autonomy_mode", string(e.autonomy))
	e.store.AddLog(ctx, store.LogEntry{
		Level:   "info",
		Message: fmt.Sprintf("Autonomy mode transitioned to %s", e.autonomy),
	})
	return e.autonomy, nil
}

func (e *Engine) Subscribe(runID string) (chan store.RunRecord, func()) {
	ch := make(chan store.RunRecord, 16)
	e.mu.Lock()
	if e.subs[runID] == nil {
		e.subs[runID] = make(map[chan store.RunRecord]struct{})
	}
	e.subs[runID][ch] = struct{}{}
	e.mu.Unlock()

	cancel := func() {
		e.mu.Lock()
		if m, ok := e.subs[runID]; ok {
			delete(m, ch)
			if len(m) == 0 {
				delete(e.subs, runID)
			}
		}
		e.mu.Unlock()
	}
	return ch, cancel
}

func (e *Engine) broadcast(r store.RunRecord) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for ch := range e.subs[r.ID] {
		select {
		case ch <- r:
		default:
		}
	}
}

func (e *Engine) matchSpec(specID, prompt string) *sandbox.Spec {
	specs, err := sandbox.LoadSpecs(e.SpecsDir())
	if err != nil {
		return nil
	}
	for i := range specs {
		if specID != "" && (specs[i].ID == specID || strings.TrimSuffix(specs[i].Filename, ".yaml") == specID) {
			return &specs[i]
		}
	}
	lp := strings.ToLower(prompt)
	for i := range specs {
		switch specs[i].ID {
		case "spec-token-bucket":
			if strings.Contains(lp, "token") || strings.Contains(lp, "rate limit") || strings.Contains(lp, "bucket") {
				return &specs[i]
			}
		case "spec-circuit-breaker":
			if strings.Contains(lp, "circuit") || strings.Contains(lp, "breaker") {
				return &specs[i]
			}
		case "spec-ttl-lru-cache":
			if strings.Contains(lp, "lru") || strings.Contains(lp, "cache") {
				return &specs[i]
			}
		case "spec-webhook-verifier", "spec-hmac-webhook":
			if strings.Contains(lp, "hmac") || strings.Contains(lp, "webhook") {
				return &specs[i]
			}
		}
	}
	return nil
}

func (e *Engine) StartRun(ctx context.Context, req StartRequest) (*store.RunRecord, error) {
	sp := e.matchSpec(req.SpecID, req.Prompt)
	title := strings.TrimSpace(req.Prompt)
	prompt := strings.TrimSpace(req.Prompt)
	specID := req.SpecID

	if sp != nil {
		specID = sp.ID
		if title == "" {
			title = sp.Title
		}
		if prompt == "" {
			prompt = sp.Intent.Goal + " — " + sp.Description
		}
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt or spec_id is required")
	}
	if len(title) > 72 {
		title = title[:72] + "..."
	}

	agentSlug := req.AgentSlug
	if agentSlug == "" {
		agentSlug = "factory-developer"
	}

	id := "run-" + uuid.NewString()[:8]
	autonomy := e.Autonomy()

	rec := &store.RunRecord{
		ID:           id,
		Title:        title,
		Prompt:       prompt,
		SpecID:       specID,
		AgentSlug:    agentSlug,
		Stage:        string(state.RunQueued),
		Status:       string(state.RunQueued),
		Autonomy:     string(autonomy),
		WorkspaceDir: filepath.Join(e.WorkspacesDir(), id),
		Promotion:    string(state.PromoteNone),
		Files:        []store.FileArtifact{},
		Steps: []store.StepEvent{
			{
				Stage:     string(state.RunQueued),
				Status:    "completed",
				Message:   fmt.Sprintf("Queued %s under %s autonomy mode", id, autonomy),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	if err := e.store.SaveRun(ctx, rec); err != nil {
		return nil, err
	}
	e.store.AddLog(ctx, store.LogEntry{
		RunID:     rec.ID,
		AgentSlug: rec.AgentSlug,
		Level:     "info",
		Message:   fmt.Sprintf("Started %s (%s)", rec.ID, rec.Title),
	})
	e.broadcast(*rec)

	pauseForApproval := req.ForceApproval || autonomy == state.AutonomyCrawl

	if req.Sync {
		e.executePlanning(context.Background(), rec.ID, sp, pauseForApproval)
		return e.store.GetRun(ctx, rec.ID)
	}
	go e.executePlanning(context.Background(), rec.ID, sp, pauseForApproval)
	return rec, nil
}

func (e *Engine) transitionRun(ctx context.Context, r *store.RunRecord, next state.State, stepMsg string, durMs int64) error {
	if err := state.FactoryRunMachine.TransitionString(&r.Status, next); err != nil {
		return err
	}
	r.Stage = string(next)
	r.Steps = append(r.Steps, store.StepEvent{
		Stage:      string(next),
		Status:     "completed",
		Message:    stepMsg,
		DurationMs: durMs,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
	if err := e.store.SaveRun(ctx, r); err != nil {
		return err
	}
	e.broadcast(*r)
	return nil
}

func (e *Engine) executePlanning(ctx context.Context, runID string, sp *sandbox.Spec, pauseForApproval bool) {
	r, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return
	}
	t0 := time.Now()
	planSummary := "Factory Architect & Researcher extracted Go package invariants and test plan"
	if sp != nil {
		planSummary = fmt.Sprintf("Loaded blueprint %s (%d steps, %d invariants)", sp.ID, len(sp.Steps), len(sp.Constraints))
	}
	if err := e.transitionRun(ctx, r, state.RunPlanning, planSummary, time.Since(t0).Milliseconds()); err != nil {
		return
	}

	if pauseForApproval {
		_ = e.transitionRun(ctx, r, state.RunAwaitingApproval, "Paused at Crawl Human-in-the-Loop checkpoint — awaiting operator approval", 0)
		e.store.AddLog(ctx, store.LogEntry{
			RunID:     r.ID,
			AgentSlug: "factory-architect",
			Level:     "warn",
			Message:   "Run paused at awaiting_approval checkpoint",
		})
		return
	}

	e.executeSynthesisAndVerify(ctx, r.ID)
}

func (e *Engine) ApproveRun(ctx context.Context, runID string) (*store.RunRecord, error) {
	r, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	gate := state.ApprovalPending
	if err := state.ApprovalMachine.Transition(&gate, state.ApprovalApproved); err != nil {
		return nil, err
	}
	if state.State(r.Status) != state.RunAwaitingApproval {
		return nil, fmt.Errorf("run %s is in state %q, not awaiting_approval", runID, r.Status)
	}
	e.store.AddLog(ctx, store.LogEntry{
		RunID:   r.ID,
		Level:   "info",
		Message: "Operator approved Human-in-the-Loop gate; resuming code synthesis",
	})
	go e.executeSynthesisAndVerify(context.Background(), r.ID)
	return r, nil
}

func (e *Engine) RejectRun(ctx context.Context, runID, reason string) (*store.RunRecord, error) {
	r, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	gate := state.ApprovalPending
	_ = state.ApprovalMachine.Transition(&gate, state.ApprovalRejected)
	if reason == "" {
		reason = "Rejected by operator at approval checkpoint"
	}
	r.Error = reason
	if err := e.transitionRun(ctx, r, state.RunFailed, reason, 0); err != nil {
		return nil, err
	}
	e.store.AddLog(ctx, store.LogEntry{
		RunID:   r.ID,
		Level:   "error",
		Message: reason,
	})
	return r, nil
}

func (e *Engine) executeSynthesisAndVerify(ctx context.Context, runID string) {
	r, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return
	}
	sp := e.matchSpec(r.SpecID, r.Prompt)
	var constraints []sandbox.SpecConstraint
	specTitle := r.Title
	if sp != nil {
		constraints = sp.Constraints
		specTitle = sp.Title
	}

	const maxAttempts = 3
	var repairFeedback string

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r.Attempts = attempt
		synthStart := time.Now()
		msg := fmt.Sprintf("Synthesizing Go implementation & table-driven tests (attempt %d/%d)", attempt, maxAttempts)
		if attempt > 1 {
			msg = fmt.Sprintf("Self-healing retry %d/%d feeding compiler/test diagnostics back to synthesizer", attempt, maxAttempts)
		}
		if err := e.transitionRun(ctx, r, state.RunSynthesizing, msg, 0); err != nil {
			return
		}

		genResp := e.gateway.GenerateCode(ctx, r.SpecID, specTitle, r.Prompt, repairFeedback)
		e.store.AddLog(ctx, store.LogEntry{
			RunID:      r.ID,
			AgentSlug:  r.AgentSlug,
			Provider:   genResp.Provider,
			Model:      genResp.Model,
			Tokens:     genResp.Tokens,
			DurationMs: genResp.DurationMs,
			Level:      "info",
			Message:    fmt.Sprintf("Synthesized Go package via %s (%s)", genResp.Provider, genResp.Model),
		})

		files := sandbox.ParseFiles(genResp.Content)
		r.Files = files
		synthDur := time.Since(synthStart).Milliseconds()

		verifyStart := time.Now()
		if err := e.transitionRun(ctx, r, state.RunVerifying, fmt.Sprintf("Running go build, go vet, and go test -race on %d files", len(files)), synthDur); err != nil {
			return
		}

		moduleSlug := r.SpecID
		if moduleSlug == "" {
			moduleSlug = r.Title
		}
		rep := sandbox.ProvisionAndVerify(ctx, r.WorkspaceDir, moduleSlug, files, constraints)
		r.BuildPassed = rep.BuildPassed
		r.VetPassed = rep.VetPassed
		r.TestsPassed = rep.TestsPassed
		r.TestCases = rep.TestCases
		r.TestOutput = rep.Output

		verifyDur := time.Since(verifyStart).Milliseconds()
		if rep.FinalState == state.VerifyPassed {
			r.Error = ""
			r.Score, r.Rubric = scoreRun(r)
			_ = e.transitionRun(ctx, r, state.RunCompleted, fmt.Sprintf("Passed go build, go vet, and %d race-checked tests (score %d/100)", len(r.TestCases), r.Score), verifyDur)
			e.store.AddLog(ctx, store.LogEntry{
				RunID:      r.ID,
				AgentSlug:  "factory-spec-verifier",
				DurationMs: verifyDur,
				Level:      "info",
				Message:    fmt.Sprintf("Verification succeeded (%d tests passed, score %d/100)", len(r.TestCases), r.Score),
			})
			return
		}

		repairFeedback = rep.Error
		r.Error = rep.Error
		_ = e.store.SaveRun(ctx, r)
		e.broadcast(*r)
	}

	_ = e.transitionRun(ctx, r, state.RunFailed, "Exhausted self-healing verification retries", 0)
	e.store.AddLog(ctx, store.LogEntry{
		RunID:     r.ID,
		AgentSlug: "factory-spec-verifier",
		Level:     "error",
		Message:   "Run failed verification after maximum retries",
	})
}

func (e *Engine) PromoteRun(ctx context.Context, runID, targetSlug string) (*store.RunRecord, error) {
	r, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if state.State(r.Status) != state.RunCompleted {
		return nil, fmt.Errorf("only completed runs can be promoted (current status: %s)", r.Status)
	}
	if r.Promotion == "" {
		r.Promotion = string(state.PromoteNone)
	}
	if err := state.PromotionMachine.TransitionString(&r.Promotion, state.PromoteRunning); err != nil {
		return nil, err
	}
	_ = e.store.SaveRun(ctx, r)

	if targetSlug == "" {
		if r.SpecID != "" {
			targetSlug = strings.TrimPrefix(r.SpecID, "spec-")
		} else {
			targetSlug = r.Title
		}
	}

	relDir, err := sandbox.Promote(r.WorkspaceDir, e.GeneratedDir(), targetSlug)
	if err != nil {
		_ = state.PromotionMachine.TransitionString(&r.Promotion, state.PromoteFailed)
		_ = e.store.SaveRun(ctx, r)
		return nil, err
	}

	_ = state.PromotionMachine.TransitionString(&r.Promotion, state.PromoteCompleted)
	r.PromotedDir = relDir
	r.Steps = append(r.Steps, store.StepEvent{
		Stage:     "promoted",
		Status:    "completed",
		Message:   fmt.Sprintf("Promoted verified Go package to %s/", relDir),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err := e.store.SaveRun(ctx, r); err != nil {
		return nil, err
	}
	e.store.AddLog(ctx, store.LogEntry{
		RunID:   r.ID,
		Level:   "info",
		Message: fmt.Sprintf("Promoted %s -> %s/", r.ID, relDir),
	})
	e.broadcast(*r)
	return r, nil
}

func scoreRun(r *store.RunRecord) (int, map[string]int) {
	rubric := map[string]int{
		"compilation":     0,
		"static_analysis": 0,
		"race_safety":     0,
		"test_depth":      0,
		"first_pass":      0,
	}
	if r.BuildPassed {
		rubric["compilation"] = 25
	}
	if r.VetPassed {
		rubric["static_analysis"] = 20
	}
	if r.TestsPassed {
		rubric["race_safety"] = 25
	}
	if len(r.TestCases) >= 2 {
		rubric["test_depth"] = 20
	} else if len(r.TestCases) == 1 {
		rubric["test_depth"] = 14
	}
	if r.Attempts <= 1 {
		rubric["first_pass"] = 10
	} else {
		rubric["first_pass"] = 6
	}
	total := 0
	for _, v := range rubric {
		total += v
	}
	return total, rubric
}
