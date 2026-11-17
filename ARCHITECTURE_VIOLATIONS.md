# Architecture Violations - Current vs. Target

## Violation #1: God Handler Pattern

### CURRENT (❌ Anti-Pattern)
```
handlers/integration.go (355 lines)
  ├─ HTTP handling (request/response)
  ├─ Signature verification (security concern)
  ├─ Job submission (queue concern)
  ├─ Workflow execution (business logic)
  │   ├─ Issue triage (domain logic)
  │   ├─ PR review (domain logic)
  │   ├─ PR feedback (domain logic)
  │   └─ Push analysis (domain logic)
  ├─ Real-time broadcasting (messaging concern)
  ├─ Metrics recording (observability concern)
  └─ Audit logging (security concern)
```

**Problem:** One function does 7 different things. Hard to test, modify, reuse.

### TARGET (✅ Clean)
```
Adapters (HTTP)
  └─ WebhookHandler.HandleGitHubWebhook()
       ↓ calls
Application (Use Cases)
  ├─ VerifyWebhookUseCase.Execute()
  │    ↓ uses
  │   Adapter (GitHub)
  │    └─ GitHubAdapter.VerifySignature()
  │
  └─ SubmitJobUseCase.Execute()
       ├─ uses Domain (Repository)
       │  └─ JobRepository.Save()
       ├─ uses Domain (Queue)
       │  └─ JobQueue.Submit()
       └─ uses Infrastructure (Publisher)
          └─ EventPublisher.Publish()
```

Each concern has one place. Each function has one responsibility.

---

## Violation #2: Global State

### CURRENT (❌ No DI)
```go
// handlers/integration.go (lines 19-29)
var (
    jobQueue          *queue.Queue          // Global!
    jobScheduler      *queue.Scheduler      // Global!
    metricsCollector  *metrics.MetricsCollector // Global!
    realtimeHub       *realtime.Hub         // Global!
    securityValidator *security.Validator   // Global!
    rateLimiter       *security.RateLimiter // Global!
    auditLog          *security.AuditLog    // Global!
    webhookProcessor  *webhooks.EventProcessor // Global!
)

func HandleGitHubWebhookIntegration(w http.ResponseWriter, r *http.Request) {
    jobQueue.Submit(...)      // Hidden dependency
    metricsCollector.Record() // Hidden dependency
    realtimeHub.Broadcast()   // Hidden dependency
}
```

**Problems:**
- Can't test without initializing everything
- Can't swap implementations (e.g., queue for database)
- Can't run tests in parallel
- Unclear dependencies

### TARGET (✅ Dependency Injection)
```go
// internal/adapters/http/webhook_handler.go
type WebhookHandler struct {
    verifyUC   application.VerifyWebhookUseCase
    submitUC   application.SubmitJobUseCase
}

func NewWebhookHandler(verify application.VerifyWebhookUseCase, 
                       submit application.SubmitJobUseCase) *WebhookHandler {
    return &WebhookHandler{verify, submit}
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    // Dependencies explicit in constructor
    h.verifyUC.Execute(...)   // Clear dependency
    h.submitUC.Execute(...)   // Clear dependency
}

// main.go
func main() {
    jobRepo := adapters.NewPostgresJobRepository(db)
    jobQueue := adapters.NewJobQueue(5, 3)
    pubsub := adapters.NewRealTimePublisher()
    
    submitUC := jobs.NewSubmitJobUseCase(jobRepo, jobQueue, pubsub)
    handler := http.NewWebhookHandler(submitUC)
    
    // Easy to test: replace jobRepo with mock, everything else works
}
```

---

## Violation #3: Business Logic in Handlers

### CURRENT (❌ Mixed Concerns)
```go
// handlers/integration.go lines 176-202
func triageIssue(trigger *webhooks.WorkflowTrigger) (map[string]interface{}, error) {
    // This is business logic, not HTTP handling!
    
    time.Sleep(100 * time.Millisecond)
    
    result := map[string]interface{}{
        "issue_number": trigger.Issue,
        "repository":   trigger.Repo,
        "priority":     "high",        // Business decision (should be in domain)
        "category":     "feature",     // Business decision (should be in domain)
        "action":       "review",      // Business decision (should be in domain)
        "suggested_labels": []string{"feature", "review-needed"},
    }
    
    if database != nil {
        database.RecordTriageResult(...)  // Handler calling database?!
    }
    
    return result, nil
}
```

**Problems:**
- Business logic mixed with HTTP handling
- Cannot test without HTTP handler
- Hard to change triage rules
- Unclear where business rules live

### TARGET (✅ Layered)
```go
// internal/domain/services/issue_service.go
type IssueTriageService struct {
    // Domain logic only
}

func (s *IssueTriageService) TriageIssue(issue *domain.Issue) (*domain.TriageResult, error) {
    // Pure business logic - no frameworks, no HTTP, no database
    result := &domain.TriageResult{
        Priority: s.determinePriority(issue),
        Category: s.determineCategory(issue),
        Action:   s.determineAction(issue),
    }
    return result, nil
}

// internal/application/workflows/issue_triage.go
type TriageIssueUseCase struct {
    service    *domain.IssueTriageService
    repo       domain.IssueRepository
    pubsub     domain.EventPublisher
}

func (u *TriageIssueUseCase) Execute(ctx context.Context, issue *domain.Issue) error {
    // Orchestrate domain logic + persistence
    result, err := u.service.TriageIssue(issue)
    if err != nil {
        return err
    }
    
    // Persist
    if err := u.repo.SaveTriageResult(result); err != nil {
        return err
    }
    
    // Publish
    u.pubsub.Publish(domain.IssueTriagedEvent{...})
    return nil
}

// internal/adapters/http/workflow_handler.go
func (h *WorkflowHandler) HandleTriageIssue(w http.ResponseWriter, r *http.Request) {
    // Pure HTTP handling - parse request, call use case, format response
    issue := h.parseIssue(r)
    result, err := h.triageUC.Execute(r.Context(), issue)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

**Benefits:**
- Business logic testable without HTTP
- Use case testable without database
- HTTP handling testable without business logic
- Each layer has clear responsibility

---

## Violation #4: Implicit Dependencies

### CURRENT (❌ Hidden Dependencies)
```go
// handlers/integration.go
func InitializeIntegration() error {
    jobQueue = queue.NewQueue(5, 3)
    metricsCollector = metrics.NewMetricsCollector(1000)
    realtimeHub = realtime.NewHub()
    // ... more initialization
    return nil
}

