# Diagram Index & Reference Guide

Complete guide to all architecture, sequence, and flow diagrams in the documentation.

## Document Overview

| Document | Purpose | Focus |
|----------|---------|-------|
| [README_DETAILED.md](../README_DETAILED.md) | Comprehensive architecture guide | System design, data flows, sequences |
| [API_FLOWS.md](./API_FLOWS.md) | API operation details | Request flows, examples, performance |
| [OBSERVABILITY_GUIDE.md](./OBSERVABILITY_GUIDE.md) | Logging & tracing guide | Implementation, usage, configuration |
| [CLEAN_ARCHITECTURE_SUMMARY.md](./CLEAN_ARCHITECTURE_SUMMARY.md) | Architecture overview | Layers, phases, deployment |

---

## System Architecture Diagrams

### 1. High-Level System Design
**Location**: `README_DETAILED.md` → System Architecture

Shows the interaction between:
- Web UI (Next.js)
- HTTP Server (Go)
- Handlers (WebhookHandler, JobHandler)
- Use Cases (SubmitJob, GetJobStatus, IssueTriage)
- Repositories (Job, Workflow)
- Database (PostgreSQL)
- Event Bus
- Observability (Logging & Tracing)

**Use this to**: Understand how all components connect

---

### 2. Domain-Driven Design Structure
**Location**: `README_DETAILED.md` → System Architecture

Shows the four-layer architecture:
- Domain Layer (Business Rules)
- Application Layer (Use Cases)
- Adapters Layer (HTTP, Database)
- Infrastructure (DI, Logging)

**Use this to**: Understand separation of concerns

---

### 3. Clean Architecture - Onion Model
**Location**: `README_DETAILED.md` → Clean Architecture Overview

Shows concentric layers:
- Outer: Adapters (HTTP, Database, GitHub)
- Middle: Application (Use Cases)
- Core: Domain (Entities, Interfaces)
- Center: Infrastructure (DI, Cross-cutting)

**Use this to**: See dependency flow and isolation

---

### 4. Component Interaction Diagram
**Location**: `README_DETAILED.md` → Component Interactions

Shows how components depend on each other:
- WebhookHandler → SubmitJobUseCase → Repository → Database
- Logging interfaces → Implementations
- Event publishing flow

**Use this to**: Trace a request through the system

---

## Data Flow Diagrams

### 5. GitHub Webhook Processing Flow
**Location**: `README_DETAILED.md` → Data Flow Diagrams

Shows the complete flow:
1. GitHub sends webhook
2. Handler parses payload
3. Job created with observability
4. UseCase executed (observed)
5. Repository saves to database
6. Events published
7. Response sent back

**Use this to**: Understand webhook processing end-to-end

---

### 6. Job Status Query Flow
**Location**: `README_DETAILED.md` → Data Flow Diagrams

Shows the query flow:
1. User requests job status
2. Handler receives request
3. UseCase executes with observability
4. Repository queries database
5. Result returned and logged

**Use this to**: Understand how data retrieval works

---

### 7. Complete Job Lifecycle
**Location**: `README_DETAILED.md` → Data Flow Diagrams

State machine showing:
- Created (initial)
- Queued (persisted)
- Processing (executing)
- Completed (finished)
- Failed (error state)

**Use this to**: Understand job states and transitions

---

## Message Sequence Diagrams

### 8. End-to-End GitHub Issue → Job Processing
**Location**: `README_DETAILED.md` → Message Sequence Diagrams

Detailed sequence showing:
- GitHub webhook arrival
- Request processing
- Handler parsing
- Observability instrumentation
- Job submission use case
- Database persistence
- Response generation
- Logging at each step

**Use this to**: Trace execution with timing information

---

### 9. Observability Tracing Flow
**Location**: `README_DETAILED.md` → Message Sequence Diagrams

Shows how tracing works:
1. UseCaseObserver wraps execution
2. Tracer.Start() creates span
3. SpanContext created
4. Pre-execution logging
5. Business logic (can set attributes)
6. Error recording if needed
7. Post-execution logging
8. Span.End() outputs trace

**Use this to**: Understand observability instrumentation

---

### 10. Error Handling Flow
**Location**: `README_DETAILED.md` → Message Sequence Diagrams

Shows error propagation:
- Error occurs in UseCase
- Recorded in span
- Logged with context
- Returned to handler
- HTTP error response sent

**Use this to**: Understand error tracking and logging

---

## Component Interaction Diagrams

### 11. Observer Pattern Implementation
**Location**: `README_DETAILED.md` → Component Interactions

Shows UseCaseObserver structure:
- Wraps SpanContext
- Contains Logger
- Contains Span
- Contains Context
- Methods for attributes, errors, events

**Use this to**: Understand the observer implementation

---

