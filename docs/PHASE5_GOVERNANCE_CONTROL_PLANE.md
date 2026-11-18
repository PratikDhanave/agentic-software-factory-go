# Phase 5: Governance & Control Plane Design
## Enterprise Safety, Compliance & Kubernetes Deployment

---

## Overview

**Phase 5 Goal**: Build the Kora-inspired control plane that governs, audits, and securely deploys agent workflows at scale

**Scope**:
- Identity layer (agent identity, user identity, service accounts)
- Policy engine (fine-grained access control)
- Credential management (secure storage, JIT injection)
- Audit trails (complete action history)
- Kubernetes deployment
- Rate limiting and resource quotas

**Integration Points**:
- Input: Approved workflows from Phase 4
- Output: Secure execution via Phase 1-3 DAG Engine
- Observability: All actions logged and audited
- Persistence: PostgreSQL for compliance records

---

## Architecture: 5 Core Components

```
┌─────────────────────────────────────────────────────────┐
│          Kora Control Plane (Phase 5)                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 1. IDENTITY LAYER                              │   │
│  │  - Agent identity & credentials                │   │
│  │  - User identity & authentication              │   │
│  │  - Service accounts for automation             │   │
│  │  - Permission mapping                          │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 2. POLICY ENGINE                               │   │
│  │  - Resource access rules (RBAC)                │   │
│  │  - Action permissions                          │   │
│  │  - Rate limiting & quotas                      │   │
│  │  - Compliance gates                            │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 3. CREDENTIAL MANAGER                          │   │
│  │  - Secure storage (encrypted at rest)          │   │
│  │  - Just-in-time injection                      │   │
│  │  - Scope limiting                              │   │
│  │  - Rotation & expiry                           │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 4. AUDIT TRAIL                                 │   │
│  │  - All agent actions logged                    │   │
│  │  - All user interactions tracked               │   │
│  │  - Immutable audit log                         │   │
│  │  - Compliance reports                          │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 5. DEPLOYMENT & RESOURCE MANAGEMENT            │   │
│  │  - Kubernetes manifests                        │   │
│  │  - Resource quotas                             │   │
│  │  - Pod security policies                       │   │
│  │  - Network policies                            │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
└─────────────────────────────────────────────────────────┘
              ↓
    Phase 1-3 DAG Engine (with security context)
```

---

## Component 1: Identity Layer

### Identity Model

```go
type Principal struct {
    ID          string        // UUID
    Type        PrincipalType // agent | user | service
    Name        string
    DisplayName string
    Credentials map[string]interface{}
    Permissions []Permission
    Active      bool
    CreatedAt   time.Time
}

type PrincipalType string
const (
    PrincipalAgent   PrincipalType = "agent"
    PrincipalUser    PrincipalType = "user"
    PrincipalService PrincipalType = "service"
)

type Permission struct {
    Resource   string    // workflow | execution | credential | audit
    Action     string    // create | read | update | delete | execute
    Scope      string    // * (all), project:X, team:Y
    Conditions []Condition // time-based, rate-based, etc
}
```

### Authentication

```
Agent Workflow Trigger
    ↓
Identify Principal (agent/user)
    ↓
Verify Credentials (API key, token, JWT)
    ↓
Create Security Context with:
    - Principal ID
    - Traced Permissions
    - Audit ID
    - Start time
```

---

## Component 2: Policy Engine

### RBAC Model

```
Policy Rules:
  Rule 1: developers can execute workflows tagged 'feature'
  Rule 2: security-team can override decisions
  Rule 3: agents cannot access production without approval
  Rule 4: rate limit: 100 workflows/hour per user
  Rule 5: compliance: all prod workflows logged to immutable store
```

### Enforcement

```go
type PolicyEngine interface {
    Evaluate(ctx SecurityContext, action Action) (Decision, error)
    // Decision: Allow | Deny | RequireApproval
}

type Action struct {
    Resource    string            // workflow_id, credential_id
    Operation   string            // create, execute, read
    Attributes  map[string]string // tags, tier, region
}

type Decision struct {
    Result            string        // allow | deny | require_approval
    Reason            string        // why this decision
    RequiredApprovers []string      // if require_approval
    CacheExpiry       time.Duration // how long to cache
}
```

---

## Component 3: Credential Manager

### Secure Credential Storage

