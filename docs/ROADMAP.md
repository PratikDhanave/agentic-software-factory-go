# Agentic Software Factory: 24-Month Production Roadmap

**Foundation Date:** 2026-11-18  
**Target Completion:** 2027-11-18  
**Last Updated:** 2026-11-19

---

## Executive Summary

The Agentic Software Factory is transforming from a local monolith into a **production-grade enterprise control plane for AI agents**. The roadmap is divided into 6 phases, each building on the previous foundation.

**Key Principles:**
- **State Machines First:** Explicit governance for every agent lifecycle
- **Cryptographic Identity:** Every agent has verifiable identity
- **Policy Before Execution:** No action without authorization
- **Audit Everything:** Complete compliance trail
- **Kubernetes Native:** GitOps-driven deployments

---

## 24-Month Timeline

```
├─ PHASE 1: Foundation (Months 1-3)
│  └─ ✅ COMPLETE: Docker containerization + backend patterns
│
├─ PHASE 2: Governance (Months 4-6)
│  └─ ✅ COMPLETE: Identity + Credentials + Policy engine
│
├─ PHASE 3: Deployment (Months 7-9)
│  └─ 🟡 IN PROGRESS: Helm + Kustomize + GitOps
│
├─ PHASE 4: Scaling (Months 10-12)
│  └─ 🔮 PLANNED: Multi-tenancy + Service discovery + MCP
│
├─ PHASE 5: Production (Months 13-18)
│  └─ 🔮 PLANNED: Observability + Security hardening
│
└─ PHASE 6: Enterprise (Months 19-24)
   └─ 🔮 PLANNED: Compliance + Advanced governance
```

---

## Phase Breakdown

### PHASE 1: Foundation & Containerization ✅
**Months:** 1-3 | **Status:** Complete | **Commits:** ef334e8

**Deliverables:**
- ✅ Dockerfile (multi-stage build, alpine runtime)
- ✅ docker-compose.yml (local development)
- ✅ Structured logging (`internal/platform/logging/`)
- ✅ Error handling with %w wrapping (`internal/platform/errors/`)
- ✅ Type-safe HTTP clients (`internal/platform/clients/`)
- ✅ Audit logging framework (`internal/platform/audit/`)
- ✅ Makefile (7 Docker targets)
- ✅ Binary builds and runs successfully

**Architecture:**
```
Backend Patterns from Kora
  ├─ Explicit SQL (no ORMs)
  ├─ Service clients in internal/platform/
  ├─ Error wrapping with %w
  ├─ Structured logging with context
  └─ Testing pyramid discipline
```

**Success Metrics:**
- Factory builds to 150MB Docker image
- Tests pass with race detector
- All code uses structured patterns

---

### PHASE 2: Governance Layer ✅
**Months:** 4-6 | **Status:** Complete | **Commits:** b814328

**Deliverables:**
- ✅ Agent Identity System (RSA-2048 signing keys)
  * Register agents with cryptographic identity
  * Public/private key generation
  * Signature creation and verification
  * Identity registry with lookup

- ✅ Capability-Based Credentials
  * Issue time-bound credentials (subject + actions)
  * Verify signature + expiry + revocation
  * Revoke credentials immediately
  * Refresh for credential renewal
  * Check capabilities (subject + action matching)

- ✅ Policy Engine
  * Rule-based access control (allow/deny)
  * Wildcard subject matching ("code:*")
  * Condition evaluation
  * Default deny behavior
  * Pluggable policies (default, restrictive, custom)

**Architecture:**
```
Agent Action
  ↓
Credential Valid? (signature + expiry + not revoked)
  ↓
Policy Rules Match? (subject + actions)
  ↓
Conditions Met? (time, agent type, etc.)
  ↓
✓ Allow / ✗ Deny → Log audit event
```

**Testing:**
- 8 unit tests for identity (all passing)
- 10 unit tests for credentials (all passing)
- Race detector enabled
- Deterministic test fixtures

---

### PHASE 3: Deployment & Operations 🟡
**Months:** 7-9 | **Status:** In Progress | **Commits:** a1b66ee

**Deliverables:**
- ✅ Helm Chart (helm/agentic-factory/)
  * Chart.yaml, values.yaml, values-dev.yaml, values-prod.yaml
  * Templates for deployment, service, configmap, RBAC, PVC
  * Development: 1 replica, debug logging, 256Mi
  * Production: 3 replicas, HPA, 1Gi storage

