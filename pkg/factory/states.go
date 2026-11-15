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
//
//	pending -> running            dispatch to a worker
//	running -> completed          success (terminal)
//	running -> failed             retries exhausted (terminal)
//	running -> pending            retryable failure, or crash recovery on restart
//
// A task is only ever observed as "failed" once its retries are exhausted;
// retryable failures re-queue directly to "pending".
var TaskMachine = state.NewMachine("task", TaskPending, map[state.State][]state.State{
	TaskPending: {TaskRunning},
	TaskRunning: {TaskCompleted, TaskFailed, TaskPending},
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
})

// Machines lists every state machine owned by the factory, for
// introspection (CLI, docs, agents).
func Machines() []*state.Machine {
	return []*state.Machine{TaskMachine, WorkerMachine}
}
