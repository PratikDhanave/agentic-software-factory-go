package repositories

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type WorkflowRepository interface {
	Save(ctx context.Context, workflow *entities.Workflow) error
	GetByID(ctx context.Context, id string) (*entities.Workflow, error)
	GetByStatus(ctx context.Context, status string) ([]*entities.Workflow, error)
	Update(ctx context.Context, workflow *entities.Workflow) error
	List(ctx context.Context, limit int, offset int) ([]*entities.Workflow, error)
}
