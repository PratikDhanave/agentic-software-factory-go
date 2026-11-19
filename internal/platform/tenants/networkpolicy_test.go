package tenants

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateNetworkPolicy(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	policy, err := m.GenerateNetworkPolicy(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate policy: %v", err)
	}

	if policy == nil {
		t.Fatal("policy is nil")
	}

	if policy.Namespace != tenant.Namespace {
		t.Errorf("expected namespace %s, got %s", tenant.Namespace, policy.Namespace)
	}

	if len(policy.Spec.PolicyTypes) != 2 {
		t.Errorf("expected 2 policy types (Ingress, Egress), got %d", len(policy.Spec.PolicyTypes))
	}
}

func TestNetworkPolicyIngress(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	policy, err := m.GenerateNetworkPolicy(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate policy: %v", err)
	}

	if len(policy.Spec.Ingress) == 0 {
		t.Fatal("ingress rules are empty")
	}

	ingressRule := policy.Spec.Ingress[0]
	if len(ingressRule.From) == 0 {
		t.Fatal("ingress From peers are empty")
	}

	hasNamespacePeer := false
	for _, peer := range ingressRule.From {
		if peer.NamespaceSelector != nil && peer.NamespaceSelector.MatchLabels["name"] == "factory-system" {
			hasNamespacePeer = true
			break
		}
	}

	if !hasNamespacePeer {
		t.Fatal("ingress does not allow factory-system namespace")
	}
}

func TestNetworkPolicyEgress(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant, err := m.Register(ctx, "test-tenant", nil)
	if err != nil {
		t.Fatalf("failed to register tenant: %v", err)
	}

	policy, err := m.GenerateNetworkPolicy(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("failed to generate policy: %v", err)
	}

	if len(policy.Spec.Egress) == 0 {
		t.Fatal("egress rules are empty")
	}

	egressRule := policy.Spec.Egress[0]
	if len(egressRule.To) == 0 {
		t.Fatal("egress To peers are empty")
	}

	hasNamespacePeer := false
	for _, peer := range egressRule.To {
		if peer.NamespaceSelector != nil && peer.NamespaceSelector.MatchLabels["name"] == "factory-system" {
			hasNamespacePeer = true
			break
		}
	}

	if !hasNamespacePeer {
		t.Fatal("egress does not allow factory-system namespace")
	}
}

func TestPolicyAllowsIngress(t *testing.T) {
	m := New()
	ctx := context.Background()

	tenant1, err := m.Register(ctx, "tenant1", nil)
	if err != nil {
		t.Fatalf("failed to register tenant1: %v", err)
	}

	tenant2, err := m.Register(ctx, "tenant2", nil)
	if err != nil {
		t.Fatalf("failed to register tenant2: %v", err)
	}

	policy, err := m.GenerateNetworkPolicy(ctx, tenant1.ID)
	if err != nil {
		t.Fatalf("failed to generate policy: %v", err)
	}

	// Policy allows ingress from same namespace (pod selector) or factory-system namespace
	// So tenant2 should be allowed since the PodSelector matches all pods in the namespace
	allowed, err := m.PolicyAllowsIngress(ctx, policy, tenant2.ID)
	if err != nil {
		t.Fatalf("failed to check ingress: %v", err)
	}

	// The policy allows any peer (empty PodSelector matches all), so this should be allowed
	if !allowed {
		t.Fatal("expected ingress to be allowed for peer in same namespace")
	}
}

func TestValidateNetworkPolicy(t *testing.T) {
	tests := []struct {
		name        string
		policy      *networkingv1.NetworkPolicy
		shouldError bool
	}{
		{
			name:        "nil policy",
			policy:      nil,
			shouldError: true,
		},
		{
			name: "valid policy",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				},
			},
			shouldError: false,
		},
		{
			name: "missing namespace",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				},
			},
			shouldError: true,
		},
		{
			name: "missing name",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkPolicy(tt.policy)
			if (err != nil) != tt.shouldError {
				t.Errorf("expected error=%v, got %v", tt.shouldError, err)
			}
		})
	}
}
