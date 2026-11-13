package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== AI-TDD Verification Loop Demo (Chapter 5) ===")
	fmt.Println("Demonstrating automated verification of agent-generated code")
	fmt.Println()

	ctx := context.Background()
	verifier := verify.NewVerificationEngine(workingDir)

	fmt.Println("Running verification pipeline...")
	fmt.Println("Phase 1: Build verification (go build ./...)")
	fmt.Println("Phase 2: Test verification (go test -v ./...)")
	fmt.Println("Phase 3: Lint verification (golangci-lint run ./...)")
	fmt.Println()

	result, err := verifier.RunVerification(ctx, "")
	if err != nil {
		fmt.Printf("Verification error: %v\n", err)
	}

	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Retries: %d\n", result.RetryCount)
	fmt.Println()

	if result.Success {
		fmt.Println("✓ All verification passed!")
		fmt.Println()
		fmt.Println("Build output:")
		fmt.Println(result.BuildOutput)
		fmt.Println("Test output:")
		fmt.Println(result.TestOutput)
		fmt.Println("Lint output:")
		fmt.Println(result.LintOutput)
	} else {
		fmt.Printf("✗ Verification failed with %d errors:\n\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("%d. [%s] %s:%d\n", i+1, err.Type, err.File, err.Line)
			fmt.Printf("   %s\n", err.Message)
			if err.Suggestion != "" {
				fmt.Printf("   Suggestion: %s\n", err.Suggestion)
			}
			fmt.Println()
		}

		// Show fix prompt
		fmt.Println("=== GENERATED FIX PROMPT FOR AGENT ===")
		fixPrompt := verify.GenerateFixPrompt(result, "Original specification would be here")
		fmt.Println(fixPrompt)
		os.Exit(1)
	}
}