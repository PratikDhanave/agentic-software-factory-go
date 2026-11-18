package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence/sqlite"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/usecases"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// Explicit stages of the Autonomous Software Factory Assembly Line.
const (
	RunStageQueued            state.State = "queued"
	RunStageSpecSynthesis     state.State = "spec_synthesis"
	RunStageAgentGovernance   state.State = "agent_governance"
	RunStageAwaitingApproval  state.State = "awaiting_approval"
	RunStageWorkspaceInit     state.State = "workspace_init"
	RunStageCodeSynthesis     state.State = "code_synthesis"
	RunStageVerificationLoop  state.State = "verification_loop"
	RunStageJudgeEvaluation   state.State = "judge_evaluation"
	RunStageCompleted         state.State = "completed"
	RunStageFailed            state.State = "failed"
)

// FactoryRunMachine declares every legal stage transition of an end-to-end FactoryRun.
var FactoryRunMachine = state.NewMachine("factory-assembly-line", RunStageQueued, map[state.State][]state.State{
	RunStageQueued:           {RunStageSpecSynthesis, RunStageFailed},
	RunStageSpecSynthesis:    {RunStageAgentGovernance, RunStageFailed},
	RunStageAgentGovernance:  {RunStageAwaitingApproval, RunStageWorkspaceInit, RunStageFailed},
	RunStageAwaitingApproval: {RunStageWorkspaceInit, RunStageFailed},
	RunStageWorkspaceInit:    {RunStageCodeSynthesis, RunStageFailed},
	RunStageCodeSynthesis:    {RunStageVerificationLoop, RunStageFailed},
	RunStageVerificationLoop: {RunStageJudgeEvaluation, RunStageCodeSynthesis, RunStageFailed},
	RunStageJudgeEvaluation:  {RunStageCompleted, RunStageCodeSynthesis, RunStageFailed},
}).WithCategory("assembly").
	WithDescription("6-stage Autonomous Software Factory Assembly Line from specification synthesis to verified Go deliverable.").
	WithStateDescriptions(map[state.State]string{
		RunStageQueued:           "Run created and persisted in SQLite.",
		RunStageSpecSynthesis:    "Stage 1: Synthesizing or validating machine-readable Specification YAML.",
		RunStageAgentGovernance:  "Stage 2: 10 specialized agents coordinating architecture, security, and policy.",
		RunStageAwaitingApproval: "Paused at Human-in-the-Loop checkpoint awaiting operator approval.",
		RunStageWorkspaceInit:    "Stage 3: Provisioning isolated Go module sandbox under .asf/workspaces/.",
		RunStageCodeSynthesis:    "Stage 4: Generating multi-file Go implementation and table-driven tests.",
		RunStageVerificationLoop: "Stage 5: Running go build, go test -race, go vet, and self-healing retries.",
		RunStageJudgeEvaluation:  "Stage 6: Evaluating 5-criterion engineering quality rubric.",
		RunStageCompleted:        "All stages verified; package ready for promotion to generated/.",
		RunStageFailed:           "Run terminated due to unrecoverable validation or verification failure.",
	})

// FactoryRunStatusMachine explicitly governs the high-level run status field.
var FactoryRunStatusMachine = state.NewMachine("factory-run-status", "running", map[state.State][]state.State{
	"running":           {"awaiting_approval", "completed", "failed"},
	"awaiting_approval": {"running", "failed"},
}).WithCategory("assembly").
	WithDescription("High-level status of a FactoryRun across active execution, human approval pause, and completion.").
	WithStateDescriptions(map[state.State]string{
		"running":           "Actively advancing through assembly line stages.",
		"awaiting_approval": "Paused awaiting human operator approval.",
		"completed":         "Successfully verified and scored.",
		"failed":            "Terminated with error.",
	})

// ArtifactPromotionMachine governs exporting verified workspace packages into generated/<slug>/.
var ArtifactPromotionMachine = state.NewMachine("artifact-promotion", "unpromoted", map[state.State][]state.State{
	"unpromoted": {"promoting"},
	"promoting":  {"promoted", "failed"},
	"promoted":   {"promoting"},
	"failed":     {"promoting"},
}).WithCategory("assembly").
	WithDescription("Controls promotion of verified workspace Go packages into the repository generated/ directory.").
	WithStateDescriptions(map[state.State]string{
		"unpromoted": "Verified in .asf/workspaces/, not yet exported to generated/.",
		"promoting":  "Validating target path boundary and copying verified .go files.",
		"promoted":   "Exported cleanly into generated/<slug>/.",
		"failed":     "Promotion failed; eligible to retry.",
	})

