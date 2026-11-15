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
)

// openAICompatClient implements Client against any server speaking the
// Chat Completions wire format (OpenAI, Ollama, vLLM, Groq, Mistral,
// LM Studio, ...). Implemented over plain net/http so no provider SDK is
// required for the generic path.
type openAICompatClient struct {
	provider  string
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	http      *http.Client
}

// newOpenAICompatible returns nil (prompt-only mode) when required
// configuration is missing; ConfigError explains what was missing.
func newOpenAICompatible(provider, baseURL, apiKey string) Client {
	model := os.Getenv("ASF_MODEL")
	if baseURL == "" || model == "" {
		return nil
	}
	if provider == "openai" && apiKey == "" {
		return nil
	}
	return &openAICompatClient{
		provider:  provider,
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		model:     model,
		maxTokens: 16000,
		http:      &http.Client{},
	}
}

func (c *openAICompatClient) Provider() string { return c.provider }
func (c *openAICompatClient) Model() string    { return c.model }

type chatRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Messages  []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (c *openAICompatClient) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s request failed: %w", c.provider, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("%s response read failed: %w", c.provider, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("%s auth failed (check the configured API key): HTTP 401", c.provider)
	}
	if resp.StatusCode != http.StatusOK {
		excerpt := string(data)
		if len(excerpt) > 300 {
			excerpt = excerpt[:300]
		}
		return "", fmt.Errorf("%s API error (HTTP %d): %s", c.provider, resp.StatusCode, excerpt)
	}

	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("%s response parse failed: %w", c.provider, err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("%s API error: %s", c.provider, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%s returned no choices", c.provider)
	}

	choice := parsed.Choices[0]
	if choice.FinishReason == "length" {
		return choice.Message.Content, fmt.Errorf("output truncated at %d tokens", c.maxTokens)
	}
	return choice.Message.Content, nil
}
