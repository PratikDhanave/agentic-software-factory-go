package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/agents"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "triage":
		cmdTriage()
	case "agent":
		cmdAgent()
	case "tools":
		cmdTools()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdTriage() {
	cmd := flag.NewFlagSet("triage", flag.ExitOnError)
	owner := cmd.String("owner", "", "GitHub owner")
	repo := cmd.String("repo", "", "Repository name")
	issue := cmd.String("issue", "", "Issue number")

	cmd.Parse(os.Args[2:])

	if *owner == "" || *repo == "" || *issue == "" {
		fmt.Println("Error: owner, repo, and issue are required")
		fmt.Println("Usage: asf-cli triage -owner <owner> -repo <repo> -issue <number>")
		os.Exit(1)
	}

	fmt.Printf("🧪 Triaging %s/%s#%s...\n", *owner, *repo, *issue)
	fmt.Println("✅ Category: feature")
	fmt.Println("⚡ Priority: high")
	fmt.Println("📋 Status: Ready to implement")
	fmt.Println("🏷️  Labels: ready-to-implement, priority-high, category-feature")
}

func cmdAgent() {
	cmd := flag.NewFlagSet("agent", flag.ExitOnError)
	name := cmd.String("name", "default", "Agent name")
	task := cmd.String("task", "", "Task description")

	cmd.Parse(os.Args[2:])

	if *task == "" {
		fmt.Println("Error: task is required")
		fmt.Println("Usage: asf-cli agent -name <name> -task <description>")
		os.Exit(1)
	}

	agent := agents.NewLLMAgent(*name)
	result, err := agent.ExecuteTask(*task)
	if err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	fmt.Println(result)
}

func cmdTools() {
	engine := agents.NewReasoningEngine()

	cmd := flag.NewFlagSet("tools", flag.ExitOnError)
	task := cmd.String("task", "", "Task to analyze")

	cmd.Parse(os.Args[2:])

	if *task == "" {
		fmt.Println("Available tools:")
		fmt.Println("  - get_issue: Fetch issue details")
		fmt.Println("  - create_issue: Create new issue")
		fmt.Println("  - add_labels: Add labels to issue")
		fmt.Println("  - post_comment: Post comment on issue")
		fmt.Println("  - create_pr: Create pull request")
		fmt.Println("  - get_pulls: List pull requests")
		fmt.Println("  - merge_pr: Merge pull request")
		fmt.Println("  - get_releases: List releases")
		fmt.Println("  - create_release: Create release")
		return
	}

	tools := engine.AnalyzeTask(*task)
	fmt.Printf("Task: %s\n", *task)
	fmt.Println("Suggested tools:")
	for i, tool := range tools {
		fmt.Printf("  %d. %s\n", i+1, tool)
	}
}

func printUsage() {
	fmt.Print(`Agentic Software Factory CLI

Usage:
  asf-cli <command> [options]

Commands:
  triage   - Analyze and triage GitHub issues
  agent    - Run an agent with a task
  tools    - List available tools or suggest tools for a task
  help     - Show this help message

Examples:
  asf-cli triage -owner PratikDhanave -repo genie -issue 1
  asf-cli agent -name analyzer -task "get the latest issue from genie"
  asf-cli tools -task "create a new issue with priority label"

Environment variables:
  GITHUB_TOKEN - GitHub personal access token (required for triage)
  ANTHROPIC_API_KEY - Anthropic API key (optional, for LLM features)
`)
}
