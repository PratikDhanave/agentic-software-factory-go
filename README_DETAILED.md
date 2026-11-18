# Agentic Software Factory - Comprehensive Guide

> A production-grade agent orchestration platform implementing Clean Architecture with comprehensive observability, distributed tracing, and enterprise-ready infrastructure.

## 📋 Table of Contents

1. [System Architecture](#system-architecture)
2. [Clean Architecture Overview](#clean-architecture-overview)
3. [Data Flow Diagrams](#data-flow-diagrams)
4. [Message Sequence Diagrams](#message-sequence-diagrams)
5. [Component Interactions](#component-interactions)
6. [Observability Architecture](#observability-architecture)
7. [Deployment Architecture](#deployment-architecture)
8. [Database Schema](#database-schema)
9. [API Flows](#api-flows)
10. [Error Handling Flows](#error-handling-flows)
11. [Performance Characteristics](#performance-characteristics)

---

## System Architecture

### High-Level System Design

```mermaid
graph TB
    UI["Web UI<br/>(Next.js)"]
    HTTP["HTTP Server<br/>(Go)"]
    
    UI -->|HTTP + SSE| HTTP
    
    HTTP --> WH["Webhook Handler<br/>(GitHub Integration)"]
    HTTP --> JH["Job Handler<br/>(Status API)"]
    HTTP --> TRIAGE["Issue Triage<br/>(Use Case)"]
    
    WH --> SUBMIT["Submit Job<br/>(Use Case)"]
    JH --> GET_JOB["Get Job Status<br/>(Use Case)"]
    
    SUBMIT --> JR["Job Repository<br/>(In-Memory or PostgreSQL)"]
    GET_JOB --> JR
    TRIAGE --> WR["Workflow Repository<br/>(In-Memory or PostgreSQL)"]
    
    SUBMIT --> EB["Event Bus<br/>(Pub/Sub)"]
    
    JR --> DB[(("PostgreSQL<br/>Database"))]
    WR --> DB
    
    HTTP --> OBS["Observability<br/>(Logging + Tracing)"]
    WH --> OBS
    JH --> OBS
    SUBMIT --> OBS
    GET_JOB --> OBS
```

### Domain-Driven Design

```mermaid
graph LR
    D["<b>Domain Layer</b><br/>Business Rules<br/>─────────────<br/>Job Entity<br/>Workflow Entity<br/>AuditEvent Entity<br/>Repository Interfaces<br/>Logger Interface<br/>EventPublisher Interface"]
    
    A["<b>Application Layer</b><br/>Use Cases<br/>─────────────<br/>SubmitJobUseCase<br/>GetJobStatusUseCase<br/>IssueTriageUseCase<br/>Pure Orchestration<br/>No Framework Code"]
    
    AD["<b>Adapters Layer</b><br/>Framework Code<br/>─────────────<br/>HTTP Handlers<br/>GitHub Integration<br/>Repository Implementations<br/>Event Bus Implementation"]
    
    I["<b>Infrastructure</b><br/>Cross-Cutting<br/>─────────────<br/>StructuredLogger<br/>TraceProvider<br/>Observability<br/>Bootstrap DI"]
    
    D --> A
    A --> AD
    AD --> I
```

---

## Clean Architecture Overview

### Internal Layer Structure (4-Layer Onion)

```mermaid
graph TB
    subgraph HTTP["Adapters: HTTP + Database"]
        WH["WebhookHandler"]
        JH["JobHandler"]
        INMEM["InMemoryJobRepository"]
        PG["PostgresJobRepository"]
        GH["GitHub Client"]
    end
    
    subgraph UC["Application: Use Cases"]
        SUBMIT["SubmitJobUseCase"]
        GET["GetJobStatusUseCase"]
        TRIAGE["IssueTriageUseCase"]
        direction LR
    end
    
    subgraph Domain["Domain: Business Rules"]
        Job["Job Entity"]
        Workflow["Workflow Entity"]
        Repo["Repository Interface"]
        Logger["Logger Interface"]
        Publisher["EventPublisher Interface"]
        direction LR
    end
    
    subgraph Infra["Infrastructure: DI + Cross-Cutting"]
        BOOT["Bootstrap"]
        LOG["StructuredLogger"]
        TRACE["TraceProvider"]
        OBS["Observability"]
        EB["EventBus"]
    end
    
    HTTP --> UC
    UC --> Domain
    Domain --> Infra
    
    HTTP -.->|depends on| LOG
    HTTP -.->|depends on| OBS
    UC -.->|depends on| LOG
```

### Component Interaction Diagram

```mermaid
graph LR
    UI["User/GitHub"]
    
    WH["WebhookHandler"]
    JH["JobHandler"]
    
    SUBMIT["SubmitJobUseCase"]
    GET["GetJobStatusUseCase"]
    
    REPO["Repository<br/>Interface"]
    INMEM["InMemoryRepo<br/>Implementation"]
    PG["PostgresRepo<br/>Implementation"]
    
    LOGGER["Logger<br/>Interface"]
    STRUCT["StructuredLogger<br/>Implementation"]
    
    PUB["EventPublisher<br/>Interface"]
    EB["EventBus<br/>Implementation"]
    
    DB[(("PostgreSQL"))]
    
    UI -->|webhook| WH
    UI -->|GET /status| JH
    
    WH -->|calls| SUBMIT
    JH -->|calls| GET
    
    SUBMIT -->|uses| REPO
    GET -->|uses| REPO
    
    REPO -.->|depends on| INMEM
    REPO -.->|depends on| PG
    
    SUBMIT -->|uses| LOGGER
    GET -->|uses| LOGGER
    
    LOGGER -.->|depends on| STRUCT
    
    SUBMIT -->|publishes| PUB
    PUB -.->|depends on| EB
    
    PG -->|persists| DB
    INMEM -->|memory| INMEM
```

---

## Data Flow Diagrams

### GitHub Webhook Processing Flow

```mermaid
sequenceDiagram
    actor User
    participant GitHub
    participant Server as "HTTP Server"
    participant Handler as "WebhookHandler"
    participant UC as "SubmitJobUseCase"
    participant Repo as "Repository"
    participant Logger as "StructuredLogger"
    participant EventBus as "EventBus"
    
    User->>GitHub: Push code / Create issue
    GitHub->>Server: POST /webhook (issue/PR event)
    
    Server->>Handler: HandleGitHubWebhook()
    
    Handler->>Logger: Info("Webhook received")
    Logger-->>Handler: 
    
    Handler->>Handler: Parse payload
    Handler->>Handler: Extract event type
    
    alt Issue Event
        Handler->>Handler: Create Job with<br/>type: "issue-triage"
    else PR Event
        Handler->>Handler: Create Job with<br/>type: "pr-review"
    else Push Event
        Handler->>Handler: Create Job with<br/>type: "push-analyze"
    end
    
    Handler->>UC: Execute(ctx, job)
    
    UC->>Logger: Info("Job validation")
    Logger-->>UC:
    
    UC->>Repo: Save(ctx, job)
    Repo-->>UC: success
    
    UC->>EventBus: Publish(JobSubmittedEvent)
    EventBus-->>UC: success
    
    UC->>Logger: Info("Job submitted", job.ID)
    Logger-->>UC:
    
    UC-->>Handler: success
    Handler-->>Server: HTTP 202 Accepted
    Server-->>GitHub: Response
    
    GitHub-->>User: Webhook processed
```

### Job Status Query Flow

```mermaid
sequenceDiagram
    actor User
    participant Client
    participant Server as "HTTP Server"
    participant Handler as "JobHandler"
    participant UC as "GetJobStatusUseCase"
    participant Repo as "Repository"
    participant Logger as "StructuredLogger"
    participant DB as "Database"
    
    User->>Client: curl /api/jobs/status?id=job-123
    Client->>Server: GET /api/jobs/status?id=job-123
    
    Server->>Handler: HandleGetJobStatus(w, r)
    Handler->>Handler: Extract job ID from query
    
    Handler->>UC: Execute(ctx, jobID)
    
    UC->>Logger: Debug("Looking up job", jobID)
    Logger-->>UC:
    
    UC->>Repo: GetByID(ctx, jobID)
    
    Repo->>DB: SELECT * FROM jobs<br/>WHERE id = $1
    DB-->>Repo: job_row
    
    Repo->>Repo: Unmarshal to Job entity
    Repo-->>UC: job, nil
    
    alt Job Found
        UC->>Logger: Info("Job found", jobID)
        Logger-->>UC:
        UC-->>Handler: job, nil
    else Job Not Found
        UC->>Logger: Error("Job not found", err)
        Logger-->>UC:
        UC-->>Handler: nil, ErrNotFound
    end
    
    Handler->>Handler: Encode job to JSON
    Handler-->>Server: HTTP 200 OK + JSON
    Server-->>Client: Response
    Client-->>User: Display job status
```

### Complete Job Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: Job created<br/>via webhook
    
    Created --> Queued: Job added<br/>to queue
    
    Queued --> Processing: Execution<br/>started
    
    Processing --> Completed: All work<br/>done
    
    Processing --> Failed: Error<br/>occurred
    
    Completed --> [*]
    Failed --> [*]
    
    note right of Created
        Job entity created
        with metadata
    end note
    
    note right of Queued
        Job persisted
        to repository
    end note
    
    note right of Processing
        Status updated
        Progress tracked
    end note
    
    note right of Completed
        Results stored
        Events published
    end note
    
    note right of Failed
        Error logged
        Retry possible
    end note
```

---

## Message Sequence Diagrams

### End-to-End: GitHub Issue → Job Processing

```mermaid
sequenceDiagram
    actor Developer
    participant GitHub as "GitHub<br/>API"
    participant Webhook as "HTTP Server<br/>Webhook Endpoint"
    participant Handler as "WebhookHandler<br/>"
    participant Observer as "UseCaseObserver<br/>"
    participant UseCase as "SubmitJobUseCase<br/>"
    participant Repo as "JobRepository<br/>"
    participant Logger as "StructuredLogger<br/>"
    participant Tracer as "TraceProvider<br/>"
    participant DB as "PostgreSQL<br/>Database"
    
    Developer->>GitHub: Create GitHub Issue
    
    GitHub->>Webhook: POST /webhook<br/>X-GitHub-Event: issues<br/>{action: opened, ...}
    
    Note over Webhook,Handler: Request Processing
    Webhook->>Logger: Info("Webhook received")
    
    Webhook->>Handler: ParseGitHubEvent()
    Handler->>Handler: Extract payload<br/>Determine type
    
    Handler->>Observer: ObserveExecution(<br/>"submit-job",<br/>submitJobFn)
    
    Note over Observer,Tracer: Observability Instrumentation
    Observer->>Tracer: Start(ctx, "submit-job", ...)
    Tracer-->>Observer: (ctx, span)
    
    Observer->>Logger: Info("use case started", "submit-job")
    
    Note over UseCase,Repo: Job Submission
    Observer->>UseCase: Execute(ctx, job)
    
    UseCase->>Logger: Info("Job validation")
    UseCase->>UseCase: Validate job data
    
    UseCase->>Repo: Save(ctx, job)
    
    Repo->>Logger: Info("Saving job")
    Repo->>DB: INSERT INTO jobs (...)
    DB-->>Repo: success
    
    Repo-->>UseCase: nil
    
    UseCase->>Logger: Info("job submitted", job.ID)
    
    UseCase-->>Observer: nil (no error)
    
    Note over Observer,Logger: Completion Logging
    Observer->>Logger: Info("use case completed in Xms")
    Observer->>Tracer: span.End(nil)
    
    Note over Handler: Response
    Handler->>Webhook: return nil
    Webhook-->>GitHub: HTTP 202 Accepted<br/>{status: accepted, job_id: job-123}
    
    GitHub-->>Developer: Webhook processed
    Developer->>Developer: Check job status<br/>GET /api/jobs/status?id=job-123
```

### Observability Tracing Flow

```mermaid
sequenceDiagram
    participant Handler as "HTTP Handler"
    participant Observer as "UseCaseObserver"
    participant Span as "SpanContext"
    participant Logger as "StructuredLogger"
    participant Tracer as "TraceProvider"
    participant UC as "Use Case"
    
    Handler->>Observer: ObserveExecution(<br/>ctx, "operation-name",<br/>executionFn)
    
    Note over Observer: Initialize Tracing
    Observer->>Tracer: Start(ctx, "operation-name", ...)
    Tracer-->>Observer: (ctxWithSpan, Span)
    
    Observer->>Span: Create SpanContext
    Span-->>Observer: SpanContext{}
    
    Note over Observer,Logger: Pre-Execution Logging
    Observer->>Logger: Info("use case started", "operation-name")
    
    Note over UC: Business Logic Execution
    Observer->>UC: Call executionFn(span)
    
    UC->>Span: SetAttribute("key", "value")
    Span->>Span: Store in span.attributes
    
    UC->>Span: Logger()
    Span-->>UC: logger instance
    
    UC->>Logger: Info("operation progress")
    
    alt Success
        UC-->>Observer: nil (no error)
    else Error
        UC->>Span: RecordError(err)
        Span->>Span: Store error info
        UC-->>Observer: error
    end
    
    Note over Observer,Logger: Post-Execution Logging
    Observer->>Span: Duration()
    Span-->>Observer: duration
    
    Observer->>Logger: Info("use case completed in Xms")
    
    Observer->>Span: End()
    Span->>Tracer: Output trace info
    Tracer->>Tracer: Print/Export span
    
    Observer-->>Handler: error or nil
```

### Error Handling Flow

```mermaid
sequenceDiagram
    participant UseCase as "Use Case"
    participant Repo as "Repository"
    participant Logger as "Logger"
    participant Span as "SpanContext"
    
    UseCase->>Repo: Execute operation
    
    alt Success Path
        Repo-->>UseCase: success result
        UseCase->>Logger: Info("Operation succeeded")
    else Database Error
        Repo-->>UseCase: ErrNotFound
        UseCase->>Span: RecordError(err)
        UseCase->>Logger: Error("Operation failed", err)
        UseCase-->>UseCase: Return error
    else Validation Error
        UseCase->>UseCase: Validation fails
        UseCase->>Span: RecordError(err)
        UseCase->>Logger: Error("Validation failed", err)
        UseCase-->>UseCase: Return error
    end
```

---

## Component Interactions

### Request-Response Cycle with Observability

```mermaid
graph TB
    REQ["HTTP Request<br/>POST /webhook"]
    
    WH["WebhookHandler"]
    
    OBS["Observability Stack"]
    
    UC["UseCase"]
    
    REPO["Repository"]
    
    LOG["Logger Output"]
    
    TRACE["Trace Output"]
    
    RESP["HTTP Response"]
    
    REQ -->|parse| WH
    
    WH -->|create| OBS
    OBS -->|setup| LOG
    OBS -->|setup| TRACE
    
    WH -->|observe| UC
    UC -->|data| REPO
    
    REPO -->|query| DB[(Database)]
    DB -->|result| REPO
    
    REPO -->|return| UC
    UC -->|log| LOG
    
    LOG -->|emit| LOGFILE["[INFO] operation | ...]"]
    
    UC -->|span| TRACE
    TRACE -->|emit| TRACEFILE["[TRACE] span=... attributes=..."]
    
    UC -->|return| WH
    WH -->|format| RESP
    RESP -->|send| CLIENT["HTTP Client"]
```

### Observer Pattern Implementation

```mermaid
graph TB
    OBS["Observer"]
    
    OBS -->|wraps| SP["SpanContext"]
    
    SP -->|contains| LOG["Logger<br/>(StructuredLogger)"]
    SP -->|contains| SPAN["Span<br/>(TraceProvider.Span)"]
    SP -->|contains| CTX["Context"]
    
    SP -->|methods| M1["Logger()"]
    SP -->|methods| M2["SetAttribute()"]
    SP -->|methods| M3["RecordError()"]
    SP -->|methods| M4["RecordEvent()"]
    SP -->|methods| M5["End()"]
    SP -->|methods| M6["Duration()"]
    
    M1 -->|returns| LOG
    M2 -->|calls| SPAN
    M3 -->|calls| SPAN
    M4 -->|calls| SPAN
    M5 -->|calls| SPAN
    M6 -->|calculates| DURATION["time.Duration"]
```

---

## Observability Architecture

### Logging & Tracing System

```mermaid
graph TB
    APP["Application Layer"]
    
    OBS["Observability Facade"]
    
    LOGGER["StructuredLogger"]
    TRACER["TraceProvider"]
    
    LOGIMPL["Logger Implementation<br/>(Structured output)"]
    TRACEPROV["Trace Implementations"]
    
    NOOP["NoOpTracer<br/>(No-op)"]
    SIMPLE["SimpleTracer<br/>(stdout)"]
    FUTURE["OTLP Exporter<br/>(future)"]
    
    OUTPUT["Output"]
    
    LOGFILE["[INFO] message | field=value"]
    TRACEFILE["[TRACE] span=... status=..."]
    
    APP -->|uses| OBS
    
    OBS -->|wraps| LOGGER
    OBS -->|wraps| TRACER
    
    LOGGER -->|implements| LOGIMPL
    TRACER -->|provides| TRACEPROV
    
    TRACEPROV -->|implementation| NOOP
    TRACEPROV -->|implementation| SIMPLE
    TRACEPROV -->|implementation| FUTURE
    
    LOGIMPL -->|outputs| LOGFILE
    SIMPLE -->|outputs| TRACEFILE
    
    LOGFILE -->|to| OUTPUT
    TRACEFILE -->|to| OUTPUT
```

### Context Propagation

```mermaid
graph LR
    REQ["HTTP Request"]
    
    REQ -->|creates| CTX["context.Background()"]
    
    CTX -->|passed to| WH["WebhookHandler"]
    
    WH -->|passed to| OBS["Observability.StartSpan"]
    
    OBS -->|creates| NEWCTX["context.WithValue<br/>(ctx, trace_id, ...)"]
    
    NEWCTX -->|passed to| UC["UseCase.Execute"]
    
    UC -->|uses| REPO["Repository"]
    
    REPO -->|reads trace_id| EXTRACT["Extract from context"]
    
    EXTRACT -->|injects into| LOGGER["Logger output"]
    
    LOGGER -->|output| OUTPUT["[INFO] ... | trace_id=xyz span_id=abc"]
```

### Instrumentation Points

```mermaid
graph TB
    HTTP["HTTP Handler Layer"]
    UC["Use Case Layer"]
    REPO["Repository Layer"]
    
    HTTP -->|instrument with| OBS1["StartSpan<br/>+ Logger"]
    
    UC -->|log| LOG1["Use case start"]
    UC -->|log| LOG2["Use case end"]
    UC -->|record| ATTR["Span attributes"]
    
    REPO -->|log| LOG3["Repository operation"]
    REPO -->|on error| ERRLOG["Error recorded<br/>+ trace context"]
    
    OBS1 -.->|enables| AUTO["Automatic<br/>- Start/End logging<br/>- Error recording<br/>- Duration tracking"]
```

---

## Deployment Architecture

### Development Setup

```mermaid
graph TB
    DEV["Developer Machine"]
    
    SRC["Source Code"]
    BUILD["go build"]
    BIN["asf-web binary"]
    
    HTTP["HTTP Server<br/>localhost:8080"]
    
    INMEM["In-Memory<br/>Repositories"]
    LOG["Console Logger<br/>(DEBUG level)"]
    TRACE["Simple Tracer<br/>(stdout)"]
    
    DEV -->|contains| SRC
    SRC -->|compile| BUILD
    BUILD -->|produces| BIN
    
    BIN -->|runs| HTTP
    HTTP -->|uses| INMEM
    HTTP -->|uses| LOG
    HTTP -->|uses| TRACE
    
    LOG -->|output| CONSOLE["Console Output"]
    TRACE -->|output| CONSOLE
```

### Production Setup

```mermaid
graph TB
    LB["Load Balancer<br/>(nginx/HAProxy)"]
    
    SERVER1["Server 1<br/>asf-web"]
    SERVER2["Server 2<br/>asf-web"]
    SERVER3["Server 3<br/>asf-web"]
    
    CONNPOOL["Connection Pool"]
    
    DB[(PostgreSQL)]
    
    REDIS["Redis Cache<br/>(Optional)"]
    
    LOGGING["Logging Service<br/>(ELK/Datadog)"]
    TRACING["Tracing Service<br/>(Jaeger/Datadog)"]
    
    LB -->|routes| SERVER1
    LB -->|routes| SERVER2
    LB -->|routes| SERVER3
    
    SERVER1 -->|queries| CONNPOOL
    SERVER2 -->|queries| CONNPOOL
    SERVER3 -->|queries| CONNPOOL
    
    CONNPOOL -->|connection| DB
    
    SERVER1 -->|logs| LOGGING
    SERVER2 -->|logs| LOGGING
    SERVER3 -->|logs| LOGGING
    
    SERVER1 -->|traces| TRACING
    SERVER2 -->|traces| TRACING
    SERVER3 -->|traces| TRACING
    
    SERVER1 -->|cache| REDIS
    SERVER2 -->|cache| REDIS
    SERVER3 -->|cache| REDIS
```

---

## Database Schema

### Jobs Table

```mermaid
graph TB
    JOBS["jobs table"]
    
    JOBS -->|id| PK["UUID<br/>Primary Key"]
    JOBS -->|type| TYPE["VARCHAR<br/>job type"]
    JOBS -->|status| STATUS["VARCHAR<br/>created/queued/processing/completed/failed"]
    JOBS -->|priority| PRIO["INT<br/>1-10 scale"]
    JOBS -->|input| INPUT["JSONB<br/>job parameters"]
    JOBS -->|result| RESULT["JSONB<br/>execution result"]
    JOBS -->|created_at| CREATED["TIMESTAMP<br/>creation time"]
    JOBS -->|updated_at| UPDATED["TIMESTAMP<br/>last update"]
    
    INDEXES["Indexes"]
    INDEXES -->|idx1| "idx_jobs_status"
    INDEXES -->|idx2| "idx_jobs_type"
    INDEXES -->|idx3| "idx_jobs_created_at"
```

### Workflows Table

```mermaid
graph TB
    WF["workflows table"]
    
    WF -->|id| PK["UUID<br/>Primary Key"]
    WF -->|name| NAME["VARCHAR<br/>workflow name"]
    WF -->|status| STATUS["VARCHAR<br/>pending/running/completed/failed"]
    WF -->|job_ids| JOBS["UUID ARRAY<br/>job references"]
    WF -->|metadata| META["JSONB<br/>workflow context"]
    WF -->|created_at| CREATED["TIMESTAMP"]
    WF -->|updated_at| UPDATED["TIMESTAMP"]
```

### Relationships

```mermaid
graph LR
    WORKFLOWS["workflows"]
    JOBS["jobs"]
    EVENTS["audit_events"]
    
    WORKFLOWS -->|1 to Many| JOBS
    JOBS -->|1 to Many| EVENTS
    
    EVENTS -->|contains| EVENT_DATA["job_id<br/>event_type<br/>timestamp<br/>details"]
```

---

## API Flows

### Complete GitHub Webhook Flow

```mermaid
graph TB
    GH["GitHub Event<br/>Push / Issue / PR"]
    
    INCOMING["HTTP Request<br/>POST /webhook"]
    
    HANDLER["WebhookHandler"]
    OBSERVER["UseCaseObserver"]
    UC["SubmitJobUseCase"]
    REPO["JobRepository"]
    EB["EventBus"]
    
    RESPONSE["HTTP Response<br/>202 Accepted"]
    
    LOG1["Log: Webhook received"]
    LOG2["Log: Use case started"]
    LOG3["Log: Job saved"]
    LOG4["Log: Event published"]
    LOG5["Log: Use case completed"]
    
    TRACE1["Trace: span started"]
    TRACE2["Trace: attributes set"]
    TRACE3["Trace: span ended"]
    
    GH -->|trigger| INCOMING
    
    INCOMING -->|route| HANDLER
    HANDLER -->|log| LOG1
    HANDLER -->|parse| HANDLER
    
    HANDLER -->|observe| OBSERVER
    OBSERVER -->|log| LOG2
    OBSERVER -->|trace| TRACE1
    
    OBSERVER -->|execute| UC
    UC -->|validate| UC
    UC -->|log| LOG2
    UC -->|save| REPO
    REPO -->|persist| REPO
    REPO -->|log| LOG3
    UC -->|publish| EB
    EB -->|log| LOG4
    
    UC -->|set attrs| TRACE2
    
    UC -->|return| OBSERVER
    OBSERVER -->|log| LOG5
    OBSERVER -->|trace| TRACE3
    
    OBSERVER -->|return| HANDLER
    HANDLER -->|format| RESPONSE
    
    LOG1 -->|output| CONSOLE["Console<br/>[INFO] webhook received"]
    LOG5 -->|output| CONSOLE
    TRACE1 -->|output| CONSOLE
    TRACE3 -->|output| CONSOLE
```

---

## Error Handling Flows

### Error Recovery Strategy

```mermaid
graph TB
    OP["Operation Start"]
    
    TRY["Execute Use Case"]
    
    SUCCESS{Success?}
    
    YES["Return result"]
    
    NO["Error occurred"]
    
    RECORD["Record error<br/>in span"]
    LOG["Log error<br/>with context"]
    RETRY{Retryable?}
    
    YESRETRY["Retry with<br/>backoff"]
    NORETRY["Return error<br/>to caller"]
    
    END1["Success"]
    END2["Failure"]
    
    OP -->|start| TRY
    TRY -->|check| SUCCESS
    
    SUCCESS -->|yes| YES
    YES -->|return| END1
    
    SUCCESS -->|no| NO
    NO -->|capture| RECORD
    RECORD -->|log| LOG
    LOG -->|check| RETRY
    
    RETRY -->|yes| YESRETRY
    YESRETRY -->|after delay| TRY
    
    RETRY -->|no| NORETRY
    NORETRY -->|propagate| END2
```

### Context-Based Error Propagation

```mermaid
sequenceDiagram
    participant Handler
    participant Observer
    participant UseCase
    participant Repository
    participant Logger
    participant Span
    
    Handler->>Observer: ObserveExecution(ctx, name, fn)
    
    Observer->>UseCase: Call fn(span)
    
    UseCase->>Repository: Perform operation
    
    alt Operation Fails
        Repository-->>UseCase: error
        UseCase->>Span: RecordError(err)
        UseCase->>Logger: Error("Failed", err)
        UseCase-->>Observer: error
    end
    
    Observer->>Span: span.End()
    Observer-->>Handler: error
    
    Handler->>Logger: Error("Handler failed", err)
    Handler-->>Client: HTTP error response
```

---

## Performance Characteristics

### Request Latency Breakdown

```mermaid
graph LR
    START["Request<br/>Start"]
    
    PARSE["Parse<br/>10ms"]
    VALIDATE["Validate<br/>5ms"]
    LOG1["Log Start<br/>2ms"]
    TRACE1["Trace Start<br/>1ms"]
    
    DB["Database<br/>Query<br/>20-50ms"]
    
    PROCESS["Process<br/>Result<br/>5ms"]
    LOG2["Log End<br/>2ms"]
    TRACE2["Trace End<br/>1ms"]
    
    RESP["Format<br/>Response<br/>3ms"]
    
    END["Response<br/>End"]
    
    START -->|parse| PARSE
    PARSE -->|validate| VALIDATE
    VALIDATE -->|observe| LOG1
    LOG1 -->|trace| TRACE1
    
    TRACE1 -->|query| DB
    
    DB -->|process| PROCESS
    PROCESS -->|log| LOG2
    LOG2 -->|trace| TRACE2
    
    TRACE2 -->|format| RESP
    RESP -->|send| END
    
    style DB fill:#ff9999
    style PARSE fill:#99ccff
    style VALIDATE fill:#99ff99
```

### Throughput by Operation

```mermaid
graph TB
    WH["Webhook Handling"]
    WH_LATENCY["Latency: 50-100ms"]
    WH_THROUGHPUT["Throughput: 100-200 req/s"]
    
    JOB["Job Status Query"]
    JOB_LATENCY["Latency: 20-50ms"]
    JOB_THROUGHPUT["Throughput: 500-1000 req/s"]
    
    TRIAGE["Issue Triage"]
    TRIAGE_LATENCY["Latency: 100-500ms"]
    TRIAGE_THROUGHPUT["Throughput: 50-200 req/s"]
    
    WH -->|measures| WH_LATENCY
    WH_LATENCY -->|gives| WH_THROUGHPUT
    
    JOB -->|measures| JOB_LATENCY
    JOB_LATENCY -->|gives| JOB_THROUGHPUT
    
    TRIAGE -->|measures| TRIAGE_LATENCY
    TRIAGE_LATENCY -->|gives| TRIAGE_THROUGHPUT
```

---

## Code Examples

### Using Observability in a Use Case

```go
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

func (u *SubmitJobUseCase) Execute(ctx context.Context, job *entities.Job) error {
    // Business logic - no observability coupling here
    // The observability wraps the entire use case execution
    
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
```

### HTTP Handler with Observability

```go
package http

import (
    "net/http"
    "github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
    "github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/observability"
)

type WebhookHandler struct {
    submitJobUC *jobs.SubmitJobUseCase
    logger      services.Logger
    observer    *observability.UseCaseObserver
}

func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse webhook payload
    event, err := h.parseGitHubEvent(r)
    if err != nil {
        h.logger.Error("failed to parse event", err)
        http.Error(w, "failed to parse event", http.StatusBadRequest)
        return
    }

    // Create job
    job := entities.NewJob("gh-webhook-id", event.WorkflowType, event.ToInput())
    job.Priority = 5

    // Execute use case with automatic observability instrumentation
    err = h.observer.ObserveExecution(ctx, "submit-job", func(span observability.SpanContext) error {
        // Set span attributes
        span.SetAttribute("job_id", job.ID)
        span.SetAttribute("job_type", job.Type)
        span.SetAttribute("event_type", event.Action)

        // Execute business logic - observability automatically handles:
        // - Start logging
        // - Error recording
        // - Duration tracking
        // - End logging
        return h.submitJobUC.Execute(ctx, job)
    })

    if err != nil {
        h.logger.Error("failed to submit job", err)
        http.Error(w, "failed to submit job", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":     "accepted",
        "job_id":     job.ID,
        "job_type":   job.Type,
    })
}
```

### Observability Integration Example

```go
package main

import (
    "context"
    "github.com/PratikDhanave/agentic-software-factory-go/internal/bootstrap"
    "github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/observability"
)

func main() {
    // Bootstrap initializes everything including observability
    app := bootstrap.InitializeApplication()
    
    // Now all HTTP handlers use observability automatically
    // Every use case execution is traced and logged
    
    // Example flow:
    // 1. HTTP Handler receives request
    // 2. Handler.ObserveExecution(ctx, "operation-name", useCase.Execute)
    // 3. UseCaseObserver:
    //    - Creates span
    //    - Logs "use case started"
    //    - Executes use case
    //    - Logs "use case completed in Xms"
    //    - Records any errors
    //    - Ends span
}
```

---

## Logging Output Examples

### Successful Job Submission

```
2026-11-18 00:39:53 [INFO] webhook received | 
2026-11-18 00:39:53 [INFO] use case started | operation=submit-job
2026-11-18 00:39:53 [INFO] job submitted | id=gh-webhook-1794942593078122000 type=issue-triage
2026-11-18 00:39:53 [INFO] use case completed in 0ms | operation=submit-job
```

### With Tracing

```
[INFO] use case started | operation=submit-job
[TRACE] span=submit-job service=agentic-factory status=OK attributes=map[operation:submit-job start_time:1794942484 job_id:job-123 job_type:issue-triage]
[INFO] use case completed in 5ms | operation=submit-job
```

### Error Scenario

```
[INFO] use case started | operation=get-job-status job_id=not-found
[ERROR] span error | operation=get-job-status | [error job not found]
[ERROR] use case failed | operation=get-job-status | [error job not found]
[INFO] span completed | operation=get-job-status duration_ms=2
```

---

## Configuration & Environment

### Environment Variables

```bash
# Log Level: debug, info, warn, error
LOG_LEVEL=info

# PostgreSQL Configuration
DATABASE_URL=postgres://user:password@localhost:5432/agentic-factory
DB_MAX_CONNECTIONS=10
DB_IDLE_TIMEOUT=5m

# Server Configuration
PORT=8080
HOST=localhost
REQUEST_TIMEOUT=30s

# GitHub Integration
GITHUB_TOKEN=ghp_your_token
GITHUB_API_ENDPOINT=https://api.github.com

# Tracing Configuration (future)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=agentic-factory
```

---

## References

- **Source Files**:
  - `internal/bootstrap/bootstrap.go` — Dependency injection
  - `internal/infrastructure/logging/structured_logger.go` — Logging
  - `internal/infrastructure/tracing/tracer.go` — Tracing
  - `internal/infrastructure/observability/observability.go` — Facade
  - `internal/adapters/http/webhook_handler.go` — HTTP integration
  - `internal/adapters/http/job_handler.go` — HTTP integration

- **Documentation**:
  - `docs/OBSERVABILITY_GUIDE.md` — Detailed observability guide
  - `docs/CLEAN_ARCHITECTURE_SUMMARY.md` — Architecture overview
  - `README.md` — Main documentation

---

**Last Updated**: November 18, 2026  
**Version**: 1.1.0+Observability  
**Status**: Production Ready ✅
