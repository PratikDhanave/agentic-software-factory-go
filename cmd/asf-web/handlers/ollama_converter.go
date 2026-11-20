package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const ollamaEndpoint = "http://localhost:11434/api/generate"
const coderModel = "qwen2.5-coder:14b"

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	Context    []int  `json:"context,omitempty"`
	TotalDuration int64 `json:"total_duration,omitempty"`
	LoadDuration int64 `json:"load_duration,omitempty"`
	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount int `json:"eval_count,omitempty"`
	EvalDuration int64 `json:"eval_duration,omitempty"`
}

func convertCodeWithOllama(sourceLanguage, sourceCode string) (string, error) {
	prompt := buildConversionPrompt(sourceLanguage, sourceCode)

	reqBody := OllamaRequest{
		Model:  coderModel,
		Prompt: prompt,
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(ollamaEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ollamaResp OllamaResponse
	respBody, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(respBody, &ollamaResp)

	flutterCode := extractFlutterCode(ollamaResp.Response)
	return flutterCode, nil
}

func buildConversionPrompt(sourceLanguage, sourceCode string) string {
	langDesc := "iOS/Swift"
	if sourceLanguage == "kotlin" {
		langDesc = "Android/Kotlin"
	}

	prompt := `You are an expert code converter. Your task is to convert the following ` + langDesc + ` code to equivalent Dart/Flutter code.

## Rules:
1. Convert all UI components to Flutter widgets
2. Use Material Design when applicable
3. Keep the same functionality and logic
4. Use Dart idioms and best practices
5. Include necessary imports
6. Return ONLY the converted Dart code without explanations or markdown

## ` + langDesc + ` Code:
` + sourceCode + `

## Converted Flutter/Dart Code:
Please provide the equivalent Flutter code:`

	return prompt
}

func extractFlutterCode(response string) string {
	// Remove markdown code fences if present
	response = strings.TrimSpace(response)

	// Remove opening fence
	if strings.HasPrefix(response, "```dart") {
		response = strings.TrimPrefix(response, "```dart")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	// Remove closing fence
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

// Test Ollama connection
func isOllamaAvailable() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
