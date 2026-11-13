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

	fmt.Println("=== Team Operating Model Demo (Chapter 9) ===")
	fmt.Println("Demonstrating agent integration into team workflows")
	fmt.Println()

	fmt.Println("=== CODE REVIEW INTEGRATION ===")
	fmt.Println()
	fmt.Println("Git Hook: .git/hooks/pre-push")
	fmt.Println(`#!/bin/bash
# Auto-run ASF verification before push
asf verify || exit 1
echo "✓ Pre-push verification passed"`)
	fmt.Println()

	fmt.Println("GitHub Actions: .github/workflows/asf-review.yml")
	fmt.Println(`name: ASF Code Review
on: [pull_request]
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go install github.com/PratikDhanave/agentic-software-factory-go/cmd/asf@latest
      - run: asf harness run ${{ github.event.pull_request.title }}
      - run: asf factory submit review ${{ github.event.pull_request.title }}`)
	fmt.Println()

	fmt.Println("=== SHARED CONTEXT REPOSITORY ===")
	fmt.Println()
	fmt.Println("Team context structure:")
	fmt.Println("  shared-context/")
	fmt.Println("  ├── adrs/           # Architectural decisions")
	fmt.Println("  ├── specs/          # Team specifications")
	fmt.Println("  ├── patterns/       # Approved code patterns")
	fmt.Println("  ├── conventions/    # Coding conventions")
	fmt.Println("  └── runbooks/       # Operational runbooks")
	fmt.Println()

	fmt.Println("=== AGENT-ASSISTED REVIEW WORKFLOW ===")
	fmt.Println()
	fmt.Println("1. Developer creates PR")
	fmt.Println("2. ASF bot analyzes changes")
	fmt.Println("3. Bot runs verification harness")
	fmt.Println("4. Bot checks spec compliance")
	fmt.Println("5. Bot posts review summary")
	fmt.Println("6. Human reviews + approves")
	fmt.Println("7. Merge triggers deployment")
	fmt.Println()

	fmt.Println("=== TEAM METRICS ===")
	fmt.Println()
	fmt.Println("Track per-team:")
	fmt.Println("  - Agent-assisted PRs / total PRs")
	fmt.Println("  - Verification pass rate")
	fmt.Println("  - Time to merge (agent vs manual)")
	fmt.Println("  - Defect escape rate")
	fmt.Println("  - Developer satisfaction score")
	fmt.Println()

	fmt.Printf("Working directory: %s\n", workingDir)
	fmt.Println("Configure team settings in specs/team.yaml")
}