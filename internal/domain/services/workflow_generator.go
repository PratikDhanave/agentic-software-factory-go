package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type WorkflowGenerator struct {
	agentRegistry *AgentRegistry
	agents        map[entities.AgentRole]AgentInterface
	llmClient     LLMClient
	logger        Logger
}

func NewWorkflowGenerator(
	agentRegistry *AgentRegistry,
	llmClient LLMClient,
	logger Logger,
) *WorkflowGenerator {
	return &WorkflowGenerator{
		agentRegistry: agentRegistry,
		agents:        make(map[entities.AgentRole]AgentInterface),
		llmClient:     llmClient,
		logger:        logger,
	}
}

func (wg *WorkflowGenerator) CreateAgent(role entities.AgentRole) AgentInterface {
	if agent, exists := wg.agents[role]; exists {
		return agent
	}

	caps := wg.agentRegistry.Get(role)
	if caps == nil {
		wg.logger.Error(fmt.Sprintf("Agent capabilities not found for role: %s", role), nil)
		return nil
	}

	agent := NewBaseAgent(role, caps, wg.llmClient, wg.logger)
	wg.agents[role] = agent
	return agent
}

type WorkflowGenerationRequest struct {
	TriggerEvent     map[string]interface{}
	TriggerType      string
	Classification   string
	WorkflowTemplate *entities.WorkflowDefinition
	ApprovalRequired bool
}

type WorkflowGenerationResult struct {
	Workflow  *entities.WorkflowDefinition
	Decisions map[string]*entities.Decision
	Session   *entities.CoordinationSession
}

