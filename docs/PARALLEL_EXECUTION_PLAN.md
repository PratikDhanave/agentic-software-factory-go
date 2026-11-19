# Parallel Execution Plan: Phase 3 Deployment + Phase 4 Implementation

**Goal:** Maximize progress by running both phases in parallel  
**Duration:** 2-4 weeks (estimated)  
**Success:** Phase 3 deployed to Kubernetes + Phase 4 at 100%

---

## Work Streams

### STREAM A: Phase 3 Deployment
**Prerequisites:** Docker daemon running  
**Owner:** DevOps / Infrastructure  
**Duration:** 1-2 weeks after Docker available

#### Week 1: Local Testing & Validation
```bash
# Create local kind cluster
make kind-up

# Deploy with Helm
make helm-deploy-kind

# Verify pods are healthy
kubectl get pods -n factory-system
kubectl logs -n factory-system -l app.kubernetes.io/name=agentic-factory

# Test service accessibility
kubectl port-forward -n factory-system svc/agentic-factory 8080:8080
curl http://localhost:8080/health
```

**Checklist:**
- [ ] kind cluster creates successfully
- [ ] Helm deployment succeeds
- [ ] Pods reach "Ready" state
- [ ] Service responds to HTTP requests
- [ ] Logs show no errors

#### Week 2: GitOps Setup & Validation
```bash
# Install ArgoCD in kind cluster
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Apply our applications
kubectl apply -f argocd/applications/dev.yaml

# Watch ArgoCD sync
kubectl get application -n argocd -w

# Verify GitOps works (make a change to Git, see it auto-deploy)
```

**Checklist:**
- [ ] ArgoCD installs successfully
- [ ] Applications sync from Git
- [ ] Auto-sync works (dev app)
- [ ] Manual-sync works (prod app simulation)
- [ ] Git changes trigger deployment

#### Deliverables
- ✅ Local Kubernetes cluster (kind) running
- ✅ Factory deployed via Helm
- ✅ ArgoCD managing deployments
- ✅ GitOps workflow validated
- ✅ Deployment guide documented

---

### STREAM B: Phase 4 Completion (60% → 100%)
**Current State:** Core logic complete (tenants + discovery + MCP)  
**Owner:** Backend / Infrastructure  
**Duration:** 2-3 weeks (parallel to Phase 3)

#### Week 1: Kubernetes Integration

**Task B1.1: NetworkPolicy for Tenant Isolation**
```yaml
# File: config/base/networkpolicy.yaml
# Tenant-to-tenant isolation using NetworkPolicy
# - Inbound: Only same namespace + factory-system
# - Outbound: Same namespace + factory-system + DNS
```

```go
// File: internal/platform/tenants/networkpolicy.go
// Generate NetworkPolicy for each tenant namespace
func (m *Manager) GenerateNetworkPolicy(tenantID string) *networkingv1.NetworkPolicy
```

**Task B1.2: ResourceQuota per Tenant**
```yaml
# File: config/base/resourcequota.yaml
# Per-tenant limits:
# - CPU: 10 cores max
# - Memory: 20Gi max
# - Pods: 100 max
```

```go
// File: internal/platform/tenants/quota.go
// Apply ResourceQuota when tenant is created
func (m *Manager) ApplyQuota(ctx context.Context, tenantID string) error
```

**Task B1.3: Namespace Isolation Enforcement**
```go
// File: internal/platform/tenants/isolation.go
// Kubernetes integration
// - Create namespace on tenant registration
// - Apply RBAC per tenant
// - Enforce resource limits
```

**Tests:** 8 new tests (Kubernetes integration)

#### Week 2: Credential Exchange & Boundaries

**Task B2.1: Credential Refresh at Tenant Boundaries**
```go
// File: internal/platform/credentials/boundary.go
// When agent calls service in another tenant:
// 1. Issue short-lived credential (5min TTL)
// 2. Exchange capability for specific action
// 3. Verify credential at service boundary
```

**Task B2.2: MCP Server Registration**
```go
// File: internal/platform/mcp/registration.go
// When agent starts, register with discovery:
// 1. Register MCP capabilities
// 2. Set health to "healthy"
// 3. Register with service discovery
// 4. Subscribe to health checks
```

**Task B2.3: Cross-Tenant Service Calls**
```go
// File: internal/platform/mcp/cross_tenant.go
// MCPClient enhancement:
// 1. DiscoverService (cross-tenant, with policy check)
// 2. CallService (exchanges credentials at boundary)
// 3. HandleResponse (verify service identity)
```

**Tests:** 12 new tests (credential exchange, cross-tenant calls)

#### Week 3: Integration & E2E Testing

**Task B3.1: Multi-Tenant Integration Tests**
```bash
# Scenario 1: Register 3 tenants
# Scenario 2: Each tenant creates agents
# Scenario 3: Agent A calls service in Tenant B (verify policy)
# Scenario 4: Verify Tenant C can't see B's services
# Scenario 5: Verify isolation with NetworkPolicy
```

