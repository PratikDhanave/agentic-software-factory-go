package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// ExecuteWorkflowUseCase orchestrates workflow execution
type ExecuteWorkflowUseCase struct {
	workflowRepo repositories.WorkflowRepository
	dagBuilder   *services.DAGBuilder
	dagExecutor  *services.DAGExecutor
	logger       services.LoggerInterface
}

// NewExecuteWorkflowUseCase creates a new execute workflow use case
func NewExecuteWorkflowUseCase(
	workflowRepo repositories.WorkflowRepository,
	dagBuilder *services.DAGBuilder,
	dagExecutor *services.DAGExecutor,
	logger services.LoggerInterface,
) *ExecuteWorkflowUseCase {
	return &ExecuteWorkflowUseCase{
		workflowRepo: workflowRepo,
		dagBuilder:   dagBuilder,
		dagExecutor:  dagExecutor,
		logger:       logger,
	}
}

// Execute runs a workflow definition with given trigger event
func (u *ExecuteWorkflowUseCase) Execute(ctx context.Context,
	definitionID string, triggerEvent map[string]interface{}) (*entities.WorkflowExecution, error) {

	// Step 1: Load workflow definition
	def, err := u.workflowRepo.GetDefinition(ctx, definitionID)
	if err != nil {
		u.logger.Error("failed to load workflow definition", err, "definition_id", definitionID)
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	if def == nil {
		return nil, fmt.Errorf("workflow definition not found: %s", definitionID)
	}

	u.logger.Info("workflow definition loaded",
		"definition_id", definitionID,
		"name", def.Name,
		"version", def.Version,
	)

	// Step 2: Create execution instance
	executionID := fmt.Sprintf("exec-%d", time.Now().UnixNano())
	execution := entities.NewWorkflowExecution(executionID, definitionID, triggerEvent)
	now := time.Now()
	execution.StartedAt = &now
	execution.Status = entities.WorkflowExecRunning

	// Step 3: Save execution (started)
	err = u.workflowRepo.SaveExecution(ctx, execution)
	if err != nil {
		u.logger.Error("failed to save workflow execution", err, "execution_id", executionID)
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	u.logger.Info("workflow execution started",
		"execution_id", executionID,
		"definition_id", definitionID,
	)

	// Step 4: Build DAG from workflow definition
	dag, err := u.dagBuilder.Build(def)
	if err != nil {
		u.logger.Error("failed to build DAG", err, "definition_id", definitionID)
		execution.Status = entities.WorkflowExecFailed
		u.workflowRepo.UpdateExecutionStatus(ctx, executionID, entities.WorkflowExecFailed)
		return nil, fmt.Errorf("failed to build DAG: %w", err)
	}

	u.logger.Info("DAG built successfully",
		"execution_id", executionID,
		"nodes", dag.Size(),
		"levels", len(dag.GetExecutionLevels()),
	)

	// Step 5: Execute DAG
	err = u.dagExecutor.Execute(ctx, dag, execution)
	if err != nil {
		u.logger.Error("DAG execution failed", err, "execution_id", executionID)
		execution.Status = entities.WorkflowExecFailed
	} else {
		u.logger.Info("DAG execution completed successfully", "execution_id", executionID)
		execution.Status = entities.WorkflowExecCompleted
	}

	// Step 6: Update execution status
	updateErr := u.workflowRepo.UpdateExecutionStatus(ctx, executionID, execution.Status)
	if updateErr != nil {
		u.logger.Error("failed to update execution status", updateErr, "execution_id", executionID)
	}

	// Step 7: Save final results
	saveErr := u.workflowRepo.SaveExecution(ctx, execution)
	if saveErr != nil {
		u.logger.Error("failed to save execution results", saveErr, "execution_id", executionID)
	}

	// Log summary
	duration := time.Duration(0)
	if execution.StartedAt != nil {
		duration = time.Since(*execution.StartedAt)
	}
	u.logger.Info("workflow execution completed",
		"execution_id", executionID,
		"status", execution.Status,
		"duration_ms", duration.Milliseconds(),
		"jobs_completed", len(execution.NodesMap),
	)

	return execution, err
}
