package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== Complete Harness Demo (Chapter 6) ===")
	fmt.Println("Integrating context, specifications, and verification loops")
	fmt.Println()

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <working-dir> <spec-id>")
		fmt.Println("Available specs: run 'asf spec list' to see")
		os.Exit(1)
	}

	specID := os.Args[2]

	ctx := context.Background()
	h := harness.NewCompleteHarness(workingDir)

	fmt.Printf("Executing complete harness for spec: %s\n\n", specID)

	result, err := h.Execute(ctx, specID)
	if err != nil {
		fmt.Printf("Harness error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Spec ID: %s\n", result.SpecID)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("Retries: %d\n", result.RetryCount)
	fmt.Printf("Context Items Loaded: %d\n", len(result.ContextItems))
	fmt.Println()

	if result.Success {
		fmt.Println("✓ Complete harness succeeded!")
		fmt.Println("All verification phases passed:")
		fmt.Println("  - Context loaded and injected")
		fmt.Println("  - Specification validated")
		fmt.Println("  - Build verification passed")
		fmt.Println("  - Test verification passed")
		fmt.Println("  - Lint verification passed")
	} else {
		fmt.Println("✗ Harness failed after max retries")
		fmt.Printf("Final output: %s\n", result.FinalOutput)
		if result.AgentFeedback != "" {
			fmt.Println("\n--- Agent Feedback for Retry ---")
			fmt.Println(result.AgentFeedback)
		}
		os.Exit(1)
	}
}