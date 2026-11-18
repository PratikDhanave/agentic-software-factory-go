package harness

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/workspace"
)

// Explicit states for the closed-loop self-healing compiler/test harness.
const (
	HarnessSpecLoaded     state.State = "spec_loaded"
	HarnessWorkspaceReady state.State = "workspace_ready"
	HarnessSynthesizing   state.State = "synthesizing"
	HarnessVerifying      state.State = "verifying"
	HarnessRepairing      state.State = "repairing"
	HarnessSucceeded      state.State = "succeeded"
	HarnessExhausted      state.State = "exhausted"
)

// HarnessMachine declares the AI-TDD self-healing loop inside an isolated Go workspace.
var HarnessMachine = state.NewMachine("harness-self-heal", HarnessSpecLoaded, map[state.State][]state.State{
	HarnessSpecLoaded:     {HarnessWorkspaceReady, HarnessVerifying},
	HarnessWorkspaceReady: {HarnessSynthesizing, HarnessExhausted},
	HarnessSynthesizing:   {HarnessVerifying, HarnessExhausted},
	HarnessVerifying:      {HarnessSucceeded, HarnessRepairing, HarnessExhausted},
	HarnessRepairing:      {HarnessSynthesizing, HarnessExhausted},
}).WithCategory("verification").
	WithDescription("Self-healing code synthesis and compiler/race-test repair loop inside an isolated Go workspace.").
	WithStateDescriptions(map[state.State]string{
		HarnessSpecLoaded:     "Validated specification and repository context loaded.",
		HarnessWorkspaceReady: "Isolated Go module sandbox provisioned under .asf/workspaces/.",
		HarnessSynthesizing:   "Invoking LLM/synthesizer to emit Go source and test files.",
		HarnessVerifying:      "Running go build, go test, go vet, and spec constraints.",
		HarnessRepairing:      "Constructing compiler/test diagnostic repair prompt for next attempt.",
		HarnessSucceeded:      "All verification gates passed cleanly.",
		HarnessExhausted:      "Retry budget exhausted or unrecoverable generation error.",
	})

// CompleteHarness integrates context, specs, isolated workspaces, and self-healing verification.
type CompleteHarness struct {
	contextEngine *asfContext.ContextEngine
	specEngine    *spec.SpecEngine
	verifier      *verify.VerificationEngine
	workspaces    *workspace.WorkspaceManager
	workingDir    string
	maxRetries    int
}

// HarnessResult represents the complete harness execution result.
type HarnessResult struct {
	SpecID        string                     `json:"spec_id"`
	Success       bool                       `json:"success"`
	FinalState    state.State                `json:"final_state,omitempty"`
	StateTrace    []state.State              `json:"state_trace,omitempty"`
	WorkspaceDir  string                     `json:"workspace_dir,omitempty"`
	FilesWritten  []string                   `json:"files_written,omitempty"`
	Diff          map[string]interface{}     `json:"diff,omitempty"`
	ContextItems  []asfContext.ContextItem   `json:"-"`
	SpecPrompt    string                     `json:"spec_prompt,omitempty"`
	Verification  *verify.VerificationResult `json:"verification,omitempty"`
	TotalDuration time.Duration              `json:"total_duration"`
	RetryCount    int                        `json:"retry_count"`
	AgentFeedback string                     `json:"agent_feedback,omitempty"`
	FinalOutput   string                     `json:"final_output"`
}

// AgentClient defines the interface for agent code generation.
type AgentClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// HarnessConfig configures the complete harness.
type HarnessConfig struct {
	WorkingDir          string
	MaxRetries          int
	VerificationTimeout time.Duration
	ContextDirs         []string
}

func DefaultHarnessConfig() HarnessConfig {
	return HarnessConfig{
		WorkingDir:          ".",
		MaxRetries:          3,
		VerificationTimeout: 5 * time.Minute,
		ContextDirs:         []string{"docs", "specs", "internal", "pkg", "cmd"},
	}
}

func NewCompleteHarness(workingDir string) *CompleteHarness {
	return &CompleteHarness{
		contextEngine: asfContext.NewContextEngine(workingDir),
		specEngine:    spec.NewSpecEngine(filepath.Join(workingDir, "specs")),
		verifier:      verify.NewVerificationEngine(workingDir),
		workspaces:    workspace.NewWorkspaceManager(workingDir),
		workingDir:    workingDir,
		maxRetries:    3,
	}
}

func NewHarnessFromConfig(config HarnessConfig) *CompleteHarness {
	h := NewCompleteHarness(config.WorkingDir)
	if config.MaxRetries >= 0 {
		h.maxRetries = config.MaxRetries
	}
	if config.VerificationTimeout > 0 {
		h.verifier.Timeout = config.VerificationTimeout
	}
	return h
}

// Execute runs verification on workingDir against specID (headless verification mode).
func (h *CompleteHarness) Execute(ctx context.Context, specID string) (*HarnessResult, error) {
	result := &HarnessResult{SpecID: specID}
	start := time.Now()
	defer func() { result.TotalDuration = time.Since(start) }()

	s, err := h.loadValidatedSpec(specID)
	if err != nil {
		return nil, err
	}

	contextItems, err := h.contextEngine.LoadContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading context: %w", err)
	}
	result.ContextItems = contextItems

	specPrompt := h.specEngine.GeneratePrompt(s)
	contextStr := asfContext.FormatForAgent(contextItems)
	fullPrompt := fmt.Sprintf("%s\n\n## Project Context\n%s", specPrompt, contextStr)
	result.SpecPrompt = fullPrompt

	verification, _ := h.verifier.RunVerificationForSpec(ctx, "", s)
	result.Verification = verification
	if verification != nil && verification.Success {
		result.Success = true
		result.FinalOutput = "All verification passed"
		return result, nil
	}

	if verification != nil {
		result.AgentFeedback = verify.GenerateFixPrompt(verification, fullPrompt)
	}
	result.Success = false
	result.FinalOutput = "Verification failed"
	return result, nil
}

