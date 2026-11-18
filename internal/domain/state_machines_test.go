package domain

import (
	"testing"
)

// ============================================================================
// TESTS FOR DECISION STATE MACHINE
// ============================================================================

func TestDecisionStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    DecisionState
		to      DecisionState
		allowed bool
	}{
		// Valid transitions from pending
		{"Pending → Approved", DecisionPending, DecisionApproved, true},
		{"Pending → Rejected", DecisionPending, DecisionRejected, true},

		// Invalid transitions from pending
		{"Pending → Correct (invalid - must go through Approved)", DecisionPending, DecisionCorrect, false},
		{"Pending → Incorrect (invalid)", DecisionPending, DecisionIncorrect, false},
		{"Pending → Pending (no-op invalid)", DecisionPending, DecisionPending, false},

		// Valid transitions from approved
		{"Approved → Correct", DecisionApproved, DecisionCorrect, true},
		{"Approved → Incorrect", DecisionApproved, DecisionIncorrect, true},

		// Invalid transitions from approved
		{"Approved → Pending (no reversal)", DecisionApproved, DecisionPending, false},
		{"Approved → Rejected (no reversal)", DecisionApproved, DecisionRejected, false},

		// No transitions from rejected (terminal)
		{"Rejected → Approved (no reversal)", DecisionRejected, DecisionApproved, false},
		{"Rejected → Pending (no reversal)", DecisionRejected, DecisionPending, false},

		// No transitions from terminal states
		{"Correct → Incorrect", DecisionCorrect, DecisionIncorrect, false},
		{"Correct → Pending", DecisionCorrect, DecisionPending, false},
		{"Incorrect → Correct", DecisionIncorrect, DecisionCorrect, false},
		{"Incorrect → Pending", DecisionIncorrect, DecisionPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestDecisionStateTerminal(t *testing.T) {
	tests := []struct {
		state      DecisionState
		isTerminal bool
	}{
		{DecisionPending, false},
		{DecisionApproved, false},
		{DecisionRejected, true},
		{DecisionCorrect, true},
		{DecisionIncorrect, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.isTerminal {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.state, got, tt.isTerminal)
			}
		})
	}
}

func TestDecisionStateValid(t *testing.T) {
	valid := []DecisionState{DecisionPending, DecisionApproved, DecisionRejected,
		DecisionCorrect, DecisionIncorrect}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("IsValid(%s) = false, want true", s)
		}
	}

	invalid := DecisionState("invalid_state")
	if invalid.IsValid() {
		t.Errorf("IsValid(invalid_state) = true, want false")
	}
}

// ============================================================================
// TESTS FOR APPROVAL STATE MACHINE
// ============================================================================

func TestApprovalStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    ApprovalState
		to      ApprovalState
		allowed bool
	}{
		// Valid transitions from pending
		{"Pending → Approved", ApprovalPending, ApprovalApproved, true},
		{"Pending → Rejected", ApprovalPending, ApprovalRejected, true},
		{"Pending → Expired", ApprovalPending, ApprovalExpired, true},

		// Invalid transitions from pending
		{"Pending → Pending (no-op)", ApprovalPending, ApprovalPending, false},

		// No transitions from terminal states
		{"Approved → Rejected", ApprovalApproved, ApprovalRejected, false},
		{"Rejected → Approved", ApprovalRejected, ApprovalApproved, false},
		{"Expired → Approved", ApprovalExpired, ApprovalApproved, false},
		{"Approved → Pending", ApprovalApproved, ApprovalPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestApprovalStateTerminal(t *testing.T) {
	tests := []struct {
		state      ApprovalState
		isTerminal bool
	}{
		{ApprovalPending, false},
		{ApprovalApproved, true},
		{ApprovalRejected, true},
		{ApprovalExpired, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.isTerminal {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.state, got, tt.isTerminal)
			}
		})
	}
}

// ============================================================================
// TESTS FOR EXECUTION STATE MACHINE
// ============================================================================

func TestExecutionStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    ExecutionState
		to      ExecutionState
		allowed bool
	}{
		// From queued
		{"Queued → Executing", ExecutionQueued, ExecutionExecuting, true},
		{"Queued → AwaitingGate", ExecutionQueued, ExecutionAwaitingGate, true},
		{"Queued → Cancelled", ExecutionQueued, ExecutionCancelled, true},
		{"Queued → Completed (invalid)", ExecutionQueued, ExecutionCompleted, false},
		{"Queued → Queued (no-op)", ExecutionQueued, ExecutionQueued, false},

		// From executing
		{"Executing → Completed", ExecutionExecuting, ExecutionCompleted, true},
		{"Executing → Failed", ExecutionExecuting, ExecutionFailed, true},
		{"Executing → AwaitingGate", ExecutionExecuting, ExecutionAwaitingGate, true},
		{"Executing → Cancelled", ExecutionExecuting, ExecutionCancelled, true},
		{"Executing → Executing (no-op)", ExecutionExecuting, ExecutionExecuting, false},

		// From awaiting gate
		{"AwaitingGate → Executing", ExecutionAwaitingGate, ExecutionExecuting, true},
		{"AwaitingGate → Failed", ExecutionAwaitingGate, ExecutionFailed, true},
		{"AwaitingGate → Cancelled", ExecutionAwaitingGate, ExecutionCancelled, true},
		{"AwaitingGate → Completed (invalid)", ExecutionAwaitingGate, ExecutionCompleted, false},

		// From terminal states
		{"Completed → Executing", ExecutionCompleted, ExecutionExecuting, false},
		{"Failed → Executing", ExecutionFailed, ExecutionExecuting, false},
		{"Cancelled → Executing", ExecutionCancelled, ExecutionExecuting, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestExecutionStateTerminal(t *testing.T) {
	tests := []struct {
		state      ExecutionState
		isTerminal bool
	}{
		{ExecutionQueued, false},
		{ExecutionExecuting, false},
		{ExecutionAwaitingGate, false},
		{ExecutionCompleted, true},
		{ExecutionFailed, true},
		{ExecutionCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.isTerminal {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.state, got, tt.isTerminal)
			}
		})
	}
}

func TestExecutionStateIsRunning(t *testing.T) {
	tests := []struct {
		state     ExecutionState
		isRunning bool
	}{
		{ExecutionQueued, false},
		{ExecutionExecuting, true},
		{ExecutionAwaitingGate, true},
		{ExecutionCompleted, false},
		{ExecutionFailed, false},
		{ExecutionCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsRunning(); got != tt.isRunning {
				t.Errorf("IsRunning(%s) = %v, want %v", tt.state, got, tt.isRunning)
			}
		})
	}
}

// ============================================================================
// TESTS FOR TASK STATE MACHINE
// ============================================================================

func TestTaskStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    TaskState
		to      TaskState
		allowed bool
	}{
		// From pending
		{"Pending → Running", TaskPending, TaskRunning, true},
		{"Pending → Skipped", TaskPending, TaskSkipped, true},
		{"Pending → Completed (invalid)", TaskPending, TaskCompleted, false},

		// From running
		{"Running → Completed", TaskRunning, TaskCompleted, true},
		{"Running → Failed", TaskRunning, TaskFailed, true},
		{"Running → Timedout", TaskRunning, TaskTimedout, true},
		{"Running → Skipped", TaskRunning, TaskSkipped, true},

		// From failed (retryable)
		{"Failed → Retrying", TaskFailed, TaskRetrying, true},
		{"Failed → Skipped", TaskFailed, TaskSkipped, true},
		{"Failed → Completed (invalid)", TaskFailed, TaskCompleted, false},

		// From retrying
		{"Retrying → Running", TaskRetrying, TaskRunning, true},
		{"Retrying → Completed", TaskRetrying, TaskCompleted, true},
		{"Retrying → Failed", TaskRetrying, TaskFailed, true},
		{"Retrying → Timedout", TaskRetrying, TaskTimedout, true},

		// From terminal states
		{"Completed → Running", TaskCompleted, TaskRunning, false},
		{"Timedout → Running", TaskTimedout, TaskRunning, false},
		{"Skipped → Running", TaskSkipped, TaskRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestTaskStateIsRetryable(t *testing.T) {
	tests := []struct {
		state      TaskState
		isRetryable bool
	}{
		{TaskPending, false},
		{TaskRunning, false},
		{TaskCompleted, false},
		{TaskFailed, true},
		{TaskRetrying, false},
		{TaskSkipped, false},
		{TaskTimedout, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsRetryable(); got != tt.isRetryable {
				t.Errorf("IsRetryable(%s) = %v, want %v", tt.state, got, tt.isRetryable)
			}
		})
	}
}

// ============================================================================
// TESTS FOR SESSION STATE MACHINE
// ============================================================================

func TestSessionStateTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    SessionState
		to      SessionState
		allowed bool
	}{
		// From queued
		{"Queued → Analyzing", SessionQueued, SessionAnalyzing, true},
		{"Queued → Cancelled", SessionQueued, SessionCancelled, true},
		{"Queued → Executing (skip stages invalid)", SessionQueued, SessionExecuting, false},

		// From analyzing
		{"Analyzing → PlanningAgents", SessionAnalyzing, SessionPlanningAgents, true},
		{"Analyzing → Failed", SessionAnalyzing, SessionFailed, true},

		// From planning agents
		{"PlanningAgents → Executing", SessionPlanningAgents, SessionExecuting, true},
		{"PlanningAgents → Failed", SessionPlanningAgents, SessionFailed, true},

		// From executing
		{"Executing → AwaitingGate", SessionExecuting, SessionAwaitingGate, true},
		{"Executing → Verifying", SessionExecuting, SessionVerifying, true},
		{"Executing → Failed", SessionExecuting, SessionFailed, true},

		// From awaiting gate
		{"AwaitingGate → Executing", SessionAwaitingGate, SessionExecuting, true},
		{"AwaitingGate → Failed", SessionAwaitingGate, SessionFailed, true},

		// From verifying
		{"Verifying → Completed", SessionVerifying, SessionCompleted, true},
		{"Verifying → Failed", SessionVerifying, SessionFailed, true},

		// From terminal states
		{"Completed → Executing", SessionCompleted, SessionExecuting, false},
		{"Failed → Executing", SessionFailed, SessionExecuting, false},
		{"Cancelled → Executing", SessionCancelled, SessionExecuting, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestSessionStateIsRunning(t *testing.T) {
	tests := []struct {
		state     SessionState
		isRunning bool
	}{
		{SessionQueued, false},
		{SessionAnalyzing, true},
		{SessionPlanningAgents, true},
		{SessionExecuting, true},
		{SessionAwaitingGate, true},
		{SessionVerifying, true},
		{SessionCompleted, false},
		{SessionFailed, false},
		{SessionCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsRunning(); got != tt.isRunning {
				t.Errorf("IsRunning(%s) = %v, want %v", tt.state, got, tt.isRunning)
			}
		})
	}
}

// ============================================================================
// TESTS FOR FACTORY PHASE STATE MACHINE
// ============================================================================

func TestFactoryPhaseTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    FactoryPhase
		to      FactoryPhase
		allowed bool
	}{
		// Forward progression
		{"Crawl → Walk", PhaseCrawl, PhaseWalk, true},
		{"Crawl → Run", PhaseCrawl, PhaseRun, true},
		{"Walk → Run", PhaseWalk, PhaseRun, true},

		// Backtracking
		{"Walk → Crawl", PhaseWalk, PhaseCrawl, true},
		{"Run → Walk", PhaseRun, PhaseWalk, true},
		{"Run → Crawl", PhaseRun, PhaseCrawl, true},

		// No-ops
		{"Crawl → Crawl", PhaseCrawl, PhaseCrawl, false},
		{"Walk → Walk", PhaseWalk, PhaseWalk, false},
		{"Run → Run", PhaseRun, PhaseRun, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := tt.from.CanTransitionTo(tt.to)
			if allowed != tt.allowed {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, allowed, tt.allowed)
			}
		})
	}
}

