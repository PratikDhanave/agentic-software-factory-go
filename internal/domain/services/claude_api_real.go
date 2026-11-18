package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RealClaudeAPIClient uses actual Claude API for decisions
type RealClaudeAPIClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewRealClaudeAPIClient creates a client for real Claude API
func NewRealClaudeAPIClient(apiKey, model string) *RealClaudeAPIClient {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &RealClaudeAPIClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1",
		client:  &http.Client{},
	}
}

// GenerateDecision uses Claude to generate a workflow decision
func (c *RealClaudeAPIClient) GenerateDecision(ctx context.Context, req *ClaudeDecisionRequest) (*ClaudeDecisionResponse, error) {
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
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	// Parse Claude's response
	responseText := apiResp.Content[0].Text
	decision := c.parseDecisionResponse(responseText)

	return decision, nil
}

// AnalyzeCode uses Claude to analyze code for quality issues
func (c *RealClaudeAPIClient) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Analyze this code for quality issues, complexity, and best practices:

%s

Respond with JSON:
{
  "complexity": "simple|medium|complex",
  "issues": ["issue1", "issue2"],
  "quality": 0.0-1.0,
  "suggestions": ["suggestion1", "suggestion2"]
}`, code)

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
		return nil, err
	}

	// Extract JSON from response
	analysis := c.extractJSON(apiResp.Content[0].Text)
	return analysis, nil
}

// ReviewSpecification uses Claude to review a specification
func (c *RealClaudeAPIClient) ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Review this specification for feasibility, risks, and improvements:

%s

Respond with JSON:
{
  "feasible": true|false,
  "complexity": "simple|medium|complex",
  "risks": ["risk1", "risk2"],
  "improvement_suggestions": ["suggestion1", "suggestion2"],
  "effort_estimate": "time estimate"
}`, spec)

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
		return nil, err
	}

	review := c.extractJSON(apiResp.Content[0].Text)
	return review, nil
}

// TriageIssue uses Claude to triage and classify a GitHub issue
func (c *RealClaudeAPIClient) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Triage this GitHub issue:

%s

Respond with JSON:
{
  "category": "bug|feature|enhancement|documentation|question",
  "priority": "low|medium|high|critical",
  "suggested_agent": "Architect|Developer|QA|Security|Performance",
  "effort_estimate": "time estimate",
  "keywords": ["keyword1", "keyword2"],
  "blockers": ["blocker1", "blocker2"]
}`, issue)

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
		return nil, err
	}

	triage := c.extractJSON(apiResp.Content[0].Text)
	return triage, nil
}

// Helper methods

func (c *RealClaudeAPIClient) buildDecisionPrompt(req *ClaudeDecisionRequest) string {
	prompt := fmt.Sprintf(`You are a %s AI agent in a software factory workflow system.

Issue Title: %s
Issue Description: %s

Your task is to generate a decision for how to handle this issue. Consider the issue type and provide a clear, actionable decision.

Respond ONLY with valid JSON (no markdown, no explanation):
{
  "decision": "specific action to take",
  "reasoning": "why this approach",
  "confidence": 0.0-1.0,
  "alternatives": ["option2", "option3"],
  "estimated_effort": "time estimate"
}`, req.AgentRole, req.IssueTitle, req.IssueDescription)

	if len(req.PreviousDecisions) > 0 {
		prompt += "\n\nPrevious decisions for context:\n"
		for i, prev := range req.PreviousDecisions {
			if decision, ok := prev["decision"].(string); ok {
				prompt += fmt.Sprintf("%d. %s\n", i+1, decision)
			}
		}
	}

	return prompt
}

func (c *RealClaudeAPIClient) callAPI(ctx context.Context, endpoint string, req *claudeAPIRequest) ([]byte, error) {
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
		return nil, fmt.Errorf("Claude API error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *RealClaudeAPIClient) parseDecisionResponse(text string) *ClaudeDecisionResponse {
	// Extract JSON from response
	data := c.extractJSON(text)

	decision := &ClaudeDecisionResponse{
		Decision:    "",
		Reasoning:   "",
		Confidence:  0.85,
		Alternatives: []string{},
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

func (c *RealClaudeAPIClient) extractJSON(text string) map[string]interface{} {
	// Find JSON in the response
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

// API request/response types

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
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
