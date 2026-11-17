package services

import "time"

type JobSubmittedEvent struct {
	JobID     string
	JobType   string
	Timestamp time.Time
}

func (e JobSubmittedEvent) EventType() string {
	return "job.submitted"
}

type JobStartedEvent struct {
	JobID     string
	Timestamp time.Time
}

func (e JobStartedEvent) EventType() string {
	return "job.started"
}

type JobCompletedEvent struct {
	JobID     string
	Success   bool
	Timestamp time.Time
}

func (e JobCompletedEvent) EventType() string {
	return "job.completed"
}

type WorkflowStartedEvent struct {
	WorkflowID string
	WorkflowType string
	Timestamp  time.Time
}

func (e WorkflowStartedEvent) EventType() string {
	return "workflow.started"
}

type WorkflowCompletedEvent struct {
	WorkflowID string
	Success    bool
	Score      float64
	Timestamp  time.Time
}

func (e WorkflowCompletedEvent) EventType() string {
	return "workflow.completed"
}
