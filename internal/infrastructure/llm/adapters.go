package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
)

// PromptAdapter adapts services.LLMProvider to harness.AgentClient,
// eval.LLMClient, and pkg/llm.Client.
type PromptAdapter struct {
	provider     services.LLMProvider
	systemPrompt string
	maxTokens    int
	temperature  float64
}

var (
	_ harness.AgentClient = (*PromptAdapter)(nil)
	_ eval.LLMClient      = (*PromptAdapter)(nil)
)

func NewPromptAdapter(provider services.LLMProvider, systemPrompt string) *PromptAdapter {
	return &PromptAdapter{
		provider:     provider,
		systemPrompt: systemPrompt,
		maxTokens:    4096,
		temperature:  0.2,
	}
}

func (a *PromptAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := a.provider.Complete(ctx, &services.LLMRequest{
		SystemPrompt: a.systemPrompt,
		Prompt:       prompt,
		MaxTokens:    a.maxTokens,
		Temperature:  a.temperature,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (a *PromptAdapter) Analyze(prompt string) (string, error) {
	return a.Generate(context.Background(), prompt)
}

// ClaudeClientAdapter implements services.ClaudeClient on top of any LLMProvider.
type ClaudeClientAdapter struct {
	provider services.LLMProvider
}

var _ services.ClaudeClient = (*ClaudeClientAdapter)(nil)

func NewClaudeClientAdapter(provider services.LLMProvider) *ClaudeClientAdapter {
	return &ClaudeClientAdapter{provider: provider}
}

func (c *ClaudeClientAdapter) GenerateDecision(ctx context.Context, req *services.ClaudeDecisionRequest) (*services.ClaudeDecisionResponse, error) {
	prompt := fmt.Sprintf(`You are the %s agent in an Agentic Software Factory.

Issue: %s
Description: %s

Return valid JSON only:
{
  "decision": "specific action to take",
  "reasoning": "why this approach",
  "confidence": 0.85,
  "alternatives": ["option1", "option2"]
}`, req.AgentRole, req.IssueTitle, req.IssueDescription)

	resp, err := c.provider.Complete(ctx, &services.LLMRequest{
		Prompt:      prompt,
		Temperature: 0.3,
		MaxTokens:   1024,
		JsonMode:    true,
	})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Decision     string   `json:"decision"`
		Reasoning    string   `json:"reasoning"`
		Confidence   float64  `json:"confidence"`
		Alternatives []string `json:"alternatives"`
	}
	clean := services.ExtractJSONBlock(resp.Content)
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return &services.ClaudeDecisionResponse{
			Decision:   resp.Content,
			Reasoning:  "Unstructured provider output",
			Confidence: 0.75,
		}, nil
	}

	return &services.ClaudeDecisionResponse{
		Decision:     parsed.Decision,
		Reasoning:    parsed.Reasoning,
		Confidence:   parsed.Confidence,
		Alternatives: parsed.Alternatives,
	}, nil
}

func (c *ClaudeClientAdapter) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	return c.callJSONMap(ctx, fmt.Sprintf("Analyze this code and return JSON with complexity, issues, and quality (0-1):\n\n%s", code))
}

func (c *ClaudeClientAdapter) ReviewSpecification(ctx context.Context, specText string) (map[string]interface{}, error) {
	return c.callJSONMap(ctx, fmt.Sprintf("Review this specification and return JSON with feasible, complexity, risks, and improvement_suggestions:\n\n%s", specText))
}

func (c *ClaudeClientAdapter) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
	return c.callJSONMap(ctx, fmt.Sprintf("Triage this issue and return JSON with category, priority, suggested_agent, and effort_estimate:\n\n%s", issue))
}

func (c *ClaudeClientAdapter) callJSONMap(ctx context.Context, prompt string) (map[string]interface{}, error) {
	resp, err := c.provider.Complete(ctx, &services.LLMRequest{
		Prompt:      prompt,
		Temperature: 0.2,
		MaxTokens:   1024,
		JsonMode:    true,
	})
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(services.ExtractJSONBlock(resp.Content)), &out); err != nil {
		out["raw"] = resp.Content
	}
	return out, nil
}
