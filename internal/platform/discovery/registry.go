package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HealthStatus represents the health of a service instance
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthDraining  HealthStatus = "draining"
)

// ServiceInstance represents a registered service
type ServiceInstance struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	AgentID   string            `json:"agent_id"`
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Health    HealthStatus      `json:"health"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Registry manages service discovery
type Registry struct {
	instances map[string]*ServiceInstance
	watchers  map[string][]chan *ServiceInstance
	mu        sync.RWMutex
}

// New creates a new service registry
func New() *Registry {
	return &Registry{
		instances: make(map[string]*ServiceInstance),
		watchers:  make(map[string][]chan *ServiceInstance),
	}
}

// Register adds a service to the registry
func (r *Registry) Register(ctx context.Context, si *ServiceInstance) error {
	if si.Name == "" {
		return fmt.Errorf("service name required")
	}
	if si.TenantID == "" {
		return fmt.Errorf("tenant ID required")
	}
	if si.Address == "" {
		return fmt.Errorf("service address required")
	}
	if si.Port == 0 {
		return fmt.Errorf("service port required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	si.ID = uuid.New().String()
	si.CreatedAt = now
	si.UpdatedAt = now
	si.Health = HealthHealthy

	r.instances[si.ID] = si

	// Notify watchers
	r.notifyWatchers(si.Name, si)

	return nil
}

// Deregister removes a service from the registry
func (r *Registry) Deregister(ctx context.Context, serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[serviceID]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	delete(r.instances, serviceID)

	// Notify watchers
	r.notifyWatchers(instance.Name, nil)

	return nil
}

// Discover finds all healthy instances of a service in a tenant
func (r *Registry) Discover(ctx context.Context, serviceName string, tenantID string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []*ServiceInstance
	for _, si := range r.instances {
		if si.Name == serviceName && si.TenantID == tenantID && si.Health == HealthHealthy {
			instances = append(instances, si)
		}
	}

	return instances, nil
}

// DiscoverAll discovers all instances of a service (across tenants if allowed by policy)
func (r *Registry) DiscoverAll(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []*ServiceInstance
	for _, si := range r.instances {
		if si.Name == serviceName && si.Health == HealthHealthy {
			instances = append(instances, si)
		}
	}

	return instances, nil
}

// Get retrieves a specific service instance
func (r *Registry) Get(ctx context.Context, serviceID string) (*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, ok := r.instances[serviceID]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return instance, nil
}

// HealthCheck updates the health status of a service
func (r *Registry) HealthCheck(ctx context.Context, serviceID string, healthy bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[serviceID]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	oldHealth := instance.Health
	if healthy {
		instance.Health = HealthHealthy
	} else {
		instance.Health = HealthUnhealthy
	}
	instance.UpdatedAt = time.Now().UTC()

	// Notify watchers if health changed
	if oldHealth != instance.Health {
		r.notifyWatchers(instance.Name, instance)
	}

	return nil
}

// Watch watches for changes to a service
func (r *Registry) Watch(ctx context.Context, serviceName string) chan *ServiceInstance {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan *ServiceInstance, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], ch)

	return ch
}

// notifyWatchers sends notifications to all watchers of a service
func (r *Registry) notifyWatchers(serviceName string, instance *ServiceInstance) {
	watchers, ok := r.watchers[serviceName]
	if !ok {
		return
	}

	for _, ch := range watchers {
		select {
		case ch <- instance:
		default:
			// Channel full, skip notification
		}
	}
}

// ListByTenant returns all services registered for a tenant
func (r *Registry) ListByTenant(ctx context.Context, tenantID string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []*ServiceInstance
	for _, si := range r.instances {
		if si.TenantID == tenantID {
			instances = append(instances, si)
		}
	}

	return instances, nil
}

// ListByAgent returns all services registered by an agent
func (r *Registry) ListByAgent(ctx context.Context, agentID string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []*ServiceInstance
	for _, si := range r.instances {
		if si.AgentID == agentID {
			instances = append(instances, si)
		}
	}

	return instances, nil
}

// Count returns the number of registered services
func (r *Registry) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.instances), nil
}
