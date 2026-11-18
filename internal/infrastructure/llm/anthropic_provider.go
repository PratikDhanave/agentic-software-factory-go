package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicModel   = "claude-sonnet-4-20250514"
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider calls the Anthropic Messages API using standard net/http.
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewAnthropicProvider creates an Anthropic Claude provider.
func NewAnthropicProvider(apiKey, baseURL, model string) *AnthropicProvider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}
	}
	if baseURL == "" {
		baseURL = os.Getenv("ANTHROPIC_BASE_URL")
		if baseURL == "" {
			baseURL = defaultAnthropicBaseURL
		}
	}
	if model == "" {
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = os.Getenv("CLAUDE_MODEL")
		}
		if model == "" {
			model = defaultAnthropicModel
		}
	}
	return &AnthropicProvider{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *AnthropicProvider) ProviderName() string {
	return "anthropic"
}

func (p *AnthropicProvider) WithHTTPClient(client *http.Client) *AnthropicProvider {
	if client != nil {
		p.httpClient = client
	}
	return p
}

func (p *AnthropicProvider) Configured() bool {
	return p.apiKey != ""
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	if !p.Configured() {
		return fmt.Errorf("anthropic API key not configured")
	}
	return nil
}

func (p *AnthropicProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return p.Complete(ctx, req)
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("anthropic API key is empty")
	}

	model := p.model
	if req != nil && req.Model != "" {
		model = req.Model
	}
	maxTokens := 2048
	if req != nil && req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}
	temp := 0.3
	if req != nil && req.Temperature > 0 {
		temp = req.Temperature
	}

	payload := map[string]interface{}{
		"model":       model,
		"max_tokens":  maxTokens,
		"temperature": temp,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}
	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic HTTP error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPStatusError{
			Provider:   "anthropic",
			StatusCode: resp.StatusCode,
			Body:       string(respBytes),
		}
	}

	var parsed struct {
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	var builder strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" || block.Text != "" {
			builder.WriteString(block.Text)
		}
	}

	return &services.LLMResponse{
		Content:  builder.String(),
		Tokens:   parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		Provider: p.ProviderName(),
		Model:    parsed.Model,
	}, nil
}
