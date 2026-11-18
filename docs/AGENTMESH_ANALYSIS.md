# AgentMesh Deep Dive: Peer-to-Peer Agent Networking

**Analysis Date:** 2026-11-18  
**Repository:** https://github.com/google/agentmesh  
**Focus:** P2P networking, identity, authorization, and agent-to-agent communication

---

## Executive Summary

**AgentMesh** is **not a control plane**; it's a **peer-to-peer private network for AI agents**. While Kora centralizes governance at a control plane, AgentMesh distributes agent networking across a mesh of autonomous nodes that discover and call each other directly.

**Core Concept:**
> A node runs next to an agent and gives it three things: a way to publish tools and models to the network, a way to find and call what other nodes publish, and an identity every other node can verify.

---

## Comparison: Kora vs AgentMesh

| Aspect | Kora | AgentMesh |
|--------|------|-----------|
| **Architecture** | Centralized control plane | P2P mesh of autonomous nodes |
| **Agent Identity** | SPIFFE JWT-SVID | Biscuit credentials + libp2p peer ID |
| **Communication** | HTTP through gateways | Direct P2P (libp2p) with relay fallback |
| **Credential Model** | Signed JWTs | Biscuit (capability-based, attenuable) |
| **Policy** | OPA Datalog (centralized) | Biscuit Datalog (distributed) |
| **Services Discovery** | Central service registry | DHT + gossip (libp2p Kad-DHT) |
| **Router Role** | Policy gateway | Relay node (NAT traversal) |
| **Model Serving** | Centralized inference gateway | OpenAI-compatible API per node |
| **Networking** | Kubernetes-native (Istio/Envoy) | Works anywhere (laptops, containers, phones, clusters) |
| **Use Case** | Enterprise governance | Mesh of autonomous agents |

---

## Part 1: Architecture Components

### Component Overview

| Program | Role |
|---------|------|
| **agentmesh-control-plane** | Issues short-lived credentials, holds authorization policy, verifies who is joining |
| **agentmesh-router** | Well-known relay peer, enables P2P connections across NAT, hosts discovery table |
| **agentmesh-node** | Runs next to your agent, enrolls with mesh, exposes local services, provides MCP server + OpenAI API |
| **agentmesh-one** | All-in-one (control plane + router + console) for laptops and small deployments |

### Three-Thing Model

Each node provides:

1. **Publish Tools & Models**
   - Agent exposes local capabilities to the mesh
   - Other agents discover and call them
   - MCP protocol for tool discovery

2. **Find & Call Services**
   - DHT-based discovery (libp2p Kad-DHT)
   - Direct P2P connection when possible
   - Relay through router when behind NAT
   - OpenAI-compatible endpoint for model requests

3. **Verifiable Identity**
   - Biscuit credentials bound to peer ID
   - Offline verification (no central authority needed for each call)
   - Cryptographic attestation

---

## Part 2: Identity & Authorization Model

### Two Key Properties

```
1. Nothing is reachable by default
   - A node exposes no services until its config says so
   - Policy grants are evaluated on service NAMES, never IPs
   - Deny by default

2. Identity comes from your identity provider
   - OpenID Connect token or bootstrap token
   - Control plane turns it into short-lived offline-verifiable credential
   - Bound to the node's own key
```

### Two-Token Model

**Inside the mesh:** Biscuit credentials
```
Block 0: Authority rules (RBAC from policy)
Block 1+: Task-scoped attenuation rules (service-specific grants)
```

**At borders:** Standard JWTs
```
Inbound: OIDC → Biscuit exchange
Outbound: Biscuit → JWT exchange for cloud STS
```

### Biscuit: Cryptographic Authorization Tokens

Unlike JWTs (which are opaque to receivers), Biscuits are verifiable locally:

```go
// A Biscuit contains:
type Biscuit struct {
  RootKeyID    string        // Which authority issued this?
  Authority    Block         // Authority rules (RBAC)
  Blocks       []Block       // Attenuations (task-scoped)
  Proof        Signature     // Cryptographic proof
}

// A member verifies locally without calling the control plane
// Because the proof chain is verifiable offline
```

