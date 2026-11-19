package tenants

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceQuotaConfig holds quota limits for a tenant
type ResourceQuotaConfig struct {
	MaxPods             int64
	MaxCPU              string
	MaxMemory           string
	MaxStorage          string
	MaxPersistentVolumes int64
}

// DefaultResourceQuotaConfig returns default quota configuration
func DefaultResourceQuotaConfig() *ResourceQuotaConfig {
	return &ResourceQuotaConfig{
		MaxPods:              100,
		MaxCPU:               "10",
		MaxMemory:            "20Gi",
		MaxStorage:           "100Gi",
		MaxPersistentVolumes: 10,
	}
}

// GenerateResourceQuota creates a Kubernetes ResourceQuota for a tenant
func (m *Manager) GenerateResourceQuota(ctx context.Context, tenantID string) (*corev1.ResourceQuota, error) {
	tenant, err := m.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	config := DefaultResourceQuotaConfig()

	// Override MaxPods with tenant quota if set
	if tenant.QuotaMaxAgents > 0 {
		config.MaxPods = int64(tenant.QuotaMaxAgents)
	}

	cpuQuantity, err := resource.ParseQuantity(config.MaxCPU)
	if err != nil {
		return nil, fmt.Errorf("parse CPU quantity: %w", err)
	}

	memoryQuantity, err := resource.ParseQuantity(config.MaxMemory)
	if err != nil {
		return nil, fmt.Errorf("parse memory quantity: %w", err)
	}

	storageQuantity, err := resource.ParseQuantity(config.MaxStorage)
	if err != nil {
		return nil, fmt.Errorf("parse storage quantity: %w", err)
	}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("tenant-quota-%s", tenantID),
			Namespace: tenant.Namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods:                   *resource.NewQuantity(config.MaxPods, resource.DecimalSI),
				corev1.ResourceCPU:                    cpuQuantity,
				corev1.ResourceMemory:                 memoryQuantity,
				corev1.ResourceStorage:                storageQuantity,
				corev1.ResourcePersistentVolumeClaims: *resource.NewQuantity(config.MaxPersistentVolumes, resource.DecimalSI),
			},
		},
	}

	return quota, nil
}

// QuotaExceeded checks if a quota is at or exceeding its limit
func (m *Manager) QuotaExceeded(ctx context.Context, quota *corev1.ResourceQuota, used corev1.ResourceList) (bool, error) {
	if quota == nil {
		return false, fmt.Errorf("quota is nil")
	}

	for resource, limit := range quota.Spec.Hard {
		usedQuantity, exists := used[resource]
		if !exists {
			continue
		}

		if usedQuantity.Cmp(limit) >= 0 {
			return true, nil
		}
	}

	return false, nil
}

// ValidateResourceQuota validates a quota configuration
func ValidateResourceQuota(quota *corev1.ResourceQuota) error {
	if quota == nil {
		return fmt.Errorf("quota is nil")
	}

	if quota.Namespace == "" {
		return fmt.Errorf("quota namespace is empty")
	}

	if quota.Name == "" {
		return fmt.Errorf("quota name is empty")
	}

	if len(quota.Spec.Hard) == 0 {
		return fmt.Errorf("quota hard limits are empty")
	}

	return nil
}

// GetQuotaUtilization returns current resource utilization (simulated)
func GetQuotaUtilization(quota *corev1.ResourceQuota, used corev1.ResourceList) map[string]float64 {
	utilization := make(map[string]float64)

	for resource, limit := range quota.Spec.Hard {
		usedQuantity, exists := used[resource]
		if !exists {
			utilization[resource.String()] = 0.0
			continue
		}

		pct := float64(usedQuantity.Value()) / float64(limit.Value()) * 100.0
		utilization[resource.String()] = pct
	}

	return utilization
}
