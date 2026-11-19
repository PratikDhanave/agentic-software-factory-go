package discovery

import (
	"context"
	"testing"
)

func TestRegisterService(t *testing.T) {
	r := New()
	ctx := context.Background()

	si := &ServiceInstance{
		TenantID: "tenant-1",
		AgentID:  "agent-1",
		Name:     "code-analyzer",
		Address:  "agent-1.example.com",
		Port:     8080,
	}

	err := r.Register(ctx, si)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if si.ID == "" {
		t.Error("service ID is empty")
	}

	if si.Health != HealthHealthy {
		t.Errorf("got health %s, want healthy", si.Health)
	}
}

func TestDiscoverService(t *testing.T) {
	r := New()
	ctx := context.Background()

	si := &ServiceInstance{
		TenantID: "tenant-1",
		AgentID:  "agent-1",
		Name:     "code-analyzer",
		Address:  "agent-1.example.com",
		Port:     8080,
	}

	r.Register(ctx, si)

	instances, err := r.Discover(ctx, "code-analyzer", "tenant-1")
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	if len(instances) != 1 {
		t.Errorf("got %d instances, want 1", len(instances))
	}

	if instances[0].Name != "code-analyzer" {
		t.Error("service name mismatch")
	}
}

func TestHealthCheck(t *testing.T) {
	r := New()
	ctx := context.Background()

	si := &ServiceInstance{
		TenantID: "tenant-1",
		AgentID:  "agent-1",
		Name:     "code-analyzer",
		Address:  "agent-1.example.com",
		Port:     8080,
	}

	r.Register(ctx, si)

	// Mark as unhealthy
	r.HealthCheck(ctx, si.ID, false)

	// Should not appear in discovery
	instances, _ := r.Discover(ctx, "code-analyzer", "tenant-1")
	if len(instances) != 0 {
		t.Error("unhealthy instance should not be discovered")
	}
}

func TestDeregisterService(t *testing.T) {
	r := New()
	ctx := context.Background()

	si := &ServiceInstance{
		TenantID: "tenant-1",
		AgentID:  "agent-1",
		Name:     "code-analyzer",
		Address:  "agent-1.example.com",
		Port:     8080,
	}

	r.Register(ctx, si)

	err := r.Deregister(ctx, si.ID)
	if err != nil {
		t.Fatalf("deregister failed: %v", err)
	}

	instances, _ := r.Discover(ctx, "code-analyzer", "tenant-1")
	if len(instances) != 0 {
		t.Error("deregistered service should not be discovered")
	}
}

func TestListByTenant(t *testing.T) {
	r := New()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		si := &ServiceInstance{
			TenantID: "tenant-1",
			AgentID:  "agent-" + string(rune('1'+i)),
			Name:     "service-" + string(rune('a'+i)),
			Address:  "agent.example.com",
			Port:     8080 + i,
		}
		r.Register(ctx, si)
	}

	instances, _ := r.ListByTenant(ctx, "tenant-1")
	if len(instances) != 3 {
		t.Errorf("got %d instances, want 3", len(instances))
	}
}

func TestCountServices(t *testing.T) {
	r := New()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		si := &ServiceInstance{
			TenantID: "tenant-1",
			AgentID:  "agent-" + string(rune('1'+i)),
			Name:     "service-" + string(rune('a'+i)),
			Address:  "agent.example.com",
			Port:     8080 + i,
		}
		r.Register(ctx, si)
	}

	count, _ := r.Count(ctx)
	if count != 5 {
		t.Errorf("got count %d, want 5", count)
	}
}
