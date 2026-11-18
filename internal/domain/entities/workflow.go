package entities

import (
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// WorkflowPipelineMachine explicitly governs the Workflow entity.
var WorkflowPipelineMachine = state.NewMachine("workflow-pipeline", "pending", map[state.State][]state.State{
	"pending": {"running", "failed"},
	"running": {"success", "failed"},
}).WithCategory("execution").
	WithDescription("High-level workflow run lifecycle (pending -> running -> success/failed).")

type Workflow struct {
	ID        string
	Type      string
	Status    string
	Input     map[string]interface{}
	Output    map[string]interface{}
	Error     string
	CreatedAt time.Time
	StartedAt *time.Time
	EndedAt   *time.Time
	JobIDs    []string
	Score     float64
}

func NewWorkflow(id, workflowType string, input map[string]interface{}) *Workflow {
	return &Workflow{
		ID:        id,
		Type:      workflowType,
		Status:    string(WorkflowPipelineMachine.Initial()),
		Input:     input,
		CreatedAt: time.Now(),
		JobIDs:    []string{},
	}
}

func (w *Workflow) IsRunning() bool {
	return w.Status == "running"
}

func (w *Workflow) IsPending() bool {
	return w.Status == "pending"
}

func (w *Workflow) IsCompleted() bool {
	return WorkflowPipelineMachine.IsTerminal(state.State(w.Status))
}

func (w *Workflow) Start() error {
	if err := WorkflowPipelineMachine.TransitionString(&w.Status, "running"); err != nil {
		return fmt.Errorf("cannot start non-pending workflow: status=%s (%w)", w.Status, err)
	}
	now := time.Now()
	w.StartedAt = &now
	return nil
}

func (w *Workflow) Complete(output map[string]interface{}, score float64) error {
	if err := WorkflowPipelineMachine.TransitionString(&w.Status, "success"); err != nil {
		return fmt.Errorf("cannot complete non-running workflow: status=%s (%w)", w.Status, err)
	}
	w.Output = output
	w.Score = score
	now := time.Now()
	w.EndedAt = &now
	return nil
}

func (w *Workflow) Fail(err error) error {
	if trErr := WorkflowPipelineMachine.TransitionString(&w.Status, "failed"); trErr != nil {
		return fmt.Errorf("cannot fail completed workflow: status=%s (%w)", w.Status, trErr)
	}
	if err != nil {
		w.Error = err.Error()
	}
	now := time.Now()
	w.EndedAt = &now
	return nil
}

func (w *Workflow) AddJob(jobID string) {
	w.JobIDs = append(w.JobIDs, jobID)
}

func (w *Workflow) Validate() error {
	if w.ID == "" {
		return fmt.Errorf("workflow ID required")
	}
	if w.Type == "" {
		return fmt.Errorf("workflow type required")
	}
	return nil
}

type WorkflowDefinitionStatus string

const (
	WorkflowDefCreated  WorkflowDefinitionStatus = "created"
	WorkflowDefActive   WorkflowDefinitionStatus = "active"
	WorkflowDefArchived WorkflowDefinitionStatus = "archived"
)

// WorkflowDefinitionMachine governs DAG blueprint definitions stored in SQLite.
var WorkflowDefinitionMachine = state.NewMachine("workflow-definition", state.State(WorkflowDefCreated), map[state.State][]state.State{
	state.State(WorkflowDefCreated): {state.State(WorkflowDefActive), state.State(WorkflowDefArchived)},
	state.State(WorkflowDefActive):  {state.State(WorkflowDefArchived)},
}).WithCategory("orchestration").
	WithDescription("Lifecycle of a synthesized multi-agent DAG workflow definition.").
	WithStateDescriptions(map[state.State]string{
		state.State(WorkflowDefCreated):  "Synthesized and validated; ready for activation.",
		state.State(WorkflowDefActive):   "Active blueprint eligible for DAG execution.",
		state.State(WorkflowDefArchived): "Superseded or retired workflow definition.",
	})

type WorkflowDefinition struct {
	ID          string
	Name        string
	Version     string
	Description string
	JobsMap     map[string]*JobDef
	Status      WorkflowDefinitionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type JobDef struct {
	Task      string
	DependsOn []string
	Input     map[string]interface{}
	Timeout   int
}

func NewWorkflowDefinition(id, name, version string, jobs map[string]*JobDef) *WorkflowDefinition {
	return &WorkflowDefinition{
		ID:        id,
		Name:      name,
		Version:   version,
		JobsMap:   jobs,
		Status:    WorkflowDefCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (wd *WorkflowDefinition) Activate() error {
	cur := state.State(wd.Status)
	if err := WorkflowDefinitionMachine.Transition(&cur, state.State(WorkflowDefActive)); err != nil {
		return err
	}
	wd.Status = WorkflowDefinitionStatus(cur)
	wd.UpdatedAt = time.Now()
	return nil
}

func (wd *WorkflowDefinition) Archive() error {
	cur := state.State(wd.Status)
	if err := WorkflowDefinitionMachine.Transition(&cur, state.State(WorkflowDefArchived)); err != nil {
		return err
	}
	wd.Status = WorkflowDefinitionStatus(cur)
	wd.UpdatedAt = time.Now()
	return nil
}

func (wd *WorkflowDefinition) Validate() error {
	if wd.Name == "" {
		return fmt.Errorf("workflow name required")
	}
	if wd.Version == "" {
		return fmt.Errorf("workflow version required")
	}
	if len(wd.JobsMap) == 0 {
		return fmt.Errorf("workflow must have at least one job")
	}

	for jobID, job := range wd.JobsMap {
		if job.Task == "" {
			return fmt.Errorf("job %s task cannot be empty", jobID)
		}
		for _, dep := range job.DependsOn {
			if _, exists := wd.JobsMap[dep]; !exists {
				return fmt.Errorf("job %s depends on non-existent job %s", jobID, dep)
			}
		}
	}
	return nil
}

type WorkflowExecutionStatus string

const (
	WorkflowExecPending   WorkflowExecutionStatus = "pending"
	WorkflowExecRunning   WorkflowExecutionStatus = "running"
	WorkflowExecCompleted WorkflowExecutionStatus = "completed"
	WorkflowExecFailed    WorkflowExecutionStatus = "failed"
)

// WorkflowExecutionMachine explicitly governs level-parallel DAG runs persisted in SQLite.
var WorkflowExecutionMachine = state.NewMachine("workflow-execution", state.State(WorkflowExecPending), map[state.State][]state.State{
	state.State(WorkflowExecPending): {state.State(WorkflowExecRunning), state.State(WorkflowExecFailed)},
	state.State(WorkflowExecRunning): {state.State(WorkflowExecCompleted), state.State(WorkflowExecFailed)},
}).WithCategory("orchestration").
	WithDescription("Level-parallel topological DAG execution run persisted in SQLite.").
	WithStateDescriptions(map[state.State]string{
		state.State(WorkflowExecPending):   "DAG nodes initialized and awaiting topological dispatch.",
		state.State(WorkflowExecRunning):   "Executing topological DAG levels concurrently.",
		state.State(WorkflowExecCompleted): "Every DAG node completed without error.",
		state.State(WorkflowExecFailed):    "One or more DAG nodes failed or timed out.",
	})

type WorkflowExecution struct {
	ID           string
	DefinitionID string
	TriggerEvent map[string]interface{}
	NodesMap     map[string]*DAGNode
	Status       WorkflowExecutionStatus
	Results      map[string]interface{}
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
}

func NewWorkflowExecution(id, defID string, event map[string]interface{}) *WorkflowExecution {
	return &WorkflowExecution{
		ID:           id,
		DefinitionID: defID,
		TriggerEvent: event,
		NodesMap:     make(map[string]*DAGNode),
		Status:       WorkflowExecPending,
		Results:      make(map[string]interface{}),
		CreatedAt:    time.Now(),
	}
}

func (we *WorkflowExecution) TransitionTo(to WorkflowExecutionStatus) error {
	if we.Status == to {
		return nil
	}
	cur := state.State(we.Status)
	if err := WorkflowExecutionMachine.Transition(&cur, state.State(to)); err != nil {
		return err
	}
	we.Status = WorkflowExecutionStatus(cur)
	now := time.Now()
	if to == WorkflowExecRunning && we.StartedAt == nil {
		we.StartedAt = &now
	}
	if WorkflowExecutionMachine.IsTerminal(cur) {
		we.CompletedAt = &now
	}
	return nil
}

type JobExecutionStatus string

const (
	JobExecPending   JobExecutionStatus = "pending"
	JobExecRunning   JobExecutionStatus = "running"
	JobExecCompleted JobExecutionStatus = "completed"
	JobExecFailed    JobExecutionStatus = "failed"
	JobExecSkipped   JobExecutionStatus = "skipped"
)

// DAGNodeMachine governs individual nodes inside a topological DAG execution.
var DAGNodeMachine = state.NewMachine("dag-node", state.State(JobExecPending), map[state.State][]state.State{
	state.State(JobExecPending): {state.State(JobExecRunning), state.State(JobExecSkipped)},
	state.State(JobExecRunning): {state.State(JobExecCompleted), state.State(JobExecFailed)},
}).WithCategory("orchestration").
	WithDescription("Individual node execution inside a topological multi-agent DAG.").
	WithStateDescriptions(map[state.State]string{
		state.State(JobExecPending):   "Waiting for parent DAG nodes in earlier levels to finish.",
		state.State(JobExecRunning):   "Agent/tool handler actively executing.",
		state.State(JobExecCompleted): "Node succeeded and published output to downstream dependents.",
		state.State(JobExecFailed):    "Node returned an error or exceeded timeout.",
		state.State(JobExecSkipped):   "Skipped due to upstream dependency failure.",
	})

type DAGNode struct {
	JobID       string
	TaskName    string
	DependsOn   []string
	Input       map[string]interface{}
	Status      JobExecutionStatus
	Result      interface{}
	Error       *string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    time.Duration
}

func NewDAGNode(jobID, taskName string, dependsOn []string, input map[string]interface{}) *DAGNode {
	return &DAGNode{
		JobID:     jobID,
		TaskName:  taskName,
		DependsOn: dependsOn,
		Input:     input,
		Status:    JobExecPending,
	}
}

func (n *DAGNode) Start() {
	cur := state.State(n.Status)
	if err := DAGNodeMachine.Transition(&cur, state.State(JobExecRunning)); err == nil {
		now := time.Now()
		n.StartedAt = &now
		n.Status = JobExecutionStatus(cur)
	}
}

func (n *DAGNode) Complete(result interface{}) {
	cur := state.State(n.Status)
	if err := DAGNodeMachine.Transition(&cur, state.State(JobExecCompleted)); err == nil {
		now := time.Now()
		n.CompletedAt = &now
		n.Status = JobExecutionStatus(cur)
		n.Result = result
		if n.StartedAt != nil {
			n.Duration = now.Sub(*n.StartedAt)
		}
	}
}

func (n *DAGNode) Fail(err error) {
	cur := state.State(n.Status)
	if trErr := DAGNodeMachine.Transition(&cur, state.State(JobExecFailed)); trErr == nil {
		now := time.Now()
		n.CompletedAt = &now
		n.Status = JobExecutionStatus(cur)
		if err != nil {
			errMsg := err.Error()
			n.Error = &errMsg
		}
		if n.StartedAt != nil {
			n.Duration = now.Sub(*n.StartedAt)
		}
	}
}
