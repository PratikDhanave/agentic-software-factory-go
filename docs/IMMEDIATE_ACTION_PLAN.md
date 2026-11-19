# Immediate Action Plan - Next 24-48 Hours

**Goal:** Kick off both streams with maximum parallelism  
**Start Date:** 2026-11-19  
**Current State:** Phase 1-4 code complete, tests passing, committed to main

---

## Phase 3: Kubernetes Deployment (STREAM A)

### Status: Ready to Execute (blocked on Docker daemon)
**Current Setup:**
- ✅ Helm chart complete (Chart.yaml, values.yaml, templates)
- ✅ Helm lint validation passing
- ✅ Kustomize overlays validated (base, dev, prod)
- ✅ ArgoCD applications defined
- ✅ Makefile targets created (kind-install, helm-deploy-kind, etc.)

### Tasks (Do First, When Docker Available)

**Task A.1.1: Set Up Local kind Cluster (1 hour)**
```bash
# Prerequisites installed?
which kind docker kubectl helm

# If not:
# - brew install kind
# - brew install docker (or use Docker Desktop)
# - brew install kubectl
# - brew install helm

# Then run:
make kind-up

# Verify:
kubectl get nodes
kubectl cluster-info
```

**Task A.1.2: Deploy Factory to kind (30 min)**
```bash
# Install Helm chart
make helm-deploy-kind

# Verify deployment
kubectl get pods -n factory-system
kubectl get svc -n factory-system

# Check pod status
kubectl logs -n factory-system -l app=agentic-factory

# Port forward to test locally
kubectl port-forward -n factory-system svc/agentic-factory 8080:8080
curl http://localhost:8080/health
```

**Task A.1.3: Validate Health Probes (15 min)**
```bash
# Wait for pods to be ready
kubectl wait --for=condition=Ready pod -l app=agentic-factory -n factory-system --timeout=60s

# Verify liveness probe
kubectl get pods -n factory-system -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")]}'

# Verify readiness probe
kubectl describe pod -n factory-system -l app=agentic-factory
```

---

## Phase 4: Multi-Tenancy Implementation (STREAM B)

### Status: 60% Complete, Ready to Extend
**Current Implementation:**
- ✅ Tenant Manager (register, get, list, delete, quota, policy)
- ✅ Service Discovery Registry (register, discover, health check, watchers)
- ✅ MCP Server & Client (capabilities, service discovery)
- ✅ All 25 unit tests passing

### Tasks (Start Immediately, No Docker Required)

**Task B.1.1: Implement Kubernetes NetworkPolicy (2 hours)**

Create: `internal/platform/tenants/networkpolicy.go`

```go
package tenants

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateNetworkPolicy creates tenant isolation policy
func (m *Manager) GenerateNetworkPolicy(tenantID string) *networkingv1.NetworkPolicy {
	tenant, _ := m.Get(context.Background(), tenantID)
	
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant-isolation-" + tenantID,
			Namespace: tenant.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				// Allow same namespace + control plane
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "factory-system",
								},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				// Allow to same namespace + control plane + DNS
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "factory-system",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &port53},
						{Protocol: &udp, Port: &port53},
					},
				},
			},
		},
	}
}
```

**Tests:** Create 4 unit tests
- TestGenerateNetworkPolicy (validates structure)
- TestNetworkPolicyIngress (verify same-namespace access)
- TestNetworkPolicyEgress (verify DNS allowed)
- TestNetworkPolicyIsolation (cross-tenant denial)

**Task B.1.2: Implement ResourceQuota Enforcement (1.5 hours)**

Create: `internal/platform/tenants/quota.go`

```go
package tenants

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplyQuota creates Kubernetes ResourceQuota for tenant
func (m *Manager) ApplyQuota(ctx context.Context, tenantID string) error {
	tenant, err := m.Get(ctx, tenantID)
	if err != nil {
		return err
	}
	
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant-quota-" + tenantID,
			Namespace: tenant.Namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods:           *resource.NewQuantity(int64(tenant.QuotaMaxAgents), resource.DecimalSI),
				corev1.ResourceCPU:            *resource.NewMilliQuantity(10000, resource.DecimalSI), // 10 cores
				corev1.ResourceMemory:         *resource.NewQuantity(20*1024*1024*1024, resource.BinarySI), // 20Gi
				corev1.ResourceStorage:        *resource.NewQuantity(100*1024*1024*1024, resource.BinarySI), // 100Gi
				corev1.ResourcePersistentVolumeClaims: *resource.NewQuantity(10, resource.DecimalSI),
			},
		},
	}
	
	// Apply quota to cluster
	return m.applyToKubernetes(ctx, quota)
}
```

**Tests:** Create 4 unit tests
- TestApplyQuota (validates structure)
- TestQuotaCPU (verify CPU limit)
- TestQuotaMemory (verify memory limit)
- TestQuotaEnforcement (simulate exhaustion)

**Task B.1.3: Credential Boundary Exchange (2 hours)**

Create: `internal/platform/credentials/boundary.go`

