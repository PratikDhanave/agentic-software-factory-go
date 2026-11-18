package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

// WorkflowMemoryStore is an in-memory implementation of WorkflowRepository
type WorkflowMemoryStore struct {
	mu         sync.RWMutex
	definitions map[string]*entities.WorkflowDefinition
	executions  map[string]*entities.WorkflowExecution
	jobExecs    map[string]map[string]*entities.DAGNode
}

// NewWorkflowMemoryStore creates a new in-memory workflow store
func NewWorkflowMemoryStore() repositories.WorkflowRepository {
	return &WorkflowMemoryStore{
		definitions: make(map[string]*entities.WorkflowDefinition),
		executions:  make(map[string]*entities.WorkflowExecution),
		jobExecs:    make(map[string]map[string]*entities.DAGNode),
	}
}

// SaveDefinition saves a workflow definition
func (s *WorkflowMemoryStore) SaveDefinition(ctx context.Context, def *entities.WorkflowDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if def == nil {
		return fmt.Errorf("workflow definition cannot be nil")
	}

	s.definitions[def.ID] = def
	return nil
}

// GetDefinition retrieves a workflow definition by ID
func (s *WorkflowMemoryStore) GetDefinition(ctx context.Context, id string) (*entities.WorkflowDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.definitions[id], nil
}

// GetDefinitionByName retrieves a workflow definition by name
func (s *WorkflowMemoryStore) GetDefinitionByName(ctx context.Context, name string) (*entities.WorkflowDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, def := range s.definitions {
		if def.Name == name {
			return def, nil
		}
	}
	return nil, nil
}

// ListDefinitions lists workflow definitions
func (s *WorkflowMemoryStore) ListDefinitions(ctx context.Context, limit int, offset int) ([]*entities.WorkflowDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	defs := make([]*entities.WorkflowDefinition, 0, len(s.definitions))
	for _, def := range s.definitions {
		defs = append(defs, def)
	}
	return defs, nil
}

// DeleteDefinition deletes a workflow definition
func (s *WorkflowMemoryStore) DeleteDefinition(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.definitions, id)
	return nil
}

// SaveExecution saves a workflow execution
func (s *WorkflowMemoryStore) SaveExecution(ctx context.Context, exec *entities.WorkflowExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exec == nil {
		return fmt.Errorf("workflow execution cannot be nil")
	}

	s.executions[exec.ID] = exec
	return nil
}

// GetExecution retrieves a workflow execution
func (s *WorkflowMemoryStore) GetExecution(ctx context.Context, id string) (*entities.WorkflowExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.executions[id], nil
}

// UpdateExecutionStatus updates execution status
func (s *WorkflowMemoryStore) UpdateExecutionStatus(ctx context.Context, id string, status entities.WorkflowExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exec, exists := s.executions[id]; exists {
		exec.Status = status
		return nil
	}
	return fmt.Errorf("execution not found: %s", id)
}

// ListExecutions lists workflow executions
func (s *WorkflowMemoryStore) ListExecutions(ctx context.Context, definitionID string, limit int, offset int) ([]*entities.WorkflowExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	execs := make([]*entities.WorkflowExecution, 0)
	for _, exec := range s.executions {
		if exec.DefinitionID == definitionID {
			execs = append(execs, exec)
		}
	}
	return execs, nil
}

// SaveJobExecution saves a job execution
func (s *WorkflowMemoryStore) SaveJobExecution(ctx context.Context, jobExec *entities.DAGNode, executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobExecs[executionID]; !exists {
		s.jobExecs[executionID] = make(map[string]*entities.DAGNode)
	}

	s.jobExecs[executionID][jobExec.JobID] = jobExec
	return nil
}

// GetJobExecution retrieves a job execution
func (s *WorkflowMemoryStore) GetJobExecution(ctx context.Context, executionID string, jobID string) (*entities.DAGNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if execs, exists := s.jobExecs[executionID]; exists {
		return execs[jobID], nil
	}
	return nil, nil
}

// UpdateJobStatus updates job status
func (s *WorkflowMemoryStore) UpdateJobStatus(ctx context.Context, executionID string, jobID string, status entities.JobExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if execs, exists := s.jobExecs[executionID]; exists {
		if jobExec, exists := execs[jobID]; exists {
			jobExec.Status = status
			return nil
		}
	}
	return fmt.Errorf("job execution not found")
}
