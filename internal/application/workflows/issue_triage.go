package workflows

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type TriageResult struct {
	IssueName string
	Priority  string
	Category  string
	Labels    []string
	Confidence float64
}

type IssueTriageUseCase struct {
	workflowRepo repositories.WorkflowRepository
	logger       services.Logger
	publisher    services.EventPublisher
}

func NewIssueTriageUseCase(
	workflowRepo repositories.WorkflowRepository,
	logger services.Logger,
	publisher services.EventPublisher,
) *IssueTriageUseCase {
	return &IssueTriageUseCase{
		workflowRepo: workflowRepo,
		logger:       logger,
		publisher:    publisher,
	}
}

func (u *IssueTriageUseCase) Execute(ctx context.Context, workflow *entities.Workflow) (*TriageResult, error) {
	if err := workflow.Validate(); err != nil {
		u.logger.Error("invalid workflow", err)
		return nil, err
	}

	if err := workflow.Start(); err != nil {
		u.logger.Error("failed to start workflow", err)
		return nil, err
	}

	event := services.WorkflowStartedEvent{
		WorkflowID:   workflow.ID,
		WorkflowType: workflow.Type,
	}
	u.publisher.Publish(event)

	result := u.triageIssue(workflow.Input)

	if err := workflow.Complete(map[string]interface{}{
		"priority": result.Priority,
		"category": result.Category,
		"labels":   result.Labels,
	}, result.Confidence); err != nil {
		u.logger.Error("failed to complete workflow", err)
		return nil, err
	}

	publishEvent := services.WorkflowCompletedEvent{
		WorkflowID: workflow.ID,
		Success:    true,
		Score:      result.Confidence,
	}
	u.publisher.Publish(publishEvent)

	u.logger.Info("issue triaged", "id", workflow.ID, "priority", result.Priority)
	return result, nil
}

func (u *IssueTriageUseCase) triageIssue(input map[string]interface{}) *TriageResult {
	result := &TriageResult{
		Confidence: 0.85,
		Labels:     []string{},
	}

	if title, ok := input["title"].(string); ok {
		result.IssueName = title
	}

	result.Priority = "medium"
	result.Category = "feature"
	result.Labels = []string{"triage-needed", "review"}

	return result
}