func (wg *WorkflowGenerator) Generate(ctx context.Context, req *WorkflowGenerationRequest) (*WorkflowGenerationResult, error) {
	session := entities.NewCoordinationSession(req.TriggerEvent)
	session.Start()

	wg.logger.Info("workflow generation started",
		"session_id", session.ID,
		"trigger_type", req.TriggerType,
		"classification", req.Classification,
	)

	// Determine agents to consult based on classification
	requiredAgents := wg.getAgentsForClassification(req.Classification)

	decisions := make(map[string]*entities.Decision)
	priorContext := make(map[string]interface{})

	// Consult each agent in sequence
	for _, agentRole := range requiredAgents {
		agent := wg.CreateAgent(agentRole)
		if agent == nil {
			wg.logger.Error("failed to create agent", fmt.Errorf("role: %s", agentRole))
			continue
		}

		decision, err := wg.consultAgent(ctx, agent, req.TriggerEvent, priorContext)
		if err != nil {
			wg.logger.Error("agent consultation failed", err, "agent_role", agentRole)
			continue
		}

		decisions[string(agentRole)] = decision
		session.Decisions[string(agentRole)] = decision
		priorContext[string(agentRole)] = decision

		wg.logger.Debug("agent decision recorded",
			"agent_role", agentRole,
			"decision", decision.Choice,
			"confidence", decision.Confidence,
		)
	}

	// Generate workflow from decisions
	workflow := wg.generateWorkflow(req.TriggerEvent, req.Classification, decisions)

	// Validate workflow
	if err := workflow.Validate(); err != nil {
		wg.logger.Error("workflow validation failed", err)
		session.Fail()
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Check if approval needed
	needsApproval := req.ApprovalRequired || wg.needsApprovalFromDecisions(decisions)
	if needsApproval {
		session.RequestApproval()
	} else {
		session.Complete()
	}

	wg.logger.Info("workflow generation completed",
		"session_id", session.ID,
		"workflow_id", workflow.ID,
		"needs_approval", needsApproval,
	)

	return &WorkflowGenerationResult{
		Workflow:  workflow,
		Decisions: decisions,
		Session:   session,
	}, nil
}

func (wg *WorkflowGenerator) consultAgent(
	ctx context.Context,
	agent AgentInterface,
	event map[string]interface{},
	priorContext map[string]interface{},
) (*entities.Decision, error) {

	query := wg.buildAgentQuery(agent.GetRole(), event)

	req := &AgentRequest{
		AgentRole:    agent.GetRole(),
		Event:        event,
		PriorContext: priorContext,
		Query:        query,
	}

	resp, err := agent.Decide(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Decision, nil
}

func (wg *WorkflowGenerator) buildAgentQuery(role entities.AgentRole, event map[string]interface{}) string {
	queries := map[entities.AgentRole]string{
		entities.ProductAgent: "What is the priority and scope of this work? What stakeholders are involved?",
		entities.ArchitectAgent: "What is the design approach? What are the architectural implications?",
		entities.DesignAgent: "What design patterns and structure should we use?",
		entities.DeveloperAgent: "What is the implementation strategy? What is the effort estimate?",
		entities.SecurityAgent: "What security concerns exist? What security testing is required?",
		entities.QAAgent: "What testing strategy should we use? What coverage is needed?",
		entities.DevOpsAgent: "What deployment strategy should we use?",
		entities.CoordinatorAgent: "Based on other agents' decisions, what workflow should we generate?",
	}

	if query, exists := queries[role]; exists {
		return query
	}

	return "What decision should we make for this event?"
}

func (wg *WorkflowGenerator) generateWorkflow(
	event map[string]interface{},
	classification string,
	decisions map[string]*entities.Decision,
) *entities.WorkflowDefinition {

	// Generate jobs based on agent decisions
	jobs := wg.generateJobsFromDecisions(classification, decisions)

	workflow := entities.NewWorkflowDefinition(
		uuid.New().String(),
		fmt.Sprintf("%s-workflow", classification),
		"1.0",
		jobs,
	)

	return workflow
}

func (wg *WorkflowGenerator) generateJobsFromDecisions(
	classification string,
	decisions map[string]*entities.Decision,
) map[string]*entities.JobDef {

	jobs := make(map[string]*entities.JobDef)

	// Always start with analysis
	jobs["analyze"] = &entities.JobDef{
		Task:      fmt.Sprintf("analyze-%s", classification),
		DependsOn: []string{},
		Input:     make(map[string]interface{}),
	}

	// Add jobs based on decisions
	for agentRole, decision := range decisions {
		if agentRole == string(entities.ProductAgent) {
			jobs["prioritize"] = &entities.JobDef{
				Task:      "prioritize-work",
				DependsOn: []string{"analyze"},
				Input: map[string]interface{}{
					"priority": decision.Choice,
				},
			}
		}

		if agentRole == string(entities.SecurityAgent) {
			jobs["security_check"] = &entities.JobDef{
				Task:      "security-analysis",
				DependsOn: []string{"analyze"},
				Input: map[string]interface{}{
					"security_level": decision.Choice,
				},
			}
		}

		if agentRole == string(entities.DeveloperAgent) {
			jobs["implement"] = &entities.JobDef{
				Task:      "implement-solution",
				DependsOn: []string{"analyze", "prioritize"},
				Input: map[string]interface{}{
					"effort_estimate": decision.Impact["effort"],
				},
			}
		}
	}

	// Final verification job
	deps := []string{"analyze"}
	if _, hasImpl := jobs["implement"]; hasImpl {
		deps = append(deps, "implement")
	}
	jobs["verify"] = &entities.JobDef{
		Task:      "verify-solution",
		DependsOn: deps,
		Input:     make(map[string]interface{}),
	}

	return jobs
}

func (wg *WorkflowGenerator) needsApprovalFromDecisions(decisions map[string]*entities.Decision) bool {
	for _, decision := range decisions {
		if decision.NeedsApproval || decision.Confidence < 0.7 {
			return true
		}
	}
	return false
}

func (wg *WorkflowGenerator) getAgentsForClassification(classification string) []entities.AgentRole {
	agentsByClassification := map[string][]entities.AgentRole{
		"bug-fix": {
			entities.ArchitectAgent,
			entities.SecurityAgent,
			entities.DeveloperAgent,
			entities.QAAgent,
		},
		"feature": {
			entities.ProductAgent,
			entities.DesignAgent,
			entities.DeveloperAgent,
			entities.QAAgent,
		},
		"security-issue": {
			entities.SecurityAgent,
			entities.ArchitectAgent,
			entities.DeveloperAgent,
			entities.QAAgent,
		},
		"performance": {
			entities.PerformanceAgent,
			entities.DeveloperAgent,
			entities.QAAgent,
		},
		"refactor": {
			entities.ArchitectAgent,
			entities.DeveloperAgent,
			entities.QAAgent,
		},
	}

	if agents, exists := agentsByClassification[classification]; exists {
		return agents
	}

	return []entities.AgentRole{
		entities.ProductAgent,
		entities.DesignAgent,
		entities.DeveloperAgent,
	}
}