```go
package credentials

import (
	"context"
	"fmt"
	"time"
)

// BoundaryExchange issues short-lived credential for cross-tenant service call
func (b *Broker) BoundaryExchange(ctx context.Context, 
	agentID string, 
	targetTenant string, 
	targetService string,
	requiredActions []string) (*Credential, error) {
	
	// Verify agent can call target service
	agentCred, err := b.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not authorized: %w", err)
	}
	
	// Issue short-lived credential (5 min TTL)
	exchanged := &Credential{
		AgentID: agentID,
		Subject: fmt.Sprintf("%s:%s", targetTenant, targetService),
		Actions: requiredActions,
		IssuedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute), // Short TTL
	}
	
	// Sign with agent's key
	signature, err := b.signer.Sign(agentID, exchanged)
	if err != nil {
		return nil, fmt.Errorf("sign credential: %w", err)
	}
	
	exchanged.Signature = signature
	return exchanged, nil
}
```

**Tests:** Create 4 unit tests
- TestBoundaryExchange (validates credential created)
- TestExchangedCredentialExpiry (5 min TTL)
- TestExchangedCredentialSignature (verify signable)
- TestCrossTenantAuthorization (policy check)

---

## Parallel Execution Schedule (Next 2 Days)

```
TODAY (Wed 11/19)
├─ Stream A: Waiting on Docker daemon
│  └─ [blocked] Once Docker available: Run Task A.1.1 (kind setup)
│
└─ Stream B: Start immediately
   ├─ 2hrs: Task B.1.1 - NetworkPolicy (code + 4 tests)
   ├─ 1.5hrs: Task B.1.2 - ResourceQuota (code + 4 tests)
   └─ 2hrs: Task B.1.3 - Credential Boundary (code + 4 tests)

TOMORROW (Thu 11/20)
├─ Stream A: (once Docker available)
│  ├─ 1hr: Task A.1.2 - Deploy to kind
│  └─ 15min: Task A.1.3 - Validate health probes
│
└─ Stream B: Continue
   ├─ 2hrs: Task B.2.1 - MCP Service Registration
   ├─ 1.5hrs: Task B.2.2 - Cross-Tenant Service Discovery
   └─ 2hrs: Task B.2.3 - Integration Tests (6 E2E scenarios)

WEEKEND (Fri 11/21+)
├─ Stream A: Production readiness
└─ Stream B: Final validation + polish
```

---

## Checklist: Ready to Start

- [ ] All Phase 1-4 code committed ✅ (see commit ee9e537)
- [ ] 25 unit tests passing ✅
- [ ] Helm chart valid ✅
- [ ] Kustomize overlays valid ✅
- [ ] Docker daemon ready? ⏳ (check with `docker ps`)
- [ ] Go 1.23+ available? (check with `go version`)
- [ ] kubectl installed? (check with `kubectl version`)
- [ ] Kubernetes client lib in go.mod? (verify)
- [ ] GitHub branch clean? ✅ (all committed)

---

## Git Workflow

Each task completion:
```bash
# After task code (not tests) done:
git add <task-files>
git commit -m "Task B.1.1: NetworkPolicy implementation

- GenerateNetworkPolicy for tenant isolation
- Ingress: same-namespace + control-plane only
- Egress: same-namespace + control-plane + DNS"

# After tests done:
git add <test-files>
git commit -m "Task B.1.1: NetworkPolicy tests (4 tests)

- TestGenerateNetworkPolicy: validates structure
- TestNetworkPolicyIngress: same-namespace access
- TestNetworkPolicyEgress: DNS egress rules
- TestNetworkPolicyIsolation: cross-tenant denial
  
All tests passing with race detector enabled."
```

---

## Next 48-Hour Goals

**Stream A (Kubernetes Deployment):**
- [ ] kind cluster created locally
- [ ] Helm deployment succeeds
- [ ] Pods reach Ready state
- [ ] Service responds to HTTP /health

**Stream B (Multi-Tenancy):**
- [ ] 12 new unit tests written + passing
- [ ] NetworkPolicy, ResourceQuota, Credential Boundary complete
- [ ] 100% test coverage maintained
- [ ] All code committed to main

**Combined:**
- [ ] No regressions in existing 25 tests
- [ ] Git history is clean and logical
- [ ] Ready to merge PR (if branch workflow)

---

## If Docker Not Available

Stream A can be simulated/mocked:
1. Validate Helm chart syntax (already done: `helm lint`)
2. Validate Kustomize overlays (already done: `kustomize build`)
3. Test MCP protocol locally (doesn't need Kubernetes)
4. Write integration tests that mock `kubectl` output

**Stream B does NOT need Docker** — all Kubernetes types are available via Go client library imports. Test by mocking the Kubernetes client.

---

## Success Signal

**After 48 hours:**
```
Phase 3: ✅ Deployment infrastructure ready
  - Local Kubernetes working
  - Helm chart deployed
  - Service healthy

Phase 4: ✅ Multi-tenancy fundamentals
  - NetworkPolicy isolates tenants
  - ResourceQuota enforces limits  
  - Credential boundary exchange works
  - 12 new tests passing

Combined: ✅ Ready for integration phase
  - 37+ unit tests (25 existing + 12 new)
  - 0 regressions
  - 100% pass rate
```

---

## References

- **Parallel Execution Plan:** `docs/PARALLEL_EXECUTION_PLAN.md`
- **Phase 3 Guide:** `docs/PHASE3_DEPLOYMENT.md`
- **Phase 4 Guide:** `docs/PHASE4_SCALING.md`
- **Code Reference:**
  - Tenants: `internal/platform/tenants/`
  - Discovery: `internal/platform/discovery/`
  - MCP: `internal/platform/mcp/`
  - Credentials: `internal/platform/credentials/`

