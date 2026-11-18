package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	adapterhttp "github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/http"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence/sqlite"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	appservices "github.com/PratikDhanave/agentic-software-factory-go/internal/application/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/usecases"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/workflows"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	infrallm "github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/llm"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/logging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/messaging"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/observability"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/tracing"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
)

// Application preserves backward compatibility for lightweight unit consumers.
type Application struct {
	WebhookHandler *adapterhttp.WebhookHandler
	JobHandler     *adapterhttp.JobHandler
	Logger         services.Logger
	TriageUseCase  *workflows.IssueTriageUseCase
	SubmitJobUC    *jobs.SubmitJobUseCase
	GetJobStatusUC *jobs.GetJobStatusUseCase
	Observability  *observability.Observability
}

// Container wires the complete unified Agentic Software Factory stack:
// SQLite WAL persistence, Multi-Provider LLM Gateway, Policy & HITL Governance,
// 10-Agent Coordination, Level-Parallel DAG Executor, Isolated Workspace Harness,
// and Continuous Self-Improvement.
type Container struct {
	WorkingDir          string
	Store               *sqlite.Store
	WorkflowRepo        *sqlite.WorkflowRepository
	JobRepo             *sqlite.JobRepository
	SessionRepo          *sqlite.CoordinationSessionRepository
	LearningRepo         *sqlite.LearningRepository
	FactoryRunRepo       *sqlite.FactoryRunRepository
	Pipeline             *application.FactoryPipeline
	LLMGateway           *infrallm.RouterProvider
	PromptClient        *infrallm.PromptAdapter
	ClaudeAdapter       *infrallm.ClaudeClientAdapter
	Logger              services.Logger
	Observability       *observability.Observability
	PolicyEngine        *services.PolicyEngine
	AgentRegistry       *services.AgentRegistry
	KnowledgeGraph      *services.KnowledgeGraphService
	WorkflowGenerator   *services.WorkflowGenerator
	CoordinateUC        *usecases.CoordinateWorkflowUseCase
	InteractiveDecisions *appservices.InteractiveDecisionHandler
	DAGBuilder          *services.DAGBuilder
	DAGExecutor         *services.DAGExecutor
	ExecuteWorkflowUC   *usecases.ExecuteWorkflowUseCase
	SelfImproving       *application.SelfImprovingService
	Factory             *factory.FactoryOrchestrator
	FactoryDef          *factorydef.FactoryDefinition
	Harness             *harness.CompleteHarness
	SpecEngine          *spec.SpecEngine
	CoordinationHandler *adapterhttp.CoordinationHandler
	StartedAt           time.Time

	phaseMu sync.RWMutex
	phase   string // crawl | walk | run
}

