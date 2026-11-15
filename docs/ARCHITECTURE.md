# Architecture

A visual walkthrough of the Agentic Software Factory. Every diagram reflects the actual code — file references point at the implementation.

## The big picture

Four layers: access surfaces on top, the orchestrator in the middle, the intelligence/execution engines below it, and durable state at the bottom. The LLM provider layer plugs into the side — everything works without it (prompt-only mode).

```mermaid
graph TD
    subgraph Access["Access (cmd/)"]
        CLI["asf CLI<br/>cmd/asf"]
        WebUI["Next.js dashboard<br/>web-ui/"]
        API["HTTP API + SSE<br/>cmd/asf-web"]
        MCP["MCP server<br/>pkg/mcp"]
    end

    subgraph Guard["pkg/webguard"]
        G["loopback / Origin / Content-Type<br/>checks + optional bearer token"]
    end

    subgraph Orchestration["pkg/factory"]
        Orch["FactoryOrchestrator"]
        Queue["task queue<br/>(buffered channel, cap 100)"]
        Workers["worker pool<br/>(auto-scaling goroutines)"]
    end

    subgraph Engines["Intelligence & execution"]
        Context["Context engine<br/>pkg/context"]
        Spec["Spec engine<br/>pkg/spec"]
        Verify["Verification engine<br/>pkg/verify"]
        Eval["LLM-as-a-judge<br/>pkg/eval"]
    end

    subgraph Providers["pkg/llm (provider-agnostic)"]
        Anthropic["anthropic"]
        OpenAI["openai"]
        Ollama["ollama (local)"]
        Compat["openai-compatible"]
    end

    DB[("SQLite<br/>.asf/factory.db<br/>pkg/persistence")]

    WebUI -->|/api proxy| API
    CLI --> Orch
    API --> G --> Orch
    MCP --> G
    Orch --> Queue --> Workers
    Workers --> Context
    Workers --> Spec
    Workers --> Verify
    Workers --> Eval
    Eval --> Providers
    Workers --> Providers
    Orch <--> DB
```

## How a task executes

The full round trip from a dashboard click to a judged artifact. The same flow serves the CLI (`asf factory submit`) and MCP (`submit_task`) — they just enter at `SubmitTask`.

```mermaid
sequenceDiagram
    participant UI as Dashboard (LaunchPad)
    participant API as asf-web
    participant Orch as Orchestrator
    participant DB as SQLite
    participant W as Worker goroutine
    participant LLM as LLM provider
    participant Judge as LLM-as-a-judge

    UI->>API: POST /api/factory/launch/feature {spec_id}
    API->>Orch: SubmitTask(generate, spec)
    Orch->>Orch: register in results map (visible immediately)
    Orch->>DB: persist (pending)
    API-->>UI: 202 {task_id}
    Note over UI: FactoryStatus & ActivityFeed refresh instantly<br/>(taskLaunched DOM event)

    Orch->>W: dispatch via channel
    W->>DB: persist (running)
    W->>W: build prompt = spec + context (budgeted)
    W->>LLM: Generate(prompt)
    LLM-->>W: artifact
    W->>Judge: EvaluateCode(spec, artifact, context)
    Judge-->>W: verdict {score, feedback, improvements}
    alt score below threshold & attempts left
        W->>LLM: Generate(prompt + previous attempt + feedback)
        LLM-->>W: revised artifact
        W->>Judge: EvaluateCode(...)
    end
    W->>DB: persist (completed, result + verdict)

    UI->>API: GET /api/tasks/{id}/stream (SSE)
    API-->>UI: state-change events until terminal
```

## The improvement loop

What happens inside a `generate` task (`pkg/factory` → `executeGenerate`). The judge never discards a produced artifact — a broken judge degrades to a note, not a failure.

```mermaid
flowchart TD
    Start([generate task]) --> Build["Assemble prompt:<br/>spec prompt + project context<br/>(truncated to provider's context budget)"]
    Build --> HasLLM{LLM provider<br/>configured?}
    HasLLM -- no --> PromptOnly["Result = assembled prompt + note<br/>(prompt-only mode)"] --> Done([task completed])
    HasLLM -- yes --> Gen["Generate artifact<br/>(concurrency slot + 10-min timeout)"]
    Gen --> Judge["Judge scores artifact against spec<br/>correctness 35% · quality 25% · compliance 20%<br/>security 15% · docs 5%"]
    Judge -- judge errored --> Note["Keep artifact,<br/>attach 'evaluation skipped' note"] --> Done
    Judge --> Pass{score ≥ 0.75?}
    Pass -- yes --> Attach["Attach verdict to result:<br/>score, per-criterion breakdown,<br/>feedback, improvements, attempts"] --> Done
    Pass -- "no & attempts left<br/>(ASF_EVAL_MAX_ATTEMPTS, default 2)" --> Revise["Re-prompt with previous attempt<br/>+ judge feedback + improvement list"] --> Gen
    Pass -- "no & attempts exhausted" --> Attach
```

