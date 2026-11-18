package repositories

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

// WorkflowRepository interface for workflow persistence
type WorkflowRepository interface {
	// Workflow definition operations
	SaveDefinition(ctx context.Context, def *entities.WorkflowDefinition) error
	GetDefinition(ctx context.Context, id string) (*entities.WorkflowDefinition, error)
	GetDefinitionByName(ctx context.Context, name string) (*entities.WorkflowDefinition, error)
	ListDefinitions(ctx context.Context, limit int, offset int) ([]*entities.WorkflowDefinition, error)
	DeleteDefinition(ctx context.Context, id string) error

	// Workflow execution operations
	SaveExecution(ctx context.Context, exec *entities.WorkflowExecution) error
	GetExecution(ctx context.Context, id string) (*entities.WorkflowExecution, error)
	UpdateExecutionStatus(ctx context.Context, id string, status entities.WorkflowExecutionStatus) error
	ListExecutions(ctx context.Context, definitionID string, limit int, offset int) ([]*entities.WorkflowExecution, error)

	// Job execution operations
	SaveJobExecution(ctx context.Context, jobExec *entities.DAGNode, executionID string) error
	GetJobExecution(ctx context.Context, executionID string, jobID string) (*entities.DAGNode, error)
	UpdateJobStatus(ctx context.Context, executionID string, jobID string, status entities.JobExecutionStatus) error
}
