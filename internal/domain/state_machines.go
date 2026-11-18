package domain

import (
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// ============================================================================
// Explicit Domain State Machines — Unified on pkg/state.Machine
// ============================================================================
// Every domain lifecycle is declared once as a pkg/state.Machine so UX, DX,
// and AX (CLI, HTTP API, MCP tools, and UI explorer) share the exact same
// source of truth.

// 1. FACTORY AUTONOMY PHASE MACHINE (Crawl <-> Walk <-> Run)
type FactoryPhase string

const (
	PhaseCrawl FactoryPhase = "crawl" // Supervised: human approval on every decision
	PhaseWalk  FactoryPhase = "walk"  // Balanced: auto-approve >=90% confidence & learned patterns
	PhaseRun   FactoryPhase = "run"   // Autonomous: auto-execute >=70% confidence
)

var FactoryPhaseMachine = state.NewMachine("factory-autonomy-phase", state.State(PhaseCrawl), map[state.State][]state.State{
	state.State(PhaseCrawl): {state.State(PhaseWalk), state.State(PhaseRun)},
	state.State(PhaseWalk):  {state.State(PhaseRun), state.State(PhaseCrawl)},
	state.State(PhaseRun):   {state.State(PhaseWalk), state.State(PhaseCrawl)},
}).WithCategory("governance").
	WithDescription("Controls human-in-the-loop autonomy level across Crawl (supervised), Walk (balanced), and Run (autonomous).").
	WithStateDescriptions(map[state.State]string{
		state.State(PhaseCrawl): "Supervised mode: every agent governance gate pauses for human review.",
		state.State(PhaseWalk):  "Balanced mode: high-confidence (>=90%) and learned patterns proceed automatically.",
		state.State(PhaseRun):   "Autonomous mode: decisions >=70% confidence execute end-to-end without blocking.",
	})

func (p FactoryPhase) IsValid() bool {
	return FactoryPhaseMachine.IsValid(state.State(p))
}

func (p FactoryPhase) CanTransitionTo(next FactoryPhase) bool {
	return FactoryPhaseMachine.Can(state.State(p), state.State(next))
}

func (p FactoryPhase) String() string { return string(p) }

// 2. AGENT DECISION LIFECYCLE MACHINE
type DecisionState string

const (
	DecisionPending   DecisionState = "pending"
	DecisionApproved  DecisionState = "approved"
	DecisionRejected  DecisionState = "rejected"
	DecisionCorrect   DecisionState = "correct"
	DecisionIncorrect DecisionState = "incorrect"
)

var DecisionLifecycleMachine = state.NewMachine("agent-decision", state.State(DecisionPending), map[state.State][]state.State{
	state.State(DecisionPending):  {state.State(DecisionApproved), state.State(DecisionRejected)},
	state.State(DecisionApproved): {state.State(DecisionCorrect), state.State(DecisionIncorrect)},
}).WithCategory("governance").
	WithDescription("Tracks each specialized agent decision from proposal through approval and verified build outcome for continuous calibration.").
	WithStateDescriptions(map[state.State]string{
		state.State(DecisionPending):   "Proposed by agent; awaiting policy evaluation or human approval.",
		state.State(DecisionApproved):  "Approved via confidence policy or human reviewer.",
		state.State(DecisionRejected):  "Rejected by human reviewer or policy veto (terminal).",
		state.State(DecisionCorrect):   "Downstream build, tests, and judge rubric succeeded (feeds positive calibration).",
		state.State(DecisionIncorrect): "Downstream verification failed (triggers confidence recalibration).",
	})

func (s DecisionState) IsValid() bool {
	return DecisionLifecycleMachine.IsValid(state.State(s))
}

func (s DecisionState) CanTransitionTo(next DecisionState) bool {
	return DecisionLifecycleMachine.Can(state.State(s), state.State(next))
}

func (s DecisionState) IsTerminal() bool {
	return DecisionLifecycleMachine.IsTerminal(state.State(s))
}

func (s DecisionState) String() string { return string(s) }

// 3. HUMAN APPROVAL GATE MACHINE
type ApprovalState string

const (
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalRejected ApprovalState = "rejected"
	ApprovalExpired  ApprovalState = "expired"
)

var ApprovalGateMachine = state.NewMachine("human-approval-gate", state.State(ApprovalPending), map[state.State][]state.State{
	state.State(ApprovalPending): {
		state.State(ApprovalApproved),
		state.State(ApprovalRejected),
		state.State(ApprovalExpired),
	},
}).WithCategory("governance").
	WithDescription("Human-in-the-loop review ticket lifecycle for paused sessions and builds.").
	WithStateDescriptions(map[state.State]string{
		state.State(ApprovalPending):  "Awaiting human operator review in the Governance inbox.",
		state.State(ApprovalApproved): "Approved by operator (optionally with strategy override).",
		state.State(ApprovalRejected): "Rejected by operator; halts downstream execution.",
		state.State(ApprovalExpired):  "Approval window timed out without operator action.",
	})

func (s ApprovalState) IsValid() bool {
	return ApprovalGateMachine.IsValid(state.State(s))
}

func (s ApprovalState) CanTransitionTo(next ApprovalState) bool {
	return ApprovalGateMachine.Can(state.State(s), state.State(next))
}

func (s ApprovalState) IsTerminal() bool {
	return ApprovalGateMachine.IsTerminal(state.State(s))
}

func (s ApprovalState) String() string { return string(s) }

// 4. WORKFLOW EXECUTION LIFECYCLE MACHINE (Domain-Level)
type ExecutionState string

const (
	ExecutionQueued       ExecutionState = "queued"
	ExecutionExecuting    ExecutionState = "executing"
	ExecutionCompleted    ExecutionState = "completed"
	ExecutionFailed       ExecutionState = "failed"
	ExecutionCancelled    ExecutionState = "cancelled"
	ExecutionAwaitingGate ExecutionState = "awaiting_gate"
)

var ExecutionLifecycleMachine = state.NewMachine("domain-execution", state.State(ExecutionQueued), map[state.State][]state.State{
	state.State(ExecutionQueued): {
		state.State(ExecutionExecuting),
		state.State(ExecutionCancelled),
		state.State(ExecutionAwaitingGate),
	},
	state.State(ExecutionExecuting): {
		state.State(ExecutionCompleted),
		state.State(ExecutionFailed),
		state.State(ExecutionCancelled),
		state.State(ExecutionAwaitingGate),
	},
	state.State(ExecutionAwaitingGate): {
		state.State(ExecutionExecuting),
		state.State(ExecutionFailed),
		state.State(ExecutionCancelled),
	},
}).WithCategory("execution").
	WithDescription("Gate-aware workflow execution lifecycle supporting pause/resume at governance checkpoints.").
	WithStateDescriptions(map[state.State]string{
		state.State(ExecutionQueued):       "Queued for execution.",
		state.State(ExecutionExecuting):    "Actively executing workflow steps.",
		state.State(ExecutionAwaitingGate): "Paused at a human-in-the-loop approval gate.",
		state.State(ExecutionCompleted):    "All steps completed cleanly.",
		state.State(ExecutionFailed):       "Execution terminated due to step error.",
		state.State(ExecutionCancelled):    "Cancelled by operator.",
	})

func (s ExecutionState) IsValid() bool {
	return ExecutionLifecycleMachine.IsValid(state.State(s))
}

func (s ExecutionState) CanTransitionTo(next ExecutionState) bool {
	return ExecutionLifecycleMachine.Can(state.State(s), state.State(next))
}

func (s ExecutionState) IsTerminal() bool {
	return ExecutionLifecycleMachine.IsTerminal(state.State(s))
}

func (s ExecutionState) IsRunning() bool {
	return s == ExecutionExecuting || s == ExecutionAwaitingGate
}

func (s ExecutionState) String() string { return string(s) }

// 5. RETRYABLE WORKFLOW STEP / TASK MACHINE
type TaskState string

const (
	TaskPending   TaskState = "pending"
	TaskRunning   TaskState = "running"
	TaskCompleted TaskState = "completed"
	TaskFailed    TaskState = "failed"
	TaskRetrying  TaskState = "retrying"
	TaskSkipped   TaskState = "skipped"
	TaskTimedout  TaskState = "timedout"
)

var DomainTaskMachine = state.NewMachine("retryable-step", state.State(TaskPending), map[state.State][]state.State{
	state.State(TaskPending):  {state.State(TaskRunning), state.State(TaskSkipped)},
	state.State(TaskRunning):  {state.State(TaskCompleted), state.State(TaskFailed), state.State(TaskTimedout), state.State(TaskSkipped)},
	state.State(TaskFailed):   {state.State(TaskRetrying), state.State(TaskSkipped)},
	state.State(TaskRetrying): {state.State(TaskRunning), state.State(TaskCompleted), state.State(TaskFailed), state.State(TaskTimedout)},
}).WithCategory("execution").
	WithDescription("Individual DAG step lifecycle with retry, timeout, and upstream-skip semantics.").
	WithStateDescriptions(map[state.State]string{
		state.State(TaskPending):   "Waiting for upstream dependencies.",
		state.State(TaskRunning):   "Executing step handler.",
		state.State(TaskFailed):    "Step failed; eligible for retry or skip.",
		state.State(TaskRetrying):  "Backoff retry in progress.",
		state.State(TaskCompleted): "Step finished with valid output.",
		state.State(TaskSkipped):   "Skipped because an upstream dependency failed.",
		state.State(TaskTimedout):  "Exceeded step execution timeout.",
	})

func (s TaskState) IsValid() bool {
	return DomainTaskMachine.IsValid(state.State(s))
}

func (s TaskState) CanTransitionTo(next TaskState) bool {
	return DomainTaskMachine.Can(state.State(s), state.State(next))
}

func (s TaskState) IsTerminal() bool {
	return s == TaskCompleted || s == TaskSkipped || s == TaskTimedout || s == TaskFailed
}

func (s TaskState) IsRetryable() bool {
	return s == TaskFailed
}

func (s TaskState) String() string { return string(s) }

// 6. HIGH-LEVEL SESSION LIFECYCLE MACHINE
type SessionState string

const (
	SessionQueued         SessionState = "queued"
	SessionAnalyzing      SessionState = "analyzing"
	SessionPlanningAgents SessionState = "planning_agents"
	SessionExecuting      SessionState = "executing"
	SessionAwaitingGate   SessionState = "awaiting_gate"
	SessionVerifying      SessionState = "verifying"
	SessionCompleted      SessionState = "completed"
	SessionFailed         SessionState = "failed"
	SessionCancelled      SessionState = "cancelled"
)

var SessionLifecycleMachine = state.NewMachine("session-lifecycle", state.State(SessionQueued), map[state.State][]state.State{
	state.State(SessionQueued):         {state.State(SessionAnalyzing), state.State(SessionCancelled)},
	state.State(SessionAnalyzing):      {state.State(SessionPlanningAgents), state.State(SessionFailed), state.State(SessionCancelled)},
	state.State(SessionPlanningAgents): {state.State(SessionExecuting), state.State(SessionFailed), state.State(SessionCancelled)},
	state.State(SessionExecuting):      {state.State(SessionAwaitingGate), state.State(SessionVerifying), state.State(SessionFailed), state.State(SessionCancelled)},
	state.State(SessionAwaitingGate):   {state.State(SessionExecuting), state.State(SessionFailed), state.State(SessionCancelled)},
	state.State(SessionVerifying):      {state.State(SessionCompleted), state.State(SessionFailed), state.State(SessionCancelled)},
}).WithCategory("governance").
	WithDescription("End-to-end multi-agent planning, gate review, and verification session lifecycle.")

func (s SessionState) IsValid() bool {
	return SessionLifecycleMachine.IsValid(state.State(s))
}

func (s SessionState) CanTransitionTo(next SessionState) bool {
	return SessionLifecycleMachine.Can(state.State(s), state.State(next))
}

func (s SessionState) IsTerminal() bool {
	return SessionLifecycleMachine.IsTerminal(state.State(s))
}

func (s SessionState) IsRunning() bool {
	return !s.IsTerminal() && s != SessionQueued
}

func (s SessionState) String() string { return string(s) }

// DomainMachines returns all explicit state machines declared in package domain.
func DomainMachines() []*state.Machine {
	return []*state.Machine{
		FactoryPhaseMachine,
		DecisionLifecycleMachine,
		ApprovalGateMachine,
		ExecutionLifecycleMachine,
		DomainTaskMachine,
		SessionLifecycleMachine,
	}
}

// ============================================================================
// OBSERVABLE STATE CHANGE EVENT & VALIDATOR
// ============================================================================

type StateChangeEvent struct {
	Entity    string                 `json:"entity"`
	EntityID  string                 `json:"entity_id"`
	OldState  string                 `json:"old_state"`
	NewState  string                 `json:"new_state"`
	Reason    string                 `json:"reason"`
	Timestamp int64                  `json:"timestamp"`
	ActorID   string                 `json:"actor_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type StateValidator interface {
	IsValid() bool
	CanTransitionTo(next interface{}) bool
	IsTerminal() bool
	String() string
}

type InvalidTransitionError struct {
	From   string
	To     string
	Reason string
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition: %s → %s (%s)", e.From, e.To, e.Reason)
}

func NewInvalidTransitionError(from, to, reason string) error {
	return &InvalidTransitionError{From: from, To: to, Reason: reason}
}

func ValidateTransition(from, to interface{}) error {
	switch fromState := from.(type) {
	case DecisionState:
		if toState, ok := to.(DecisionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(fromState.String(), toState.String(), "decision state transition not allowed")
			}
			return nil
		}
	case ApprovalState:
		if toState, ok := to.(ApprovalState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(fromState.String(), toState.String(), "approval state transition not allowed")
			}
			return nil
		}
	case ExecutionState:
		if toState, ok := to.(ExecutionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(fromState.String(), toState.String(), "execution state transition not allowed")
			}
			return nil
		}
	case TaskState:
		if toState, ok := to.(TaskState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(fromState.String(), toState.String(), "task state transition not allowed")
			}
			return nil
		}
	case SessionState:
		if toState, ok := to.(SessionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(fromState.String(), toState.String(), "session state transition not allowed")
			}
			return nil
		}
	case FactoryPhase:
		if toPhase, ok := to.(FactoryPhase); ok {
			if !fromState.CanTransitionTo(toPhase) {
				return NewInvalidTransitionError(fromState.String(), toPhase.String(), "factory phase transition not allowed")
			}
			return nil
		}
	}
	return NewInvalidTransitionError(fmt.Sprintf("%T", from), fmt.Sprintf("%T", to), "unsupported state type")
}