## Task lifecycle (explicit state machine)

Declared in `pkg/factory/states.go`; every state name below is the exact string persisted to SQLite and served over JSON. `completed` and `failed` are the only terminal states — consumers use `TaskMachine.IsTerminal()`, never hardcoded comparisons. See [STATE_MACHINES.md](STATE_MACHINES.md) for all machines (regenerate with `asf factory states --markdown`).

```mermaid
stateDiagram-v2
    [*] --> pending
    pending --> running: dispatched to a worker
    running --> completed: success
    running --> failed: retries exhausted (3)
    running --> pending: retryable failure / crash recovery
    completed --> [*]
    failed --> [*]
```

## Durability and crash recovery

Tasks are the durable unit; workers are ephemeral and rebuilt each boot. A crash mid-task re-queues the work — at-least-once semantics, so task handlers should be idempotent.

```mermaid
sequenceDiagram
    participant P1 as Process (before crash)
    participant DB as SQLite
    participant P2 as Process (after restart)

    P1->>DB: task persisted as "running"
    Note over P1: 💥 crash / restart
    P2->>DB: restore all tasks on boot
    DB-->>P2: task still "running"
    P2->>P2: explicit transition running → pending<br/>(the crash-recovery edge of TaskMachine)
    P2->>P2: re-queue task
    Note over P2: workers recreated fresh (always idle);<br/>only tasks survive restarts
```

## Provider selection

How `pkg/llm.FromEnv()` decides what powers the factory. A locally running Ollama is free, so it is safe to auto-enable; an ambient `ant auth login` profile spends real credits, so it sits behind an explicit opt-in.

```mermaid
flowchart TD
    Start([factory boot]) --> Explicit{ASF_LLM_PROVIDER set?}
    Explicit -- anthropic --> A[Anthropic client<br/>official SDK, adaptive thinking]
    Explicit -- openai --> O[OpenAI client<br/>needs OPENAI_API_KEY + ASF_MODEL]
    Explicit -- ollama --> L[Ollama client<br/>default localhost:11434,<br/>auto-picks installed chat model]
    Explicit -- openai-compatible --> C[Generic Chat-Completions client<br/>needs ASF_LLM_BASE_URL + ASF_MODEL]
    Explicit -- unset --> D1{ANTHROPIC_API_KEY /<br/>AUTH_TOKEN?}
    D1 -- yes --> A
    D1 -- no --> D2{OPENAI_API_KEY?}
    D2 -- yes --> O
    D2 -- no --> D3{Ollama reachable<br/>on localhost?}
    D3 -- yes --> L
    D3 -- no --> D4{ant profile on disk<br/>AND ASF_ENABLE_LLM=1?}
    D4 -- yes --> A
    D4 -- no --> P[Prompt-only mode<br/>tasks return assembled prompts]
```

## Dashboard data flow

Everything on the page is live; nothing is hardcoded. Components poll every 3 s and refresh instantly when a launch happens (via the `taskLaunched` DOM event).

```mermaid
graph LR
    subgraph Browser["web-ui (Next.js)"]
        LP["LaunchPad<br/>launch state machine:<br/>idle→loading→success/error<br/>(auto-reset)"]
        FS["FactoryStatus<br/>connection machine:<br/>connecting→live/unreachable"]
        AF["ActivityFeed"]
    end

    subgraph Proxy["next.config.js rewrite (dev only)"]
        R["/api/* → localhost:8080"]
    end

    subgraph Server["asf-web"]
        E1["POST /api/factory/launch/{action}"]
        E2["GET /api/factory/status"]
        E3["GET /api/tasks"]
        E4["GET /api/specs"]
    end

    LP -->|launch click| R --> E1
    LP -->|spec options| E4
    FS -->|poll 3s| E2
    AF -->|poll 3s| E3
    LP -. taskLaunched event .-> FS
    LP -. taskLaunched event .-> AF
```

## Where to go deeper

| Topic | Document |
|---|---|
| Install & first run | [GETTING_STARTED.md](GETTING_STARTED.md) |
| Every environment variable | [CONFIGURATION.md](CONFIGURATION.md) |
| HTTP API with examples | [API.md](API.md) |
| Threat model & guards | [SECURITY.md](SECURITY.md) |
| Contributing & conventions | [DEVELOPMENT.md](DEVELOPMENT.md) |
| All state machines (generated) | [STATE_MACHINES.md](STATE_MACHINES.md) |
