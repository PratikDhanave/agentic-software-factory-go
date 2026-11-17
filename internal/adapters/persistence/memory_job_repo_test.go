package persistence

import (
	"context"
	"fmt"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

func TestInMemoryJobRepository_Save(t *testing.T) {
	repo := NewInMemoryJobRepository()
	job := entities.NewJob("job-1", "test", nil)

	err := repo.Save(context.Background(), job)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	retrieved, err := repo.GetByID(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != "job-1" {
		t.Errorf("expected job-1, got %s", retrieved.ID)
	}
}

func TestInMemoryJobRepository_SaveNil(t *testing.T) {
	repo := NewInMemoryJobRepository()

	err := repo.Save(context.Background(), nil)
	if err == nil {
		t.Error("expected error when saving nil")
	}
}

func TestInMemoryJobRepository_GetByStatus(t *testing.T) {
	repo := NewInMemoryJobRepository()

	job1 := entities.NewJob("job-1", "test", nil)
	job1.Status = "pending"
	job2 := entities.NewJob("job-2", "test", nil)
	job2.Status = "running"

	repo.Save(context.Background(), job1)
	repo.Save(context.Background(), job2)

	pending, _ := repo.GetByStatus(context.Background(), "pending")
	if len(pending) != 1 {
		t.Errorf("expected 1 pending job, got %d", len(pending))
	}

	running, _ := repo.GetByStatus(context.Background(), "running")
	if len(running) != 1 {
		t.Errorf("expected 1 running job, got %d", len(running))
	}
}

func TestInMemoryJobRepository_Update(t *testing.T) {
	repo := NewInMemoryJobRepository()
	job := entities.NewJob("job-1", "test", nil)
	repo.Save(context.Background(), job)

	job.Status = "success"
	err := repo.Update(context.Background(), job)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := repo.GetByID(context.Background(), "job-1")
	if retrieved.Status != "success" {
		t.Errorf("expected success, got %s", retrieved.Status)
	}
}

func TestInMemoryJobRepository_List(t *testing.T) {
	repo := NewInMemoryJobRepository()

	for i := 0; i < 15; i++ {
		job := entities.NewJob(fmt.Sprintf("job-%d", i), "test", nil)
		repo.Save(context.Background(), job)
	}

	jobs, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs, got %d", len(jobs))
	}
}
