package verify

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
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
}).WithCategory("verification").
	WithDescription("Sequential Go compiler, unit/race test, static analysis (go vet), and spec constraint gate.").
	WithStateDescriptions(map[state.State]string{
		PhaseBuild:       "Compiling Go packages with go build -buildvcs=false ./...",
		PhaseTest:        "Running unit & concurrency tests with go test -v ./...",
		PhaseLint:        "Running static analysis with go vet ./...",
		PhaseConstraints: "Checking MUST and MUST_NOT specification invariants.",
		PhasePassed:      "All verification phases passed cleanly.",
		PhaseFailed:      "Verification halted on the first failing gate with structured diagnostics.",
	})

// VerificationEngine runs AI-TDD verification loops against a target workspace directory.
type VerificationEngine struct {
	WorkingDir    string
	Timeout       time.Duration
	GoTestTimeout time.Duration
	LintTimeout   time.Duration
	MaxRetries    int
}

// VerificationResult represents the outcome of a verification run.
type VerificationResult struct {
	Success     bool                `json:"success"`
	Phase       state.State         `json:"phase"`        // final phase: PhasePassed or PhaseFailed
	FailedPhase state.State         `json:"failed_phase"` // which phase failed, if any
	TestOutput  string              `json:"test_output"`
	LintOutput  string              `json:"lint_output"`
	BuildOutput string              `json:"build_output"`
	Errors      []VerificationError `json:"errors"`
	Duration    time.Duration       `json:"duration"`
	RetryCount  int                 `json:"retry_count"`
}

