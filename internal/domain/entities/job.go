package entities

import (
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// JobMachine explicitly governs background & webhook jobs with bounded retries.
var JobMachine = state.NewMachine("background-job", "pending", map[state.State][]state.State{
	"pending": {"running", "failed"},
	"running": {"success", "failed", "pending"},
	"failed":  {"pending"},
}).WithCategory("execution").
	WithDescription("Webhook and background job lifecycle supporting bounded backoff retries.").
	WithStateDescriptions(map[state.State]string{
		"pending": "Enqueued and waiting for a worker slot.",
		"running": "Actively executing job payload.",
		"success": "Completed cleanly (terminal).",
		"failed":  "Execution failed; transitions back to pending while retries remain.",
	})

// EntityMachines returns all explicit state machines declared in package entities.
func EntityMachines() []*state.Machine {
	return []*state.Machine{
		CoordinationSessionMachine,
		WorkflowDefinitionMachine,
		WorkflowExecutionMachine,
		DAGNodeMachine,
		JobMachine,
		WorkflowPipelineMachine,
	}
}

type Job struct {
	ID          string
	Type        string
	Status      string // pending, running, success, failed
	Input       map[string]interface{}
	Output      map[string]interface{}
	Error       string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Retries     int
	MaxRetries  int
	Priority    int
}

func NewJob(id, jobType string, input map[string]interface{}) *Job {
	return &Job{
		ID:        id,
		Type:      jobType,
		Status:    string(JobMachine.Initial()),
		Input:     input,
		CreatedAt: time.Now(),
	}
}

func (j *Job) IsRunning() bool {
	return j.Status == "running"
}

func (j *Job) IsPending() bool {
	return j.Status == "pending"
}

func (j *Job) IsCompleted() bool {
	return j.Status == "success" || j.Status == "failed"
}

func (j *Job) Start() error {
	if !j.IsPending() {
		return fmt.Errorf("cannot start non-pending job: status=%s", j.Status)
	}
	if err := JobMachine.TransitionString(&j.Status, "running"); err != nil {
		return err
	}
	now := time.Now()
	j.StartedAt = &now
	return nil
}

func (j *Job) Complete(output map[string]interface{}) error {
	if !j.IsRunning() {
		return fmt.Errorf("cannot complete non-running job: status=%s", j.Status)
	}
	if err := JobMachine.TransitionString(&j.Status, "success"); err != nil {
		return err
	}
	j.Output = output
	now := time.Now()
	j.CompletedAt = &now
	return nil
}

func (j *Job) Fail(err error) error {
	if j.IsCompleted() {
		return fmt.Errorf("cannot fail completed job: status=%s", j.Status)
	}
	if trErr := JobMachine.TransitionString(&j.Status, "failed"); trErr != nil {
		return trErr
	}
	if err != nil {
		j.Error = err.Error()
	}
	now := time.Now()
	j.CompletedAt = &now
	return nil
}

func (j *Job) CanRetry() bool {
	return j.Retries < j.MaxRetries
}

func (j *Job) Retry() error {
	if !j.CanRetry() {
		return fmt.Errorf("cannot retry: exceeded max retries (%d)", j.MaxRetries)
	}
	if j.Status == "failed" || j.Status == "running" {
		if err := JobMachine.TransitionString(&j.Status, "pending"); err != nil {
			return err
		}
	} else {
		j.Status = "pending"
	}
	j.Retries++
	j.StartedAt = nil
	j.CompletedAt = nil
	j.Error = ""
	return nil
}

func (j *Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job ID required")
	}
	if j.Type == "" {
		return fmt.Errorf("job type required")
	}
	if j.MaxRetries < 0 {
		j.MaxRetries = 3
	}
	return nil
}