### 12. Context Propagation
**Location**: `README_DETAILED.md` → Observability Architecture

Shows how context flows:
1. HTTP Request creates context.Background()
2. Passed through handlers
3. ObserveExecution creates context.WithValue()
4. Passed to UseCase
5. Used by Repository
6. trace_id and span_id extracted for logging

**Use this to**: Understand context flow and trace ID injection

---

### 13. Instrumentation Points
**Location**: `README_DETAILED.md` → Observability Architecture

Shows where observability is integrated:
- HTTP Handler Layer: StartSpan + Logger
- Use Case Layer: Pre/post logging, attributes
- Repository Layer: Operation logging, error handling

**Use this to**: See where to add observability

---

## Deployment Architecture Diagrams

### 14. Development Setup
**Location**: `README_DETAILED.md` → Deployment Architecture

Shows local development environment:
- Source code → build → binary
- In-memory repositories
- Console logger (DEBUG level)
- Simple tracer (stdout)

**Use this to**: Understand development flow

---

### 15. Production Setup
**Location**: `README_DETAILED.md` → Deployment Architecture

Shows production environment:
- Load balancer (nginx/HAProxy)
- Multiple servers
- PostgreSQL database
- Redis cache (optional)
- ELK logging service
- Jaeger tracing service

**Use this to**: Plan production deployment

---

## Database Schema Diagrams

### 16. Jobs Table Schema
**Location**: `README_DETAILED.md` → Database Schema

Shows job table structure:
- id (UUID, PK)
- type, status, priority
- input (JSONB)
- result (JSONB)
- timestamps
- Indexes for performance

**Use this to**: Understand data model

---

### 17. Workflows Table Schema
**Location**: `README_DETAILED.md` → Database Schema

Shows workflow table structure:
- id (UUID, PK)
- name, status
- job_ids (UUID ARRAY)
- metadata (JSONB)
- timestamps

**Use this to**: See workflow storage

---

### 18. Table Relationships
**Location**: `README_DETAILED.md` → Database Schema

Shows relationships:
- 1 workflow → many jobs
- 1 job → many audit_events

**Use this to**: Understand data relationships

---

## API Flow Diagrams

### 19. GitHub Webhook to Job Creation - Full Flow
**Location**: `docs/API_FLOWS.md` → Webhook Processing

Comprehensive sequence showing:
- GitHub webhook
- Request parsing
- Payload processing
- Observability instrumentation
- Business logic
- Completion
- Response formatting

**Color coded sections:**
- Blue: Request Parsing
- Green: Payload Processing
- Orange: Observability
- Purple: Business Logic
- Red: Completion

**Use this to**: Understand webhook processing with timestamps

---

### 20. Status Query with Caching
**Location**: `docs/API_FLOWS.md` → Job Status Query

Shows query flow with cache:
- Request handling
- Observability setup
- Cache hit/miss paths
- Database query
- Response formatting

**Use this to**: Understand query optimization

---

### 21. Database Error Handling
**Location**: `docs/API_FLOWS.md` → Error Scenarios

Shows error flows:
- Connection errors
- Not found errors
- Query errors
- Span error recording
- Error logging
- Error response

**Use this to**: Understand error handling

---

### 22. Validation Error Flow
**Location**: `docs/API_FLOWS.md` → Error Scenarios

Shows validation:
- Input validation
- Error recording
- Error logging
- Bad request response

**Use this to**: Understand validation flow

---

## Concurrency & Performance Diagrams

### 23. Multiple Webhooks Processing
**Location**: `docs/API_FLOWS.md` → Concurrent Requests

Shows:
- Multiple concurrent requests
- Goroutine pool
- Database connection pool
- Thread-safe logging and tracing

**Use this to**: Understand concurrent request handling

---

### 24. End-to-End Latency Distribution
**Location**: `docs/API_FLOWS.md` → Performance Metrics

Shows latency breakdown:
- HTTP processing: 1-2ms
- Handler logic: 2-5ms
- Database query: 20-50ms (dominant)
- Logging: 1-2ms
- Tracing: 1-2ms
- **Total: 25-61ms**

**Use this to**: Understand performance characteristics

---

### 25. Operation Benchmarks
**Location**: `docs/API_FLOWS.md` → Performance Metrics

Shows performance for different operations:

| Operation | Avg Latency | P99 | Throughput |
|-----------|------------|-----|-----------|
| Webhook | 50-100ms | 150ms | 100-200/s |
| Job Query | 20-30ms | 50ms | 500-1000/s |
| Issue Triage | 500-1000ms | 1500ms | 50-100/s |

**Use this to**: Set performance expectations

---

## State & Workflow Diagrams

### 26. Job Status State Machine
**Location**: `docs/API_FLOWS.md` → State Transitions

