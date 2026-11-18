package http

import (
	"encoding/json"
	"net/http"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/usecases"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	domainservices "github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type CoordinationHandler struct {
	coordinateUseCase      *usecases.CoordinateWorkflowUseCase
	interactiveHandler     *services.InteractiveDecisionHandler
	logger                 domainservices.Logger
}

func NewCoordinationHandler(
	coordinateUseCase *usecases.CoordinateWorkflowUseCase,
	interactiveHandler *services.InteractiveDecisionHandler,
	logger domainservices.Logger,
) *CoordinationHandler {
	return &CoordinationHandler{
		coordinateUseCase:  coordinateUseCase,
		interactiveHandler: interactiveHandler,
		logger:             logger,
	}
}

type GenerateWorkflowRequest struct {
	TriggerType      string                 `json:"trigger_type"`
	Classification   string                 `json:"classification"`
	TriggerEvent     map[string]interface{} `json:"trigger_event"`
	ApprovalRequired bool                   `json:"approval_required"`
}

type GenerateWorkflowResponse struct {
	SessionID           string                        `json:"session_id"`
	WorkflowID          string                        `json:"workflow_id"`
	WorkflowName        string                        `json:"workflow_name"`
	Status              string                        `json:"status"`
	RequiresApproval    bool                          `json:"requires_approval"`
	Decisions           map[string]*entities.Decision `json:"decisions"`
	PendingApprovals    []string                      `json:"pending_approvals"`
}

func (h *CoordinationHandler) GenerateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	input := &usecases.CoordinateWorkflowInput{
		TriggerEvent:     req.TriggerEvent,
		TriggerType:      req.TriggerType,
		Classification:   req.Classification,
		ApprovalRequired: req.ApprovalRequired,
	}

	output, err := h.coordinateUseCase.Execute(r.Context(), input)
	if err != nil {
		h.logger.Error("workflow generation failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pendingApprovals := []string{}
	for agentRole, decision := range output.Decisions {
		if decision.NeedsApproval {
			pendingApprovals = append(pendingApprovals, agentRole)
		}
	}

	resp := GenerateWorkflowResponse{
		SessionID:        output.Session.ID,
		WorkflowID:       output.Workflow.ID,
		WorkflowName:     output.Workflow.Name,
		Status:           string(output.Session.Status),
		RequiresApproval: output.RequiresApproval,
		Decisions:        output.Decisions,
		PendingApprovals: pendingApprovals,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type ApprovalRequestResponse struct {
	SessionID    string                     `json:"session_id"`
	Approvals    []*services.InteractiveDecision `json:"approvals"`
	TotalPending int                        `json:"total_pending"`
}

func (h *CoordinationHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvals, err := h.interactiveHandler.GetPendingApprovals()
	if err != nil {
		h.logger.Error("failed to get pending approvals", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	interactiveDecisions := []*services.InteractiveDecision{}
	for _, approval := range approvals {
		for agentRole, decision := range approval.Decisions {
			if decision.Confidence < 0.7 {
				iDec := &services.InteractiveDecision{
					SessionID:      approval.SessionID,
					AgentRole:      agentRole,
					CurrentChoice:  decision.Choice,
					Confidence:     decision.Confidence,
					Options:        decision.Alternatives,
					RequiresAction: true,
				}
				interactiveDecisions = append(interactiveDecisions, iDec)
			}
		}
	}

	resp := ApprovalRequestResponse{
		Approvals:    interactiveDecisions,
		TotalPending: len(interactiveDecisions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type ApprovalSubmissionRequest struct {
	SessionID string            `json:"session_id"`
	Approver  string            `json:"approver"`
	Approved  bool              `json:"approved"`
	Comments  string            `json:"comments"`
	Overrides map[string]string `json:"overrides"`
}

type ApprovalSubmissionResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func (h *CoordinationHandler) SubmitApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApprovalSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode approval request", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	approvalResp := &services.ApprovalResponse{
		SessionID: req.SessionID,
		Approver:  req.Approver,
		Approved:  req.Approved,
		Comments:  req.Comments,
		Overrides: req.Overrides,
	}

	err := h.interactiveHandler.ProcessApprovalResponse(approvalResp)
	if err != nil {
		h.logger.Error("failed to process approval", err)
		resp := ApprovalSubmissionResponse{
			SessionID: req.SessionID,
			Status:    "rejected",
			Message:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := ApprovalSubmissionResponse{
		SessionID: req.SessionID,
		Status:    "approved",
		Message:   "Workflow approved and queued for execution",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type SessionStatusResponse struct {
	SessionID              string                 `json:"session_id"`
	Status                 string                 `json:"status"`
	TotalDecisions         int                    `json:"total_decisions"`
	DecisionsRequiringApproval int               `json:"decisions_requiring_approval"`
	Decisions              map[string]interface{} `json:"decisions"`
}

func (h *CoordinationHandler) GetSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id parameter required", http.StatusBadRequest)
		return
	}

	session, err := h.interactiveHandler.GetSessionStatus(sessionID)
	if err != nil {
		h.logger.Error("failed to get session status", err, "session_id", sessionID)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	summary, _ := h.interactiveHandler.ListDecisionsSummary(sessionID)

	decisionsMap := map[string]interface{}{}
	if summary != nil {
		decisionsMap = summary
	}

	resp := SessionStatusResponse{
		SessionID:              session.ID,
		Status:                 string(session.Status),
		TotalDecisions:         len(session.Decisions),
		DecisionsRequiringApproval: 0,
		Decisions:              decisionsMap,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *CoordinationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/workflows/generate", h.GenerateWorkflow)
	mux.HandleFunc("/api/v1/approvals/pending", h.GetPendingApprovals)
	mux.HandleFunc("/api/v1/approvals/submit", h.SubmitApproval)
	mux.HandleFunc("/api/v1/sessions/status", h.GetSessionStatus)
}
