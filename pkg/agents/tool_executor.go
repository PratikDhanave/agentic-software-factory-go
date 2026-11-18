package agents

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/access"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type ToolExecutor struct {
	ollama        llm.Client
	mcp           *access.MCPToolHandler
	maxIterations int
}

type ExecutionResult struct {
	Success    bool          `json:"success"`
	FinalOutput string       `json:"final_output"`
	ToolCalls  []ToolCall    `json:"tool_calls"`
	Reasoning  string        `json:"reasoning"`
	Iterations int           `json:"iterations"`
}

type ToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
	Result interface{}            `json:"result"`
	Error  string                 `json:"error,omitempty"`
}

func NewToolExecutor(ollamaURL, model string) *ToolExecutor {
	return NewToolExecutorWithClient(llm.NewOllamaClient(ollamaURL, model))
}

func NewToolExecutorWithClient(client llm.Client) *ToolExecutor {
	return &ToolExecutor{
		ollama:        client,
		mcp:           access.NewMCPToolHandler(""),
		maxIterations: 5,
	}
}

// Execute runs an agentic loop: Ollama decides → tools execute → Ollama decides next
func (te *ToolExecutor) Execute(task string) (*ExecutionResult, error) {
	log.Printf("[ToolExecutor] Starting: %s", task)

	result := &ExecutionResult{
		ToolCalls: []ToolCall{},
	}

	// Get available tools description
	toolsDesc := te.describeTools()

	for iteration := 0; iteration < te.maxIterations; iteration++ {
		result.Iterations = iteration + 1

		// Ask Ollama what to do
		prompt := fmt.Sprintf(`Task: %s

Available tools:
%s

Based on the task, which tool should be called next and with what parameters?
Respond in JSON format:
{
  "tool": "tool_name",
  "params": {"key": "value"},
  "reasoning": "why this tool"
}

Or if done:
{
  "done": true,
  "output": "final result"
}`, task, toolsDesc)

		response, err := te.ollama.Analyze(prompt)
		if err != nil {
			return nil, fmt.Errorf("Ollama error: %w", err)
		}

		result.Reasoning = response

		// Parse response
		var decision map[string]interface{}
		if err := json.Unmarshal([]byte(response), &decision); err != nil {
			// Try extracting JSON from response
			jsonStr := extractJSON(response)
			if jsonStr == "" {
				return nil, fmt.Errorf("could not parse Ollama response")
			}
			if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
				return nil, fmt.Errorf("could not parse tool decision: %w", err)
			}
		}

		// Check if done
		if done, ok := decision["done"].(bool); ok && done {
			result.Success = true
			if output, ok := decision["output"].(string); ok {
				result.FinalOutput = output
			}
			log.Printf("[ToolExecutor] Completed in %d iterations", iteration+1)
			return result, nil
		}

		// Execute tool
		toolName, ok := decision["tool"].(string)
		if !ok {
			return nil, fmt.Errorf("no tool specified in decision")
		}

		params, ok := decision["params"].(map[string]interface{})
		if !ok {
			params = make(map[string]interface{})
		}

		log.Printf("[ToolExecutor] Iteration %d: Calling %s", iteration+1, toolName)

		// Call tool
		toolCall := ToolCall{
			Tool:   toolName,
			Params: params,
		}

		toolResult := te.mcp.ExecuteTool(access.GitHubToolCall{
			Tool:   toolName,
			Params: params,
		})

		if toolResult.Success {
			toolCall.Result = toolResult.Result
		} else {
			toolCall.Error = toolResult.Error
		}

		result.ToolCalls = append(result.ToolCalls, toolCall)
	}

	result.Success = false
	return result, fmt.Errorf("max iterations reached without completion")
}

func (te *ToolExecutor) describeTools() string {
	tools := te.mcp.GetToolDefinitions()
	var desc strings.Builder
	for i, tool := range tools {
		name := tool["name"].(string)
		description := tool["description"].(string)
		desc.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, name, description))
	}
	return desc.String()
}

func extractJSON(text string) string {
	// Find JSON object in response
	re := regexp.MustCompile(`\{[^{}]*\}`)
	matches := re.FindAllString(text, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return ""
}
