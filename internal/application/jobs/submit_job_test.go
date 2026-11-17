package jobs

import (
	"context"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type mockJobRepository struct {
	saved []*entities.Job
	err   error
}

func (m *mockJobRepository) Save(ctx context.Context, job *entities.Job) error {
	if m.err != nil {
		return m.err
	}
	m.saved = append(m.saved, job)
	return nil
}

func (m *mockJobRepository) GetByID(ctx context.Context, id string) (*entities.Job, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, j := range m.saved {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, nil
}

func (m *mockJobRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Job, error) {
	return nil, nil
}

func (m *mockJobRepository) Update(ctx context.Context, job *entities.Job) error {
	return nil
}

func (m *mockJobRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Job, error) {
	return nil, nil
}

type mockLogger struct{}

func (m *mockLogger) Info(msg string, args ...interface{})            {}
func (m *mockLogger) Error(msg string, err error, args ...interface{}) {}
func (m *mockLogger) Debug(msg string, args ...interface{})            {}
func (m *mockLogger) Warn(msg string, args ...interface{})             {}

type mockPublisher struct {
	events []services.Event
}

func (m *mockPublisher) Publish(event services.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockPublisher) Subscribe(eventType string, handler func(services.Event) error) error {
	return nil
}

func TestSubmitJobUseCase(t *testing.T) {
	repo := &mockJobRepository{}
	logger := &mockLogger{}
	pub := &mockPublisher{}

	uc := NewSubmitJobUseCase(repo, logger, pub)

	job := entities.NewJob("job-1", "test-job", map[string]interface{}{"test": "data"})

	err := uc.Execute(context.Background(), job)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(repo.saved) != 1 {
		t.Errorf("expected 1 saved job, got %d", len(repo.saved))
	}

	if len(pub.events) != 1 {
		t.Errorf("expected 1 published event, got %d", len(pub.events))
	}

	if pub.events[0].EventType() != "job.submitted" {
		t.Errorf("expected job.submitted event, got %s", pub.events[0].EventType())
	}
}

func TestSubmitJobUseCase_InvalidJob(t *testing.T) {
	repo := &mockJobRepository{}
	logger := &mockLogger{}
	pub := &mockPublisher{}

	uc := NewSubmitJobUseCase(repo, logger, pub)

	job := &entities.Job{} // Invalid (no ID)

	err := uc.Execute(context.Background(), job)

	if err == nil {
		t.Error("expected error for invalid job")
	}

	if len(repo.saved) != 0 {
		t.Errorf("expected 0 saved jobs, got %d", len(repo.saved))
	}
}
