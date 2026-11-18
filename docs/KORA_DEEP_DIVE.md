# Kora Deep Dive: What We Can Learn & Adopt

**Analysis Date:** 2026-11-18  
**Repository:** https://github.com/runkora/kora  
**Focus:** Architecture patterns, frontend patterns, backend conventions, and agent governance

---

## Executive Summary

Kora is **the control plane for AI agents** — not an agent framework, but platform engineering applied to agent governance. It answers three fundamental questions for every agent in production:

1. **Who is this agent?** (Identity & attestation)
2. **What can it access?** (Policy & enforcement)
3. **How does it connect?** (Credential brokering)

**Key Insight:** Kora separates concerns beautifully. Our Factory should adopt their:
- **Frontend state architecture** (server components + BFF + hooks + context)
- **Backend conventions** (no ORMs, explicit SQL, standardized error handling)
- **Development process** (PRD → ADR → implementation)
- **Linting rules** (enforced conventions prevent drift)

---

## Part 1: Identity & Governance Model

### Three-Question Architecture

**Question 1: Who is this agent?**
- Agents get machine-verifiable identity at deploy time
- Humans and agents share one identity model
- Delegation is explicit with defined scope and full accountability
- **Why it matters:** Traceability. Every action traces back to the human who initiated it

**Question 2: What can it access?**
- Every action checked against policy **before** execution
- Admin-defined tool allowlist (everything outside is rejected)
- Enforcement at the edge, inside each pod
- **Why it matters:** Agents are never trusted to behave

**Question 3: How does it connect?**
- Agents never handle credentials directly
- Platform intercepts outbound calls
- Just-in-time credential injection
- **Why it matters:** No credential sprawl, upstream authority is preserved

### How This Applies to Our Factory

**Current State:** We have state machines for entities but no governance layer for multi-agent coordination.

**What We Should Add:**

1. **Agent Identity Registry** (`internal/governance/identity.go`)
   ```go
   type AgentIdentity struct {
       ID       string              // Unique agent ID
       Name     string              // Human-readable name
       Type     AgentType           // General-purpose, code-reviewer, etc.
       Delegator string              // Who authorized this agent?
       Scope    []string            // What can it access?
       CreatedAt time.Time          // Attestation time
   }
   ```

2. **Policy Enforcement** (`internal/governance/policy.go`)
   ```go
   type AgentPolicy struct {
       AgentID      string
       AllowedTools []string        // Whitelist only
       AllowedScopes []string       // Resource boundaries
       RateLimit    int             // Requests/min
       Timeout      time.Duration   // Max execution time
   }
   ```

3. **Credential Broker** (`internal/governance/credential_broker.go`)
   - Agents request tokens at runtime
   - Broker validates against policy
   - Tokens injected into execution environment
   - Credentials never stored on agent

---

## Part 2: Frontend Architecture (THE GOLD MINE)

### The Golden Path Pattern

Kora's frontend state management pattern is **production-proven** and solves every problem we face:

#### 1. **Type-Safe API Contract** (`src/shared/api.ts`)

```typescript
// Raw backend response (mirror Go JSON exactly)
export interface WidgetRaw {
  id: string;
  name: string;
  status: string;
  created_at: string;
}

// View type (UI-ready)
export interface WidgetRow {
  id: string;
  name: string;
  status: WidgetStatus;  // Type-safe enum
  createdAt: string;
}

// Mapping function
export function toWidgetRow(raw: WidgetRaw): WidgetRow {
  return { id: raw.id, name: raw.name, status: raw.status as WidgetStatus, ... };
}
```

**Key:** Separate raw (Go JSON) from view (UI-ready). One maps to other with explicit functions.

#### 2. **BFF (Backend For Frontend)**

```typescript
// src/app/api/widgets/route.ts
export const GET = withAuth(async (_req: NextRequest, session) => {
  const result = await fetchWidgetsFromGateway(session.accessToken);
  
  if (!result.ok) {
    return NextResponse.json(
      { code: 'upstream_error', message: 'Failed to load widgets' },
      { status: 502 }
    );
  }
  
  return NextResponse.json({ data: result.widgets });
});
```

**Key:** Browser never holds auth token. BFF proxies via session cookie. Token → Authorization header server-side only.

#### 3. **State Architecture: Hook + Context + Components**