// handlers/integration.go
func HandleGitHubWebhookIntegration(w http.ResponseWriter, r *http.Request) {
    // What dependencies does this function need?
    // You have to read the entire file to find out!
    jobQueue.Submit(...)      // ← Where does jobQueue come from?
    metricsCollector.Record() // ← Where does metricsCollector come from?
}
```

**Problems:**
- Cannot tell what dependencies are needed just by looking at the function
- If you forget to call InitializeIntegration(), you get nil pointer panic
- Hard to test - must initialize everything

### TARGET (✅ Explicit Dependencies)
```go
// internal/adapters/http/webhook_handler.go
type WebhookHandler struct {
    verifyUC   application.VerifyWebhookUseCase
    submitUC   application.SubmitJobUseCase
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    // Look at the struct fields: clear what this needs
    h.verifyUC.Execute(...)   // ← Explicitly available
    h.submitUC.Execute(...)   // ← Explicitly available
}

// Test without globals
func TestHandleGitHubWebhook(t *testing.T) {
    mockVerify := &mockVerifyUseCase{} // Create just what you need
    mockSubmit := &mockSubmitUseCase{}
    
    handler := NewWebhookHandler(mockVerify, mockSubmit)
    // No global state, no hidden initialization
    
    handler.HandleGitHubWebhook(w, r)
    assert.NoError(t, err)
}
```

---

## Violation #5: Tight Coupling to Queue

### CURRENT (❌ Specific Implementation)
```go
// handlers/integration.go
func HandleGitHubWebhookIntegration(...) {
    jobQueue.Submit(job)  // ← Tightly coupled to queue.Queue type
}

// If you want to use a database instead of a queue, you must:
// 1. Find all references to jobQueue
// 2. Add conditional logic (if useQueue then jobQueue else db)
// 3. Test both paths
// 4. Maintain two code paths forever
```

### TARGET (✅ Abstraction)
```go
// internal/domain/repositories/job_repo.go
type JobRepository interface {
    Save(job *domain.Job) error
    GetByID(id string) (*domain.Job, error)
}

// internal/application/jobs/submit_job.go
type SubmitJobUseCase struct {
    repo domain.JobRepository  // ← Abstraction, not concrete type
}

func (u *SubmitJobUseCase) Execute(ctx context.Context, job *domain.Job) error {
    return u.repo.Save(job)   // ← Works with any implementation
}

// internal/adapters/persistence/queue_repo.go
type QueueJobRepository struct {
    queue *queue.Queue
}
func (r *QueueJobRepository) Save(job *domain.Job) error {
    return r.queue.Submit(...)
}

// internal/adapters/persistence/postgres_repo.go
type PostgresJobRepository struct {
    db *sql.DB
}
func (r *PostgresJobRepository) Save(job *domain.Job) error {
    return r.db.SaveJob(...)
}

// In main.go, choose at startup:
func main() {
    var repo domain.JobRepository
    
    if config.UseQueue {
        repo = adapters.NewQueueJobRepository(queue)
    } else {
        repo = adapters.NewPostgresJobRepository(db)
    }
    
    submitUC := jobs.NewSubmitJobUseCase(repo)
    // Same code path, different storage
}
```

**Benefits:**
- Change storage backend without touching business logic
- Easy to test with both implementations
- One code path for all scenarios

---

## Summary of Violations

| Violation | Current | Target |
|-----------|---------|--------|
| **God Handler** | 355-line integration.go doing 7 things | WebhookHandler (HTTP only) → VerifyUseCase → SubmitUseCase |
| **Global State** | 8 global variables | Constructor dependency injection |
| **Mixed Logic** | Business logic in handlers | Domain logic → Use cases → HTTP handlers |
| **Hidden Dependencies** | Must read whole file to understand | Explicit in struct fields |
| **Tight Coupling** | jobQueue.Submit() hardcoded | domain.JobRepository interface |
| **Testability** | Need full initialization for unit test | Mock one dependency, test logic |
| **Reusability** | Triage only works via HTTP | Triage use case works via HTTP, API, scheduler, CLI |

---

## Why This Matters

**Current State Issues:**
- 🔴 Adding new workflow requires modifying integration.go
- 🔴 Unit tests take 10+ seconds (full initialization)
- 🔴 Cannot test business logic without HTTP mocking
- 🔴 Cannot swap queue for database
- 🔴 Cannot use triage logic from CLI or scheduler
- 🔴 New team member: "where do I add new webhook handling?" → 20-minute investigation

**After Refactoring:**
- ✅ New workflow = new use case class (isolated, testable)
- ✅ Unit tests take <100ms (mocks)
- ✅ Business logic tests independent from HTTP
- ✅ Swap implementations in main.go
- ✅ Triage logic reusable everywhere
- ✅ New team member: "add use case in internal/application/workflows/"

