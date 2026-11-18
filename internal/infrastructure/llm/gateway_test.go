package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type handlerTransport struct {
	handler http.HandlerFunc
}

func (t handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.handler(rec, req)
	return rec.Result(), nil
}

func clientForHandler(h http.HandlerFunc) *http.Client {
	return &http.Client{Transport: handlerTransport{handler: h}}
}

func TestAnthropicOpenAIOllamaAndMockProviders(t *testing.T) {
	ctx := context.Background()

	// 1. Anthropic HTTP contract
	anthropicClient := clientForHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected anthropic path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "sk-ant-test" {
			t.Errorf("missing x-api-key header")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]string{
				{"type": "text", "text": `{"choice":"ship","confidence":0.94}`},
			},
			"usage": map[string]int{"input_tokens": 20, "output_tokens": 12},
		})
	})

	ap := NewAnthropicProvider("sk-ant-test", "https://api.anthropic.com", "claude-sonnet-4-20250514").WithHTTPClient(anthropicClient)
	aResp, err := ap.Complete(ctx, &services.LLMRequest{Prompt: "decide", JsonMode: true})
	if err != nil {
		t.Fatalf("AnthropicProvider failed: %v", err)
	}
	if aResp.Provider != "anthropic" || aResp.Tokens != 32 {
		t.Fatalf("unexpected anthropic response: %+v", aResp)
	}

	// 2. OpenAI-compatible HTTP contract
	openaiClient := clientForHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected openai path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model": "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "hello from openai"}},
			},
			"usage": map[string]int{"total_tokens": 18},
		})
	})

	op := NewOpenAIProvider("sk-oai-test", "https://api.openai.com/v1", "gpt-4o-mini").WithHTTPClient(openaiClient)
	oResp, err := op.Complete(ctx, &services.LLMRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("OpenAIProvider failed: %v", err)
	}
	if oResp.Content != "hello from openai" || oResp.Tokens != 18 {
		t.Fatalf("unexpected openai response: %+v", oResp)
	}

	// 3. Ollama HTTP contract
	ollamaClient := clientForHandler(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model":             "mistral",
			"response":          "hello from ollama",
			"prompt_eval_count": 7,
			"eval_count":        5,
		})
	})

	olp := NewOllamaProvider("http://localhost:11434", "mistral").WithHTTPClient(ollamaClient)
	olResp, err := olp.Complete(ctx, &services.LLMRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("OllamaProvider failed: %v", err)
	}
	if olResp.Content != "hello from ollama" || olResp.Tokens != 12 {
		t.Fatalf("unexpected ollama response: %+v", olResp)
	}

	// 4. Router fallback to MockProvider when primary fails with 401
	failClient := clientForHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	})

	router := NewRouterProvider(GatewayConfig{
		PreferredProvider: "anthropic",
		AnthropicAPIKey:   "bad-key",
		HTTPClient:        failClient,
		MaxRetries:        0,
		AllowMockFallback: true,
	})
	fallbackResp, err := router.Complete(ctx, &services.LLMRequest{
		Prompt:   `Provide JSON with "choice" and "alternatives"`,
		JsonMode: true,
	})
	if err != nil {
		t.Fatalf("expected RouterProvider fallback to mock, got error: %v", err)
	}
	if fallbackResp.Provider != "mock" {
		t.Fatalf("expected mock fallback provider, got %s", fallbackResp.Provider)
	}

	// 5. ExtractJSONBlock fenced parsing
	fenced := "Sure!\n```json\n{\"ok\": true}\n```\nDone."
	if got := services.ExtractJSONBlock(fenced); got != `{"ok": true}` {
		t.Fatalf("ExtractJSONBlock = %q, want {\"ok\": true}", got)
	}
}