```
Server Service (widgets-server.ts)
  ↓ shared by both
  ├→ RSC Page (page.tsx) [server-side fetch, pre-populate]
  └→ BFF Route (api/widgets/route.ts) [client-side re-fetch]

Client Service (widgets-api.ts)
  ↓ calls BFF via route registry
  
Hook (use-widgets.ts) [WidgetsStateShape interface]
  ↓ mounted by
  
Context Provider (widgets-context.tsx)
  ↓ consumed by
  
Tab Shell (widgets-tab-shell.tsx) [use client boundary]
  ↓ renders
  
Feature Components (widgets-tab.tsx) [read-only context consumers]
```

**Key Pattern:**
- Hook defines `StateShape` interface (contract)
- Both useHook + future Zustand store implement it
- Components reference interface, not implementation
- Swapping state management = zero component changes

#### 4. **Route Registry** (`src/lib/routing/api-routes.ts`)

```typescript
// Define once
export const ApiRoute = {
  widgets: {
    GET: (): `${typeof API_BASE}/widgets` => `${API_BASE}/widgets`,
  },
};

// Use typed
fetch(ApiRoute.widgets.GET())  // ✓ Type-checked

// Not allowed
fetch('/api/widgets')  // ✗ Raw string literal banned
```

**Key:** No magic strings. TypeScript enforces API route consistency.

#### 5. **Error Handling Pattern**

```typescript
// Server service returns discriminated union (never throws)
type Result =
  | { ok: true; widgets: WidgetRow[] }
  | { ok: false; status: number; message: string };

// Hook pattern-matches on result.error (no try/catch needed)
const refresh = useCallback(async () => {
  const result = await fetchWidgets();
  if (result.error) {
    setError(result.error);
  } else {
    setWidgets(result.widgets);
  }
}, []);

// Component handles error state explicitly
if (status === 'error') return <div>Error: {error} <button>Retry</button></div>;
```

**Key:** No exceptions in async flows. Discriminated unions make error handling linear.

#### 6. **Server-Side Rendering (RSC) + Client Hydration**

```typescript
// page.tsx (RSC) — fetches server-side
export default async function WidgetsPage() {
  const session = await getSession();
  const result = isSessionValid(session)
    ? await fetchWidgetsFromGateway(session.accessToken)
    : null;

  const initialData = result?.ok ? result.widgets : [];
  const initialError = result && !result.ok ? result.message : null;

  // Pass to client shell with pre-fetched data
  return <WidgetsTabShell initialWidgets={initialData} initialError={initialError} />;
}

// Tab shell mounts provider with initial data (zero loading flash)
export function WidgetsTabShell({ initialWidgets, initialError }) {
  return (
    <WidgetsProvider initialData={initialWidgets} initialError={initialError}>
      <WidgetsTab />
    </WidgetsProvider>
  );
}
```

**Key:** HTML delivered to browser already has real content. No loading skeleton on first paint.

---

## Part 3: Backend Conventions & Enforcement

### Automated Enforcement via AST Rules

Kora uses `gocritic/ruleguard` to enforce conventions. Violations fail `make check`. Examples:

| Rule | Bans | Why |
|------|------|-----|
| `noRawJsonDecodeRequest` | `json.NewDecoder(r.Body).Decode()` in handlers | Use `api.DecodeJSON(ctx, r, w, logger, &req)` for consistency |
| `noRawHttpClientLiteral` | `http.Client{...}` in feature code | Use typed client in `internal/platform/clients/{service}/` |
| `spanSetStatusLocalErr` | `span.SetStatus(codes.Error, err.Error())` | Use `observability.RecordFailure(span, err, sentinel)` |
| `noFmtErrorfStringErr` | `fmt.Errorf("...: %s", err)` | Use `fmt.Errorf("...: %w", err)` for error wrapping |
| `noHttpErrorInHandler` | `http.Error(w, ...)` in handlers | Use `api.ErrorResponse(...)` |

**Adoption for Factory:**

Create `.golangci.yml` rules enforcing our conventions:

```yaml
# State machine violations
rules:
  - name: stateTransitionValidation
    pattern: |
      $M.Transition(&$V, $S)
      where $M.type ~ "Machine" && m["Transition"].type ~ "func.*error"
    message: "Must check Transition() error; use if err := m.Transition(...) { return err }"

  - name: rawStringStateComparison
    pattern: |
      if $V == "pending" || $V == "running" || ...
    message: "Use type-safe state constants; don't compare raw strings"

  - name: unregisteredStateMachine
    pattern: |
      NewMachine(...)  
      where not_found_in_registry()
    message: "All state machines must be registered in internal/stateregistry"
```

### Key Backend Patterns to Adopt

#### 1. **No ORMs, Explicit SQL with pgx**

