package harness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
)

// CompleteHarness integrates context, specs, and verification
type CompleteHarness struct {
	contextEngine *context.ContextEngine
	specEngine    *spec.SpecEngine
	verifier      *verify.VerificationEngine
	workingDir    string
	maxRetries    int
}

// HarnessResult represents the complete harness execution result
type HarnessResult struct {
	SpecID          string
	Success         bool
	ContextItems    []context.ContextItem
	SpecPrompt      string
	Verification    *verify.VerificationResult
	TotalDuration   time.Duration
	RetryCount      int
	AgentFeedback   string
	FinalOutput     string
}

// NewCompleteHarness creates a new complete harness
func NewCompleteHarness(workingDir string) *CompleteHarness {
	return &CompleteHarness{
		contextEngine: context.NewContextEngine(workingDir),
		specEngine:    spec.NewSpecEngine(filepath.Join(workingDir, "specs")),
		verifier:      verify.NewVerificationEngine(workingDir),
		workingDir:    workingDir,
		maxRetries:    3,
	}
}

// Execute runs the complete harness for a specification
func (h *CompleteHarness) Execute(ctx context.Context, specID string) (*HarnessResult, error) {
	result := &HarnessResult{
		SpecID: specID,
	}

	start := time.Now()
	defer func() {
		result.TotalDuration = time.Since(start)
	}()

	// Load specification
	specs, err := h.specEngine.LoadAllSpecs()
	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}

	s, ok := specs[specID]
	if !ok {
		return nil, fmt.Errorf("spec %s not found", specID)
	}

	// Validate spec
	if errors := h.specEngine.ValidateSpec(s); len(errors) > 0 {
		return nil, fmt.Errorf("invalid spec: %s", strings.Join(errors, "; "))
	}

	// Load context
	contextItems, err := h.contextEngine.LoadContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading context: %w", err)
	}
	result.ContextItems = contextItems

	// Generate prompt from spec + context
	specPrompt := h.specEngine.GeneratePrompt(s)
	contextStr := context.FormatForAgent(contextItems)
	fullPrompt := fmt.Sprintf("%s\n\n## Project Context\n%s", specPrompt, contextStr)
	result.SpecPrompt = fullPrompt

	// In a real implementation, this prompt would be sent to the agent
	// and the agent would generate code. For now, we assume code is already generated.

	// Run verification loop with retries
	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		result.RetryCount = attempt

		verification, err := h.verifier.RunVerification(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("verification error: %w", err)
		}

		result.Verification = verification

		if verification.Success {
			result.Success = true
			result.FinalOutput = "All verification passed"
			return result, nil
		}

		// Generate feedback for agent retry
		if attempt < h.maxRetries {
			result.AgentFeedback = verify.GenerateFixPrompt(verification, fullPrompt)
			// In real implementation, send feedback to agent and regenerate code
			// For now, we'll just note that a retry would happen
		}
	}

	result.Success = false
	result.FinalOutput = fmt.Sprintf("Verification failed after %d retries", h.maxRetries)
	return result, nil
}

// ExecuteWithAgent runs the harness with an actual agent
// This is a placeholder for integration with an agent client
func (h *CompleteHarness) ExecuteWithAgent(ctx context.Context, specID string, agentClient AgentClient) (*HarnessResult, error) {
	// Load spec and context
	specs, _ := h.specEngine.LoadAllSpecs()
	s := specs[specID]

	contextItems, _ := h.contextEngine.LoadContext(ctx)

	// Generate initial prompt
	specPrompt := h.specEngine.GeneratePrompt(s)
	contextStr := context.FormatForAgent(contextItems)
	prompt := fmt.Sprintf("%s\n\n## Project Context\n%s", specPrompt, contextStr)

	// Retry loop with agent
	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		// Call agent to generate code
		generatedCode, err := agentClient.Generate(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("agent generation failed: %w", err)
		}

		// Write generated code to working directory
		// (In real implementation, this would be more sophisticated)

		// Run verification
		verification, err := h.verifier.RunVerification(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("verification error: %w", err)
		}

		if verification.Success {
			return &HarnessResult{
				SpecID:       specID,
				Success:      true,
				Verification: verification,
				FinalOutput:  generatedCode,
			}, nil
		}

		// Generate feedback for next attempt
		prompt = verify.GenerateFixPrompt(verification, prompt)
	}

	return &HarnessResult{
		SpecID:    specID,
		Success:   false,
		FinalOutput: "Max retries exceeded",
	}, nil
}

// AgentClient defines the interface for agent code generation
type AgentClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// HarnessConfig configures the complete harness
type HarnessConfig struct {
	WorkingDir       string
	MaxRetries       int
	VerificationTimeout time.Duration
	ContextDirs      []string
}

// DefaultHarnessConfig returns a default configuration
func DefaultHarnessConfig() HarnessConfig {
	return HarnessConfig{
		WorkingDir:        ".",
		MaxRetries:        3,
		VerificationTimeout: 5 * time.Minute,
		ContextDirs:       []string{"docs", "specs", "internal", "pkg", "cmd"},
	}
}

// NewHarnessFromConfig creates a harness from configuration
func NewHarnessFromConfig(config HarnessConfig) *CompleteHarness {
	h := NewCompleteHarness(config.WorkingDir)
	h.maxRetries = config.MaxRetries
	h.verifier.timeout = config.VerificationTimeout
	return h
}