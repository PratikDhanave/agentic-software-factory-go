# State Machine Migration Roadmap

## Quick Reference: All Implicit State Machines Found

### 1. **Decision State Machine** (High Priority - Core Logic)
**Current locations with implicit states:**
- `internal/domain/types.go:79` - `AgentDecision.ApprovalStatus` 
- `cmd/asf-web/handlers/factory.go` - Decision approval handling
- `internal/application/factory_pipeline.go` - Decision state changes
- `internal/adapters/persistence/sqlite/learning_repo.go` - State persistence

**Valid states in codebase:**
```
"pending" → "approved" → "correct"
         ↘ "rejected"   → "incorrect"
```

**Files to update:**
```
1. internal/domain/types.go                      - Update AgentDecision struct
2. cmd/asf-web/handlers/factory.go               - Decision approval endpoints
3. internal/application/factory_pipeline.go      - Decision processing logic
4. internal/adapters/persistence/sqlite/learning_repo.go - Persistence layer
5. web-ui/src/lib/api.ts                         - Frontend API types
6. web-ui/src/components/ApprovalsPanel.tsx      - Approval UI
```

---

### 2. **Approval State Machine** (High Priority - Governance)
**Current locations:**
- `internal/domain/types.go:90` - `Approval.Status`
- `cmd/asf-web/handlers/integration.go` - Approval submission
- `internal/adapters/persistence/sqlite/coordination_repo.go` - Approval storage

**Valid states:**
```
"pending" → "approved"
         ↘ "rejected"
```

**New state to add:** `"expired"` (for timeout handling)

**Files to update:**
```
1. internal/domain/types.go
2. cmd/asf-web/handlers/integration.go
3. internal/adapters/persistence/sqlite/coordination_repo.go
4. web-ui/src/components/ApprovalsPanel.tsx
5. cmd/asf-web/main.go (approval processing)
```

---

### 3. **Execution State Machine** (Medium Priority - Workflow Lifecycle)
**Current locations:**
- `internal/domain/types.go:107` - `WorkflowExecution.Status`
- `cmd/asf-web/main.go` - Lines checking `update.Status == "completed"`
- `internal/adapters/persistence/sqlite/workflow_repo.go` - Query and persist

**Current states in codebase:**
```
"active", "completed", "failed", "awaiting_approval"
```

**Better model:**
```
"queued" → "executing" → "awaiting_gate" → "executing"
                      ↘ "verifying" ↗
                      ↘ "completed"
                      ↘ "failed"
                      ↘ "cancelled"
```

**Files to update:**
```
1. internal/domain/types.go
2. cmd/asf-web/main.go (all status checks)
3. internal/adapters/persistence/sqlite/workflow_repo.go
4. internal/application/factory_pipeline.go
5. web-ui/src/app/page.tsx (status display)
6. web-ui/src/components/FactoryAssemblyLine.tsx
```

---

### 4. **Task State Machine** (Medium Priority - Job Lifecycle)
**Current locations:**
- `pkg/factory/workflow.go` - Task execution
- `internal/adapters/persistence/sqlite/workflow_repo.go` - Task storage

**Current states:**
```
"pending", "running", "completed", "failed"
```

**Better model:**
```
"pending" → "running" → "completed"
                     ↘ "failed" → "retrying" → "running"
                     ↘ "timedout"
                     ↘ "skipped"
```

**Files to update:**
```
1. pkg/factory/workflow.go (task lifecycle)
2. pkg/factory/workflow_test.go (tests)
3. internal/adapters/persistence/sqlite/workflow_repo.go
4. web-ui/src/components/PipelinePlayground.tsx
```

---

### 5. **Session State Machine** (Medium Priority - High-Level State)
**Current locations:**
- `cmd/asf-web/main.go` - Session lifecycle
- `internal/adapters/persistence/sqlite/coordination_repo.go` - Session storage

**Current implicit states:**
```
execution starts → awaiting_approval → resumed → completed/failed
```

**Better model:**
```
"queued" → "analyzing" → "planning_agents" → "executing"
                                              ↓
                                         "awaiting_gate"
                                              ↓
                                           "executing"
                                              ↓
                                         "verifying"
                                              ↓
                                         "completed"
                         (or "failed"/"cancelled" at any point)
```