// VerificationError represents a single verification failure.
type VerificationError struct {
	Type       string `json:"type"` // "test", "lint", "build", "constraint"
	File       string `json:"file"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Severity   string `json:"severity"` // "error", "warning"
}

// NewVerificationEngine creates a new verification engine.
func NewVerificationEngine(workingDir string) *VerificationEngine {
	return &VerificationEngine{
		WorkingDir:    workingDir,
		Timeout:       5 * time.Minute,
		GoTestTimeout: 2 * time.Minute,
		LintTimeout:   1 * time.Minute,
		MaxRetries:    3,
	}
}

// WithWorkingDir returns a shallow copy targeting dir so concurrent workers
// can verify isolated workspaces independently.
func (e *VerificationEngine) WithWorkingDir(dir string) *VerificationEngine {
	cp := *e
	cp.WorkingDir = dir
	return &cp
}

type phaseStep struct {
	phase state.State
	run   func(context.Context, *VerificationResult) error
}

func (e *VerificationEngine) pipeline(specPath string, specObj *spec.Specification) []phaseStep {
	steps := []phaseStep{
		{PhaseBuild, e.verifyBuild},
		{PhaseTest, e.verifyTests},
		{PhaseLint, e.verifyLint},
	}
	if specPath != "" || specObj != nil {
		steps = append(steps, phaseStep{PhaseConstraints, func(ctx context.Context, r *VerificationResult) error {
			return e.verifyConstraints(ctx, specPath, specObj, r)
		}})
	}
	return steps
}

// RunVerification runs the complete verification pipeline on e.WorkingDir.
func (e *VerificationEngine) RunVerification(ctx context.Context, specPath string) (*VerificationResult, error) {
	return e.RunVerificationForSpec(ctx, specPath, nil)
}

// RunVerificationForSpec runs build -> test -> lint/vet -> spec constraints.
func (e *VerificationEngine) RunVerificationForSpec(ctx context.Context, specPath string, specObj *spec.Specification) (*VerificationResult, error) {
	result := &VerificationResult{
		Errors: make([]VerificationError, 0),
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	runCtx := ctx
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	for _, step := range e.pipeline(specPath, specObj) {
		result.Phase = step.phase
		if err := step.run(runCtx, result); err != nil {
			result.FailedPhase = step.phase
			result.Phase = PhaseFailed
			result.Success = false
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

func (e *VerificationEngine) verifyBuild(ctx context.Context, result *VerificationResult) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	result.BuildOutput = string(output)

	if err != nil {
		parsed := e.parseBuildErrors(string(output))
		if len(parsed) == 0 {
			parsed = []VerificationError{{
				Type:     "build",
				Message:  strings.TrimSpace(string(output)),
				Severity: "error",
			}}
		}
		result.Errors = append(result.Errors, parsed...)
		return fmt.Errorf("build failed")
	}
	return nil
}

func (e *VerificationEngine) verifyTests(ctx context.Context, result *VerificationResult) error {
	timeout := e.GoTestTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(testCtx, "go", "test", "-buildvcs=false", "-v", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	result.TestOutput = string(output)

	if err != nil {
		parsed := e.parseTestErrors(string(output))
		if len(parsed) == 0 {
			parsed = e.parseBuildErrors(string(output))
		}
		if len(parsed) == 0 {
			parsed = []VerificationError{{
				Type:     "test",
				Message:  strings.TrimSpace(string(output)),
				Severity: "error",
			}}
		}
		result.Errors = append(result.Errors, parsed...)
		return fmt.Errorf("tests failed")
	}
	return nil
}

func (e *VerificationEngine) verifyLint(ctx context.Context, result *VerificationResult) error {
	timeout := e.LintTimeout
	if timeout <= 0 {
		timeout = time.Minute
	}
	lintCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if strings.EqualFold(os.Getenv("ASF_STRICT_LINT"), "true") {
		if _, err := exec.LookPath("golangci-lint"); err == nil {
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
	}

	// Run standard `go vet` for fast, reliable static analysis
	cmd := exec.CommandContext(lintCtx, "go", "vet", "-buildvcs=false", "./...")
	cmd.Dir = e.WorkingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.LintOutput = string(output)
		parsed := e.parseBuildErrors(string(output))
		for i := range parsed {
			parsed[i].Type = "lint"
		}
		if len(parsed) == 0 {
			parsed = []VerificationError{{
				Type:     "lint",
				Message:  strings.TrimSpace(string(output)),
				Severity: "error",
			}}
		}
		result.Errors = append(result.Errors, parsed...)
		return fmt.Errorf("go vet failed")
	}
	result.LintOutput = "go vet passed"
	return nil
}

func (e *VerificationEngine) verifyConstraints(ctx context.Context, specPath string, specObj *spec.Specification, result *VerificationResult) error {
	s := specObj
	if s == nil && specPath != "" {
		engine := spec.NewSpecEngine(filepath.Dir(specPath))
		loaded, err := engine.LoadSpec(specPath)
		if err == nil {
			s = loaded
		}
	}
	if s == nil {
		return nil
	}

	var combined strings.Builder
	_ = filepath.WalkDir(e.WorkingDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && (d.Name() == ".git" || d.Name() == ".asf" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			if b, readErr := os.ReadFile(path); readErr == nil {
				combined.Write(b)
				combined.WriteByte('\n')
			}
		}
		return nil
	})

	validator := spec.NewSpecValidator(s)
	violations := validator.ValidateConstraints(combined.String())
	for _, v := range violations {
		result.Errors = append(result.Errors, VerificationError{
			Type:     "constraint",
			Message:  v,
			Severity: "error",
		})
	}
	if len(violations) > 0 {
		return fmt.Errorf("%d specification constraint violation(s)", len(violations))
	}
	return nil
}

// parseBuildErrors extracts Go compiler/vet errors of form `file.go:line:col: message`.
func (e *VerificationEngine) parseBuildErrors(output string) []VerificationError {
	var errors []VerificationError
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ".go:")
		if idx < 0 {
			continue
		}
		file := line[:idx+3]
		rest := line[idx+4:]
		parts := strings.SplitN(rest, ":", 3)
		if len(parts) < 2 {
			continue
		}
		lineNum := 0
		_, _ = fmt.Sscanf(parts[0], "%d", &lineNum)
		msg := strings.TrimSpace(parts[len(parts)-1])
		if msg == "" {
			continue
		}
		errors = append(errors, VerificationError{
			Type:     "build",
			File:     file,
			Line:     lineNum,
			Message:  msg,
			Severity: "error",
		})
	}
	return errors
}

// parseTestErrors extracts failing test cases and assertion lines from `go test -v`.
func (e *VerificationEngine) parseTestErrors(output string) []VerificationError {
	var errors []VerificationError
	currentTest := ""
	var recentDetail string

	for _, rawLine := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "=== RUN") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				currentTest = parts[2]
				recentDetail = ""
			}
			continue
		}
		if strings.Contains(trimmed, "_test.go:") {
			recentDetail = trimmed
		}
		if strings.HasPrefix(trimmed, "--- FAIL:") {
			msg := trimmed
			if recentDetail != "" {
				msg = trimmed + " (" + recentDetail + ")"
			}
			errors = append(errors, VerificationError{
				Type:     "test",
				File:     currentTest,
				Message:  msg,
				Severity: "error",
			})
		}
	}
	return errors
}

func (e *VerificationEngine) parseLintErrors(output string) []VerificationError {
	parsed := e.parseBuildErrors(output)
	for i := range parsed {
		parsed[i].Type = "lint"
	}
	return parsed
}

func (e *VerificationEngine) RetryWithFeedback(ctx context.Context, specPath string, feedback string) (*VerificationResult, error) {
	return e.RunVerification(ctx, specPath)
}

// GenerateFixPrompt creates a structured self-healing prompt from compiler, test,
// or constraint failures so the agent can repair its previous output.
func GenerateFixPrompt(result *VerificationResult, specContent string) string {
	var sb strings.Builder

	sb.WriteString("# Verification Failed - Self-Healing Repair Required\n\n")
	sb.WriteString("Your previous code failed automated verification. Fix EVERY error below and output the complete corrected Go file(s) inside fenced ```go path=<filename.go> blocks.\n\n")

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

	if result.BuildOutput != "" && result.FailedPhase == PhaseBuild {
		sb.WriteString("## Raw Compiler Output\n```\n")
		sb.WriteString(strings.TrimSpace(result.BuildOutput))
		sb.WriteString("\n```\n\n")
	} else if result.TestOutput != "" && result.FailedPhase == PhaseTest {
		sb.WriteString("## Raw Test Output\n```\n")
		sb.WriteString(strings.TrimSpace(result.TestOutput))
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Original Specification\n\n")
	sb.WriteString(specContent)
	sb.WriteString("\n\nReturn complete, compilable Go code and passing unit tests.\n")

	return sb.String()
}
