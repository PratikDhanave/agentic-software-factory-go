package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== Specification Engineering Demo (Chapter 4) ===")
	fmt.Println("Demonstrating machine-readable specifications for agent intent shaping")
	fmt.Println()

	specEngine := spec.NewSpecEngine(filepath.Join(workingDir, "specs"))

	// Load all specs
	specs, err := specEngine.LoadAllSpecs()
	if err != nil {
		fmt.Printf("Error loading specs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d specifications:\n\n", len(specs))

	for id, s := range specs {
		fmt.Printf("=== %s: %s (v%s) ===\n", id, s.Title, s.Version)
		fmt.Printf("Description: %s\n", s.Description)
		fmt.Printf("Goal: %s\n", s.Intent.Goal)
		fmt.Printf("Constraints: %d\n", len(s.Constraints))
		fmt.Printf("Inputs: %d\n", len(s.Inputs))
		fmt.Printf("Steps: %d\n", len(s.Steps))
		fmt.Println()

		// Validate
		errors := specEngine.ValidateSpec(s)
		if len(errors) > 0 {
			fmt.Println("Validation errors:")
			for _, err := range errors {
				fmt.Printf("  - %s\n", err)
			}
		} else {
			fmt.Println("✓ Specification is valid")
		}
		fmt.Println()

		// Generate prompt
		prompt := specEngine.GeneratePrompt(s)
		fmt.Println("--- Generated Agent Prompt ---")
		fmt.Println(prompt)
		fmt.Println()
	}

	// Demo: Create a new spec template
	fmt.Println("=== CREATING NEW SPEC TEMPLATE ===")
	templatePath := filepath.Join(workingDir, "specs", "demo-spec.yaml")
	if err := spec.WriteSpecTemplate(templatePath); err != nil {
		fmt.Printf("Error writing template: %v\n", err)
	} else {
		fmt.Printf("Template written to %s\n", templatePath)
	}

	// Demo: Validator
	if len(specs) > 0 {
		for _, s := range specs {
			validator := spec.NewSpecValidator(s)
			fmt.Printf("\n=== VALIDATOR FOR %s ===\n", s.ID)
			fmt.Println("Would validate agent outputs against specification")
		}
	}
}