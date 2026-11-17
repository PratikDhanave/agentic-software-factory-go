package analysis

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type BugDetector struct {
	ollama *llm.OllamaClient
}

type Bug struct {
	ID           string `json:"id"`
	Severity     string `json:"severity"` // critical, high, medium, low
	Type         string `json:"type"`     // null-pointer, race-condition, memory-leak, etc
	Location     string `json:"location"`
	Description  string `json:"description"`
	Impact       string `json:"impact"`
	FixSuggestion string `json:"fix_suggestion"`
	Confidence   int    `json:"confidence"` // 0-100
}

type BugAnalysis struct {
	Bugs          []Bug  `json:"bugs"`
	TotalIssues   int    `json:"total_issues"`
	CriticalCount int    `json:"critical_count"`
	FixedCode     string `json:"fixed_code,omitempty"`
	Summary       string `json:"summary"`
}

func NewBugDetector(ollamaURL, model string) *BugDetector {
	return &BugDetector{
		ollama: llm.NewOllamaClient(ollamaURL, model),
	}
}

// Detect finds potential bugs in code
func (bd *BugDetector) Detect(code, language string) (*BugAnalysis, error) {
	log.Printf("[BugDetect] Analyzing %s code", language)

	prompt := fmt.Sprintf(`Analyze this %s code for bugs and issues:

%s

Look for:
- Logic errors
- Null pointer dereferences
- Resource leaks
- Race conditions
- Off-by-one errors
- Type mismatches
- Uncaught exceptions
- Performance issues

Return JSON:
{
  "bugs": [
    {"id": "B1", "severity": "critical|high|medium|low", "type": "...", "location": "line/function", "description": "...", "impact": "...", "fix_suggestion": "...", "confidence": 85}
  ],
  "total_issues": N,
  "critical_count": N,
  "summary": "..."
}`, language, code)

	response, err := bd.ollama.Analyze(prompt)
	if err != nil {
		return nil, fmt.Errorf("bug detection failed: %w", err)
	}

	var analysis BugAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		analysis.Summary = response
	}

	log.Printf("[BugDetect] Found %d issues (%d critical)", analysis.TotalIssues, analysis.CriticalCount)
	return &analysis, nil
}

// Fix attempts to fix bugs in code
func (bd *BugDetector) Fix(code, language string, bugs []Bug) (string, error) {
	log.Printf("[BugDetect] Fixing %d bugs", len(bugs))

	bugDesc := "Bugs to fix:\n"
	for i, bug := range bugs {
		bugDesc += fmt.Sprintf("%d. %s (line: %s): %s\nFix: %s\n", i+1, bug.Type, bug.Location, bug.Description, bug.FixSuggestion)
	}

	prompt := fmt.Sprintf(`Fix these bugs in the %s code:

%s

Original code:
%s

Return ONLY the fixed code with comments explaining changes.`, language, bugDesc, code)

	fixedCode, err := bd.ollama.Analyze(prompt)
	if err != nil {
		return "", err
	}

	return fixedCode, nil
}

// QuickCheck does a fast scan for critical issues
func (bd *BugDetector) QuickCheck(code string) (bool, []string, error) {
	prompt := fmt.Sprintf(`Quick security scan for critical issues:

%s

List any CRITICAL issues found (security, crashes, data loss):
Return JSON: {"has_critical": true/false, "issues": ["issue1", "issue2"]}`, code)

	response, err := bd.ollama.Analyze(prompt)
	if err != nil {
		return false, nil, err
	}

	var result struct {
		HasCritical bool     `json:"has_critical"`
		Issues      []string `json:"issues"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return false, nil, nil
	}

	return result.HasCritical, result.Issues, nil
}
