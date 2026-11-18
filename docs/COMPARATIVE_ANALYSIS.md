# Comparative Analysis: Kora vs AgentMesh vs Our Factory

**Date:** 2026-11-18  
**Purpose:** Understand what we can learn from leading agent platforms and define our strategic direction

---

## Three Platforms, Three Problems

| Problem | Kora | AgentMesh | Our Factory |
|---------|------|-----------|-------------|
| **What?** | Control plane for governed agents | P2P network for autonomous agents | Workflow orchestrator for local agents |
| **Where?** | Enterprise (K8s, multi-tenant) | Anywhere (laptops, clouds, mesh) | Local development & single deployment |
| **Who?** | Platform engineering | Protocol engineering | Agentic engineering |
| **Trust Model** | Centralized authority | Distributed peers | Implicit (same repo/deploy) |
| **Governance** | Policy-first | Capability-first | Intent-driven |

---

## Detailed Comparison

### 1. Architecture Philosophy

#### Kora
```
Centralized Control Plane
├── Identity Service (SPIFFE/JWT)
├── Policy Service (OPA/Datalog)
├── Credential Broker
├── Inference Gateway
└── Egress Brokers

Agents run in sandboxes
Every action checked against policy before execution
```

**Philosophy:** "Every agent action must be verified by a central authority before it runs."

#### AgentMesh
```
Mesh of Autonomous Nodes
├── agentmesh-node (next to each agent)
├── agentmesh-router (relay for NAT)
├── agentmesh-control-plane (credential issuer)
└── Peer-to-peer discovery (Kad-DHT)

Agents connect directly to each other
Credentials are verifiable locally (no central authority per call)
```

**Philosophy:** "Every peer must be able to verify every credential locally."

#### Our Factory
```
Single Monolithic Control
├── DAG Executor (workflow engine)
├── State Machine Registry (explicit states)
├── Agent Launcher (creates agents)
├── Multi-Agent Coordinator (orchestrates)
└── Observability (tracing, logging)

Agents run in process or as goroutines
No external infrastructure required
```

**Philosophy:** "Simplicity first; add complexity only when needed."

---

### 2. Identity Model

#### Kora: SPIFFE JWT-SVID

```
Human or Agent requests token
        ↓
Control Plane verifies identity (OIDC, K8s SA, SPIFFE)
        ↓
Issues signed JWT (short-lived, offline verifiable)
        ↓
JWT used for all API calls
        ↓
Policy gateway checks JWT + policy before allowing action
```

**Properties:**
- Centralized issuance
- Offline verification
- Delegation with defined scope
- Multi-tenant isolation
- Revocation requires new token

#### AgentMesh: Biscuit + Peer ID

```
Agent enrolls with control plane (OIDC or bootstrap token)
        ↓
Control plane turns it into Biscuit (capability-based)
        ↓
Biscuit bound to agent's cryptographic key
        ↓
Agent carries Biscuit to peers
        ↓
Any peer verifies Biscuit locally (no call home needed)
        ↓
Agent can attenuate Biscuit for sub-tasks (narrow scope)
```

**Properties:**
- Distributed verification
- Cryptographic attestation
- Capability-based (what you can do)
- Attenuable (narrowable by intermediaries)
- Local verification (no central authority)

#### Our Factory: Implicit Identity

```
Agent created by factory during launch
        ↓
Agent tagged with:
  - ID (unique)
  - Type (code-reviewer, bug-finder, etc.)
  - Parent (which agent launched it?)
  - Scope (what can it access?)
        ↓
State machine tracks lifecycle
        ↓
Observability logs all state transitions
```

**Current:** No explicit credential model (they all share the same process space)

**Future:** Adopt capability-based model from AgentMesh for multi-process agents

---

### 3. Backend Architecture Patterns

#### Kora Backend Patterns

```go
// No ORMs
rows, err := db.Query(ctx, 
  "SELECT id, name FROM widgets WHERE status = $1", 
  "active")

// Service clients in internal/platform/
type InferenceClient struct {
  base *httpclient.Base
}

// Logging with context
logger.Info(ctx, "action completed", 
  logging.Field("action_id", actionID))

// Error wrapping with %w
if err != nil {
  return fmt.Errorf("operation failed: %w", err)
}

// No test frameworks (plain testing package)
t.Run("case", func(t *testing.T) {
  if got != want {
    t.Errorf("got %v, want %v", got, want)
  }
})

// AST-enforced conventions
// (can't use http.Error, json.NewDecoder, etc.)
```

#### AgentMesh Backend Patterns

```protobuf
// All contracts in .proto (one schema, two encodings)
message EnrollRequest {
  string identity_provider_token = 1;
  string peer_id = 2;
}

// Mesh protocol: binary protobuf
// Operator plane: protojson

// Policy is Datalog text (portable across implementations)
PolicyConfigGetResponse.datalog_rules = "node(\"alice\") :- true."

// Instant types use google.protobuf.Timestamp (not int64)
message Event {
  google.protobuf.Timestamp event_time = 1;
}

// Testing pyramid with strict limits
// Unit tests: fast, most coverage
// Integration tests: <10s each, API compliance
// E2E tests: Bats scripts, critical user journeys only
```

