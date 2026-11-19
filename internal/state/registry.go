package state

// 1. Autonomous Factory Run Lifecycle
const (
	RunQueued           State = "queued"
	RunPlanning         State = "planning"
	RunAwaitingApproval State = "awaiting_approval"
	RunSynthesizing     State = "synthesizing"
	RunVerifying        State = "verifying"
	RunCompleted        State = "completed"
	RunFailed           State = "failed"
)

var FactoryRunMachine = NewMachine("factory-run", RunQueued, map[State][]State{
	RunQueued:           {RunPlanning, RunFailed},
	RunPlanning:         {RunAwaitingApproval, RunSynthesizing, RunFailed},
	RunAwaitingApproval: {RunSynthesizing, RunFailed},
	RunSynthesizing:     {RunVerifying, RunFailed},
	RunVerifying:        {RunCompleted, RunSynthesizing, RunFailed},
}).WithCategory("assembly").
	WithDescription("End-to-end autonomous Go software build lifecycle with human checkpoint and self-healing retry loop.").
	WithStateDescriptions(map[State]string{
		RunQueued:           "Run persisted in SQLite and awaiting dispatch.",
		RunPlanning:         "Agents analyzing specification invariants and architecture.",
		RunAwaitingApproval: "Paused at Human-in-the-Loop checkpoint awaiting operator approval.",
		RunSynthesizing:     "Generating multi-file Go source and unit tests in an isolated module.",
		RunVerifying:        "Running go build, go vet, and go test -race (loops back to synthesizing on error).",
		RunCompleted:        "Verified package passed all compiler, race, and quality gates.",
		RunFailed:           "Terminated after exhausting self-healing retries or operator rejection.",
	})

// 2. Autonomy Mode Machine (Crawl <-> Walk <-> Run)
const (
	AutonomyCrawl State = "crawl"
	AutonomyWalk  State = "walk"
	AutonomyRun   State = "run"
)

var AutonomyMachine = NewMachine("autonomy-mode", AutonomyWalk, map[State][]State{
	AutonomyCrawl: {AutonomyWalk, AutonomyRun},
	AutonomyWalk:  {AutonomyCrawl, AutonomyRun},
	AutonomyRun:   {AutonomyCrawl, AutonomyWalk},
}).WithCategory("governance").
	WithDescription("Controls human-in-the-loop checkpoints: Crawl (supervised), Walk (balanced), Run (autonomous).").
	WithStateDescriptions(map[State]string{
		AutonomyCrawl: "Supervised: every run pauses at awaiting_approval before writing code.",
		AutonomyWalk:  "Balanced: standard blueprint runs proceed automatically unless flagged.",
		AutonomyRun:   "Autonomous: end-to-end execution without manual gates.",
	})

// 3. Human Approval Gate Machine
const (
	ApprovalPending  State = "pending"
	ApprovalApproved State = "approved"
	ApprovalRejected State = "rejected"
)

var ApprovalMachine = NewMachine("approval-gate", ApprovalPending, map[State][]State{
	ApprovalPending: {ApprovalApproved, ApprovalRejected},
}).WithCategory("governance").
	WithDescription("Human review decision for runs paused in awaiting_approval.").
	WithStateDescriptions(map[State]string{
		ApprovalPending:  "Waiting for operator review in the Control Plane.",
		ApprovalApproved: "Approved by operator; resumes sandbox code synthesis.",
		ApprovalRejected: "Rejected by operator; transitions run to failed.",
	})

// 4. Sandbox Verification Pipeline Machine
const (
	VerifyBuild       State = "build"
	VerifyVet         State = "vet"
	VerifyTestRace    State = "test_race"
	VerifyConstraints State = "constraints"
	VerifyPassed      State = "passed"
	VerifyFailed      State = "failed"
)

var SandboxVerifyMachine = NewMachine("sandbox-verify", VerifyBuild, map[State][]State{
	VerifyBuild:       {VerifyVet, VerifyFailed},
	VerifyVet:         {VerifyTestRace, VerifyFailed},
	VerifyTestRace:    {VerifyConstraints, VerifyPassed, VerifyFailed},
	VerifyConstraints: {VerifyPassed, VerifyFailed},
}).WithCategory("verification").
	WithDescription("Sequential Go compiler, go vet static analysis, go test -race, and spec invariant pipeline.").
	WithStateDescriptions(map[State]string{
		VerifyBuild:       "Compiling isolated Go module with go build -buildvcs=false ./...",
		VerifyVet:         "Running static analysis with go vet -buildvcs=false ./...",
		VerifyTestRace:    "Executing unit & concurrency suite with go test -v -race ./...",
		VerifyConstraints: "Checking MUST and MUST_NOT specification constraints.",
		VerifyPassed:      "All verification gates succeeded.",
		VerifyFailed:      "Verification failed with compiler/test diagnostics for self-healing.",
	})

