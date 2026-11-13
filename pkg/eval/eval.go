package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// EvaluationEngine runs LLM-as-a-judge evaluations on agent outputs
type EvaluationEngine struct {
	client LLMClient
}

// LLMClient is an interface for calling an LLM
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// EvaluationResult represents the result of an evaluation
type EvaluationResult struct {
	Score        float64            // 0.0 to 1.0
	Passed       bool               // Whether it meets the threshold
	Criteria     map[string]float64 // Per-criterion scores
	Feedback     string             // Natural language feedback
	Improvements []string           // Specific improvement suggestions
}

// EvaluationCriteria defines the criteria for evaluation
type EvaluationCriteria struct {
	Correctness     float64 // Weight for functional correctness
	CodeQuality     float64 // Weight for code quality/style
	SpecCompliance  float64 // Weight for spec adherence
	Security        float64 // Weight for security practices
	Documentation   float64 // Weight for documentation
	Threshold       float64 // Minimum score to pass (default 0.75)
}

// DefaultEvaluationCriteria returns default evaluation criteria
func DefaultEvaluationCriteria() EvaluationCriteria {
	return EvaluationCriteria{
		Correctness:    0.35,
		CodeQuality:    0.25,
		SpecCompliance: 0.20,
		Security:       0.15,
		Documentation:  0.05,
		Threshold:      0.75,
	}
}

// EvaluationPromptTemplate is the template for LLM-as-a-judge evaluation
const EvaluationPromptTemplate = `You are an expert software engineer evaluating AI-generated code. Score the following code against the specification.

## Specification
{{.spec}}

## Generated Code
{{.code}}

## Context (constraints, ADRs, patterns)
{{.context}}

## Evaluation Criteria
1. **Correctness ({{.weights.correctness}}%)**: Does the code implement the spec correctly? Handle edge cases? Pass tests?
2. **Code Quality ({{.weights.code_quality}}%)**: Is it idiomatic, readable, well-structured? Proper error handling?
3. **Spec Compliance ({{.weights.spec_compliance}}%)**: Does it follow all constraints, use specified libraries, match the plan?
4. **Security ({{.weights.security}}%)**: Any vulnerabilities? Proper input validation? No hardcoded secrets?
5. **Documentation ({{.weights.documentation}}%)**: Adequate comments, docstrings, README?

## Output Format
Return a JSON object with exactly this structure:
{
  "score": 0.0-1.0,
  "passed": true/false,
  "criteria": {
    "correctness": 0.0-1.0,
    "code_quality": 0.0-1.0,
    "spec_compliance": 0.0-1.0,
    "security": 0.0-1.0,
    "documentation": 0.0-1.0
  },
  "feedback": "Detailed explanation of scores...",
  "improvements": [
    "Specific improvement 1",
    "Specific improvement 2"
  ]
}`

// NewEvaluationEngine creates a new evaluation engine
func NewEvaluationEngine(client LLMClient) *EvaluationEngine {
	return &EvaluationEngine{client: client}
}

// EvaluateCode evaluates generated code against a specification
func (e *EvaluationEngine) EvaluateCode(ctx context.Context, specContent, codeContent, contextContent string, criteria EvaluationCriteria) (*EvaluationResult, error) {
	// Build the prompt
	prompt := strings.Replace(EvaluationPromptTemplate, "{{.spec}}", specContent, 1)
	prompt = strings.Replace(prompt, "{{.code}}", codeContent, 1)
	prompt = strings.Replace(prompt, "{{.context}}", contextContent, 1)
	prompt = strings.Replace(prompt, "{{.weights.correctness}}", fmt.Sprintf("%.0f", criteria.Correctness*100), 1)
	prompt = strings.Replace(prompt, "{{.weights.code_quality}}", fmt.Sprintf("%.0f", criteria.CodeQuality*100), 1)
	prompt = strings.Replace(prompt, "{{.weights.spec_compliance}}", fmt.Sprintf("%.0f", criteria.SpecCompliance*100), 1)
	prompt = strings.Replace(prompt, "{{.weights.security}}", fmt.Sprintf("%.0f", criteria.Security*100), 1)
	prompt = strings.Replace(prompt, "{{.weights.documentation}}", fmt.Sprintf("%.0f", criteria.Documentation*100), 1)

	// Call LLM
	response, err := e.client.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM evaluation failed: %w", err)
	}

	// Parse JSON response
	var result EvaluationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract JSON from markdown code block
		jsonStr := extractJSON(response)
		if jsonStr == "" {
			return nil, fmt.Errorf("failed to parse evaluation response: %w", err)
		}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
		}
	}

	// Determine pass/fail based on threshold
	result.Passed = result.Score >= criteria.Threshold

	return &result, nil
}

// extractJSON extracts JSON from a markdown code block or raw response
func extractJSON(response string) string {
	// Try to find JSON in code block
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		return response[start : end+1]
	}
	return ""
}

// BenchmarkSuite runs benchmark evaluations across multiple configurations
type BenchmarkSuite struct {
	engine   *EvaluationEngine
	criteria EvaluationCriteria
	results  []BenchmarkResult
}

// BenchmarkResult represents a single benchmark run
type BenchmarkResult struct {
	ConfigName   string
	SpecID       string
	Score        float64
	Passed       bool
	Duration     float64
	Timestamp    string
	Criteria     map[string]float64
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(client LLMClient, criteria EvaluationCriteria) *BenchmarkSuite {
	return &BenchmarkSuite{
		engine:   NewEvaluationEngine(client),
		criteria: criteria,
		results:  make([]BenchmarkResult, 0),
	}
}

// RunBenchmark runs a benchmark evaluation
func (b *BenchmarkSuite) RunBenchmark(ctx context.Context, configName, specID, specContent, codeContent, contextContent string) (*BenchmarkResult, error) {
	result, err := b.engine.EvaluateCode(ctx, specContent, codeContent, contextContent, b.criteria)
	if err != nil {
		return nil, err
	}

	br := BenchmarkResult{
		ConfigName: configName,
		SpecID:     specID,
		Score:      result.Score,
		Passed:     result.Passed,
		Criteria:   result.Criteria,
	}

	b.results = append(b.results, br)
	return &br, nil
}

// CompareConfigs compares two configurations across the same specs
func (b *BenchmarkSuite) CompareConfigs(configA, configB string) (map[string]float64, error) {
	comparison := make(map[string]float64)

	for _, r := range b.results {
		if r.ConfigName == configA {
			comparison[configA+"_"+r.SpecID] = r.Score
		} else if r.ConfigName == configB {
			comparison[configB+"_"+r.SpecID] = r.Score
		}
	}

	return comparison, nil
}

// GetBestConfig returns the best performing configuration
func (b *BenchmarkSuite) GetBestConfig() string {
	scores := make(map[string][]float64)

	for _, r := range b.results {
		scores[r.ConfigName] = append(scores[r.ConfigName], r.Score)
	}

	bestConfig := ""
	bestAvg := 0.0

	for config, vals := range scores {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		avg := sum / float64(len(vals))
		if avg > bestAvg {
			bestAvg = avg
			bestConfig = config
		}
	}

	return bestConfig
}