```go
type Credential struct {
    ID          string
    Owner       string        // agent_id or user_id
    Type        string        // github_token, aws_key, etc
    Name        string
    Value       string        // encrypted
    Scopes      []string      // what can use this credential
    ExpiresAt   *time.Time
    LastUsedAt  *time.Time
    AuditID     string        // tied to audit trail
}

type CredentialRequest struct {
    CredentialID string        // what credential
    Purpose      string        // why needed (logged)
    Duration     time.Duration // max 1 hour
    Scope        []string      // limit access
}

type InjectedCredential struct {
    Value     string
    ExpiresAt time.Time
    RequestID string // for audit trail
}
```

### Just-in-Time Flow

```
Agent needs credential:
    ↓
Request via CredentialRequest (purpose, duration, scope)
    ↓
PolicyEngine evaluates (allowed? audit?)
    ↓
Retrieve from encrypted store
    ↓
Inject into execution context (not to disk)
    ↓
Auto-revoke after expiry or completion
    ↓
Log access to audit trail
```

---

## Component 4: Audit Trail

### Audit Log Model

```go
type AuditEntry struct {
    ID              string
    Timestamp       time.Time
    Principal       string        // who (agent/user)
    Action          string        // what (execute, read, etc)
    Resource        string        // which resource
    Status          string        // success | failure | denied
    Details         map[string]interface{} // context
    IPAddress       string
    UserAgent       string
    ChangesFrom     interface{}   // before
    ChangesTo       interface{}   // after
    Error           string        // if failed
    CorrelationID   string        // trace across calls
    Hash            string        // for integrity
    PreviousHash    string        // linked list chain
}
```

### Immutability

```
Audit entries:
  1. Written sequentially to PostgreSQL
  2. Each entry includes hash of previous entry
  3. Forms immutable linked list
  4. Cryptographic verification possible
  5. Tamper detection via hash chain break
```

### Compliance Reports

```go
type ComplianceReport struct {
    Period     DateRange
    Principal  string
    Actions    []AuditEntry
    Summary    map[string]int    // action counts
    Violations []PolicyViolation // denied actions
    RiskScore  float64           // 0-100
}
```

---

## Component 5: Kubernetes Deployment

### Deployment Model

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: factory-stack

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: control-plane
  namespace: factory-stack
