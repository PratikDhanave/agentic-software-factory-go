package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
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
	case "init":
		initProject(workingDir)
	case "context":
		runContextEngine(workingDir)
	case "spec":
		if len(os.Args) < 3 {
			fmt.Println("Usage: asf spec <load|validate|generate|template> [spec-id]")
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
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ASF - Agentic Software Factory CLI

Usage: asf <command> [args...]

Commands:
  init              Initialize a new ASF project structure
  context           Load and display project context
  spec              Specification management
    load <spec-id>    Load and display a specification
    validate <spec-id> Validate a specification
    generate <spec-id> Generate agent prompt from specification
    template [path]   Write specification template
  verify            Run verification (build, test, lint)
  harness           Complete harness operations
    run <spec-id>     Run complete harness for specification
    status            Show harness status
  factory           Factory orchestration
    start             Start the factory
    stop              Stop the factory
    status            Show factory status
    submit <type> <spec-id> Submit a task to the factory
  help              Show this help

Examples:
  asf init
  asf spec template specs/my-spec.yaml
  asf spec generate spec-001
  asf verify
  asf harness run spec-001
  asf factory start
  asf factory submit generate spec-001
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
	engine := context.NewContextEngine(workingDir)

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
	formatted := context.FormatForAgent(items)
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

func runFactory(workingDir string, args []string) {
	subcommand := args[0]
	ctx := context.Background()

	switch subcommand {
	case "start":
		fo := factory.NewFactoryOrchestrator(workingDir, 4)
		if err := fo.Start(ctx); err != nil {
			fmt.Printf("Factory start error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Factory started with 4 workers")
		fmt.Println("Press Ctrl+C to stop")
		select {} // Wait forever

	case "status":
		fo := factory.NewFactoryOrchestrator(workingDir, 4)
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

		fo := factory.NewFactoryOrchestrator(workingDir, 4)
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