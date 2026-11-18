package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements LLMProvider for OpenAI API
type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates an OpenAI provider
func NewOpenAIProvider(config ProviderConfig) (LLMProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("openai provider requires APIKey")
	}

	model := config.Model
	if model == "" {
		model = "gpt-4-turbo"
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		apiKey:  config.APIKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	prompt := buildDecisionPrompt(req)

	apiReq := map[string]interface{}{
		"model":       o.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.7,
	}

	respBody, err := o.callAPI(ctx, "/chat/completions", apiReq)
	if err != nil {
		return nil, err
	}

	var apiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	content := extractOpenAIContent(apiResp)
	return parseDecisionResponse(content), nil
}

func (o *OpenAIProvider) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Analyze this code for quality issues:

%s

Respond with JSON: {"complexity": "simple|medium|complex", "issues": [], "quality": 0-1}`, code)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return extractJSON(resp), nil
}

func (o *OpenAIProvider) ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Review this specification:

%s

Respond with JSON: {"feasible": true|false, "complexity": "simple|medium|complex", "risks": []}`, spec)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return extractJSON(resp), nil
}

func (o *OpenAIProvider) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Triage this issue:

%s

Respond with JSON: {"category": "bug|feature|enhancement", "priority": "low|medium|high", "agent": "Architect|Developer|QA"}`, issue)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return extractJSON(resp), nil
}

// Helper methods

func (o *OpenAIProvider) callPrompt(ctx context.Context, prompt string) (string, error) {
	apiReq := map[string]interface{}{
		"model":       o.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.7,
	}

	respBody, err := o.callAPI(ctx, "/chat/completions", apiReq)
	if err != nil {
		return "", err
	}

	var apiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	return extractOpenAIContent(apiResp), nil
}

func (o *OpenAIProvider) callAPI(ctx context.Context, endpoint string, req interface{}) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api error: %d", resp.StatusCode)
	}

	return respBody, nil
}

func extractOpenAIContent(resp map[string]interface{}) string {
	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content
				}
			}
		}
	}
	return ""
}

// Shared helper functions

func buildDecisionPrompt(req *DecisionRequest) string {
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

func parseDecisionResponse(text string) *DecisionResponse {
	data := extractJSON(text)

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

func extractJSON(text string) map[string]interface{} {
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