**Key Innovation:** Credentials can be attenuated by intermediaries without re-signing.

```
Alice holds: "can call any service"
↓ (Alice delegates to Bob for one task)
Bob gets: "can call service:analyze, service:store only"
↓ (Bob's credential is verifiable by any peer without asking Alice)
Charlie verifies Bob's credential locally
```

---

## Part 3: Networking & Discovery

### libp2p: P2P Foundation

AgentMesh uses **libp2p** (Protocol Labs' modular P2P networking library):

```go
// Built on libp2p:
- go-libp2p                    // Core P2P stack
- go-libp2p-kad-dht           // Kademlia DHT (service discovery)
- go-libp2p-pubsub            // Gossip protocol (peer discovery)
- go-libp2p-gostream          // Stream multiplexing
- go-libp2p-http              // HTTP over libp2p
- go-libp2p-kbucket           // Routing table
```

### Three Network Layers

1. **Mesh Protocol** (authenticated P2P)
   - Enrollment, key exchange, auth streams
   - Direct node-to-node
   - Protected by libp2p handshake

2. **Data Plane** (service calls)
   - Tools & model requests over P2P streams
   - OpenAI-compatible endpoint (HTTP)
   - Authorized by Biscuit credentials

3. **Relay Layer** (NAT traversal)
   - Router as well-known relay
   - Nodes connect to router
   - Router relays traffic between nodes that can't reach each other directly

### Discovery Flow

```
1. Node enrolls with control plane
   ↓ Receives Biscuit + router address
2. Node connects to router (well-known peer)
   ↓ Registers itself in DHT
3. Other nodes query DHT for "service:analyze"
   ↓ Discover node1, node2, node3
4. Nodes attempt direct connection to discovered peer
   ↓ Falls back to relay if behind NAT
5. Direct P2P stream established
   ↓ MCP or OpenAI-compatible protocol
```

---

## Part 4: Practical Integration Points

### For Our Factory

**AgentMesh solves a different problem than Kora.** However, we can learn:

#### 1. **Distributed Authorization Model**

Instead of centralized policy evaluation (Kora), use capability-based credentials (Biscuit):

```go
// Our Factory could do this:

type AgentCapability struct {
  Subject    string         // Which agent?
  Actions    []string       // What can it do? (call_tool, invoke_service, etc.)
  Resources  []string       // On what? (service names, not IPs)
  ExpiresAt  time.Time      // When?
  Signature  string         // Cryptographic proof
}

// Every agent carries a credential
// Peers verify locally without calling home
// Credentials can be attenuated (Alice grants Bob a narrower scope)
```

#### 2. **Service Discovery via DHT**

Replace central service registry with DHT:

```go
// AgentMesh does:
dht.Provide(ctx, "service/analyze")     // "I offer this service"
providers := dht.FindProviders(ctx, "service/analyze")  // "Who offers this?"

// Our Factory could use same pattern:
// Agents register capabilities in DHT
// Other agents discover capabilities dynamically
// No central registry to manage
```

#### 3. **MCP Protocol Integration**

AgentMesh exposes agents as MCP servers:

```bash
agentmesh-node run                       # Local MCP server on 127.0.0.1:8080
agentmesh-node skill install             # Register the mesh as a skill
```

We already have MCP in our Factory. AgentMesh shows how to wire it up.

#### 4. **Multi-Protocol Support**

AgentMesh speaks multiple languages:
- MCP JSON-RPC (tool discovery)
- OpenAI-compatible API (model serving)
- gRPC (inter-component communication)
- Protobuf (serialization everywhere)

**Adoption:** Define everything in `.proto` files, generate clients for multiple languages, never define wire formats as Go structs + JSON tags.

#### 5. **Zero-Config, Safe Defaults**

AgentMesh's philosophy:

```
"The goal is zero trust with zero configuration.
A component started with no flags must be secure.
Every default is the safe value: deny, verify, short-lived, loopback-only, bounded."
```

**For us:** Our Agent Orchestrator should require zero config to be secure.

---

## Part 5: What We Should NOT Copy

### 1. **Full libp2p Integration (Overkill for us)**

We're not building a global P2P mesh. We're coordinating agents in controlled environments (single codebase, same infrastructure).

**Skip:** libp2p Kad-DHT, relay negotiation, NAT traversal

**Keep:** The *idea* of decentralized discovery (use existing service mesh if on K8s, or simple registry if monolithic)

### 2. **Biscuit as a Mandatory Policy Engine**

Biscuit is powerful for distributed systems. We're centralized.

**Skip:** Full Biscuit implementation with attenuation chains

**Keep:** The *idea* of local credential verification (tokens agents can validate without calling home)

### 3. **Multiple Binary Architecture**

AgentMesh ships 6+ binaries (control-plane, router, node, console, etc.). We ship one.

**Skip:** All this complexity for now

**Keep:** Clean separation of concerns internally (identity service, policy service, execution service)

---

## Part 6: Code Conventions (Different from Kora)

### 1. **One Schema, Two Encodings**

All communication via `.proto` files:

```protobuf
// api/agentmesh.proto
message EnrollRequest {
  string identity_provider_token = 1;
  string peer_id = 2;
}

// Wire format #1: Binary protobuf (mesh protocol)
// Wire format #2: ProtoJSON (operator plane)
// Never: Go struct + JSON tags
```

**Why:** Schema versioning, cross-language support, forward compatibility.

### 2. **Strict Modularity**

Components communicate **only via API**:

```
agentmesh-control-plane
agentmesh-router
agentmesh-node

All communicate through api/agentmesh.proto, nothing else.
No shared internal Go packages between binaries.
```

**For us:** Even in a monolith, treat services as separate modules with explicit APIs.

### 3. **Testing Pyramid (Strict)**

```
E2E Tests (Bats) — Only critical user journeys
  ↑ (slow, only what matters)
Integration Tests (Go) — Module interactions, API compliance
  ↑ (time-bounded, <10s per test)
Unit Tests (Go) — Isolated functions
  ↑ (fast, most coverage)
```

**Key rule:** If a test can be at a lower level, it *must* be there.

### 4. **No Third-Party Test Frameworks**

```go
// ❌ Don't do this (AgentMesh forbids it)
import "github.com/onsi/ginkgo"
import "github.com/stretchr/testify/assert"

// ✅ Do this
import "testing"
t.Run("case", func(t *testing.T) {
  if got != want {
    t.Errorf("got %v, want %v", got, want)
  }
})
```

**Why:** Tests are production code. Same linter, same readability standard.

### 5. **Flaky Test = Bug Report**

```
"A flaky test is a defect in the code, not in the test.
A test that fails once and passes on rerun found a race.
That race exists in production too.

Do not retry, widen timeout, or add time.Sleep.
Reproduce it under load, find the race, fix the code."
```

---

## Part 7: Practical Takeaways for Our Factory

### Immediate Adoptions (Low Effort, High Value)

1. **Protobuf for Internal APIs**
   - Define all service contracts in `.proto`
   - Generate Go, TypeScript, Python clients
   - Enforce schema versioning
   - **Effort:** 2-3 days, **Value:** Cross-language support, versioning

2. **Testing Pyramid Discipline**
   - Push coverage down (unit > integration > E2E)
   - Time-bound integration tests (<10s)
   - Bats for critical user journeys only
   - **Effort:** 1 week refactor, **Value:** 10x faster test suite

3. **Credential Model**
   - Add capability-based credentials (simple version of Biscuit)
   - Agents carry tokens, peers verify locally
   - Add `ExpiresAt`, `Signature` fields
   - **Effort:** 1 week, **Value:** Decentralized trust

### Medium-Term Adoptions (3-4 Weeks)

1. **Service Discovery**
   - If on Kubernetes: use built-in DNS
   - If monolithic: simple registry
   - Agents query discovery, not hardcoded addresses
   - **Value:** Dynamic scaling

2. **MCP as Standard Agent Interface**
   - All agents expose capabilities via MCP
   - Control plane discovers via MCP
   - Agents call each other's MCP endpoints
   - **Value:** Standard agent-to-agent protocol

3. **OpenAI-Compatible Model Endpoint**
   - Each agent serves `POST /v1/chat/completions`
   - Control plane routes to best provider
   - Agents can use each other's models
   - **Value:** Model federation

---

## Part 8: Comparison Table with Kora

### When to Use What

| Scenario | Kora | AgentMesh | Our Factory |
|----------|------|-----------|-------------|
| **Enterprise governance** | ✅ | ❌ | Need both |
| **Distributed trust** | ⚠️ (centralized) | ✅ | Adopt from AgentMesh |
| **Service discovery** | Registry | DHT/Gossip | Registry (simpler) |
| **Credential verification** | Central JWT | Local Biscuit | Hybrid |
| **Multi-tenant isolation** | ✅ | ❌ | ✅ |
| **P2P networking** | ❌ | ✅ | ❌ |
| **Frontend state management** | ✅ | ❌ | ✅ |
| **Backend conventions** | ✅ | ✅ | Adopt from both |

---

## Part 9: Integration Architecture for Our Factory

**Proposed Hybrid Approach:**

```
┌─────────────────────────────────────────────────┐
│          Agent Orchestrator (Central)            │
│  - Launches agents, monitors execution          │
│  - Enforces governance (from Kora)              │
└──────────────┬──────────────────────────────────┘
               │
        ┌──────┴──────┐
        ↓             ↓
   ┌────────┐    ┌────────┐
   │ Agent  │    │ Agent  │
   │  Node  │◄──►│  Node  │
   │ (MCP)  │    │ (MCP)  │
   └────────┘    └────────┘
        ↑             ↑
        └─────┬───────┘
              │
      (AgentMesh-inspired)
   Service discovery + P2P
      + Credential exchange
```

**Components:**
1. **Central Control** (from Kora): Launch, governance, policy
2. **P2P Agents** (from AgentMesh): Discover each other, call directly
3. **Frontend** (from Kora): State management, user interface
4. **Backend** (both): Conventions, testing, error handling

---

## Part 10: Adoption Roadmap

### Phase 1: Credential Model (2 weeks)

Add capability-based credentials to agents:
```go
type AgentCredential struct {
  AgentID    string
  Capabilities []string
  ExpiresAt  time.Time
  Signature  string
}
```

### Phase 2: MCP Integration (2 weeks)

Expose all agents as MCP servers:
```bash
agent --mcp-port 8080  # Exposes capabilities
```

### Phase 3: Service Discovery (2 weeks)

Agents discover each other's capabilities:
```go
// Query service registry
agents := registry.FindAgents(filterFunc)
```

### Phase 4: Cross-Agent Calling (3 weeks)

Agents call each other directly:
```go
result := agent.CallAgent("agent-123", "analyze", input)
```

---

## References

- AgentMesh README: https://github.com/google/agentmesh/blob/main/README.md
- AgentMesh Architecture: https://github.com/google/agentmesh/tree/main/site/content/docs/concepts
- AgentMesh AGENTS.md: https://github.com/google/agentmesh/blob/main/AGENTS.md
- Biscuit Documentation: https://www.biscuitsec.org/
- libp2p Specification: https://docs.libp2p.io/

---

## Conclusion

**Kora** teaches us centralized governance, frontend architecture, and backend conventions.

**AgentMesh** teaches us distributed trust, capability-based credentials, and P2P discovery.

**Our Factory** should adopt:
- **From Kora:** Governance model, frontend patterns, enforcement via linting
- **From AgentMesh:** Distributed credentials, MCP integration, service discovery ideas
- **Keep simple:** One binary, explicit APIs, strong testing discipline

The goal: Agents that coordinate safely, discover dynamically, and trust each other without a central gatekeeper for every call.
