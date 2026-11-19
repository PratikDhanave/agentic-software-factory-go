# Kora Platform: Production Deployment Architecture

**Analysis Date:** 2026-11-19  
**Repository:** clear-route/kora-platform (private)  
**Focus:** Kubernetes-based production deployment of Kora control plane

---

## Overview

**kora-platform** is a **GitOps-based deployment infrastructure** for Kora using ArgoCD, Helm, and Kustomize. It shows how to operationalize the Kora control plane in production Kubernetes environments.

**Key insight:** This bridges the gap between understanding Kora's architecture (what we learned from runkora/kora) and deploying it at scale.

---

## Architecture Components

### 1. GitOps Infrastructure (ArgoCD)

```yaml
# argocd/dev/kora.yaml - Application definition
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kora-platform

spec:
  sources:
    - chart: kora-crds            # CRD definitions
    - chart: kora                  # Main control plane
    - path: platform/dev           # Kustomization overlays
  
  destination:
    namespace: kora-system
    
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**Philosophy:** 
- Single source of truth (Git)
- Declarative infrastructure
- Automatic drift correction
- Version-controlled everything

### 2. Multi-Environment Strategy

```
Environments:
├── dev/           # Development (testing, smoke tests)
│   ├── kora.yaml
│   ├── tenants/
│   ├── patches/
│   └── smoke/     # Smoke test agents
└── prod/          # Production (customers, real workloads)
    ├── kora.yaml
    ├── tenants/
    └── patches/

Shared:
├── platform/base/ # Common Kustomization base
├── values/        # Helm values per environment
└── tenants/       # Tenant configurations
```

### 3. Helm-based Deployment

```makefile
# From Makefile
KORA_CRDS_CHART = oci://ghcr.io/runkora/charts/kora-crds
KORA_CRDS_VERSION = 0.13.0-rc.7  # Version-locked in ArgoCD apps

# Two chart dependencies:
# 1. kora-crds - Kubernetes CRDs for Kora resources
# 2. kora - Main control plane Helm chart
```

**Key pattern:** Version-locking in ArgoCD apps ensures reproducible deployments.

### 4. Tenant Management

```
Tenants (Multi-tenant Kora instances):
├── _common/              # Shared configurations (roles, RBAC)
├── kora/                 # Internal Kora instance
│   ├── dev/
│   │   ├── smoke/        # Smoke tests with Kora agents
│   │   ├── evals/        # Evaluation jobs
│   │   └── kustomization.yaml
│   └── prod/
├── neso/                 # Customer tenant
│   ├── dev/
│   └── prod/
└── constellation/        # Another customer tenant
    ├── dev/
    └── prod/
```

**Multi-tenancy model:**
- Each tenant = separate namespace (`kora-tenant-{name}`)
- Shared control plane (kora-system)
- Isolated data + policies per tenant
- ArgoCD manages each tenant independently

### 5. Kustomize Overlay Structure

```yaml
# tenants/kora/dev/kustomization.yaml
namespace: kora-tenant-kora

resources:
  - ../base                                # Shared tenant base
  - ../../_common/dev/roles                # Common RBAC
  - connections/clear-route-github.yaml    # GitHub integration
  - connections/kubernetes.yaml            # K8s cluster access
  - evals/                                 # Evaluation configs
  - smoke/                                 # Smoke tests

patches:
  - path: patches/budgets.yaml             # Resource limits
```

**Pattern:** Layer-based composition
- Base: Common tenant setup
- Common: Shared RBAC, roles, policies
- Environment: Dev/prod overrides
- Features: Optional modules (evals, smoke tests)

---

## Deployment Model

### How Kora Gets Deployed

```
1. Developer pushes to clear-route/kora-platform main
   ↓
2. ArgoCD detects change (watches main branch)
   ↓
3. ArgoCD pulls Helm charts from ghcr.io/runkora/charts
   ↓
4. Helm renders values + charts → Kubernetes manifests
   ↓
5. Kustomize applies overlays (dev/prod patches)
   ↓
6. ArgoCD applies manifests to Kubernetes cluster
   ↓
7. kora-system namespace gets deployed/updated
   ↓
8. Each tenant namespace syncs independently
```

### Validation Pipeline

```makefile
# From Makefile targets
make kora-schemas     # Generate JSON schemas from Helm charts
make lint             # Validate all manifests
  - kubeconform (K8s schema validation)
  - Custom CRD schemas (runkora.io/...)
  - Kustomization validation
  - YAML syntax check
