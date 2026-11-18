package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ClaudeProvider implements LLMProvider for Claude API
type ClaudeProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaudeProvider creates a Claude provider
func NewClaudeProvider(config ProviderConfig) (LLMProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("claude provider requires APIKey")
	}

	model := config.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &ClaudeProvider{
		apiKey:  config.APIKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1",
		client:  &http.Client{},
	}, nil
}

func (c *ClaudeProvider) Name() string {
	return "claude"
}

func (c *ClaudeProvider) GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	prompt := c.buildDecisionPrompt(req)

	apiReq := &claudeAPIRequest{
		Model:     c.model,
		MaxTokens: 1024,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	respBody, err := c.callAPI(ctx, "/messages", apiReq)
	if err != nil {
		return nil, err
	}

	var apiResp claudeAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return c.parseDecisionResponse(apiResp.Content[0].Text), nil
}

func (c *ClaudeProvider) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Analyze this code for quality issues:

%s

Respond with JSON: {"complexity": "simple|medium|complex", "issues": [], "quality": 0-1}`, code)

	resp, err := c.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return c.extractJSON(resp), nil
}

func (c *ClaudeProvider) ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Review this specification:

%s

Respond with JSON: {"feasible": true|false, "complexity": "simple|medium|complex", "risks": [], "improvements": []}`, spec)

	resp, err := c.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return c.extractJSON(resp), nil
}

func (c *ClaudeProvider) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Triage this issue:

%s

Respond with JSON: {"category": "bug|feature|enhancement", "priority": "low|medium|high", "agent": "Architect|Developer|QA", "effort": "time"}`, issue)

	resp, err := c.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return c.extractJSON(resp), nil
}

// Helper methods

func (c *ClaudeProvider) buildDecisionPrompt(req *DecisionRequest) string {
	return fmt.Sprintf(`You are a %s AI agent in a software factory.

Issue: %s
Description: %s

Provide a decision. Respond ONLY with JSON:
{
  "decision": "action to take",
  "reasoning": "why",
  "confidence": 0.0-1.0,
  "alternatives": []
}`, req.AgentRole, req.IssueTitle, req.IssueDescription)
}

func (c *ClaudeProvider) callPrompt(ctx context.Context, prompt string) (string, error) {
	apiReq := &claudeAPIRequest{
		Model:     c.model,
		MaxTokens: 1024,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	respBody, err := c.callAPI(ctx, "/messages", apiReq)
	if err != nil {
		return "", err
	}

	var apiResp claudeAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return apiResp.Content[0].Text, nil
}

func (c *ClaudeProvider) callAPI(ctx context.Context, endpoint string, req *claudeAPIRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *ClaudeProvider) parseDecisionResponse(text string) *DecisionResponse {
	data := c.extractJSON(text)

	decision := &DecisionResponse{
		Decision:   "Process issue",
		Reasoning:  "Based on analysis",
		Confidence: 0.85,
	}

	if d, ok := data["decision"].(string); ok {
		decision.Decision = d
	}
	if r, ok := data["reasoning"].(string); ok {
		decision.Reasoning = r
	}
	if conf, ok := data["confidence"].(float64); ok {
		decision.Confidence = conf
	}
	if alts, ok := data["alternatives"].([]interface{}); ok {
		for _, alt := range alts {
			if altStr, ok := alt.(string); ok {
				decision.Alternatives = append(decision.Alternatives, altStr)
			}
		}
	}

	return decision
}

func (c *ClaudeProvider) extractJSON(text string) map[string]interface{} {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return map[string]interface{}{}
	}

	jsonStr := text[start : end+1]
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return map[string]interface{}{}
	}

	return data
}

// API types
type claudeAPIRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeAPIResponse struct {
	Content []claudeContent `json:"content"`
	Model   string          `json:"model"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
