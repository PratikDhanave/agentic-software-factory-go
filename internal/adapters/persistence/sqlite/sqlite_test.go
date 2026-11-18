package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

func TestSQLiteRepositoriesAndConcurrency(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "factory_test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open sqlite store: %v", err)
	}
	defer store.Close()

	// 1. Workflow definition & execution roundtrip
	wfRepo := NewWorkflowRepository(store)
	def := entities.NewWorkflowDefinition("wf-1", "spec-to-ship", "1.0", map[string]*entities.JobDef{
		"generate": {Task: "generate_code", DependsOn: []string{}},
		"verify":   {Task: "verify_code", DependsOn: []string{"generate"}},
	})
	if err := wfRepo.SaveDefinition(ctx, def); err != nil {
		t.Fatalf("SaveDefinition: %v", err)
	}
	loadedDef, err := wfRepo.GetDefinition(ctx, "wf-1")
	if err != nil || loadedDef.Name != "spec-to-ship" || len(loadedDef.JobsMap) != 2 {
		t.Fatalf("GetDefinition mismatch: %+v, err=%v", loadedDef, err)
	}

	exec := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{"issue": 42})
	if err := wfRepo.SaveExecution(ctx, exec); err != nil {
		t.Fatalf("SaveExecution: %v", err)
	}
	if err := wfRepo.UpdateExecutionStatus(ctx, "exec-1", entities.WorkflowExecCompleted); err != nil {
		t.Fatalf("UpdateExecutionStatus: %v", err)
	}
	loadedExec, err := wfRepo.GetExecution(ctx, "exec-1")
	if err != nil || loadedExec.Status != entities.WorkflowExecCompleted {
		t.Fatalf("GetExecution mismatch: %+v, err=%v", loadedExec, err)
	}

	// 2. Coordination session & approval flow
	coordRepo := NewCoordinationSessionRepository(store)
	session := entities.NewCoordinationSession(map[string]interface{}{"title": "Add OAuth login"})
	session.Decisions["architect"] = &entities.Decision{
		AgentRole:     entities.ArchitectAgent,
		Choice:        "jwt-pkce",
		Confidence:    0.62,
		NeedsApproval: true,
	}
	session.RequestApproval()
	if err := coordRepo.SaveSession(session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	pending, err := coordRepo.ListPendingApprovals()
	if err != nil || len(pending) != 1 {
		t.Fatalf("expected 1 pending approval, got %d (err=%v)", len(pending), err)
	}
	if err := coordRepo.ApproveSession(session.ID, "pratik"); err != nil {
		t.Fatalf("ApproveSession: %v", err)
	}
	approvedSession, err := coordRepo.GetSession(session.ID)
	if err != nil || approvedSession.Status != entities.SessionExecuting {
		t.Fatalf("expected executing after approval, got %+v (err=%v)", approvedSession, err)
	}

	// 3. Learning repository & concurrent WAL writes
	learnRepo := NewLearningRepository(store)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dec := &domain.AgentDecision{
				ID:             uuid.New(),
				ExecutionID:    uuid.New(),
				AgentRole:      "Architect",
				IssueType:      "auth",
				Decision:       "Use OAuth2 PKCE",
				Reasoning:      "oauth session jwt security",
				Confidence:     0.92,
				ApprovalStatus: "approved",
			}
			if err := learnRepo.SaveDecision(ctx, dec); err != nil {
				t.Errorf("concurrent SaveDecision: %v", err)
			}
		}()
	}
	wg.Wait()

	decisions, err := learnRepo.GetRecentDecisions(ctx, "architect", 50)
	if err != nil || len(decisions) != 12 {
		t.Fatalf("expected 12 persisted decisions, got %d (err=%v)", len(decisions), err)
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats["total_agent_decisions"].(int) != 12 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
