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

	fmt.Println("=== Autonomy Calibration Demo (Chapter 7) ===")
	fmt.Println("Demonstrating AEMM maturity model and Elastic Loop framework")
	fmt.Println()

	fmt.Println("=== Autonomy Spectrum (AEMM) ===")
	fmt.Println()
	fmt.Println("Level 0 - Manual: Developer writes all code")
	fmt.Println("Level 1 - Assisted: Agent suggests completions, developer approves")
	fmt.Println("Level 2 - Supervised: Agent generates functions, developer reviews")
	fmt.Println("Level 3 - Managed: Agent implements features, developer validates")
	fmt.Println("Level 4 - Autonomous: Agent owns features end-to-end, human on exception")
	fmt.Println()

	fmt.Println("=== Elastic Loop Framework ===")
	fmt.Println()
	fmt.Println("The Elastic Loop dynamically adjusts autonomy based on:")
	fmt.Println("  1. Task Risk Score (0-100)")
	fmt.Println("  2. Context Quality Score (0-100)")
	fmt.Println("  3. Agent Confidence Score (0-100)")
	fmt.Println("  4. Historical Success Rate (0-100)")
	fmt.Println()
	fmt.Println("Autonomy Level = f(Risk, Context, Confidence, History)")
	fmt.Println()

	fmt.Println("=== Decision Gates ===")
	fmt.Println()
	fmt.Println("Gate 1 - Spec Validation: Spec must pass validation before execution")
	fmt.Println("Gate 2 - Context Sufficiency: Context quality must exceed threshold")
	fmt.Println("Gate 3 - Verification Pass: All tests/lint must pass")
	fmt.Println("Gate 4 - Human Approval: Required for high-risk tasks")
	fmt.Println()

	// Example calculation
	risk := 30
	contextQuality := 85
	confidence := 90
	history := 95

	autonomyScore := (contextQuality + confidence + history - risk) / 3
	fmt.Printf("=== Example Calculation ===\n")
	fmt.Printf("Risk: %d, Context: %d, Confidence: %d, History: %d\n", risk, contextQuality, confidence, history)
	fmt.Printf("Autonomy Score: %d/100\n", autonomyScore)

	if autonomyScore >= 80 {
		fmt.Println("Decision: Level 4 Autonomous (with monitoring)")
	} else if autonomyScore >= 60 {
		fmt.Println("Decision: Level 3 Managed (validation required)")
	} else if autonomyScore >= 40 {
		fmt.Println("Decision: Level 2 Supervised (review required)")
	} else {
		fmt.Println("Decision: Level 1 Assisted (approval required)")
	}
	fmt.Println()

	fmt.Println("=== Working Directory ===")
	fmt.Printf("Run from: %s\n", workingDir)
	fmt.Println("Configure autonomy thresholds in specs/autonomy.yaml")
}