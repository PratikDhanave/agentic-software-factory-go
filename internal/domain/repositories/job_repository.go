package repositories

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type JobRepository interface {
	Save(ctx context.Context, job *entities.Job) error
	GetByID(ctx context.Context, id string) (*entities.Job, error)
	GetByStatus(ctx context.Context, status string) ([]*entities.Job, error)
	Update(ctx context.Context, job *entities.Job) error
	List(ctx context.Context, limit int, offset int) ([]*entities.Job, error)
}
