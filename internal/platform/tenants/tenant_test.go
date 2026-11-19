package tenants

import (
	"context"
	"testing"
)

func TestRegisterTenant(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "team-a", map[string]string{"region": "us-west"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if tenant.ID == "" {
		t.Error("tenant ID is empty")
	}

	if tenant.Name != "team-a" {
		t.Errorf("got name %s, want team-a", tenant.Name)
	}

	if tenant.Namespace != "factory-tenant-team-a" {
		t.Errorf("got namespace %s, want factory-tenant-team-a", tenant.Namespace)
	}

	if tenant.Status != "active" {
		t.Errorf("got status %s, want active", tenant.Status)
	}
}

func TestGetTenant(t *testing.T) {
	m := New()
	ctx := context.Background()

	registered, _ := m.Register(ctx, "team-b", nil)

	retrieved, err := m.Get(ctx, registered.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if retrieved.Name != registered.Name {
		t.Errorf("name mismatch")
	}
}

func TestGetTenantByName(t *testing.T) {
	m := New()
	ctx := context.Background()

	registered, _ := m.Register(ctx, "team-c", nil)

	retrieved, err := m.GetByName(ctx, "team-c")
	if err != nil {
		t.Fatalf("get by name failed: %v", err)
	}

	if retrieved.ID != registered.ID {
		t.Error("ID mismatch")
	}
}

func TestListTenants(t *testing.T) {
	m := New()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		m.Register(ctx, "team-"+string(rune('a'+i)), nil)
	}

	tenants, err := m.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(tenants) != 3 {
		t.Errorf("got %d tenants, want 3", len(tenants))
	}
}

func TestSetPolicy(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, _ := m.Register(ctx, "team-d", nil)

	err := m.SetPolicy(ctx, tenant.ID, "policy-123")
	if err != nil {
		t.Fatalf("set policy failed: %v", err)
	}

	retrieved, _ := m.Get(ctx, tenant.ID)
	if retrieved.PolicyID != "policy-123" {
		t.Errorf("got policy %s, want policy-123", retrieved.PolicyID)
	}
}

func TestSetQuota(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, _ := m.Register(ctx, "team-e", nil)

	err := m.SetQuota(ctx, tenant.ID, 100)
	if err != nil {
		t.Fatalf("set quota failed: %v", err)
	}

	retrieved, _ := m.Get(ctx, tenant.ID)
	if retrieved.QuotaMaxAgents != 100 {
		t.Errorf("got quota %d, want 100", retrieved.QuotaMaxAgents)
	}
}

func TestDeleteTenant(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, _ := m.Register(ctx, "team-f", nil)

	err := m.Delete(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = m.Get(ctx, tenant.ID)
	if err == nil {
		t.Error("expected error when getting deleted tenant")
	}
}

func TestDuplicateTenantName(t *testing.T) {
	m := New()
	ctx := context.Background()

	m.Register(ctx, "team-dup", nil)

	_, err := m.Register(ctx, "team-dup", nil)
	if err == nil {
		t.Error("expected error for duplicate tenant name")
	}
}

func TestCountTenants(t *testing.T) {
	m := New()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m.Register(ctx, "team-"+string(rune('a'+i)), nil)
	}

	count, err := m.Count(ctx)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}

	if count != 5 {
		t.Errorf("got count %d, want 5", count)
	}
}
