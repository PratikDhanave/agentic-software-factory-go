# DAG Workflow Engine - Full Implementation Plan

## Overview
This plan covers all 5 aspects: design review, examples, Phase 1 implementation, handler integration, and deep dives.

---

## Part 1: Design Review & Refinements

### Current Design Assumptions (verify these)
- ✅ **YAML Definitions**: Workflows are immutable templates, executed as instances
- ✅ **Topological Sort**: DAG is validated once during build, then executed level-by-level
- ✅ **Fail-Fast**: Any job failure stops downstream execution
- ✅ **Template Variables**: `{{ jobs.X.output }}` for data passing
- ✅ **Idempotent Jobs**: Each job can be retried safely

### Design Questions to Address

**1. Error Handling**
- Should failed jobs be retried automatically? (exponential backoff?)
- Do we continue other branches if one fails? (or stop entire workflow?)
- **Recommendation**: Fail-fast for now (Phase 4 adds retry logic)

**2. Conditional Logic**
- Should workflows support `if: condition` for jobs?
- Example: Only notify if analysis shows critical issues
- **Recommendation**: Phase 1 is deterministic; Phase 4 adds conditionals

**3. Long-Running Jobs**
- What if a job takes > 30 seconds?
- Do we support async/background jobs?
- **Recommendation**: Phase 1 is synchronous; Phase 4 adds background jobs

**4. Job Timeout**
- Should individual jobs have timeouts?
- What's the max workflow duration?
- **Recommendation**: Add configurable per-job timeout; global timeout = 5 minutes

**5. Workflow Versioning**
- Can workflows change after creation?
- Do we track workflow versions?
- **Recommendation**: Workflows are immutable; create new version for changes

**6. Parallel Limits**
- Should we limit concurrent jobs across all workflows?
- Example: Max 10 parallel jobs system-wide to prevent overload
- **Recommendation**: Phase 1 uses goroutine pool (default 4-8); configurable

---

## Part 2: Example YAML Workflows

### Example 1: GitHub Issue Triage Workflow
```yaml
name: "github-issue-triage"
version: "1.0"
description: "Comprehensive issue triage with analysis, classification, and assignment"

on:
  github_event:
    type: "issues"
    action: "opened"

jobs:
  extract_issue_data:
    task: "extract-github-issue"
    input:
      issue_number: "{{ trigger.issue.number }}"
      repo: "{{ trigger.repository.full_name }}"
  
  triage:
    task: "issue-triage"
    depends_on: ["extract_issue_data"]
    input:
      issue: "{{ jobs.extract_issue_data.output }}"
  
  analyze:
    task: "code-analysis"
    depends_on: ["extract_issue_data"]
    input:
      issue_body: "{{ jobs.extract_issue_data.output.body }}"
  
  classify:
    task: "label-classifier"
    depends_on: ["extract_issue_data"]
    input:
      issue_title: "{{ jobs.extract_issue_data.output.title }}"
      issue_body: "{{ jobs.extract_issue_data.output.body }}"
  
  decide_assignment:
    task: "auto-assign"
    depends_on: ["analyze", "classify", "triage"]
    input:
      analysis: "{{ jobs.analyze.output }}"
      labels: "{{ jobs.classify.output }}"
      severity: "{{ jobs.triage.output.severity }}"
  
  notify:
    task: "notify-team"
    depends_on: ["decide_assignment"]
    input:
      assigned_to: "{{ jobs.decide_assignment.output.assignee }}"
      issue_url: "{{ trigger.issue.html_url }}"
      severity: "{{ jobs.triage.output.severity }}"
```

**Execution Timeline**:
- t=0ms: `extract_issue_data` starts
- t=15ms: `extract_issue_data` completes → triggers [triage, analyze, classify] in parallel
- t=35ms: All 3 jobs complete → triggers `decide_assignment`
- t=50ms: `decide_assignment` completes → triggers `notify`
- t=65ms: `notify` completes → workflow done
- **Total: 65ms** (vs 110ms if sequential)

