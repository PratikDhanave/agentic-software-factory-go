# Phase 1: Foundation & Containerization (Months 1-3)

**Status:** In Progress  
**Goal:** Factory runs locally in Docker + K8s with proper backend patterns  
**Completion Date:** 2026-11-26 (target)

---

## What We're Building

### Phase 1A: Code Patterns & Testing (Week 1)
- ✅ **Logging**: Structured logging with context (internal/platform/logging)
- ✅ **Error Handling**: %w wrapping utilities (internal/platform/errors)
- ✅ **HTTP Clients**: Typed service clients with retry/timeout (internal/platform/clients)
- ✅ **Audit Logging**: Event tracking for compliance (internal/platform/audit)
- 🟡 **Testing Pyramid**: Refactor tests (unit > integration > E2E)

### Phase 1B: Containerization (Week 2)
- ✅ **Dockerfile**: Multi-stage build, alpine runtime
- ✅ **docker-compose.yml**: Local development with volume mounts
- 🟡 **Build Pipeline**: Automated Docker image tagging
- 🟡 **Registry**: ghcr.io image push

### Phase 1C: Kubernetes Support (Week 3)
- 🟡 **kind Cluster**: Local K8s testing (make kind-up)
- 🟡 **Helm Chart Structure**: Basic templates for deployment
- 🟡 **Local Testing**: Deploy to kind cluster

---

## Quick Start

### Local Development (Using Dockerfile)

\`\`\`bash
# Build and run in Docker
make docker-build
make docker-up

# Factory will be available at http://localhost:8080

# View logs
make docker-logs

# Stop everything
make docker-down
\`\`\`

### Local Development (Native Go)

\`\`\`bash
# Build and run directly
make run

# Or with hot reload
make dev
\`\`\`

### Testing

\`\`\`bash
# Run all tests (unit + race detector)
make test

# Run specific package
go test -race ./internal/platform/logging/...

# Run with verbose output
go test -v -race ./...
\`\`\`

---

## Backend Patterns Adopted

### 1. Error Wrapping with %w

**Before:**
\`\`\`go
return fmt.Errorf("operation failed: %s", err.Error())  // ❌ Loses error chain
\`\`\`

**After:**
\`\`\`go
return fmt.Errorf("operation failed: %w", err)  // ✅ Preserves chain

// Enables proper error inspection:
if errors.Is(err, io.EOF) { ... }
if errors.As(err, &myError) { ... }
\`\`\`

### 2. Structured Logging

\`\`\`go
import "github.com/PratikDhanave/agentic-software-factory-go/internal/platform/logging"

logging.Info(ctx, "agent enrolled",
  logging.NewField("agent_id", agentID),
  logging.NewField("agent_type", agentType))
\`\`\`

### 3. Service Clients (Type-Safe)

\`\`\`go
// internal/platform/clients/orchestrator/client.go
type Client struct {
  c *clients.Client
}

func (cl *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
  resp, err := cl.c.Get(ctx, "/agents/"+id)
  if err != nil {
    return nil, fmt.Errorf("get agent: %w", err)
  }
  // ...
}
\`\`\`

### 4. Audit Logging

\`\`\`go
import "github.com/PratikDhanave/agentic-software-factory-go/internal/platform/audit"

audit.LogTransition(ctx, agentID, "pending", "running", "user requested start")
\`\`\`

---

## Success Criteria

✅ Factory builds to Docker image (alpine, 150MB compressed)  
✅ Local docker-compose runs without errors  
✅ All new code uses structured logging  
✅ All errors wrapped with %w  
✅ Audit events logged for state changes  
✅ Tests pass with race detector  
✅ Code patterns documented and enforced