```go
// ❌ Kora doesn't do this
import "gorm.io/gorm"
db.Where("status = ?", "active").Find(&widgets)

// ✅ Kora does this
rows, err := db.Query(ctx, 
  "SELECT id, name, status FROM widgets WHERE status = $1", 
  "active")
if err != nil {
  return nil, fmt.Errorf("query widgets: %w", err)
}
```

**Why:** Control. Explicit SQL shows exactly what database sees. Easy to optimize. No magic behavior.

#### 2. **Structured Error Wrapping**

```go
// ❌ Don't do this
return fmt.Errorf("failed: %s", err.Error())

// ✅ Do this
return fmt.Errorf("operation: %w", err)
```

**Why:** `%w` preserves the error chain. Tools like `errors.Is()` work. `%s` loses the chain.

#### 3. **Service-to-Service Clients**

```go
// ❌ Don't do this
client := &http.Client{}
resp, err := client.Get("http://...")

// ✅ Do this
// internal/platform/clients/inference/client.go
type Client struct {
  base *httpclient.Base
}

func (c *Client) InferenceRequest(ctx context.Context, req *InferenceRequest) (*Response, error) {
  // Tracing, retries, timeout, logging all built-in
  return c.base.Post(ctx, "/v1/inference", req)
}
```

**Why:** Tracing, retries, timeout, logging centralized. Used everywhere. Consistent telemetry.

#### 4. **Configuration Strategy**

```go
// ❌ Don't do this
type Config struct {
  APIKey string `yaml:"api_key"`  // Secrets in YAML!
}

// ✅ Do this
type Config struct {
  APIKey string `yaml:"-"`  // Secrets never in YAML
}

func NewConfig() Config {
  return Config{
    APIKey: os.Getenv("API_KEY"),  // Load from environment only
  }
}
```

**Why:** Secrets never committed. No accidental leaks in git history.

#### 5. **Logging Convention**

```go
// ❌ Don't do this
log.Println("User created")
fmt.Printf("Status: %s\n", status)

// ✅ Do this
logger.Info(ctx, "user created", logging.Field("user_id", userID))
logger.Error(ctx, "execution failed", logging.Field("error", err))
```

**Why:** Structured logs. Query-able. Full context (trace ID, session, etc.) injected automatically.

---

## Part 4: Development Workflow

### PRD → ADR → Implementation

Kora requires this process for significant features:

1. **PRD** (Product Requirements Document)
   - Problem statement
   - Solution overview
   - Success criteria
   - File: `docs/prd/{feature}.md`

2. **ADR** (Architecture Decision Record)
   - Context: Why are we solving this?
   - Options: What approaches did we consider?
   - Decision: Which did we choose and why?
   - Consequences: What changes as a result?
   - File: `docs/adr/{date}-{feature}.md`

3. **Implementation**
   - Code references ADR in commits
   - PR references ADR (not design discussions scattered in PR comments)
   - Decision rationale stays in docs, not lost in git history

**For Our Factory:**

Apply this to major features:
- Multi-agent coordination flow
- State machine registration & validation
- Agent scheduling & DAG execution improvements
- Policy enforcement system

---

## Part 5: What We Should NOT Copy

### 1. **Kubernetes-Native Architecture**

Kora runs on Kubernetes with SPIFFE, Istio, Envoy, etc. We're a monolithic Go app. Skip this unless we're multi-tenant at scale.

### 2. **Microservices Complexity**

Kora has 20+ services (identity-proxy, policy-gateway, egress-gateway, etc.). We're not there yet. Keep the Factory as a single binary with clean internal layers.

### 3. **Full Policy-As-Code**

Kora uses OPA (Open Policy Agent) for every decision. Too much for us at this stage. Start simpler:
- Role-based access (admin, operator, viewer)
- Tool-level capabilities (which tools each agent can call)
- Escalate to OPA later if needed

---

## Part 6: Recommended Adoption Plan

### Phase 1: Frontend (2-3 weeks) — HIGH PRIORITY

**Goal:** Adopt Kora's state architecture for Agent Orchestrator

**Deliverables:**
1. Refactor AgentOrchestrator to use hook + context + RSC pattern
   - `_types/index.ts` - Agent types
   - `_services/agents-server.ts` - Server-side fetch
   - `_services/agents-api.ts` - Client service
   - `_hooks/use-agents.ts` - Hook with WidgetsStateShape equivalent
   - `_context/agents-context.tsx` - Provider
   - `_components/agents-tab-shell.tsx` - RSC boundary
   - `app/api/agents/route.ts` - BFF proxy

2. Implement route registry pattern
   - `src/lib/routing/api-routes.ts` - Centralize all API URLs

