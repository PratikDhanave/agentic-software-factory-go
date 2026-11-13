package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== Software Factory Orchestration Demo (Chapter 11) ===")
	fmt.Println("Demonstrating Orchestrator-Worker-Validator distributed system")
	fmt.Println()

	ctx := context.Background()
	fo := factory.NewFactoryOrchestrator(workingDir, 4)

	fmt.Println("Starting factory with 4 workers...")
	if err := fo.Start(ctx); err != nil {
		fmt.Printf("Factory start error: %v\n", err)
		os.Exit(1)
	}

	// Submit some tasks
	tasks := []*factory.Task{
		{Type: "generate", SpecID: "spec-001", Priority: 1},
		{Type: "verify", SpecID: "spec-001", Priority: 2},
		{Type: "generate", SpecID: "spec-002", Priority: 1},
		{Type: "review", SpecID: "spec-001", Priority: 3},
		{Type: "refactor", SpecID: "spec-002", Priority: 2},
	}

	fmt.Println("\nSubmitting tasks...")
	for _, task := range tasks {
		if err := fo.SubmitTask(task); err != nil {
			fmt.Printf("Submit error: %v\n", err)
		} else {
			fmt.Printf("  Submitted: %s (%s)\n", task.ID[:8], task.Type)
		}
	}

	// Wait for processing
	fmt.Println("\nProcessing tasks (5 seconds)...")
	time.Sleep(5 * time.Second)

	// Check status
	status := fo.GetFactoryStatus()
	fmt.Println("\n=== Factory Status ===")
	for k, v := range status {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Check results
	fmt.Println("\n=== Task Results ===")
	for _, task := range tasks {
		result, ok := fo.GetTaskResult(task.ID)
		if ok {
			fmt.Printf("  %s (%s): %s\n", result.ID[:8], result.Type, result.Status)
			if result.Error != "" {
				fmt.Printf("    Error: %s\n", result.Error)
			}
		}
	}

	fmt.Println("\nStopping factory...")
	fo.Stop()
	fmt.Println("Factory stopped")

	fmt.Println("\n=== Architecture ===")
	fmt.Println("Orchestrator -> Workers (generation, verification, review, refactor)")
	fmt.Println("Tasks queued -> Workers execute -> Results stored")
	fmt.Println("Auto-scaling based on queue depth and worker utilization")
}