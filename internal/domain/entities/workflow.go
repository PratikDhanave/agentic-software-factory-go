package entities

import (
	"fmt"
	"time"
)

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
		Status:    "pending",
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
	return w.Status == "success" || w.Status == "failed"
}

func (w *Workflow) Start() error {
	if !w.IsPending() {
		return fmt.Errorf("cannot start non-pending workflow: status=%s", w.Status)
	}
	w.Status = "running"
	now := time.Now()
	w.StartedAt = &now
	return nil
}

func (w *Workflow) Complete(output map[string]interface{}, score float64) error {
	if !w.IsRunning() {
		return fmt.Errorf("cannot complete non-running workflow: status=%s", w.Status)
	}
	w.Status = "success"
	w.Output = output
	w.Score = score
	now := time.Now()
	w.EndedAt = &now
	return nil
}

func (w *Workflow) Fail(err error) error {
	if w.IsCompleted() {
		return fmt.Errorf("cannot fail completed workflow: status=%s", w.Status)
	}
	w.Status = "failed"
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
	WorkflowDefCreated   WorkflowDefinitionStatus = "created"
	WorkflowDefActive    WorkflowDefinitionStatus = "active"
	WorkflowDefArchived  WorkflowDefinitionStatus = "archived"
)

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
	WorkflowExecPending    WorkflowExecutionStatus = "pending"
	WorkflowExecRunning    WorkflowExecutionStatus = "running"
	WorkflowExecCompleted  WorkflowExecutionStatus = "completed"
	WorkflowExecFailed     WorkflowExecutionStatus = "failed"
)

type WorkflowExecution struct {
	ID             string
	DefinitionID   string
	TriggerEvent   map[string]interface{}
	NodesMap       map[string]*DAGNode
	Status         WorkflowExecutionStatus
	Results        map[string]interface{}
	StartedAt      *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
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

type JobExecutionStatus string

const (
	JobExecPending    JobExecutionStatus = "pending"
	JobExecRunning    JobExecutionStatus = "running"
	JobExecCompleted  JobExecutionStatus = "completed"
	JobExecFailed     JobExecutionStatus = "failed"
)

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
	now := time.Now()
	n.StartedAt = &now
	n.Status = JobExecRunning
}

func (n *DAGNode) Complete(result interface{}) {
	now := time.Now()
	n.CompletedAt = &now
	n.Status = JobExecCompleted
	n.Result = result
	if n.StartedAt != nil {
		n.Duration = now.Sub(*n.StartedAt)
	}
}

func (n *DAGNode) Fail(err error) {
	now := time.Now()
	n.CompletedAt = &now
	n.Status = JobExecFailed
	errMsg := err.Error()
	n.Error = &errMsg
	if n.StartedAt != nil {
		n.Duration = now.Sub(*n.StartedAt)
	}
}
