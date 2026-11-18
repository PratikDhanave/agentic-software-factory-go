package services

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	domainservices "github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/persistence"
)

func TestInteractiveDecisionHandler_RequestApproval(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{
		"issue": "test-issue",
	})
	session.Decisions[string(entities.SecurityAgent)] = &entities.Decision{
		AgentRole:  entities.SecurityAgent,
		Choice:     "intensive-testing",
		Confidence: 0.60,
	}
	sessionRepo.SaveSession(session)

	approvalReq := &ApprovalRequest{
		SessionID: session.ID,
		Decisions: session.Decisions,
	}

	decisions, err := handler.RequestApproval(approvalReq)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}

	if len(decisions) != 1 {
		t.Errorf("Expected 1 decision requiring approval, got %d", len(decisions))
	}

	if decisions[0].Confidence != 0.60 {
		t.Errorf("Expected confidence 0.60, got %f", decisions[0].Confidence)
	}
}

func TestInteractiveDecisionHandler_ProcessApprovalResponse_Approved(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{})
	session.Status = entities.SessionWaitingApproval
	session.Decisions[string(entities.DeveloperAgent)] = &entities.Decision{
		AgentRole:  entities.DeveloperAgent,
		Choice:     "implement-feature",
		Confidence: 0.75,
	}
	sessionRepo.SaveSession(session)

	response := &ApprovalResponse{
		SessionID: session.ID,
		Approver:  "product-lead",
		Approved:  true,
		Comments:  "Looks good",
		Overrides: map[string]string{},
	}

	err := handler.ProcessApprovalResponse(response)
	if err != nil {
		t.Fatalf("ProcessApprovalResponse failed: %v", err)
	}

	updated, _ := sessionRepo.GetSession(session.ID)
	if updated.Status != entities.SessionExecuting {
		t.Errorf("Expected status executing, got %s", updated.Status)
	}
}

func TestInteractiveDecisionHandler_ProcessApprovalResponse_WithOverride(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{})
	session.Status = entities.SessionWaitingApproval
	session.Decisions[string(entities.SecurityAgent)] = &entities.Decision{
		AgentRole:  entities.SecurityAgent,
		Choice:     "basic-testing",
		Confidence: 0.65,
	}
	sessionRepo.SaveSession(session)

	response := &ApprovalResponse{
		SessionID: session.ID,
		Approver:  "security-lead",
		Approved:  true,
		Comments:  "Need more security testing",
		Overrides: map[string]string{
			string(entities.SecurityAgent): "intensive-testing",
		},
	}

	err := handler.ProcessApprovalResponse(response)
	if err != nil {
		t.Fatalf("ProcessApprovalResponse failed: %v", err)
	}

	updated, _ := sessionRepo.GetSession(session.ID)
	securityDecision := updated.Decisions[string(entities.SecurityAgent)]

	if securityDecision.Choice != "intensive-testing" {
		t.Errorf("Expected override to 'intensive-testing', got %s", securityDecision.Choice)
	}

	if !contains(securityDecision.Reasoning, "security-lead") {
		t.Errorf("Expected override reason to mention approver")
	}
}

func TestInteractiveDecisionHandler_ProcessApprovalResponse_Rejected(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{})
	session.Status = entities.SessionWaitingApproval
	sessionRepo.SaveSession(session)

	response := &ApprovalResponse{
		SessionID: session.ID,
		Approver:  "product-lead",
		Approved:  false,
		Comments:  "Does not align with roadmap",
		Overrides: map[string]string{},
	}

	err := handler.ProcessApprovalResponse(response)
	if err == nil {
		t.Fatal("Expected error when rejecting workflow")
	}

	updated, _ := sessionRepo.GetSession(session.ID)
	if updated.Status != entities.SessionFailed {
		t.Errorf("Expected status failed, got %s", updated.Status)
	}
}

func TestInteractiveDecisionHandler_UpdateDecisionWithUserInput(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{})
	session.Decisions[string(entities.ArchitectAgent)] = &entities.Decision{
		AgentRole:  entities.ArchitectAgent,
		Choice:     "monolithic",
		Confidence: 0.72,
	}
	sessionRepo.SaveSession(session)

	err := handler.UpdateDecisionWithUserInput(
		session.ID,
		string(entities.ArchitectAgent),
		"microservices",
		"Better scalability for our use case",
	)

	if err != nil {
		t.Fatalf("UpdateDecisionWithUserInput failed: %v", err)
	}

	updated, _ := sessionRepo.GetSession(session.ID)
	decision := updated.Decisions[string(entities.ArchitectAgent)]

	if decision.Choice != "microservices" {
		t.Errorf("Expected choice microservices, got %s", decision.Choice)
	}

	if !contains(decision.Reasoning, "scalability") {
		t.Errorf("Expected reasoning to contain 'scalability'")
	}
}

func TestInteractiveDecisionHandler_GetPendingApprovals(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session1 := entities.NewCoordinationSession(map[string]interface{}{})
	session1.Status = entities.SessionWaitingApproval
	session1.Decisions[string(entities.DeveloperAgent)] = &entities.Decision{
		Choice: "implement",
	}

	session2 := entities.NewCoordinationSession(map[string]interface{}{})
	session2.Status = entities.SessionCoordinating

	sessionRepo.SaveSession(session1)
	sessionRepo.SaveSession(session2)

	approvals, err := handler.GetPendingApprovals()
	if err != nil {
		t.Fatalf("GetPendingApprovals failed: %v", err)
	}

	if len(approvals) != 1 {
		t.Errorf("Expected 1 pending approval, got %d", len(approvals))
	}
}

func TestInteractiveDecisionHandler_ListDecisionsSummary(t *testing.T) {
	sessionRepo := persistence.NewCoordinationSessionMemoryStore()
	handler := NewInteractiveDecisionHandler(sessionRepo, &domainservices.MockLogger{})

	session := entities.NewCoordinationSession(map[string]interface{}{})
	session.Decisions[string(entities.ProductAgent)] = &entities.Decision{
		Choice:     "high-priority",
		Confidence: 0.95,
	}
	session.Decisions[string(entities.SecurityAgent)] = &entities.Decision{
		Choice:     "basic-testing",
		Confidence: 0.60,
		NeedsApproval: true,
	}
	sessionRepo.SaveSession(session)

	summary, err := handler.ListDecisionsSummary(session.ID)
	if err != nil {
		t.Fatalf("ListDecisionsSummary failed: %v", err)
	}

	if summary["total_decisions"] != 2 {
		t.Errorf("Expected 2 decisions, got %v", summary["total_decisions"])
	}

	if summary["decisions_requiring_approval"] != 1 {
		t.Errorf("Expected 1 decision requiring approval, got %v", summary["decisions_requiring_approval"])
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
