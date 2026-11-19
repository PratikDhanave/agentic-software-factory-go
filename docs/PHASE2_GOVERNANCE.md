# Phase 2: Governance Layer (Months 4-6)

**Goal:** Agent identity + credential model + policy evaluation  
**Status:** Starting  
**Completion Date:** 2026-12-17 (target)

---

## Overview

Phase 2 builds the **governance foundation** for secure multi-tenant agent operation. We introduce:

1. **Agent Identity System** — Each agent has a cryptographically-signed identity
2. **Capability-Based Credentials** — Agents operate under explicit capability grants
3. **Policy Engine** — Simple rule-based policy evaluation (not OPA yet)
4. **Credential Lifecycle** — Issue, verify, revoke credentials
5. **State Machine Validation** — Only valid state transitions allowed by policy

This phase is **simpler than Kora/AgentMesh** but establishes the patterns for scaling to them later.

---

## Architecture

```
Agent Created
    ↓
Identity System issues agent ID + signing key
    ↓
Credential Broker issues capability credential
    ↓
Agent signs requests with key
    ↓
Policy Engine evaluates:
  - Is identity enrolled?
  - Are capabilities in scope?
  - Is action allowed?
    ↓
✓ Allow → Log audit event → Execute
✗ Deny → Log audit event → Return 403
```

---

## Implementation Plan

### Part 1: Agent Identity (Week 1)

**internal/platform/identity/identity.go:**
```go
type AgentIdentity struct {
  ID          string    // Unique agent ID (UUID)
  Type        string    // "code-reviewer", "bug-finder", etc.
  PublicKey   string    // For signature verification
  CreatedAt   time.Time // Timestamp
  CreatedBy   string    // Which system/user created this
}

type SigningKey struct {
  AgentID    string
  PrivateKey string  // Stored securely
  PublicKey  string
}
```

**internal/platform/identity/registry.go:**
- `Register(ctx, identity)` — Register new agent + generate signing key
- `Get(ctx, agentID)` — Look up agent by ID
- `Verify(ctx, agentID, signature)` — Verify signature with agent's public key

**Tests:**
- Agent registration flow
- Signature verification
- Key generation security
- Duplicate prevention

### Part 2: Credential System (Week 1-2)

**internal/platform/credentials/credential.go:**
```go
type Credential struct {
  ID         string    // Unique credential ID
  AgentID    string    // Which agent holds this
  Subject    string    // "code:analyze", "tool:invoke", "workflow:execute"
  Actions    []string  // ["read", "write"] for subject
  ExpiresAt  time.Time // Short-lived (hours to days)
  IssuedAt   time.Time
  Signature  string    // Signed by identity system
  Metadata   map[string]string
}

type CredentialRequest struct {
  AgentID    string
  Subject    string
  Actions    []string
  Duration   time.Duration
}
```

**internal/platform/credentials/broker.go:**
- `Issue(ctx, request)` — Create new credential
- `Verify(ctx, credential)` — Check signature and expiry
- `Revoke(ctx, credentialID)` — Invalidate immediately
- `Refresh(ctx, credentialID)` — Extend expiry

**internal/platform/credentials/store.go:**
- In-memory or SQLite-backed credential store
- Track issued/revoked credentials
- Cleanup expired credentials

**Tests:**
- Issue/verify flow
- Expiry enforcement
- Revocation flow
- Credential reuse detection

### Part 3: Policy Engine (Week 2)

**internal/platform/policy/policy.go:**
```go
type Policy struct {
  ID          string
  Name        string
  Rules       []Rule
  Description string
  CreatedAt   time.Time
}

type Rule struct {
  ID         string
  Subject    string    // "code:analyze", "*"
  Actions    []string  // ["read", "write"] or ["*"]
  Effect     string    // "allow" or "deny"
  Conditions []Condition
}

type Condition struct {
  Field    string  // "agent.type", "time.hour"
  Operator string  // "equals", "in", "gt", "lt"
  Value    interface{}
}
```

**internal/platform/policy/engine.go:**
```go
type Engine struct {
  policies []*Policy
}

func (e *Engine) Evaluate(ctx context.Context, req *EvaluationRequest) *EvaluationResult {
  // 1. Check if agent is enrolled
  // 2. Check if credential is valid
  // 3. Evaluate policy rules
  // 4. Return allow/deny + reason
}

type EvaluationRequest struct {
  AgentID     string
  Credential  *Credential
  Subject     string
  Action      string
  Context     map[string]interface{}
}

type EvaluationResult struct {
  Allowed bool
  Reason  string  // "policy matched", "credential expired", etc.
}
```

**Default policies:**
- Allow agents created by system to access their own operations
- Deny everything else by default
- Allow admin override

**Tests:**
- Simple allow/deny rules
- Wildcard matching
- Condition evaluation
- Default deny behavior

### Part 4: Integration with State Machines (Week 2-3)

**Update state transitions to check policy:**

```go
func (m *Machine) Transition(ctx context.Context, to string, cred *Credential) error {
  // 1. Check state machine validity
  if !m.allowed(m.current, to) {
    return fmt.Errorf("invalid transition: %s -> %s", m.current, to)
  }

  // 2. Check policy
  result := policyEngine.Evaluate(ctx, &EvaluationRequest{
    AgentID:    agentID,
    Credential: cred,
    Subject:    fmt.Sprintf("state:%s", to),
    Action:     "transition",
  })
  if !result.Allowed {
    audit.LogError(ctx, agentID, fmt.Sprintf("policy denied transition: %s", result.Reason), nil)
    return fmt.Errorf("policy denied: %s", result.Reason)
  }

  // 3. Execute transition
  m.current = to
  audit.LogTransition(ctx, agentID, from, to, "policy allowed")
  return nil
}
```

