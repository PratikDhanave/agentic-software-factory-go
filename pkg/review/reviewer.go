package review

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type CodeReviewer struct {
	ollama *llm.OllamaClient
}

type ReviewRequest struct {
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	FileName    string   `json:"file_name"`
	Context     string   `json:"context"`
	FocusAreas  []string `json:"focus_areas"`
}

type ReviewFinding struct {
	Severity string `json:"severity"` // critical, major, minor, info
	Line     int    `json:"line"`
	Category string `json:"category"` // performance, security, style, logic, test
	Message  string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type ReviewResult struct {
	FileName  string           `json:"file_name"`
	Score     int              `json:"score"` // 0-100
	Findings  []ReviewFinding  `json:"findings"`
	Summary   string           `json:"summary"`
	Approved  bool             `json:"approved"`
}

func NewCodeReviewer(ollamaURL, model string) *CodeReviewer {
	return &CodeReviewer{
		ollama: llm.NewOllamaClient(ollamaURL, model),
	}
}

// Review analyzes code and returns findings
func (cr *CodeReviewer) Review(req *ReviewRequest) (*ReviewResult, error) {
	log.Printf("[Review] Analyzing %s (%s)", req.FileName, req.Language)

	prompt := cr.buildReviewPrompt(req)

	response, err := cr.ollama.Analyze(prompt)
	if err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}

	// Parse response
	var result ReviewResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		result.FileName = req.FileName
		result.Summary = response
		result.Score = 50
	}

	result.FileName = req.FileName
	result.Approved = result.Score >= 70 && len(result.Findings) == 0

	log.Printf("[Review] Score: %d, Issues: %d", result.Score, len(result.Findings))
	return &result, nil
}

// ReviewMultiple reviews multiple code files
func (cr *CodeReviewer) ReviewMultiple(requests []ReviewRequest) ([]ReviewResult, error) {
	results := make([]ReviewResult, 0, len(requests))

	for _, req := range requests {
		result, err := cr.Review(&req)
		if err != nil {
			log.Printf("[Review] Error reviewing %s: %v", req.FileName, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// SuggestImprovements provides improvement recommendations
func (cr *CodeReviewer) SuggestImprovements(code, language string) ([]string, error) {
	prompt := fmt.Sprintf(`Review this %s code and suggest 3-5 improvements:

%s

Focus on:
- Code clarity and maintainability
- Performance optimization
- Error handling
- Testing coverage
- Security best practices

Return suggestions as JSON array of strings.`, language, code)

	response, err := cr.ollama.Analyze(prompt)
	if err != nil {
		return nil, err
	}

	var suggestions []string
	if err := json.Unmarshal([]byte(response), &suggestions); err != nil {
		// Fall back to parsing text response
		suggestions = strings.Split(response, "\n")
	}

	return suggestions, nil
}

func (cr *CodeReviewer) buildReviewPrompt(req *ReviewRequest) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Review this %s code:\n\n", req.Language))
	builder.WriteString(fmt.Sprintf("File: %s\n\n", req.FileName))
	builder.WriteString(fmt.Sprintf("Code:\n%s\n\n", req.Code))

	if req.Context != "" {
		builder.WriteString(fmt.Sprintf("Context: %s\n\n", req.Context))
	}

	if len(req.FocusAreas) > 0 {
		builder.WriteString("Focus areas:\n")
		for _, area := range req.FocusAreas {
			builder.WriteString(fmt.Sprintf("- %s\n", area))
		}
		builder.WriteString("\n")
	}

	builder.WriteString(`Return JSON with:
{
  "score": 0-100,
  "findings": [
    {"severity": "critical|major|minor", "line": N, "category": "performance|security|style|logic", "message": "...", "suggestion": "..."}
  ],
  "summary": "overall assessment"
}`)

	return builder.String()
}
