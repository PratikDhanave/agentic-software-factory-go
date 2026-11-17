package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type InMemoryJobRepository struct {
	jobs map[string]*entities.Job
	mu   sync.RWMutex
}

func NewInMemoryJobRepository() repositories.JobRepository {
	return &InMemoryJobRepository{
		jobs: make(map[string]*entities.Job),
	}
}

func (r *InMemoryJobRepository) Save(ctx context.Context, job *entities.Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepository) GetByID(ctx context.Context, id string) (*entities.Job, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return job, nil
}

func (r *InMemoryJobRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*entities.Job
	for _, job := range r.jobs {
		if job.Status == status {
			result = append(result, job)
		}
	}

	return result, nil
}

func (r *InMemoryJobRepository) Update(ctx context.Context, job *entities.Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[job.ID]; !exists {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var result []*entities.Job
	count := 0
	for _, job := range r.jobs {
		if count >= offset && len(result) < limit {
			result = append(result, job)
		}
		count++
	}

	return result, nil
}
