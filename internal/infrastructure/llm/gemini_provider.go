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
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	defaultGeminiModel   = "gemini-2.5-flash"
)

// GeminiProvider calls the Google Gemini GenerateContent REST API using standard net/http.
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewGeminiProvider creates a Google Gemini provider reading GEMINI_API_KEY or GOOGLE_API_KEY.
func NewGeminiProvider(apiKey, baseURL, model string) *GeminiProvider {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
	}
	if baseURL == "" {
		baseURL = os.Getenv("GEMINI_BASE_URL")
		if baseURL == "" {
			baseURL = defaultGeminiBaseURL
		}
	}
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
		if model == "" {
			model = defaultGeminiModel
		}
	}
	return &GeminiProvider{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *GeminiProvider) ProviderName() string {
	return "gemini"
}

func (p *GeminiProvider) ModelName() string {
	return p.model
}

func (p *GeminiProvider) WithHTTPClient(client *http.Client) *GeminiProvider {
	if client != nil {
		p.httpClient = client
	}
	return p
}

func (p *GeminiProvider) Configured() bool {
	return p.apiKey != ""
}

func (p *GeminiProvider) HealthCheck(ctx context.Context) error {
	if !p.Configured() {
		return fmt.Errorf("gemini API key not configured (set GEMINI_API_KEY or GOOGLE_API_KEY)")
	}
	return nil
}

func (p *GeminiProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return p.Complete(ctx, req)
}

func (p *GeminiProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("gemini API key not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("nil LLMRequest")
	}

	start := time.Now()
	model := p.model
	if strings.HasPrefix(req.Model, "gemini-") {
		model = req.Model
	}

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": req.Prompt},
				},
			},
		},
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		payload["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{
				{"text": req.SystemPrompt},
			},
		}
	}

	genCfg := map[string]interface{}{}
	if req.Temperature > 0 {
		genCfg["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		genCfg["maxOutputTokens"] = req.MaxTokens
	}
	if req.JsonMode {
		genCfg["responseMimeType"] = "application/json"
	}
	if len(genCfg) > 0 {
		payload["generationConfig"] = genCfg
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent", p.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPStatusError{
			Provider:   "gemini",
			StatusCode: resp.StatusCode,
			Body:       string(respBytes),
		}
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decoding gemini response: %w", err)
	}

	_ = start
	var out strings.Builder
	if len(parsed.Candidates) > 0 {
		for _, part := range parsed.Candidates[0].Content.Parts {
			out.WriteString(part.Text)
		}
	}

	return &services.LLMResponse{
		Content:  strings.TrimSpace(out.String()),
		Provider: "gemini",
		Model:    model,
		Tokens:   parsed.UsageMetadata.TotalTokenCount,
	}, nil
}
