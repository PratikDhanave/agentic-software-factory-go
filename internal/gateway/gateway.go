// Package gateway provides a unified, hot-reloadable Model Gateway across
// Google Gemini, Anthropic Claude, OpenAI, Local Ollama, and the built-in
// offline deterministic Go synthesizer.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

type ProviderSlot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	Configured  bool   `json:"configured"`
	MaskedKey   string `json:"masked_key,omitempty"`
	State       string `json:"state"`
	Description string `json:"description"`
}

type UpdateConfig struct {
	PreferredProvider string `json:"preferred_provider"`
	GeminiAPIKey      string `json:"gemini_api_key"`
	GeminiModel       string `json:"gemini_model"`
	AnthropicAPIKey   string `json:"anthropic_api_key"`
	AnthropicModel    string `json:"anthropic_model"`
	OpenAIAPIKey      string `json:"openai_api_key"`
	OpenAIModel       string `json:"openai_model"`
	OllamaURL         string `json:"ollama_url"`
	OllamaModel       string `json:"ollama_model"`
}

type Response struct {
	Content    string
	Provider   string
	Model      string
	Tokens     int
	DurationMs int64
}

type Gateway struct {
	mu     sync.RWMutex
	store  *store.Store
	client *http.Client

	preferred      string
	geminiKey      string
	geminiModel    string
	anthropicKey   string
	anthropicModel string
	openaiKey      string
	openaiModel    string
	ollamaURL      string
	ollamaModel    string
}

func New(st *store.Store) *Gateway {
	ctx := context.Background()
	firstNonEmpty := func(vals ...string) string {
		for _, v := range vals {
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	g := &Gateway{
		store:          st,
		client:         &http.Client{Timeout: 45 * time.Second},
		preferred:      st.GetSetting(ctx, "preferred_provider", "auto"),
		geminiKey:      firstNonEmpty(st.GetSetting(ctx, "gemini_api_key", ""), os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY")),
		geminiModel:    firstNonEmpty(st.GetSetting(ctx, "gemini_model", ""), os.Getenv("GEMINI_MODEL"), "gemini-2.5-pro"),
		anthropicKey:   firstNonEmpty(st.GetSetting(ctx, "anthropic_api_key", ""), os.Getenv("ANTHROPIC_API_KEY")),
		anthropicModel: firstNonEmpty(st.GetSetting(ctx, "anthropic_model", ""), "claude-3-7-sonnet-latest"),
		openaiKey:      firstNonEmpty(st.GetSetting(ctx, "openai_api_key", ""), os.Getenv("OPENAI_API_KEY")),
		openaiModel:    firstNonEmpty(st.GetSetting(ctx, "openai_model", ""), "gpt-4o"),
		ollamaURL:      firstNonEmpty(st.GetSetting(ctx, "ollama_url", ""), os.Getenv("OLLAMA_URL")),
		ollamaModel:    firstNonEmpty(st.GetSetting(ctx, "ollama_model", ""), "qwen2.5-coder:7b"),
	}
	return g
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		if s == "" {
			return ""
		}
		return "••••••••"
	}
	return s[:4] + "••••" + s[len(s)-4:]
}

func (g *Gateway) ActiveProvider() (id, model string) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.resolveLocked()
}

func (g *Gateway) resolveLocked() (string, string) {
	switch g.preferred {
	case "gemini":
		if g.geminiKey != "" {
			return "gemini", g.geminiModel
		}
	case "anthropic":
		if g.anthropicKey != "" {
			return "anthropic", g.anthropicModel
		}
	case "openai":
		if g.openaiKey != "" {
			return "openai", g.openaiModel
		}
	case "ollama":
		if g.ollamaURL != "" {
			return "ollama", g.ollamaModel
		}
	case "synth":
		return "synth", "asf-deterministic-go-v2"
	}
	if g.geminiKey != "" {
		return "gemini", g.geminiModel
	}
	if g.anthropicKey != "" {
		return "anthropic", g.anthropicModel
	}
	if g.openaiKey != "" {
		return "openai", g.openaiModel
	}
	if g.ollamaURL != "" {
		return "ollama", g.ollamaModel
	}
	return "synth", "asf-deterministic-go-v2"
}

func (g *Gateway) Status() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	activeID, activeModel := g.resolveLocked()
	slotState := func(id string, configured bool) string {
		st := string(state.ProviderUnconfigured)
		if configured {
			_ = state.GatewayProviderMachine.TransitionString(&st, state.ProviderStandby)
		}
		if id == activeID {
			_ = state.GatewayProviderMachine.TransitionString(&st, state.ProviderActive)
		}
		return st
	}

	slots := []ProviderSlot{
		{
			ID:          "gemini",
			Name:        "Google Gemini",
			Model:       g.geminiModel,
			Configured:  g.geminiKey != "",
			MaskedKey:   maskSecret(g.geminiKey),
			State:       slotState("gemini", g.geminiKey != ""),
			Description: "Primary reasoning & Go code synthesis via Gemini 2.5 Pro / Flash.",
		},
		{
			ID:          "anthropic",
			Name:        "Anthropic Claude",
			Model:       g.anthropicModel,
			Configured:  g.anthropicKey != "",
			MaskedKey:   maskSecret(g.anthropicKey),
			State:       slotState("anthropic", g.anthropicKey != ""),
			Description: "Claude Sonnet code generation & security review provider.",
		},
		{
			ID:          "openai",
			Name:        "OpenAI",
			Model:       g.openaiModel,
			Configured:  g.openaiKey != "",
			MaskedKey:   maskSecret(g.openaiKey),
			State:       slotState("openai", g.openaiKey != ""),
			Description: "GPT-4o / o3-mini fallback inference endpoint.",
		},
		{
			ID:          "ollama",
			Name:        "Local Ollama",
			Model:       g.ollamaModel,
			Configured:  g.ollamaURL != "",
			MaskedKey:   g.ollamaURL,
			State:       slotState("ollama", g.ollamaURL != ""),
			Description: "Self-hosted local model server (e.g. http://localhost:11434).",
		},
		{
			ID:          "synth",
			Name:        "Built-in Deterministic Go Synth",
			Model:       "asf-deterministic-go-v2",
			Configured:  true,
			State:       slotState("synth", true),
			Description: "Zero-key offline Go package & race-test generator — works out of the box.",
		},
	}

	return map[string]any{
		"active_provider":    activeID,
		"active_model":       activeModel,
		"preferred_provider": g.preferred,
		"live_llm":           activeID != "synth",
		"providers":          slots,
	}
}