// NewContainer initializes the full production container rooted at workingDir.
func NewContainer(workingDir string) (*Container, error) {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger := logging.NewStructuredLogger(logLevel)
	tracer := tracing.NewSimpleTracer("agentic-software-factory")
	obs := observability.NewObservability(logger, tracer)

	dbPath := os.Getenv("ASF_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(workingDir, ".asf", "factory.db")
	}
	store, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening unified sqlite store: %w", err)
	}

	wfRepo := sqlite.NewWorkflowRepository(store)
	jobRepo := sqlite.NewJobRepository(store)
	sessionRepo := sqlite.NewCoordinationSessionRepository(store)
	learningRepo := sqlite.NewLearningRepository(store)
	factoryRunRepo := sqlite.NewFactoryRunRepository(store)

	gatewayCfg := infrallm.DefaultGatewayConfigFromEnv(logger)
	llmGateway := infrallm.NewRouterProvider(gatewayCfg)
	promptClient := infrallm.NewPromptAdapter(llmGateway, "You are an expert Go software engineering agent inside an automated Agentic Software Factory.")
	claudeAdapter := infrallm.NewClaudeClientAdapter(llmGateway)

	policyEngine := services.NewPolicyEngine(logger)
	seedDefaultPolicies(policyEngine)

	agentRegistry := services.NewAgentRegistry()
	knowledgeGraph := services.NewKnowledgeGraphService(nil, logger)
	workflowGen := services.NewWorkflowGenerator(agentRegistry, llmGateway, logger)

	coordinateUC := usecases.NewCoordinateWorkflowUseCase(
		workflowGen,
		knowledgeGraph,
		agentRegistry,
		wfRepo,
		logger,
	).WithSessionRepository(sessionRepo)

	interactiveHandler := appservices.NewInteractiveDecisionHandler(sessionRepo, logger)
	coordHTTP := adapterhttp.NewCoordinationHandler(coordinateUC, interactiveHandler, logger)

	fo, err := factory.NewFactoryOrchestratorWithConfig(factory.FactoryConfig{
		WorkingDir:    workingDir,
		MaxWorkers:    4,
		SharedDB:      store.DB(),
		Agent:         promptClient,
		EvalClient:    promptClient,
		AgentProvider: llmGateway.ProviderName(),
		AgentModel:    activeModelName(llmGateway.ProviderName()),
	})
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("creating factory orchestrator: %w", err)
	}
	if err := fo.Start(context.Background()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("starting factory orchestrator: %w", err)
	}

	dagJobExec := &factoryDAGJobExecutor{
		llm:     llmGateway,
		factory: fo,
		logger:  logger,
	}
	dagBuilder := services.NewDAGBuilder(logger)
	dagExecutor := services.NewDAGExecutor(dagJobExec, logger, 4)
	executeWorkflowUC := usecases.NewExecuteWorkflowUseCase(wfRepo, dagBuilder, dagExecutor, logger)

	selfImproving := application.NewSelfImprovingService(learningRepo, claudeAdapter)

	completeHarness := harness.NewCompleteHarness(workingDir)
	specEngine := spec.NewSpecEngine(filepath.Join(workingDir, "specs"))

	pipeline := application.NewFactoryPipeline(application.FactoryPipelineDeps{
		WorkingDir:    workingDir,
		RunRepo:       factoryRunRepo,
		SessionRepo:   sessionRepo,
		LearningRepo:  learningRepo,
		SpecEngine:    specEngine,
		CoordinateUC:  coordinateUC,
		ExecuteDAGUC:  executeWorkflowUC,
		Harness:       completeHarness,
		LLMProvider:   llmGateway,
		AgentClient:   promptClient,
		EvalClient:    promptClient,
		SelfImproving: selfImproving,
		Logger:        logger,
	})

	var factoryDef *factorydef.FactoryDefinition
	if defPath, findErr := factorydef.FindFactoryDefinition(workingDir); findErr == nil && defPath != "" {
		factoryDef, _ = factorydef.LoadFactoryDefinition(defPath)
	}

	initialPhase := strings.ToLower(strings.TrimSpace(os.Getenv("ASF_PHASE")))
	if initialPhase != "crawl" && initialPhase != "walk" && initialPhase != "run" {
		initialPhase = "walk"
	}

	return &Container{
		WorkingDir:           workingDir,
		Store:                store,
		WorkflowRepo:         wfRepo,
		JobRepo:              jobRepo,
		SessionRepo:          sessionRepo,
		LearningRepo:         learningRepo,
		FactoryRunRepo:       factoryRunRepo,
		Pipeline:             pipeline,
		LLMGateway:           llmGateway,
		PromptClient:         promptClient,
		ClaudeAdapter:        claudeAdapter,
		Logger:               logger,
		Observability:        obs,
		PolicyEngine:         policyEngine,
		AgentRegistry:        agentRegistry,
		KnowledgeGraph:       knowledgeGraph,
		WorkflowGenerator:    workflowGen,
		CoordinateUC:         coordinateUC,
		InteractiveDecisions: interactiveHandler,
		DAGBuilder:           dagBuilder,
		DAGExecutor:          dagExecutor,
		ExecuteWorkflowUC:    executeWorkflowUC,
		SelfImproving:        selfImproving,
		Factory:              fo,
		FactoryDef:           factoryDef,
		Harness:              completeHarness,
		SpecEngine:           specEngine,
		CoordinationHandler:  coordHTTP,
		StartedAt:            time.Now(),
		phase:                initialPhase,
	}, nil
}

func (c *Container) Phase() string {
	c.phaseMu.RLock()
	defer c.phaseMu.RUnlock()
	return c.phase
}

func (c *Container) SetPhase(phase string) error {
	p := domain.FactoryPhase(strings.ToLower(strings.TrimSpace(phase)))
	if !p.IsValid() {
		return fmt.Errorf("invalid phase %q (must be crawl, walk, or run)", phase)
	}
	c.phaseMu.Lock()
	defer c.phaseMu.Unlock()
	cur := domain.FactoryPhase(c.phase)
	if cur == p {
		return nil
	}
	if cur.IsValid() && !cur.CanTransitionTo(p) {
		return fmt.Errorf("phase transition %q -> %q denied by %s", cur, p, domain.FactoryPhaseMachine.Name())
	}
	c.phase = string(p)
	return nil
}

