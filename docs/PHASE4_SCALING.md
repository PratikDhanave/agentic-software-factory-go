# Phase 4: Scaling & Multi-Tenancy (Months 10-12)

**Goal:** Factory supports 10+ tenants with isolation and distributed service discovery  
**Status:** Planning  
**Completion Date:** 2027-02-28 (target)

---

## Overview

Phase 4 transforms the Factory from single-tenant to multi-tenant with:
- **Namespace isolation** (one tenant per namespace)
- **Service discovery** (simple registry + DNS)
- **MCP as standard interface** (agent-to-agent communication)
- **Distributed verification** (credential verification without central call)

---

## Architecture

```
Control Plane (factory-system)
├── Identity Service
├── Policy Service
├── Credential Broker
├── Service Registry
└── Observability

Tenant Namespaces (factory-tenant-{name})
├── factory-tenant-team-a
│   ├── Agents (isolated)
│   ├── Workflows
│   ├── State (isolated SQLite)
│   └── ConfigMap (tenant-specific policies)
│
├── factory-tenant-team-b
│   ├── Agents
│   ├── Workflows
│   ├── State
│   └── ConfigMap
│
└── factory-tenant-team-c
    └── ...

Service Discovery
├── Kubernetes DNS (service.namespace.svc.cluster.local)
├── MCP Registry (tool catalog)
└── Peer Discovery (agent-to-agent)
```

---

## Implementation Plan

### Part 1: Tenant Abstraction (Week 1)

**internal/platform/tenants/tenant.go:**
```go
type Tenant struct {
  ID             string
  Name           string
  Namespace      string
  CreatedAt      time.Time
  StoragePath    string              // Path to tenant's SQLite DB
  PolicyID       string              // Reference to policy
  CredentialTTL  time.Duration       // Tenant-specific TTL
  QuotaMaxAgents int
}

type TenantManager struct {
  tenants map[string]*Tenant
  store   *Store // Shared persistence
}

// Register creates new tenant
func (tm *TenantManager) Register(ctx context.Context, name string) (*Tenant, error)

// Get retrieves tenant by ID or name
func (tm *TenantManager) Get(ctx context.Context, id string) (*Tenant, error)

// List returns all tenants
func (tm *TenantManager) List(ctx context.Context) ([]*Tenant, error)

// Delete removes tenant (cascades)
func (tm *TenantManager) Delete(ctx context.Context, id string) error
```

**Tenant isolation:**
- Each tenant has own namespace (factory-tenant-{name})
- Each tenant has own SQLite database in persistent volume
- Each tenant has own ConfigMap with policies
- RBAC restricts access to tenant namespace

### Part 2: Service Discovery (Week 1-2)

**internal/platform/discovery/registry.go:**
```go
type ServiceInstance struct {
  ID       string                 // UUID
  TenantID string
  AgentID  string
  Name     string                 // "code-analyzer", "tool-runner"
  Address  string                 // "agent-name.factory-tenant-x.svc.cluster.local:8080"
  Port     int
  Metadata map[string]string      // version, capabilities, etc.
  Health   HealthStatus           // healthy, unhealthy, draining
  Tags     []string               // ["mcp", "v2", "stable"]
}

type Registry struct {
  instances map[string]*ServiceInstance
  mu        sync.RWMutex
}

// Register agent with discovery
func (r *Registry) Register(ctx context.Context, si *ServiceInstance) error

// Deregister removes service
func (r *Registry) Deregister(ctx context.Context, serviceID string) error

// Discover finds services by name
func (r *Registry) Discover(ctx context.Context, serviceName string, tenantID string) ([]*ServiceInstance, error)

// Watch watches for service changes
func (r *Registry) Watch(ctx context.Context, serviceName string, tenantID string) chan *ServiceInstance

// HealthCheck marks service as healthy/unhealthy
func (r *Registry) HealthCheck(ctx context.Context, serviceID string, healthy bool) error
```

**Kubernetes integration:**
- Use native Kubernetes DNS (service.namespace.svc.cluster.local)
- Registry backed by Kubernetes Service resources
- Automatic de-registration on pod termination

### Part 3: MCP as Standard Interface (Week 2)

**internal/platform/mcp/mcp.go:**
```go
// MCPCapability describes what an agent can do
type MCPCapability struct {
  Name        string
  Description string
  InputSchema interface{}  // JSON Schema
  OutputSchema interface{}
  Idempotent  bool
}

// MCPServer exposes agent capabilities via standard protocol
type MCPServer struct {
  agentID      string
  capabilities []*MCPCapability
  tools        map[string]func(ctx context.Context, input interface{}) (interface{}, error)
}

// RegisterCapability adds new capability
func (s *MCPServer) RegisterCapability(cap *MCPCapability, handler func(ctx, input)) error

// ServeHTTP handles MCP requests
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request)

// MCPClient discovers and calls services
type MCPClient struct {
  registry *discovery.Registry
  tenantID string
}

// ListCapabilities discovers service capabilities
func (c *MCPClient) ListCapabilities(ctx context.Context, agentName string) ([]*MCPCapability, error)

// CallCapability invokes service capability
func (c *MCPClient) CallCapability(ctx context.Context, agentName string, capName string, input interface{}) (interface{}, error)
```

