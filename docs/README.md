# Documentation

Visual, diagram-first documentation for the Agentic Software Factory. Start with the architecture if you want the mental model; start with getting started if you want it running.

| Document | What's inside |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | The visual walkthrough: layered architecture, task execution sequence, the generate→judge→revise improvement loop, provider selection, crash recovery, dashboard data flow |
| [GETTING_STARTED.md](GETTING_STARTED.md) | Clone → build → run in prompt-only, Ollama, or cloud mode; first task; troubleshooting |
| [CONFIGURATION.md](CONFIGURATION.md) | Every environment variable, defaults, interactions, and common recipes |
| [API.md](API.md) | HTTP + SSE + MCP reference with real request/response examples |
| [SECURITY.md](SECURITY.md) | The request guard as a decision tree, threat table, credential policy, verified behavior |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Build/test loop, load-bearing conventions, how to add a provider or task type |
| [STATE_MACHINES.md](STATE_MACHINES.md) | **Generated** — every explicit state machine (task, worker, verification, integration). Regenerate with `asf factory states --markdown > docs/STATE_MACHINES.md` |
| [ADR-001-Example.md](ADR-001-Example.md) | Example Architectural Decision Record (ADRs are highest-priority agent context) |

The [root README](../README.md) remains the single-page overview; these documents go deeper per topic.