// PipelineMachines returns all explicit state machines owned by the FactoryPipeline.
func PipelineMachines() []*state.Machine {
	return []*state.Machine{
		FactoryRunMachine,
		FactoryRunStatusMachine,
		ArtifactPromotionMachine,
	}
}

type StageTimelineEntry struct {
	Stage      string `json:"stage"`
	Status     string `json:"status"` // running, completed, paused, failed
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Timestamp  string `json:"timestamp"`
}

type FactoryRunInput struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	SpecID         string   `json:"spec_id"`
	Constraints    []string `json:"constraints"`
	ForceApproval  bool     `json:"force_approval"`
	Phase          string   `json:"phase"`
}

// FactoryPipeline orchestrates the closed-loop 6-stage Software Factory Assembly Line:
// Spec Synthesis -> 10-Agent Policy & HITL Governance -> Isolated Workspace Provisioning ->
// Multi-File Code & Test Synthesis -> Self-Healing Compiler/Test Verification ->
// LLM-as-a-Judge Rubric Gate & Learning Feedback.
type FactoryPipeline struct {
	workingDir    string
	runRepo       *sqlite.FactoryRunRepository
	sessionRepo   *sqlite.CoordinationSessionRepository
	learningRepo  *sqlite.LearningRepository
	specEngine    *spec.SpecEngine
	coordinateUC  *usecases.CoordinateWorkflowUseCase
	executeDAGUC  *usecases.ExecuteWorkflowUseCase
	harness       *harness.CompleteHarness
	llmProvider   services.LLMProvider
	agentClient   harness.AgentClient
	evaluator     *eval.EvaluationEngine
	selfImproving *SelfImprovingService
	logger        services.Logger

	subMu       sync.RWMutex
	subscribers map[string][]chan *sqlite.FactoryRunRecord
}

type FactoryPipelineDeps struct {
	WorkingDir    string
	RunRepo       *sqlite.FactoryRunRepository
	SessionRepo   *sqlite.CoordinationSessionRepository
	LearningRepo  *sqlite.LearningRepository
	SpecEngine    *spec.SpecEngine
	CoordinateUC  *usecases.CoordinateWorkflowUseCase
	ExecuteDAGUC  *usecases.ExecuteWorkflowUseCase
	Harness       *harness.CompleteHarness
	LLMProvider   services.LLMProvider
	AgentClient   harness.AgentClient
	EvalClient    eval.LLMClient
	SelfImproving *SelfImprovingService
	Logger        services.Logger
}

func NewFactoryPipeline(deps FactoryPipelineDeps) *FactoryPipeline {
	var evaluator *eval.EvaluationEngine
	if deps.EvalClient != nil {
		evaluator = eval.NewEvaluationEngine(deps.EvalClient)
	}
	return &FactoryPipeline{
		workingDir:    deps.WorkingDir,
		runRepo:       deps.RunRepo,
		sessionRepo:   deps.SessionRepo,
		learningRepo:  deps.LearningRepo,
		specEngine:    deps.SpecEngine,
		coordinateUC:  deps.CoordinateUC,
		executeDAGUC:  deps.ExecuteDAGUC,
		harness:       deps.Harness,
		llmProvider:   deps.LLMProvider,
		agentClient:   deps.AgentClient,
		evaluator:     evaluator,
		selfImproving: deps.SelfImproving,
		logger:        deps.Logger,
		subscribers:   make(map[string][]chan *sqlite.FactoryRunRecord),
	}
}

