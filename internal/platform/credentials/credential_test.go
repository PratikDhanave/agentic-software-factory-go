package credentials

import (
	"context"
	"testing"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/platform/identity"
)

func setupBrokerWithAgent(t *testing.T) (*Broker, string, string) {
	sys := identity.New()
	broker := NewBroker(sys)
	ctx := context.Background()

	agentIdentity, sigKey, err := sys.Register(ctx, &identity.RegistrationRequest{
		Type:      "test",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("register agent failed: %v", err)
	}

	broker.SetSigningKey(agentIdentity.ID, sigKey.PrivateKey)
	return broker, agentIdentity.ID, sigKey.PrivateKey
}

func TestIssueCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	cred, err := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read", "execute"},
		Duration: 24 * time.Hour,
	})

	if err != nil {
		t.Fatalf("issue credential failed: %v", err)
	}

	if cred.ID == "" {
		t.Error("credential ID is empty")
	}

	if cred.Subject != "code:analyze" {
		t.Errorf("got subject %s, want code:analyze", cred.Subject)
	}

	if len(cred.Actions) != 2 {
		t.Errorf("got %d actions, want 2", len(cred.Actions))
	}

	if cred.Signature == "" {
		t.Error("signature is empty")
	}
}

func TestVerifyValidCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	cred, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: 24 * time.Hour,
	})

	valid, reason, err := broker.Verify(ctx, cred)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	if !valid {
		t.Errorf("credential verification failed: %s", reason)
	}
}

func TestVerifyExpiredCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	cred, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: -1 * time.Hour, // already expired
	})

	valid, reason, _ := broker.Verify(ctx, cred)
	if valid {
		t.Error("expected expired credential to fail verification")
	}

	if reason != "credential expired" {
		t.Errorf("got reason %s, want 'credential expired'", reason)
	}
}

func TestRevokeCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	cred, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: 24 * time.Hour,
	})

	err := broker.Revoke(ctx, cred.ID)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	valid, reason, _ := broker.Verify(ctx, cred)
	if valid {
		t.Error("revoked credential should fail verification")
	}

	if reason != "credential revoked" {
		t.Errorf("got reason %s, want 'credential revoked'", reason)
	}
}

func TestRefreshCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	oldCred, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: 1 * time.Hour,
	})

	newCred, err := broker.Refresh(ctx, oldCred.ID, 24*time.Hour)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if newCred.ID == oldCred.ID {
		t.Error("refreshed credential should have new ID")
	}

	if newCred.ExpiresAt.Before(oldCred.ExpiresAt.Add(20 * time.Hour)) {
		t.Error("refreshed credential expiry is too soon")
	}

	// Old credential should be revoked
	oldValid, _, _ := broker.Verify(ctx, oldCred)
	if oldValid {
		t.Error("old credential should be revoked after refresh")
	}

	// New credential should be valid
	newValid, reason, _ := broker.Verify(ctx, newCred)
	if !newValid {
		t.Errorf("new credential verification failed: %s", reason)
	}
}

func TestListForAgent(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	// Issue multiple credentials
	cred1, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: 24 * time.Hour,
	})

	cred2, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "workflow:execute",
		Actions:  []string{"write"},
		Duration: 24 * time.Hour,
	})

	credentials, err := broker.ListForAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(credentials) != 2 {
		t.Errorf("got %d credentials, want 2", len(credentials))
	}

	// Revoke one and list again
	broker.Revoke(ctx, cred1.ID)
	credentials, _ = broker.ListForAgent(ctx, agentID)
	if len(credentials) != 1 {
		t.Errorf("got %d credentials after revoke, want 1", len(credentials))
	}

	if credentials[0].ID != cred2.ID {
		t.Error("wrong credential in list after revoke")
	}
}

func TestHasCapability(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read", "execute"},
		Duration: 24 * time.Hour,
	})

	tests := []struct {
		subject string
		action  string
		want    bool
	}{
		{"code:analyze", "read", true},
		{"code:analyze", "execute", true},
		{"code:analyze", "write", false},
		{"code:other", "read", false},
	}

	for _, tt := range tests {
		has, err := broker.HasCapability(ctx, agentID, tt.subject, tt.action)
		if err != nil {
			t.Fatalf("has capability failed: %v", err)
		}

		if has != tt.want {
			t.Errorf("has capability(%s, %s) = %v, want %v",
				tt.subject, tt.action, has, tt.want)
		}
	}
}

func TestGetCredential(t *testing.T) {
	broker, agentID, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	issued, _ := broker.Issue(ctx, &CredentialRequest{
		AgentID:  agentID,
		Subject:  "code:analyze",
		Actions:  []string{"read"},
		Duration: 24 * time.Hour,
	})

	retrieved, err := broker.Get(ctx, issued.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if retrieved.ID != issued.ID {
		t.Error("credential ID mismatch")
	}
}

func TestGetNonexistentCredential(t *testing.T) {
	broker, _, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	_, err := broker.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent credential")
	}
}

func TestIssueMissingAgentID(t *testing.T) {
	broker, _, _ := setupBrokerWithAgent(t)
	ctx := context.Background()

	_, err := broker.Issue(ctx, &CredentialRequest{
		AgentID: "",
		Subject: "code:analyze",
		Actions: []string{"read"},
	})

	if err == nil {
		t.Error("expected error for missing agent ID")
	}
}
