package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

// JobExecutor interface for executing individual jobs
type JobExecutor interface {
	Execute(ctx context.Context, taskName string, input map[string]interface{}) (interface{}, error)
}

// LoggerInterface aliases Logger so all domain services share a single logger contract.
type LoggerInterface = Logger

// DAGExecutor executes a DAG of jobs with parallelism and dependency management
type DAGExecutor struct {
	jobExecutor   JobExecutor
	logger        LoggerInterface
	maxParallel   int
	defaultTimeout time.Duration
}

// ExecutionResult tracks the result of a single job execution
type ExecutionResult struct {
	JobID    string
	Result   interface{}
	Error    error
	Duration time.Duration
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor(jobExecutor JobExecutor, logger LoggerInterface, maxParallel int) *DAGExecutor {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	return &DAGExecutor{
		jobExecutor:    jobExecutor,
		logger:         logger,
		maxParallel:    maxParallel,
		defaultTimeout: 5 * time.Minute,
	}
}

// Execute runs the DAG level by level with parallel execution at each level
func (e *DAGExecutor) Execute(ctx context.Context, dag *DirectedAcyclicGraph,
	execution *entities.WorkflowExecution) error {

	if dag.IsEmpty() {
		return fmt.Errorf("cannot execute empty DAG")
	}

	levels := dag.GetExecutionLevels()
	if len(levels) == 0 {
		return fmt.Errorf("no execution levels computed for DAG")
	}

	_ = execution.TransitionTo(entities.WorkflowExecRunning)

	// Track results and errors
	results := make(map[string]interface{})
	errors := make(map[string]error)
	var errorsMutex sync.RWMutex

	e.logger.Info("DAG execution started",
		"execution_id", execution.ID,
		"total_jobs", dag.Size(),
		"levels", len(levels),
	)

	// Execute each level
	for levelNum, jobIDs := range levels {
		e.logger.Info("level execution started",
			"level", levelNum,
			"jobs", len(jobIDs),
			"job_ids", jobIDs,
		)

		// Execute all jobs in this level in parallel
		levelResults, levelErrors := e.executeLevel(ctx, dag, levelNum, jobIDs, results, execution)

		// Merge results and errors
		for jobID, result := range levelResults {
			results[jobID] = result
		}

		errorsMutex.Lock()
		for jobID, err := range levelErrors {
			errors[jobID] = err
		}
		errorsMutex.Unlock()

		// Check for failures - fail fast
		failedCount := 0
		for _, err := range levelErrors {
			if err != nil {
				failedCount++
			}
		}

		if failedCount > 0 {
			e.logger.Error("level execution failed",
				fmt.Errorf("jobs failed: %d", failedCount),
				"level", levelNum,
			)

			// Mark failed jobs and their dependents
			failedJobs := make(map[string]error)
			for jobID, err := range levelErrors {
				if err != nil {
					failedJobs[jobID] = err
				}
			}
			e.markDownstreamFailed(dag, errors, failedJobs)

			// Update execution with partial results
			execution.NodesMap = e.buildExecutionNodes(dag, results, errors)
			execution.Results = results
			_ = execution.TransitionTo(entities.WorkflowExecFailed)

			return fmt.Errorf("level %d execution failed with %d errors", levelNum, failedCount)
		}

		// Log level completion
		e.logger.Info("level execution completed",
			"level", levelNum,
			"jobs_completed", len(jobIDs),
		)
	}

	// Update execution with final results
	execution.NodesMap = e.buildExecutionNodes(dag, results, errors)
	execution.Results = results
	_ = execution.TransitionTo(entities.WorkflowExecCompleted)

	e.logger.Info("DAG execution completed",
		"execution_id", execution.ID,
		"total_jobs", dag.Size(),
		"success", true,
	)

	return nil
}

// executeLevel runs all jobs in a single level in parallel
func (e *DAGExecutor) executeLevel(ctx context.Context, dag *DirectedAcyclicGraph,
	levelNum int, jobIDs []string, priorResults map[string]interface{},
	execution *entities.WorkflowExecution) (map[string]interface{}, map[string]error) {

	results := make(map[string]interface{})
	errors := make(map[string]error)
	var resultsMutex sync.RWMutex
	var errorsMutex sync.RWMutex

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	for _, jobID := range jobIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute the job
			result, err := e.executeJob(ctx, dag, id, priorResults, execution)

			resultsMutex.Lock()
			if err == nil {
				results[id] = result
			}
			resultsMutex.Unlock()

			errorsMutex.Lock()
			errors[id] = err
			errorsMutex.Unlock()
		}(jobID)
	}

	wg.Wait()
	return results, errors
}