spec:
  replicas: 3
  selector:
    matchLabels:
      app: factory-control-plane
  template:
    metadata:
      labels:
        app: factory-control-plane
    spec:
      serviceAccountName: factory-control-plane
      containers:
      - name: control-plane
        image: factory:control-plane:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: factory-secrets
              key: database-url
        - name: AUDIT_LOG_DESTINATION
          value: s3://factory-audit-logs/

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: factory-control-plane
  namespace: factory-stack

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: factory-control-plane
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["create", "delete", "get", "list"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
```

### Resource Quotas

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: factory-quota
  namespace: factory-stack
spec:
  hard:
    requests.cpu: "100"
    requests.memory: "200Gi"
    pods: "1000"
  scopeSelector:
    matchExpressions:
    - operator: In
      scopeName: PriorityClass
      values: ["high", "medium"]
```

---

## Security Best Practices

### Secrets Management

```
✅ All credentials encrypted at rest (AES-256)
✅ Encrypted in transit (TLS 1.3+)
✅ Never logged to files or stdout
✅ Injected as environment/volume mount only
✅ Automatic rotation policies
✅ Access requires permission + audit
```

### Network Security

```
✅ All traffic encrypted (mTLS)
✅ Network policies: agents can't reach each other
✅ No direct credential access from agents
✅ All requests proxied through control plane
✅ IP-based rate limiting
✅ DDoS protection via API gateway
```

### Audit & Compliance

```
✅ Every action logged (success + failure)
✅ Immutable audit trail (hash chain)
✅ Credential access logged with purpose
✅ Policy violations logged + alerted
✅ Compliance reports generated automatically
✅ SOC2/HIPAA/PCI-DSS ready
```

---

## Implementation Roadmap: Phase 5

### Phase 5.1: Identity & Authentication (Weeks 1-2)
- [ ] Principal model (agent, user, service)
- [ ] Permission system
- [ ] API key management
- [ ] JWT token validation

### Phase 5.2: Policy Engine (Weeks 3-4)
- [ ] RBAC policy model
- [ ] Policy evaluation engine
- [ ] Rate limiting
- [ ] Approval workflows

### Phase 5.3: Credential Manager (Weeks 5-6)
- [ ] Encrypted credential storage
- [ ] JIT credential injection
- [ ] Scope limiting
- [ ] Automatic rotation

### Phase 5.4: Audit Trail (Weeks 7-8)
- [ ] Audit log model
- [ ] Immutable storage
- [ ] Hash chain integrity
- [ ] Compliance reporting

### Phase 5.5: Kubernetes & Deployment (Weeks 9-10)
- [ ] K8s manifests
- [ ] Resource quotas
- [ ] Network policies
- [ ] Pod security policies

---

## Example: Complete Secure Workflow

```
1. GitHub Issue Created
   "Fix auth bug" (Issue #42)
   
2. Kora Identity Check
   ✓ User: developer@acme.com
   ✓ Token valid
   ✓ Permissions: can_trigger_workflows
   ✓ Audit ID: audit_123
   
3. Policy Evaluation
   ✓ Org policy: "feature" tag workflows allowed
   ✓ Rate limit: 50/hour remaining
   ✓ No approval required
   
4. Phase 4 Coordination
   ✓ Agents decide: 10 steps, 95% confidence
   ✓ No human approval needed
   
5. Credential Injection
   ✓ Request: GitHub token for PR creation
   ✓ Check: developer has access
   ✓ Inject: token into execution context
   ✓ Scope: github.com/myorg only
   ✓ Duration: 30 minutes
   
6. Phase 1-3 Execution
   ✓ Security context passed through DAG
   ✓ All job outputs audited
   
7. Completion & Audit
   ✓ Action: 10 jobs executed successfully
   ✓ User: developer@acme.com
   ✓ Agent: developer-agent (id: agent_42)
   ✓ Resources: PR created #123
   ✓ Time: 125ms
   ✓ Audit: entry_audit_123_final
   
8. Compliance
   ✓ All logged to immutable audit trail
   ✓ Ready for SOC2 audit
```

---

## Success Criteria

**Security**
- ✅ All credentials encrypted at rest and in transit
- ✅ Zero credential exposure to agents
- ✅ Fine-grained access control (RBAC)
- ✅ All actions audited with full context

**Governance**
- ✅ Immutable audit trail with integrity checks
- ✅ Automated compliance reporting
- ✅ Policy enforcement across all operations
- ✅ Rate limiting and resource quotas

**Operations**
- ✅ Kubernetes-native deployment
- ✅ High availability (3+ replicas)
- ✅ Automatic failover
- ✅ Horizontal scalability

**Compliance**
- ✅ SOC2 Type II ready
- ✅ HIPAA compliance possible
- ✅ PCI-DSS for financial workflows
- ✅ GDPR data deletion support

---

## Integration with Full Stack

```
Complete Factory Stack:

Layer 5: Governance (Phase 5) ← You are here
├─ Identity, Policy, Credentials
├─ Audit & Compliance
└─ K8s Deployment

Layer 4: Coordination (Phase 4) ✅
├─ 10 Agents, Knowledge Graph
└─ Interactive Approvals

Layer 3: Execution (Phase 1-3) ✅
├─ DAG Engine, 30-50% speedup
└─ Automatic Parallelism

Layer 2: Integration
├─ GitHub Webhooks
├─ Event Routing
└─ Agent Adapters

Layer 1: Observability
├─ Logging, Tracing
├─ Metrics, Monitoring
└─ Performance Analytics
```

---

## Vision: Enterprise-Ready Software Factory

A complete, open-source, Kubernetes-native system for:

1. **Secure Execution** — All credentials encrypted, JIT injection
2. **Fine-grained Control** — RBAC with approval workflows
3. **Complete Audit** — Immutable trails for compliance
4. **Intelligent Decisions** — Multi-agent coordination + learning
5. **Efficient Execution** — DAG parallelism, 30-50% faster
6. **Enterprise Scale** — HA, horizontal scaling, SOC2/HIPAA ready

**Result**: The most secure, compliant, intelligent software factory platform.

EOF
cat /Users/pratik.dhanave/go/src/github.com/PratikDhanave/agentic-software-factory-go/docs/PHASE5_GOVERNANCE_CONTROL_PLANE.md | wc -l
