package domain

import "fmt"

// ============================================================================
// State Machine Definitions - Explicit, Type-Safe, Observable
// ============================================================================
// Each state machine is immutable, testable, and self-documenting.
// All transitions are explicit. Invalid transitions panic with helpful messages.

// ============================================================================
// 1. FACTORY PHASE STATE MACHINE
// ============================================================================
// Controls the maturity level: Crawl (manual approval) → Walk (learned patterns)
// → Run (autonomous execution)

type FactoryPhase string

const (
	PhaseCrawl FactoryPhase = "crawl" // Manual approval on every decision
	PhaseWalk  FactoryPhase = "walk"  // Auto-approve >=90% confidence + learned patterns
	PhaseRun   FactoryPhase = "run"   // Autonomous execution >=70% confidence
)

func (p FactoryPhase) IsValid() bool {
	return p == PhaseCrawl || p == PhaseWalk || p == PhaseRun
}

func (p FactoryPhase) CanTransitionTo(next FactoryPhase) bool {
	// Forward progression only: crawl → walk → run
	switch p {
	case PhaseCrawl:
		return next == PhaseWalk || next == PhaseRun
	case PhaseWalk:
		return next == PhaseRun || next == PhaseCrawl // Can step back to crawl
	case PhaseRun:
		return next == PhaseWalk || next == PhaseCrawl // Can step back
	}
	return false
}

func (p FactoryPhase) String() string { return string(p) }

// ============================================================================
// 2. AGENT DECISION STATE MACHINE
// ============================================================================
// A decision progresses: pending approval → human action (approved/rejected) →
// outcome verification (correct/incorrect)

type DecisionState string

const (
	DecisionPending      DecisionState = "pending"       // Awaiting human review or auto-approval
	DecisionApproved     DecisionState = "approved"      // Human or auto-approved (confidence threshold)
	DecisionRejected     DecisionState = "rejected"      // Explicitly rejected by human
	DecisionCorrect      DecisionState = "correct"       // Verified correct in production
	DecisionIncorrect    DecisionState = "incorrect"     // Verified incorrect - learn from failure
)

func (s DecisionState) IsValid() bool {
	return s == DecisionPending || s == DecisionApproved || s == DecisionRejected ||
		s == DecisionCorrect || s == DecisionIncorrect
}

func (s DecisionState) CanTransitionTo(next DecisionState) bool {
	switch s {
	case DecisionPending:
		// From pending: human can approve/reject, or auto-approve based on confidence
		return next == DecisionApproved || next == DecisionRejected
	case DecisionApproved:
		// From approved: verify correctness in production
		return next == DecisionCorrect || next == DecisionIncorrect
	case DecisionRejected:
		// Rejected decisions don't progress (terminal state for this branch)
		return false
	case DecisionCorrect, DecisionIncorrect:
		// Both are terminal states
		return false
	}
	return false
}

func (s DecisionState) IsTerminal() bool {
	return s == DecisionCorrect || s == DecisionIncorrect || s == DecisionRejected
}

func (s DecisionState) String() string { return string(s) }

// ============================================================================
// 3. APPROVAL STATE MACHINE
// ============================================================================
// Simpler than decision state: pending → (approved/rejected) only

type ApprovalState string

const (
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalRejected ApprovalState = "rejected"
	ApprovalExpired  ApprovalState = "expired" // Auto-rejected after timeout
)

func (s ApprovalState) IsValid() bool {
	return s == ApprovalPending || s == ApprovalApproved || s == ApprovalRejected || s == ApprovalExpired
}

func (s ApprovalState) CanTransitionTo(next ApprovalState) bool {
	switch s {
	case ApprovalPending:
		// From pending: can be approved, rejected, or expire
		return next == ApprovalApproved || next == ApprovalRejected || next == ApprovalExpired
	case ApprovalApproved, ApprovalRejected, ApprovalExpired:
		// All terminal states
		return false
	}
	return false
}

func (s ApprovalState) IsTerminal() bool {
	return s == ApprovalApproved || s == ApprovalRejected || s == ApprovalExpired
}

func (s ApprovalState) String() string { return string(s) }

// ============================================================================
// 4. WORKFLOW EXECUTION STATE MACHINE
// ============================================================================
// A workflow execution lifecycle: queued → executing → (completed/failed)

type ExecutionState string

