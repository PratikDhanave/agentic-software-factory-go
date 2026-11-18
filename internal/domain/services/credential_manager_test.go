package services

import (
	"testing"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type MockCredentialRepository struct {
	credentials map[string]*entities.Credential
}

func NewMockCredentialRepository() *MockCredentialRepository {
	return &MockCredentialRepository{
		credentials: make(map[string]*entities.Credential),
	}
}

func (mcr *MockCredentialRepository) SaveCredential(credential *entities.Credential) error {
	mcr.credentials[credential.ID] = credential
	return nil
}

func (mcr *MockCredentialRepository) GetCredential(id string) (*entities.Credential, error) {
	cred, exists := mcr.credentials[id]
	if !exists {
		return nil, repositories.ErrNotFound
	}
	return cred, nil
}

func (mcr *MockCredentialRepository) UpdateCredential(credential *entities.Credential) error {
	mcr.credentials[credential.ID] = credential
	return nil
}

func (mcr *MockCredentialRepository) GetCredentialsByOwner(ownerID string) ([]*entities.Credential, error) {
	creds := []*entities.Credential{}
	for _, c := range mcr.credentials {
		if c.Owner == ownerID {
			creds = append(creds, c)
		}
	}
	return creds, nil
}

func (mcr *MockCredentialRepository) GetCredentialsByType(credType entities.CredentialType) ([]*entities.Credential, error) {
	creds := []*entities.Credential{}
	for _, c := range mcr.credentials {
		if c.Type == credType {
			creds = append(creds, c)
		}
	}
	return creds, nil
}

func (mcr *MockCredentialRepository) ListActiveCredentials() ([]*entities.Credential, error) {
	creds := []*entities.Credential{}
	for _, c := range mcr.credentials {
		if c.Active {
			creds = append(creds, c)
		}
	}
	return creds, nil
}

func (mcr *MockCredentialRepository) DeleteCredential(id string) error {
	delete(mcr.credentials, id)
	return nil
}

type MockAuditRepository struct {
	entries []*entities.AuditEntry
}

func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		entries: []*entities.AuditEntry{},
	}
}

func (mar *MockAuditRepository) AppendEntry(entry *entities.AuditEntry) error {
	mar.entries = append(mar.entries, entry)
	return nil
}