#### Our Factory Adoption

**From Kora:**
- Explicit SQL (no ORMs)
- Service clients in `internal/platform/`
- Structured logging with context
- Error wrapping with `%w`
- Plain `testing` package

**From AgentMesh:**
- Protobuf for all contracts
- Datalog for policy (later)
- Testing pyramid discipline
- Instant types (not int64)

**Keep Simple:**
- One binary (not 20+ microservices)
- Monolithic database (SQLite for now)
- No complex deployment (K8s optional)

---

### 4. Frontend Architecture

#### Kora Frontend Pattern (Excellent)

```
RSC Page (server-side fetch)
  ↓
Pre-fetches data with auth token
  ↓
Passes initial data to Tab Shell
  ↓
Tab Shell mounts Provider + Hook
  ↓
Hook manages state (or Zustand later)
  ↓
Context distributes to Components
  ↓
Components read-only (no fetch, no logic)

BFF Route (api/widgets/route.ts)
  ↓
Proxies client fetches through session cookie
  ↓
Calls shared server service
  ↓
Returns typed JSON

Client Service (widgets-api.ts)
  ↓
Calls BFF via route registry
  ↓
Never throws (returns discriminated union)
  ↓
Hook pattern-matches on result
```

**Key Innovation:** Interface as upgrade contract
```typescript
interface WidgetsStateShape {
  widgets: WidgetRow[];
  status: LoadStatus;
  error: string | null;
  refresh(): Promise<void>;
}

// Both useWidgets hook and future Zustand store implement this
// Components only know about interface
// Swapping implementation = zero component changes
```

#### AgentMesh Frontend Pattern (Not Applicable)

AgentMesh is Go-only. No frontend guidance.

#### Our Factory Adoption

**Apply Kora's pattern to Agent Orchestrator:**

```
_types/index.ts
_services/agents-server.ts (server fetch)
_services/agents-api.ts (client fetch via BFF)
_hooks/use-agents.ts (hook + AgentStateShape interface)
_context/agents-context.tsx (provider)
_components/agents-tab-shell.tsx (RSC boundary)
app/api/agents/route.ts (BFF proxy)

Result: Type-safe, zero loading flashes, components stay simple
```

---

### 5. Development Workflow

#### Kora: PRD → ADR → Implementation

```
1. PRD (Product Requirements)
   - Problem statement
   - Solution overview
   - Success criteria
   - File: docs/prd/

2. ADR (Architecture Decision Record)
   - Context: Why are we solving this?
   - Options: What approaches?
   - Decision: Which and why?
   - Consequences: What changes?
   - File: docs/adr/{date}-{feature}.md

3. Implementation
   - Code references ADR in commits
   - Decision rationale stays in docs
   - Not lost in PR comments
```

#### AgentMesh: Strict Modularity + Testing Pyramid

```
1. Modularity
   - Components communicate only via API
   - agentmesh-node knows ONLY agentmesh.proto
   - No shared internal Go packages between binaries

2. Testing Pyramid (enforced)
   - Unit tests: Most coverage, fast
   - Integration: API compliance, <10s each
   - E2E: Critical user journeys only (Bats)
   - Flaky test = bug in code, not test

3. Linting
   - AST-based rule checking
   - Custom rules for domain conventions
   - Can't opt-out (no //nolint)
```

#### Our Factory Adoption

**Start with:**
1. PRD template for major features
2. ADR template for decisions
3. Testing pyramid (push coverage down)
4. Extend `.golangci.yml` with state machine rules
5. Plain `testing` package (no frameworks)
6. Flaky test investigation (not retries)

---

## Decision Matrix: What to Adopt When

### Phase 1: Foundation (Now)

| Item | From | Priority | Effort | Value |
|------|------|----------|--------|-------|
| RSC + BFF + hook pattern | Kora | Critical | 2w | Eliminates UI loading flashes |
| Testing pyramid | AgentMesh | High | 1w | 10x faster tests |
| Protobuf for APIs | AgentMesh | Medium | 2w | Multi-language support |
| AST-enforced rules | Kora | High | 1w | Catch bugs at lint time |

### Phase 2: Governance (Weeks 2-4)

| Item | From | Priority | Effort | Value |
|------|------|----------|--------|-------|
| Capability-based credentials | AgentMesh | High | 1w | Local verification |
| State machine registry | Both | Critical | 1w | Type-safe orchestration |
| Error handling | Both | High | 3d | Consistent errors |

### Phase 3: Distribution (Weeks 5-8)

| Item | From | Priority | Effort | Value |
|------|------|----------|--------|-------|
| MCP integration | AgentMesh | Medium | 2w | Standard agent interface |
| Service discovery | AgentMesh | Medium | 2w | Dynamic routing |
| Cross-agent calling | Both | High | 3w | Agent-to-agent comms |

### Phase 4: Scale (Weeks 9+)

