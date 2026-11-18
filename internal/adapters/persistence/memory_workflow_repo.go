package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type InMemoryWorkflowRepository struct {
	definitions map[string]*entities.WorkflowDefinition
	executions  map[string]*entities.WorkflowExecution
	jobExecs    map[string]map[string]*entities.DAGNode // execID -> jobID -> DAGNode
	mu          sync.RWMutex
}

func NewInMemoryWorkflowRepository() repositories.WorkflowRepository {
	return &InMemoryWorkflowRepository{
		definitions: make(map[string]*entities.WorkflowDefinition),
		executions:  make(map[string]*entities.WorkflowExecution),
		jobExecs:    make(map[string]map[string]*entities.DAGNode),
	}
}

// Workflow definition operations
func (r *InMemoryWorkflowRepository) SaveDefinition(ctx context.Context, def *entities.WorkflowDefinition) error {
	if def == nil {
		return fmt.Errorf("workflow definition cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.definitions[def.ID] = def
	return nil
}

func (r *InMemoryWorkflowRepository) GetDefinition(ctx context.Context, id string) (*entities.WorkflowDefinition, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.definitions[id]
	if !exists {
		return nil, repositories.ErrNotFound
	}

	return def, nil
}

func (r *InMemoryWorkflowRepository) GetDefinitionByName(ctx context.Context, name string) (*entities.WorkflowDefinition, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, def := range r.definitions {
		if def.Name == name {
			return def, nil
		}
	}

	return nil, repositories.ErrNotFound
}

func (r *InMemoryWorkflowRepository) ListDefinitions(ctx context.Context, limit int, offset int) ([]*entities.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var result []*entities.WorkflowDefinition
	count := 0
	for _, def := range r.definitions {
		if count >= offset && len(result) < limit {
			result = append(result, def)
		}
		count++
	}

	return result, nil
}

func (r *InMemoryWorkflowRepository) DeleteDefinition(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.definitions[id]; !exists {
		return repositories.ErrNotFound
	}

	delete(r.definitions, id)
	return nil
}

// Workflow execution operations
func (r *InMemoryWorkflowRepository) SaveExecution(ctx context.Context, exec *entities.WorkflowExecution) error {
	if exec == nil {
		return fmt.Errorf("workflow execution cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.executions[exec.ID] = exec
	return nil
}

func (r *InMemoryWorkflowRepository) GetExecution(ctx context.Context, id string) (*entities.WorkflowExecution, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	exec, exists := r.executions[id]
	if !exists {
		return nil, repositories.ErrNotFound
	}

	return exec, nil
}

func (r *InMemoryWorkflowRepository) UpdateExecutionStatus(ctx context.Context, id string, status entities.WorkflowExecutionStatus) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	exec, exists := r.executions[id]
	if !exists {
		return repositories.ErrNotFound
	}

	exec.Status = status
	return nil
}

func (r *InMemoryWorkflowRepository) ListExecutions(ctx context.Context, definitionID string, limit int, offset int) ([]*entities.WorkflowExecution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var result []*entities.WorkflowExecution
	count := 0
	for _, exec := range r.executions {
		if exec.DefinitionID == definitionID {
			if count >= offset && len(result) < limit {
				result = append(result, exec)
			}
			count++
		}
	}

	return result, nil
}

// Job execution operations
func (r *InMemoryWorkflowRepository) SaveJobExecution(ctx context.Context, jobExec *entities.DAGNode, executionID string) error {
	if jobExec == nil {
		return fmt.Errorf("job execution cannot be nil")
	}
	if executionID == "" {
		return fmt.Errorf("execution id cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.jobExecs[executionID] == nil {
		r.jobExecs[executionID] = make(map[string]*entities.DAGNode)
	}

	r.jobExecs[executionID][jobExec.JobID] = jobExec
	return nil
}

func (r *InMemoryWorkflowRepository) GetJobExecution(ctx context.Context, executionID string, jobID string) (*entities.DAGNode, error) {
	if executionID == "" || jobID == "" {
		return nil, fmt.Errorf("execution id and job id cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	jobExec, exists := r.jobExecs[executionID][jobID]
	if !exists {
		return nil, repositories.ErrNotFound
	}

	return jobExec, nil
}

func (r *InMemoryWorkflowRepository) UpdateJobStatus(ctx context.Context, executionID string, jobID string, status entities.JobExecutionStatus) error {
	if executionID == "" || jobID == "" {
		return fmt.Errorf("execution id and job id cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	jobExec, exists := r.jobExecs[executionID][jobID]
	if !exists {
		return repositories.ErrNotFound
	}

	jobExec.Status = status
	return nil
}
