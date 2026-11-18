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
	defaultOllamaURL   = "http://localhost:11434"
	defaultOllamaModel = "mistral"
)

// OllamaProvider calls a local or remote Ollama server (/api/generate).
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_URL")
		if baseURL == "" {
			baseURL = defaultOllamaURL
		}
	}
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = defaultOllamaModel
		}
	}
	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *OllamaProvider) ProviderName() string {
	return "ollama"
}

func (p *OllamaProvider) WithHTTPClient(client *http.Client) *OllamaProvider {
	if client != nil {
		p.httpClient = client
	}
	return p
}

func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama unreachable at %s: %w", p.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ollama health check status %d", resp.StatusCode)
	}
	return nil
}

func (p *OllamaProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return p.Complete(ctx, req)
}

func (p *OllamaProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	model := p.model
	if req != nil && req.Model != "" && !strings.HasPrefix(req.Model, "claude-") && !strings.HasPrefix(req.Model, "gpt-") {
		model = req.Model
	}

	payload := map[string]interface{}{
		"model":  model,
		"prompt": req.Prompt,
		"stream": false,
	}
	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}
	if req.JsonMode {
		payload["format"] = "json"
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama HTTP error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPStatusError{
			Provider:   "ollama",
			StatusCode: resp.StatusCode,
			Body:       string(respBytes),
		}
	}

	var parsed struct {
		Model           string `json:"model"`
		Response        string `json:"response"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return &services.LLMResponse{
		Content:  parsed.Response,
		Tokens:   parsed.PromptEvalCount + parsed.EvalCount,
		Provider: p.ProviderName(),
		Model:    parsed.Model,
	}, nil
}
