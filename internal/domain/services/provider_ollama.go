package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider implements LLMProvider for Ollama (local models)
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaProvider creates an Ollama provider for local models
func NewOllamaProvider(config ProviderConfig) (LLMProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := config.Model
	if model == "" {
		model = "llama2" // or mistral, neural-chat, etc.
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}, nil
}

func (o *OllamaProvider) Name() string {
	return "ollama"
}

func (o *OllamaProvider) GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	prompt := buildDecisionPrompt(req)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return parseDecisionResponse(resp), nil
}

func (o *OllamaProvider) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Analyze this code for quality issues:

%s

Respond with JSON: {"complexity": "simple|medium|complex", "issues": [], "quality": 0-1}`, code)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return extractJSON(resp), nil
}

func (o *OllamaProvider) ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Review this specification:

%s

Respond with JSON: {"feasible": true|false, "complexity": "simple|medium|complex", "risks": []}`, spec)

	resp, err := o.callPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return extractJSON(resp), nil
}

func (o *OllamaProvider) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
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

func (o *OllamaProvider) callPrompt(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama connection failed: %w (is Ollama running at %s?)", err, o.baseURL)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama error: %d", resp.StatusCode)
	}

	var apiResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	return apiResp.Response, nil
}