### Example 2: Pull Request Review Workflow
```yaml
name: "github-pr-review"
version: "1.0"
description: "Complete PR review with linting, security, testing, performance analysis"

on:
  github_event:
    type: "pull_request"
    action: "opened"

jobs:
  lint_check:
    task: "code-lint"
    input:
      files: "{{ trigger.pull_request.files }}"
  
  security_scan:
    task: "security-analysis"
    input:
      files: "{{ trigger.pull_request.files }}"
  
  run_tests:
    task: "run-tests"
    input:
      branch: "{{ trigger.pull_request.head.ref }}"
  
  performance_test:
    task: "perf-analysis"
    depends_on: ["run_tests"]
    input:
      test_results: "{{ jobs.run_tests.output }}"
  
  coverage_check:
    task: "coverage-analysis"
    depends_on: ["run_tests"]
    input:
      test_results: "{{ jobs.run_tests.output }}"
  
  summarize_review:
    task: "pr-summary"
    depends_on: ["lint_check", "security_scan", "performance_test", "coverage_check"]
    input:
      lint_issues: "{{ jobs.lint_check.output }}"
      security_issues: "{{ jobs.security_scan.output }}"
      perf_impact: "{{ jobs.performance_test.output }}"
      coverage_delta: "{{ jobs.coverage_check.output }}"
  
  post_comment:
    task: "post-pr-comment"
    depends_on: ["summarize_review"]
    input:
      summary: "{{ jobs.summarize_review.output }}"
      pr_number: "{{ trigger.pull_request.number }}"
```

**Execution Timeline**:
- t=0ms: [lint_check, security_scan, run_tests] start in parallel
- t=20ms: lint_check completes
- t=25ms: security_scan completes
- t=50ms: run_tests completes → triggers [performance_test, coverage_check]
- t=65ms: performance_test + coverage_check complete → triggers summarize_review
- t=80ms: summarize_review completes → triggers post_comment
- t=95ms: post_comment completes → workflow done
- **Total: 95ms** (vs 230ms if sequential)

### Example 3: Bug Reproduction & Fix Workflow
```yaml
name: "bug-reproduce-and-fix"
version: "1.0"
description: "Reproduce bug, determine root cause, generate fix, test fix"

on:
  github_event:
    type: "issues"
    action: "opened"
    labels:
      - "bug"

jobs:
  parse_bug_report:
    task: "parse-bug-report"
    input:
      issue_number: "{{ trigger.issue.number }}"
      body: "{{ trigger.issue.body }}"
  
  reproduce_bug:
    task: "reproduce-bug"
    depends_on: ["parse_bug_report"]
    input:
      steps: "{{ jobs.parse_bug_report.output.reproduction_steps }}"
      environment: "{{ jobs.parse_bug_report.output.environment }}"
  
  analyze_root_cause:
    task: "analyze-root-cause"
    depends_on: ["reproduce_bug"]
    input:
      error_logs: "{{ jobs.reproduce_bug.output.logs }}"
      stacktrace: "{{ jobs.reproduce_bug.output.stacktrace }}"
  
  generate_fix:
    task: "generate-fix"
    depends_on: ["analyze_root_cause"]
    input:
      root_cause: "{{ jobs.analyze_root_cause.output.cause }}"
      affected_files: "{{ jobs.analyze_root_cause.output.files }}"
  
  verify_fix:
    task: "verify-fix"
    depends_on: ["generate_fix"]
    input:
      fix_patch: "{{ jobs.generate_fix.output.patch }}"
      test_cases: "{{ jobs.parse_bug_report.output.test_cases }}"
  
  create_pr:
    task: "create-pull-request"
    depends_on: ["verify_fix"]
    input:
      fix_code: "{{ jobs.generate_fix.output.code }}"
      issue_number: "{{ trigger.issue.number }}"
      description: "Fixes #{{ trigger.issue.number }}: {{ jobs.analyze_root_cause.output.summary }}"
```

---

## Part 3: Phase 1 Implementation - Core Data Structures

