package tenants

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateNetworkPolicy creates a NetworkPolicy for tenant isolation
func (m *Manager) GenerateNetworkPolicy(ctx context.Context, tenantID string) (*networkingv1.NetworkPolicy, error) {
	tenant, err := m.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("tenant-isolation-%s", tenantID),
			Namespace: tenant.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "factory-system",
								},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "factory-system",
								},
							},
						},
					},
				},
			},
		},
	}

	return policy, nil
}

// PolicyAllowsIngress checks if a network policy allows ingress from source tenant
func (m *Manager) PolicyAllowsIngress(ctx context.Context, policy *networkingv1.NetworkPolicy, sourceTenantID string) (bool, error) {
	if policy == nil {
		return false, fmt.Errorf("policy is nil")
	}

	for _, rule := range policy.Spec.Ingress {
		for _, peer := range rule.From {
			if peer.NamespaceSelector != nil {
				sourceTenant, err := m.Get(ctx, sourceTenantID)
				if err != nil {
					return false, fmt.Errorf("get source tenant: %w", err)
				}

				if peer.NamespaceSelector.MatchLabels["name"] == "factory-system" {
					return true, nil
				}

				if peer.NamespaceSelector.MatchLabels["name"] == sourceTenant.Namespace {
					return true, nil
				}
			}

			if peer.PodSelector != nil {
				return true, nil
			}
		}
	}

	return false, nil
}

// ValidateNetworkPolicy validates that a policy structure is correct
func ValidateNetworkPolicy(policy *networkingv1.NetworkPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}

	if policy.Namespace == "" {
		return fmt.Errorf("policy namespace is empty")
	}

	if policy.Name == "" {
		return fmt.Errorf("policy name is empty")
	}

	if len(policy.Spec.PolicyTypes) == 0 {
		return fmt.Errorf("policy types not specified")
	}

	return nil
}