```

**Safety gates before deployment:**
- Schema validation
- Custom CRD validation
- Drift detection (ServerSideDiff)
- RespectIgnoreDifferences (for operator-managed fields)

---

## Key Configurations

### 1. Pod Security & Network Policies

```yaml
# From ArgoCD Application spec
syncPolicy:
  managedNamespaceMetadata:
    labels:
      istio-injection: enabled                              # Service mesh
      pod-security.kubernetes.io/enforce: restricted        # Strict PSA
      pod-security.kubernetes.io/warn: restricted
      pod-security.kubernetes.io/audit: restricted
```

**Security-first:** Every namespace gets strict pod security policies.

### 2. Sync Policies with Backoff

```yaml
syncPolicy:
  automated:
    prune: true      # Delete removed resources
    selfHeal: true   # Revert manual changes
  retry:
    limit: 8         # Up to 8 retries
    backoff:
      duration: 30s
      factor: 2      # Exponential backoff: 30s, 60s, 2m, 4m, 8m...
      maxDuration: 5m
```

**Why backoff:** RDS provisioning takes ~10 minutes; retries handle startup sequencing.

### 3. Smoke Tests in Dev

```yaml
# tenants/kora/dev/smoke/
├── kustomization.yaml
├── agents.yaml           # Kora agents for testing
├── triggers.yaml         # Trigger configurations
├── verifier.yaml         # Verification jobs
├── verifier-triggers.yaml
└── system-prompt.yaml    # Agent system prompts
```

**Pattern:** Smoke tests are actual Kora resources, not external test runners.

### 4. Model Catalog

```yaml
# platform/base/models/
├── anthropic-claude-haiku-5-5.yaml
├── google-gemini-3-1-pro.yaml
├── openai-gpt-6-luna.yaml
├── perplexity-pplx-decider-flash-v1-2.yaml
└── [25+ model configurations]
```

**Model as code:** Each model is a Kubernetes resource, declaratively defined.

---

## Comparison: Kora Core vs Kora Platform

| Aspect | Kora Core (runkora/kora) | Kora Platform (clear-route) |
|--------|--------------------------|----------------------------|
| **What?** | Control plane source code | Production deployment infra |
| **Stack** | Go + PostgreSQL | Helm + ArgoCD + Kustomize |
| **Environment** | Local development (make up) | Kubernetes clusters |
| **Multi-tenant** | Built-in (design) | Configured (practice) |
| **Deployment** | Manual (make redeploy) | GitOps (ArgoCD) |
| **Validation** | Tests + linting | kubeconform + CRD schemas |
| **Versioning** | Semantic (binaries) | Chart versions (Helm) |

---

## What We Can Learn for Our Factory

### 1. GitOps Model (Highly Applicable)

Apply ArgoCD + Kustomize pattern to Agent Factory:

```yaml
# applications/factory.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: agent-factory

spec:
  sources:
    - chart: agent-factory
      targetRevision: 1.0.0
    - path: config/dev  # or prod
  
  destination:
    namespace: factory-system
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**Benefits:**
- Declarative infrastructure
- Automatic drift correction
- Version-controlled everything
- Easy rollbacks

### 2. Multi-Tenant Architecture (Medium Value)

If Factory grows to serve multiple teams/orgs:

```
factory-system/              # Control plane
├── orchestrator
├── agent-launcher
└── observability

factory-tenant-team-a/      # Tenant isolation
└── agents, workflows, state

factory-tenant-team-b/
└── agents, workflows, state
```

### 3. Environment Layering (Highly Applicable)

Use Kustomize overlays instead of env variables:

```
config/
├── base/                 # Common
├── dev/                  # Development overrides
│   ├── kustomization.yaml
│   └── patches/
└── prod/                 # Production overrides
    ├── kustomization.yaml
    └── patches/
```

**Current approach:** Environment variables are fragile. Kustomize is explicit.

### 4. Model as Code (Applicable)

Instead of hardcoding models, make them Kubernetes resources:

```yaml
# models/claude.yaml
apiVersion: factory.io/v1
kind: Model
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  version: 5.0
  parameters:
    temperature: 0.7
    max_tokens: 2048
  availability: "production"
```

**Benefit:** Models become discoverable, versionable, tenant-specific.

### 5. Helm for Distribution (Future Goal)

Package Agent Factory as Helm chart:

```bash
# Install Factory with one command
helm install factory oci://registry/agent-factory \
  --values values/prod.yaml
```

**Current:** We ship a Go binary. Helm adds K8s support.

---

