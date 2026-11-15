package verify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// Verification pipeline phases. The pipeline advances phase by phase and
// stops at the first failing phase.
const (
	PhaseBuild       state.State = "build"
	PhaseTest        state.State = "test"
	PhaseLint        state.State = "lint"
	PhaseConstraints state.State = "constraints"
	PhasePassed      state.State = "passed"
	PhaseFailed      state.State = "failed"
)

// PhaseMachine is the explicit verification pipeline.
var PhaseMachine = state.NewMachine("verification", PhaseBuild, map[state.State][]state.State{
	PhaseBuild:       {PhaseTest, PhaseFailed},
	PhaseTest:        {PhaseLint, PhaseFailed},
	PhaseLint:        {PhaseConstraints, PhasePassed, PhaseFailed},
	PhaseConstraints: {PhasePassed, PhaseFailed},
})

// VerificationEngine runs AI-TDD verification loops
type VerificationEngine struct {
	WorkingDir     string
	Timeout        time.Duration
	GoTestTimeout  time.Duration
	LintTimeout    time.Duration
	MaxRetries     int
}

// VerificationResult represents the outcome of a verification run
type VerificationResult struct {
	Success         bool
	Phase           state.State // final phase: PhasePassed or PhaseFailed
	FailedPhase     state.State // which phase failed, if any
	TestOutput      string
	LintOutput      string
	BuildOutput     string
	Errors          []VerificationError
	Duration        time.Duration
	RetryCount      int
}

// VerificationError represents a single verification failure
type VerificationError struct {
	Type        string // "test", "lint", "build", "constraint"
	File        string
	Line        int
	Message     string
	Suggestion  string
	Severity    string // "error", "warning"
}

// NewVerificationEngine creates a new verification engine
func NewVerificationEngine(workingDir string) *VerificationEngine {
	return &VerificationEngine{
		WorkingDir:    workingDir,
		Timeout:       5 * time.Minute,
		GoTestTimeout: 2 * time.Minute,
		LintTimeout:   1 * time.Minute,
		MaxRetries:    3,
	}
}

// phaseStep binds a pipeline phase to its implementation.
type phaseStep struct {
	phase state.State
	run   func(context.Context, *VerificationResult) error
}

// pipeline returns the phase sequence for this run. The constraints phase
// only participates when a spec path is provided.
func (e *VerificationEngine) pipeline(specPath string) []phaseStep {
	steps := []phaseStep{
		{PhaseBuild, e.verifyBuild},
		{PhaseTest, e.verifyTests},
		{PhaseLint, e.verifyLint},
	}
	if specPath != "" {
		steps = append(steps, phaseStep{PhaseConstraints, func(ctx context.Context, r *VerificationResult) error {
			return e.verifyConstraints(ctx, specPath, r)
		}})
	}
	return steps
}