const (
	ExecutionQueued      ExecutionState = "queued"        // Waiting to start
	ExecutionExecuting   ExecutionState = "executing"     // Currently running
	ExecutionCompleted   ExecutionState = "completed"     // Finished successfully
	ExecutionFailed      ExecutionState = "failed"        // Finished with error
	ExecutionCancelled   ExecutionState = "cancelled"     // Cancelled by user
	ExecutionAwaitingGate ExecutionState = "awaiting_gate" // Blocked on approval gate
)

func (s ExecutionState) IsValid() bool {
	return s == ExecutionQueued || s == ExecutionExecuting || s == ExecutionCompleted ||
		s == ExecutionFailed || s == ExecutionCancelled || s == ExecutionAwaitingGate
}

func (s ExecutionState) CanTransitionTo(next ExecutionState) bool {
	switch s {
	case ExecutionQueued:
		// From queued: can start executing or be cancelled
		return next == ExecutionExecuting || next == ExecutionCancelled || next == ExecutionAwaitingGate
	case ExecutionExecuting:
		// From executing: can complete, fail, hit gate, or be cancelled
		return next == ExecutionCompleted || next == ExecutionFailed || next == ExecutionCancelled || next == ExecutionAwaitingGate
	case ExecutionAwaitingGate:
		// From approval gate: can resume executing, fail, or be cancelled
		return next == ExecutionExecuting || next == ExecutionFailed || next == ExecutionCancelled
	case ExecutionCompleted, ExecutionFailed, ExecutionCancelled:
		// Terminal states
		return false
	}
	return false
}

func (s ExecutionState) IsTerminal() bool {
	return s == ExecutionCompleted || s == ExecutionFailed || s == ExecutionCancelled
}

func (s ExecutionState) IsRunning() bool {
	return s == ExecutionExecuting || s == ExecutionAwaitingGate
}

func (s ExecutionState) String() string { return string(s) }

// ============================================================================
// 5. TASK STATE MACHINE
// ============================================================================
// Individual task lifecycle within a workflow

type TaskState string

const (
	TaskPending    TaskState = "pending"    // Not started
	TaskRunning    TaskState = "running"    // Currently executing
	TaskCompleted  TaskState = "completed"  // Finished successfully
	TaskFailed     TaskState = "failed"     // Failed (may retry)
	TaskRetrying   TaskState = "retrying"   // Retrying after failure
	TaskSkipped    TaskState = "skipped"    // Skipped (dependency failed)
	TaskTimedout   TaskState = "timedout"   // Exceeded timeout
)

func (s TaskState) IsValid() bool {
	return s == TaskPending || s == TaskRunning || s == TaskCompleted ||
		s == TaskFailed || s == TaskRetrying || s == TaskSkipped || s == TaskTimedout
}

func (s TaskState) CanTransitionTo(next TaskState) bool {
	switch s {
	case TaskPending:
		// From pending: can start running or be skipped
		return next == TaskRunning || next == TaskSkipped
	case TaskRunning:
		// From running: can complete, fail, timeout, or be skipped
		return next == TaskCompleted || next == TaskFailed || next == TaskTimedout || next == TaskSkipped
	case TaskFailed:
		// From failed: can retry or be skipped
		return next == TaskRetrying || next == TaskSkipped
	case TaskRetrying:
		// From retrying: can run again, fail again, or timeout
		return next == TaskRunning || next == TaskCompleted || next == TaskFailed || next == TaskTimedout
	case TaskCompleted, TaskSkipped, TaskTimedout:
		// Terminal states
		return false
	}
	return false
}

func (s TaskState) IsTerminal() bool {
	return s == TaskCompleted || s == TaskSkipped || s == TaskTimedout || s == TaskFailed
}

func (s TaskState) IsRetryable() bool {
	return s == TaskFailed
}

func (s TaskState) String() string { return string(s) }

// ============================================================================
// 6. SESSION STATE MACHINE
// ============================================================================
// High-level workflow session state across the factory assembly line

type SessionState string

