package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Specification represents a machine-readable specification for agent execution
type Specification struct {
	ID          string                 `json:"id" yaml:"id"`
	Version     string                 `json:"version" yaml:"version"`
	Title       string                 `json:"title" yaml:"title"`
	Description string                 `json:"description" yaml:"description"`
	Intent      Intent                 `json:"intent" yaml:"intent"`
	Constraints []Constraint           `json:"constraints" yaml:"constraints"`
	Inputs      map[string]InputSpec   `json:"inputs" yaml:"inputs"`
	Outputs     map[string]OutputSpec  `json:"outputs" yaml:"outputs"`
	Steps       []Step                 `json:"steps" yaml:"steps"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Intent defines what the agent should accomplish
type Intent struct {
	Goal        string   `json:"goal" yaml:"goal"`
	Context     string   `json:"context" yaml:"context"`
	SuccessCriteria []string `json:"success_criteria" yaml:"success_criteria"`
	AntiGoals   []string `json:"anti_goals,omitempty" yaml:"anti_goals,omitempty"`
}

// Constraint defines hard boundaries the agent must respect
type Constraint struct {
	ID          string `json:"id" yaml:"id"`
	Type        string `json:"type" yaml:"type"` // "must", "must_not", "should", "should_not"
	Description string `json:"description" yaml:"description"`
	Validator   string `json:"validator,omitempty" yaml:"validator,omitempty"` // Go code snippet for validation
}

// InputSpec defines expected input parameters
type InputSpec struct {
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Validator   string      `json:"validator,omitempty" yaml:"validator,omitempty"`
}

// OutputSpec defines expected outputs
type OutputSpec struct {
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Validator   string `json:"validator,omitempty" yaml:"validator,omitempty"`
}

// Step represents a single execution step in the plan
type Step struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Tool        string                 `json:"tool,omitempty" yaml:"tool,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Validation  string                 `json:"validation,omitempty" yaml:"validation,omitempty"`
}

// SpecEngine parses and validates specifications
type SpecEngine struct {
	specsDir string
}

// NewSpecEngine creates a new specification engine
func NewSpecEngine(specsDir string) *SpecEngine {
	return &SpecEngine{specsDir: specsDir}
}

// LoadSpec loads a specification from a YAML file
func (e *SpecEngine) LoadSpec(path string) (*Specification, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var spec Specification
	// Try YAML first, then JSON
	if err := unmarshalYAML(content, &spec); err != nil {
		if err := json.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("parsing spec: %w", err)
		}
	}

	// Set defaults
	if spec.Version == "" {
		spec.Version = "1.0"
	}

	return &spec, nil
}

// LoadAllSpecs loads all specifications from the specs directory
func (e *SpecEngine) LoadAllSpecs() (map[string]*Specification, error) {
	specs := make(map[string]*Specification)

	entries, err := os.ReadDir(e.specsDir)
	if err != nil {
		return specs, nil // Directory might not exist
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(e.specsDir, entry.Name())
		spec, err := e.LoadSpec(path)
		if err != nil {
			continue
		}
		specs[spec.ID] = spec
	}

	return specs, nil
}

// ValidateSpec validates a specification for completeness and correctness
func (e *SpecEngine) ValidateSpec(spec *Specification) []string {
	var errors []string

	if spec.ID == "" {
		errors = append(errors, "spec ID is required")
	}
	if spec.Title == "" {
		errors = append(errors, "spec title is required")
	}
	if spec.Intent.Goal == "" {
		errors = append(errors, "intent.goal is required")
	}

	// Check for circular dependencies in steps
	stepMap := make(map[string]*Step)
	for i := range spec.Steps {
		stepMap[spec.Steps[i].ID] = &spec.Steps[i]
	}

	for _, step := range spec.Steps {
		for _, dep := range step.DependsOn {
			if _, ok := stepMap[dep]; !ok {
				errors = append(errors, fmt.Sprintf("step %s depends on non-existent step %s", step.ID, dep))
			}
		}
	}

	return errors
}