// executeJob executes a single job with timeout and dependency checking
func (e *DAGExecutor) executeJob(ctx context.Context, dag *DirectedAcyclicGraph,
	jobID string, priorResults map[string]interface{},
	execution *entities.WorkflowExecution) (interface{}, error) {

	node := dag.GetNode(jobID)
	if node == nil {
		return nil, fmt.Errorf("job %s not found in DAG", jobID)
	}

	dagNode := entities.NewDAGNode(jobID, node.TaskName, node.DependsOn, node.Input)
	dagNode.Start()

	startTime := time.Now()
	defer func() {
		if dagNode.CompletedAt == nil {
			dagNode.Duration = time.Since(startTime)
		}
	}()

	// Check dependencies are in priorResults (should be there for non-first-level jobs)
	for _, dep := range node.DependsOn {
		if _, exists := priorResults[dep]; !exists {
			err := fmt.Errorf("dependency %s not found in results", dep)
			dagNode.Fail(err)
			return nil, err
		}
	}

	// Resolve input variables from prior results
	resolvedInput, err := e.resolveInput(node.Input, priorResults)
	if err != nil {
		dagNode.Fail(err)
		return nil, fmt.Errorf("failed to resolve input: %w", err)
	}

	dagNode.Input = resolvedInput

	e.logger.Info("job execution started",
		"job_id", jobID,
		"task", node.TaskName,
	)

	// Execute with timeout
	timeout := e.defaultTimeout
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the job
	result, err := e.jobExecutor.Execute(jobCtx, node.TaskName, resolvedInput)

	if err != nil {
		dagNode.Fail(err)
		e.logger.Error("job execution failed",
			err,
			"job_id", jobID,
			"task", node.TaskName,
			"duration_ms", dagNode.Duration.Milliseconds(),
		)
		return nil, err
	}

	dagNode.Complete(result)

	e.logger.Info("job execution completed",
		"job_id", jobID,
		"task", node.TaskName,
		"duration_ms", dagNode.Duration.Milliseconds(),
	)

	return result, nil
}

// resolveInput replaces template variables with actual values from prior job results
func (e *DAGExecutor) resolveInput(input map[string]interface{},
	results map[string]interface{}) (map[string]interface{}, error) {

	resolved := make(map[string]interface{})

	for key, value := range input {
		if strVal, ok := value.(string); ok {
			// Check for template syntax: {{ jobs.xxx.output }}
			if resolvedVal, isTemplate := e.resolveTemplate(strVal, results); isTemplate {
				if resolvedVal == nil {
					return nil, fmt.Errorf("template resolution failed for key %s: %s", key, strVal)
				}
				resolved[key] = resolvedVal
			} else {
				resolved[key] = strVal
			}
		} else {
			resolved[key] = value
		}
	}

	return resolved, nil
}

// resolveTemplate replaces {{ jobs.xxx.output }} with actual values
func (e *DAGExecutor) resolveTemplate(template string, results map[string]interface{}) (interface{}, bool) {
	trimmed := strings.TrimSpace(template)
	if len(trimmed) < 10 || !strings.HasPrefix(trimmed, "{{") || !strings.HasSuffix(trimmed, "}}") {
		return nil, false
	}

	// Extract content: "jobs.xxx.output"
	content := strings.TrimSpace(trimmed[2 : len(trimmed)-2])

	// Parse: "jobs.xxx.output" or "jobs.xxx.output.field"
	parts := make([]string, 0)
	currentPart := ""
	for _, char := range content {
		if char == '.' {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
		} else {
			currentPart += string(char)
		}
	}
	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	if len(parts) < 2 || parts[0] != "jobs" {
		return nil, false
	}

	jobID := parts[1]

	// Lookup job result
	jobResult, exists := results[jobID]
	if !exists {
		return nil, true // Template but not found
	}

	// If only "jobs.xxx", return the whole result
	if len(parts) == 2 {
		return jobResult, true
	}

	// Navigate field path: skip "output" (it means "the result itself"), then navigate remaining fields
	current := jobResult
	for i := 2; i < len(parts); i++ {
		field := parts[i]

		// Skip "output" - it's just part of the syntax
		if field == "output" {
			continue
		}

		// Try to access field
		if mapVal, ok := current.(map[string]interface{}); ok {
			if val, exists := mapVal[field]; exists {
				current = val
			} else {
				return nil, true // Field not found
			}
		} else {
			return nil, true // Can't navigate further
		}
	}

	return current, true
}

// markDownstreamFailed marks all jobs that depend on failed jobs as failed
func (e *DAGExecutor) markDownstreamFailed(dag *DirectedAcyclicGraph,
	allErrors map[string]error, levelErrors map[string]error) {

	// For each failed job, mark all its dependents as failed
	for failedID := range levelErrors {
		e.markDependentsFailed(dag, failedID, allErrors)
	}
}

// markDependentsFailed recursively marks all dependents as failed
func (e *DAGExecutor) markDependentsFailed(dag *DirectedAcyclicGraph,
	jobID string, errors map[string]error) {

	dependents := dag.GetDependents(jobID)
	for _, dependent := range dependents {
		if _, exists := errors[dependent]; !exists {
			errors[dependent] = fmt.Errorf("skipped: upstream job %s failed", jobID)
			e.markDependentsFailed(dag, dependent, errors)
		}
	}
}

// buildExecutionNodes converts DAG nodes with execution results into DAGNode entities
func (e *DAGExecutor) buildExecutionNodes(dag *DirectedAcyclicGraph,
	results map[string]interface{}, errors map[string]error) map[string]*entities.DAGNode {

	nodes := make(map[string]*entities.DAGNode)

	for jobID, dagNodeRef := range dag.GetAllNodes() {
		dagNode := entities.NewDAGNode(jobID, dagNodeRef.TaskName, dagNodeRef.DependsOn, map[string]interface{}{})

		// Set result or error
		if err, exists := errors[jobID]; exists && err != nil {
			dagNode.Fail(err)
		} else if result, exists := results[jobID]; exists {
			dagNode.Complete(result)
		}

		nodes[jobID] = dagNode
	}

	return nodes
}