// RunVerification runs the complete verification pipeline, stopping at the
// first failing phase. The result records which phase it reached and, on
// failure, which phase failed.
func (e *VerificationEngine) RunVerification(ctx context.Context, specPath string) (*VerificationResult, error) {
	result := &VerificationResult{
		Errors: make([]VerificationError, 0),
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	for _, step := range e.pipeline(specPath) {
		result.Phase = step.phase
		if err := step.run(ctx, result); err != nil {
			result.FailedPhase = step.phase
			result.Phase = PhaseFailed
			return result, fmt.Errorf("%s verification failed: %w", step.phase, err)
		}
	}

	result.Success = len(result.Errors) == 0
	if result.Success {
		result.Phase = PhasePassed
	} else {
		result.Phase = PhaseFailed
	}
	return result, nil
}

// verifyBuild verifies the code compiles
func (e *VerificationEngine) verifyBuild(ctx context.Context, result *VerificationResult) error {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	result.BuildOutput = string(output)

	if err != nil {
		result.Errors = append(result.Errors, e.parseBuildErrors(string(output))...)
		return fmt.Errorf("build failed")
	}
	return nil
}

// verifyTests runs go test
func (e *VerificationEngine) verifyTests(ctx context.Context, result *VerificationResult) error {
	testCtx, cancel := context.WithTimeout(ctx, e.GoTestTimeout)
	defer cancel()

	cmd := exec.CommandContext(testCtx, "go", "test", "-v", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	result.TestOutput = string(output)

	if err != nil {
		result.Errors = append(result.Errors, e.parseTestErrors(string(output))...)
		return fmt.Errorf("tests failed")
	}
	return nil
}

// verifyLint runs golangci-lint
func (e *VerificationEngine) verifyLint(ctx context.Context, result *VerificationResult) error {
	// Check if golangci-lint is available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		// Skip lint if not installed
		result.LintOutput = "golangci-lint not installed, skipping"
		return nil
	}

	lintCtx, cancel := context.WithTimeout(ctx, e.LintTimeout)
	defer cancel()

	cmd := exec.CommandContext(lintCtx, "golangci-lint", "run", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	result.LintOutput = string(output)

	if err != nil {
		result.Errors = append(result.Errors, e.parseLintErrors(string(output))...)
		return fmt.Errorf("lint failed")
	}
	return nil
}

// verifyConstraints verifies against specification constraints
func (e *VerificationEngine) verifyConstraints(ctx context.Context, specPath string, result *VerificationResult) error {
	// Load spec and validate
	// This would integrate with pkg/spec
	return nil
}

// parseBuildErrors extracts errors from go build output
func (e *VerificationEngine) parseBuildErrors(output string) []VerificationError {
	var errors []VerificationError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, ".go:") && (strings.Contains(line, "error") || strings.Contains(line, "Error")) {
			// Parse: file.go:line:col: message
			parts := strings.SplitN(line, ":", 4)
			if len(parts) >= 4 {
				file := parts[0]
				lineNum := 0
				fmt.Sscanf(parts[1], "%d", &lineNum)
				message := strings.TrimSpace(parts[3])

				errors = append(errors, VerificationError{
					Type:     "build",
					File:     file,
					Line:     lineNum,
					Message:  message,
					Severity: "error",
				})
			}
		}
	}
	return errors
}

// parseTestErrors extracts errors from go test output
func (e *VerificationEngine) parseTestErrors(output string) []VerificationError {
	var errors []VerificationError
	lines := strings.Split(output, "\n")

	currentTest := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "=== RUN") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentTest = parts[2]
			}
		}
		if strings.Contains(line, "FAIL") && strings.Contains(line, currentTest) {
			errors = append(errors, VerificationError{
				Type:     "test",
				File:     currentTest,
				Message:  line,
				Severity: "error",
			})
		}
	}
	return errors
}

// parseLintErrors extracts errors from golangci-lint output
func (e *VerificationEngine) parseLintErrors(output string) []VerificationError {
	var errors []VerificationError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, ".go:") && (strings.Contains(line, "error") || strings.Contains(line, "warning")) {
			// Parse: file.go:line:col: message
			parts := strings.SplitN(line, ":", 4)
			if len(parts) >= 4 {
				file := parts[0]
				lineNum := 0
				fmt.Sscanf(parts[1], "%d", &lineNum)
				message := strings.TrimSpace(parts[3])
				severity := "error"
				if strings.Contains(line, "warning") {
					severity = "warning"
				}

				errors = append(errors, VerificationError{
					Type:     "lint",
					File:     file,
					Line:     lineNum,
					Message:  message,
					Severity: severity,
				})
			}
		}
	}
	return errors
}

// RetryWithFeedback retries verification with agent feedback
func (e *VerificationEngine) RetryWithFeedback(ctx context.Context, specPath string, feedback string) (*VerificationResult, error) {
	// In a real implementation, this would feed errors back to the agent
	// and trigger a code generation retry
	return e.RunVerification(ctx, specPath)
}

// GenerateFixPrompt creates a prompt for the agent to fix verification errors
func GenerateFixPrompt(result *VerificationResult, specContent string) string {
	var sb strings.Builder

	sb.WriteString("# Verification Failed - Please Fix\n\n")
	sb.WriteString("The following verification errors were found:\n\n")

	for _, err := range result.Errors {
		sb.WriteString(fmt.Sprintf("## %s: %s\n", strings.ToUpper(err.Type), err.Message))
		if err.File != "" {
			sb.WriteString(fmt.Sprintf("**File**: %s", err.File))
			if err.Line > 0 {
				sb.WriteString(fmt.Sprintf(":%d", err.Line))
			}
			sb.WriteString("\n")
		}
		if err.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("**Suggestion**: %s\n", err.Suggestion))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Original Specification\n\n")
	sb.WriteString(specContent)
	sb.WriteString("\n\nPlease fix the code and ensure all verification passes.\n")

	return sb.String()
}