package entities

import (
	"fmt"
	"time"
)

type Job struct {
	ID           string
	Type         string
	Status       string // pending, running, success, failed
	Input        map[string]interface{}
	Output       map[string]interface{}
	Error        string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Retries      int
	MaxRetries   int
	Priority     int
}

func NewJob(id, jobType string, input map[string]interface{}) *Job {
	return &Job{
		ID:        id,
		Type:      jobType,
		Status:    "pending",
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
	j.Status = "running"
	now := time.Now()
	j.StartedAt = &now
	return nil
}

func (j *Job) Complete(output map[string]interface{}) error {
	if !j.IsRunning() {
		return fmt.Errorf("cannot complete non-running job: status=%s", j.Status)
	}
	j.Status = "success"
	j.Output = output
	now := time.Now()
	j.CompletedAt = &now
	return nil
}

func (j *Job) Fail(err error) error {
	if j.IsCompleted() {
		return fmt.Errorf("cannot fail completed job: status=%s", j.Status)
	}
	j.Status = "failed"
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
	j.Retries++
	j.Status = "pending"
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