**Tests:**
- Policy-enforced transitions
- Audit logging of all transitions
- Credential expiry blocking transitions
- Admin bypass

### Part 5: Audit Trail Integration (Week 3)

**Update audit logging to include policy decisions:**

```go
func LogPolicyDecision(ctx context.Context, agentID string, decision *EvaluationResult) {
  audit.Log(ctx, audit.PolicyEvaluated, 
    fmt.Sprintf("policy decision: %v", decision.Allowed),
    agentID,
    map[string]interface{}{
      "allowed": decision.Allowed,
      "reason":  decision.Reason,
      "subject": decision.Subject,
      "action":  decision.Action,
    })
}
```

**Tests:**
- Audit events for credential issuance
- Audit events for policy decisions
- Query audit trail by agent/time
- Compliance exports

---

## Database Schema

```sql
-- Agent identities
CREATE TABLE agent_identities (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  public_key TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_by TEXT,
  metadata TEXT,  -- JSON
  UNIQUE(id)
);

-- Credentials
CREATE TABLE credentials (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  subject TEXT NOT NULL,
  actions TEXT NOT NULL,  -- JSON array
  issued_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  revoked_at DATETIME,
  signature TEXT NOT NULL,
  metadata TEXT,  -- JSON
  FOREIGN KEY (agent_id) REFERENCES agent_identities(id),
  INDEX idx_agent_id (agent_id),
  INDEX idx_expires_at (expires_at),
  INDEX idx_revoked_at (revoked_at)
);

-- Policies
CREATE TABLE policies (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  rules TEXT NOT NULL,  -- JSON array
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  enabled BOOLEAN DEFAULT 1
);

-- Policy evaluation audit log
CREATE TABLE policy_audit_log (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  credential_id TEXT,
  subject TEXT NOT NULL,
  action TEXT NOT NULL,
  allowed BOOLEAN NOT NULL,
  reason TEXT,
  evaluated_at DATETIME NOT NULL,
  FOREIGN KEY (agent_id) REFERENCES agent_identities(id),
  FOREIGN KEY (credential_id) REFERENCES credentials(id),
  INDEX idx_agent_id (agent_id),
  INDEX idx_evaluated_at (evaluated_at),
  INDEX idx_allowed (allowed)
);
```

---

## Code Examples

### Example 1: Register an Agent

```go
ctx := context.Background()

// Create identity
identity, sigKey, err := identitySystem.Register(ctx, &identity.RegistrationRequest{
  Type:      "code-reviewer",
  CreatedBy: "system",
})
if err != nil {
  return audit.LogError(ctx, "", "agent registration failed", err)
}

audit.Log(ctx, audit.AgentCreated, 
  fmt.Sprintf("agent %s created", identity.ID),
  identity.ID,
  map[string]interface{}{
    "type":        identity.Type,
    "public_key":  identity.PublicKey[:16] + "...",
  })
```

### Example 2: Issue Credential

```go
// Agent requests capability
cred, err := credentialBroker.Issue(ctx, &CredentialRequest{
  AgentID:  agentID,
  Subject:  "code:analyze",
  Actions:  []string{"read", "execute"},
  Duration: 24 * time.Hour,
})
if err != nil {
  return audit.LogError(ctx, agentID, "credential issuance failed", err)
}

audit.Log(ctx, audit.CredentialIssued,
  fmt.Sprintf("credential issued for %s", cred.Subject),
  agentID,
  map[string]interface{}{
    "credential_id": cred.ID,
    "subject":       cred.Subject,
    "expires_at":    cred.ExpiresAt,
  })
```

### Example 3: Policy-Protected Transition

```go
// Agent tries to transition state
err := stateMachine.Transition(ctx, agentID, "pending", "running", credential)
if err != nil {
  audit.LogError(ctx, agentID, fmt.Sprintf("state transition denied: %v", err), err)
  return nil, err
}

audit.LogTransition(ctx, agentID, "pending", "running", "credential allowed")
```

---

## Testing Strategy

**Unit Tests (70%):**
- Identity registration and verification
- Credential issuance and verification
- Policy rule evaluation
- Signature validation

**Integration Tests (25%):**
- End-to-end credential flow
- Policy + state machine integration
- Audit log persistence
- Credential expiry enforcement

**E2E Tests (5%):**
- Register agent → issue credential → execute action flow
- Revoke credential → verify denied flow

---

## Success Criteria

✅ Agent identity system with signing keys  
✅ Capability-based credentials issued and verified  
✅ Policy engine evaluates allow/deny  
✅ State transitions enforced by policy  
✅ All decisions logged to audit trail  
✅ Credentials with expiry enforcement  
✅ Database schema for persistence  
✅ Tests pass with race detector  

---

## Next Steps After Phase 2

**Phase 3 (Months 7-9): Deployment**
- ArgoCD integration
- Kustomize overlays
- Multi-environment configs

**Phase 4 (Months 10-12): Scaling**
- Multi-tenancy with namespace isolation
- Service discovery
- MCP as standard interface

---

## References

- **MEGA_DEEPDIVE.md**: Governance patterns from Kora/AgentMesh
- **PHASE1_DEPLOYMENT.md**: Backend patterns baseline
- **internal/platform/**: All platform utilities (logging, errors, audit, clients)

