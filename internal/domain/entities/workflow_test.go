package entities

import (
	"fmt"
	"testing"
)

func TestNewWorkflow(t *testing.T) {
	input := map[string]interface{}{"issue": 1}
	w := NewWorkflow("wf-1", "issue-triage", input)

	if w.ID != "wf-1" {
		t.Errorf("expected ID wf-1, got %s", w.ID)
	}
	if w.Type != "issue-triage" {
		t.Errorf("expected type issue-triage, got %s", w.Type)
	}
	if w.Status != "pending" {
		t.Errorf("expected status pending, got %s", w.Status)
	}
}

func TestWorkflowLifecycle(t *testing.T) {
	w := NewWorkflow("wf-1", "test", nil)

	if !w.IsPending() {
		t.Error("new workflow should be pending")
	}

	if err := w.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if !w.IsRunning() {
		t.Error("started workflow should be running")
	}

	output := map[string]interface{}{"result": "ok"}
	if err := w.Complete(output, 0.95); err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if !w.IsCompleted() {
		t.Error("completed workflow should be in completed state")
	}
	if w.Output["result"] != "ok" {
		t.Error("output not set correctly")
	}
	if w.Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", w.Score)
	}
}

func TestWorkflowCannotStartTwice(t *testing.T) {
	w := NewWorkflow("wf-1", "test", nil)
	w.Start()

	if err := w.Start(); err == nil {
		t.Error("expected error when starting running workflow")
	}
}

func TestWorkflowFail(t *testing.T) {
	w := NewWorkflow("wf-1", "test", nil)
	w.Start()

	testErr := fmt.Errorf("test error")
	if err := w.Fail(testErr); err != nil {
		t.Fatalf("fail failed: %v", err)
	}

	if w.Status != "failed" {
		t.Errorf("expected status failed, got %s", w.Status)
	}
	if w.Error != "test error" {
		t.Errorf("expected error message, got %s", w.Error)
	}
}

func TestWorkflowAddJob(t *testing.T) {
	w := NewWorkflow("wf-1", "test", nil)

	w.AddJob("job-1")
	w.AddJob("job-2")

	if len(w.JobIDs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(w.JobIDs))
	}
	if w.JobIDs[0] != "job-1" {
		t.Errorf("expected job-1, got %s", w.JobIDs[0])
	}
}

func TestWorkflowValidation(t *testing.T) {
	tests := []struct {
		name    string
		w       *Workflow
		wantErr bool
	}{
		{
			name:    "valid workflow",
			w:       NewWorkflow("wf-1", "test", nil),
			wantErr: false,
		},
		{
			name: "missing ID",
			w: &Workflow{
				Type: "test",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			w: &Workflow{
				ID: "wf-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.w.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