## Deployment Workflow (For Our Factory)

### Phase 1: Local Development (Current)
```bash
make up          # Start everything locally
make redeploy    # Update a service
```

### Phase 2: Containerization (Next)
```bash
docker build -t factory:latest .
docker push ghcr.io/pratikdhanave/factory:latest
```

### Phase 3: Kubernetes Support (Future)
```bash
helm install factory ./chart \
  --values values/dev.yaml
# Or via ArgoCD:
argocd app create factory-platform
```

### Phase 4: GitOps (Mature)
```
# Everything in Git
# ArgoCD auto-deploys
# Terraform/CDK for infrastructure
# Helm for applications
```

---

## Production Readiness Checklist

Kora Platform demonstrates:

- ✅ **Version-locked dependencies** (Helm charts, CRDs)
- ✅ **Multi-environment support** (dev/prod with patches)
- ✅ **Automated validation** (kubeconform, schema checks)
- ✅ **Multi-tenancy isolation** (separate namespaces)
- ✅ **Smoke tests** (actual agents, not external test scripts)
- ✅ **Pod security policies** (restricted by default)
- ✅ **Automatic healing** (ArgoCD selfHeal)
- ✅ **Resource isolation** (budgets, RBAC)
- ✅ **Model versioning** (models as code)
- ✅ **GitOps practices** (single source of truth)

**For our Factory:**
- 🟡 Currently: Local deployment only
- 🟢 Add: Docker containers
- 🟢 Add: Helm charts
- 🟡 Later: ArgoCD integration
- 🟡 Later: Multi-tenant support

---

## Integration With Our Factory

### How Kora Fits Into Our Stack

```
Our Factory (Agentic Software Factory)
├── Agent Orchestrator (control center)
├── State Machines (explicit governance)
├── DAG Executor (workflow engine)
└── Multi-Agent Coordinator

+ Kora Platform (operational layer)
├── Identity & Policy (agent governance)
├── Credential Broker (secure communication)
├── Inference Gateway (model routing)
└── Observability (audit trails)

+ Deployment (Kubernetes)
├── ArgoCD (GitOps)
├── Helm (packaging)
└── Kustomize (environments)
```

### Deployment Topology

```
Agent Factory (control plane)
├── Orchestrator API
├── State registry
├── Credential manager
└── Observability

Kora Platform (governance overlay - optional)
├── Policy engine
├── Identity service
├── Egress brokers
└── Audit logging

Kubernetes Cluster
├── factory-system (control)
├── factory-tenant-prod (production agents)
└── factory-tenant-dev (development agents)
```

---

## Actionable Next Steps

### For Our Factory (3-6 Months)

**Phase 1: Containerization (1-2 weeks)**
- Create Dockerfile for Factory binary
- Push to container registry (ghcr.io)
- Document build process

**Phase 2: Helm Chart (2-3 weeks)**
- Create helm/agent-factory/ directory
- Templates for:
  - Deployment (orchestrator, executor, coordinator)
  - ConfigMap (state machines, policies)
  - Service (API endpoint)
  - RBAC (agent permissions)
- Values files for dev/prod

**Phase 3: Kustomize Overlays (1-2 weeks)**
- Create config/base/ with common setup
- dev/ and prod/ overlays
- Patches for resource limits, replicas

**Phase 4: ArgoCD Integration (1-2 weeks)**
- Create ArgoCD Application CRDs
- Multi-tenant support
- Smoke test agents (like Kora does)

**Phase 5: Observability (Ongoing)**
- State change events (like Kora StateChangeEvent)
- Audit logs per tenant
- Metrics dashboard

---

## References

- **Kora Core:** https://github.com/runkora/kora
- **Kora Platform:** clear-route/kora-platform (private)
- **ArgoCD:** https://argo-cd.readthedocs.io/
- **Helm:** https://helm.sh/docs/
- **Kustomize:** https://kustomize.io/

---

## Summary

**Kora Platform** shows the production path for agent control planes. It bridges the gap between:

1. **Understanding** (Kora's architecture)
2. **Implementing** (Our Factory's orchestration)
3. **Operating** (Kubernetes deployment)

The key insight: **GitOps + Kustomize + Multi-tenancy** is how you scale agent governance from "runs locally" to "runs in production for 100+ customers."

For our Factory:
- **Now:** Single binary, state machines, local orchestration
- **Next 6 months:** Add Helm + Kustomize support
- **Future:** ArgoCD + multi-tenant + production-grade governance

This is our path to a production-ready agentic platform.