func (g *Gateway) Update(ctx context.Context, cfg UpdateConfig) map[string]any {
	g.mu.Lock()
	if cfg.PreferredProvider != "" {
		g.preferred = strings.TrimSpace(cfg.PreferredProvider)
		_ = g.store.SetSetting(ctx, "preferred_provider", g.preferred)
	}
	if cfg.GeminiAPIKey != "" {
		val := strings.TrimSpace(cfg.GeminiAPIKey)
		if val == "clear" {
			val = ""
		}
		g.geminiKey = val
		_ = g.store.SetSetting(ctx, "gemini_api_key", val)
	}
	if cfg.GeminiModel != "" {
		g.geminiModel = strings.TrimSpace(cfg.GeminiModel)
		_ = g.store.SetSetting(ctx, "gemini_model", g.geminiModel)
	}
	if cfg.AnthropicAPIKey != "" {
		val := strings.TrimSpace(cfg.AnthropicAPIKey)
		if val == "clear" {
			val = ""
		}
		g.anthropicKey = val
		_ = g.store.SetSetting(ctx, "anthropic_api_key", val)
	}
	if cfg.AnthropicModel != "" {
		g.anthropicModel = strings.TrimSpace(cfg.AnthropicModel)
		_ = g.store.SetSetting(ctx, "anthropic_model", g.anthropicModel)
	}
	if cfg.OpenAIAPIKey != "" {
		val := strings.TrimSpace(cfg.OpenAIAPIKey)
		if val == "clear" {
			val = ""
		}
		g.openaiKey = val
		_ = g.store.SetSetting(ctx, "openai_api_key", val)
	}
	if cfg.OpenAIModel != "" {
		g.openaiModel = strings.TrimSpace(cfg.OpenAIModel)
		_ = g.store.SetSetting(ctx, "openai_model", g.openaiModel)
	}
	if cfg.OllamaURL != "" {
		val := strings.TrimSpace(cfg.OllamaURL)
		if val == "clear" {
			val = ""
		}
		g.ollamaURL = val
		_ = g.store.SetSetting(ctx, "ollama_url", val)
	}
	if cfg.OllamaModel != "" {
		g.ollamaModel = strings.TrimSpace(cfg.OllamaModel)
		_ = g.store.SetSetting(ctx, "ollama_model", g.ollamaModel)
	}
	g.mu.Unlock()
	return g.Status()
}

// GenerateCode calls the active LLM provider or falls back cleanly to deterministic Go synthesis.
func (g *Gateway) GenerateCode(ctx context.Context, specID, title, prompt, repairFeedback string) Response {
	start := time.Now()
	provider, model := g.ActiveProvider()

	sysPrompt := "You are Factory Go Developer. Output ONLY valid Go source files and *_test.go files enclosed in fenced blocks formatted strictly as:\n```go path=filename.go\npackage ...\n```\nUse only the Go standard library so `go test -race ./...` passes with zero external downloads."
	userPrompt := fmt.Sprintf("Specification: %s (%s)\nTask: %s", title, specID, prompt)
	if repairFeedback != "" {
		userPrompt += "\n\nFix the following compiler/test failure:\n" + repairFeedback
	}

	if provider != "synth" {
		if text, tokens, err := g.callProvider(ctx, provider, model, sysPrompt, userPrompt); err == nil && strings.Contains(text, "```go") {
			return Response{
				Content:    text,
				Provider:   provider,
				Model:      model,
				Tokens:     tokens,
				DurationMs: time.Since(start).Milliseconds(),
			}
		}
	}

	out := SynthesizeGo(specID, title, prompt)
	return Response{
		Content:    out,
		Provider:   "synth",
		Model:      "asf-deterministic-go-v2",
		Tokens:     len(out) / 4,
		DurationMs: maxInt64(1, time.Since(start).Milliseconds()),
	}
}