// 5. Artifact Promotion Machine
const (
	PromoteNone      State = "unpromoted"
	PromoteRunning   State = "promoting"
	PromoteCompleted State = "promoted"
	PromoteFailed    State = "failed"
)

var PromotionMachine = NewMachine("artifact-promotion", PromoteNone, map[State][]State{
	PromoteNone:      {PromoteRunning},
	PromoteRunning:   {PromoteCompleted, PromoteFailed},
	PromoteCompleted: {PromoteRunning},
	PromoteFailed:    {PromoteRunning},
}).WithCategory("assembly").
	WithDescription("Exports verified Go files from .asf/workspaces/<run> into generated/<package>/.").
	WithStateDescriptions(map[State]string{
		PromoteNone:      "Verified in sandbox; not yet exported to generated/.",
		PromoteRunning:   "Validating path boundaries and writing .go files.",
		PromoteCompleted: "Exported cleanly into generated/<package>/.",
		PromoteFailed:    "Export failed; retryable.",
	})

// 6. Registered Agent Lifecycle Machine
const (
	AgentHealthy  State = "healthy"
	AgentBusy     State = "busy"
	AgentDegraded State = "degraded"
	AgentDisabled State = "disabled"
)

var AgentStatusMachine = NewMachine("agent-status", AgentHealthy, map[State][]State{
	AgentHealthy:  {AgentBusy, AgentDegraded, AgentDisabled},
	AgentBusy:     {AgentHealthy, AgentDegraded},
	AgentDegraded: {AgentHealthy, AgentDisabled},
	AgentDisabled: {AgentHealthy},
}).WithCategory("execution").
	WithDescription("Operational health and lease state of registered Factory Agent types.").
	WithStateDescriptions(map[State]string{
		AgentHealthy:  "Registered and ready to accept runs or consultations.",
		AgentBusy:     "Actively executing a factory run step.",
		AgentDegraded: "Recent model gateway call returned a transient error.",
		AgentDisabled: "Paused by operator.",
	})

// 7. Model Gateway Provider Machine
const (
	ProviderUnconfigured State = "unconfigured"
	ProviderStandby      State = "standby"
	ProviderActive       State = "active"
	ProviderDegraded     State = "degraded"
)

var GatewayProviderMachine = NewMachine("gateway-provider", ProviderUnconfigured, map[State][]State{
	ProviderUnconfigured: {ProviderStandby, ProviderActive},
	ProviderStandby:      {ProviderActive, ProviderUnconfigured},
	ProviderActive:       {ProviderStandby, ProviderDegraded, ProviderUnconfigured},
	ProviderDegraded:     {ProviderActive, ProviderStandby},
}).WithCategory("infrastructure").
	WithDescription("Model Gateway provider slot lifecycle across Gemini, Claude, OpenAI, Ollama, and Offline Synth.").
	WithStateDescriptions(map[State]string{
		ProviderUnconfigured: "No API key configured yet.",
		ProviderStandby:      "Credentials configured and ready as fallback or manual selection.",
		ProviderActive:       "Selected primary inference provider.",
		ProviderDegraded:     "Rate-limited or unreachable; failing over to next standby provider.",
	})

// 8. Webhook & Event Trigger Machine
const (
	TriggerReceived   State = "received"
	TriggerDispatched State = "dispatched"
	TriggerCompleted  State = "completed"
	TriggerIgnored    State = "ignored"
	TriggerFailed     State = "failed"
)

var TriggerMachine = NewMachine("trigger-delivery", TriggerReceived, map[State][]State{
	TriggerReceived:   {TriggerDispatched, TriggerIgnored, TriggerFailed},
	TriggerDispatched: {TriggerCompleted, TriggerFailed},
}).WithCategory("orchestration").
	WithDescription("Inbound GitHub webhook and manual event trigger lifecycle.").
	WithStateDescriptions(map[State]string{
		TriggerReceived:   "Webhook/event payload received and signature checked.",
		TriggerDispatched: "Matched trigger rule and launched a Factory Run.",
		TriggerCompleted:  "Triggered run finished.",
		TriggerIgnored:    "Event did not match an active trigger rule.",
		TriggerFailed:     "Invalid payload or failed run dispatch.",
	})

func All() []*Machine {
	return []*Machine{
		FactoryRunMachine,
		AutonomyMachine,
		ApprovalMachine,
		SandboxVerifyMachine,
		PromotionMachine,
		AgentStatusMachine,
		GatewayProviderMachine,
		TriggerMachine,
	}
}

func Exports() []Export {
	all := All()
	out := make([]Export, 0, len(all))
	for _, m := range all {
		out = append(out, m.Export())
	}
	return out
}

func ByName(name string) (*Machine, bool) {
	for _, m := range All() {
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}