// StartRun creates a durable FactoryRunRecord and executes the assembly line
// asynchronously (or synchronously when syncMode is true).
func (fp *FactoryPipeline) StartRun(ctx context.Context, in FactoryRunInput, syncMode bool) (*sqlite.FactoryRunRecord, error) {
	if strings.TrimSpace(in.Title) == "" && strings.TrimSpace(in.SpecID) == "" {
		return nil, fmt.Errorf("either title or spec_id is required")
	}

	phase := strings.ToLower(strings.TrimSpace(in.Phase))
	if phase == "" {
		phase = "walk"
	}

	title := strings.TrimSpace(in.Title)
	desc := strings.TrimSpace(in.Description)
	if in.SpecID != "" && title == "" {
		if specs, err := fp.specEngine.LoadAllSpecs(); err == nil {
			if sp, ok := specs[in.SpecID]; ok {
				title = sp.Title
				if desc == "" {
					desc = sp.Description
				}
			}
		}
		if title == "" {
			title = in.SpecID
		}
	}

	runID := "run-" + uuid.New().String()[:8]
	rec := &sqlite.FactoryRunRecord{
		ID:          runID,
		Title:       title,
		Description: desc,
		SpecID:      in.SpecID,
		PhaseMode:   phase,
		Stage:       string(RunStageQueued),
		Status:      "running",
		Payload: map[string]interface{}{
			"constraints": in.Constraints,
			"timeline": []StageTimelineEntry{
				{
					Stage:     string(RunStageQueued),
					Status:    "completed",
					Message:   "Factory assembly line run created",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				},
			},
		},
		CreatedAt: time.Now(),
	}

	if err := fp.runRepo.Save(ctx, rec); err != nil {
		return nil, fmt.Errorf("saving initial factory run: %w", err)
	}
	fp.broadcast(rec)

	if syncMode {
		fp.driveFromIntake(ctx, rec, in)
		return fp.runRepo.Get(ctx, rec.ID)
	}

	go fp.driveFromIntake(context.Background(), rec, in)
	return rec, nil
}

