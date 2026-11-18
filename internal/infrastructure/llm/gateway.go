package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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
	PreferredProvider string // auto | anthropic | openai | ollama | mock
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

// RouterProvider dispatches completions across configured LLM providers with
// retry/backoff and optional deterministic offline fallback.
type RouterProvider struct {
	primary   services.LLMProvider
	fallbacks []services.LLMProvider
	maxRetry  int
	logger    services.Logger
}

func NewRouterProvider(cfg GatewayConfig) *RouterProvider {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}

	anthropic := NewAnthropicProvider(cfg.AnthropicAPIKey, cfg.AnthropicBaseURL, cfg.AnthropicModel).WithHTTPClient(cfg.HTTPClient)
	openai := NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel).WithHTTPClient(cfg.HTTPClient)
	ollama := NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel).WithHTTPClient(cfg.HTTPClient)
	mock := NewMockProvider()

	var chain []services.LLMProvider
	switch strings.ToLower(strings.TrimSpace(cfg.PreferredProvider)) {
	case "anthropic", "claude":
		chain = append(chain, anthropic)
	case "openai":
		chain = append(chain, openai)
	case "ollama":
		chain = append(chain, ollama)
	case "mock", "offline":
		chain = append(chain, mock)
	default: // "auto"
		if anthropic.Configured() {
			chain = append(chain, anthropic)
		}
		if openai.Configured() {
			chain = append(chain, openai)
		}
		// Include Ollama if explicitly configured via env or reachable locally
		if cfg.OllamaURL != "" || os.Getenv("OLLAMA_URL") != "" {
			chain = append(chain, ollama)
		} else if err := ollama.HealthCheck(context.Background()); err == nil {
			chain = append(chain, ollama)
		}
	}

	if len(chain) == 0 || (cfg.AllowMockFallback && chain[len(chain)-1].ProviderName() != "mock") {
		if len(chain) == 0 && cfg.Logger != nil {
			cfg.Logger.Info("no live LLM credentials detected; using deterministic offline mock provider")
		}
		if len(chain) == 0 || cfg.AllowMockFallback {
			chain = append(chain, mock)
		}
	}

	return &RouterProvider{
		primary:   chain[0],
		fallbacks: chain[1:],
		maxRetry:  cfg.MaxRetries,
		logger:    cfg.Logger,
	}
}

func (r *RouterProvider) ProviderName() string {
	return r.primary.ProviderName()
}

func (r *RouterProvider) HealthCheck(ctx context.Context) error {
	return r.primary.HealthCheck(ctx)
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

	providers := append([]services.LLMProvider{r.primary}, r.fallbacks...)
	var lastErr error

	for _, provider := range providers {
		resp, err := r.callWithRetry(ctx, provider, req)
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
		if r.logger != nil {
			r.logger.Warn("LLM provider failed; trying next fallback if available",
				"provider", provider.ProviderName(),
				"error", err.Error(),
			)
		}
	}

	return nil, lastErr
}

func (r *RouterProvider) callWithRetry(ctx context.Context, provider services.LLMProvider, req *services.LLMRequest) (*services.LLMResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
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
