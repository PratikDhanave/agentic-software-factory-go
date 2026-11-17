# Clean Architecture Refactoring Plan

## Current Violations

### 1. **Integration Handler Orchestration** (handlers/integration.go)
**Problem:** Single handler doing everything
- Verifies signatures (should be in webhook adapter)
- Submits jobs (should be in job use case)
- Calls business logic (should delegate to use cases)
- Broadcasts messages (should use pub/sub interface)
- Records metrics (should be injected)
- Logs audits (should be injected)

**Current Flow:**
```
HTTP Request 
  → HandleGitHubWebhookIntegration
    → Verify (embedded)
    → Queue.Submit (direct call)
    → executeWorkflow (embedded)
    → realtimeHub.Broadcast (direct call)
    → metricsCollector.Record (direct call)
    → auditLog.Log (direct call)
```

**Should Be:**
```
HTTP Request
  → WebhookHandler (adapter)
    → VerifyWebhookUseCase (application)
      → GitHubWebhookVerifier (adapter interface)
      → WebhookRepository (domain interface)
    → SubmitJobUseCase (application)
      → JobQueue (domain interface)
      → EventPublisher (infrastructure interface)
    → RecordMetricsUseCase (application)
      → MetricsCollector (infrastructure interface)
```

### 2. **Global State** (handlers/integration.go lines 19-29)
**Problem:** Global variables for all subsystems
```go
var (
    jobQueue          *queue.Queue
    jobScheduler      *queue.Scheduler
    metricsCollector  *metrics.MetricsCollector
    realtimeHub       *realtime.Hub
    securityValidator *security.Validator
    rateLimiter       *security.RateLimiter
    auditLog          *security.AuditLog
    webhookProcessor  *webhooks.EventProcessor
)
```

**Issues:**
- No dependency injection
- Tight coupling
- Cannot test in isolation
- Cannot swap implementations

**Fix:** Inject via constructor

### 3. **Mixed Concerns** (handlers/integration.go executeWorkflow)
**Problem:** Single function handles multiple workflows
- triageIssue, reviewPullRequest, handlePRFeedback, analyzePush all embedded
- Each is a separate use case

**Should Be:** Separate use case classes
```
IssueTriageUseCase
PullRequestReviewUseCase
PullRequestFeedbackUseCase
PushAnalysisUseCase
```

### 4. **Business Logic in Handlers**
**Problem:** Workflow logic in HTTP handlers
- triageIssue returns business logic result
- reviewPullRequest calculates scores
- These should be in domain/application

**Fix:** Move to use cases or domain services

### 5. **Weak Database Integration**
**Problem:** Database operations scattered
- `database.RecordTriageResult` called from handler
- No clear repository pattern
- Mixed concerns in handlers/database.go

**Fix:** Create proper repositories in adapters/persistence

---

## Refactoring Phases

### Phase 1: Create Domain Layer (Low Risk)
**Files to Create:**
- `internal/domain/entities/job.go`
- `internal/domain/entities/workflow.go`
- `internal/domain/entities/audit_event.go`
- `internal/domain/repositories/job_repo.go`
- `internal/domain/repositories/workflow_repo.go`
- `internal/domain/services/logger.go`
- `internal/domain/services/event_publisher.go`

**Changes:** None to existing files yet. Just new interfaces.

**Example:**
```go
// internal/domain/repositories/job_repo.go
package repositories

import "github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"

type JobRepository interface {
    Save(job *entities.Job) error
    GetByID(id string) (*entities.Job, error)
    GetByStatus(status string) ([]*entities.Job, error)
    Update(job *entities.Job) error
}
```

```go
// internal/domain/entities/job.go
package entities

import "time"

type Job struct {
    ID           string
    Type         string
    Status       string // pending, running, success, failed
    Input        map[string]interface{}
    Output       map[string]interface{}
    Error        string
    CreatedAt    time.Time
    StartedAt    *time.Time
    CompletedAt  *time.Time
    Retries      int
    MaxRetries   int
}

func (j *Job) IsRunning() bool {
    return j.Status == "running"
}

func (j *Job) Complete(output map[string]interface{}) error {
    if !j.IsRunning() {
        return fmt.Errorf("cannot complete non-running job")
    }
    j.Status = "success"
    j.Output = output
    j.CompletedAt = timePtr(time.Now())
    return nil
}

func (j *Job) Fail(err error) {
    j.Status = "failed"
    j.Error = err.Error()
    j.CompletedAt = timePtr(time.Now())
}
```

### Phase 2: Create Application Layer (Medium Risk)
**Files to Create:**
- `internal/application/jobs/submit_job.go` (SubmitJobUseCase)
- `internal/application/jobs/get_job_status.go` (GetJobStatusUseCase)
- `internal/application/webhooks/verify_webhook.go` (VerifyWebhookUseCase)
- `internal/application/workflows/issue_triage.go` (TriageIssueUseCase)
- `internal/application/workflows/review_pr.go` (ReviewPRUseCase)
- `internal/application/orchestrator/factory_orchestrator.go` (Main coordinator)

**Example:**
```go
// internal/application/jobs/submit_job.go
package jobs

type SubmitJobUseCase struct {
    jobRepo domain.JobRepository
    queue   domain.JobQueue
    pubsub  domain.EventPublisher
    logger  domain.Logger
}

func (u *SubmitJobUseCase) Execute(ctx context.Context, job *domain.Job) error {
    // Validate job
    if err := job.Validate(); err != nil {
        u.logger.Error("invalid job", err)
        return err
    }
    
    // Save to repository
    if err := u.jobRepo.Save(job); err != nil {
        u.logger.Error("failed to save job", err)
        return err
    }
    
    // Submit to queue
    if err := u.queue.Submit(job); err != nil {
        u.logger.Error("failed to submit job", err)
        return err
    }
    
    // Publish event
    u.pubsub.Publish(domain.JobSubmittedEvent{JobID: job.ID})
    
    return nil
}
```

