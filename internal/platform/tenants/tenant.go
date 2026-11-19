package tenants

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tenant represents an isolated organization/team
type Tenant struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	CreatedAt      time.Time         `json:"created_at"`
	StoragePath    string            `json:"storage_path"`
	PolicyID       string            `json:"policy_id"`
	CredentialTTL  time.Duration     `json:"credential_ttl"`
	QuotaMaxAgents int               `json:"quota_max_agents"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Status         string            `json:"status"` // active, inactive, deleted
}

// Manager manages tenant lifecycle
type Manager struct {
	tenants map[string]*Tenant
	mu      sync.RWMutex
}

// New creates a new tenant manager
func New() *Manager {
	return &Manager{
		tenants: make(map[string]*Tenant),
	}
}

// Register creates a new tenant
func (m *Manager) Register(ctx context.Context, name string, metadata map[string]string) (*Tenant, error) {
	if name == "" {
		return nil, fmt.Errorf("tenant name required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, t := range m.tenants {
		if t.Name == name && t.Status != "deleted" {
			return nil, fmt.Errorf("tenant %s already exists", name)
		}
	}

	tenantID := uuid.New().String()
	namespace := fmt.Sprintf("factory-tenant-%s", name)

	tenant := &Tenant{
		ID:             tenantID,
		Name:           name,
		Namespace:      namespace,
		CreatedAt:      time.Now().UTC(),
		StoragePath:    fmt.Sprintf("/data/tenants/%s", tenantID),
		PolicyID:       "",
		CredentialTTL:  24 * time.Hour, // default
		QuotaMaxAgents: 50,
		Metadata:       metadata,
		Status:         "active",
	}

	m.tenants[tenantID] = tenant
	return tenant, nil
}

// Get retrieves a tenant by ID
func (m *Manager) Get(ctx context.Context, tenantID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	if tenant.Status == "deleted" {
		return nil, fmt.Errorf("tenant is deleted: %s", tenantID)
	}

	return tenant, nil
}

// GetByName retrieves a tenant by name
func (m *Manager) GetByName(ctx context.Context, name string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tenant := range m.tenants {
		if tenant.Name == name && tenant.Status != "deleted" {
			return tenant, nil
		}
	}

	return nil, fmt.Errorf("tenant not found: %s", name)
}

// List returns all active tenants
func (m *Manager) List(ctx context.Context) ([]*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tenants []*Tenant
	for _, tenant := range m.tenants {
		if tenant.Status != "deleted" {
			tenants = append(tenants, tenant)
		}
	}

	return tenants, nil
}

// Update updates tenant metadata
func (m *Manager) Update(ctx context.Context, tenantID string, metadata map[string]string) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	if tenant.Status == "deleted" {
		return nil, fmt.Errorf("cannot update deleted tenant")
	}

	if metadata != nil {
		tenant.Metadata = metadata
	}

	return tenant, nil
}

// SetPolicy assigns a policy to a tenant
func (m *Manager) SetPolicy(ctx context.Context, tenantID string, policyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	if tenant.Status == "deleted" {
		return fmt.Errorf("cannot update deleted tenant")
	}

	tenant.PolicyID = policyID
	return nil
}

// SetCredentialTTL sets the credential time-to-live for a tenant
func (m *Manager) SetCredentialTTL(ctx context.Context, tenantID string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	if tenant.Status == "deleted" {
		return fmt.Errorf("cannot update deleted tenant")
	}

	if ttl < 1*time.Minute || ttl > 365*24*time.Hour {
		return fmt.Errorf("credential TTL must be between 1 minute and 1 year")
	}

	tenant.CredentialTTL = ttl
	return nil
}

// SetQuota sets the maximum agents for a tenant
func (m *Manager) SetQuota(ctx context.Context, tenantID string, maxAgents int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	if tenant.Status == "deleted" {
		return fmt.Errorf("cannot update deleted tenant")
	}

	if maxAgents < 1 {
		return fmt.Errorf("quota must be at least 1")
	}

	tenant.QuotaMaxAgents = maxAgents
	return nil
}

// Delete marks a tenant as deleted (soft delete)
func (m *Manager) Delete(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	tenant.Status = "deleted"
	return nil
}

// Count returns the number of active tenants
func (m *Manager) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, tenant := range m.tenants {
		if tenant.Status != "deleted" {
			count++
		}
	}

	return count, nil
}
