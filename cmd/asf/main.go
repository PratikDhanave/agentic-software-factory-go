package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/bootstrap"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/stateregistry"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/agents"
	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/mcp"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	workingDir, _ := os.Getwd()

	switch command {
	case "build", "ship":
		runBuildPipeline(workingDir, os.Args[2:])
	case "init":
		initProject(workingDir)
	case "context":
		runContextEngine(workingDir)
	case "spec":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf spec <load|validate|generate|template|list> [spec-id]")
			os.Exit(1)
		}
		runSpecEngine(workingDir, os.Args[2:])
	case "verify":
		runVerification(workingDir)
	case "harness":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf harness <run|status> [spec-id]")
			os.Exit(1)
		}
		runHarness(workingDir, os.Args[2:])
	case "factory":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf factory <start|stop|status|submit> [args...]")
			os.Exit(1)
		}
		runFactory(workingDir, os.Args[2:])
	case "mcp":
		runMCPServer(workingDir, os.Args[2:])
	case "web":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf web <serve|generate> [args...]")
			os.Exit(1)
		}
		runWebCommand(workingDir, os.Args[2:])
	case "integration":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf integration <list|status> [args...]")
			os.Exit(1)
		}
		runIntegrationCommand(workingDir, os.Args[2:])
	case "triage":
		runTriageCommand(os.Args[2:])
	case "agent":
		runAgentCommand(os.Args[2:])
	case "tools":
		runToolsCommand(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// runWebCommand handles the web subcommand
func runWebCommand(workingDir string, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: asf web <serve|generate> [args...]")
		os.Exit(1)
	}

	switch args[0] {
	case "serve":
		if len(args) < 2 {
			fmt.Println("Usage: asf web serve [port]")
			os.Exit(1)
		}
		port := "8080"
		if len(args) >= 3 {
			port = args[2]
		}
		fmt.Printf("Starting ASF web server on port %s...\n", port)
		// TODO: Implement web server - serve the web UI from web/static and web/templates
	case "generate":
		if len(args) < 2 {
			fmt.Println("Usage: asf web generate [output-dir]")
			os.Exit(1)
		}
		outputDir := args[1]
		fmt.Printf("Generating web assets to %s...\n", outputDir)
		// TODO: Generate web assets from templates
	default:
		fmt.Printf("Unknown web subcommand: %s\n", args[0])
		fmt.Println("Available subcommands: serve, generate")
		os.Exit(1)
	}
}

// runIntegrationCommand handles the integration subcommand
func runIntegrationCommand(workingDir string, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: asf integration <list|status> [args...]")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		fmt.Println("Listing integrations...")
		// TODO: List registered integrations from persistence
	case "status":
		if len(args) < 2 {
			fmt.Println("Usage: asf integration status [integration-name]")
			os.Exit(1)
		}
		name := args[1]
		fmt.Printf("Checking status of integration: %s...\n", name)
		// TODO: Check status of specific integration
	default:
		fmt.Printf("Unknown integration subcommand: %s\n", args[0])
		fmt.Println("Available subcommands: list, status")
		os.Exit(1)
	}
}

