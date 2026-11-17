# Clean Architecture - Factory Structure

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│          Frameworks & UI (HTTP, Web)        │
├─────────────────────────────────────────────┤
│  Interface Adapters (Handlers, Presenters)  │
├─────────────────────────────────────────────┤
│  Application (Use Cases, Orchestration)     │
├─────────────────────────────────────────────┤
│  Domain (Entities, Business Rules)          │
└─────────────────────────────────────────────┘
     ↑              ↑              ↑
     └──────────────┴──────────────┘
   Dependencies Point INWARD Only
```

## Project Structure

```
cmd/
├── asf-web/
│   ├── main.go                 # Entry point, dependency injection
│   └── handlers/               # HTTP adapters
│       ├── webhook.go
│       ├── job.go
│       └── metrics.go

internal/
├── domain/                     # Core business logic (framework-independent)
│   ├── entities/
│   │   ├── workflow.go         # Workflow entity
│   │   ├── job.go              # Job entity
│   │   └── audit_event.go      # Audit entity
│   ├── repositories/           # Interfaces only
│   │   ├── workflow_repo.go
│   │   └── job_repo.go
│   └── services/               # Business logic
│       ├── workflow_service.go
│       └── security_service.go

├── application/                # Use cases & orchestration
│   ├── workflows/
│   │   ├── issue_triage.go     # Use case: triage issue
│   │   ├── review_pr.go        # Use case: review PR
│   │   └── execute_workflow.go # Use case: execute workflow
│   ├── jobs/
│   │   ├── submit_job.go       # Use case: submit job
│   │   └── get_job_status.go   # Use case: get status
│   └── orchestrator/
│       └── factory_orchestrator.go  # Coordinates all use cases

├── adapters/                   # Framework-specific implementations
│   ├── http/
│   │   ├── server.go           # HTTP server setup
│   │   └── router.go           # Route definitions
│   ├── persistence/
│   │   ├── postgres_repo.go    # PostgreSQL adapter
│   │   └── in_memory_repo.go   # In-memory adapter
│   ├── messaging/
│   │   ├── github_webhook.go   # GitHub webhook adapter
│   │   └── realtime_hub.go     # Realtime messaging adapter
│   └── external/
│       ├── ollama_client.go    # Ollama LLM adapter
│       └── github_client.go    # GitHub API adapter

└── infrastructure/             # Cross-cutting concerns
    ├── config/
    │   └── config.go           # Configuration management
    ├── logger/
    │   └── logger.go           # Logging abstraction
    ├── metrics/
    │   └── metrics.go          # Metrics collection
    └── errors/
        └── errors.go           # Domain-specific errors
```

## Layer Responsibilities

### Domain Layer (Innermost)
**Location**: `internal/domain/`

**Responsibilities:**
- Define business entities (Workflow, Job, AuditEvent)
- Express business rules and constraints
- Define repository interfaces (contracts)
- Domain services that orchestrate entities
- NO dependencies on external frameworks
- Language-agnostic business logic

**Example:**
```go
// internal/domain/entities/workflow.go
package entities

type Workflow struct {
    ID       string
    Type     string
    Status   string
    Input    map[string]interface{}
    Output   map[string]interface{}
}

func (w *Workflow) IsRunning() bool {
    return w.Status == "running"
}

func (w *Workflow) Complete(output map[string]interface{}) error {
    if !w.IsRunning() {
        return errors.New("workflow not running")
    }
    w.Status = "success"
    w.Output = output
    return nil
}
```

### Application Layer (Use Cases)
**Location**: `internal/application/`

**Responsibilities:**
- Implement use cases (application-level business logic)
- Orchestrate domain objects
- Handle workflow and job management
- Call repository methods
- Translate between domain and adapters
- NO UI or framework knowledge

**Example:**
```go
// internal/application/workflows/issue_triage.go
package workflows

type TriageIssueUseCase struct {
    workflowRepo domain.WorkflowRepository
    logger       domain.Logger
}

func (u *TriageIssueUseCase) Execute(issue *domain.Issue) (*domain.TriageResult, error) {
    workflow := domain.NewWorkflow("issue-triage")
    
    result := analyzeIssue(issue)
    
    if err := u.workflowRepo.Save(workflow); err != nil {
        u.logger.Error("failed to save workflow", err)
        return nil, err
    }
    
    return result, nil
}
```

### Adapters Layer (Interface Adapters)
**Location**: `internal/adapters/`

**Responsibilities:**
- Convert external data to domain objects
- Implement repository interfaces
- Handle HTTP requests/responses
- Connect to external services
- Database implementations
- Message brokers

**Example:**
```go
// internal/adapters/persistence/postgres_repo.go
package persistence

