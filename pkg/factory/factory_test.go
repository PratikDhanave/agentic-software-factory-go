package factory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
)

// scriptedAgent simulates a model whose first artifact fails the judge and
// whose revision passes. Judge calls are recognized by the evaluation
// template's fixed preamble.
type scriptedAgent struct {
	calls int
}

func (s *scriptedAgent) Generate(_ context.Context, prompt string) (string, error) {
	s.calls++
	if strings.Contains(prompt, "evaluating AI-generated code") {
		if strings.Contains(prompt, "attempt-2 artifact") {
			return `{"score": 0.9, "criteria": {}, "feedback": "solid", "improvements": []}`, nil
		}
		return `{"score": 0.4, "criteria": {}, "feedback": "missing error handling", "improvements": ["add error handling"]}`, nil
	}
	if strings.Contains(prompt, "Judge Feedback") {
		return "attempt-2 artifact", nil
	}
	return "attempt-1 artifact", nil
}

func newTestFactory(t *testing.T) (*FactoryOrchestrator, *scriptedAgent) {
	t.Helper()
	// Isolate from any real credentials so NewFactoryOrchestrator starts in
	// prompt-only mode; the stub is injected afterwards.
	t.Setenv("ASF_LLM_PROVIDER", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_CONFIG_DIR", t.TempDir())

	dir := t.TempDir()
	specYAML := `id: "spec-001"
version: "1.0"
title: "Test Spec"
description: "A test specification"
intent:
  goal: "Produce a test artifact"
  context: "Testing"
  success_criteria:
    - "Artifact exists"
`
	if err := os.MkdirAll(filepath.Join(dir, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "specs", "spec-001.yaml"), []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	fo, err := NewFactoryOrchestrator(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if fo.agent != nil {
		t.Fatal("expected prompt-only factory before stub injection")
	}

	stub := &scriptedAgent{}
	fo.agent = stub
	fo.agentModel = "scripted-test-model"
	fo.agentProvider = "test"
	fo.llmSlots = make(chan struct{}, 2)
	fo.evaluator = eval.NewEvaluationEngine(stub)
	fo.evalCriteria = eval.DefaultEvaluationCriteria() // threshold 0.75
	return fo, stub
}

func TestGenerateJudgeLoopRevisesUntilPassing(t *testing.T) {
	fo, stub := newTestFactory(t)
	task := &Task{ID: "t1", Type: "generate", SpecID: "spec-001", Status: TaskRunning}

	if err := fo.executeGenerate(context.Background(), task); err != nil {
		t.Fatalf("executeGenerate: %v", err)
	}

	result := task.Result.(map[string]interface{})
	if result["output"] != "attempt-2 artifact" {
		t.Errorf("expected revised artifact, got %q", result["output"])
	}
	if result["attempts"] != 2 {
		t.Errorf("expected 2 attempts, got %v", result["attempts"])
	}
	verdict := result["evaluation"].(map[string]interface{})
	if verdict["passed"] != true {
		t.Errorf("expected final verdict to pass, got %+v", verdict)
	}
	// generate + judge + revised generate + judge
	if stub.calls != 4 {
		t.Errorf("expected 4 LLM calls, got %d", stub.calls)
	}
}

func TestEvaluateTaskJudgesLatestGenerateOutput(t *testing.T) {
	fo, _ := newTestFactory(t)

	gen := &Task{ID: "gen-1", Type: "generate", SpecID: "spec-001", Status: TaskRunning}
	if err := fo.executeGenerate(context.Background(), gen); err != nil {
		t.Fatalf("executeGenerate: %v", err)
	}
	setTaskState(gen, TaskCompleted)
	fo.results[gen.ID] = gen

	evalTask := &Task{ID: "eval-1", Type: "evaluate", SpecID: "spec-001", Status: TaskRunning}
	if err := fo.executeEvaluate(context.Background(), evalTask); err != nil {
		t.Fatalf("executeEvaluate: %v", err)
	}

	result := evalTask.Result.(map[string]interface{})
	if result["evaluated_task"] != "gen-1" {
		t.Errorf("expected evaluation of gen-1, got %v", result["evaluated_task"])
	}
	verdict := result["evaluation"].(map[string]interface{})
	if verdict["passed"] != true {
		t.Errorf("expected passing verdict, got %+v", verdict)
	}
}

func TestEvaluateTaskWithoutGenerateOutputFails(t *testing.T) {
	fo, _ := newTestFactory(t)
	task := &Task{ID: "eval-2", Type: "evaluate", SpecID: "spec-001", Status: TaskRunning}
	err := fo.executeEvaluate(context.Background(), task)
	if err == nil || !strings.Contains(err.Error(), "no completed generate output") {
		t.Errorf("expected missing-output error, got %v", err)
	}
}
