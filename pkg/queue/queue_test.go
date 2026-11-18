package queue

import (
	"testing"
	"time"
)

func TestQueueSubmit(t *testing.T) {
	queue := NewQueue(2, 3)

	job := &Job{
		ID:   "test-1",
		Type: "test",
		ExecutionFn: func(j *Job) error {
			j.Output = map[string]interface{}{"result": "success"}
			return nil
		},
	}

	// Submit before starting workers so initial pending state is deterministic.
	if err := queue.Submit(job); err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	retrieved, ok := queue.Get("test-1")
	if !ok {
		t.Fatal("Job not found")
	}
	if retrieved.Status != "pending" {
		t.Errorf("Expected pending before worker start, got %s", retrieved.Status)
	}

	queue.Start()
	defer queue.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, exists := queue.Get("test-1")
		if exists && current.Status == "success" {
			if current.Output["result"] != "success" {
				t.Fatalf("Expected output result=success, got %v", current.Output)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Timed out waiting for queued job to complete")
}

func TestQueueStats(t *testing.T) {
	queue := NewQueue(2, 3)
	stats := queue.Stats()

	total := stats["total"].(int)
	if total != 0 {
		t.Errorf("Expected 0 jobs, got %d", total)
	}
}

func TestScheduler(t *testing.T) {
	queue := NewQueue(2, 3)
	scheduler := NewScheduler(queue)
	scheduler.Start()
	defer scheduler.Stop()

	job := &Job{
		ID:   "scheduled-1",
		Type: "test",
		ExecutionFn: func(j *Job) error {
			return nil
		},
	}

	runAt := time.Now().Add(100 * time.Millisecond)
	if err := scheduler.Schedule("sched-1", runAt, job); err != nil {
		t.Fatalf("Failed to schedule: %v", err)
	}

	scheduled := scheduler.ListScheduled()
	if len(scheduled) != 1 {
		t.Errorf("Expected 1 scheduled job, got %d", len(scheduled))
	}
}
