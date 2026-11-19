package tenants

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateResourceQuota(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	quota, err := m.GenerateResourceQuota(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate quota: %v", err)
	}

	if quota == nil {
		t.Fatal("quota is nil")
	}

	if quota.Namespace != tenant.Namespace {
		t.Errorf("expected namespace %s, got %s", tenant.Namespace, quota.Namespace)
	}

	if len(quota.Spec.Hard) == 0 {
		t.Fatal("quota hard limits are empty")
	}
}

func TestQuotaCPULimit(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	quota, err := m.GenerateResourceQuota(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate quota: %v", err)
	}

	cpuLimit, exists := quota.Spec.Hard[corev1.ResourceCPU]
	if !exists {
		t.Fatal("CPU limit not found in quota")
	}

	expectedCPU, _ := resource.ParseQuantity("10")
	if cpuLimit.Cmp(expectedCPU) != 0 {
		t.Errorf("expected CPU %v, got %v", expectedCPU, cpuLimit)
	}
}

func TestQuotaMemoryLimit(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	quota, err := m.GenerateResourceQuota(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate quota: %v", err)
	}

	memLimit, exists := quota.Spec.Hard[corev1.ResourceMemory]
	if !exists {
		t.Fatal("Memory limit not found in quota")
	}

	expectedMem, _ := resource.ParseQuantity("20Gi")
	if memLimit.Cmp(expectedMem) != 0 {
		t.Errorf("expected memory %v, got %v", expectedMem, memLimit)
	}
}

func TestQuotaExceeded(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	quota, err := m.GenerateResourceQuota(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate quota: %v", err)
	}

	tests := []struct {
		name      string
		used      corev1.ResourceList
		exceeded  bool
	}{
		{
			name:      "no usage",
			used:      corev1.ResourceList{},
			exceeded:  false,
		},
		{
			name: "partial CPU usage",
			used: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity(5, resource.DecimalSI),
			},
			exceeded: false,
		},
		{
			name: "exceeds CPU",
			used: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity(11, resource.DecimalSI),
			},
			exceeded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exceeded, err := m.QuotaExceeded(ctx, quota, tt.used)
			if err != nil {
				t.Fatalf("failed to check quota: %v", err)
			}

			if exceeded != tt.exceeded {
				t.Errorf("expected exceeded=%v, got %v", tt.exceeded, exceeded)
			}
		})
	}
}

func TestValidateResourceQuota(t *testing.T) {
	tests := []struct {
		name        string
		quota       *corev1.ResourceQuota
		shouldError bool
	}{
		{
			name:        "nil quota",
			quota:       nil,
			shouldError: true,
		},
		{
			name: "valid quota",
			quota: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-quota",
					Namespace: "test-ns",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: corev1.ResourceList{
						corev1.ResourceCPU: *resource.NewQuantity(10, resource.DecimalSI),
					},
				},
			},
			shouldError: false,
		},
		{
			name: "missing namespace",
			quota: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-quota",
				},
				Spec: corev1.ResourceQuotaSpec{
					Hard: corev1.ResourceList{
						corev1.ResourceCPU: *resource.NewQuantity(10, resource.DecimalSI),
					},
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceQuota(tt.quota)
			if (err != nil) != tt.shouldError {
				t.Errorf("expected error=%v, got %v", tt.shouldError, err)
			}
		})
	}
}

func TestGetQuotaUtilization(t *testing.T) {
	config := DefaultResourceQuotaConfig()

	cpuLimit, _ := resource.ParseQuantity(config.MaxCPU)
	memLimit, _ := resource.ParseQuantity(config.MaxMemory)

	quota := &corev1.ResourceQuota{
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:    cpuLimit,
				corev1.ResourceMemory: memLimit,
			},
		},
	}

	used := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewQuantity(5, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(10*1024*1024*1024, resource.BinarySI), // 10Gi
	}

	util := GetQuotaUtilization(quota, used)

	if len(util) == 0 {
		t.Fatal("utilization is empty")
	}

	if cpuUtil, exists := util["cpu"]; exists {
		if cpuUtil != 50.0 {
			t.Errorf("expected CPU utilization 50%%, got %f%%", cpuUtil)
		}
	}
}
