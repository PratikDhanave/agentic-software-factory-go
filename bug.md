# Bug Report: State Machine Audit

**Generated:** 2026-11-18  
**Audit Scope:** State machine implementations, type safety, API consistency, DB persistence, test coverage  
**Agents:** 4 parallel QA auditors  
**Status:** Collecting findings...

---

## Summary

Comprehensive audit of state machine usage across the Factory Stack to identify:
- Type safety violations
- Missing state registrations
- API inconsistencies
- DB persistence issues
- Test coverage gaps
- Validation bypass opportunities

---

## Findings Summary

| Severity | Count | Status |
|----------|-------|--------|
| **CRITICAL** | 9 | Needs immediate fix |
| **HIGH** | 11 | Should fix before next release |
| **MEDIUM** | 8 | Nice to have |
| **LOW** | 1 | Polish |
| **Total** | **29** | **100% Identified** |

---

## Critical Issues (9 bugs)

1. **Missing state machine for AuditStatus** (Type Safety)
   - File: `audit.go` 
   - Impact: Audit entity uses raw strings without validation

2. **State machine bypass in Job.Retry()** (Type Safety)
   - File: `job.go` line 126
   - Impact: Retry logic can skip state validation using else clause

3. **Missing state machine for AgentExecution** (Type Safety)
   - File: `agent.go` line 118
   - Impact: Agent execution lifecycle has no type safety

4. **Hardcoded state strings vs domain mismatch** (API Consistency)
   - File: `handlers/factory.go` lines 163, 166, 169
   - Impact: API sends states that domain machines never produce

5. **Unchecked state transitions in 7-layer agent handler** (API Consistency)
   - File: `handlers/agents.go` lines 101, 118, 122, 130, 139, 147, 157, 167, 234
   - Impact: Errors silently ignored, invalid states sent to clients

6. **State mutation without synchronization** (API Consistency)
   - File: `handlers/agents.go` lines 98-109
   - Impact: Shared state modified across goroutines without protection

7. **SessionLifecycleMachine missing .WithStateDescriptions()** (Registration & Persistence)
   - File: `internal/domain/state_machines.go` lines 245-253
   - Impact: 9 states have no descriptions, breaking MCP discovery

8. **Raw string literals in learning_repo.go** (Registration & Persistence)
   - File: `learning_repo.go` lines 38, 280-285, 313-316
   - Impact: DB persistence uses hardcoded strings instead of constants

9. **Systematic unchecked error returns (17+ locations)** (Validation & Tests)
   - Files: `factory_pipeline.go` (10), `execute_workflow_usecase.go` (4), `dag_executor.go` (3)
   - Impact: Transition failures silently ignored, inconsistent states not detected

---

## High Priority Issues (11 bugs)

| # | Bug | File | Severity | Impact |
|---|-----|------|----------|--------|
| 11 | Raw string comparisons in Job entity | `job.go` | HIGH | Type-unsafe state handling |
| 12 | Raw string comparisons in Workflow entity | `workflow.go` | HIGH | Missing validation |
| 13 | Raw string comparisons for ApprovalStatus | Multiple | HIGH | Validation bypass |
| 14 | Arbitrary triage SSE status values | `handlers/triage.go` | HIGH | No domain alignment |
| 15 | Hardcoded job status strings missing constants | `handlers/integration.go` | HIGH | No single source of truth |
| 16 | Dashboard handler fabricates non-domain states | `handlers/dashboard.go` | HIGH | Invalid states to clients |
| 17 | FactoryRunStatusMachine uses raw strings | `factory_pipeline.go` lines 68-78 | HIGH | Inconsistent with other machines |
| 18 | ArtifactPromotionMachine uses raw strings | `factory_pipeline.go` lines 81-93 | HIGH | Inconsistent approach |
| 19 | DAGNode methods don't return errors | `workflow.go` lines 306-342 | HIGH | Callers can't detect failures |
| 20 | CoordinationSession methods silently ignore errors | `agent.go` lines 155-173 | HIGH | 10-agent coordination fails silently |
| 21 | Raw string comparison in FactoryPipeline.PromoteRun() | Multiple | HIGH | Type-unsafe logic |

---

## Medium Priority Issues (8 bugs)

| # | Bug | File | Type |
|---|-----|------|------|
| 22 | Missing state descriptions in WorkflowPipelineMachine | `workflow.go` | Missing metadata |
| 23 | Default ApprovalStatus set to raw string | Multiple | Type safety |
| 24 | Inconsistent JSON field names for status | `handlers/` | API schema |
| 25 | Raw string status assignment in web handler | `handlers/` | Type safety |
| 26 | SevenLayerMachine uses raw strings | `pkg/factory/states.go` lines 50-72 | Consistency |
| 27 | Transition calls use raw strings | `factory_pipeline.go` lines 419, 534, 575, 582, 596, 601, 762 | Type safety |
| 28 | WorkflowDefinition.Activate()/Archive() not tested | `entities/workflow.go` | Test coverage |
| 29 | WorkflowExecution.TransitionTo() not tested | `entities/workflow.go` | Test coverage |

---

## Low Priority Issues (1 bug)

| # | Bug | File | Type |
|---|-----|------|------|
| 30 | Empty state string handling not validated | Various | Edge case |

---

## Agents' Reports

### Agent 1: Type Safety Violations ✅
**Bugs Found:** 10  
**Task:** Find raw string comparisons, missing type constants, state handling inconsistencies

Identified 3 critical missing state machines and 7 type safety violations where raw strings bypass validation.

### Agent 2: API Consistency ✅
**Bugs Found:** 8  
**Task:** Find HTTP response inconsistencies, serialization issues

Found 3 critical issues where handlers send invalid states to clients, plus 5 high-priority API contract violations.

### Agent 3: Registration & Persistence ✅
**Bugs Found:** 6  
**Task:** Find unregistered state types, DB query issues, validation bypass

Identified 1 critical missing description + 3 high-priority raw string issues in DB persistence layer.

### Agent 4: Validation & Tests ✅
**Bugs Found:** 5+  
**Task:** Find missing tests, validation gaps, edge case coverage

Found 4+ critical systematic error-handling gaps (17+ unchecked Transition calls) + missing test files for 4 entity types.

---

## Remediation Plan

> [To be populated with priorities and fixes...]

---

## Notes

- State machine framework: `pkg/state/Machine` (unified approach)
- Central registry: `internal/stateregistry/registry.go`
- Domain machines: `internal/domain/state_machines.go`
- Entity machines: `internal/domain/entities/`
- Application machines: `internal/application/factory_pipeline.go`
- Factory machines: `pkg/factory/states.go`

---

## Related Documentation

- [Explicit State Machines](docs/EXPLICIT_STATE_MACHINES.md)
- [State Machine Migration Guide](docs/STATE_MACHINE_MIGRATION.md)
- [State Machine Summary](docs/STATE_MACHINES_SUMMARY.md)