3. Add `loading.tsx` for Suspense skeleton

**Benefits:**
- Zero loading flashes
- Type-safe API contracts
- Cleaner component logic
- State management future-proof

### Phase 2: Backend Conventions (2-3 weeks)

**Goal:** Add enforcement rules and enforce patterns

**Deliverables:**
1. Extend `.golangci.yml` with state machine rules
2. Create `internal/platform/clients/` structure for service-to-service calls
3. Document configuration strategy (secrets in environment only)
4. Add centralized logging setup (`internal/platform/logging/`)

**Benefits:**
- Catch bugs at lint time
- Consistent error handling
- Centralized observability

### Phase 3: Governance Model (3-4 weeks)

**Goal:** Add agent identity & policy enforcement

**Deliverables:**
1. `internal/governance/identity.go` - Agent identity registry
2. `internal/governance/policy.go` - Policy evaluation
3. `internal/governance/credential_broker.go` - Credential injection
4. Update state machines to include governance states

**Benefits:**
- Auditability (every action traces to human)
- Security (agents never hold credentials)
- Accountability (policy enforcement before execution)

### Phase 4: Development Workflow (Ongoing)

**Goal:** Adopt PRD → ADR → implementation process

**Deliverables:**
1. Create `docs/prd/template.md`
2. Create `docs/adr/template.md`
3. Document all major architectural decisions going forward
4. Reference ADRs in commit messages and PRs

**Benefits:**
- Decision rationale captured
- New team members understand "why"
- Reversible decisions (ADRs serve as reference)

---

## Specific Code Patterns to Steal

### 1. API Mux Pattern

Instead of using gin/chi, Kora defines a simple interface:

```go
type Controller interface {
  GroupedBy() string               // /v1/agents/...
  Middleware() []MiddlewareFunc
  Boot(mux *Mux)                   // Register routes
}

type Mux struct {
  // internal implementation
}

// Usage
mux.Handle("GET", "/agents/{id}", handler)

// Automatic: validation, error response format, logging, tracing
```

**Why:** Minimal dependencies. Full control. Consistent error responses everywhere.

### 2. Declarative Error Handling

```go
var (
  ErrUnauthorized    = errors.New("unauthorized")
  ErrNotFound        = errors.New("not found")
  ErrInvalidInput    = errors.New("invalid input")
)

// In handler
if !isAuthorized(ctx, req) {
  return api.ErrorResponse(w, http.StatusForbidden, ErrUnauthorized)
}
```

**Why:** Error semantics are explicit. Mapping to HTTP status is centralized.

### 3. Configuration via Environment + YAML

```go
type Config struct {
  Port     int       `yaml:"port"`       // From YAML
  APIKey   string    `yaml:"-"`          // From env only
}

func LoadConfig() Config {
  c := loadYAML("config/base.yaml")
  if os.Getenv("API_KEY") != "" {
    c.APIKey = os.Getenv("API_KEY")
  }
  return c
}
```

**Why:** Secrets never in config. Defaults in YAML. Environment overrides. Clear intent.

---

## Adoption Checklist

- [ ] **Frontend:** Refactor Agent Orchestrator to use RSC + hook + context pattern
- [ ] **Frontend:** Implement route registry for API URLs
- [ ] **Backend:** Extend `.golangci.yml` with custom rules
- [ ] **Backend:** Create `internal/platform/clients/` for service clients
- [ ] **Backend:** Centralize logging setup
- [ ] **Governance:** Implement agent identity registry
- [ ] **Governance:** Implement policy enforcement
- [ ] **Docs:** Create PRD/ADR templates
- [ ] **Docs:** Document state machine conventions
- [ ] **Testing:** Automated verification of conventions via AST rules

---

## References

- Kora AGENTS.md: https://github.com/runkora/kora/blob/main/AGENTS.md
- Kora frontend state pattern: `/tmp/kora/frontend/docs/golden-paths/frontend-state-pattern.md`
- Kora architecture: https://github.com/runkora/kora/blob/main/ARCHITECTURE.md
- Kora ADRs: https://github.com/runkora/kora/tree/main/docs/adr

---

## Next Steps

1. **Review** this document with the team
2. **Pick Phase 1** (frontend) as first sprint
3. **Create ADR** for each major change
4. **Gradual adoption** - don't rewrite everything at once
5. **Document** as we go (PRDs/ADRs)

**Key Principle:** Steal what works (frontend architecture, backend conventions, enforcement via linting), adapt to our scale (don't replicate full Kubernetes setup), and gradually build governance on top.
