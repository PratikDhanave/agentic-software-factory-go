package jobs

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type SubmitJobUseCase struct {
	jobRepo    repositories.JobRepository
	logger     services.Logger
	publisher  services.EventPublisher
}

func NewSubmitJobUseCase(
	jobRepo repositories.JobRepository,
	logger services.Logger,
	publisher services.EventPublisher,
) *SubmitJobUseCase {
	return &SubmitJobUseCase{
		jobRepo:   jobRepo,
		logger:    logger,
		publisher: publisher,
	}
}

func (u *SubmitJobUseCase) Execute(ctx context.Context, job *entities.Job) error {
	if err := job.Validate(); err != nil {
		u.logger.Error("invalid job", err)
		return err
	}

	if err := u.jobRepo.Save(ctx, job); err != nil {
		u.logger.Error("failed to save job", err)
		return err
	}

	event := services.JobSubmittedEvent{
		JobID:   job.ID,
		JobType: job.Type,
	}

	if err := u.publisher.Publish(event); err != nil {
		u.logger.Error("failed to publish job submitted event", err)
		return err
	}

	u.logger.Info("job submitted", "id", job.ID, "type", job.Type)
	return nil
}