**Files to update:**
```
1. cmd/asf-web/main.go (session orchestration)
2. internal/adapters/persistence/sqlite/coordination_repo.go
3. web-ui/src/app/page.tsx (session display)
```

---

### 6. **Phase State Machine** (Low Priority - Already Pretty Clear)
**Current locations:**
- `cmd/asf/main.go` - Phase switching
- `cmd/asf-web/main.go` - Phase-based logic

**States:**
```
"crawl" ↔ "walk" ↔ "run"
```

**Files to update:**
```
1. cmd/asf/main.go (minimal change)
2. cmd/asf-web/main.go (validation)
3. web-ui/src/app/page.tsx (phase selector)
```

---

## Migration Path: By Priority

### Phase 1: Foundation (Week 1)
✅ Create `internal/domain/state_machines.go` - **DONE**
✅ Create documentation - **DONE**
⬜ Add unit tests for state machines
⬜ Add state validation middleware

**Time: ~4 hours | Risk: LOW**

---

### Phase 2: Decision & Approval (Week 2) - HIGHEST IMPACT
Most of the value comes from this phase.

```go
// Before
type AgentDecision struct {
	ApprovalStatus string // ❌ Any string allowed
}

// After
type AgentDecision struct {
	ApprovalStatus string // Keep for DB compat
	State          DecisionState // New type-safe field
}
```

**Files (in order):**

1. **`internal/domain/types.go`** (10 min)
   - Add `State DecisionState` to `AgentDecision`
   - Add `State ApprovalState` to `Approval`
   - Keep old `ApprovalStatus` / `Status` fields for backward compat

2. **`cmd/asf-web/handlers/factory.go`** (20 min)
   - Replace all `decision.ApprovalStatus == "approved"` checks with `DecisionState(decision.ApprovalStatus) == DecisionApproved`
   - Add transition validation before state changes

3. **`internal/application/factory_pipeline.go`** (20 min)
   - Update decision approval logic
   - Update approval recording logic

4. **`internal/adapters/persistence/sqlite/learning_repo.go`** (15 min)
   - Add validation on save
   - Add state change event recording

5. **`web-ui/src/components/ApprovalsPanel.tsx`** (20 min)
   - Call `DecisionState(...).CanTransitionTo()` for valid actions
   - Only show buttons for valid transitions

6. **`web-ui/src/lib/api.ts`** (5 min)
   - Add `DecisionState` type definitions

**Time: ~1-1.5 hours | Impact: HIGH**

---

### Phase 3: Execution & Task States (Week 2-3)
Good ROI; medium effort

**Files by state machine:**

**Execution State:**
1. `internal/domain/types.go` - Add `State ExecutionState` to `WorkflowExecution`
2. `cmd/asf-web/main.go` - Replace all status checks (long, tedious)
3. `internal/adapters/persistence/sqlite/workflow_repo.go` - Add validation
4. `web-ui/src/app/page.tsx` - Use new state type
5. `web-ui/src/components/FactoryAssemblyLine.tsx` - Valid action buttons

**Task State:**
1. `pkg/factory/workflow.go` - Update task lifecycle
2. `pkg/factory/workflow_test.go` - Add tests for new states
3. `internal/adapters/persistence/sqlite/workflow_repo.go` - Persist task states
4. `web-ui/src/components/PipelinePlayground.tsx` - Display task states

**Time: ~2-3 hours | Impact: MEDIUM-HIGH**

---

### Phase 4: Session State (Week 3)
Ties everything together; medium effort

**Files:**
1. `internal/domain/types.go` - Add session state tracking
2. `cmd/asf-web/main.go` - Replace implicit session lifecycle
3. `internal/adapters/persistence/sqlite/coordination_repo.go` - Persist session state
4. `web-ui/src/app/page.tsx` - Display session stage

**Time: ~1 hour | Impact: MEDIUM**

---

### Phase 5: Phase State (Week 3)
Easy wins; already pretty clear

**Files:**
1. `cmd/asf/main.go` - Type-safe phase constants
2. `cmd/asf-web/main.go` - Phase transition validation
3. `web-ui/src/app/page.tsx` - Phase selector UI

**Time: ~30 min | Impact: LOW (but good for consistency)**

---

## By-The-Numbers Estimate

