package queue

import (
	"testing"
	"time"
)

func TestQueueSubmit(t *testing.T) {
	queue := NewQueue(2, 3)
	queue.Start()
	defer queue.Stop()

	job := &Job{
		ID:   "test-1",
		Type: "test",
		ExecutionFn: func(j *Job) error {
			j.Output = map[string]interface{}{"result": "success"}
			return nil
		},
	}

	if err := queue.Submit(job); err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	retrieved, ok := queue.Get("test-1")
	if !ok {
		t.Fatal("Job not found")
	}

	if retrieved.Status != "pending" {
		t.Errorf("Expected pending, got %s", retrieved.Status)
	}
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
