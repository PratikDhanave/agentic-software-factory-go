package jobs

import (
	"context"
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type GetJobStatusUseCase struct {
	jobRepo repositories.JobRepository
	logger  services.Logger
}

func NewGetJobStatusUseCase(
	jobRepo repositories.JobRepository,
	logger services.Logger,
) *GetJobStatusUseCase {
	return &GetJobStatusUseCase{
		jobRepo: jobRepo,
		logger:  logger,
	}
}

func (u *GetJobStatusUseCase) Execute(ctx context.Context, jobID string) (*entities.Job, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID required")
	}

	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		u.logger.Error("failed to get job", err, "id", jobID)
		return nil, err
	}

	if job == nil {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}
