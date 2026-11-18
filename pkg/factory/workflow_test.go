package factory

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/codegen"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

func TestSpecToShip(t *testing.T) {
	engine := NewWorkflowEngineWithClient(llm.NewDeterministicClient())

	spec := &codegen.Specification{
		ID:          "test-1",
		Title:       "Simple Function",
		Description: "Create a function that adds two numbers",
		Language:    "go",
		Constraints: []string{
			"Must handle edge cases",
			"Include error handling",
		},
	}

	result, err := engine.SpecToShip(spec)
	if err != nil {
		t.Fatalf("SpecToShip failed: %v", err)
	}

	if result.Status == "failed" {
		t.Fatalf("Workflow failed: %v", result)
	}

	if result.GeneratedCode == "" {
		t.Fatal("No code generated")
	}

	if result.Tests == "" {
		t.Fatal("No tests generated")
	}
}

func TestCodeReview(t *testing.T) {
	engine := NewWorkflowEngineWithClient(llm.NewDeterministicClient())

	code := `func Add(a, b int) int { return a + b }`

	result, err := engine.ReviewAndFix(code, "go")
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	if result.ReviewResult == nil {
		t.Fatal("No review result")
	}

	if result.ReviewResult.Score < 0 || result.ReviewResult.Score > 100 {
		t.Fatalf("Invalid score: %d", result.ReviewResult.Score)
	}
}