// Consult queries a specific Factory Agent persona using the active gateway provider.
func (g *Gateway) Consult(ctx context.Context, agent *store.AgentRecord, prompt string) Response {
	start := time.Now()
	provider, model := g.ActiveProvider()

	if provider != "synth" {
		if text, tokens, err := g.callProvider(ctx, provider, model, agent.SystemPrompt, prompt); err == nil && strings.TrimSpace(text) != "" {
			return Response{
				Content:    text,
				Provider:   provider,
				Model:      model,
				Tokens:     tokens,
				DurationMs: time.Since(start).Milliseconds(),
			}
		}
	}

	reply := SynthesizeConsultation(agent.Role, agent.Name, prompt)
	return Response{
		Content:    reply,
		Provider:   "synth",
		Model:      "asf-deterministic-go-v2",
		Tokens:     len(reply) / 4,
		DurationMs: maxInt64(1, time.Since(start).Milliseconds()),
	}
}

func (g *Gateway) callProvider(ctx context.Context, provider, model, sysPrompt, userPrompt string) (string, int, error) {
	g.mu.RLock()
	geminiKey := g.geminiKey
	anthropicKey := g.anthropicKey
	openaiKey := g.openaiKey
	ollamaURL := g.ollamaURL
	g.mu.RUnlock()

	switch provider {
	case "gemini":
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, geminiKey)
		payload := map[string]any{
			"systemInstruction": map[string]any{
				"parts": []map[string]string{{"text": sysPrompt}},
			},
			"contents": []map[string]any{
				{"role": "user", "parts": []map[string]string{{"text": userPrompt}}},
			},
			"generationConfig": map[string]any{"temperature": 0.2},
		}
		body, err := g.postJSON(ctx, url, payload, nil)
		if err != nil {
			return "", 0, err
		}
		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				TotalTokenCount int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return "", 0, fmt.Errorf("empty gemini response")
		}
		text := parsed.Candidates[0].Content.Parts[0].Text
		tok := parsed.UsageMetadata.TotalTokenCount
		if tok == 0 {
			tok = len(text) / 4
		}
		return text, tok, nil

	case "anthropic":
		payload := map[string]any{
			"model":      model,
			"max_tokens": 4096,
			"system":     sysPrompt,
			"messages":   []map[string]string{{"role": "user", "content": userPrompt}},
		}
		headers := map[string]string{
			"x-api-key":         anthropicKey,
			"anthropic-version": "2023-06-01",
		}
		body, err := g.postJSON(ctx, "https://api.anthropic.com/v1/messages", payload, headers)
		if err != nil {
			return "", 0, err
		}
		var parsed struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Content) == 0 {
			return "", 0, fmt.Errorf("empty anthropic response")
		}
		return parsed.Content[0].Text, parsed.Usage.InputTokens + parsed.Usage.OutputTokens, nil

	case "openai":
		payload := map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": sysPrompt},
				{"role": "user", "content": userPrompt},
			},
			"temperature": 0.2,
		}
		headers := map[string]string{"Authorization": "Bearer " + openaiKey}
		body, err := g.postJSON(ctx, "https://api.openai.com/v1/chat/completions", payload, headers)
		if err != nil {
			return "", 0, err
		}
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
			return "", 0, fmt.Errorf("empty openai response")
		}
		return parsed.Choices[0].Message.Content, parsed.Usage.TotalTokens, nil

	case "ollama":
		endpoint := strings.TrimRight(ollamaURL, "/") + "/api/chat"
		payload := map[string]any{
			"model":  model,
			"stream": false,
			"messages": []map[string]string{
				{"role": "system", "content": sysPrompt},
				{"role": "user", "content": userPrompt},
			},
		}
		body, err := g.postJSON(ctx, endpoint, payload, nil)
		if err != nil {
			return "", 0, err
		}
		var parsed struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return "", 0, err
		}
		return parsed.Message.Content, len(parsed.Message.Content) / 4, nil
	}
	return "", 0, fmt.Errorf("unsupported provider %s", provider)
}

func (g *Gateway) postJSON(ctx context.Context, url string, payload any, headers map[string]string) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(out))
	}
	return out, nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