### Phase 3: Create Adapters (Medium Risk)
**Files to Create:**
- `internal/adapters/http/server.go` (HTTP setup)
- `internal/adapters/http/webhook_handler.go` (GitHub webhook adapter)
- `internal/adapters/http/job_handler.go` (Job API handler)
- `internal/adapters/persistence/postgres_job_repo.go` (JobRepository impl)
- `internal/adapters/persistence/postgres_workflow_repo.go` (WorkflowRepository impl)
- `internal/adapters/messaging/github_webhook.go` (GitHub adapter)
- `internal/adapters/messaging/realtime_publisher.go` (EventPublisher impl)

**Changes:** Rename existing implementations to implement interfaces

**Example:**
```go
// internal/adapters/http/webhook_handler.go
package http

type WebhookHandler struct {
    verifyUC domain.VerifyWebhookUseCase
    submitUC domain.SubmitJobUseCase
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    payload, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Hub-Signature-256")
    
    // Use case: verify webhook
    isValid, err := h.verifyUC.Execute(payload, signature)
    if err != nil || !isValid {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Parse event
    event, err := h.parseEvent(payload)
    if err != nil {
        http.Error(w, "Failed to parse", http.StatusBadRequest)
        return
    }
    
    // Use case: submit job
    job := h.eventToJob(event)
    if err := h.submitUC.Execute(r.Context(), job); err != nil {
        http.Error(w, "Failed to submit", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "status":     "accepted",
        "workflow_id": job.ID,
    })
}
```

### Phase 4: Wire Everything in Main (Low Risk)
**File to Modify:** `cmd/asf-web/main.go`

**Changes:**
- Delete `InitializeIntegration()`
- Create proper dependency injection
- Wire all use cases
- Create HTTP router

**Example:**
```go
// cmd/asf-web/main.go
func main() {
    cfg := config.Load()
    
    // Infrastructure
    db := setupDB(cfg)
    logger := infrastructure.NewLogger(cfg.LogLevel)
    metricsCollector := infrastructure.NewMetricsCollector()
    eventBus := infrastructure.NewEventBus()
    
    // Adapters (implementing domain interfaces)
    jobRepo := adapters.NewPostgresJobRepository(db)
    jobQueue := adapters.NewJobQueue(5, 3)
    gitHubAdapter := adapters.NewGitHubAdapter(os.Getenv("GITHUB_SECRET"))
    
    // Application layer (use cases)
    submitJobUC := jobs.NewSubmitJobUseCase(jobRepo, jobQueue, eventBus, logger)
    triageUC := workflows.NewTriageIssueUseCase(jobRepo, logger)
    
    // Adapters (HTTP handlers)
    webhookHandler := http.NewWebhookHandler(submitJobUC, triageUC)
    
    // HTTP server
    router := http.NewRouter(webhookHandler)
    router.Listen(cfg.Port)
}
```

### Phase 5: Migrate Handlers (High Risk)
**Files to Modify/Delete:**
- Delete `handlers/integration.go` (split into use cases + handlers)
- Modify `handlers/web.go` (use new adapters)
- Modify `handlers/database.go` (use repositories)

---

## Implementation Order

**Week 1: Foundation**
1. Create domain entities and interfaces
2. Create application use cases (5 high-value ones)
3. Add tests for use cases

**Week 2: Adapters**
1. Create HTTP handlers for 2-3 adapters
2. Create persistence repositories
3. Update main.go with DI

**Week 3: Integration**
1. Wire everything together
2. Test full flows
3. Delete old integration.go

**Week 4: Polish**
1. Add remaining adapters
2. Improve error handling
3. Add documentation

---

## High-Value First Changes

**1. TriageIssueUseCase** (Highest Value)
- Isolate issue triage logic
- Testable in isolation
- Can be reused by other handlers

**2. SubmitJobUseCase**
- Decouple queue submission from HTTP
- Reusable across webhook/API/scheduled jobs

**3. WebhookHandler** (Adapter)
- Cleaner than integration.go
- Clear input/output

**4. JobRepository** (Persistence)
- Swap queue with database without changing business logic
- Enable different storage backends

---

## Testing Benefits

### Before (Current)
```go
func TestTriageIssue(t *testing.T) {
    // Need to initialize all globals
    InitializeIntegration()
    
    // Need to mock HTTP
    req := httptest.NewRequest(...)
    
    // Can't test logic in isolation
}
```

### After (Clean Architecture)
```go
func TestTriageIssue(t *testing.T) {
    mockRepo := &MockJobRepository{}
    mockLogger := &MockLogger{}
    
    uc := workflows.NewTriageIssueUseCase(mockRepo, mockLogger)
    
    result, err := uc.Execute(context.Background(), testIssue)
    assert.NoError(t, err)
}
```

---

## Key Benefits After Refactoring

✅ **Testability** — 10x faster unit tests  
✅ **Reusability** — Use cases work with any storage/messaging  
✅ **Maintainability** — Clear responsibility per file  
✅ **Flexibility** — Swap queue for database without code changes  
✅ **Scalability** — Easy to add new adapters (Kafka, gRPC, etc.)  
✅ **Framework-agnostic** — Replace HTTP with gRPC later  

