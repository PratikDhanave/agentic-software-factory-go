package services

import (
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
	domainservices "github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type InteractiveDecision struct {
	SessionID      string
	AgentRole      string
	Question       string
	CurrentChoice  string
	Reasoning      string
	Confidence     float64
	Options        []string
	RequiresAction bool
	Status         string
}

type ApprovalRequest struct {
	SessionID    string
	Decisions    map[string]*entities.Decision
	RequiredRole string
	Deadline     string
}

type ApprovalResponse struct {
	SessionID   string
	Approver    string
	Approved    bool
	Comments    string
	Overrides   map[string]string
}

type InteractiveDecisionHandler struct {
	sessionRepo repositories.CoordinationSessionRepository
	logger      domainservices.Logger
}

func NewInteractiveDecisionHandler(
	sessionRepo repositories.CoordinationSessionRepository,
	logger domainservices.Logger,
) *InteractiveDecisionHandler {
	return &InteractiveDecisionHandler{
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

func (idh *InteractiveDecisionHandler) RequestApproval(approvalReq *ApprovalRequest) ([]*InteractiveDecision, error) {
	_, err := idh.sessionRepo.GetSession(approvalReq.SessionID)
	if err != nil {
		idh.logger.Error("failed to get session", err, "session_id", approvalReq.SessionID)
		return nil, err
	}

	interactiveDecisions := []*InteractiveDecision{}

	for agentRole, decision := range approvalReq.Decisions {
		if decision.Confidence < 0.7 {
			iDec := &InteractiveDecision{
				SessionID:      approvalReq.SessionID,
				AgentRole:      agentRole,
				Question:       fmt.Sprintf("Agent %s has low confidence (%.0f%%). Approve choice '%s'?", agentRole, decision.Confidence*100, decision.Choice),
				CurrentChoice:  decision.Choice,
				Reasoning:      decision.Reasoning,
				Confidence:     decision.Confidence,
				Options:        decision.Alternatives,
				RequiresAction: true,
				Status:         "pending_approval",
			}
			interactiveDecisions = append(interactiveDecisions, iDec)
		}
	}

	if len(interactiveDecisions) > 0 {
		idh.logger.Info("approval requested",
			"session_id", approvalReq.SessionID,
			"decisions_pending", len(interactiveDecisions),
		)
	}

	return interactiveDecisions, nil
}

func (idh *InteractiveDecisionHandler) ProcessApprovalResponse(response *ApprovalResponse) error {
	session, err := idh.sessionRepo.GetSession(response.SessionID)
	if err != nil {
		idh.logger.Error("failed to get session", err, "session_id", response.SessionID)
		return err
	}

	if !response.Approved {
		idh.logger.Info("workflow rejected",
			"session_id", response.SessionID,
			"approver", response.Approver,
			"reason", response.Comments,
		)
		idh.sessionRepo.UpdateSessionStatus(response.SessionID, entities.SessionFailed)
		return fmt.Errorf("workflow rejected by %s: %s", response.Approver, response.Comments)
	}

	for agentRole, override := range response.Overrides {
		if decision, exists := session.Decisions[agentRole]; exists {
			decision.Choice = override
			decision.Reasoning = "Overridden by: " + response.Approver + " (" + response.Comments + ")"
			idh.sessionRepo.UpdateDecision(response.SessionID, agentRole, decision)
		}
	}

	idh.logger.Info("workflow approved",
		"session_id", response.SessionID,
		"approver", response.Approver,
		"overrides", len(response.Overrides),
	)

	idh.sessionRepo.ApproveSession(response.SessionID, response.Approver)
	return nil
}

func (idh *InteractiveDecisionHandler) GetPendingApprovals() ([]*ApprovalRequest, error) {
	sessions, err := idh.sessionRepo.ListPendingApprovals()
	if err != nil {
		return nil, err
	}

	requests := []*ApprovalRequest{}
	for _, session := range sessions {
		req := &ApprovalRequest{
			SessionID:    session.ID,
			Decisions:    session.Decisions,
			RequiredRole: "approver",
		}
		requests = append(requests, req)
	}

	return requests, nil
}

func (idh *InteractiveDecisionHandler) UpdateDecisionWithUserInput(sessionID string, agentRole string, userChoice string, userReasoning string) error {
	session, err := idh.sessionRepo.GetSession(sessionID)
	if err != nil {
		return err
	}

	if decision, exists := session.Decisions[agentRole]; exists {
		decision.Choice = userChoice
		decision.Reasoning = userReasoning
		decision.NeedsApproval = false

		return idh.sessionRepo.UpdateDecision(sessionID, agentRole, decision)
	}

	return fmt.Errorf("decision not found for agent: %s", agentRole)
}

func (idh *InteractiveDecisionHandler) GetSessionStatus(sessionID string) (*entities.CoordinationSession, error) {
	return idh.sessionRepo.GetSession(sessionID)
}

func (idh *InteractiveDecisionHandler) ListDecisionsSummary(sessionID string) (map[string]interface{}, error) {
	session, err := idh.sessionRepo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"session_id":           session.ID,
		"status":               session.Status,
		"total_decisions":      len(session.Decisions),
		"decisions_requiring_approval": 0,
		"decisions_summary":    []map[string]interface{}{},
	}

	requiresApproval := 0
	for agentRole, decision := range session.Decisions {
		if decision.NeedsApproval || decision.Confidence < 0.7 {
			requiresApproval++
		}

		decisionSummary := map[string]interface{}{
			"agent":      agentRole,
			"choice":     decision.Choice,
			"confidence": decision.Confidence,
			"reasoning":  decision.Reasoning,
		}
		summary["decisions_summary"] = append(summary["decisions_summary"].([]map[string]interface{}), decisionSummary)
	}

	summary["decisions_requiring_approval"] = requiresApproval

	return summary, nil
}
