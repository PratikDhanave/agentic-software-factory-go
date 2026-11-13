package security

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SecurityEngine handles input sanitization and sandbox execution
type SecurityEngine struct {
	allowedCommands []string
	blockedPatterns []*regexp.Regexp
	sandboxImage    string
	workDir         string
}

// SandboxConfig configures the execution sandbox
type SandboxConfig struct {
	Image          string            // Docker image to use
	WorkDir        string            // Working directory inside container
	EnvVars        map[string]string // Environment variables
	NetworkMode    string            // "none", "host", "bridge"
	MemoryLimit    string            // e.g., "512m"
	CPULimit       string            // e.g., "0.5"
	Timeout        string            // e.g., "300s"
	ReadOnlyRootFS bool
	DropCapabilities []string
}

// ExecutionResult represents the result of a sandboxed execution
type ExecutionResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
	TimedOut   bool
}

// NewSecurityEngine creates a new security engine
func NewSecurityEngine(workDir string) *SecurityEngine {
	return &SecurityEngine{
		allowedCommands: []string{
			"go", "gofmt", "goimports", "golangci-lint",
			"git", "cat", "ls", "mkdir", "cp", "mv", "rm",
			"find", "grep", "sed", "awk",
		},
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(rm\s+-rf\s+/)`),
			regexp.MustCompile(`(?i)(sudo\s+)`),
			regexp.MustCompile(`(?i)(chmod\s+777)`),
			regexp.MustCompile(`(?i)(curl\s+.*\|\s*sh)`),
			regexp.MustCompile(`(?i)(wget\s+.*\|\s*sh)`),
			regexp.MustCompile(`(?i)(>\s*/dev/sd)`),
			regexp.MustCompile(`(?i)(dd\s+if=)`),
			regexp.MustCompile(`(?i)(mkfs)`),
			regexp.MustCompile(`(?i)(fdisk)`),
			regexp.MustCompile(`(?i)(mount\s+)`),
			regexp.MustCompile(`(?i)(systemctl\s+)`),
			regexp.MustCompile(`(?i)(service\s+)`),
		},
		sandboxImage: "golang:1.23-alpine",
		workDir:      workDir,
	}
}

// SanitizeInput sanitizes agent input for dangerous patterns
func (e *SecurityEngine) SanitizeInput(input string) (string, []string) {
	var violations []string
	sanitized := input

	for _, pattern := range e.blockedPatterns {
		if pattern.MatchString(input) {
			violations = append(violations, fmt.Sprintf("blocked pattern: %s", pattern.String()))
			// Replace with safe placeholder
			sanitized = pattern.ReplaceAllString(sanitized, "[BLOCKED]")
		}
	}

	return sanitized, violations
}

// ValidateCommand validates a command against allowed list
func (e *SecurityEngine) ValidateCommand(command string) (bool, string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, "empty command"
	}

	baseCmd := filepath.Base(parts[0])
	for _, allowed := range e.allowedCommands {
		if baseCmd == allowed {
			return true, ""
		}
	}

	return false, fmt.Sprintf("command not in allowed list: %s", baseCmd)
}

// ExecuteInSandbox executes a command in a Docker sandbox
func (e *SecurityEngine) ExecuteInSandbox(ctx context.Context, config SandboxConfig, command string) (*ExecutionResult, error) {
	// Validate command first
	if ok, err := e.ValidateCommand(command); !ok {
		return nil, fmt.Errorf("command validation failed: %s", err)
	}

	// Build docker run command
	args := []string{"run", "--rm"}

	if config.MemoryLimit != "" {
		args = append(args, "--memory", config.MemoryLimit)
	}
	if config.CPULimit != "" {
		args = append(args, "--cpus", config.CPULimit)
	}
	if config.NetworkMode != "" {
		args = append(args, "--network", config.NetworkMode)
	}
	if config.ReadOnlyRootFS {
		args = append(args, "--read-only")
	}
	for _, cap := range config.DropCapabilities {
		args = append(args, "--cap-drop", cap)
	}
	if config.WorkDir != "" {
		args = append(args, "--workdir", config.WorkDir)
	}
	for k, v := range config.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Mount working directory
	args = append(args, "-v", fmt.Sprintf("%s:%s", e.workDir, config.WorkDir))

	args = append(args, config.Image)
	args = append(args, "sh", "-c", command)

	// Execute with timeout
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	result := &ExecutionResult{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if strings.Contains(err.Error(), "timeout") {
			result.TimedOut = true
		}
	}

	return result, nil
}

// ExecuteGoCommand executes a Go command in the sandbox
func (e *SecurityEngine) ExecuteGoCommand(ctx context.Context, command string) (*ExecutionResult, error) {
	config := SandboxConfig{
		Image:          e.sandboxImage,
		WorkDir:        "/workspace",
		NetworkMode:    "none",
		MemoryLimit:    "1g",
		CPULimit:       "1.0",
		Timeout:        "300s",
		ReadOnlyRootFS: false,
		DropCapabilities: []string{"ALL"},
		EnvVars: map[string]string{
			"GO111MODULE": "on",
			"CGO_ENABLED": "0",
		},
	}

	return e.ExecuteInSandbox(ctx, config, command)
}

// InputValidator validates and sanitizes agent inputs
type InputValidator struct {
	maxInputLength int
	allowedChars   *regexp.Regexp
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		maxInputLength: 100000, // 100KB
		allowedChars:   regexp.MustCompile(`^[[:print:]\n\r\t]*$`),
	}
}

// Validate validates an input string
func (v *InputValidator) Validate(input string) (bool, []string) {
	var errors []string

	if len(input) > v.maxInputLength {
		errors = append(errors, fmt.Sprintf("input exceeds maximum length of %d", v.maxInputLength))
	}

	if !v.allowedChars.MatchString(input) {
		errors = append(errors, "input contains non-printable characters")
	}

	// Check for injection patterns
	injectionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(ignore\s+previous\s+instructions)`),
		regexp.MustCompile(`(?i)(system\s+prompt)`),
		regexp.MustCompile(`(?i)(you\s+are\s+now)`),
		regexp.MustCompile(`(?i)(pretend\s+to\s+be)`),
		regexp.MustCompile(`(?i)(roleplay\s+as)`),
		regexp.MustCompile(`(?i)(##+\s*instructions?)`),
		regexp.MustCompile(`(?i)(##+\s*system)`),
		regexp.MustCompile(`(?i)(##+\s*user)`),
	}

	for _, pattern := range injectionPatterns {
		if pattern.MatchString(input) {
			errors = append(errors, fmt.Sprintf("potential prompt injection detected: %s", pattern.String()))
		}
	}

	return len(errors) == 0, errors
}

// RBACManager manages role-based access control for tool execution
type RBACManager struct {
	roles map[string][]string // role -> allowed tools
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	return &RBACManager{
		roles: map[string][]string{
			"agent": {
				"go", "gofmt", "goimports", "golangci-lint",
				"read_file", "write_file", "list_files",
				"run_test", "run_build",
			},
			"orchestrator": {
				"go", "gofmt", "goimports", "golangci-lint",
				"read_file", "write_file", "list_files",
				"run_test", "run_build",
				"spawn_agent", "manage_context",
			},
			"validator": {
				"go", "gofmt", "goimports", "golangci-lint",
				"read_file", "list_files",
				"run_test", "run_build", "run_lint",
			},
		},
	}
}

// CheckPermission checks if a role has permission to use a tool
func (r *RBACManager) CheckPermission(role, tool string) bool {
	allowedTools, ok := r.roles[role]
	if !ok {
		return false
	}

	for _, t := range allowedTools {
		if t == tool {
			return true
		}
	}
	return false
}

// GetAllowedTools returns the list of allowed tools for a role
func (r *RBACManager) GetAllowedTools(role string) []string {
	return r.roles[role]
}