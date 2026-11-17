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