func (fp *FactoryPipeline) driveFromIntake(ctx context.Context, rec *sqlite.FactoryRunRecord, in FactoryRunInput) {
	// Stage 1: Specification Synthesis / Validation
	stageStart := time.Now()
	fp.transition(ctx, rec, RunStageSpecSynthesis, "running", "Synthesizing and validating machine-readable specification...", 0)

	specObj, err := fp.resolveOrSynthesizeSpec(ctx, rec, in)
	if err != nil {
		fp.failRun(ctx, rec, fmt.Errorf("spec_synthesis: %w", err))
		return
	}
	rec.SpecID = specObj.ID
	rec.Title = specObj.Title
	if rec.Description == "" {
		rec.Description = specObj.Description
	}
	specYAML, _ := yaml.Marshal(specObj)
	rec.Payload["spec"] = specObj
	rec.Payload["spec_yaml"] = string(specYAML)
	fp.appendTimeline(rec, string(RunStageSpecSynthesis), "completed", fmt.Sprintf("Validated specification %s (%d steps)", specObj.ID, len(specObj.Steps)), time.Since(stageStart))
	_ = fp.runRepo.Save(ctx, rec)
	fp.broadcast(rec)

	// Stage 2: Multi-Agent Architecture & Policy Governance
	stageStart = time.Now()
	fp.transition(ctx, rec, RunStageAgentGovernance, "running", "Coordinating specialized agents & evaluating maturity policy...", 0)

	classification := classifyRequirement(specObj.Title + " " + specObj.Description)
	forceApproval := in.ForceApproval || rec.PhaseMode == "crawl"

	coordOut, err := fp.coordinateUC.Execute(ctx, &usecases.CoordinateWorkflowInput{
		TriggerEvent: map[string]interface{}{
			"title":       specObj.Title,
			"description": specObj.Description,
			"spec_id":     specObj.ID,
			"run_id":      rec.ID,
		},
		TriggerType:      "factory_run",
		Classification:   classification,
		ApprovalRequired: forceApproval,
	})
	if err != nil {
		fp.failRun(ctx, rec, fmt.Errorf("agent_governance: %w", err))
		return
	}

	rec.SessionID = coordOut.Session.ID
	rec.Payload["workflow_id"] = coordOut.Workflow.ID
	rec.Payload["classification"] = classification

	execUUID := uuid.New()
	rec.Payload["learning_exec_uuid"] = execUUID.String()

	roles := make([]string, 0, len(coordOut.Decisions))
	for r := range coordOut.Decisions {
		roles = append(roles, r)
	}
	sort.Strings(roles)

	anyNeedsApproval := false
	decisionsView := make([]map[string]interface{}, 0, len(roles))
	var decisionIDs []string

	for _, role := range roles {
		d := coordOut.Decisions[role]
		if d == nil {
			continue
		}
		skip := !forceApproval && fp.selfImproving.ShouldSkipApproval(ctx, role, classification, d.Confidence, rec.PhaseMode)
		d.NeedsApproval = !skip
		if d.NeedsApproval {
			anyNeedsApproval = true
		}

		decID := uuid.New()
		decisionIDs = append(decisionIDs, decID.String())
		appStatus := "approved"
		if d.NeedsApproval {
			appStatus = "pending"
		}

		agentDec := &domain.AgentDecision{
			ID:               decID,
			ExecutionID:      execUUID,
			AgentRole:        role,
			IssueType:        classification,
			Decision:         d.Choice,
			Reasoning:        d.Reasoning,
			Confidence:       d.Confidence,
			Alternatives:     d.Alternatives,
			RequiresApproval: d.NeedsApproval,
			ApprovalStatus:   appStatus,
			CreatedAt:        time.Now(),
		}
		_ = fp.learningRepo.SaveDecision(ctx, agentDec)
		_ = fp.selfImproving.RecordDecision(ctx, agentDec, classification)

		decisionsView = append(decisionsView, map[string]interface{}{
			"id":              coordOut.Session.ID + ":" + role,
			"agent_role":      role,
			"decision":        d.Choice,
			"reasoning":       d.Reasoning,
			"confidence":      d.Confidence,
			"alternatives":    d.Alternatives,
			"needs_approval":  d.NeedsApproval,
			"approval_status": appStatus,
		})
	}
	rec.Payload["decisions"] = decisionsView
	rec.Payload["decision_uuids"] = decisionIDs

	if anyNeedsApproval {
		coordOut.Session.RequestApproval()
		_ = fp.sessionRepo.SaveSession(coordOut.Session)
		_ = FactoryRunStatusMachine.TransitionString(&rec.Status, "awaiting_approval")
		fp.transition(ctx, rec, RunStageAwaitingApproval, "paused", "Paused at Human-in-the-Loop Governance Gate — approve to build & verify", time.Since(stageStart))
		return
	}

	coordOut.Session.BeginExecution()
	_ = fp.sessionRepo.SaveSession(coordOut.Session)
	fp.appendTimeline(rec, string(RunStageAgentGovernance), "completed", fmt.Sprintf("Policy gate passed (%d agents aligned in %s mode)", len(decisionsView), rec.PhaseMode), time.Since(stageStart))

	fp.driveExecutionStages(ctx, rec, specObj)
}

// ResumeBySessionID resumes a paused FactoryRun once its coordination session is approved.
func (fp *FactoryPipeline) ResumeBySessionID(ctx context.Context, sessionID, approver string, syncMode bool) (*sqlite.FactoryRunRecord, error) {
	rec, err := fp.runRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return fp.ResumeRun(ctx, rec.ID, approver, syncMode)
}

// ResumeRun approves any remaining pending gates on a paused FactoryRun and drives Stages 3->6.
func (fp *FactoryPipeline) ResumeRun(ctx context.Context, runID, approver string, syncMode bool) (*sqlite.FactoryRunRecord, error) {
	rec, err := fp.runRepo.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	if rec.Stage != string(RunStageAwaitingApproval) && rec.Status != "awaiting_approval" {
		return rec, nil
	}
	if approver == "" {
		approver = "human-operator"
	}

	if rec.SessionID != "" {
		_ = fp.sessionRepo.ApproveSession(rec.SessionID, approver)
	}

	// Mark decisions approved in payload
	if decs, ok := rec.Payload["decisions"].([]interface{}); ok {
		for _, item := range decs {
			if m, isMap := item.(map[string]interface{}); isMap {
				m["needs_approval"] = false
				m["approval_status"] = "approved"
			}
		}
	}

	specs, err := fp.specEngine.LoadAllSpecs()
	if err != nil {
		return nil, err
	}
	specObj, ok := specs[rec.SpecID]
	if !ok {
		return nil, fmt.Errorf("spec %s not found for run %s", rec.SpecID, rec.ID)
	}

	_ = FactoryRunStatusMachine.TransitionString(&rec.Status, "running")
	fp.appendTimeline(rec, string(RunStageAwaitingApproval), "completed", "Approved by "+approver+" — resuming workspace execution", 0)
	_ = fp.runRepo.Save(ctx, rec)
	fp.broadcast(rec)

	if syncMode {
		fp.driveExecutionStages(ctx, rec, specObj)
		return fp.runRepo.Get(ctx, rec.ID)
	}
	go fp.driveExecutionStages(context.Background(), rec, specObj)
	return rec, nil
}

