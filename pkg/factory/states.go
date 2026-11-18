package factory

import (
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// Task lifecycle states. The string values are the exact values persisted
// to SQLite and returned over JSON — do not change them.
const (
	TaskPending   state.State = "pending"
	TaskRunning   state.State = "running"
	TaskCompleted state.State = "completed"
	TaskFailed    state.State = "failed"
)

// TaskMachine is the explicit task lifecycle.
var TaskMachine = state.NewMachine("task", TaskPending, map[state.State][]state.State{
	TaskPending: {TaskRunning},
	TaskRunning: {TaskCompleted, TaskFailed, TaskPending},
}).WithCategory("execution").
	WithDescription("Factory worker-pool task lifecycle with crash recovery and retry requeue.").
	WithStateDescriptions(map[state.State]string{
		TaskPending:   "Queued in SQLite awaiting an available worker lease.",
		TaskRunning:   "Leased by a factory worker and executing inside an isolated workspace.",
		TaskCompleted: "Task succeeded verification and evaluation gates (terminal).",
		TaskFailed:    "Retries exhausted; terminal failure recorded with diagnostics.",
	})

// Worker lifecycle states.
const (
	WorkerIdle  state.State = "idle"
	WorkerBusy  state.State = "busy"
	WorkerError state.State = "error"
)

// WorkerMachine is the explicit worker lifecycle.
var WorkerMachine = state.NewMachine("worker", WorkerIdle, map[state.State][]state.State{
	WorkerIdle:  {WorkerBusy},
	WorkerBusy:  {WorkerIdle, WorkerError},
	WorkerError: {WorkerIdle},
}).WithCategory("execution").
	WithDescription("Concurrent factory worker goroutine lease lifecycle.").
	WithStateDescriptions(map[state.State]string{
		WorkerIdle:  "Ready to claim the next pending task.",
		WorkerBusy:  "Actively executing a leased factory task.",
		WorkerError: "Recovering from a worker error before returning to idle.",
	})

// SevenLayerMachine explicitly declares the 7-layer streaming SSE pipeline lifecycle.
var SevenLayerMachine = state.NewMachine("seven-layer-stream", "idle", map[state.State][]state.State{
	"idle":    {"layer-1", "error"},
	"layer-1": {"layer-2", "error"},
	"layer-2": {"layer-3", "error"},
	"layer-3": {"layer-4", "error"},
	"layer-4": {"layer-5", "error"},
	"layer-5": {"layer-6", "error"},
	"layer-6": {"layer-7", "error"},
	"layer-7": {"done", "error"},
}).WithCategory("orchestration").
	WithDescription("7-Layer Factory Stack real-time SSE streaming execution pipeline.").
	WithStateDescriptions(map[state.State]string{
		"idle":    "Awaiting LLD YAML submission.",
		"layer-1": "Layer 1 (Factories-as-Code): Parsing LLD YAML into Specification.",
		"layer-2": "Layer 2 (Data & Context): Loading repository ADRs and context.",
		"layer-3": "Layer 3 (Compute): Provisioning isolated Go module sandbox.",
		"layer-4": "Layer 4 (Inference): Synthesizing Go code and unit tests.",
		"layer-5": "Layer 5 (Improvement): Running go build, go test, go vet & Judge.",
		"layer-6": "Layer 6 (Orchestration): Self-healing & recording state.",
		"layer-7": "Layer 7 (Access): Packaging verified deliverable.",
		"done":    "Stream completed with verified source and scorecard.",
		"error":   "Stream halted with structured error.",
	})

// Machines lists every state machine owned by the factory, for
// introspection (CLI, docs, agents).
func Machines() []*state.Machine {
	return []*state.Machine{TaskMachine, WorkerMachine, SevenLayerMachine}
}
