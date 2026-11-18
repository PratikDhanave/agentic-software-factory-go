package services

import (
	"testing"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

func TestPolicyEngine_AllowPolicy(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	policy := entities.NewPolicy("allow-workflows", "Allow all users to execute workflows")
	stmt := entities.NewStatement(entities.EffectAllow)
	stmt.AddResource(entities.ResourceWorkflow)
	stmt.AddAction(entities.ActionExecute)
	stmt.AddPrincipal(entities.PrincipalUser)
	policy.AddStatement(stmt)

	engine.AddPolicy(policy)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	req := &PolicyEvaluationRequest{
		SecurityContext: ctx,
		Resource:        entities.ResourceWorkflow,
		Action:          entities.ActionExecute,
		ResourceID:      "workflow-123",
	}

	result, err := engine.Evaluate(req)

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if result.Decision != entities.DecisionAllow {
		t.Errorf("Expected allow, got %s", result.Decision)
	}
}

func TestPolicyEngine_DenyPolicy(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	policy := entities.NewPolicy("deny-credential-access", "Deny agents from accessing credentials")
	stmt := entities.NewStatement(entities.EffectDeny)
	stmt.AddResource(entities.ResourceCredential)
	stmt.AddAction(entities.ActionRead)
	stmt.AddPrincipal(entities.PrincipalAgent)
	policy.AddStatement(stmt)

	engine.AddPolicy(policy)

	ctx := entities.NewSecurityContext("agent-1", entities.PrincipalAgent)
	req := &PolicyEvaluationRequest{
		SecurityContext: ctx,
		Resource:        entities.ResourceCredential,
		Action:          entities.ActionRead,
		ResourceID:      "cred-123",
	}

	result, err := engine.Evaluate(req)

	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if result.Decision != entities.DecisionDeny {
		t.Errorf("Expected deny, got %s", result.Decision)
	}
}

func TestPolicyEngine_RateLimiting(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	config := &entities.RateLimitConfig{
		Window:   1 * time.Minute,
		MaxCount: 5,
		Metric:   "default",
	}
	engine.SetRateLimitConfig("default", config)

	policy := entities.NewPolicy("allow-workflows", "Allow workflows")
	stmt := entities.NewStatement(entities.EffectAllow)
	stmt.AddResource(entities.ResourceWorkflow)
	stmt.AddAction(entities.ActionExecute)
	policy.AddStatement(stmt)
	engine.AddPolicy(policy)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)

	for i := 0; i < 5; i++ {
		req := &PolicyEvaluationRequest{
			SecurityContext: ctx,
			Resource:        entities.ResourceWorkflow,
			Action:          entities.ActionExecute,
			ResourceID:      "workflow-123",
		}

		result, _ := engine.Evaluate(req)
		if result.Decision != entities.DecisionAllow {
			t.Errorf("Expected allow for request %d, got %s", i+1, result.Decision)
		}
	}

	req := &PolicyEvaluationRequest{
		SecurityContext: ctx,
		Resource:        entities.ResourceWorkflow,
		Action:          entities.ActionExecute,
		ResourceID:      "workflow-123",
	}

	result, _ := engine.Evaluate(req)
	if result.Decision != entities.DecisionRateLimited {
		t.Errorf("Expected rate limited after 6th request, got %s", result.Decision)
	}
}

func TestPolicyEngine_PolicyPriority(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	denyPolicy := entities.NewPolicy("deny-agents", "Deny agents from executing workflows")
	denyPolicy.Priority = 10
	denyStmt := entities.NewStatement(entities.EffectDeny)
	denyStmt.AddResource(entities.ResourceWorkflow)
	denyStmt.AddAction(entities.ActionExecute)
	denyStmt.AddPrincipal(entities.PrincipalAgent)
	denyPolicy.AddStatement(denyStmt)

	allowPolicy := entities.NewPolicy("allow-users", "Allow users")
	allowPolicy.Priority = 5
	allowStmt := entities.NewStatement(entities.EffectAllow)
	allowStmt.AddResource(entities.ResourceWorkflow)
	allowStmt.AddAction(entities.ActionExecute)
	allowStmt.AddPrincipal(entities.PrincipalUser)
	allowPolicy.AddStatement(allowStmt)

	engine.AddPolicy(denyPolicy)
	engine.AddPolicy(allowPolicy)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	req := &PolicyEvaluationRequest{
		SecurityContext: ctx,
		Resource:        entities.ResourceWorkflow,
		Action:          entities.ActionExecute,
		ResourceID:      "workflow-123",
	}

	result, _ := engine.Evaluate(req)

	if result.Decision != entities.DecisionAllow {
		t.Errorf("Expected allow for user (agent deny doesn't apply), got %s", result.Decision)
	}
}

