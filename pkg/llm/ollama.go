package llm

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// Ollama support rides on the OpenAI-compatible wire format (Ollama serves
// it under /v1) with zero-config defaults: the standard local URL, and the
// first chat-capable installed model when ASF_MODEL is unset.

// ollamaBaseURL resolves the Ollama server root (no /v1 suffix) from
// ASF_LLM_BASE_URL, OLLAMA_HOST, or the standard local default.
func ollamaBaseURL() string {
	if b := os.Getenv("ASF_LLM_BASE_URL"); b != "" {
		return strings.TrimSuffix(strings.TrimRight(b, "/"), "/v1")
	}
	if h := os.Getenv("OLLAMA_HOST"); h != "" {
		if !strings.Contains(h, "://") {
			h = "http://" + h
		}
		return strings.TrimRight(h, "/")
	}
	return "http://localhost:11434"
}

// newOllama builds the Ollama provider, or returns nil when no usable model
// can be determined (ConfigError explains why).
func newOllama() Client {
	base := ollamaBaseURL()
	model := os.Getenv("ASF_MODEL")
	if model == "" {
		model = firstOllamaChatModel(base)
	}
	if model == "" {
		return nil
	}
	return &openAICompatClient{
		provider:  "ollama",
		baseURL:   base + "/v1",
		model:     model,
		maxTokens: 16000,
		http:      &http.Client{},
	}
}

// detectOllama returns an Ollama provider only when a server is actually
// reachable — used by FromEnv auto-detection so a stopped Ollama never
// captures the factory.
func detectOllama() Client {
	if !ollamaReachable(ollamaBaseURL()) {
		return nil
	}
	return newOllama()
}

func ollamaReachable(base string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// firstOllamaChatModel picks the first installed model that looks
// chat-capable (embedding models are skipped). Returns "" when the server
// is unreachable or has no usable model.
func firstOllamaChatModel(base string) string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/api/tags")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var tags struct {
		Models []struct {
			Name    string `json:"name"`
			Details struct {
				Family string `json:"family"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return ""
	}

	for _, m := range tags.Models {
		if strings.Contains(m.Name, "embed") || strings.Contains(m.Details.Family, "bert") {
			continue
		}
		return m.Name
	}
	return ""
}