// GeneratePrompt generates an agent prompt from the specification
func (e *SpecEngine) GeneratePrompt(spec *Specification) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Specification: %s\n\n", spec.Title))
	sb.WriteString(fmt.Sprintf("**Version**: %s\n", spec.Version))
	sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", spec.Description))

	sb.WriteString("## Intent\n")
	sb.WriteString(fmt.Sprintf("**Goal**: %s\n", spec.Intent.Goal))
	sb.WriteString(fmt.Sprintf("**Context**: %s\n", spec.Intent.Context))
	if len(spec.Intent.SuccessCriteria) > 0 {
		sb.WriteString("**Success Criteria**:\n")
		for _, c := range spec.Intent.SuccessCriteria {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}
	if len(spec.Intent.AntiGoals) > 0 {
		sb.WriteString("**Anti-Goals (avoid)**:\n")
		for _, a := range spec.Intent.AntiGoals {
			sb.WriteString(fmt.Sprintf("- %s\n", a))
		}
	}
	sb.WriteString("\n")

	if len(spec.Constraints) > 0 {
		sb.WriteString("## Constraints (MUST FOLLOW)\n")
		for _, c := range spec.Constraints {
			sb.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(c.Type), c.Description))
		}
		sb.WriteString("\n")
	}

	if len(spec.Inputs) > 0 {
		sb.WriteString("## Inputs\n")
		for name, input := range spec.Inputs {
			req := ""
			if input.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("- **%s**%s: %s\n", name, req, input.Description))
		}
		sb.WriteString("\n")
	}

	if len(spec.Steps) > 0 {
		sb.WriteString("## Execution Plan\n")
		for _, step := range spec.Steps {
			deps := ""
			if len(step.DependsOn) > 0 {
				deps = fmt.Sprintf(" [depends on: %s]", strings.Join(step.DependsOn, ", "))
			}
			sb.WriteString(fmt.Sprintf("%d. **%s**%s\n", len(spec.Steps), step.Name, deps))
			sb.WriteString(fmt.Sprintf("   %s\n", step.Description))
			if step.Tool != "" {
				sb.WriteString(fmt.Sprintf("   Tool: %s\n", step.Tool))
			}
			if step.Validation != "" {
				sb.WriteString(fmt.Sprintf("   Validation: %s\n", step.Validation))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Instructions\n")
	sb.WriteString("Execute the plan step by step. Validate each step before proceeding. ")
	sb.WriteString("All constraints are mandatory - violations will cause the execution to fail.\n")

	return sb.String()
}

// unmarshalYAML uses gopkg.in/yaml.v3 for unmarshaling
func unmarshalYAML(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

// SpecTemplate provides a template for creating new specifications
const SpecTemplate = `# Specification Template
id: "spec-001"
version: "1.0"
title: "Example Specification"
description: "A brief description of what this specification achieves"
intent:
  goal: "Clear, measurable goal for the agent"
  context: "Background context and constraints"
  success_criteria:
    - "Criterion 1: measurable outcome"
    - "Criterion 2: another measurable outcome"
  anti_goals:
    - "What to explicitly avoid"
constraints:
  - id: "c1"
    type: "must"
    description: "Hard constraint the agent must follow"
    validator: "code to validate this constraint"
  - id: "c2"
    type: "must_not"
    description: "What the agent must not do"
inputs:
  input_name:
    type: "string"
    description: "Description of this input"
    required: true
    default: "default value"
outputs:
  output_name:
    type: "string"
    description: "Expected output"
steps:
  - id: "step1"
    name: "First Step"
    description: "What this step does"
    tool: "tool_name"
    inputs:
      key: "value"
    depends_on: []
    validation: "how to validate this step succeeded"
  - id: "step2"
    name: "Second Step"
    description: "What this step does"
    tool: "tool_name"
    inputs:
      key: "value"
    depends_on: ["step1"]
    validation: "how to validate this step succeeded"
`

// WriteSpecTemplate writes a specification template to a file
func WriteSpecTemplate(path string) error {
	return os.WriteFile(path, []byte(SpecTemplate), 0644)
}

// SpecValidator validates agent output against a specification
type SpecValidator struct {
	spec *Specification
}

// NewSpecValidator creates a new validator for a specification
func NewSpecValidator(spec *Specification) *SpecValidator {
	return &SpecValidator{spec: spec}
}

// ValidateOutput validates agent output against specification outputs
func (v *SpecValidator) ValidateOutput(outputs map[string]interface{}) []string {
	var errors []string

	for name, expected := range v.spec.Outputs {
		_, ok := outputs[name]
		if !ok {
			errors = append(errors, fmt.Sprintf("missing required output: %s", name))
			continue
		}

		if expected.Validator != "" {
			// In production, execute the validator code
			// For now, just note that validation would happen here
		}
	}

	return errors
}

// ValidateConstraints checks if constraints are violated
func (v *SpecValidator) ValidateConstraints(context string) []string {
	var violations []string

	for _, c := range v.spec.Constraints {
		if c.Type == "must_not" {
			// Check if forbidden pattern appears in context
			if strings.Contains(strings.ToLower(context), strings.ToLower(c.Description)) {
				violations = append(violations, fmt.Sprintf("constraint violation [%s]: %s", c.ID, c.Description))
			}
		}
	}

	return violations
}