**Task B3.2: Kubernetes Multi-Tenant Deployment**
```yaml
# Deploy 3 isolated tenants to kind cluster
# Each with:
# - Separate namespace (factory-tenant-{name})
# - ResourceQuota enforcement
# - NetworkPolicy isolation
# - Agents + services
```

**Tests:** 6 E2E tests

#### Deliverables
- ✅ NetworkPolicy integration (tenant isolation)
- ✅ ResourceQuota enforcement (per-tenant limits)
- ✅ Namespace isolation (RBAC + network)
- ✅ Credential exchange (boundary verification)
- ✅ MCP service registration (auto-discovery)
- ✅ Cross-tenant service calls (with policy)
- ✅ 26 new tests (all passing)
- ✅ Multi-tenant E2E scenarios

---

## Parallel Timeline

```
Week 1
├─ Stream A: kind cluster setup + Helm deployment
└─ Stream B: NetworkPolicy + ResourceQuota implementation

Week 2
├─ Stream A: ArgoCD setup + GitOps validation
└─ Stream B: Credential exchange + MCP registration

Week 3
├─ Stream A: Production readiness testing
└─ Stream B: Integration tests + E2E scenarios

Week 4
├─ Stream A: Deploy to production environment
└─ Stream B: Final validation + polish
```

---

## Success Criteria

### Phase 3: Deployment ✅
- [ ] kind cluster works locally
- [ ] Helm chart deploys without errors
- [ ] Pods are healthy (ready/live probes)
- [ ] Service is accessible
- [ ] ArgoCD syncs from Git
- [ ] GitOps workflow validated

### Phase 4: Multi-Tenancy ✅
- [ ] NetworkPolicy isolates tenants
- [ ] ResourceQuota enforces limits
- [ ] 3+ tenants can coexist
- [ ] Service discovery works cross-tenant
- [ ] Credentials exchanged at boundaries
- [ ] MCP capabilities registered automatically
- [ ] Cross-tenant calls follow policy
- [ ] 26 new tests passing
- [ ] E2E scenarios pass

---

## Resource Allocation

**Team Structure:**
- **Stream A Lead:** DevOps engineer (1 person)
  - Kind cluster management
  - Helm deployment
  - ArgoCD setup
  - GitOps validation

- **Stream B Lead:** Backend engineer (1-2 people)
  - Kubernetes integration
  - Credential boundary logic
  - MCP registration
  - Integration testing

**Shared:** Code review, documentation, final integration

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Docker unavailable | Use test environment or cloud K8s |
| NetworkPolicy complexity | Start simple, iterate |
| Credential exchange bugs | Heavy testing, boundary scenarios |
| Integration delays | Daily sync between streams |

---

## Completion Criteria

**Phase 3 Complete When:**
1. Helm chart deployed to local cluster
2. ArgoCD managing deployments
3. Git changes auto-deploy
4. Health checks pass

**Phase 4 Complete When:**
1. 3+ tenants isolated in Kubernetes
2. Cross-tenant service discovery works
3. Credentials exchanged at boundaries
4. 26 tests passing (all scenarios)
5. E2E tests demonstrate full flow

**Both Complete When:**
- Phase 3: Kubernetes deployment working
- Phase 4: Multi-tenant isolation verified
- Combined: 3 isolated tenants running 6+ services with cross-tenant calls

---

## Next Actions

**Immediate:**
1. Create NetworkPolicy implementation task
2. Create ResourceQuota enforcement task
3. Create credential boundary task
4. Create MCP registration task
5. Set up daily sync between teams

**When Docker Available:**
1. Create kind cluster
2. Deploy Phase 3 to cluster
3. Begin Phase 4 Kubernetes integration

**Weekly:**
1. Demo working features
2. Identify blockers
3. Adjust timeline if needed

---

## Expected Outcome

**At End of Parallel Execution:**

```
Kubernetes Cluster (kind)
├── kora-system (control plane)
│   ├── agentic-factory deployment
│   ├── identity service
│   └── observability stack
│
├── factory-tenant-team-a (isolated)
│   ├── agent-analyzer
│   ├── agent-reviewer
│   └── service-discovery
│
├── factory-tenant-team-b (isolated)
│   ├── agent-tester
│   ├── agent-optimizer
│   └── service-discovery
│
└── factory-tenant-team-c (isolated)
    ├── agent-auditor
    └── service-discovery

Cross-Tenant Calls:
├── Team A agent calls Team B service (with credential exchange)
├── Team B agent calls Team C service (with policy verification)
└── All calls logged to audit trail
```

**Production-Ready:**
- Phase 3 ✅ Kubernetes deployment infrastructure
- Phase 4 ✅ Multi-tenant isolation + service discovery
- Combined ✅ Enterprise control plane foundation

---

## References

- **Phase 3 Docs:** docs/PHASE3_DEPLOYMENT.md
- **Phase 4 Docs:** docs/PHASE4_SCALING.md
- **Implementation Guide:** This document
- **Code Base:** All Phase 1-4 code committed and tested