func (fp *FactoryPipeline) driveExecutionStages(ctx context.Context, rec *sqlite.FactoryRunRecord, specObj *spec.Specification) {
	// Also execute the coordinated DAG if present
	if wfID, ok := rec.Payload["workflow_id"].(string); ok && wfID != "" && fp.executeDAGUC != nil {
		if dagExec, err := fp.executeDAGUC.Execute(ctx, wfID, map[string]interface{}{"spec_id": specObj.ID}); err == nil {
			rec.Payload["dag_execution"] = dagExec
		}
	}

	// Stage 3: Isolated Workspace Init
	wsStart := time.Now()
	fp.transition(ctx, rec, RunStageWorkspaceInit, "running", "Provisioning isolated Go module sandbox...", 0)
	fp.appendTimeline(rec, string(RunStageWorkspaceInit), "completed", "Isolated workspace ready under .asf/workspaces/"+rec.ID, time.Since(wsStart))

	// Stage 4 & 5: Multi-File Code Synthesis + Self-Healing Compiler/Race-Test Loop
	codeStart := time.Now()
	fp.transition(ctx, rec, RunStageCodeSynthesis, "running", fmt.Sprintf("Synthesizing Go implementation & tests via %s...", fp.llmProvider.ProviderName()), 0)

	fp.transition(ctx, rec, RunStageVerificationLoop, "running", "Running go build, go test -race, go vet & spec constraints...", 0)

	harnessRes, err := fp.harness.ExecuteSpecWithAgent(ctx, specObj, rec.ID, fp.agentClient)
	if err != nil {
		fp.failRun(ctx, rec, fmt.Errorf("verification_loop: %w", err))
		return
	}

	rec.WorkspaceDir = harnessRes.WorkspaceDir
	rec.Payload["files_written"] = harnessRes.FilesWritten
	rec.Payload["diff"] = harnessRes.Diff
	rec.Payload["harness_state_trace"] = harnessRes.StateTrace
	if harnessRes.Verification != nil {
		v := harnessRes.Verification
		rec.Payload["verification"] = map[string]interface{}{
			"success":       v.Success,
			"phase":         v.Phase,
			"failed_phase":  v.FailedPhase,
			"build_passed":  v.Success || v.FailedPhase != "build",
			"tests_passed":  v.Success || (v.FailedPhase != "build" && v.FailedPhase != "test"),
			"lint_passed":   v.Success || (v.FailedPhase != "build" && v.FailedPhase != "test" && v.FailedPhase != "lint"),
			"test_output":   v.TestOutput,
			"raw_test_log":  v.TestOutput,
			"build_output":  v.BuildOutput,
			"lint_output":   v.LintOutput,
			"errors":        v.Errors,
			"retry_count":   harnessRes.RetryCount,
		}
	}
	rec.Payload["attempts"] = harnessRes.RetryCount + 1
	rec.Payload["files"] = readWorkspaceFiles(harnessRes.WorkspaceDir, harnessRes.FilesWritten)

	if !harnessRes.Success {
		fp.failRun(ctx, rec, fmt.Errorf("workspace verification failed after %d attempt(s)", harnessRes.RetryCount+1))
		return
	}

	retryNote := "passed on first attempt"
	if harnessRes.RetryCount > 0 {
		retryNote = fmt.Sprintf("self-healed after %d compiler/test retry(s)", harnessRes.RetryCount)
	}
	fp.appendTimeline(rec, string(RunStageVerificationLoop), "completed", fmt.Sprintf("go build + go test + go vet passed (%s)", retryNote), time.Since(codeStart))

	// Stage 6: LLM-as-a-Judge Quality Gate & Learning Feedback
	judgeStart := time.Now()
	fp.transition(ctx, rec, RunStageJudgeEvaluation, "running", "Scoring verified package with 5-criterion LLM-as-a-Judge rubric...", 0)

	var verdict *eval.EvaluationResult
	if fp.evaluator != nil {
		specPrompt := fp.specEngine.GeneratePrompt(specObj)
		v, evalErr := fp.evaluator.EvaluateCode(ctx, specPrompt, harnessRes.FinalOutput, "", eval.DefaultEvaluationCriteria())
		if evalErr == nil {
			verdict = v
			scoresPct := make(map[string]float64, len(v.Criteria))
			for k, val := range v.Criteria {
				scoresPct[k] = val * 100
			}
			rec.Payload["evaluation"] = map[string]interface{}{
				"score":         v.Score,
				"overall_score": v.Score * 100,
				"passed":        v.Passed,
				"criteria":      v.Criteria,
				"scores":        scoresPct,
				"feedback":      v.Feedback,
				"summary":       v.Feedback,
				"improvements":  v.Improvements,
			}
		}
	}

	scorePct := 100
	if verdict != nil {
		scorePct = int(verdict.Score * 100)
	}
	fp.appendTimeline(rec, string(RunStageJudgeEvaluation), "completed", fmt.Sprintf("Judge gate passed (%d%% rubric score)", scorePct), time.Since(judgeStart))

	// Record positive execution outcome in SelfImprovingService & CoordinationSession
	if rec.SessionID != "" {
		_ = fp.sessionRepo.UpdateSessionStatus(rec.SessionID, entities.SessionCompleted)
	}
	if execUUIDStr, ok := rec.Payload["learning_exec_uuid"].(string); ok && execUUIDStr != "" {
		_ = fp.selfImproving.RecordOutcome(ctx, execUUIDStr, true)
	}

	now := time.Now()
	rec.CompletedAt = &now
	_ = FactoryRunStatusMachine.TransitionString(&rec.Status, "completed")
	rec.Payload["promotion_state"] = string(ArtifactPromotionMachine.Initial())
	fp.transition(ctx, rec, RunStageCompleted, "completed", fmt.Sprintf("Deliverable verified & ready (%d files in %s)", len(harnessRes.FilesWritten), harnessRes.WorkspaceDir), time.Since(rec.CreatedAt))
}