func (c *Container) Close() {
	if c.Factory != nil {
		c.Factory.Stop()
	}
	if c.Store != nil {
		_ = c.Store.Close()
	}
}

// factoryDAGJobExecutor executes individual DAG job nodes using the LLM gateway
// and verification tools.
type factoryDAGJobExecutor struct {
	llm     services.LLMProvider
	factory *factory.FactoryOrchestrator
	logger  services.Logger
}

func (e *factoryDAGJobExecutor) Execute(ctx context.Context, taskName string, input map[string]interface{}) (interface{}, error) {
	start := time.Now()
	prompt := fmt.Sprintf("Execute factory workflow step %q with input: %v.\nProvide concise, actionable engineering output.", taskName, input)
	resp, err := e.llm.Complete(ctx, &services.LLMRequest{
		Prompt:      prompt,
		MaxTokens:   1024,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"task":        taskName,
		"status":      "completed",
		"provider":    resp.Provider,
		"model":       resp.Model,
		"output":      resp.Content,
		"duration_ms": time.Since(start).Milliseconds(),
	}, nil
}

func seedDefaultPolicies(pe *services.PolicyEngine) {
	allowAgents := entities.NewPolicy("pol-default-allow", "Allow Registered Factory Agents & Users")
	allowAgents.Priority = 10
	allowAgents.AddStatement(&entities.Statement{
		Effect:    entities.EffectAllow,
		Principal: []entities.PrincipalType{entities.PrincipalUser, entities.PrincipalAgent, entities.PrincipalService},
		Action:    []entities.Action{"*"},
		Resource:  []entities.Resource{"*"},
	})
	pe.AddPolicy(allowAgents)
}

func activeModelName(provider string) string {
	switch provider {
	case "anthropic":
		if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
			return m
		}
		return "claude-sonnet-4-20250514"
	case "openai":
		if m := os.Getenv("OPENAI_MODEL"); m != "" {
			return m
		}
		return "gpt-4o-mini"
	case "ollama":
		if m := os.Getenv("OLLAMA_MODEL"); m != "" {
			return m
		}
		return "mistral"
	default:
		return "deterministic-mock-v1"
	}
}

func InitializeApplication() *Application {
	logger := logging.NewStructuredLogger("info")
	tracer := tracing.NewSimpleTracer("agentic-factory")
	obs := observability.NewObservability(logger, tracer)

	jobRepo := persistence.NewInMemoryJobRepository()
	workflowRepo := persistence.NewInMemoryWorkflowRepository()
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)
	observer := observability.NewUseCaseObserver(obs)

	return &Application{
		WebhookHandler: adapterhttp.NewWebhookHandler(submitJobUC, logger, observer),
		JobHandler:     adapterhttp.NewJobHandler(getJobStatusUC, logger, observer),
		Logger:         logger,
		TriageUseCase:  triageUC,
		SubmitJobUC:    submitJobUC,
		GetJobStatusUC: getJobStatusUC,
		Observability:  obs,
	}
}

func InitializeApplicationWithSQLite(dbPath string) (*Application, *sqlite.Store, error) {
	if dbPath == "" {
		dbPath = filepath.Join(".asf", "factory.db")
	}
	store, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}

	logger := logging.NewStructuredLogger("info")
	tracer := tracing.NewSimpleTracer("agentic-factory")
	obs := observability.NewObservability(logger, tracer)

	jobRepo := sqlite.NewJobRepository(store)
	workflowRepo := sqlite.NewWorkflowRepository(store)
	eventBus := messaging.NewEventBus()

	submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, logger, eventBus)
	getJobStatusUC := jobs.NewGetJobStatusUseCase(jobRepo, logger)
	triageUC := workflows.NewIssueTriageUseCase(workflowRepo, logger, eventBus)
	observer := observability.NewUseCaseObserver(obs)

	return &Application{
		WebhookHandler: adapterhttp.NewWebhookHandler(submitJobUC, logger, observer),
		JobHandler:     adapterhttp.NewJobHandler(getJobStatusUC, logger, observer),
		Logger:         logger,
		TriageUseCase:  triageUC,
		SubmitJobUC:    submitJobUC,
		GetJobStatusUC: getJobStatusUC,
		Observability:  obs,
	}, store, nil
}