| Item | From | Priority | Effort | Value |
|------|------|----------|--------|-------|
| Multi-tenant isolation | Kora | Low | 4w | Enterprise-ready |
| Distributed tracing | Both | Low | 2w | Observability |
| Policy enforcement | Kora/AgentMesh | Low | 3w | Access control |

---

## Our Unique Path Forward

### What We Take From Each

**From Kora:**
- Frontend architecture (RSC + BFF + hook + context)
- Backend conventions (explicit SQL, service clients, structured logging)
- Development workflow (PRD/ADR/implementation)
- Error handling (error wrapping with %w)
- Linting enforcement (AST rules)

**From AgentMesh:**
- Identity model (capability-based credentials)
- Service discovery pattern
- MCP integration
- Testing discipline (pyramid, flaky = bug)
- Protobuf contracts

**From Our Factory:**
- State machines (keep!)
- DAG executor (keep!)
- Multi-agent coordination (keep!)
- Simplicity (no K8s, no distributed consensus needed)

### The Synthesis

```
Single Binary (Simple)
├── Clear Layers
│   ├── API Layer (protobuf + gRPC/REST)
│   ├── Service Layer (explicit SQL, structured logging)
│   ├── Domain Layer (state machines, entities)
│   └── Execution Layer (DAG executor, goroutines)
├── Frontend (Next.js with Kora patterns)
├── Testing (pyramid: unit > integration > E2E)
└── Observability (tracing, metrics, audit logs)

Agents (Local Processes)
├── Launched by factory
├── Tagged with identity + scope
├── Communicating via MCP
├── Publishing capabilities to registry
└── Calling each other directly

Governance (Optional, Added Later)
├── Policy enforcement (Datalog)
├── Credential attenuation (Biscuit-lite)
├── Multi-tenant isolation (Kora model)
└── Distributed verification (AgentMesh model)
```

---

## Immediate Next Steps (This Sprint)

1. **Refactor Agent Orchestrator Frontend** (Kora pattern)
   - Add RSC + BFF + hook + context
   - Zero loading flashes
   - Type-safe API contracts

2. **Extend `.golangci.yml`** (Kora + AgentMesh patterns)
   - Forbid raw string state comparisons
   - Require error wrapping with %w
   - Enforce service client pattern

3. **Define Capability Model** (AgentMesh pattern)
   - Add `AgentCapability` struct
   - Implement local verification
   - Start with simple version

4. **Reorganize Testing** (AgentMesh pattern)
   - Push coverage down (unit > integration > E2E)
   - Time-bound integration tests
   - Use Bats for critical flows

---

## Long-Term Vision (3-6 Months)

```
Month 1-2: Foundation
├── Frontend patterns (Kora)
├── Backend conventions (Both)
└── Testing pyramid (AgentMesh)

Month 2-3: Governance
├── Capability credentials (AgentMesh)
├── State machine validation (Ours)
└── Error handling (Both)

Month 3-4: Integration
├── MCP as standard interface (AgentMesh)
├── Service discovery (AgentMesh)
├── Cross-agent calling (Both)

Month 5-6: Scale
├── Multi-tenant (Kora)
├── Policy enforcement (Kora/AgentMesh)
├── Distributed observability (Both)
└── Production readiness

Result: A platform that combines:
✅ Governance rigor (Kora)
✅ Autonomous agent patterns (AgentMesh)
✅ Simplicity and focus (Ours)
```

---

## Conclusion

**Don't copy either platform entirely.** Both solve different problems at different scales.

**Steal from both strategically:**
- **Kora's frontend architecture** is production-proven and eliminates entire classes of UI bugs
- **Kora's backend conventions** make code predictable and testable
- **AgentMesh's credential model** enables trust without central authority
- **AgentMesh's testing discipline** catches races and prevents flaky tests
- **AgentMesh's MCP integration** makes agents discoverable and standard

**Keep our advantages:**
- Single binary (simpler than Kora's 20+ microservices)
- DAG execution (more flexible than linear workflows)
- State machines (better than raw status strings)
- Explicit orchestration (better than magical autoincrement)

**The result:** A platform that's simple to understand, powerful enough for complex workflows, and ready to evolve toward enterprise governance when needed.

---

## References

- **Kora:** https://github.com/runkora/kora
  - Architecture: `docs/KORA_DEEP_DIVE.md`
  - Frontend patterns: `frontend/docs/golden-paths/frontend-state-pattern.md`
  - AGENTS.md: Development guidelines

- **AgentMesh:** https://github.com/google/agentmesh
  - Architecture: `docs/AGENTMESH_ANALYSIS.md`
  - README: https://github.com/google/agentmesh/blob/main/README.md
  - AGENTS.md: Strict engineering guidelines

- **Our Factory:**
  - State Machines: `docs/EXPLICIT_STATE_MACHINES.md`
  - DAG Executor: `docs/DAG_WORKFLOW_ENGINE.md`
  - Agent Orchestrator: `docs/AGENT_ORCHESTRATOR_UI.md`
