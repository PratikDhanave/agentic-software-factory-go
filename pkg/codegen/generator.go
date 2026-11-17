package codegen

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type CodeGenerator struct {
	ollama *llm.OllamaClient
}

type Specification struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Language    string                 `json:"language"`
	Constraints []string               `json:"constraints"`
	Examples    []map[string]string    `json:"examples"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type GeneratedCode struct {
	ID          string   `json:"id"`
	Spec        string   `json:"spec_id"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Explanation string   `json:"explanation"`
	Tests       string   `json:"tests"`
	Comments    []string `json:"comments"`
}

func NewCodeGenerator(ollamaURL, model string) *CodeGenerator {
	return &CodeGenerator{
		ollama: llm.NewOllamaClient(ollamaURL, model),
	}
}

// Generate creates code from a specification
func (cg *CodeGenerator) Generate(spec *Specification) (*GeneratedCode, error) {
	log.Printf("[CodeGen] Generating %s code for: %s", spec.Language, spec.Title)

	prompt := cg.buildPrompt(spec)

	response, err := cg.ollama.Analyze(prompt)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	code := &GeneratedCode{
		Spec:     spec.ID,
		Language: spec.Language,
		Code:     cg.extractCode(response),
		Explanation: response,
	}

	log.Printf("[CodeGen] Generated %d lines of code", len(strings.Split(code.Code, "\n")))
	return code, nil
}

// GenerateTests creates tests for the specification
func (cg *CodeGenerator) GenerateTests(spec *Specification, code string) (string, error) {
	log.Printf("[CodeGen] Generating tests for: %s", spec.Title)

	prompt := fmt.Sprintf(`Write comprehensive tests for this code:

Specification: %s
Language: %s

Code:
%s

Requirements:
- Test all constraints from the spec
- Cover edge cases
- Use standard testing framework for %s
- Make tests comprehensive but reasonable

Output only the test code.`, spec.Title, spec.Language, code, spec.Language)

	tests, err := cg.ollama.Analyze(prompt)
	if err != nil {
		return "", err
	}

	return cg.extractCode(tests), nil
}

// Validate checks if generated code meets spec requirements
func (cg *CodeGenerator) Validate(spec *Specification, code string) (bool, string, error) {
	log.Printf("[CodeGen] Validating code against spec: %s", spec.ID)

	constraints := strings.Join(spec.Constraints, "\n- ")
	prompt := fmt.Sprintf(`Review this code against the requirements:

Requirements:
- %s

Code:
%s

Does the code meet ALL requirements?
Respond with JSON:
{
  "valid": true/false,
  "issues": ["issue1", "issue2"],
  "score": 0-100
}`, constraints, code)

	response, err := cg.ollama.Analyze(prompt)
	if err != nil {
		return false, "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return false, response, nil
	}

	valid := result["valid"].(bool)
	return valid, response, nil
}

func (cg *CodeGenerator) buildPrompt(spec *Specification) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Generate %s code for the following specification:\n\n", spec.Language))
	builder.WriteString(fmt.Sprintf("Title: %s\n", spec.Title))
	builder.WriteString(fmt.Sprintf("Description: %s\n\n", spec.Description))

	if len(spec.Constraints) > 0 {
		builder.WriteString("Constraints:\n")
		for _, c := range spec.Constraints {
			builder.WriteString(fmt.Sprintf("- %s\n", c))
		}
		builder.WriteString("\n")
	}

	if len(spec.Examples) > 0 {
		builder.WriteString("Examples:\n")
		for i, ex := range spec.Examples {
			builder.WriteString(fmt.Sprintf("%d. Input: %s → Output: %s\n", i+1, ex["input"], ex["output"]))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("Generate clean, well-structured, production-ready code with comments.")

	return builder.String()
}

func (cg *CodeGenerator) extractCode(response string) string {
	// Look for code blocks
	if strings.Contains(response, "```") {
		parts := strings.Split(response, "```")
		if len(parts) >= 2 {
			code := parts[1]
			// Skip language identifier on first line
			lines := strings.Split(code, "\n")
			if len(lines) > 1 {
				return strings.Join(lines[1:], "\n")
			}
			return code
		}
	}
	return response
}