type PostgresWorkflowRepo struct {
    db *sql.DB
}

func (r *PostgresWorkflowRepo) Save(w *domain.Workflow) error {
    // Convert domain.Workflow to database model
    // Execute SQL
    // Handle errors
}

func (r *PostgresWorkflowRepo) GetByID(id string) (*domain.Workflow, error) {
    // Query database
    // Convert to domain.Workflow
    // Return
}
```

### Frameworks & UI Layer (Outermost)
**Location**: `cmd/`

**Responsibilities:**
- HTTP server setup
- Route definitions
- Dependency injection/wiring
- Application bootstrap
- Logging configuration

**Example:**
```go
// cmd/asf-web/main.go
func main() {
    cfg := config.Load()
    
    // Wire dependencies
    repo := persistence.NewPostgresWorkflowRepo(cfg.DB)
    logger := infrastructure.NewLogger(cfg.LogLevel)
    
    triageUC := workflows.NewTriageIssueUseCase(repo, logger)
    
    handler := http.NewWorkflowHandler(triageUC)
    
    router := http.NewRouter(handler)
    router.Listen(cfg.Port)
}
```

## Dependency Rules

✅ **CAN DO:**
- Domain → Domain (entities call each other)
- Application → Domain (use cases use entities and repos)
- Adapters → Application (handlers call use cases)
- Adapters → Domain (direct entity handling)
- Outer layers → Inner layers
- Same layer → Same layer

❌ **CANNOT DO:**
- Domain → Application
- Domain → Adapters
- Domain → Framework
- Application → Adapters
- Application → Framework
- Inner → Outer

## Interfaces (Ports)

Domain defines interfaces, adapters implement them:

```go
// internal/domain/repositories/workflow_repo.go
package repositories

type WorkflowRepository interface {
    Save(w *domain.Workflow) error
    GetByID(id string) (*domain.Workflow, error)
    List(status string) ([]*domain.Workflow, error)
}

// internal/domain/services/logger.go
type Logger interface {
    Info(msg string)
    Error(msg string, err error)
    Debug(msg string, data map[string]interface{})
}
```

## Dependency Injection

```go
// cmd/asf-web/bootstrap/bootstrap.go
package bootstrap

func InitializeApp(cfg *config.Config) *app.FactoryApp {
    // Create all dependencies
    db := sql.Open("postgres", cfg.DSN)
    
    // Create adapters
    workflowRepo := persistence.NewPostgresWorkflowRepo(db)
    logger := infrastructure.NewLogger(cfg.LogLevel)
    
    // Create use cases
    triageUC := workflows.NewTriageIssueUseCase(workflowRepo, logger)
    submitJobUC := jobs.NewSubmitJobUseCase(jobQueue, logger)
    
    // Create handlers
    workflowHandler := http.NewWorkflowHandler(triageUC)
    jobHandler := http.NewJobHandler(submitJobUC)
    
    // Create router
    router := http.NewRouter(workflowHandler, jobHandler)
    
    return &app.FactoryApp{
        Router: router,
        DB:     db,
        Logger: logger,
    }
}
```

## Testing Strategy

**Domain tests** - Pure unit tests, no mocks needed:
```go
func TestWorkflowCompletion(t *testing.T) {
    w := entities.NewWorkflow("test")
    w.Status = "running"
    
    err := w.Complete(map[string]interface{}{"result": "ok"})
    assert.NoError(t, err)
    assert.Equal(t, "success", w.Status)
}
```

**Application tests** - Use mocks for repositories:
```go
func TestTriageIssue(t *testing.T) {
    mockRepo := &MockWorkflowRepo{}
    logger := &MockLogger{}
    
    uc := workflows.NewTriageIssueUseCase(mockRepo, logger)
    
    result, err := uc.Execute(&domain.Issue{})
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

**Integration tests** - Full stack with real database:
```go
func TestTriageWorkflowIntegration(t *testing.T) {
    db := setupTestDB()
    defer db.Close()
    
    repo := persistence.NewPostgresWorkflowRepo(db)
    uc := workflows.NewTriageIssueUseCase(repo, logger)
    
    result, _ := uc.Execute(issue)
    assert.NotNil(t, result)
}
```

## Benefits

✅ **Testability** - Domain logic has zero dependencies
✅ **Reusability** - Use cases work with any adapter
✅ **Maintainability** - Clear separation, easy to find things
✅ **Flexibility** - Swap adapters without changing domain
✅ **Scalability** - Easy to add new use cases and adapters
✅ **Framework-agnostic** - Core logic survives framework changes