func (mar *MockAuditRepository) GetEntry(id string) (*entities.AuditEntry, error) {
	for _, entry := range mar.entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (mar *MockAuditRepository) GetEntriesByPrincipal(principalID string) ([]*entities.AuditEntry, error) {
	entries := []*entities.AuditEntry{}
	for _, e := range mar.entries {
		if e.Principal == principalID {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (mar *MockAuditRepository) GetEntriesByResource(resource entities.Resource, resourceID string) ([]*entities.AuditEntry, error) {
	entries := []*entities.AuditEntry{}
	for _, e := range mar.entries {
		if e.Resource == resource && e.ResourceID == resourceID {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (mar *MockAuditRepository) GetEntriesByDateRange(start, end time.Time) ([]*entities.AuditEntry, error) {
	entries := []*entities.AuditEntry{}
	for _, e := range mar.entries {
		if e.Timestamp.After(start) && e.Timestamp.Before(end) {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (mar *MockAuditRepository) GetEntriesByStatus(status entities.AuditStatus) ([]*entities.AuditEntry, error) {
	entries := []*entities.AuditEntry{}
	for _, e := range mar.entries {
		if e.Status == status {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

func (mar *MockAuditRepository) VerifyIntegrity() bool {
	return true
}

var (
	ErrNotFound = repositories.ErrNotFound
)

func TestCredentialManager_StoreCredential(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionCreate, "*"))

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	cred.AddScope("github.com")

	err := manager.StoreCredential(ctx, cred)
	if err != nil {
		t.Fatalf("StoreCredential failed: %v", err)
	}

	retrieved, _ := credRepo.GetCredential(cred.ID)
	if retrieved == nil {
		t.Error("Credential not stored")
	}

	if retrieved.Secret == "ghp_secret123" {
		t.Error("Secret should be encrypted, not plaintext")
	}
}

func TestCredentialManager_RequestCredential(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	cred.AddScope("github.com")
	credRepo.SaveCredential(cred)

	encrypted, _ := manager.encryptSecret("ghp_secret123")
	cred.Secret = encrypted
	credRepo.UpdateCredential(cred)

	ctx := entities.NewSecurityContext("agent-1", entities.PrincipalAgent)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionRead, "*"))

	req := entities.NewCredentialRequest(cred.ID, "agent-1", entities.PrincipalAgent, "deploy", 1*time.Minute)
	req.AddScope("github.com")

	injected, err := manager.RequestCredential(ctx, req)
	if err != nil {
		t.Fatalf("RequestCredential failed: %v", err)
	}

	if injected.Value != "ghp_secret123" {
		t.Errorf("Expected secret 'ghp_secret123', got '%s'", injected.Value)
	}

	if injected.Type != entities.CredentialTypeAPIKey {
		t.Errorf("Expected type APIKey, got %s", injected.Type)
	}
}

func TestCredentialManager_UnauthorizedAccess(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	credRepo.SaveCredential(cred)

	ctx := entities.NewSecurityContext("agent-1", entities.PrincipalAgent)

	req := entities.NewCredentialRequest(cred.ID, "agent-1", entities.PrincipalAgent, "deploy", 1*time.Minute)

	_, err := manager.RequestCredential(ctx, req)
	if err == nil {
		t.Error("Expected unauthorized error, got nil")
	}
}

func TestCredentialManager_ScopeRestriction(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	cred.AddScope("github.com/myorg")

	encrypted, _ := manager.encryptSecret("ghp_secret123")
	cred.Secret = encrypted
	credRepo.SaveCredential(cred)

	ctx := entities.NewSecurityContext("agent-1", entities.PrincipalAgent)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionRead, "*"))

	req := entities.NewCredentialRequest(cred.ID, "agent-1", entities.PrincipalAgent, "deploy", 1*time.Minute)
	req.AddScope("github.com/otherorg")

	_, err := manager.RequestCredential(ctx, req)
	if err == nil {
		t.Error("Expected scope denial error, got nil")
	}
}

func TestCredentialManager_RotateCredential(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_old_secret")
	credRepo.SaveCredential(cred)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionUpdate, "*"))

	err := manager.RotateCredential(ctx, cred.ID, "ghp_new_secret")
	if err != nil {
		t.Fatalf("RotateCredential failed: %v", err)
	}

	updated, _ := credRepo.GetCredential(cred.ID)
	if updated.LastRotatedAt == nil {
		t.Error("LastRotatedAt should be set after rotation")
	}
}

func TestCredentialManager_RevokeCredential(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	credRepo.SaveCredential(cred)

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionDelete, "*"))

	err := manager.RevokeCredential(ctx, cred.ID)
	if err != nil {
		t.Fatalf("RevokeCredential failed: %v", err)
	}

	updated, _ := credRepo.GetCredential(cred.ID)
	if updated.Active {
		t.Error("Credential should be inactive after revocation")
	}
}

func TestCredentialManager_ExpiredCredential(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	expiry := time.Now().Add(-1 * time.Hour)
	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "github-token")
	cred.SetSecret("ghp_secret123")
	cred.SetExpiry(expiry)

	encrypted, _ := manager.encryptSecret("ghp_secret123")
	cred.Secret = encrypted
	credRepo.SaveCredential(cred)

	ctx := entities.NewSecurityContext("agent-1", entities.PrincipalAgent)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionRead, "*"))

	req := entities.NewCredentialRequest(cred.ID, "agent-1", entities.PrincipalAgent, "deploy", 1*time.Minute)

	_, err := manager.RequestCredential(ctx, req)
	if err == nil {
		t.Error("Expected expired credential error, got nil")
	}
}

func TestCredentialManager_AuditTrail(t *testing.T) {
	credRepo := NewMockCredentialRepository()
	auditRepo := NewMockAuditRepository()
	encKey := make([]byte, 32)
	manager := NewCredentialManager(credRepo, auditRepo, encKey, &MockLogger{})

	ctx := entities.NewSecurityContext("user-1", entities.PrincipalUser)
	ctx.AddPermissions(entities.NewPermission(entities.ResourceCredential, entities.ActionCreate, "*"))

	cred := entities.NewCredential("user-1", entities.PrincipalUser, entities.CredentialTypeAPIKey, "test-cred")
	cred.SetSecret("secret123")

	manager.StoreCredential(ctx, cred)

	entries, _ := auditRepo.GetEntriesByResource(entities.ResourceCredential, cred.ID)
	if len(entries) == 0 {
		t.Error("Expected audit entries, got none")
	}

	if entries[0].Status != entities.AuditSuccess {
		t.Errorf("Expected success status, got %s", entries[0].Status)
	}
}
