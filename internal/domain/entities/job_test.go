package entities

import (
	"fmt"
	"testing"
)

func TestNewJob(t *testing.T) {
	input := map[string]interface{}{"issue": 1}
	job := NewJob("job-1", "issue-triage", input)

	if job.ID != "job-1" {
		t.Errorf("expected ID job-1, got %s", job.ID)
	}
	if job.Type != "issue-triage" {
		t.Errorf("expected type issue-triage, got %s", job.Type)
	}
	if job.Status != "pending" {
		t.Errorf("expected status pending, got %s", job.Status)
	}
}

func TestJobLifecycle(t *testing.T) {
	job := NewJob("job-1", "test", nil)

	if !job.IsPending() {
		t.Error("new job should be pending")
	}

	if err := job.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if !job.IsRunning() {
		t.Error("started job should be running")
	}

	output := map[string]interface{}{"result": "ok"}
	if err := job.Complete(output); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if !job.IsCompleted() {
		t.Error("completed job should be in completed state")
	}
	if job.Output["result"] != "ok" {
		t.Error("output not set correctly")
	}
}

func TestJobCannotStartTwice(t *testing.T) {
	job := NewJob("job-1", "test", nil)
	job.Start()

	if err := job.Start(); err == nil {
		t.Error("expected error when starting running job")
	}
}

func TestJobRetry(t *testing.T) {
	job := NewJob("job-1", "test", nil)
	job.MaxRetries = 3

	if err := job.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if err := job.Fail(fmt.Errorf("test error")); err != nil {
		t.Fatalf("fail failed: %v", err)
	}

	if !job.CanRetry() {
		t.Error("job should be retryable")
	}

	if err := job.Retry(); err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	if !job.IsPending() {
		t.Error("retried job should be pending again")
	}
	if job.Retries != 1 {
		t.Errorf("expected 1 retry, got %d", job.Retries)
	}
}

func TestJobMaxRetries(t *testing.T) {
	job := NewJob("job-1", "test", nil)
	job.MaxRetries = 1

	for i := 0; i < 2; i++ {
		job.Start()
		job.Fail(fmt.Errorf("test"))
		if i < 1 {
			job.Retry()
		}
	}

	if job.CanRetry() {
		t.Error("job should not be retryable after max retries")
	}
}

func TestJobValidation(t *testing.T) {
	tests := []struct {
		name    string
		job     *Job
		wantErr bool
	}{
		{
			name:    "valid job",
			job:     NewJob("job-1", "test", nil),
			wantErr: false,
		},
		{
			name: "missing ID",
			job: &Job{
				Type: "test",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			job: &Job{
				ID: "job-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
