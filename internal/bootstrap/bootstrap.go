package bootstrap

import (
	"database/sql"
	"os"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/http"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/workflows"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/messaging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/observability"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/tracing"
)

type Application struct {
	WebhookHandler  *http.WebhookHandler
	JobHandler      *http.JobHandler
	Logger          services.Logger
	TriageUseCase   *workflows.IssueTriageUseCase
	SubmitJobUC     *jobs.SubmitJobUseCase
	GetJobStatusUC  *jobs.GetJobStatusUseCase
	Observability   *observability.Observability
}

func InitializeApplication() *Application {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logger := logging.NewStructuredLogger(logLevel)
	tracer := tracing.NewSimpleTracer("agentic-factory")
	obs := observability.NewObservability(logger, tracer)

	jobRepo := persistence.NewInMemoryJobRepository()
	workflowRepo := persistence.NewInMemoryWorkflowRepository()
	eventBus := messaging.NewEventBus()

	logger.Info("initializing application", "log_level", logLevel)

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)

	observer := observability.NewUseCaseObserver(obs)

	webhookHandler := http.NewWebhookHandler(submitJobUC, logger, observer)
	jobHandler := http.NewJobHandler(getJobStatusUC, logger, observer)

	logger.Info("application initialized", "mode", "in-memory")

	return &Application{
		WebhookHandler:  webhookHandler,
		JobHandler:      jobHandler,
		Logger:          logger,
		TriageUseCase:   triageUC,
		SubmitJobUC:     submitJobUC,
		GetJobStatusUC:  getJobStatusUC,
		Observability:   obs,
	}
}

func InitializeApplicationWithPostgres(db *sql.DB) *Application {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logger := logging.NewStructuredLogger(logLevel)
	tracer := tracing.NewSimpleTracer("agentic-factory")
	obs := observability.NewObservability(logger, tracer)

	jobRepo := persistence.NewPostgresJobRepository(db)
	workflowRepo := persistence.NewPostgresWorkflowRepository(db)
	eventBus := messaging.NewEventBus()

	logger.Info("initializing application", "log_level", logLevel)

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)

	observer := observability.NewUseCaseObserver(obs)

	webhookHandler := http.NewWebhookHandler(submitJobUC, logger, observer)
	jobHandler := http.NewJobHandler(getJobStatusUC, logger, observer)

	logger.Info("application initialized", "mode", "postgresql")

	return &Application{
		WebhookHandler:  webhookHandler,
		JobHandler:      jobHandler,
		Logger:          logger,
		TriageUseCase:   triageUC,
		SubmitJobUC:     submitJobUC,
		GetJobStatusUC:  getJobStatusUC,
		Observability:   obs,
	}
}
