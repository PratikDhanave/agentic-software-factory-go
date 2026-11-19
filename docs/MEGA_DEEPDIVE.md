# The Agentic Platform Trilogy: Deep Dive Analysis

**Kora × AgentMesh × kora-platform = Production Agent Control**

**Date:** 2026-11-19  
**Purpose:** Comprehensive synthesis of three agent platforms and integration roadmap for Agentic Software Factory

---

## SECTION 1: FOUNDATIONAL CONCEPTS

### Three Distinct Problems

The agent platform landscape solves three different problems:

**Kora (Control Plane):** "How do we govern agents in a multi-tenant enterprise?"
- Centralized authority
- Policy-first governance
- Credential lifecycle management
- Audit trails and observability

**AgentMesh (P2P Network):** "How do agents discover and trust each other autonomously?"
- Peer-to-peer discovery
- Cryptographic credentials
- Local verification (no central authority per call)
- Autonomous agent-to-agent communication

**kora-platform (Operations):** "How do we operationalize agent control at scale?"
- GitOps automation
- Multi-tenant infrastructure
- Production hardening
- Deployment validation

### Core Tensions Resolved

| Tension | Kora Solution | AgentMesh Solution | Factory Solution |
|---------|---------------|-------------------|------------------|
| **Centralization vs Distribution** | Centralized authority with delegation | Distributed trust with Biscuit | Monolithic with extensibility |
| **Governance vs Autonomy** | Policy enforces before execution | Capabilities enable autonomous action | State machines balance both |
| **Trust Model** | JWT tokens + policy validation | Cryptographic credentials locally verifiable | Implicit (same process) |
| **Scalability** | Multi-tenant isolation | P2P discovery scaling | Workflow DAG scaling |
| **Security** | Edge policy enforcement | Zero-trust by default | Explicit state transitions |

### Trust Models Explained