// PromoteRun exports the verified Go files from the isolated workspace into
// `<workingDir>/generated/<slug>/` so verified code lands cleanly in the repo.
func (fp *FactoryPipeline) PromoteRun(ctx context.Context, runID, targetSubdir string) (*sqlite.FactoryRunRecord, error) {
	rec, err := fp.runRepo.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	if rec.Status != "completed" || rec.WorkspaceDir == "" {
		return nil, fmt.Errorf("only completed runs with verified workspaces can be promoted")
	}

	slug := sanitizeSlug(targetSubdir)
	if slug == "" {
		slug = sanitizeSlug(rec.SpecID)
	}
	if slug == "" {
		slug = sanitizeSlug(rec.ID)
	}

	destDir := filepath.Join(fp.workingDir, "generated", slug)
	absRoot, err := filepath.Abs(filepath.Join(fp.workingDir, "generated"))
	if err != nil {
		return nil, err
	}
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return nil, err
	}
	if rel, relErr := filepath.Rel(absRoot, absDest); relErr != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("invalid promotion directory")
	}

	promState, _ := rec.Payload["promotion_state"].(string)
	_ = ArtifactPromotionMachine.TransitionString(&promState, "promoting")

	if err := os.MkdirAll(absDest, 0755); err != nil {
		_ = ArtifactPromotionMachine.TransitionString(&promState, "failed")
		rec.Payload["promotion_state"] = promState
		return nil, err
	}

	entries, err := os.ReadDir(rec.WorkspaceDir)
	if err != nil {
		_ = ArtifactPromotionMachine.TransitionString(&promState, "failed")
		rec.Payload["promotion_state"] = promState
		return nil, fmt.Errorf("reading verified workspace: %w", err)
	}
	copiedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "go.mod" || entry.Name() == "go.sum" {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(rec.WorkspaceDir, entry.Name()))
		if readErr != nil {
			_ = ArtifactPromotionMachine.TransitionString(&promState, "failed")
			rec.Payload["promotion_state"] = promState
			return nil, readErr
		}
		if writeErr := os.WriteFile(filepath.Join(absDest, entry.Name()), data, 0644); writeErr != nil {
			_ = ArtifactPromotionMachine.TransitionString(&promState, "failed")
			rec.Payload["promotion_state"] = promState
			return nil, writeErr
		}
		copiedCount++
	}

	_ = ArtifactPromotionMachine.TransitionString(&promState, "promoted")
	rec.Payload["promotion_state"] = promState
	relPromoted := filepath.Join("generated", slug)
	rec.PromotedDir = relPromoted
	fp.appendTimeline(rec, "promoted", "completed", fmt.Sprintf("Promoted %d verified Go files to %s", copiedCount, relPromoted), 0)
	_ = fp.runRepo.Save(ctx, rec)
	fp.broadcast(rec)
	return rec, nil
}

