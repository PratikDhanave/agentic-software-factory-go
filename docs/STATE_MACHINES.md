# State Machines

Every lifecycle in the Agentic Software Factory is an explicit, declared
state machine (see `pkg/state`). This document is generated — regenerate
with `asf factory states --markdown > docs/STATE_MACHINES.md`.

The same declarations are served at runtime from `GET /api/state-machines`
(asf-web) and are mirrored for the dashboard in `web-ui/src/lib/stateMachine.ts`.

## task

Initial state: `pending`

```mermaid
stateDiagram-v2
    [*] --> pending
    pending --> running
    running --> completed
    running --> failed
    running --> pending
    completed --> [*]
    failed --> [*]
```

## worker

Initial state: `idle`

```mermaid
stateDiagram-v2
    [*] --> idle
    busy --> idle
    busy --> error
    error --> idle
    idle --> busy
```

## verification

Initial state: `build`

```mermaid
stateDiagram-v2
    [*] --> build
    build --> test
    build --> failed
    constraints --> passed
    constraints --> failed
    lint --> constraints
    lint --> passed
    lint --> failed
    test --> lint
    test --> failed
    failed --> [*]
    passed --> [*]
```

## integration-connection

Initial state: `disabled`

```mermaid
stateDiagram-v2
    [*] --> disabled
    connected --> degraded
    connected --> offline
    connected --> disabled
    degraded --> connected
    degraded --> offline
    degraded --> disabled
    disabled --> offline
    offline --> connected
    offline --> disabled
```
