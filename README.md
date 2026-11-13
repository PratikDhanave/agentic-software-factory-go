# Agentic Software Factory (Go Edition)

A production-grade implementation of Agentic Software Engineering patterns in the Go ecosystem. This factory implements a scalable, durable, and observable infrastructure for running autonomous agents across the software development lifecycle.

## 🏗️ Architecture: The Factory Stack

The implementation follows a layered architecture inspired by modern agentic infrastructure patterns:

```mermaid
graph TD
    subgraph Access
        CLI[ASF CLI]
        WebUI[Next.js Dashboard]
        MCP[MCP Server]
        Slack[Slack Adapter]
        Teams[Teams Adapter]
    end

    subgraph Orchestration
        Orch[Factory Orchestrator]
        Queue[Task Queue]
        Workers[Worker Pool]
    end

    subgraph Intelligence
        Context[Context Engine]
        Specs[Spec Engine]
        Eval[Evaluation Engine]
    end

    subgraph Execution
        Harness[Complete Harness]
        Verify[Verification Engine]
        Sandbox[Docker Sandbox]
    end

    subgraph Persistence
        DB[(SQLite Store)]
    end

    Access --> Orch
    Orch --> Queue
    Queue --> Workers
    Workers --> Harness
    Harness --> Context
    Harness --> Specs
    Harness --> Verify
    Verify --> Sandbox
    Orch --> DB
    Workers --> DB
    Eval --> Harness
```

---

## 🌊 Message Flow: Task Execution Sequence

This diagram explains how a task (e.g., code generation) flows from an external integration through the factory to completion.

```mermaid
sequenceDiagram
    participant User as User (Slack/Teams)
    participant Adapter as Integration Adapter
    participant Orch as Factory Orchestrator
    participant DB as SQLite Persistence
    participant Worker as Agent Worker
    participant Harness as Complete Harness
    participant Sandbox as Docker Sandbox

    User->>Adapter: @asf generate feature-x
    Adapter->>Orch: SubmitTask(generate, spec-id)
    Orch->>DB: Persist Task (Pending)
    Orch->>Worker: Dispatch Task
    Worker->>DB: Update Status (Running)
    Worker->>Harness: Execute(spec-id)
    
    rect rgb(240, 240, 240)
    Note over Harness, Sandbox: AI-TDD Loop
    Harness->>Harness: Load Context & Spec
    Harness->>Harness: Generate Agent Prompt
    Harness->>Sandbox: Execute Verification (Build/Test)
    Sandbox-->>Harness: Results/Errors
    Harness-->>Harness: Retry with Feedback (if failed)
    end

    Harness-->>Worker: Final Result
    Worker->>DB: Update Status (Completed)
    Worker-->>Orch: Task Finished
    Orch-->>Adapter: Notify Completion
    Adapter-->>User: Result: PR Created / Feature Verified
```

---

## 🔄 Verification Flow: AI-TDD Loop

The verification engine handles multi-phase checks to ensure agent-generated code meets quality standards.

```mermaid
flowchart TD
    Start([Start Verification]) --> Build{Go Build}
    Build -- Fail --> ParseBuild[Parse Build Errors]
    ParseBuild --> Feedback([Generate Fix Prompt])
    
    Build -- Pass --> Test{Go Test}
    Test -- Fail --> ParseTest[Parse Test Errors]
    ParseTest --> Feedback
    
    Test -- Pass --> Lint{GolangCI-Lint}
    Lint -- Fail --> ParseLint[Parse Lint Errors]
    ParseLint --> Feedback
    
    Lint -- Pass --> Constraints{Spec Constraints}
    Constraints -- Fail --> ParseCons[Identify Violations]
    ParseCons --> Feedback
    
    Constraints -- Pass --> Success([Verification Passed])
```

---

## 🚀 Core Components

### 1. **Factory Orchestrator** (`pkg/factory`)
Manages a pool of concurrent workers. It uses a durable task queue and auto-scales workers based on load.

### 2. **Context Engine** (`pkg/context`)
Curates project-specific context (ADRs, existing specs, source code) and handles token budgeting for LLM prompts.

### 3. **Specification Engineering** (`pkg/spec`)
Uses machine-readable YAML/JSON specifications to define goals, constraints, and execution plans for agents.

### 4. **Verification Engine** (`pkg/verify`)
Executes the AI-TDD loop, running builds, tests, and linters in isolated environments to provide concrete feedback to agents.

### 5. **Security Sandbox** (`pkg/security`)
Executes potentially unsafe agent commands inside Docker containers with resource limits and restricted network access.

### 6. **MCP Server** (`pkg/mcp`)
Exposes the factory's tools and resources via the Model Context Protocol, allowing external agents to interact with the factory.

### 7. **Integration Layer** (`pkg/integration`)
Headless adapters for Slack, Microsoft Teams, and GitHub, allowing teams to trigger factory tasks from their existing tools.

---

## 🛠️ Getting Started

### Prerequisites
- Go 1.23+
- Docker (for sandboxed execution)
- Node.js (for Web UI)

### Installation
```bash
# Build the ASF CLI
go build -o asf ./cmd/asf

# Install Web UI dependencies
cd web-ui && npm install
```

### Running the Factory
```bash
# Start the factory orchestrator
./asf factory start

# Start the MCP server
./asf mcp start

# Start the Web Dashboard
cd web-ui && npm run dev
```

### Submitting a Task
```bash
# Generate code from a specification
./asf factory submit generate spec-001

# Run verification manually
./asf verify
```

---

## 🎨 Web Dashboard
The Next.js dashboard follows the **Google UX style** (white background, clean typography) and provides a real-time view of:
- Factory status and worker activity
- Task queue and results
- Context and Specification browser
- Verification logs

---

## 🧪 Testing
Playwright tests are included to verify the end-to-end flow:
```bash
npx playwright test
```

---

## 📖 Related Patterns
This implementation maps to the **Warp Factory Stack**:
- **Factories-as-code**: `pkg/factorydef`
- **Orchestration**: `pkg/factory`
- **Improvement**: `pkg/eval` + `pkg/verify`
- **Compute**: `pkg/security`
- **Access**: `pkg/mcp` + `pkg/integration`