**MCP endpoints:**
- `GET /mcp/capabilities` — List agent capabilities
- `POST /mcp/call/:capability` — Invoke capability
- `GET /mcp/status` — Agent status

### Part 4: Distributed Verification (Week 2-3)

**Credential refresh at tenant boundary:**

```go
// When agent calls service in another tenant, exchange credentials
func (c *MCPClient) CallCapability(ctx context.Context, serviceName, capName string, input interface{}) (interface{}, error) {
  // 1. Look up service
  instances, err := c.registry.Discover(ctx, serviceName, c.tenantID)
  if err != nil {
    return nil, errors.Wrap(err, "discover service")
  }

  // 2. Get credential for target service
  cred, err := c.broker.Issue(ctx, &CredentialRequest{
    AgentID:  currentAgentID,
    Subject:  serviceName,
    Actions:  []string{capName},
    Duration: 5 * time.Minute,  // Short-lived for this call
  })
  if err != nil {
    return nil, errors.Wrap(err, "issue credential")
  }

  // 3. Call remote service with credential
  req := &MCPRequest{
    Capability: capName,
    Input:      input,
    Credential: cred,
  }

  resp, err := http.Post(instances[0].Address + "/mcp/call/" + capName, "application/json", ...)
  if err != nil {
    return nil, errors.Wrap(err, "call service")
  }

  return resp.Output, nil
}
```

### Part 5: Multi-Tenant Kubernetes Setup (Week 3)

**Kustomization for multi-tenant:**

```yaml
# config/tenants/team-a/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: factory-tenant-team-a

bases:
  - ../../base

namePrefix: team-a-

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: tenant-config
    literals:
      - tenant_id=team-a
      - tenant_name=team-a
      - quota_agents=50

images:
  - name: ghcr.io/pratikdhanave/agentic-factory
    newTag: v1.0.0
```

**NetworkPolicy for isolation:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tenant-isolation
  namespace: factory-tenant-team-a
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: factory-tenant-team-a
    - namespaceSelector:
        matchLabels:
          name: factory-system
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: factory-tenant-team-a
    - namespaceSelector:
        matchLabels:
          name: factory-system
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 53  # DNS
```

**ResourceQuota per tenant:**

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: team-a-quota
  namespace: factory-tenant-team-a
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    limits.cpu: "20"
    limits.memory: "40Gi"
    pods: "100"
    services: "10"
```

---

## Testing Strategy

**Unit Tests (70%):**
- Tenant isolation logic
- Service discovery registry
- MCP capability matching
- Credential exchange
- Distributed verification

**Integration Tests (25%):**
- Multi-tenant agent creation
- Cross-tenant MCP calls
- Credential refresh flows
- Service discovery updates
- NetworkPolicy enforcement

**E2E Tests (5%):**
- Register 3 tenants
- Each tenant creates agents
- Agent A calls service in Tenant B
- Verify isolation (Tenant C can't access B)

---

## Success Criteria

✅ 10+ tenants can coexist in one cluster  
✅ Each tenant has isolated namespace + storage  
✅ RBAC prevents cross-tenant access  
✅ Service discovery works across tenants  
✅ MCP is standard interface for all agent communication  
✅ Credentials refreshed at tenant boundaries  
✅ NetworkPolicy enforces isolation  
✅ ResourceQuota enforces tenant limits  
✅ All tests pass with race detector  

---

## Production Readiness

After Phase 4, we're ready for multi-tenant production:

| Capability | Status |
|-----------|--------|
| Single-tenant deployment | ✅ Phase 3 |
| Multi-tenant isolation | ✅ Phase 4 |
| Service discovery | ✅ Phase 4 |
| Distributed verification | ✅ Phase 4 |
| MCP standard interface | ✅ Phase 4 |
| Resource quotas | ✅ Phase 4 |
| Network isolation | ✅ Phase 4 |
| Observability | 🟡 Phase 5 |

---

## Next Steps (Phase 5+)

**Phase 5 (Months 13-18): Production & Observability**
- Prometheus metrics
- OpenTelemetry tracing
- Structured logging aggregation
- Security hardening
- Performance optimization

**Phase 6 (Months 19-24): Enterprise**
- Compliance framework (audit exports)
- Advanced governance (RBAC roles)
- SLA management
- Support for 100+ tenants

---

## References

- **docs/PHASE1_DEPLOYMENT.md**: Backend patterns
- **docs/PHASE2_GOVERNANCE.md**: Identity + credentials
- **docs/PHASE3_DEPLOYMENT.md**: Kubernetes + GitOps
- **docs/MEGA_DEEPDIVE.md**: Architecture inspiration