### Files to Create

1. **`internal/domain/entities/workflow.go`** (200 lines)
   - `Workflow` entity (definition template)
   - `WorkflowDefinition` struct (from YAML)
   - `Job` struct (single job in workflow)
   - Validation methods

2. **`internal/domain/entities/workflow_execution.go`** (250 lines)
   - `WorkflowExecution` entity (instance)
   - `DAGNode` (single job in DAG)
   - `WorkflowProgress` tracking
   - Status enums and transitions

3. **`internal/domain/services/dag_builder.go`** (350 lines)
   - `DAGBuilder` interface and implementation
   - `Build(def WorkflowDefinition) → DirectedAcyclicGraph`
   - Cycle detection (DFS)
   - Topological sorting
   - Level assignment

4. **`internal/domain/services/directed_acyclic_graph.go`** (150 lines)
   - `DirectedAcyclicGraph` struct
   - Graph operations (add node, add edge)
   - Query operations (get dependencies, get dependents)
   - Validation methods

5. **`internal/domain/repositories/workflow_repository.go`** (100 lines)
   - Interface for CRUD operations
   - `SaveWorkflow`, `GetWorkflow`, `ListWorkflows`
   - `SaveExecution`, `GetExecution`, `UpdateExecutionStatus`

6. **`internal/infrastructure/persistence/workflow_store.go`** (400 lines)
   - PostgreSQL implementation
   - SQL queries for workflows and executions
   - JSON serialization of DAG and results

### Implementation Order
1. Start with entities (Workflow, WorkflowExecution, DAGNode)
2. Add DAGBuilder with cycle detection
3. Add DirectedAcyclicGraph data structure
4. Add repository interfaces
5. Add PostgreSQL persistence
6. Create tests for each component

---

## Part 4: Handler Integration

### Changes to WebhookHandler

**Current**:
```go
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    // Parse event
    // Create single job
    // Execute SubmitJobUseCase
    // Return 202
}
```

**New (DAG Support)**:
```go
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    // Parse event
    
    // Check if workflow exists for this event type
    workflow, _ := h.workflowRepo.FindByTrigger(eventType)
    
    if workflow != nil {
        // Execute workflow DAG
        execution, _ := h.workflowExecutor.Execute(ctx, workflow, event)
        return h.respondWithWorkflowExecution(w, execution)
    } else {
        // Fallback to single-job execution (backward compatibility)
        job := h.createJob(event)
        h.submitJobUC.Execute(ctx, job)
        return h.respondWithJob(w, job)
    }
}
```

### New Use Case: ExecuteWorkflowUseCase

```go
type ExecuteWorkflowUseCase struct {
    workflowRepo      WorkflowRepository
    dagExecutor       DAGExecutor
    logger           Logger
}

func (u *ExecuteWorkflowUseCase) Execute(ctx context.Context, workflow *Workflow, event interface{}) (*WorkflowExecution, error) {
    // 1. Build DAG from workflow definition
    // 2. Create WorkflowExecution instance
    // 3. Execute DAG (level by level)
    // 4. Collect results
    // 5. Update execution status
    // 6. Return execution with results
}
```

### Database Migrations

Create migration file `migrations/004_add_workflows_tables.sql`:
```sql
CREATE TABLE workflows (
    id UUID PRIMARY KEY,
    name VARCHAR NOT NULL,
    version VARCHAR NOT NULL,
    definition JSONB NOT NULL,
    triggers JSONB NOT NULL,
    status VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    trigger_event JSONB NOT NULL,
    dag JSONB NOT NULL,
    status VARCHAR NOT NULL,
    progress JSONB NOT NULL,
    results JSONB,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE job_executions (
    id UUID PRIMARY KEY,
    workflow_execution_id UUID NOT NULL REFERENCES workflow_executions(id),
    job_id VARCHAR NOT NULL,
    task_name VARCHAR NOT NULL,
    input JSONB NOT NULL,
    status VARCHAR NOT NULL,
    result JSONB,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_workflow_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_job_executions_workflow_execution_id ON job_executions(workflow_execution_id);
```

