package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type InMemoryWorkflowRepository struct {
	workflows map[string]*entities.Workflow
	mu        sync.RWMutex
}

func NewInMemoryWorkflowRepository() repositories.WorkflowRepository {
	return &InMemoryWorkflowRepository{
		workflows: make(map[string]*entities.Workflow),
	}
}

func (r *InMemoryWorkflowRepository) Save(ctx context.Context, workflow *entities.Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows[workflow.ID] = workflow
	return nil
}

func (r *InMemoryWorkflowRepository) GetByID(ctx context.Context, id string) (*entities.Workflow, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	workflow, exists := r.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return workflow, nil
}

func (r *InMemoryWorkflowRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*entities.Workflow
	for _, workflow := range r.workflows {
		if workflow.Status == status {
			result = append(result, workflow)
		}
	}

	return result, nil
}

func (r *InMemoryWorkflowRepository) Update(ctx context.Context, workflow *entities.Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workflows[workflow.ID]; !exists {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}

	r.workflows[workflow.ID] = workflow
	return nil
}

func (r *InMemoryWorkflowRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var result []*entities.Workflow
	count := 0
	for _, workflow := range r.workflows {
		if count >= offset && len(result) < limit {
			result = append(result, workflow)
		}
		count++
	}

	return result, nil
}