- ✅ Kustomization (config/)
  * Base: Common resources
  * Dev overlay: Development-specific settings
  * Prod overlay: Production hardening

- ✅ ArgoCD Applications (argocd/applications/)
  * Dev: Automated sync from main branch
  * Prod: Manual sync from release tags

- ✅ Makefile Targets
  * kind cluster setup (make kind-up / kind-down)
  * Helm validation (make helm-lint)
  * Kustomize validation (make kustomize-validate)
  * Deployment (make helm-deploy-kind)

**What's Next in Phase 3:**
- [ ] Test Helm chart deployment to kind cluster
- [ ] Configure GitHub Actions for image build/push
- [ ] Set up ArgoCD in kind cluster
- [ ] Validate GitOps sync workflow
- [ ] Performance profiling under load

**Success Metrics:**
- Helm chart deploys without errors
- Pods are healthy (ready/live probes)
- ArgoCD syncs from Git
- Rollbacks work via GitOps

---

### PHASE 4: Scaling & Multi-Tenancy 🔮
**Months:** 10-12 | **Status:** Planned | **Commits:** a1b66ee (planning only)

**Planned Deliverables:**

1. **Tenant Management** (`internal/platform/tenants/`)
   - Register/Get/List/Delete tenants
   - Namespace per tenant (factory-tenant-{name})
   - Storage isolation (separate SQLite per tenant)
   - Resource quotas per tenant

2. **Service Discovery** (`internal/platform/discovery/`)
   - Service registry for agents
   - Kubernetes DNS integration
   - Health checking
   - Cross-tenant discovery (with policy constraints)

3. **MCP as Standard Interface** (`internal/platform/mcp/`)
   - Capability declarations
   - Service-to-service calls
   - Tool discovery and invocation
   - Standard protocol for agent communication

4. **Kubernetes Multi-Tenancy**
   - NetworkPolicy: Inbound from same namespace + factory-system
   - ResourceQuota: CPU, memory, pod limits per tenant
   - RBAC: Tenant-scoped roles
   - Kustomization for 10+ tenants

5. **Distributed Verification**
   - Short-lived credentials for cross-tenant calls
   - Credential exchange without central authority

**Architecture:**
```
Control Plane (factory-system)
  ├─ Identity Service
  ├─ Credential Broker
  ├─ Policy Service
  └─ Service Registry

Tenant Namespaces
  ├─ factory-tenant-team-a (agents, workflows)
  ├─ factory-tenant-team-b (agents, workflows)
  └─ factory-tenant-team-c (agents, workflows)

Service Discovery
  ├─ Kubernetes DNS
  ├─ MCP Registry
  └─ Peer Discovery
```

**Success Metrics:**
- 10+ tenants in one cluster
- Namespace isolation verified
- Service discovery works cross-tenant
- MCP is standard interface
- Tests pass with race detector

---

### PHASE 5: Production & Observability 🔮
**Months:** 13-18 | **Status:** Planned

**Planned Deliverables:**
- Observability stack (Prometheus, OpenTelemetry, Loki)
- Structured logging aggregation
- Distributed tracing
- Metrics dashboard
- Security hardening (pod policies, RBAC)
- Performance optimization
- SLA management

---

### PHASE 6: Enterprise 🔮
**Months:** 19-24 | **Status:** Planned

**Planned Deliverables:**
- Compliance framework (audit exports)
- Advanced governance (data policies)
- Support for 100+ tenants
- Enterprise features (HSM support, etc.)
- SLA tracking and reporting

---

## Technical Architecture

### Core Stack

```
Language: Go 1.23
Database: SQLite (Phase 1-3), PostgreSQL (Phase 5+)
Container: Docker (alpine)
Orchestration: Kubernetes
GitOps: ArgoCD
Package Manager: Helm
Overlay: Kustomize
Identity: RSA-2048 signing keys
Credentials: Capability-based model
Policy: Rule-based engine
Communication: MCP (agent-to-agent)
```

### Design Decisions

**From Kora (Runkora/kora):**
- Backend patterns (explicit SQL, service clients)
- Error handling (%w wrapping)
- Frontend architecture (RSC + BFF)
- Testing pyramid discipline
- Structured logging
- PRD → ADR → Implementation workflow

