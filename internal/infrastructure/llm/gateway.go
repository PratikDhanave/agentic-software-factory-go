package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// HTTPStatusError represents a non-2xx HTTP status returned by an LLM provider.
type HTTPStatusError struct {
	Provider   string
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%s API status %d: %s", e.Provider, e.StatusCode, e.Body)
}

func (e *HTTPStatusError) Transient() bool {
	return e.StatusCode == 429 || e.StatusCode == 529 || e.StatusCode >= 500
}

// GatewayConfig configures multi-provider routing and retry policies.
type GatewayConfig struct {
	PreferredProvider string // auto | gemini | anthropic | openai | ollama | mock
	GeminiAPIKey      string
	GeminiBaseURL     string
	GeminiModel       string
	AnthropicAPIKey   string
	AnthropicBaseURL  string
	AnthropicModel    string
	OpenAIAPIKey      string
	OpenAIBaseURL     string
	OpenAIModel       string
	OllamaURL         string
	OllamaModel       string
	HTTPClient        *http.Client
	MaxRetries        int
	AllowMockFallback bool
	Logger            services.Logger
}

// DefaultGatewayConfigFromEnv builds GatewayConfig from standard environment variables.
func DefaultGatewayConfigFromEnv(logger services.Logger) GatewayConfig {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("ASF_LLM_PROVIDER")))
	if mode == "" {
		mode = "auto"
	}
	allowFallback := true
	if strings.EqualFold(os.Getenv("ASF_DISABLE_MOCK_FALLBACK"), "true") {
		allowFallback = false
	}
	return GatewayConfig{
		PreferredProvider: mode,
		MaxRetries:        2,
		AllowMockFallback: allowFallback,
		Logger:            logger,
	}
}

// ProviderStatus describes a single provider slot in the Model Gateway for UI & AX introspection.
type ProviderStatus struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Model      string `json:"model"`
	Configured bool   `json:"configured"`
	Active     bool   `json:"active"`
	MaskedKey  string `json:"masked_key,omitempty"`
	EnvVar     string `json:"env_var"`
}

// RouterProvider dispatches completions across configured LLM providers with
// thread-safe runtime reconfiguration, retry/backoff, and deterministic offline fallback.
type RouterProvider struct {
	mu        sync.RWMutex
	cfg       GatewayConfig
	primary   services.LLMProvider
	fallbacks []services.LLMProvider
	statuses  []ProviderStatus
	maxRetry  int
	logger    services.Logger
}

func NewRouterProvider(cfg GatewayConfig) *RouterProvider {
	r := &RouterProvider{
		logger: cfg.Logger,
	}
	r.Reconfigure(cfg)
	return r
}

func maskSecret(secret string) string {
	s := strings.TrimSpace(secret)
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "••••••••"
	}
	return s[:4] + "••••••••" + s[len(s)-4:]
}