func TestFactoryPhaseValid(t *testing.T) {
	valid := []FactoryPhase{PhaseCrawl, PhaseWalk, PhaseRun}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("IsValid(%s) = false, want true", p)
		}
	}

	invalid := FactoryPhase("invalid_phase")
	if invalid.IsValid() {
		t.Errorf("IsValid(invalid_phase) = true, want false")
	}
}

// ============================================================================
// TESTS FOR TRANSITION VALIDATION
// ============================================================================

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    interface{}
		to      interface{}
		wantErr bool
	}{
		// Valid transitions
		{"Decision: Pending → Approved", DecisionPending, DecisionApproved, false},
		{"Approval: Pending → Approved", ApprovalPending, ApprovalApproved, false},
		{"Execution: Queued → Executing", ExecutionQueued, ExecutionExecuting, false},

		// Invalid transitions
		{"Decision: Pending → Correct", DecisionPending, DecisionCorrect, true},
		{"Approval: Approved → Pending", ApprovalApproved, ApprovalPending, true},
		{"Execution: Completed → Executing", ExecutionCompleted, ExecutionExecuting, true},

		// Type mismatch
		{"Type mismatch", DecisionPending, "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// TESTS FOR STATE MACHINE PATHS (Complex Scenarios)
// ============================================================================

func TestCompleteDecisionPath(t *testing.T) {
	// Happy path: pending → approved → correct
	path := []DecisionState{
		DecisionPending,
		DecisionApproved,
		DecisionCorrect,
	}

	for i := 0; i < len(path)-1; i++ {
		if !path[i].CanTransitionTo(path[i+1]) {
			t.Errorf("Cannot transition from %s to %s", path[i], path[i+1])
		}
	}
}

func TestRejectionPath(t *testing.T) {
	// Rejection path: pending → rejected (no feedback on outcome)
	path := []DecisionState{
		DecisionPending,
		DecisionRejected,
	}

	if !path[0].CanTransitionTo(path[1]) {
		t.Errorf("Cannot reject pending decision")
	}

	// Rejected decisions don't transition further
	if path[1].CanTransitionTo(DecisionCorrect) {
		t.Errorf("Rejected decision should not transition to correct")
	}
}

func TestExecutionWithGate(t *testing.T) {
	// Path through approval gate: queued → executing → awaiting_gate → executing → completed
	path := []ExecutionState{
		ExecutionQueued,
		ExecutionExecuting,
		ExecutionAwaitingGate,
		ExecutionExecuting,
		ExecutionCompleted,
	}

	for i := 0; i < len(path)-1; i++ {
		if !path[i].CanTransitionTo(path[i+1]) {
			t.Errorf("Cannot transition from %s to %s", path[i], path[i+1])
		}
	}
}
