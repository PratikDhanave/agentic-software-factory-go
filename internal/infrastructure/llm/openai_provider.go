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
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

// OpenAIProvider calls any OpenAI-compatible /chat/completions endpoint
// (OpenAI, LiteLLM, vLLM, Groq, Gemini OpenAI compatibility endpoint, etc.).
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = defaultOpenAIBaseURL
		}
	}
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = defaultOpenAIModel
		}
	}
	return &OpenAIProvider{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *OpenAIProvider) ProviderName() string {
	return "openai"
}

func (p *OpenAIProvider) WithHTTPClient(client *http.Client) *OpenAIProvider {
	if client != nil {
		p.httpClient = client
	}
	return p
}

func (p *OpenAIProvider) Configured() bool {
	return p.apiKey != "" || os.Getenv("OPENAI_BASE_URL") != ""
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	if !p.Configured() {
		return fmt.Errorf("openai API key or base URL not configured")
	}
	return nil
}

func (p *OpenAIProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return p.Complete(ctx, req)
}

func (p *OpenAIProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("openai provider is not configured")
	}

	model := p.model
	if req != nil && req.Model != "" && !strings.HasPrefix(req.Model, "claude-") {
		model = req.Model
	}

	messages := make([]map[string]string, 0, 2)
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": req.Prompt,
	})

	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.JsonMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai HTTP error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPStatusError{
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			Body:       string(respBytes),
		}
	}

	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai response returned zero choices")
	}

	return &services.LLMResponse{
		Content:  parsed.Choices[0].Message.Content,
		Tokens:   parsed.Usage.TotalTokens,
		Provider: p.ProviderName(),
		Model:    parsed.Model,
	}, nil
}
