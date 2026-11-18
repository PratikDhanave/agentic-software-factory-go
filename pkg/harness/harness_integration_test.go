package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
)

type scriptedAgent struct {
	mu      sync.Mutex
	prompts []string
	outputs []string
}

func (a *scriptedAgent) Generate(ctx context.Context, prompt string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prompts = append(a.prompts, prompt)
	idx := len(a.prompts) - 1
	if idx >= len(a.outputs) {
		idx = len(a.outputs) - 1
	}
	return a.outputs[idx], nil
}

func TestSelfHealingHarnessRecoversFromCompileFailure(t *testing.T) {
	workDir := t.TempDir()
	specsDir := filepath.Join(workDir, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		t.Fatal(err)
	}

	specYAML := `id: "spec-calc"
version: "1.0"
title: "Safe Integer Adder"
description: "Implement Add(a, b int) int with unit tests"
intent:
  goal: "Provide a tested Add function"
  context: "Standalone arithmetic utility"
  success_criteria:
    - "Compiles cleanly and passes go test"
constraints:
  - id: "c1"
    type: "must_not"
    description: "unsafe.Pointer"
steps:
  - id: "s1"
    name: "Implement Add"
    description: "Write solution.go and solution_test.go"
`
	if err := os.WriteFile(filepath.Join(specsDir, "calc.yaml"), []byte(specYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Attempt 1 intentionally references an undefined symbol `missingHelper` so `go build` fails.
	brokenAttempt := "```go path=solution.go\npackage calc\n\nfunc Add(a, b int) int {\n\treturn missingHelper(a, b)\n}\n```\n" +
		"```go path=solution_test.go\npackage calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"wrong sum\")\n\t}\n}\n```"

	// Attempt 2 fixes the compiler error after receiving the structured compiler error feedback.
	fixedAttempt := "```go path=solution.go\npackage calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n```\n" +
		"```go path=solution_test.go\npackage calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif got := Add(2, 3); got != 5 {\n\t\tt.Fatalf(\"got %d, want 5\", got)\n\t}\n}\n```"

	agent := &scriptedAgent{outputs: []string{brokenAttempt, fixedAttempt}}

	h := NewCompleteHarness(workDir)
	res, err := h.ExecuteWithAgent(context.Background(), "spec-calc", agent)
	if err != nil {
		t.Fatalf("ExecuteWithAgent error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected self-healing harness to succeed on retry, got result: %+v (verification: %+v)", res, res.Verification)
	}
	if res.RetryCount != 1 {
		t.Fatalf("expected RetryCount=1, got %d", res.RetryCount)
	}
	if len(res.FilesWritten) != 2 {
		t.Fatalf("expected 2 files written, got %v", res.FilesWritten)
	}
	if len(agent.prompts) != 2 || !strings.Contains(agent.prompts[1], "missingHelper") {
		t.Fatalf("expected retry prompt to include compiler error 'missingHelper', got:\n%s", agent.prompts[len(agent.prompts)-1])
	}
}

func TestWorkspacePathTraversalRejected(t *testing.T) {
	workDir := t.TempDir()
	h := NewCompleteHarness(workDir)
	s := &spec.Specification{
		ID:    "spec-escape",
		Title: "Traversal Check",
		Intent: spec.Intent{
			Goal: "Ensure path traversal is rejected",
		},
	}

	malicious := `{"files":[{"path":"../../escaped.go","action":"create","content":"package p"}]}`
	agent := &scriptedAgent{outputs: []string{malicious}}

	_, err := h.ExecuteSpecWithAgent(context.Background(), s, "task-sec", agent)
	if err == nil || !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("expected path traversal rejection error, got: %v", err)
	}
}