---

## Part 5: Deep Dive Topics

### 5.1 Cycle Detection Strategy
**Problem**: DAG must have no cycles
**Solution**: DFS with 3-color marking (white/gray/black)
- White: unvisited
- Gray: currently visiting
- Black: fully visited
**If we find edge to gray node → cycle detected**

### 5.2 Topological Sort Algorithm
**Problem**: Order jobs so dependencies are satisfied
**Solution**: Kahn's algorithm (BFS from nodes with 0 in-degree)
1. Build in-degree map
2. Queue all nodes with in-degree 0
3. For each queued node:
   - Add to result
   - Decrement in-degree of neighbors
   - If neighbor now has 0 in-degree, queue it
**Result**: Jobs grouped into execution levels**

### 5.3 Input Variable Resolution
**Problem**: `{{ jobs.triage.output }}` must resolve to actual result
**Solution**: Template engine with context
1. Parse template: `jobs.X.output` → jobID, field
2. Lookup job result in `results[jobID]`
3. Extract field if present
4. Replace in input map

### 5.4 Error Propagation Strategy
**Problem**: Job A fails → Jobs B,C (depending on A) should fail fast
**Solution**: Check dependency results before executing job
```go
for _, dep := range node.DependsOn {
    if depErr, exists := errors[dep]; exists && depErr != nil {
        errors[id] = fmt.Errorf("dependency %s failed: %w", dep, depErr)
        return  // Don't execute this job
    }
}
```

### 5.5 Observability Integration
**Problem**: Log individual job execution within workflow context
**Solution**: Enhanced context with workflow metadata
```go
ctx = context.WithValue(ctx, "workflow_id", execution.ID)
ctx = context.WithValue(ctx, "job_id", jobID)
ctx = context.WithValue(ctx, "execution_level", level)
// Logger picks these up automatically
```

### 5.6 Resource Management
**Problem**: Unlimited goroutines for large workflows
**Solution**: Goroutine pool with semaphore
```go
semaphore := make(chan struct{}, maxParallel)
for _, jobID := range level {
    semaphore <- struct{}{}  // Acquire
    go func(id string) {
        defer func() { <-semaphore }()  // Release
        // Execute job
    }(jobID)
}
```

---

## Implementation Timeline

### Week 1: Foundation
- [ ] Create workflow entities (Workflow, WorkflowExecution, DAGNode)
- [ ] Implement DAGBuilder with cycle detection
- [ ] Add DirectedAcyclicGraph data structure
- [ ] Write comprehensive tests

### Week 2: Persistence
- [ ] Create workflow repository interface
- [ ] Implement PostgreSQL persistence
- [ ] Create database migrations
- [ ] Add workflow query operations

### Week 3: Execution
- [ ] Implement DAGExecutor
- [ ] Add input variable resolution
- [ ] Create ExecuteWorkflowUseCase
- [ ] Wire into WebhookHandler

### Week 4: Integration & Polish
- [ ] Add observability (logging + tracing)
- [ ] Create example workflow YAML files
- [ ] Update documentation with examples
- [ ] Performance testing and optimization

---

## Success Criteria

✅ Workflows defined in YAML and persisted
✅ DAG built with cycle detection and topological sort
✅ Jobs execute in correct order with proper parallelism
✅ Dependencies automatically resolved
✅ All logs include workflow_id and job_id context
✅ Traces show per-job timing
✅ Example workflows (Triage, PR Review, Bug Fix) work end-to-end
✅ Error cases handled (dependency failure, timeout, etc.)
✅ Backward compatible with single-job execution
✅ Performance: 50-70% faster than sequential execution

---

## Rollback Plan

If Phase 1 implementation encounters issues:
1. All workflow tables in separate schema (can be dropped)
2. Single-job execution path remains untouched
3. Feature flag can disable DAG executor (use SubmitJobUseCase instead)
4. No changes to existing API contracts
