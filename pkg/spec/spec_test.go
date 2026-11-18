package spec

import (
	"strings"
	"testing"
)

func TestValidateSpecCycleDetectionAndStepNumbering(t *testing.T) {
	engine := NewSpecEngine("")

	cyclic := &Specification{
		ID:    "spec-cyclic",
		Title: "Cyclic Spec",
		Intent: Intent{
			Goal: "Verify cycle detection",
		},
		Steps: []Step{
			{ID: "step1", Name: "First", DependsOn: []string{"step2"}},
			{ID: "step2", Name: "Second", DependsOn: []string{"step1"}},
		},
	}

	errs := engine.ValidateSpec(cyclic)
	if len(errs) == 0 {
		t.Fatal("expected cycle validation error, got none")
	}

	valid := &Specification{
		ID:    "spec-valid",
		Title: "Valid Spec",
		Intent: Intent{
			Goal: "Verify prompt step numbering",
		},
		Steps: []Step{
			{ID: "step1", Name: "First Step"},
			{ID: "step2", Name: "Second Step", DependsOn: []string{"step1"}},
			{ID: "step3", Name: "Third Step", DependsOn: []string{"step2"}},
		},
	}

	if errs := engine.ValidateSpec(valid); len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %v", errs)
	}

	prompt := engine.GeneratePrompt(valid)
	if !strings.Contains(prompt, "1. **First Step**") ||
		!strings.Contains(prompt, "2. **Second Step**") ||
		!strings.Contains(prompt, "3. **Third Step**") {
		t.Fatalf("expected sequential step numbers 1, 2, 3 in prompt, got:\n%s", prompt)
	}
}
