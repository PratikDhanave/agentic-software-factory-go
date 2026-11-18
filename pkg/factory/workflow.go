package factory

import (
	"fmt"
	"log"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/agents"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/analysis"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/codegen"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/docs"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/review"
)

type WorkflowEngine struct {
	codeGen     *codegen.CodeGenerator
	reviewer    *review.CodeReviewer
	bugDetector *analysis.BugDetector
	docGen      *docs.DocGenerator
	executor    *agents.ToolExecutor
}

type WorkflowResult struct {
	Status      string                 `json:"status"` // success, failed, partial
	GeneratedCode string               `json:"generated_code"`
	Tests       string                 `json:"tests"`
	ReviewResult *review.ReviewResult   `json:"review_result"`
	Bugs        []analysis.Bug         `json:"bugs"`
	Documentation *docs.Documentation  `json:"documentation"`
	Metrics     map[string]interface{} `json:"metrics"`
}

func NewWorkflowEngine(ollamaURL, model string) *WorkflowEngine {
	return NewWorkflowEngineWithClient(llm.NewOllamaClient(ollamaURL, model))
}

func NewWorkflowEngineWithClient(client llm.Client) *WorkflowEngine {
	if client == nil {
		client = llm.NewDeterministicClient()
	}
	return &WorkflowEngine{
		codeGen:     codegen.NewCodeGeneratorWithClient(client),
		reviewer:    review.NewCodeReviewerWithClient(client),
		bugDetector: analysis.NewBugDetectorWithClient(client),
		docGen:      docs.NewDocGeneratorWithClient(client),
		executor:    agents.NewToolExecutorWithClient(client),
	}
}

// SpecToShip handles: spec → code → test → review → fix → docs
func (we *WorkflowEngine) SpecToShip(spec *codegen.Specification) (*WorkflowResult, error) {
	log.Printf("[Workflow] SpecToShip: %s", spec.Title)

	result := &WorkflowResult{
		Metrics: make(map[string]interface{}),
	}

	// Step 1: Generate Code
	log.Println("[Workflow] Step 1: Code Generation")
	generated, err := we.codeGen.Generate(spec)
	if err != nil {
		result.Status = "failed"
		return result, fmt.Errorf("code generation failed: %w", err)
	}
	result.GeneratedCode = generated.Code

	// Step 2: Generate Tests
	log.Println("[Workflow] Step 2: Test Generation")
	tests, err := we.codeGen.GenerateTests(spec, generated.Code)
	if err != nil {
		log.Printf("[Workflow] Warning: test generation failed: %v", err)
	}
	result.Tests = tests

	// Step 3: Detect Bugs
	log.Println("[Workflow] Step 3: Bug Detection")
	bugAnalysis, err := we.bugDetector.Detect(generated.Code, spec.Language)
	if err != nil {
		log.Printf("[Workflow] Warning: bug detection failed: %v", err)
	} else {
		result.Bugs = bugAnalysis.Bugs
		result.Metrics["bugs_found"] = bugAnalysis.TotalIssues
		result.Metrics["critical_bugs"] = bugAnalysis.CriticalCount

		// Fix bugs if found
		if bugAnalysis.TotalIssues > 0 {
			log.Println("[Workflow] Step 3b: Bug Fixing")
			fixedCode, err := we.bugDetector.Fix(generated.Code, spec.Language, bugAnalysis.Bugs)
			if err == nil {
				result.GeneratedCode = fixedCode
				result.Metrics["bugs_fixed"] = true
			}
		}
	}

	// Step 4: Code Review
	log.Println("[Workflow] Step 4: Code Review")
	reviewReq := review.ReviewRequest{
		Code:     result.GeneratedCode,
		Language: spec.Language,
		FileName: fmt.Sprintf("generated.%s", spec.Language),
		Context:  spec.Description,
		FocusAreas: []string{"security", "performance", "readability"},
	}
	reviewResult, err := we.reviewer.Review(&reviewReq)
	if err != nil {
		log.Printf("[Workflow] Warning: review failed: %v", err)
	} else {
		result.ReviewResult = reviewResult
		result.Metrics["code_score"] = reviewResult.Score
	}

	// Step 5: Generate Documentation
	log.Println("[Workflow] Step 5: Documentation Generation")
	docResult, err := we.docGen.GenerateFull(spec.Title, spec.Description, result.GeneratedCode, spec.Language)
	if err != nil {
		log.Printf("[Workflow] Warning: docs generation failed: %v", err)
	} else {
		result.Documentation = docResult
	}

	// Determine status
	if reviewResult != nil && reviewResult.Score >= 70 {
		result.Status = "success"
	} else if bugAnalysis != nil && bugAnalysis.CriticalCount == 0 {
		result.Status = "partial"
	} else {
		result.Status = "review_needed"
	}

	log.Printf("[Workflow] Complete: %s", result.Status)
	return result, nil
}

// ReviewAndFix handles: code → review → fix → retest
func (we *WorkflowEngine) ReviewAndFix(code, language string) (*WorkflowResult, error) {
	log.Printf("[Workflow] ReviewAndFix for %s code", language)

	result := &WorkflowResult{
		Metrics: make(map[string]interface{}),
		GeneratedCode: code,
	}

	// Review
	reviewReq := review.ReviewRequest{
		Code:     code,
		Language: language,
	}
	reviewResult, err := we.reviewer.Review(&reviewReq)
	if err != nil {
		return result, err
	}
	result.ReviewResult = reviewResult

	// If issues found, detect bugs
	if reviewResult.Score < 80 {
		bugAnalysis, err := we.bugDetector.Detect(code, language)
		if err == nil && bugAnalysis.TotalIssues > 0 {
			result.Bugs = bugAnalysis.Bugs

			// Fix bugs
			fixedCode, err := we.bugDetector.Fix(code, language, bugAnalysis.Bugs)
			if err == nil {
				result.GeneratedCode = fixedCode
				result.Metrics["fixed"] = true
			}
		}
	}

	result.Status = "complete"
	return result, nil
}

// ExecuteTask delegates to tool executor for complex multi-step tasks
func (we *WorkflowEngine) ExecuteTask(task string) (*agents.ExecutionResult, error) {
	log.Printf("[Workflow] ExecuteTask: %s", task)
	return we.executor.Execute(task)
}