func (fp *FactoryPipeline) resolveOrSynthesizeSpec(ctx context.Context, rec *sqlite.FactoryRunRecord, in FactoryRunInput) (*spec.Specification, error) {
	if in.SpecID != "" {
		all, err := fp.specEngine.LoadAllSpecs()
		if err != nil {
			return nil, err
		}
		if existing, ok := all[in.SpecID]; ok {
			if errs := fp.specEngine.ValidateSpec(existing); len(errs) > 0 {
				return nil, fmt.Errorf("invalid spec %s: %s", in.SpecID, strings.Join(errs, "; "))
			}
			return existing, nil
		}
	}

	// Synthesize a structured Specification from natural-language intent
	specID := in.SpecID
	if specID == "" {
		specID = "spec-" + sanitizeSlug(in.Title)
		if len(specID) > 36 {
			specID = specID[:36]
		}
	}

	prompt := fmt.Sprintf(`Synthesize a machine-readable software specification in JSON format:
Title: %s
Description: %s

Return JSON with "title", "description", "intent" ({"goal", "context", "success_criteria": []}), "constraints" ([{"id", "type": "must_not", "description"}]), and "steps" ([{"id", "name", "description", "depends_on": []}]).`, in.Title, in.Description)

	s := &spec.Specification{
		ID:          specID,
		Version:     "1.0",
		Title:       in.Title,
		Description: in.Description,
		Intent: spec.Intent{
			Goal:    in.Description,
			Context: "Synthesized by ASF Factory Pipeline",
			SuccessCriteria: []string{
				"Compiles cleanly with go build ./...",
				"Passes go test -race -v ./...",
			},
		},
		Constraints: []spec.Constraint{
			{ID: "c1", Type: "must_not", Description: "unsafe.Pointer"},
		},
		Steps: []spec.Step{
			{ID: "design", Name: "Design Types & Invariants", Description: "Define concurrency-safe Go data structures and errors"},
			{ID: "implement", Name: "Implement Core Package", Description: "Implement methods satisfying specification goal", DependsOn: []string{"design"}},
			{ID: "test", Name: "Write Table-Driven & Race Tests", Description: "Verify edge cases and concurrent goroutine safety", DependsOn: []string{"implement"}},
		},
	}
	if s.Intent.Goal == "" {
		s.Intent.Goal = in.Title
	}

	if fp.llmProvider != nil {
		if resp, err := fp.llmProvider.Complete(ctx, &services.LLMRequest{
			Prompt:      prompt,
			MaxTokens:   1500,
			Temperature: 0.2,
			JsonMode:    true,
		}); err == nil {
			var parsed spec.Specification
			if jsonErr := json.Unmarshal([]byte(services.ExtractJSONBlock(resp.Content)), &parsed); jsonErr == nil && parsed.Intent.Goal != "" {
				parsed.ID = specID
				parsed.Version = "1.0"
				if parsed.Title == "" {
					parsed.Title = in.Title
				}
				if len(fp.specEngine.ValidateSpec(&parsed)) == 0 {
					s = &parsed
				}
			}
		}
	}

	for i, c := range in.Constraints {
		if strings.TrimSpace(c) != "" {
			s.Constraints = append(s.Constraints, spec.Constraint{
				ID:          fmt.Sprintf("user-%d", i+1),
				Type:        "must_not",
				Description: strings.TrimSpace(c),
			})
		}
	}

	specsDir := filepath.Join(fp.workingDir, "specs")
	_ = os.MkdirAll(specsDir, 0755)
	if yamlBytes, err := yaml.Marshal(s); err == nil {
		_ = os.WriteFile(filepath.Join(specsDir, specID+".yaml"), yamlBytes, 0644)
	}
	return s, nil
}

