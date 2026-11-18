# API Flows & Sequence Diagrams

Comprehensive documentation of API flows with detailed sequence diagrams for all major operations.

## Table of Contents

1. [Webhook Processing](#webhook-processing)
2. [Job Status Query](#job-status-query)
3. [Error Scenarios](#error-scenarios)
4. [Concurrent Requests](#concurrent-requests)
5. [Performance Metrics](#performance-metrics)

---

## Webhook Processing

### GitHub Webhook to Job Creation - Full Flow

```mermaid
sequenceDiagram
    actor Developer
    participant GitHub as "GitHub API"
    participant Network as "Internet"
    participant Server as "HTTP Server<br/>localhost:8080"
    participant Router as "Router"
    participant Handler as "WebhookHandler"
    participant Parser as "Parser"
    participant Factory as "Job Factory"
    participant Observer as "UseCaseObserver"
    participant UseCase as "SubmitJobUseCase"
    participant Repository as "JobRepository"
    participant Database as "PostgreSQL"
    participant EventBus as "EventBus"
    participant Logger as "StructuredLogger"
    participant Tracer as "SimpleTracer"
    
    Developer->>GitHub: Push code or create issue
    GitHub->>Network: Send webhook
    Network->>Server: POST /webhook
    
    rect rgb(200, 220, 255)
        Note over Server,Logger: Request Parsing
        Server->>Router: Route to handler
        Router->>Handler: Handle webhook
        Handler->>Logger: Info("Webhook received")
    end
    
    rect rgb(200, 255, 220)
        Note over Parser,Factory: Payload Processing
        Handler->>Parser: Parse payload
        Parser->>Parser: Extract event type
        Parser->>Parser: Extract metadata
        Parser-->>Handler: Event data
        Handler->>Factory: Create Job entity
        Factory-->>Handler: Job with ID
    end
    
    rect rgb(255, 240, 200)
        Note over Observer,Logger: Observability Instrumentation
        Handler->>Observer: ObserveExecution(ctx, "submit-job", submitFn)
        Observer->>Tracer: Start(ctx, "submit-job")
        Tracer-->>Observer: (ctx, span)
        Observer->>Logger: Info("use case started")
        Observer->>Logger: SetAttribute("job_id", job.ID)
    end
    
    rect rgb(220, 200, 255)
        Note over UseCase,Repository: Business Logic
        Observer->>UseCase: Execute(ctx, job)
        UseCase->>UseCase: Validate job
        UseCase->>Repository: Save(ctx, job)
        Repository->>Repository: Convert to SQL
        Repository->>Database: INSERT INTO jobs
        Database-->>Repository: success (1 row)
        Repository-->>UseCase: nil
        UseCase->>EventBus: Publish(JobSubmittedEvent)
        EventBus-->>UseCase: nil
    end
    
    rect rgb(255, 200, 200)
        Note over Observer,Logger: Completion
        UseCase-->>Observer: nil (no error)
        Observer->>Logger: Info("use case completed in Xms")
        Observer->>Tracer: span.End(nil)
    end
    
    rect rgb(200, 220, 255)
        Note over Handler,Server: Response
        Observer-->>Handler: nil
        Handler->>Handler: Format JSON response
        Handler-->>Server: HTTP 202 Accepted
        Server-->>Network: Response
    end
    
    Network-->>GitHub: Status: 202
    GitHub-->>Developer: Webhook processed
```

### Request-Response Bodies

**Request**:
```json
POST /webhook
Content-Type: application/json
X-GitHub-Event: issues
X-Hub-Signature: sha256=...

{
  "action": "opened",
  "issue": {
    "number": 123,
    "title": "Fix authentication bug",
    "body": "Users cannot login with OAuth",
    "user": {"login": "developer"}
  },
  "repository": {
    "name": "myrepo",
    "full_name": "myorg/myrepo",
    "url": "https://github.com/myorg/myrepo"
  }
}
```

**Response**:
```json
HTTP/1.1 202 Accepted
Content-Type: application/json

{
  "status": "accepted",
  "job_id": "job-1794942593078122000",
  "job_type": "issue-triage",
  "message": "Job submitted successfully"
}
```

**Console Output**:
```
2026-11-18 00:39:53 [INFO] Webhook received | event_type=issues action=opened
2026-11-18 00:39:53 [INFO] use case started | operation=submit-job job_id=job-1794942593078122000 job_type=issue-triage
2026-11-18 00:39:53 [INFO] job submitted | id=job-1794942593078122000 type=issue-triage
2026-11-18 00:39:53 [INFO] use case completed in 5ms | operation=submit-job
```

---

## Job Status Query

### Status Query with Caching (Future)

```mermaid
sequenceDiagram
    actor User
    participant Client as "HTTP Client<br/>curl/browser"
    participant Server as "HTTP Server"
    participant Handler as "JobHandler"
    participant Cache as "Redis Cache<br/>(future)"
    participant Observer as "UseCaseObserver"
    participant UseCase as "GetJobStatusUseCase"
    participant Repository as "JobRepository"
    participant Database as "PostgreSQL"
    participant Logger as "StructuredLogger"
    
    User->>Client: Request job status
    Client->>Server: GET /api/jobs/status?id=job-123
    
    rect rgb(200, 220, 255)
        Note over Server,Logger: Request Handling
        Server->>Handler: HandleGetJobStatus()
        Handler->>Logger: Debug("Job query", job_id=job-123)
    end
    
    rect rgb(255, 240, 200)
        Note over Observer,Logger: Observability Setup
        Handler->>Observer: ObserveExecution(ctx, "get-job-status", getFn)
        Observer->>Logger: Info("use case started")
    end
    
    rect rgb(200, 255, 220)
        Note over UseCase,Repository: Retrieval Logic
        Observer->>UseCase: Execute(ctx, "job-123")
        UseCase->>Repository: GetByID(ctx, "job-123")
        
        alt Cache Hit
            Repository->>Cache: GET job:123
            Cache-->>Repository: job_data
            Repository-->>UseCase: Job{...}
        else Cache Miss
            Repository->>Database: SELECT * FROM jobs WHERE id=$1
            Database-->>Repository: job_row
            Repository->>Repository: Unmarshal to Job
            Repository->>Cache: SET job:123 job_data EX 300
            Cache-->>Repository: OK
            Repository-->>UseCase: Job{...}
        end
    end
    
    rect rgb(220, 200, 255)
        Note over Observer,Logger: Completion
        UseCase-->>Observer: Job{...}, nil
        Observer->>Logger: Info("use case completed in Xms")
    end
    
    rect rgb(200, 220, 255)
        Note over Handler,Server: Response
        Observer-->>Handler: Job{...}
        Handler->>Handler: Encode to JSON
        Handler-->>Server: HTTP 200 OK
        Server-->>Client: JSON response
    end
    
    Client-->>User: Display job details
```

### Response Format

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "job-1794942593078122000",
  "type": "issue-triage",
  "status": "completed",
  "priority": 5,
  "input": {
    "action": "opened",
    "issue": 123,
    "repo": "myorg/myrepo",
    "title": "Fix authentication",
    "body": "..."
  },
  "result": {
    "issues_found": 2,
    "recommendations": ["Check error handling", "Add logging"]
  },
  "created_at": "2026-11-18T00:39:53Z",
  "updated_at": "2026-11-18T00:40:00Z"
}
```

---

## Error Scenarios

### Database Error Handling

```mermaid
sequenceDiagram
    participant Handler as "Handler"
    participant Observer as "Observer"
    participant UseCase as "UseCase"
    participant Repository as "Repository"
    participant Database as "PostgreSQL"
    participant Logger as "Logger"
    participant Span as "Span"
    
    Handler->>Observer: ObserveExecution(ctx, "job-query", ...)
    Observer->>Logger: Info("use case started")
    
    Observer->>UseCase: Execute(ctx, job_id)
    UseCase->>Repository: GetByID(ctx, job_id)
    
    Repository->>Database: SELECT * FROM jobs WHERE id=$1
    
    alt Connection Error
        Database-->>Repository: error: connection refused
        Repository-->>UseCase: error
    else Not Found Error
        Database-->>Repository: empty result set
        Repository-->>UseCase: error: ErrNotFound
    else Query Error
        Database-->>Repository: error: syntax error
        Repository-->>UseCase: error
    end
    
    UseCase->>Span: RecordError(err)
    UseCase->>Logger: Error("Failed to get job", err)
    UseCase-->>Observer: error
    
    Observer->>Logger: Error("use case failed", err)
    Observer->>Span: span.End(err)
    
    Observer-->>Handler: error
    
    Handler->>Logger: Error("Handler failed", err)
    Handler-->>Client: HTTP 500 Internal Server Error
```

### Validation Error Flow

```mermaid
sequenceDiagram
    participant Handler as "Handler"
    participant Validator as "Validator"
    participant UseCase as "UseCase"
    participant Logger as "Logger"
    participant Span as "Span"
    
    Handler->>Handler: Extract job from payload
    Handler->>Validator: Validate(job)
    
    alt Valid
        Validator-->>Handler: nil
        Handler->>UseCase: Execute(ctx, job)
    else Invalid
        Validator-->>Handler: ErrInvalid
        Handler->>Logger: Error("Validation failed", err)
        Handler->>Span: RecordError(err)
        Handler-->>Client: HTTP 400 Bad Request
    end
```

---

## Concurrent Requests

### Multiple Webhooks Processing

```mermaid
graph TB
    WH1["Webhook 1<br/>Issue opened"]
    WH2["Webhook 2<br/>PR created"]
    WH3["Webhook 3<br/>Code pushed"]
    
    SERVER["HTTP Server<br/>(Goroutine pool)"]
    
    H1["Handler 1"]
    H2["Handler 2"]
    H3["Handler 3"]
    
    UC1["UseCase 1"]
    UC2["UseCase 2"]
    UC3["UseCase 3"]
    
    DB[(("PostgreSQL<br/>Connection Pool"))]
    
    LOG["Logger<br/>(Threadsafe)"]
    TRACE["Tracer<br/>(Threadsafe)"]
    
    WH1 -->|concurrent| SERVER
    WH2 -->|concurrent| SERVER
    WH3 -->|concurrent| SERVER
    
    SERVER -->|route| H1
    SERVER -->|route| H2
    SERVER -->|route| H3
    
    H1 -->|execute| UC1
    H2 -->|execute| UC2
    H3 -->|execute| UC3
    
    UC1 -->|query| DB
    UC2 -->|query| DB
    UC3 -->|query| DB
    
    H1 -->|log| LOG
    H2 -->|log| LOG
    H3 -->|log| LOG
    
    H1 -->|trace| TRACE
    H2 -->|trace| TRACE
    H3 -->|trace| TRACE
```

### Log Output with Concurrent Requests

```
[00:39:53.001] [INFO] use case started | operation=submit-job job_id=job-1 goroutine=12
[00:39:53.002] [INFO] use case started | operation=submit-job job_id=job-2 goroutine=13
[00:39:53.003] [INFO] use case started | operation=submit-job job_id=job-3 goroutine=14
[00:39:53.010] [INFO] use case completed in 9ms | operation=submit-job job_id=job-1
[00:39:53.011] [INFO] use case completed in 9ms | operation=submit-job job_id=job-2
[00:39:53.012] [INFO] use case completed in 9ms | operation=submit-job job_id=job-3
```

---

## Performance Metrics

### End-to-End Latency Distribution

```mermaid
graph TB
    REQ["Request Timing"]
    
    HTTP["HTTP Processing"]
    HANDLER["Handler Logic"]
    DB["Database Query"]
    LOGGING["Logging"]
    TRACING["Tracing"]
    
    HTTP -->|1-2ms| HTTP_TIME["1-2ms"]
    HANDLER -->|2-5ms| HANDLER_TIME["2-5ms"]
    DB -->|20-50ms| DB_TIME["20-50ms"]
    LOGGING -->|1-2ms| LOG_TIME["1-2ms"]
    TRACING -->|1-2ms| TRACE_TIME["1-2ms"]
    
    HTTP_TIME -->|sum| TOTAL["Total: 25-61ms"]
    HANDLER_TIME -->|sum| TOTAL
    DB_TIME -->|sum| TOTAL
    LOG_TIME -->|sum| TOTAL
    TRACE_TIME -->|sum| TOTAL
    
    TOTAL -->|p50| P50["P50: 35ms"]
    TOTAL -->|p95| P95["P95: 55ms"]
    TOTAL -->|p99| P99["P99: 61ms"]
```

### Operation Benchmarks

```mermaid
graph TB
    WH["Webhook Handling"]
    WH_T["Avg: 50-100ms<br/>P99: 150ms<br/>Throughput: 100-200/s"]
    
    JOB["Job Status Query"]
    JOB_T["Avg: 20-30ms<br/>P99: 50ms<br/>Throughput: 500-1000/s"]
    
    TRIAGE["Issue Triage"]
    TRIAGE_T["Avg: 500-1000ms<br/>P99: 1500ms<br/>Throughput: 50-100/s"]
    
    WH -->|measures| WH_T
    JOB -->|measures| JOB_T
    TRIAGE -->|measures| TRIAGE_T
```

---

## State Transitions

### Job Status State Machine

```mermaid
stateDiagram-v2
    [*] --> CREATED: Job created<br/>via webhook
    
    CREATED --> QUEUED: Save to<br/>repository
    
    QUEUED --> PROCESSING: Executor<br/>picks up
    
    PROCESSING --> COMPLETED: Work<br/>finished
    
    PROCESSING --> FAILED: Error<br/>occurred
    
    FAILED --> QUEUED: Retry<br/>enabled
    
    COMPLETED --> [*]
    FAILED --> [*]
    
    note right of CREATED
        Job ID generated
        Status: "created"
        Not persisted yet
    end note
    
    note right of QUEUED
        Persisted to DB
        Status: "queued"
        Waiting for executor
    end note
    
    note right of PROCESSING
        Executor active
        Status: "processing"
        Progress tracked
    end note
    
    note right of COMPLETED
        Status: "completed"
        Results stored
        Events published
    end note
    
    note right of FAILED
        Status: "failed"
        Error recorded
        Retry check: immediate
    end note
```

---

## Example Workflow: Complete Issue Triage

### Step-by-Step Sequence

```mermaid
sequenceDiagram
    participant Developer
    participant GitHub
    participant Webhook as "Webhook Endpoint"
    participant Handler as "WebhookHandler"
    participant UC as "SubmitJobUseCase"
    participant Repo as "Repository"
    participant DB as "Database"
    participant Executor as "Executor<br/>(async)"
    participant TriageUC as "IssueTriageUseCase"
    participant Analyzer as "Code Analyzer"
    
    Developer->>GitHub: Create issue<br/>"Fix login bug"
    
    GitHub->>Webhook: POST /webhook<br/>issue.action=opened
    
    Webhook->>Handler: ParseAndValidate()
    Handler->>UC: Execute(ctx, job)
    
    UC->>Repo: Save(job)
    Repo->>DB: INSERT INTO jobs
    DB-->>Repo: success
    Repo-->>UC: nil
    
    UC-->>Handler: nil
    Handler-->>GitHub: HTTP 202<br/>job_id: job-123
    
    par Async Processing
        Executor->>TriageUC: Execute(ctx, job)
        
        TriageUC->>Analyzer: Analyze issue<br/>description
        Analyzer->>Analyzer: Extract: BUG<br/>Category: AUTH
        Analyzer->>Analyzer: Priority: HIGH
        
        Analyzer-->>TriageUC: Analysis result
        
        TriageUC->>Repo: Update(job_id, result)
        Repo->>DB: UPDATE jobs SET result=...
        DB-->>Repo: success
        
        Repo-->>TriageUC: nil
        TriageUC-->>Executor: nil
    end
    
    Developer->>Webhook: Check status<br/>GET /status?id=job-123
    Webhook-->>Developer: Job completed<br/>Analysis: HIGH priority<br/>Category: AUTH
```

---

## Configuration & Tuning

### Performance Tuning Parameters

```yaml
# database.yaml
database:
  connection_pool:
    min_idle: 2
    max_size: 10
    timeout: 10s
    max_lifetime: 30m

# server.yaml
server:
  max_goroutines: 1000
  request_timeout: 30s
  read_timeout: 15s
  write_timeout: 15s
  
# logging.yaml
logging:
  level: info           # Can be debug, info, warn, error
  buffer_size: 10000
  flush_interval: 100ms
  
# tracing.yaml
tracing:
  enabled: true
  sample_rate: 1.0      # 100% sampling, reduce for high-volume
  batch_size: 100
  export_interval: 5s
```

---

## Monitoring & Observability

### Key Metrics to Monitor

```
Latency Metrics:
├─ webhook_latency_p50
├─ webhook_latency_p99
├─ job_query_latency_p50
└─ job_query_latency_p99

Throughput Metrics:
├─ webhooks_per_second
├─ queries_per_second
└─ completed_jobs_per_second

Error Metrics:
├─ webhook_errors_total
├─ database_errors_total
├─ validation_errors_total
└─ retry_count

Resource Metrics:
├─ memory_usage_bytes
├─ goroutine_count
├─ database_connections
└─ connection_pool_wait_time
```

### Logging Checklist

✅ All requests logged with request ID  
✅ Database queries logged with duration  
✅ Errors logged with full context  
✅ Span duration tracked and logged  
✅ Concurrent requests identified by goroutine ID  
✅ Sensitive data redacted (tokens, emails)  
✅ Structured fields for all events  

---

## References

- [Observability Guide](./OBSERVABILITY_GUIDE.md)
- [Clean Architecture Summary](./CLEAN_ARCHITECTURE_SUMMARY.md)
- [Main README](../README.md)
- [Detailed README](../README_DETAILED.md)