Shows all job states:
- CREATED → QUEUED → PROCESSING → COMPLETED/FAILED
- Retry path: FAILED → QUEUED
- Notes for each state

**Use this to**: Understand job lifecycle

---

### 27. Complete Issue Triage Workflow
**Location**: `docs/API_FLOWS.md` → Example Workflow

Shows full workflow:
1. Developer creates GitHub issue
2. Webhook received
3. Job created and saved
4. Async processing starts
5. Issue analyzed
6. Result stored
7. Status queryable

**Use this to**: See end-to-end scenario

---

## Observability Diagrams

### 28. Logging & Tracing System Architecture
**Location**: `README_DETAILED.md` → Observability Architecture

Shows:
- Application → StructuredLogger
- Application → TraceProvider
- Logger implementations (stdout)
- Trace implementations (NoOp, Simple, OTLP future)
- Output formats

**Use this to**: Understand observability stack

---

### 29. Context Propagation for Tracing
**Location**: `README_DETAILED.md` → Observability Architecture

Shows:
- Context.Background() creation
- trace_id and span_id injection
- Propagation through layers
- Extraction in logger

**Use this to**: Understand trace context flow

---

### 30. Instrumentation Points Map
**Location**: `README_DETAILED.md` → Observability Architecture

Shows where observability is added:
- HTTP Handler: StartSpan + Logger
- Use Case: Pre/post logging + attributes
- Repository: Operation logging + error handling
- Result: Automatic tracing and logging

**Use this to**: Know where observability is active

---

## How to Read Diagrams

### Mermaid Sequence Diagrams
```
Actor → Participant1 → Participant2
         (message →)
         (← response)
```

**Key elements:**
- Actors (left side)
- Participants (top)
- Arrows show message flow
- Time flows downward
- `rect` boxes show logical sections
- `alt` for conditional logic

---

### Mermaid State Diagrams
```
[*] → State1 → State2 → [*]
       ↓        ↑
       └────────┘
```

**Key elements:**
- [*] = start/end
- States (rectangles)
- Transitions (arrows)
- Loops for retries

---

### Mermaid Graph/Flowchart Diagrams
```
Node1 → Node2
Node2 → Node3A
Node2 → Node3B
```

**Key elements:**
- Nodes (boxes/shapes)
- Edges (connections)
- Hierarchy (top-down)
- Branching (decisions)

---

## Quick Reference

### Finding Specific Information

**Q: How does a webhook get processed?**
A: See diagram #19 (Webhook Processing Flow) or #8 (End-to-End Sequence)

**Q: What happens when a database error occurs?**
A: See diagram #21 (Database Error Handling)

**Q: How is tracing instrumented?**
A: See diagram #9 (Observability Tracing Flow) or #30 (Instrumentation Points)

**Q: What's the performance breakdown?**
A: See diagram #24 (Latency Distribution) or #25 (Benchmarks)

**Q: How does a job move through states?**
A: See diagram #26 (State Machine)

**Q: What's the architecture?**
A: See diagrams #1-4 (System Architecture)

**Q: How do concurrent requests work?**
A: See diagram #23 (Concurrent Requests)

**Q: What's the database schema?**
A: See diagrams #16-18 (Database Schema)

---

## Diagram Dependencies

```
System Architecture (#1-4)
  └─→ Data Flows (#5-7)
        └─→ Message Sequences (#8-10)
              └─→ API Flows (#19-22)
                    └─→ Performance (#24-25)

Observability (#28-30)
  └─→ Tracing Flow (#9)
  └─→ Instrumentation (#30)

Database (#16-18)
  └─→ Schema understanding
  └─→ Query flows (#20)

States (#26)
  └─→ Lifecycle understanding
  └─→ Error paths (#21-22)

Production (#15)
  └─→ Deployment planning
  └─→ Scaling (#23)
```

---

## Color Coding Convention

| Color | Meaning | Used in |
|-------|---------|---------|
| Blue | Parsing/Setup | #19, #8 |
| Green | Processing/Logic | #19 |
| Orange | Observability | #19, #9 |
| Purple | Business Logic | #19 |
| Red | Completion/Errors | #19, #21 |
| Light Gray | Optional/Future | Deployment |

---

## Related Documentation

- **Code**: See `internal/` directory for implementations
- **API Examples**: See `docs/API_FLOWS.md`
- **Configuration**: See `docs/OBSERVABILITY_GUIDE.md`
- **Architecture**: See `docs/CLEAN_ARCHITECTURE_SUMMARY.md`
- **Main Guide**: See `README_DETAILED.md`

---

**Last Updated**: November 18, 2026  
**Total Diagrams**: 30  
**Diagram Types**: Sequence, State Machine, Flowchart, Graph  
**Coverage**: 100% of main flows