func (fp *FactoryPipeline) Subscribe(runID string) (<-chan *sqlite.FactoryRunRecord, func()) {
	ch := make(chan *sqlite.FactoryRunRecord, 16)
	fp.subMu.Lock()
	fp.subscribers[runID] = append(fp.subscribers[runID], ch)
	fp.subMu.Unlock()

	unsubscribe := func() {
		fp.subMu.Lock()
		defer fp.subMu.Unlock()
		subs := fp.subscribers[runID]
		for i, c := range subs {
			if c == ch {
				fp.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	return ch, unsubscribe
}

func (fp *FactoryPipeline) broadcast(rec *sqlite.FactoryRunRecord) {
	fp.subMu.RLock()
	subs := append([]chan *sqlite.FactoryRunRecord(nil), fp.subscribers[rec.ID]...)
	fp.subMu.RUnlock()

	// Marshal + unmarshal payload copy to prevent concurrent map mutation races
	raw, _ := json.Marshal(rec)
	for _, ch := range subs {
		var cp sqlite.FactoryRunRecord
		_ = json.Unmarshal(raw, &cp)
		select {
		case ch <- &cp:
		default:
		}
	}
}

func (fp *FactoryPipeline) transition(ctx context.Context, rec *sqlite.FactoryRunRecord, next state.State, tlStatus, message string, dur time.Duration) {
	if err := FactoryRunMachine.Validate(state.State(rec.Stage), next); err != nil && fp.logger != nil {
		fp.logger.Warn("factory run state transition note", "from", rec.Stage, "to", string(next), "note", err.Error())
	}
	rec.Stage = string(next)
	fp.appendTimeline(rec, string(next), tlStatus, message, dur)
	_ = fp.runRepo.Save(ctx, rec)
	fp.broadcast(rec)
}

func (fp *FactoryPipeline) failRun(ctx context.Context, rec *sqlite.FactoryRunRecord, err error) {
	now := time.Now()
	rec.CompletedAt = &now
	_ = FactoryRunStatusMachine.TransitionString(&rec.Status, "failed")
	rec.Error = err.Error()
	fp.transition(ctx, rec, RunStageFailed, "failed", err.Error(), 0)
}

func (fp *FactoryPipeline) appendTimeline(rec *sqlite.FactoryRunRecord, stage, status, message string, dur time.Duration) {
	entry := StageTimelineEntry{
		Stage:      stage,
		Status:     status,
		Message:    message,
		DurationMs: dur.Milliseconds(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.Marshal(rec.Payload["timeline"])
	var list []StageTimelineEntry
	_ = json.Unmarshal(raw, &list)
	list = append(list, entry)
	rec.Payload["timeline"] = list
}

func readWorkspaceFiles(wsDir string, relPaths []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(relPaths))
	for _, rel := range relPaths {
		data, err := os.ReadFile(filepath.Join(wsDir, rel))
		if err != nil {
			continue
		}
		content := string(data)
		out = append(out, map[string]interface{}{
			"path":    rel,
			"lines":   strings.Count(content, "\n") + 1,
			"content": content,
		})
	}
	return out
}

func classifyRequirement(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "security") || strings.Contains(lower, "hmac") || strings.Contains(lower, "signature") || strings.Contains(lower, "vulnerability"):
		return "security-vulnerability"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "oauth") || strings.Contains(lower, "jwt"):
		return "auth"
	case strings.Contains(lower, "cache") || strings.Contains(lower, "latency") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "performance"):
		return "performance-issue"
	default:
		return "feature"
	}
}

func sanitizeSlug(raw string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
