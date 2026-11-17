package agents

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/access"
)

type MCPAgent struct {
	name       string
	mcp        *access.MCPToolHandler
	systemPrompt string
}

func NewMCPAgent(name string) *MCPAgent {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Printf("[MCPAgent] Warning: GITHUB_TOKEN not set")
	}

	return &MCPAgent{
		name:    name,
		mcp:     access.NewMCPToolHandler(token),
		systemPrompt: fmt.Sprintf(`You are %s, an AI agent with access to GitHub tools.

Available tools:
- get_issue: Fetch issue details
- create_issue: Create a new issue
- add_labels: Add labels to issues
- post_comment: Post comments on issues
- create_pr: Create pull requests

When you need to use a tool, respond with:
{
  "action": "use_tool",
  "tool": "tool_name",
  "params": {...}
}

Otherwise respond with your analysis or next steps.`, name),
	}
}

// ExecuteTask runs an agent task with MCP tool access
func (a *MCPAgent) ExecuteTask(task string) (string, error) {
	log.Printf("[%s] Task: %s", a.name, task)

	// In a real implementation, this would call an LLM
	// For now, return available tools
	tools := a.mcp.GetToolDefinitions()
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")

	result := fmt.Sprintf(`Agent: %s
Task: %s

Available Tools:
%s

Ready to execute tasks using these tools.`, a.name, task, string(toolsJSON))

	log.Printf("[%s] Result: %s", a.name, result)
	return result, nil
}

// CallTool invokes an MCP tool
func (a *MCPAgent) CallTool(toolName string, params map[string]interface{}) (access.GitHubToolResult, error) {
	log.Printf("[%s] Calling tool: %s", a.name, toolName)

	toolCall := access.GitHubToolCall{
		Tool:   toolName,
		Params: params,
	}

	result := a.mcp.ExecuteTool(toolCall)
	return result, nil
}

// GetTools returns available tool definitions
func (a *MCPAgent) GetTools() []map[string]interface{} {
	return a.mcp.GetToolDefinitions()
}
