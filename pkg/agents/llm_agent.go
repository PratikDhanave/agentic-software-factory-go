package agents

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/access"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type LLMAgent struct {
	name       string
	mcp        *access.MCPToolHandler
	ollama     *llm.OllamaClient
	maxSteps   int
}

func NewLLMAgent(name string) *LLMAgent {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "mistral"
	}

	return &LLMAgent{
		name:     name,
		mcp:      access.NewMCPToolHandler(os.Getenv("GITHUB_TOKEN")),
		ollama:   llm.NewOllamaClient(ollamaURL, ollamaModel),
		maxSteps: 10,
	}
}

type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Message struct {
	Role    string        `json:"role"`
	Content interface{}   `json:"content"`
}

// ExecuteTask runs agent reasoning loop with Ollama
func (a *LLMAgent) ExecuteTask(task string) (string, error) {
	log.Printf("[LLMAgent] Task: %s", task)

	// Try Ollama first
	if err := a.ollama.Health(); err == nil {
		return a.ExecuteTaskWithOllama(task)
	}

	log.Printf("[LLMAgent] Ollama not available, using reasoning mode")
	return a.ExecuteTaskWithReasoning(task)
}

// ExecuteTaskWithOllama uses Ollama for reasoning
func (a *LLMAgent) ExecuteTaskWithOllama(task string) (string, error) {
	log.Printf("[LLMAgent] Using Ollama for analysis")

	prompt := fmt.Sprintf(`You are a GitHub automation agent. Your task: "%s"

Analyze this and determine:
1. Which GitHub tools are needed
2. The order to call them
3. Expected outcomes

Be concise.`, task)

	analysis, err := a.ollama.Analyze(prompt)
	if err != nil {
		log.Printf("[LLMAgent] Ollama error: %v, falling back to reasoning", err)
		return a.ExecuteTaskWithReasoning(task)
	}

	response := map[string]interface{}{
		"agent":    a.name,
		"task":     task,
		"mode":     "ollama",
		"analysis": analysis,
		"tools":    a.getToolNames(),
	}

	result, _ := json.MarshalIndent(response, "", "  ")
	return string(result), nil
}

// ExecuteTaskWithReasoning provides non-LLM reasoning
func (a *LLMAgent) ExecuteTaskWithReasoning(task string) (string, error) {
	log.Printf("[LLMAgent] Reasoning mode (no ANTHROPIC_API_KEY)")

	// Pattern-based tool selection
	toolSuggestions := a.suggestTools(task)

	response := map[string]interface{}{
		"agent":               a.name,
		"task":                task,
		"mode":                "reasoning",
		"suggested_tools":     toolSuggestions,
		"available_tools":     a.getToolNames(),
		"reasoning_summary":   fmt.Sprintf("Task requires: %v tools based on keywords", len(toolSuggestions)),
	}

	result, _ := json.MarshalIndent(response, "", "  ")
	return string(result), nil
}

// suggestTools uses pattern matching to suggest appropriate tools
func (a *LLMAgent) suggestTools(task string) []string {
	var suggested []string

	// Pattern matching
	if contains(task, "get", "fetch", "read") {
		suggested = append(suggested, "get_issue")
	}
	if contains(task, "create", "new") {
		suggested = append(suggested, "create_issue")
	}
	if contains(task, "label", "tag") {
		suggested = append(suggested, "add_labels")
	}
	if contains(task, "comment", "reply", "respond") {
		suggested = append(suggested, "post_comment")
	}
	if contains(task, "pr", "pull request", "merge") {
		suggested = append(suggested, "create_pr")
	}

	if len(suggested) == 0 {
		suggested = append(suggested, "get_issue") // Default
	}

	return suggested
}

// CallToolBasedOnReasoning executes a tool decision
func (a *LLMAgent) CallToolBasedOnReasoning(toolName string, params map[string]interface{}) (interface{}, error) {
	log.Printf("[LLMAgent] Calling tool: %s", toolName)

	toolCall := access.GitHubToolCall{
		Tool:   toolName,
		Params: params,
	}

	result := a.mcp.ExecuteTool(toolCall)
	return result, nil
}

func (a *LLMAgent) getToolNames() []string {
	return []string{
		"get_issue",
		"create_issue",
		"add_labels",
		"post_comment",
		"create_pr",
	}
}

func contains(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if len(s) > 0 && len(substr) > 0 {
			// Simple substring check (case-insensitive would be better)
			for i := 0; i < len(s)-len(substr)+1; i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