// ExecuteWithAgent runs the full closed-loop software factory cycle inside an
// isolated workspace: generate -> write files -> build/test/vet/constraints ->
// self-heal with compiler/test feedback up to maxRetries.
func (h *CompleteHarness) ExecuteWithAgent(ctx context.Context, specID string, agentClient AgentClient) (*HarnessResult, error) {
	s, err := h.loadValidatedSpec(specID)
	if err != nil {
		return nil, err
	}
	return h.ExecuteSpecWithAgent(ctx, s, "task-"+uuid.New().String()[:8], agentClient)
}

// ExecuteSpecWithAgent executes an arbitrary Specification inside a fresh
// isolated workspace for taskID.
func (h *CompleteHarness) ExecuteSpecWithAgent(ctx context.Context, s *spec.Specification, taskID string, agentClient AgentClient) (*HarnessResult, error) {
	if s == nil {
		return nil, fmt.Errorf("specification is nil")
	}
	if agentClient == nil {
		return nil, fmt.Errorf("agentClient is nil")
	}

	result := &HarnessResult{
		SpecID:     s.ID,
		FinalState: HarnessSpecLoaded,
		StateTrace: []state.State{HarnessSpecLoaded},
	}
	stepHarness := func(next state.State) {
		if err := HarnessMachine.Transition(&result.FinalState, next); err == nil {
			result.StateTrace = append(result.StateTrace, next)
		}
	}
	start := time.Now()
	defer func() { result.TotalDuration = time.Since(start) }()

	ws, err := h.workspaces.CreateModuleWorkspace(taskID, "asf/generated/"+sanitizeModuleSegment(s.ID))
	if err != nil {
		return nil, fmt.Errorf("provisioning isolated workspace: %w", err)
	}
	result.WorkspaceDir = ws.Dir
	stepHarness(HarnessWorkspaceReady)

	contextItems, _ := h.contextEngine.LoadContext(ctx)
	result.ContextItems = contextItems

	specPrompt := h.specEngine.GeneratePrompt(s)
	contextStr := asfContext.FormatForAgent(contextItems)
	if len(contextStr) > 24000 {
		contextStr = contextStr[:24000] + "\n...[truncated]..."
	}

	artifactInstruction := "\n\n## Output Format Requirement\n" +
		"Emit complete, compilable Go source and unit tests as fenced blocks with explicit file paths:\n" +
		"```go path=solution.go\npackage generated\n...\n```\n" +
		"```go path=solution_test.go\npackage generated\n...\n```\n"

	baseSpecPrompt := specPrompt + artifactInstruction
	prompt := fmt.Sprintf("%s\n\n## Project Context\n%s", baseSpecPrompt, contextStr)
	result.SpecPrompt = prompt

	wsVerifier := h.verifier.WithWorkingDir(ws.Dir)
	var lastGenerated string
	var lastVerification *verify.VerificationResult

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		result.RetryCount = attempt
		stepHarness(HarnessSynthesizing)

		generatedCode, genErr := agentClient.Generate(ctx, prompt)
		if genErr != nil {
			stepHarness(HarnessExhausted)
			return nil, fmt.Errorf("agent generation failed on attempt %d: %w", attempt+1, genErr)
		}
		lastGenerated = generatedCode
		stepHarness(HarnessVerifying)

		artifacts := ParseArtifacts(generatedCode)
		if len(artifacts) == 0 {
			lastVerification = &verify.VerificationResult{
				Success:     false,
				Phase:       verify.PhaseFailed,
				FailedPhase: verify.PhaseBuild,
				Errors: []verify.VerificationError{{
					Type:     "build",
					Message:  "agent response did not contain any Go code artifacts",
					Severity: "error",
				}},
			}
		} else {
			if _, applyErr := ws.ApplyArtifacts(artifacts); applyErr != nil {
				stepHarness(HarnessExhausted)
				return nil, fmt.Errorf("applying generated artifacts: %w", applyErr)
			}
			lastVerification, _ = wsVerifier.RunVerificationForSpec(ctx, "", s)
		}

		result.FilesWritten = ws.FilesWritten
		result.Diff = ws.DiffSummary()
		result.Verification = lastVerification

		if lastVerification != nil && lastVerification.Success {
			stepHarness(HarnessSucceeded)
			result.Success = true
			result.FinalOutput = lastGenerated
			return result, nil
		}

		if attempt < h.maxRetries && lastVerification != nil {
			stepHarness(HarnessRepairing)
			feedback := verify.GenerateFixPrompt(lastVerification, baseSpecPrompt)
			result.AgentFeedback = feedback
			prompt = feedback
		}
	}

	stepHarness(HarnessExhausted)
	result.Success = false
	result.FinalOutput = lastGenerated
	return result, nil
}

func (h *CompleteHarness) loadValidatedSpec(specID string) (*spec.Specification, error) {
	specs, err := h.specEngine.LoadAllSpecs()
	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}
	s, ok := specs[specID]
	if !ok {
		return nil, fmt.Errorf("spec %s not found", specID)
	}
	if errs := h.specEngine.ValidateSpec(s); len(errs) > 0 {
		return nil, fmt.Errorf("invalid spec: %s", strings.Join(errs, "; "))
	}
	return s, nil
}

func sanitizeModuleSegment(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "spec"
	}
	return b.String()
}
