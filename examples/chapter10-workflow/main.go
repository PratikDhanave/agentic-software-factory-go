package main

import (
	"fmt"
	"os"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== Developer Workflow Demo (Chapter 10) ===")
	fmt.Println("Demonstrating the shift from hand work to head work")
	fmt.Println()

	fmt.Println("=== ASF CLI WORKFLOW ===")
	fmt.Println()
	fmt.Println("1. Initialize project:")
	fmt.Println("   asf init")
	fmt.Println()
	fmt.Println("2. Write specification:")
	fmt.Println("   asf spec template specs/my-feature.yaml")
	fmt.Println("   # Edit specs/my-feature.yaml")
	fmt.Println()
	fmt.Println("3. Validate specification:")
	fmt.Println("   asf spec validate my-feature")
	fmt.Println()
	fmt.Println("4. Generate agent prompt:")
	fmt.Println("   asf spec generate my-feature")
	fmt.Println()
	fmt.Println("5. Run verification:")
	fmt.Println("   asf verify")
	fmt.Println()
	fmt.Println("6. Run complete harness:")
	fmt.Println("   asf harness run my-feature")
	fmt.Println()

	fmt.Println("=== DAILY DEVELOPER LOOP ===")
	fmt.Println()
	fmt.Println("Morning:")
	fmt.Println("  - Pull latest changes")
	fmt.Println("  - Review agent-generated PRs from overnight")
	fmt.Println("  - Approve or request changes")
	fmt.Println()
	fmt.Println("Development:")
	fmt.Println("  - Write specs for new features")
	fmt.Println("  - Let agent implement from specs")
	fmt.Println("  - Run asf verify continuously")
	fmt.Println("  - Commit when harness passes")
	fmt.Println()
	fmt.Println("Evening:")
	fmt.Println("  - Push changes")
	fmt.Println("  - Factory runs overnight batch jobs")
	fmt.Println("  - Review results in morning")
	fmt.Println()

	fmt.Println("=== SPEC AUTHORING PATTERNS ===")
	fmt.Println()
	fmt.Println("Good specs are:")
	fmt.Println("  - Specific: Clear success criteria")
	fmt.Println("  - Constrained: Explicit anti-goals")
	fmt.Println("  - Testable: Verifiable outputs")
	fmt.Println("  - Composable: Reusable steps")
	fmt.Println()
	fmt.Println("Bad specs:")
	fmt.Println("  - Vague: 'Make it fast'")
	fmt.Println("  - Unconstrained: No boundaries")
	fmt.Println("  - Untestable: No clear outputs")
	fmt.Println("  - Monolithic: Can't be decomposed")
	fmt.Println()

	fmt.Println("=== PROMPT ENGINEERING FOR SPECS ===")
	fmt.Println()
	fmt.Println("Instead of: 'Build a user service'")
	fmt.Println("Write spec with:")
	fmt.Println("  - Goal: 'Implement user CRUD with email verification'")
	fmt.Println("  - Constraints: 'Must use PostgreSQL, must hash passwords'")
	fmt.Println("  - Anti-goals: 'No plain text passwords, no SQL injection'")
	fmt.Println("  - Steps: [create schema, implement repo, add handlers, add tests]")
	fmt.Println()

	fmt.Printf("Working directory: %s\n", workingDir)
	fmt.Println("Your specs live in specs/ directory")
}