**JWT (Kora's Model)**
```
Human/Service enrolls → Control plane issues JWT
                     ↓
JWT includes claims (identity, scope, expiry)
                     ↓
Every request validates JWT at gateway
                     ↓
Policy engine checks claims against RBAC
```
Single point of verification; revocation requires new token.

**Biscuit (AgentMesh's Model)**
```
Authority issues base credential
                     ↓
Intermediate agent attenuates for specific task
                     ↓
Peer verifies credential cryptographically
                     ↓
No central authority needed for verification
```
Local verification; attenuation without re-signing.

**Implicit (Our Factory's Current Model)**
```
Agent created in same process
                     ↓
State machine tracks lifecycle
                     ↓
All communication local/in-memory
```
Simplicity; not suitable for distributed multi-tenant.

---

## SECTION 2: KORA DEEP DIVE (800+ words)

### 2.1 Complete Architecture Walkthrough

Kora is organized around **three architectural questions**:

1. **Who is this agent?** → Identity + Attestation
2. **What can it access?** → Policy + Enforcement
3. **How does it connect?** → Credentials + Brokers

#### Identity Layer (SPIFFE/JWT-SVID)

```go
// From runkora/kora design
type AgentIdentity struct {
  ID           string    // Unique agent ID
  Type         string    // "code-reviewer", "bug-finder", etc.
  Attestation  string    // Workload attestation (SPIFFE)
  CreatedAt    time.Time // When identity was issued
  ExpiresAt    time.Time // Short-lived (minutes to hours)
  Delegator    string    // Who authorized this agent
  Scope        []string  // What this agent can access
}
```

**Key insight:** Agents have machine-verifiable identity, same as humans. Delegation is explicit with bounded scope.

#### Policy Engine (OPA Datalog)

Kora uses **Open Policy Agent** with Datalog rules:

```datalog
# Example policy rule
allow_tool(Agent, Tool) :-
  agent(Agent),
  has_role(Agent, "code-reviewer"),
  tool(Tool),
  tool_category(Tool, "analysis"),
  not tool_restricted(Tool).

# Policy is evaluated BEFORE every action
# Invalid actions are logged and blocked
```

**Enforcement pattern:**
```
Agent requests action
       ↓
Edge gateway intercepts request
       ↓
Policy engine evaluates Datalog rules
       ↓
✓ Allowed → Forward to service
✗ Denied → Log + return 403
```

Crucially: **Policy is evaluated before execution**, not after.

#### Credential Broker (Just-In-Time Injection)

```
Agent needs to call external service
         ↓
Agent calls credential broker
         ↓
Broker validates request against policy
         ↓
Broker exchanges Biscuit → JWT for external service
         ↓
JWT injected into outbound request
         ↓
Agent never handles credentials directly
```

**Why this matters:** 
- No credential sprawl (secrets never stored on agents)
- Upstream authority preserved (external service trusts their own STS)
- Audit trail (every credential exchange is logged)
- Revocation works instantly (new token issued with updated scope)

#### Inference Gateway (Model Routing)

```
Agent requests inference
       ↓
Gateway routes based on:
  - Model availability
  - Cost optimization
  - Rate limiting per agent
  - Latency requirements
       ↓
Request marked with agent ID (for audit)
       ↓
Response includes usage metrics
```

Multi-model support with routing intelligence; agents don't choose providers directly.

### 2.2 Backend Engineering Patterns

**No ORMs, Explicit SQL**

```go
// ❌ Kora forbids
import "gorm.io/gorm"
db.Where("status = ?", "active").Find(&agents)

// ✅ Kora requires
rows, err := db.Query(ctx,
  "SELECT id, name, status FROM agents WHERE status = $1",
  "active")
```

**Why:** Control. Explicit SQL shows exactly what database sees; easy to optimize; no magic behavior.

**Error Wrapping with %w**

```go
// ❌ Kora forbids
return fmt.Errorf("operation failed: %s", err.Error())  // Loses chain

// ✅ Kora requires
return fmt.Errorf("operation failed: %w", err)         // Preserves chain
```

Enables `errors.Is()`, `errors.As()` for proper error handling.

**Service Clients in internal/platform/**

```
internal/platform/clients/
├── identity/client.go       // Type-safe identity client
├── policy/client.go         // Type-safe policy client
├── inference/client.go      // Type-safe inference client
└── httpclient/base.go       // Shared base (tracing, retries, timeout)
```

Every service-to-service call goes through typed client in `internal/platform/`. Tracing, retries, timeout all built-in. Never instantiate `http.Client{}` directly in feature code.

**Structured Logging with Context**

```go
logger.Info(ctx, "agent enrolled",
  logging.Field("agent_id", agentID),
  logging.Field("agent_type", agentType),
  logging.Field("delegator", delegatorID))

// Output includes automatic context:
// - Trace ID (for correlation)
// - User ID (from session)
// - Request ID
// - Timestamp
```

**Testing Pyramid Discipline**

```
Unit Tests (70%)
  ├─ State transition logic
  ├─ Policy evaluation
  └─ Credential validation
Integration Tests (25%)
  ├─ Agent enrollment flow
  ├─ Gateway routing
  └─ Audit logging
E2E Tests (5%)
  └─ Critical user journeys only
```

**Golden rule:** If test can run at lower level, it MUST. Flaky test = bug in code, not test.

### 2.3 Frontend Architecture (RSC + BFF + Hooks + Context)

**Server-Side Rendering (RSC)**

```tsx
// pages/agents/page.tsx (React Server Component)
export default async function AgentsPage() {
  // Fetch server-side (no loading flash)
  const agents = await fetchAgentsFromGateway(session.accessToken);
  
  // Pass to client shell
  return <AgentsTabShell initialAgents={agents} />;
}
```

**Backend For Frontend (BFF)**

```tsx
// app/api/agents/route.ts
export const GET = withAuth(async (req, session) => {
  // Session cookie holds auth token (browser never sees it)
  const result = await fetchAgentsFromGateway(session.accessToken);
  
  if (!result.ok) {
    return NextResponse.json({ code: 'upstream_error' }, { status: 502 });
  }
  
  return NextResponse.json({ data: result.agents });
});
```

**Hook + Context Pattern**

```tsx
// Define once (contract)
interface AgentStateShape {
  agents: Agent[];
  status: 'idle' | 'loading' | 'success' | 'error';
  error: string | null;
  refresh(): Promise<void>;
}

// Hook (current implementation)
function useAgents(initialData, initialError): AgentStateShape {
  const [agents, setAgents] = useState(initialData);
  // ...
  return { agents, status, error, refresh };
}

// Can be swapped for Zustand later without changing components
// Components only know about AgentStateShape interface
```

**Benefits:** Type-safe API contracts, zero loading flashes, state management future-proof.

### 2.4 Development Workflow

**PRD → ADR → Implementation**

1. **PRD (docs/prd/feature.md)**
   - Problem statement
   - Solution overview
   - Success criteria

2. **ADR (docs/adr/{date}-{feature}.md)**
   - Context: Why solve this?
   - Options: What approaches?
   - Decision: Which one, why?
   - Consequences: What changes?

3. **Implementation**
   - Code references ADR in commits
   - Decision rationale stays in docs (not PR comments)

**Automated Enforcement via AST Rules**

```yaml
# .golangci.yml custom rules
rules:
  - name: spanSetStatusLocalErr
    bans: "span.SetStatus(codes.Error, err.Error())"
    instead: "observability.RecordFailure(span, err, sentinel)"

  - name: noRawJsonDecodeRequest
    bans: "json.NewDecoder(r.Body).Decode(...)"
    instead: "api.DecodeJSON(ctx, r, w, logger, &req)"
```

Violations fail `make check`. Can't opt-out with `//nolint`.

### 2.5 Security Model

**Workload Attestation (SPIFFE)**

```
Agent pod starts
     ↓
SPIFFE attestor validates:
  - Pod identity (name, namespace)
  - Service account
  - Signing key
     ↓
Control plane issues SVID (Signed SPIFFE Verifiable Identity)
     ↓
Agent can prove identity to peers
```

**Token Exchange (RFC 8693)**

```
Agent has Biscuit credential
            ↓
Needs to call external AWS API
            ↓
Agent requests JWT from control plane:
  POST /sts/token
  Body: { credential: <biscuit>, target: "arn:aws:iam::123456789..." }
            ↓
Control plane validates scope
            ↓
Issues short-lived JWT with AWS assumed role
            ↓
Agent uses JWT at AWS STS
```

**Delegation with Scope**

```go
type DelegatedCredential struct {
  Authority      string    // Who issued this
  Agent          string    // Who holds this
  AllowedActions []string  // Scoped to specific tools
  AllowedResources []string // Scoped to specific resources
  ExpiresAt      time.Time  // Short-lived
  Signature      string     // Cryptographic proof
}
```

Delegation is **explicit and bounded**. Alice can delegate to Bob for "code-review-only" without granting Alice's full permissions.

### 2.6 Multi-Tenancy Design

```
kora-system (control plane - shared)
├── Identity service
├── Policy engine
├── Inference gateway
└── Egress brokers

kora-tenant-team-a (isolated)
├── Agent instances
├── Workflow state
└── Audit logs (team-a scoped)

kora-tenant-team-b (isolated)
├── Agent instances
├── Workflow state
└── Audit logs (team-b scoped)
```

**Isolation guarantees:**
- Network: Istio enforces pod-to-pod policies
- Storage: Separate databases per tenant
- Policy: Policy engine validates tenant boundaries
- Audit: Logs tagged with tenant ID

---

## SECTION 3: AGENTMESH DEEP DIVE (800+ words)

### 3.1 P2P Networking Fundamentals

AgentMesh uses **libp2p** (Protocol Labs' modular P2P stack):

```go
// Core stack
import (
  "github.com/libp2p/go-libp2p"
  "github.com/libp2p/go-libp2p-kad-dht"      // Kademlia DHT
  "github.com/libp2p/go-libp2p-pubsub"       // Gossip
  "github.com/libp2p/go-libp2p-gostream"     // HTTP-like streams
)
```

**Three network layers:**

1. **Transport** (physical connectivity)
   - TCP, UDP, QUIC
   - Works behind NAT (relay fallback)

2. **Handshake** (authentication)
   - Peer ID derivation from crypto key
   - NoiseXK protocol for TLS-like security
   - Mutual authentication by default

3. **Protocol** (application logic)
   - Mesh protocol (enrollment, auth streams, DHT)
   - Data plane (service calls, MCP, OpenAI API)
   - Relay (NAT traversal)

### 3.2 DHT-based Service Discovery

**Kademlia DHT (Distributed Hash Table)**

```
Agent publishes: "I provide service:analyze"
        ↓
DHT propagates through network (gossip)
        ↓
Other agents query DHT for "service:analyze"
        ↓
Discover agent1, agent2, agent3
        ↓
Attempt direct P2P connection
        ↓
Fall back to relay if behind NAT
```

**Why DHT over central registry:**
- No single point of failure
- Scales to 1000s of nodes
- Decentralized (no operator)
- Automatic propagation

### 3.3 Biscuit Credential System

**Biscuit: Cryptographic Authorization Tokens**

Unlike JWT (opaque to receiver), Biscuit is verifiable locally:

```
Block 0: Authority rules (RBAC from policy)
         can_call("alice", "service:analyze")
         can_call("bob", "service:*")

Block 1+: Attenuation rules (task-scoped)
          scoped_to("service:analyze")
          scoped_to("user:alice")
          scoped_to("duration:1h")
```

**Attenuation (Narrowing by intermediary)**

```
Alice: "I can call any service"
  ↓ (delegates to Bob for one task)
Bob: "I can call service:analyze only"
  ↓ (cryptographically signed, Alice doesn't sign again)
Charlie: "Let me verify Bob's credential"
  ↓ (checks crypto proof locally, no central authority)
Verified! Bob can call service:analyze.
```

**Key innovation:** Credentials can be narrowed by intermediaries without re-signing or calling the authority.

### 3.4 MCP Integration (Tool Discovery Protocol)

```
Agent exposes: agentmesh-node run --mcp-port 8080
        ↓
MCP server on localhost:8080
        ↓
Advertises capabilities:
  - "tool:code-analysis"
  - "tool:test-generation"
  - "model:claude-sonnet"
        ↓
Other agents discover via MCP
        ↓
Can call tools over libp2p
```

Standard protocol; agents are MCP servers; discovery is automatic.

### 3.5 Zero-Trust Architecture

**Two Properties (from AgentMesh's design)**

```
1. Nothing is reachable by default
   - Pod starts with no exposed services
   - Policy grants (by name, never by IP)
   - Deny everything except explicit allow

2. Identity comes from identity provider
   - OpenID Connect token or bootstrap token
   - Control plane turns it into Biscuit
   - Offline verifiable credential
```

**Example policy:**

```datalog
# Service must be explicitly published
can_publish("agent:alice", "service:analyze").

# Service can be called only by explicit grant
can_call("agent:bob", "service:analyze") :-
  has_peer_id("agent:bob"),
  not revoked("agent:bob").
```

### 3.6 Testing Discipline

**Testing Pyramid (Strict)**

```
Unit Tests (70%)
  - Isolated functions
  - No network calls
  - Fast (<10ms)

Integration Tests (25%)
  - Module interactions
  - Local DHT
  - API compliance
  - <10s each

E2E Tests (5%)
  - Critical user journeys only
  - Actual libp2p mesh
  - Real discovery
```

**Golden rule:** Flaky test = bug in code.

```
If test fails once and passes on retry:
  1. Identify race condition in code
  2. Fix code (not test)
  3. Run under load: stress -p 8 binary -test.run TestName
  4. Verify fixed before merging
```

### 3.7 Code Conventions

**Protobuf for All Contracts**

```protobuf
// api/agentmesh.proto
message EnrollRequest {
  string identity_provider_token = 1;
  string peer_id = 2;
}

// Wire format #1: Binary protobuf (mesh protocol)
// Wire format #2: ProtoJSON (operator plane API)
// Never: Go struct + JSON tags
```

**Benefits:**
- Schema versioning
- Cross-language support
- Forward compatibility
- Single source of truth

**Modularity Rule**

```
agentmesh-control-plane
agentmesh-router
agentmesh-node

Must communicate ONLY via api/agentmesh.proto.
No shared internal Go packages between binaries.
Complete isolation.
```

---

## SECTION 4: KORA-PLATFORM DEEP DIVE (600+ words)

### 4.1 GitOps Model (ArgoCD)

**Git as Source of Truth**

```yaml
# argocd/dev/kora.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kora-platform

spec:
  sources:
    - chart: kora-crds
      targetRevision: 0.13.0-rc.7
    - chart: kora
      targetRevision: 0.13.0-rc.7
  
  destination:
    namespace: kora-system
  
  syncPolicy:
    automated:
      prune: true        # Delete removed resources
      selfHeal: true     # Revert manual changes
```

**Workflow:**
```
Developer commits to main
        ↓
ArgoCD detects change
        ↓
Pulls Helm charts
        ↓
Renders values
        ↓
Applies manifests
        ↓
Automatic drift correction (selfHeal)
```

**Benefits:**
- Declarative infrastructure
- Version-controlled everything
- Automatic rollback (previous commit)
- Easy multi-environment (dev/prod branches)

### 4.2 Helm-based Packaging

**Version-locked dependencies**

```makefile
KORA_CRDS_VERSION ?= 0.13.0-rc.7
KORA_VERSION ?= 0.13.0-rc.7

# Pinned in source control
# Every deployment uses exact same version
# No surprises from chart changes
```

**Three chart strategy:**

```
1. kora-crds (CRD definitions)
   - KoraTenant
   - KoraAgent
   - KoraPolicy

2. kora (Control plane)
   - Identity service
   - Policy engine
   - Inference gateway

3. kora-platform (Custom overlays)
   - Dev/prod patches
   - Tenant-specific configs
```

### 4.3 Kustomize Overlay Patterns

**Layer-based Composition**

```
platform/
├── base/              # Common setup
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── configmap.yaml
│   └── rbac.yaml
├── dev/               # Dev overrides
│   ├── kustomization.yaml
│   ├── patches/
│   │   ├── resources.yaml   (small quotas)
│   │   └── replicas.yaml    (1 replica)
│   └── secrets.yaml
└── prod/              # Prod overrides
    ├── kustomization.yaml
    ├── patches/
    │   ├── resources.yaml   (large quotas)
    │   ├── replicas.yaml    (3 replicas)
    │   └── tls.yaml
    └── secrets.yaml
```

**Rendering process:**

```bash
kustomize build platform/base          # Base manifests
  ↓
Apply dev/ patches
  ↓
Render final manifests
  ↓
Validate with kubeconform
  ↓
Deploy to cluster
```

### 4.4 Multi-Tenant Architecture

**Namespace Isolation**

```
kora-system/
├── identity
├── policy
├── gateway

kora-tenant-team-a/
├── namespace (kora-tenant-team-a)
├── rbac (team-a only)
├── agents (team-a instances)
└── secrets (team-a credentials)

kora-tenant-team-b/
├── namespace (kora-tenant-team-b)
├── rbac (team-b only)
├── agents (team-b instances)
└── secrets (team-b credentials)
```

**ArgoCD manages each tenant independently:**

```yaml
# argocd/prod/tenants.yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: kora-tenants

spec:
  generators:
    - list:
        elements:
          - name: team-a
          - name: team-b
          - name: team-c

  template:
    metadata:
      name: kora-tenant-{{ name }}
    spec:
      source:
        path: tenants/{{ name }}/prod
```

### 4.5 Deployment Validation

**Automated Checks**

```bash
make kora-schemas     # Generate JSON schemas from Helm
make lint             # Validate all manifests
  - kubeconform (K8s schema validation)
  - Custom CRD schemas (runkora.io/*)
  - Kustomization validation
  - YAML syntax
```

**Safety gates:**
- Schema validation before deploy
- Drift detection (ServerSideDiff)
- RespectIgnoreDifferences (operator-managed fields)
- Retry backoff for startup sequencing

### 4.6 Production Readiness Checklist

✅ Version-locked dependencies  
✅ Multi-environment support  
✅ Automated validation  
✅ Multi-tenant isolation  
✅ Pod security policies  
✅ Automatic healing  
✅ Resource isolation  
✅ Smoke tests (actual agents)  
✅ Audit logging per tenant  
✅ GitOps automation  

---

## SECTION 5: ARCHITECTURAL INTEGRATION (500+ words)

### How Three Platforms Complement Each Other

```
┌────────────────────────────────────┐
│  Kora Control Plane (Governance)   │
│  - Identity (SPIFFE/JWT)           │
│  - Policy engine (OPA)             │
│  - Credential broker               │
│  - Inference gateway               │
└────────────┬───────────────────────┘
             │ (governs)
             ↓
┌────────────────────────────────────┐
│  Agent Mesh (Peer-to-Peer Network) │
│  - Service discovery (DHT)         │
│  - P2P connectivity (libp2p)       │
│  - Biscuit credentials (local auth)│
│  - MCP tool protocol               │
└────────────┬───────────────────────┘
             │ (operates)
             ↓
┌────────────────────────────────────┐
│  Kora Platform (Operations)        │
│  - GitOps (ArgoCD)                 │
│  - Deployment (Helm/Kustomize)     │
│  - Multi-tenancy (K8s namespaces)  │
│  - Validation (kubeconform)        │
└────────────────────────────────────┘
```

### Data Flow Through All Three

**Identity & Trust Path**

```
1. OIDC Provider
   ↓
2. Kora Control Plane exchanges token → SPIFFE SVID
   ↓
3. Agent enrolls with control plane
   ↓
4. Control plane issues Biscuit credential
   ↓
5. Agent carries Biscuit to peers (via AgentMesh)
   ↓
6. Peer verifies Biscuit locally (no call to Kora)
   ↓
7. Trust established via cryptography
```

**Service Call Path**

```
Agent A → "Find service:analyze"
        ↓
AgentMesh DHT → "agent-b provides this"
        ↓
Agent A → Agent B (direct P2P or via relay)
        ↓
Agent B validates Biscuit (locally verifiable)
        ↓
Agent B executes service:analyze
        ↓
Result returned to Agent A
        ↓
Kora gateway logs access (audit trail)
```

### Communication Paths

**Central (Kora):**
- Identity validation
- Policy decisions
- Credential exchange
- Audit logging

**Distributed (AgentMesh):**
- Service discovery
- P2P calls
- Credential verification (local)
- Tool execution

**Operational (kora-platform):**
- Deployment orchestration
- Configuration management
- Multi-tenancy
- Validation

### Governance Layers

```
Layer 1: Policy (Kora)
         "Who is allowed to do what?"
         Evaluated at gateway + peers

Layer 2: Credentials (AgentMesh/Biscuit)
         "Can you prove you're allowed?"
         Verified locally by peers

Layer 3: Audit (Kora + AgentMesh)
         "Who did what, when, and why?"
         Logged at central + distributed points

Layer 4: Operational (kora-platform)
         "Is the system healthy?"
         Monitored via observability stack
```

---

## SECTION 6: OUR FACTORY INTEGRATION (400+ words)

### Current State

**What We Have:**
- State machines (explicit governance)
- DAG executor (workflow orchestration)
- Agent orchestrator (UI for launching)
- Multi-agent coordinator (calling agents)
- Monolithic Go binary

**What We're Missing:**
- Containerization
- Multi-environment support
- Multi-tenancy
- Production deployment
- Governance layer (identity/policy)

### What We Adopt

**From Kora:**
✅ Frontend architecture (RSC + BFF + hooks)  
✅ Backend conventions (explicit SQL, service clients)  
✅ Error handling (%w wrapping)  
✅ Structured logging  
✅ Testing pyramid discipline  
✅ PRD/ADR workflow  

**From AgentMesh:**
✅ MCP as standard agent interface  
✅ Protobuf for contracts  
✅ Zero-trust principles  
✅ Testing discipline (flaky = bug)  
✅ P2P discovery ideas  

**From kora-platform:**
✅ GitOps model (ArgoCD)  
✅ Helm-based packaging  
✅ Kustomize overlay pattern  
✅ Multi-tenant architecture  
✅ Validation automation  

### What We Build Differently

**Simpler Identity Model:**
- Start with simple capability-based credentials (not full Biscuit)
- Implicit trust within process (before multi-process)
- Explicit delegation later

**Monolithic First:**
- Single binary (not 20+ microservices)
- One database (SQLite for now)
- State machines for governance (not OPA)

**Workflow-Centric:**
- DAG execution is core (not API-first)
- Agents created by factory (not self-enrolled)
- State machines orchestrate (not policies)

### 24-Month Roadmap

**Months 1-3: Foundation**
- Phase 1: Containerization (Docker)
- Phase 2: Helm chart basics
- Phase 3: Testing pyramid refactor
- **Deliverable:** Factory runs in K8s

**Months 4-6: Governance Layer**
- Phase 1: Capability-based credentials
- Phase 2: State machine validation
- Phase 3: Audit logging
- **Deliverable:** Agent identity + audit trail

**Months 7-9: Deployment**
- Phase 1: ArgoCD integration
- Phase 2: Kustomize overlays (dev/prod)
- Phase 3: Multi-environment configs
- **Deliverable:** GitOps-managed Factory

**Months 10-12: Scaling**
- Phase 1: Multi-tenancy (namespace isolation)
- Phase 2: Service discovery (simple registry)
- Phase 3: MCP as standard interface
- **Deliverable:** Factory supports 10+ tenants

**Months 13-18: Production**
- Phase 1: Observability stack (metrics, logs, traces)
- Phase 2: Security hardening (network policies)
- Phase 3: Performance optimization
- **Deliverable:** Production-grade ops

**Months 19-24: Enterprise**
- Phase 1: 100+ tenant support
- Phase 2: Policy framework (Datalog-lite)
- Phase 3: Audit compliance
- **Deliverable:** Enterprise-ready platform

---

## SECTION 7: PATTERN LIBRARY (500+ words)

### Backend Patterns

**Explicit SQL (No ORMs)**

```go
// ❌ Avoid
db.Where("status = ?", "active").Find(&agents)

// ✅ Adopt
rows, err := db.Query(ctx,
  "SELECT id, name, status FROM agents WHERE status = $1",
  "active")
if err != nil {
  return fmt.Errorf("query agents: %w", err)
}
```

**Service Clients in internal/platform/**

```
internal/platform/clients/
├── identity/client.go
├── gateway/client.go
├── compute/client.go
└── httpclient/base.go  (shared: retries, tracing, timeout)
```

Type-safe; tracing built-in; never raw `http.Client{}`.

**Error Wrapping with %w**

```go
// ❌ Breaks error chain
return fmt.Errorf("operation failed: %s", err.Error())

// ✅ Preserves chain
return fmt.Errorf("operation failed: %w", err)

// Enables
if errors.Is(err, io.EOF) { ... }
```

**Structured Logging**

```go
logger.Info(ctx, "agent enrolled",
  logging.Field("agent_id", id),
  logging.Field("agent_type", ty))

// Auto-includes: trace_id, user_id, request_id, timestamp
```

### Frontend Patterns

**RSC + BFF + Hook + Context**

```tsx
// 1. RSC fetches server-side (no flash)
page.tsx: const agents = await fetchFromGateway(token)

// 2. BFF proxies auth (token not in browser)
api/agents/route.ts: withAuth(session => fetchFromGateway(token))

// 3. Hook manages state (interface contract)
use-agents.ts: interface AgentStateShape { ... }

// 4. Context distributes to components
agents-context.tsx: useAgentsContext() [read-only components]
```

**Type-Safe API Contracts**

```typescript
// Shared between backend and frontend
interface AgentRaw {
  id: string;
  name: string;
  status: string;
}

// Frontend view type
interface AgentRow {
  id: string;
  name: string;
  status: AgentStatus;  // Type-safe enum
}

function toAgentRow(raw: AgentRaw): AgentRow { ... }
```

**Route Registry**

```typescript
// src/lib/routing/api-routes.ts
export const ApiRoute = {
  agents: {
    GET: () => `${API_BASE}/agents`,
    POST: () => `${API_BASE}/agents`,
  },
};

// Usage: fetch(ApiRoute.agents.GET())  // Type-checked
// No magic strings allowed
```

### Testing Patterns

**Testing Pyramid**

```
Unit (70%)
├─ Isolated functions
├─ Mock external calls
└─ <10ms each

Integration (25%)
├─ Module interactions
├─ Time-bounded <10s
└─ Actual state machines

E2E (5%)
├─ Critical user journeys only
└─ Real infrastructure
```

**Flaky Test = Bug**

```
If test fails once + passes on retry:
  1. Identify race condition
  2. Fix code (not test)
  3. Run under load (stress tool)
  4. Verify before merging
```

### Deployment Patterns

**GitOps + Kustomize**

```
config/
├── base/              # Common
├── dev/               # Dev patches
│   ├── kustomization.yaml
│   └── patches/
└── prod/              # Prod patches
    ├── kustomization.yaml
    └── patches/

# No env variables. Pure overlay composition.
```

**Version-locked Dependencies**

```makefile
FACTORY_VERSION = 0.5.0  # Pinned
HELM_CHART = factory:0.5.0

# Every deployment uses exact same version
# No surprises from upstream changes
```

### Identity & Credential Patterns

**Capability-Based Credentials**

```go
type AgentCapability struct {
  Subject    string         // Which agent
  Actions    []string       // What it can do
  Resources  []string       // On what
  ExpiresAt  time.Time      // When expires
  Signature  string         // Proof
}
```

**Local Verification**

```
Peer receives credential
        ↓
Verifies signature cryptographically
        ↓
No call to central authority
        ↓
Trust established
```

### Networking Patterns

**Service Discovery**

```
Agent publishes: "tool:code-analysis"
        ↓
Registry propagates
        ↓
Peers discover via registry
        ↓
Direct P2P call
        ↓
Relay fallback if needed
```

**MCP Endpoints**

```
Agent runs: agentmesh-node run --mcp-port 8080
        ↓
MCP server advertises capabilities
        ↓
Other agents discover + call
        ↓
Standard protocol (not custom RPC)
```

---

## SECTION 8: SECURITY & TRUST (300+ words)

### Zero-Trust Principles

**Nothing is reachable by default**

```
Agent starts
    ↓
Exposes nothing until config says so
    ↓
Peers cannot call without policy grant
    ↓
Grant by service NAME (not IP)
    ↓
Deny everything else
```

**Identity comes from identity provider**

```
OpenID Connect provider
    ↓
Agent enrolls with control plane
    ↓
Control plane issues credential (Biscuit-like)
    ↓
Offline verifiable
    ↓
No central call needed for each action
```

### Credential Lifecycle

**JWT Model (Kora)**

```
1. Human/agent enrolls → Central issues JWT
2. JWT has claims (identity, scope, expiry)
3. Gateway validates JWT at each request
4. Revocation requires new token
5. Instant if control plane is reachable
```

**Biscuit Model (AgentMesh)**

```
1. Authority issues base credential
2. Intermediate narrows scope (attenuation)
3. Peer verifies cryptographically (local)
4. No central authority call per action
5. Revocation happens at credential refresh
```

**Our Factory Approach (Months 4-6)**

```
1. Simple capability list per agent
2. Signed with agent's key
3. Exchanged at enrollment
4. Verified locally by peers
5. Refresh on-demand
```

### Attenuation & Delegation

**Key Innovation: Narrowing Without Re-signing**

```
Alice: scope = ["tool:*", "service:*"]
  ↓
Delegates to Bob for one task:
Bob: scope = ["tool:code-analysis"] [cryptographically signed by Alice]
  ↓
Charlie verifies Bob's credential:
  ✓ Signature is valid
  ✓ Scope is narrower than Alice's
  ✓ Trust established
```

No central authority needed for verification.

### Policy Enforcement Layers

**Layer 1: Gateway (Central)**
```
Request arrives at Kora gateway
  ↓
Policy engine evaluates OPA rules
  ↓
✓ Allow → Forward | ✗ Deny → 403
```

**Layer 2: Peer (Distributed)**
```
Peer receives request + credential
  ↓
Verifies credential signature (local)
  ↓
Checks action against credential scope
  ↓
✓ Allow → Execute | ✗ Deny → Reject
```

### Audit Trails

**Central Audit (Kora)**
```
Every action logged:
  - Agent ID
  - Action taken
  - Policy decision (allow/deny)
  - Timestamp
  - Request ID (for tracing)
```

**Distributed Audit (AgentMesh)**
```
Local agents log:
  - Peer interaction
  - Credential verified
  - Action executed
  - Timestamp
```

Combined: full audit trail across system.

---

## SECTION 9: PRODUCTION READINESS (300+ words)

### What Makes a System Production-Ready

**Multi-Tenancy**
- Namespace isolation (Kora-platform model)
- Separate databases per tenant (or row-based isolation)
- RBAC enforces boundaries
- Audit logs tagged with tenant

**Observability**
- Metrics (Prometheus): request rates, latencies, errors
- Logs (structured): every state change, error, decision
- Traces (distributed): correlate request flow across agents
- Dashboards: health, saturation, errors

**Automation**
- Deployment: GitOps (ArgoCD)
- Scaling: Autoscaling based on metrics
- Healing: Automatic restarts, circuit breakers
- Rollback: Previous version on failure

**Validation**
- Schema: JSON schema validation (kubeconform)
- Policy: Datalog enforcement before action
- State: State machine validation (ours)
- Tests: 70% unit, 25% integration, 5% E2E

**Security**
- Network policies: Pod-to-pod enforcement
- RBAC: Role-based access control
- Secrets: Never in code, encrypted at rest
- Audit: Every action logged and verified

**Resilience**
- Backoff retries: Exponential with jitter
- Circuit breakers: Fail fast on cascade
- Timeouts: Prevent hanging
- Graceful shutdown: Drain connections before exit

### Multi-Tenancy Requirements

```
1. Network isolation
   - Istio/Envoy enforces pod policies
   - Team-a pods cannot reach team-b pods

2. Storage isolation
   - Separate database per tenant (or row-based)
   - No data leakage across boundaries

3. Policy isolation
   - Each tenant has its own policies
   - Agents can only access their tenant's resources

4. Audit isolation
   - Logs tagged with tenant ID
   - Team can see their own audit trail only
```

### Observability Requirements

```
Metrics:
  - request_duration_seconds (histogram)
  - request_count_total (counter)
  - error_rate (gauge)

Logs:
  - Every state transition
  - Every policy decision
  - Every error with context

Traces:
  - Request spans across services
  - Service spans for individual operations
  - Correlation ID in every entry
```

### Automation Requirements

```
Deployment:
  - Push to main → ArgoCD detects → automatic deploy
  - Dev uses kustomization.yaml + patches/

Healing:
  - Pod crash → kubelet restarts
  - Service degradation → alerting + runbook

Scaling:
  - Load increases → HPA spins up replicas
  - Load decreases → HPA scales down
```

---

## SECTION 10: 24-MONTH ROADMAP (400+ words)

### Roadmap Overview

This maps Kora + AgentMesh + kora-platform adoption into our Factory's evolution.

### Months 1-3: Foundation

**Phase 1: Containerization**
- Create Dockerfile for Factory binary
- Push to container registry (ghcr.io)
- Document build process

**Phase 2: Helm Chart Basics**
- helm/agent-factory/ directory
- Templates: Deployment, Service, ConfigMap
- Values for dev/prod

**Phase 3: Testing Pyramid**
- Unit: 70% coverage (fast, no IO)
- Integration: 25% (time-bounded <10s)
- E2E: 5% (critical journeys only)

**Deliverable:** Factory runs in Kubernetes

### Months 4-6: Governance Layer

**Phase 1: Capability-Based Credentials**
- Simple credential model (not full Biscuit yet)
- AgentCapability struct (actions, resources, expiry)
- Signing with agent's key

**Phase 2: State Machine Validation**
- Explicit validation at state transitions
- Audit log on every transition
- Central registry (we have this)

**Phase 3: Audit Logging**
- Structure: agent_id, action, timestamp, reason
- Storage: Dedicated audit table
- Dashboard: Show audit trail per agent

**Deliverable:** Agent identity + audit trail

### Months 7-9: Deployment

**Phase 1: ArgoCD Integration**
- Application CRD for Factory
- Watch main branch for changes
- Automatic sync

**Phase 2: Kustomize Overlays**
- config/base/ (common setup)
- config/dev/ (dev patches)
- config/prod/ (prod patches)
- No env variables, pure overlays

**Phase 3: Multi-Environment Configs**
- Dev: 1 replica, small quotas
- Prod: 3 replicas, large quotas
- Canary: Blue-green deployment

**Deliverable:** GitOps-managed Factory

### Months 10-12: Scaling

**Phase 1: Multi-Tenancy**
- Namespace per tenant (factory-tenant-{name})
- Shared control plane
- Isolated data + policies

**Phase 2: Service Discovery**
- Simple registry (not DHT yet)
- Agents advertise capabilities
- Peers query registry

**Phase 3: MCP as Standard**
- Every agent runs MCP server
- Tool discovery via MCP
- Standard protocol

**Deliverable:** Factory supports 10+ tenants

### Months 13-18: Production

**Phase 1: Observability**
- Prometheus metrics (rates, latencies, errors)
- Structured logs (JSON, trace IDs)
- Grafana dashboards
- Distributed tracing (Jaeger)

**Phase 2: Security Hardening**
- Network policies (Kubernetes)
- RBAC enforcement
- Secrets management (sealed-secrets)
- Pod security policies

**Phase 3: Performance**
- Load testing
- Latency optimization
- Resource tuning
- Caching strategies

**Deliverable:** Production-grade ops

### Months 19-24: Enterprise

**Phase 1: 100+ Tenant Support**
- Horizontal scaling
- Database sharding
- Load balancing

**Phase 2: Policy Framework**
- Datalog-lite policy engine
- Policy definition language
- Deployment validation

**Phase 3: Audit Compliance**
- Compliance reporting
- Audit retention
- Data deletion on request

**Deliverable:** Enterprise-ready platform

---

## Synthesis: From Local to Production

```
Months 1-3:   Monolithic → Containerized
Months 4-6:   Implicit trust → Explicit credentials
Months 7-9:   Manual deploy → GitOps
Months 10-12: Single tenant → Multi-tenant
Months 13-18: Dev-only → Production-grade
Months 19-24: Single-team → Enterprise-ready
```

**Key principle:** Each phase delivers working software. No waiting for the full 24-month vision.

---

**Document Version:** 1.0  
**Last Updated:** 2026-11-19  
**Next Review:** Months 1-2 of roadmap execution