**From AgentMesh (Google):**
- P2P discovery ideas
- Testing discipline (flaky = bug)
- Capability-based credentials concept
- MCP protocol for tools
- Zero-trust principles

**From kora-platform (Clear Route):**
- GitOps model (ArgoCD)
- Helm-based packaging
- Kustomize overlay patterns
- Multi-tenant isolation
- Validation automation

**Our Innovations:**
- State machines for explicit governance
- Monolithic-first (not microservices)
- DAG executor for workflows
- Simple credential model (not full Biscuit yet)

---

## Key Metrics

| Phase | Metric | Target | Status |
|-------|--------|--------|--------|
| 1 | Docker image size | <200MB | ✅ ~150MB |
| 1 | Test coverage | >80% | ✅ All passing |
| 2 | Agent identity generation | <100ms | ✅ Verified |
| 2 | Credential verification | <50ms | ✅ Verified |
| 3 | Helm chart deploy | <5min | 🟡 Testing |
| 3 | Pod startup | <30s | 🟡 Testing |
| 4 | Multi-tenant isolation | 100% | 🔮 Planned |
| 4 | Service discovery latency | <100ms | 🔮 Planned |
| 5 | Observability overhead | <5% | 🔮 Planned |
| 6 | Support for tenants | 100+ | 🔮 Planned |

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Kubernetes complexity | Native patterns, Helm abstractions |
| Multi-tenancy leaks | NetworkPolicy, RBAC, namespace isolation |
| Credential rotation | TTL-based refresh, audit logging |
| Distributed verification | Cryptographic signatures, local checking |
| Performance at scale | Load testing (Phase 5), optimization |
| Operational burden | GitOps automation, ArgoCD |

---

## Success Criteria (End of Phase 6)

✅ **Operational:**
- Factory runs in production Kubernetes
- Multi-tenant isolation verified
- GitOps deploys and rollbacks work
- Observability covers all systems
- SLA compliance > 99.9%

✅ **Security:**
- All agents have cryptographic identity
- All actions logged to audit trail
- Policy enforced before execution
- Zero-trust architecture
- Compliance export available

✅ **Scale:**
- Supports 100+ tenants
- <100ms service discovery latency
- <50ms credential verification
- <5% observability overhead
- Horizontal scaling enabled

✅ **Developer Experience:**
- One-command deployment (GitOps)
- Clear error messages
- Complete audit trail
- Type-safe interfaces
- Comprehensive documentation

---

## References

- **docs/PHASE1_DEPLOYMENT.md**: Backend patterns
- **docs/PHASE2_GOVERNANCE.md**: Identity + governance
- **docs/PHASE3_DEPLOYMENT.md**: Kubernetes deployment
- **docs/PHASE4_SCALING.md**: Multi-tenancy architecture
- **docs/MEGA_DEEPDIVE.md**: Platform analysis and inspiration
- **Makefile**: Build and deployment targets
- **GitHub:** https://github.com/PratikDhanave/agentic-software-factory-go

---

## Timeline Chart

```
2026 Nov    Dec    2027 Jan   Feb   Mar   Apr   May   Jun   Jul   Aug   Sep   Oct   Nov
     |      |      |      |     |     |     |     |     |     |     |     |     |
     P1-----+------|P2-----|-----|-----|P3----|-----|P4----|-----|P5---------|P6----|
     ✅      ✅      ✅      🟡     🔮     🔮     🔮     🔮     🔮     🔮      🔮     🔮
     
     Phase 1: Foundation
     Phase 2: Governance
     Phase 3: Deployment
     Phase 4: Scaling
     Phase 5: Production
     Phase 6: Enterprise
```

---

## Next Steps

**Immediate (This Week):**
- [ ] Validate Helm chart with kustomize-validate
- [ ] Test kind cluster setup
- [ ] Deploy Phase 3 to local cluster

**Next 4 Weeks:**
- [ ] GitHub Actions for Docker build/push
- [ ] ArgoCD in kind cluster
- [ ] E2E tests for Helm + Kustomize
- [ ] Performance profiling

**Next 12 Weeks:**
- [ ] Phase 3 complete (ArgoCD syncing)
- [ ] Phase 4 planning → implementation
- [ ] Multi-tenant testing

**Next 24 Weeks:**
- [ ] Phase 4 complete (scaling)
- [ ] Phase 5 planning → production
- [ ] Enterprise features