func TestPolicyEngine_MultipleResources(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	policy := entities.NewPolicy("full-access", "Full access to all resources")
	stmt := entities.NewStatement(entities.EffectAllow)
	stmt.AddResource(entities.ResourceWorkflow)
	stmt.AddResource(entities.ResourceExecution)
	stmt.AddResource(entities.ResourceAudit)
	stmt.AddAction(entities.ActionRead)
	policy.AddStatement(stmt)

	engine.AddPolicy(policy)

	testCases := []struct {
		resource entities.Resource
		action   entities.Action
		expected entities.PolicyDecision
	}{
		{entities.ResourceWorkflow, entities.ActionRead, entities.DecisionAllow},
		{entities.ResourceExecution, entities.ActionRead, entities.DecisionAllow},
		{entities.ResourceAudit, entities.ActionRead, entities.DecisionAllow},
		{entities.ResourceCredential, entities.ActionRead, entities.DecisionDeny},
	}

	for _, tc := range testCases {
		ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
		req := &PolicyEvaluationRequest{
			SecurityContext: ctx,
			Resource:        tc.resource,
			Action:          tc.action,
			ResourceID:      "resource-123",
		}

		result, _ := engine.Evaluate(req)
		if result.Decision != tc.expected {
			t.Errorf("Resource %s: expected %s, got %s", tc.resource, tc.expected, result.Decision)
		}
	}
}

func TestPolicyEngine_WildcardMatching(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	policy := entities.NewPolicy("wildcard-allow", "Allow all actions on all resources")
	stmt := entities.NewStatement(entities.EffectAllow)
	stmt.AddResource("*")
	stmt.AddAction("*")
	policy.AddStatement(stmt)

	engine.AddPolicy(policy)

	testCases := []struct {
		resource entities.Resource
		action   entities.Action
	}{
		{entities.ResourceWorkflow, entities.ActionCreate},
		{entities.ResourceExecution, entities.ActionDelete},
		{entities.ResourceCredential, entities.ActionUpdate},
	}

	for _, tc := range testCases {
		ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
		req := &PolicyEvaluationRequest{
			SecurityContext: ctx,
			Resource:        tc.resource,
			Action:          tc.action,
			ResourceID:      "resource-123",
		}

		result, _ := engine.Evaluate(req)
		if result.Decision != entities.DecisionAllow {
			t.Errorf("Wildcard policy: expected allow for %s:%s, got %s", tc.resource, tc.action, result.Decision)
		}
	}
}

func TestPolicyEngine_InactivePolicy(t *testing.T) {
	engine := NewPolicyEngine(&MockLogger{})

	policy := entities.NewPolicy("inactive-policy", "This policy is inactive")
	policy.Active = false
	stmt := entities.NewStatement(entities.EffectAllow)
	stmt.AddResource(entities.ResourceWorkflow)
	stmt.AddAction(entities.ActionExecute)
	policy.AddStatement(stmt)

	engine.AddPolicy(policy)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	req := &PolicyEvaluationRequest{
		SecurityContext: ctx,
		Resource:        entities.ResourceWorkflow,
		Action:          entities.ActionExecute,
		ResourceID:      "workflow-123",
	}

	result, _ := engine.Evaluate(req)

	if result.Decision != entities.DecisionDeny {
		t.Errorf("Inactive policy should not allow: got %s", result.Decision)
	}
}

func TestPolicyEngine_PermissionCheck(t *testing.T) {
	principal := entities.NewPrincipal(entities.PrincipalUser, "developer")
	principal.AddPermission(entities.NewPermission(entities.ResourceWorkflow, entities.ActionExecute, "*"))
	principal.AddPermission(entities.NewPermission(entities.ResourceExecution, entities.ActionRead, "*"))

	if !principal.HasPermission(entities.ResourceWorkflow, entities.ActionExecute) {
		t.Error("Should have permission to execute workflows")
	}

	if !principal.HasPermission(entities.ResourceExecution, entities.ActionRead) {
		t.Error("Should have permission to read executions")
	}

	if principal.HasPermission(entities.ResourceCredential, entities.ActionRead) {
		t.Error("Should not have permission to read credentials")
	}
}

func TestPolicyEngine_ScopedPermissions(t *testing.T) {
	principal := entities.NewPrincipal(entities.PrincipalUser, "developer")
	principal.AddPermission(entities.NewPermission(entities.ResourceWorkflow, entities.ActionExecute, "project:frontend"))

	if !principal.HasPermissionForScope(entities.ResourceWorkflow, entities.ActionExecute, "project:frontend") {
		t.Error("Should have scoped permission")
	}

	if principal.HasPermissionForScope(entities.ResourceWorkflow, entities.ActionExecute, "project:backend") {
		t.Error("Should not have permission for different scope")
	}
}