const (
	SessionQueued         SessionState = "queued"           // Initial state
	SessionAnalyzing      SessionState = "analyzing"        // Analyzing requirements
	SessionPlanningAgents SessionState = "planning_agents"  // Selecting agents
	SessionExecuting      SessionState = "executing"        // Running agents
	SessionAwaitingGate   SessionState = "awaiting_gate"    // Blocked on approval gate
	SessionVerifying      SessionState = "verifying"        // Verifying output
	SessionCompleted      SessionState = "completed"        // Successfully completed
	SessionFailed         SessionState = "failed"           // Failed with error
	SessionCancelled      SessionState = "cancelled"        // Cancelled by user
)

func (s SessionState) IsValid() bool {
	return s == SessionQueued || s == SessionAnalyzing || s == SessionPlanningAgents ||
		s == SessionExecuting || s == SessionAwaitingGate || s == SessionVerifying ||
		s == SessionCompleted || s == SessionFailed || s == SessionCancelled
}

func (s SessionState) CanTransitionTo(next SessionState) bool {
	switch s {
	case SessionQueued:
		return next == SessionAnalyzing || next == SessionCancelled
	case SessionAnalyzing:
		return next == SessionPlanningAgents || next == SessionFailed || next == SessionCancelled
	case SessionPlanningAgents:
		return next == SessionExecuting || next == SessionFailed || next == SessionCancelled
	case SessionExecuting:
		return next == SessionAwaitingGate || next == SessionVerifying || next == SessionFailed || next == SessionCancelled
	case SessionAwaitingGate:
		return next == SessionExecuting || next == SessionFailed || next == SessionCancelled
	case SessionVerifying:
		return next == SessionCompleted || next == SessionFailed || next == SessionCancelled
	case SessionCompleted, SessionFailed, SessionCancelled:
		return false
	}
	return false
}

func (s SessionState) IsTerminal() bool {
	return s == SessionCompleted || s == SessionFailed || s == SessionCancelled
}

func (s SessionState) IsRunning() bool {
	return !s.IsTerminal() && s != SessionQueued
}

func (s SessionState) String() string { return string(s) }

// ============================================================================
// TRANSITION HELPER & ERROR TYPES
// ============================================================================

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

// ============================================================================
// OBSERVABLE STATE CHANGE EVENT (for logging, audit trails, webhooks)
// ============================================================================

type StateChangeEvent struct {
	Entity      string      // "decision", "approval", "execution", "task", "session"
	EntityID    string      // UUID
	OldState    string      // From state
	NewState    string      // To state
	Reason      string      // Why changed (e.g., "human_approved", "confidence_threshold", "timeout")
	Timestamp   int64       // Unix nanoseconds
	ActorID     string      // Who/what caused the change (user ID, agent name, system)
	Metadata    map[string]interface{} // Additional context
}

// ============================================================================
// STATE MACHINE VALIDATOR (use this before saving state changes to DB)
// ============================================================================

type StateValidator interface {
	IsValid() bool
	CanTransitionTo(next interface{}) bool
	IsTerminal() bool
	String() string
}

// ValidateTransition checks if a state transition is allowed
func ValidateTransition(from, to interface{}) error {
	switch fromState := from.(type) {
	case DecisionState:
		if toState, ok := to.(DecisionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(
					fromState.String(), toState.String(),
					"decision state transition not allowed",
				)
			}
			return nil
		}
	case ApprovalState:
		if toState, ok := to.(ApprovalState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(
					fromState.String(), toState.String(),
					"approval state transition not allowed",
				)
			}
			return nil
		}
	case ExecutionState:
		if toState, ok := to.(ExecutionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(
					fromState.String(), toState.String(),
					"execution state transition not allowed",
				)
			}
			return nil
		}
	case TaskState:
		if toState, ok := to.(TaskState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(
					fromState.String(), toState.String(),
					"task state transition not allowed",
				)
			}
			return nil
		}
	case SessionState:
		if toState, ok := to.(SessionState); ok {
			if !fromState.CanTransitionTo(toState) {
				return NewInvalidTransitionError(
					fromState.String(), toState.String(),
					"session state transition not allowed",
				)
			}
			return nil
		}
	case FactoryPhase:
		if toPhase, ok := to.(FactoryPhase); ok {
			if !fromState.CanTransitionTo(toPhase) {
				return NewInvalidTransitionError(
					fromState.String(), toPhase.String(),
					"factory phase transition not allowed",
				)
			}
			return nil
		}
	}
	return NewInvalidTransitionError(
		fmt.Sprintf("%T", from), fmt.Sprintf("%T", to),
		"unsupported state type",
	)
}