func runTriageCommand(args []string) {
	cmd := flag.NewFlagSet("triage", flag.ExitOnError)
	owner := cmd.String("owner", "", "GitHub owner")
	repo := cmd.String("repo", "", "Repository name")
	issue := cmd.String("issue", "", "Issue number")
	_ = cmd.Parse(args)

	if *owner == "" || *repo == "" || *issue == "" {
		fmt.Println("Usage: asf triage -owner <owner> -repo <repo> -issue <number>")
		os.Exit(1)
	}

	fmt.Printf("Triaging %s/%s#%s...\n", *owner, *repo, *issue)
	agent := agents.NewLLMAgent("triage")
	result, err := agent.ExecuteTask(fmt.Sprintf("Triage issue #%s in %s/%s", *issue, *owner, *repo))
	if err != nil {
		fmt.Printf("Triage error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runAgentCommand(args []string) {
	cmd := flag.NewFlagSet("agent", flag.ExitOnError)
	name := cmd.String("name", "default", "Agent name")
	task := cmd.String("task", "", "Task description")
	_ = cmd.Parse(args)

	if *task == "" {
		fmt.Println("Usage: asf agent -name <name> -task <description>")
		os.Exit(1)
	}

	agent := agents.NewLLMAgent(*name)
	result, err := agent.ExecuteTask(*task)
	if err != nil {
		fmt.Printf("Agent error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func runToolsCommand(args []string) {
	engine := agents.NewReasoningEngine()
	cmd := flag.NewFlagSet("tools", flag.ExitOnError)
	task := cmd.String("task", "", "Task to analyze")
	_ = cmd.Parse(args)

	if *task == "" {
		fmt.Println("Available tools:")
		for _, t := range []string{"get_issue", "create_issue", "add_labels", "post_comment", "create_pr", "get_pulls", "merge_pr", "get_releases", "create_release"} {
			fmt.Printf("  - %s\n", t)
		}
		return
	}

	tools := engine.AnalyzeTask(*task)
	fmt.Printf("Suggested tools for %q:\n", *task)
	for i, tool := range tools {
		fmt.Printf("  %d. %s\n", i+1, tool)
	}
}

func printUsage() {
	fmt.Print(`ASF - Agentic Software Factory CLI

Usage: asf <command> [args...]

Commands:
  build             Run the 6-stage autonomous assembly line (spec -> agents -> sandbox -> verify -> judge)
  init              Initialize a new ASF project structure
  context           Load and display project context
  spec              Specification management
    load <spec-id>    Load and display a specification
    validate <spec-id> Validate a specification
    generate <spec-id> Generate agent prompt from specification
    template [path]   Write specification template
    list              List all specifications
  verify            Run verification (build, test, lint)
  harness           Complete harness operations
    run <spec-id>     Run complete harness for specification
    status            Show harness status
  factory           Factory orchestration
    start             Start the factory
    stop              Stop the factory
    status            Show factory status
    submit <type> <spec-id> Submit a task to the factory
    states [--json|--markdown] Print the declared state machines
  mcp               Start MCP server for headless API access
    start [addr]      Start MCP server (default: localhost:8080)
  web               Web UI and asset generation
    serve             Start web server to serve the UI
    generate          Generate web assets from templates
  integration       Integration adapters
    list              List registered integrations
    status            Check integration status
  help              Show this help

Examples:
  asf init
  asf spec template specs/my-spec.yaml
  asf spec generate spec-001
  asf verify
  asf harness run spec-001
  asf factory start
  asf factory submit generate spec-001
  asf mcp start
  asf web serve
  asf web generate ./web-output
  asf integration list
  asf integration status github
`)
}

func initProject(workingDir string) {
	dirs := []string{
		"docs",
		"specs",
		"internal",
		"pkg",
		"cmd",
		"examples",
		"diagrams",
		"scripts",
	}

	for _, dir := range dirs {
		path := filepath.Join(workingDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("Error creating %s: %v\n", dir, err)
			continue
		}
		fmt.Printf("Created: %s/\n", dir)
	}

	// Create example spec template
	specTemplate := spec.SpecTemplate
	templatePath := filepath.Join(workingDir, "specs", "example.yaml")
	if err := os.WriteFile(templatePath, []byte(specTemplate), 0644); err != nil {
		fmt.Printf("Error writing template: %v\n", err)
	} else {
		fmt.Printf("Created: specs/example.yaml\n")
	}

	// Create example ADR
	adrTemplate := `# ADR-001: Example Decision - APPROVED

## Context
This is an example Architectural Decision Record.

## Decision
We decided to use Go for the agentic software factory implementation.

## The Absolute Constraint
All code MUST be written in Go 1.23+.
All dependencies MUST be vendored or in go.mod.

## Consequences
- Type safety at compile time
- Native concurrency support
- Fast startup times
`
	adrPath := filepath.Join(workingDir, "docs", "ADR-001-Example.md")
	if err := os.WriteFile(adrPath, []byte(adrTemplate), 0644); err != nil {
		fmt.Printf("Error writing ADR: %v\n", err)
	} else {
		fmt.Printf("Created: docs/ADR-001-Example.md\n")
	}

	fmt.Println("\nProject initialized! Add your specs to specs/ and ADRs to docs/")
}

func runContextEngine(workingDir string) {
	ctx := context.Background()
	engine := asfContext.NewContextEngine(workingDir)

	items, err := engine.LoadContext(ctx)
	if err != nil {
		fmt.Printf("Error loading context: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d context items:\n\n", len(items))
	for _, item := range items {
		fmt.Printf("=== %s [%s] (priority: %d, ~%d tokens) ===\n",
			item.Path, item.Category, item.Priority, item.TokenEstimate)
		// Print first 200 chars
		preview := item.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Println(preview)
		fmt.Println()
	}

	// Show formatted for agent
	formatted := asfContext.FormatForAgent(items)
	fmt.Println("--- Formatted for Agent ---")
	fmt.Println(formatted)
}

func runSpecEngine(workingDir string, args []string) {
	subcommand := args[0]
	specEngine := spec.NewSpecEngine(filepath.Join(workingDir, "specs"))

	switch subcommand {
	case "load":
		if len(args) < 2 {
			fmt.Println("Usage: asf spec load <spec-id>")
			os.Exit(1)
		}
		specID := args[1]
		specs, _ := specEngine.LoadAllSpecs()
		s, ok := specs[specID]
		if !ok {
			fmt.Printf("Spec %s not found\n", specID)
			os.Exit(1)
		}
		printSpec(s)

	case "validate":
		if len(args) < 2 {
			fmt.Println("Usage: asf spec validate <spec-id>")
			os.Exit(1)
		}
		specID := args[1]
		specs, _ := specEngine.LoadAllSpecs()
		s, ok := specs[specID]
		if !ok {
			fmt.Printf("Spec %s not found\n", specID)
			os.Exit(1)
		}
		errors := specEngine.ValidateSpec(s)
		if len(errors) > 0 {
			fmt.Println("Validation errors:")
			for _, err := range errors {
				fmt.Printf("  - %s\n", err)
			}
			os.Exit(1)
		}
		fmt.Printf("Spec %s is valid!\n", specID)

	case "generate":
		if len(args) < 2 {
			fmt.Println("Usage: asf spec generate <spec-id>")
			os.Exit(1)
		}
		specID := args[1]
		specs, _ := specEngine.LoadAllSpecs()
		s, ok := specs[specID]
		if !ok {
			fmt.Printf("Spec %s not found\n", specID)
			os.Exit(1)
		}
		prompt := specEngine.GeneratePrompt(s)
		fmt.Println(prompt)

	case "template":
		path := "specs/new-spec.yaml"
		if len(args) >= 2 {
			path = args[1]
		}
		fullPath := filepath.Join(workingDir, path)
		if err := spec.WriteSpecTemplate(fullPath); err != nil {
			fmt.Printf("Error writing template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Template written to %s\n", fullPath)

	case "list":
		specs, _ := specEngine.LoadAllSpecs()
		fmt.Printf("Found %d specifications:\n", len(specs))
		for id, s := range specs {
			fmt.Printf("  %s: %s (v%s)\n", id, s.Title, s.Version)
		}

	default:
		fmt.Printf("Unknown spec subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func printSpec(s *spec.Specification) {
	fmt.Printf("ID: %s\n", s.ID)
	fmt.Printf("Version: %s\n", s.Version)
	fmt.Printf("Title: %s\n", s.Title)
	fmt.Printf("Description: %s\n", s.Description)
	fmt.Printf("\nIntent:\n  Goal: %s\n", s.Intent.Goal)
	fmt.Printf("  Context: %s\n", s.Intent.Context)
	fmt.Printf("  Success Criteria: %v\n", s.Intent.SuccessCriteria)
	fmt.Printf("\nConstraints (%d):\n", len(s.Constraints))
	for _, c := range s.Constraints {
		fmt.Printf("  [%s] %s: %s\n", c.ID, strings.ToUpper(c.Type), c.Description)
	}
	fmt.Printf("\nInputs (%d):\n", len(s.Inputs))
	for name, input := range s.Inputs {
		req := ""
		if input.Required {
			req = " (required)"
		}
		fmt.Printf("  %s%s: %s\n", name, req, input.Description)
	}
	fmt.Printf("\nSteps (%d):\n", len(s.Steps))
	for i, step := range s.Steps {
		deps := ""
		if len(step.DependsOn) > 0 {
			deps = fmt.Sprintf(" [depends on: %s]", strings.Join(step.DependsOn, ", "))
		}
		fmt.Printf("  %d. %s%s\n", i+1, step.Name, deps)
		fmt.Printf("     %s\n", step.Description)
		if step.Tool != "" {
			fmt.Printf("     Tool: %s\n", step.Tool)
		}
	}
}

func runVerification(workingDir string) {
	ctx := context.Background()
	verifier := verify.NewVerificationEngine(workingDir)

	fmt.Println("Running verification...")
	result, err := verifier.RunVerification(ctx, "")
	if err != nil {
		fmt.Printf("Verification error: %v\n", err)
	}

	if result.Success {
		fmt.Println("✓ All verification passed!")
	} else {
		fmt.Printf("✗ Verification failed with %d errors:\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  [%s] %s:%d - %s\n", strings.ToUpper(err.Type), err.File, err.Line, err.Message)
		}
		os.Exit(1)
	}
}

func runHarness(workingDir string, args []string) {
	subcommand := args[0]

	switch subcommand {
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: asf harness run <spec-id>")
			os.Exit(1)
		}
		specID := args[1]
		ctx := context.Background()
		h := harness.NewCompleteHarness(workingDir)

		result, err := h.Execute(ctx, specID)
		if err != nil {
			fmt.Printf("Harness error: %v\n", err)
			os.Exit(1)
		}

		if result.Success {
			fmt.Printf("✓ Harness completed successfully for %s\n", specID)
			fmt.Printf("  Duration: %v\n", result.TotalDuration)
			fmt.Printf("  Retries: %d\n", result.RetryCount)
		} else {
			fmt.Printf("✗ Harness failed for %s\n", specID)
			fmt.Printf("  Duration: %v\n", result.TotalDuration)
			fmt.Printf("  Retries: %d\n", result.RetryCount)
			fmt.Printf("  Final: %s\n", result.FinalOutput)
			os.Exit(1)
		}

	case "status":
		fmt.Println("Harness status: ready")
		fmt.Println("Working directory:", workingDir)

	default:
		fmt.Printf("Unknown harness subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// stateMachines lists every explicit state machine in the system.
func stateMachines() []*state.Machine {
	return stateregistry.All()
}

func runBuildPipeline(workingDir string, args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	specID := fs.String("spec", "", "Existing spec ID in specs/")
	desc := fs.String("desc", "", "Detailed requirement description")
	promote := fs.Bool("promote", false, "Promote verified Go package into generated/<slug>/")
	_ = fs.Parse(args)

	title := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if title == "" && *specID == "" {
		fmt.Println("Usage: asf build [-spec <spec-id>] [-desc <details>] [-promote] \"<engineering requirement>\"")
		os.Exit(1)
	}
	if *desc == "" {
		*desc = title
	}

	c, err := bootstrap.NewContainer(workingDir)
	if err != nil {
		fmt.Printf("Bootstrap error: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	// Ensure CLI runs autonomously unless ASF_PHASE=crawl is explicitly requested
	if os.Getenv("ASF_PHASE") == "" {
		_ = c.SetPhase("walk")
	}

	fmt.Printf("▶ Starting Software Factory Assembly Line (LLM: %s, Phase: %s)...\n", c.LLMGateway.ProviderName(), c.Phase())
	rec, err := c.Pipeline.StartRun(context.Background(), application.FactoryRunInput{
		Title:       title,
		Description: *desc,
		SpecID:      *specID,
		Phase:       c.Phase(),
	}, true)
	if err != nil {
		fmt.Printf("✗ Factory run failed: %v\n", err)
		os.Exit(1)
	}

	if tlRaw, ok := rec.Payload["timeline"]; ok {
		b, _ := json.Marshal(tlRaw)
		var timeline []application.StageTimelineEntry
		_ = json.Unmarshal(b, &timeline)
		for _, step := range timeline {
			fmt.Printf("  [%s] %-20s %s\n", strings.ToUpper(step.Status), step.Stage, step.Message)
		}
	}

	if rec.Status == "failed" {
		fmt.Printf("\n✗ Run %s failed: %s\n", rec.ID, rec.Error)
		os.Exit(1)
	}
	if rec.Status == "awaiting_approval" {
		fmt.Printf("\n⏸ Run %s paused at Human Governance Gate (Session: %s). Approve in UI or set ASF_PHASE=walk.\n", rec.ID, rec.SessionID)
		return
	}

	if *promote {
		if promoted, pErr := c.Pipeline.PromoteRun(context.Background(), rec.ID, ""); pErr == nil {
			rec = promoted
			fmt.Printf("  [PROMOTED] Exported verified package to %s\n", rec.PromotedDir)
		}
	}

	fmt.Printf("\n✓ Software Factory Run %s COMPLETED!\n", rec.ID)
	fmt.Printf("  Spec ID       : %s\n", rec.SpecID)
	fmt.Printf("  Workspace Dir : %s\n", rec.WorkspaceDir)
	if rec.PromotedDir != "" {
		fmt.Printf("  Promoted Dir  : %s\n", rec.PromotedDir)
	}
	if files, ok := rec.Payload["files_written"]; ok {
		fmt.Printf("  Files Written : %v\n", files)
	}
}

// runFactoryStates prints the declared state machines. Formats:
//
//	asf factory states            human-readable Mermaid diagrams
//	asf factory states --json     machine-readable JSON (for agents/tools)
//	asf factory states --markdown full Markdown document
func runFactoryStates(args []string) {
	format := "mermaid"
	if len(args) > 0 {
		switch args[0] {
		case "--json":
			format = "json"
		case "--markdown":
			format = "markdown"
		}
	}

	machines := stateMachines()

	switch format {
	case "json":
		exports := make([]state.Export, 0, len(machines))
		for _, m := range machines {
			exports = append(exports, m.Export())
		}
		data, _ := json.MarshalIndent(map[string]interface{}{"machines": exports}, "", "  ")
		fmt.Println(string(data))

	case "markdown":
		fmt.Println("# State Machines")
		fmt.Println()
		fmt.Println("Every lifecycle in the Agentic Software Factory is an explicit, declared")
		fmt.Println("state machine (see `pkg/state`). This document is generated — regenerate")
		fmt.Println("with `asf factory states --markdown > docs/STATE_MACHINES.md`.")
		fmt.Println()
		fmt.Println("The same declarations are served at runtime from `GET /api/state-machines`")
		fmt.Println("(asf-web) and are mirrored for the dashboard in `web-ui/src/lib/stateMachine.ts`.")
		for _, m := range machines {
			fmt.Printf("\n## %s\n\n", m.Name())
			fmt.Printf("Initial state: `%s`\n\n", m.Initial())
			fmt.Println("```mermaid")
			fmt.Print(m.Mermaid())
			fmt.Println("```")
		}

	default:
		for _, m := range machines {
			fmt.Printf("=== %s (initial: %s) ===\n", m.Name(), m.Initial())
			fmt.Print(m.Mermaid())
			fmt.Println()
		}
	}
}

func runFactory(workingDir string, args []string) {
	subcommand := args[0]
	ctx := context.Background()

	switch subcommand {
	case "states":
		runFactoryStates(args[1:])
		return
	case "start":
		fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
		if err != nil {
			fmt.Printf("Factory start error: %v\n", err)
			os.Exit(1)
		}
		if err := fo.Start(ctx); err != nil {
			fmt.Printf("Factory start error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Factory started with 4 workers")
		fmt.Println("Press Ctrl+C to stop")
		select {} // Wait forever

	case "status":
		fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
		if err != nil {
			fmt.Printf("Factory status error: %v\n", err)
			os.Exit(1)
		}
		status := fo.GetFactoryStatus()
		fmt.Println("Factory Status:")
		for k, v := range status {
			fmt.Printf("  %s: %v\n", k, v)
		}

	case "submit":
		if len(args) < 3 {
			fmt.Println("Usage: asf factory submit <type> <spec-id>")
			os.Exit(1)
		}
		taskType := args[1]
		specID := args[2]

		fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
		if err != nil {
			fmt.Printf("Factory error: %v\n", err)
			os.Exit(1)
		}
		task := &factory.Task{
			Type:     taskType,
			SpecID:   specID,
			Priority: 1,
		}

		if err := fo.SubmitTask(task); err != nil {
			fmt.Printf("Submit error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Task submitted: %s\n", task.ID)

	default:
		fmt.Printf("Unknown factory subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func runMCPServer(workingDir string, args []string) {
	// Accept both `asf mcp start [addr]` and `asf mcp [addr]`.
	if len(args) > 0 && args[0] == "start" {
		args = args[1:]
	}
	addr := "localhost:8080"
	if len(args) > 0 && args[0] != "" {
		addr = args[0]
	}

	// Load factory definition
	factoryDefPath, err := factorydef.FindFactoryDefinition(workingDir)
	if err != nil {
		fmt.Printf("Warning: no factory definition found: %v\n", err)
	}

	var factoryDef *factorydef.FactoryDefinition
	if factoryDefPath != "" {
		factoryDef, err = factorydef.LoadFactoryDefinition(factoryDefPath)
		if err != nil {
			fmt.Printf("Warning: failed to load factory definition: %v\n", err)
		}
	}

	// Create factory
	fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
	if err != nil {
		fmt.Printf("Factory error: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	mcpServer := mcp.NewMCPServer(fo, factoryDef).WithStateMachines(stateregistry.All())

	fmt.Printf("Starting MCP server on %s...\n", addr)
	fmt.Println("Available tools: submit_task, get_task_result, get_factory_status, list_specs, get_spec, list_agents, run_verification")
	fmt.Println("Press Ctrl+C to stop")

	// Start server in background
	go func() {
		if err := mcpServer.Start(addr); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()

	// Wait for interrupt
	select {}
}