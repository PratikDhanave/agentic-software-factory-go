package usecases

import (
	"context"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type CoordinateWorkflowInput struct {
	TriggerEvent     map[string]interface{}
	TriggerType      string
	Classification   string
	ApprovalRequired bool
}

type CoordinateWorkflowOutput struct {
	Session          *entities.CoordinationSession
	Workflow         *entities.WorkflowDefinition
	Decisions        map[string]*entities.Decision
	RequiresApproval bool
	ApprovedBy       string
}

type CoordinateWorkflowUseCase struct {
	workflowGenerator *services.WorkflowGenerator
	knowledgeService  *services.KnowledgeGraphService
	agentRegistry     *services.AgentRegistry
	workflowRepo      repositories.WorkflowRepository
	sessionRepo       repositories.CoordinationSessionRepository
	logger            services.Logger
}

func NewCoordinateWorkflowUseCase(
	workflowGenerator *services.WorkflowGenerator,
	knowledgeService *services.KnowledgeGraphService,
	agentRegistry *services.AgentRegistry,
	workflowRepo repositories.WorkflowRepository,
	logger services.Logger,
) *CoordinateWorkflowUseCase {
	return &CoordinateWorkflowUseCase{
		workflowGenerator: workflowGenerator,
		knowledgeService:  knowledgeService,
		agentRegistry:     agentRegistry,
		workflowRepo:      workflowRepo,
		logger:            logger,
	}
}

func (uc *CoordinateWorkflowUseCase) WithSessionRepository(repo repositories.CoordinationSessionRepository) *CoordinateWorkflowUseCase {
	uc.sessionRepo = repo
	return uc
}

func (uc *CoordinateWorkflowUseCase) Execute(ctx context.Context, input *CoordinateWorkflowInput) (*CoordinateWorkflowOutput, error) {
	uc.logger.Info("workflow coordination started",
		"trigger_type", input.TriggerType,
		"classification", input.Classification,
	)

	genRequest := &services.WorkflowGenerationRequest{
		TriggerEvent:     input.TriggerEvent,
		TriggerType:      input.TriggerType,
		Classification:   input.Classification,
		ApprovalRequired: input.ApprovalRequired,
	}

	genResult, err := uc.workflowGenerator.Generate(ctx, genRequest)
	if err != nil {
		uc.logger.Error("workflow generation failed", err)
		return nil, err
	}

	session := genResult.Session
	workflow := genResult.Workflow
	decisions := genResult.Decisions

	if workflow != nil && uc.workflowRepo != nil {
		_ = uc.workflowRepo.SaveDefinition(ctx, workflow)
		if session != nil {
			session.WorkflowDefID = workflow.ID
		}
	}

	if session != nil && uc.sessionRepo != nil {
		_ = uc.sessionRepo.SaveSession(session)
	}

	if uc.knowledgeService != nil {
		recordDecisions(uc.knowledgeService, decisions)
	}

	output := &CoordinateWorkflowOutput{
		Session:          session,
		Workflow:         workflow,
		Decisions:        decisions,
		RequiresApproval: session.Status == entities.SessionWaitingApproval,
	}

	uc.logger.Info("workflow coordination completed",
		"session_id", session.ID,
		"workflow_id", workflow.ID,
		"requires_approval", output.RequiresApproval,
	)

	return output, nil
}

func (uc *CoordinateWorkflowUseCase) ApproveAndProceed(ctx context.Context, sessionID string, approver string) error {
	if uc.sessionRepo != nil {
		if err := uc.sessionRepo.ApproveSession(sessionID, approver); err != nil {
			return err
		}
	}
	uc.logger.Info("workflow approved and ready for execution",
		"session_id", sessionID,
		"approver", approver,
	)
	return nil
}

func (uc *CoordinateWorkflowUseCase) GetCoordinationStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"agents_available": len(uc.agentRegistry.GetAll()),
		"knowledge_base":   uc.knowledgeService.GetStats(),
	}
}

func recordDecisions(knowledgeService *services.KnowledgeGraphService, decisions map[string]*entities.Decision) {
	for agentRoleStr, decision := range decisions {
		agentRole := entities.AgentRole(agentRoleStr)
		knowledgeService.RecordDecision(agentRole, decision)
	}
}