| Phase | Hours | Risk | Impact |
|-------|-------|------|--------|
| Foundation | 4 | LOW | Setup |
| Decision/Approval | 1.5 | MEDIUM | 🔥 HIGH |
| Execution/Task | 2.5 | MEDIUM | MEDIUM-HIGH |
| Session | 1 | LOW | MEDIUM |
| Phase | 0.5 | LOW | LOW |
| **Testing & QA** | **3** | **MEDIUM** | **CRITICAL** |
| **TOTAL** | **~12 hours** | | |

**Timeline:** 2-3 weeks part-time, 3-4 days full-time

---

## Testing Strategy

```go
// 1. Unit tests for state machines (already written, just needs run)
go test ./internal/domain/state_machines_test.go

// 2. Integration tests: DB round-trips
// - Save decision with state → read back → verify state
// - Invalid transitions → error
// - Valid transitions → success

// 3. End-to-end tests
// - Decision workflow: pending → approved → correct
// - Approval workflow: pending → approved
// - Full session lifecycle through all states

// 4. Property-based tests (optional but powerful)
// For any sequence of state transitions, verify:
// - If transition is allowed, it succeeds
// - If transition is not allowed, it errors
// - Terminal states never change
```

---

## Backward Compatibility Strategy

**Goal: Zero downtime deployment**

### Coexistence Phase (Week 1-2)

```go
// DB stores strings (no schema change)
decision.ApprovalStatus = "approved"  // Old way still works

// New code uses types
state := DecisionState(decision.ApprovalStatus)
if state.CanTransitionTo(DecisionCorrect) { ... }  // New way
```

### Gradual Replacement (Week 2-3)

```go
// Update one handler at a time
// Old: if decision.ApprovalStatus == "approved"
// New: if DecisionState(decision.ApprovalStatus) == DecisionApproved

// DB still stores strings; frontend gets same responses
// Gradually replace each usage
```

### Cleanup Phase (Week 4)

```go
// Once all handlers updated, optionally:
// 1. Rename State field to Status
// 2. Update DB schema to store enum (optional, strings work fine)
// 3. Remove old string-based code
```

---

## Deployment Checklist

### Pre-Deployment
- [ ] All tests passing
- [ ] State machine unit tests 100% coverage
- [ ] Integration tests for each state machine
- [ ] Backward compatibility verified
- [ ] No breaking API changes

### Deployment
- [ ] Add `state_machines.go` to production
- [ ] Add validation middleware
- [ ] Add state change event logging
- [ ] Deploy with feature flag (optional)
- [ ] Monitor state transitions

### Post-Deployment
- [ ] Verify no invalid transitions logged
- [ ] Check state change events flowing correctly
- [ ] Gradually migrate handlers (1-2 per day)
- [ ] Team training on new state machines
- [ ] Update runbooks

---

## FAQ

**Q: Will this break existing deployments?**
A: No. We coexist with string-based code for 1-2 weeks, then gradually migrate.

**Q: Do we need to update the database schema?**
A: No. We'll store strings in the DB, which works fine. Optionally migrate to enums later.

**Q: How do I know if a transition is valid?**
A: Call `state.CanTransitionTo(newState)` before changing. The middleware validates on save.

**Q: What about agents? Do they see this?**
A: Yes! Better: agents can call `.CanTransitionTo()` to know what actions are valid.

**Q: How many test cases do we need?**
A: ~20 per state machine (5 states × 4 test cases each). We go from 100+ scattered tests to 120 organized tests.

**Q: Is this a breaking change for the API?**
A: No. API responses still return strings like `"pending"`. Internally we use types.

---

## Success Metrics

After migration, we should see:

1. **Zero typos in state handling** - Compile-time checking catches all misspellings
2. **Fewer invalid state transitions** - Validation middleware catches logic errors
3. **100% test coverage for states** - Comprehensive state machine tests
4. **Clear audit trail** - Every state change logged with context
5. **Self-documenting code** - No more guessing what states are valid
6. **Better agent experience** - Agents see valid next actions in real-time
7. **Better UX** - UI shows only valid actions; no impossible states
8. **Better DX** - New developers understand state machines by reading type definitions

---

## Questions? Issues?

If migration gets stuck:
1. Check `docs/EXPLICIT_STATE_MACHINES.md` for state definitions
2. Look at `internal/domain/state_machines.go` for the full type system
3. Refer back to this document for which files affect which state machines
4. Run the state machine tests to verify transitions work as expected