// ConfigSnapshot returns a copy of the active GatewayConfig.
func (r *RouterProvider) ConfigSnapshot() GatewayConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// Reconfigure hot-reloads provider credentials, model names, and routing priority at runtime.
func (r *RouterProvider) Reconfigure(cfg GatewayConfig) {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if strings.TrimSpace(cfg.PreferredProvider) == "" {
		cfg.PreferredProvider = "auto"
	}

	gemini := NewGeminiProvider(cfg.GeminiAPIKey, cfg.GeminiBaseURL, cfg.GeminiModel).WithHTTPClient(cfg.HTTPClient)
	anthropic := NewAnthropicProvider(cfg.AnthropicAPIKey, cfg.AnthropicBaseURL, cfg.AnthropicModel).WithHTTPClient(cfg.HTTPClient)
	openai := NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel).WithHTTPClient(cfg.HTTPClient)
	ollama := NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel).WithHTTPClient(cfg.HTTPClient)
	mock := NewMockProvider()

	ollamaReady := cfg.OllamaURL != "" || os.Getenv("OLLAMA_URL") != ""
	if !ollamaReady {
		if err := ollama.HealthCheck(context.Background()); err == nil {
			ollamaReady = true
		}
	}

	var chain []services.LLMProvider
	switch strings.ToLower(strings.TrimSpace(cfg.PreferredProvider)) {
	case "gemini", "google":
		chain = append(chain, gemini)
	case "anthropic", "claude":
		chain = append(chain, anthropic)
	case "openai":
		chain = append(chain, openai)
	case "ollama":
		chain = append(chain, ollama)
	case "mock", "offline":
		chain = append(chain, mock)
	default: // "auto" — prioritize Google Gemini first when GEMINI_API_KEY / GOOGLE_API_KEY is present
		if gemini.Configured() {
			chain = append(chain, gemini)
		}
		if anthropic.Configured() {
			chain = append(chain, anthropic)
		}
		if openai.Configured() {
			chain = append(chain, openai)
		}
		if ollamaReady {
			chain = append(chain, ollama)
		}
	}

	if len(chain) == 0 || (cfg.AllowMockFallback && chain[len(chain)-1].ProviderName() != "mock") {
		if len(chain) == 0 && cfg.Logger != nil {
			cfg.Logger.Info("no live LLM credentials detected; using deterministic offline synthesizer")
		}
		if len(chain) == 0 || cfg.AllowMockFallback {
			chain = append(chain, mock)
		}
	}

	activeID := chain[0].ProviderName()
	statuses := []ProviderStatus{
		{
			ID:         "gemini",
			Name:       "Google Gemini",
			Model:      gemini.ModelName(),
			Configured: gemini.Configured(),
			Active:     activeID == "gemini",
			MaskedKey:  maskSecret(gemini.apiKey),
			EnvVar:     "GEMINI_API_KEY",
		},
		{
			ID:         "anthropic",
			Name:       "Anthropic Claude",
			Model:      anthropic.model,
			Configured: anthropic.Configured(),
			Active:     activeID == "anthropic",
			MaskedKey:  maskSecret(anthropic.apiKey),
			EnvVar:     "ANTHROPIC_API_KEY",
		},
		{
			ID:         "openai",
			Name:       "OpenAI Compatible",
			Model:      openai.model,
			Configured: openai.Configured(),
			Active:     activeID == "openai",
			MaskedKey:  maskSecret(openai.apiKey),
			EnvVar:     "OPENAI_API_KEY",
		},
		{
			ID:         "ollama",
			Name:       "Local Ollama",
			Model:      ollama.model,
			Configured: ollamaReady,
			Active:     activeID == "ollama",
			MaskedKey:  ollama.baseURL,
			EnvVar:     "OLLAMA_URL",
		},
		{
			ID:         "mock",
			Name:       "Deterministic Go Synthesizer",
			Model:      "spec-aware-go-v2",
			Configured: true,
			Active:     activeID == "mock",
			EnvVar:     "Built-in Offline",
		},
	}

	r.mu.Lock()
	r.cfg = cfg
	r.primary = chain[0]
	r.fallbacks = chain[1:]
	r.statuses = statuses
	r.maxRetry = cfg.MaxRetries
	if cfg.Logger != nil {
		r.logger = cfg.Logger
	}
	r.mu.Unlock()
}

func (r *RouterProvider) ProviderStatuses() []ProviderStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderStatus, len(r.statuses))
	copy(out, r.statuses)
	return out
}

func (r *RouterProvider) PreferredMode() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg.PreferredProvider
}

func (r *RouterProvider) ProviderName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary.ProviderName()
}

func (r *RouterProvider) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	p := r.primary
	r.mu.RUnlock()
	return p.HealthCheck(ctx)
}

func (r *RouterProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return r.Complete(ctx, req)
}

func (r *RouterProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil LLMRequest")
	}

	tracer := otel.Tracer("asf/llm")
	ctx, span := tracer.Start(ctx, "llm.Complete")
	defer span.End()

	r.mu.RLock()
	providers := append([]services.LLMProvider{r.primary}, r.fallbacks...)
	maxRetry := r.maxRetry
	logger := r.logger
	r.mu.RUnlock()

	var lastErr error
	for _, provider := range providers {
		resp, err := r.callWithRetry(ctx, provider, req, maxRetry)
		if err == nil {
			span.SetAttributes(
				attribute.String("llm.provider", resp.Provider),
				attribute.String("llm.model", resp.Model),
				attribute.Int("llm.tokens", resp.Tokens),
			)
			return resp, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if logger != nil {
			logger.Warn("LLM provider failed; trying next fallback if available",
				"provider", provider.ProviderName(),
				"error", err.Error(),
			)
		}
	}

	return nil, lastErr
}

func (r *RouterProvider) callWithRetry(ctx context.Context, provider services.LLMProvider, req *services.LLMRequest, maxRetry int) (*services.LLMResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetry; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := provider.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		var statusErr *HTTPStatusError
		if errors.As(err, &statusErr) && !statusErr.Transient() {
			return nil, err
		}
	}
	return nil, lastErr
}
