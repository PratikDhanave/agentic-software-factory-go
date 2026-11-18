package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client is the prompt analysis contract shared across pkg/codegen, pkg/review,
// pkg/analysis, pkg/docs, and pkg/agents.
type Client interface {
	Analyze(prompt string) (string, error)
}

type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
	}
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "mistral"
		}
	}
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (o *OllamaClient) Analyze(prompt string) (string, error) {
	if strings.EqualFold(os.Getenv("ASF_LLM_PROVIDER"), "mock") {
		return NewDeterministicClient().Analyze(prompt)
	}

	log.Printf("[Ollama] Model: %s", o.model)

	reqBody := OllamaRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/api/generate", o.baseURL)
	resp, err := o.client.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("Ollama not reachable at %s: %w", o.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama error %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

func (o *OllamaClient) Health() error {
	url := fmt.Sprintf("%s/api/tags", o.baseURL)
	resp, err := o.client.Get(url)
	if err != nil {
		return fmt.Errorf("Ollama not reachable: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("[Ollama] Connected to %s", o.baseURL)
	return nil
}

// DeterministicClient provides offline, schema-valid completions for tests and
// zero-key local execution.
type DeterministicClient struct{}

func NewDeterministicClient() *DeterministicClient {
	return &DeterministicClient{}
}

func (d *DeterministicClient) Analyze(prompt string) (string, error) {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "\"findings\"") && strings.Contains(lower, "\"score\""):
		return `{"score":88,"findings":[],"summary":"Code satisfies quality, security, and readability checks."}`, nil
	case strings.Contains(lower, "\"bugs\"") && strings.Contains(lower, "\"critical_count\""):
		return `{"bugs":[],"total_issues":0,"critical_count":0,"summary":"No critical bugs found."}`, nil
	case strings.Contains(lower, "\"has_critical\""):
		return `{"has_critical":false,"issues":[]}`, nil
	case strings.Contains(lower, "\"valid\"") && strings.Contains(lower, "\"score\""):
		return `{"valid":true,"issues":[],"score":90}`, nil
	case strings.Contains(lower, "available tools:") && strings.Contains(lower, "\"done\""):
		return `{"done":true,"output":"Task completed deterministically."}`, nil
	case strings.Contains(lower, "write comprehensive tests"):
		return "```go\npackage generated\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif got := Add(2, 3); got != 5 {\n\t\tt.Fatalf(\"got %d, want 5\", got)\n\t}\n}\n```", nil
	case strings.Contains(lower, "readme.md"):
		return "# Generated Module\n\n## Overview\nProduction-ready generated Go module.\n", nil
	default:
		return "```go\npackage generated\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n```", nil
	}
}
