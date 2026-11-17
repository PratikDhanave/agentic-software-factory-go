package bootstrap

import (
	"database/sql"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/http"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/workflows"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/messaging"
)

type Application struct {
	WebhookHandler  *http.WebhookHandler
	JobHandler      *http.JobHandler
	Logger          services.Logger
	TriageUseCase   *workflows.IssueTriageUseCase
	SubmitJobUC     *jobs.SubmitJobUseCase
	GetJobStatusUC  *jobs.GetJobStatusUseCase
}

func InitializeApplication() *Application {
	logger := logging.NewStdLogger()

	jobRepo := persistence.NewInMemoryJobRepository()
	workflowRepo := persistence.NewInMemoryWorkflowRepository()
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)

	webhookHandler := http.NewWebhookHandler(submitJobUC, logger)
	jobHandler := http.NewJobHandler(getJobStatusUC, logger)

	return &Application{
		WebhookHandler:  webhookHandler,
		JobHandler:      jobHandler,
		Logger:          logger,
		TriageUseCase:   triageUC,
		SubmitJobUC:     submitJobUC,
		GetJobStatusUC:  getJobStatusUC,
	}
}

func InitializeApplicationWithPostgres(db *sql.DB) *Application {
	logger := logging.NewStdLogger()

	jobRepo := persistence.NewPostgresJobRepository(db)
	workflowRepo := persistence.NewPostgresWorkflowRepository(db)
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)

	webhookHandler := http.NewWebhookHandler(submitJobUC, logger)
	jobHandler := http.NewJobHandler(getJobStatusUC, logger)

	return &Application{
		WebhookHandler:  webhookHandler,
		JobHandler:      jobHandler,
		Logger:          logger,
		TriageUseCase:   triageUC,
		SubmitJobUC:     submitJobUC,
		GetJobStatusUC:  getJobStatusUC,
	}
}
