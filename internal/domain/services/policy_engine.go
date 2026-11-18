package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

var _ = sort.Slice

type PolicyEngine struct {
	policies          []*entities.Policy
	rateLimits        map[string]*entities.RateLimitState
	rateLimitConfigs  map[string]*entities.RateLimitConfig
	logger            Logger
}

type PolicyEvaluationRequest struct {
	SecurityContext *entities.SecurityContext
	Resource        entities.Resource
	Action          entities.Action
	ResourceID      string
	Attributes      map[string]interface{}
}

func NewPolicyEngine(logger Logger) *PolicyEngine {
	return &PolicyEngine{
		policies:         []*entities.Policy{},
		rateLimits:       make(map[string]*entities.RateLimitState),
		rateLimitConfigs: make(map[string]*entities.RateLimitConfig),
		logger:           logger,
	}
}

func (pe *PolicyEngine) AddPolicy(p *entities.Policy) {
	pe.policies = append(pe.policies, p)
	sort.Slice(pe.policies, func(i, j int) bool {
		return pe.policies[i].Priority > pe.policies[j].Priority
	})
	pe.logger.Debug("policy added", "policy_id", p.ID, "policy_name", p.Name)
}

func (pe *PolicyEngine) SetRateLimitConfig(metric string, config *entities.RateLimitConfig) {
	pe.rateLimitConfigs[metric] = config
	pe.logger.Debug("rate limit config set", "metric", metric, "max_count", config.MaxCount)
}

func (pe *PolicyEngine) Evaluate(req *PolicyEvaluationRequest) (*entities.PolicyEvaluationResult, error) {
	if !req.SecurityContext.PrincipalType.IsValid() {
		return nil, fmt.Errorf("invalid principal type")
	}

	type policyMatch struct {
		policy   *entities.Policy
		result   *entities.PolicyEvaluationResult
		priority int
	}

	denials := []policyMatch{}
	allowals := []policyMatch{}
	requireApprovals := []policyMatch{}

	for _, policy := range pe.policies {
		if !policy.Active {
			continue
		}

		result := pe.evaluatePolicy(policy, req)
		if result == nil {
			continue
		}

		match := policyMatch{
			policy:   policy,
			result:   result,
			priority: policy.Priority,
		}

		switch result.Decision {
		case entities.DecisionDeny:
			denials = append(denials, match)
		case entities.DecisionAllow:
			allowals = append(allowals, match)
		case entities.DecisionRequireApproval:
			requireApprovals = append(requireApprovals, match)
		}
	}

	sort.Slice(denials, func(i, j int) bool {
		return denials[i].priority > denials[j].priority
	})
	sort.Slice(allowals, func(i, j int) bool {
		return allowals[i].priority > allowals[j].priority
	})
	sort.Slice(requireApprovals, func(i, j int) bool {
		return requireApprovals[i].priority > requireApprovals[j].priority
	})

	if len(denials) > 0 {
		result := denials[0].result
		pe.logger.Info("policy evaluation: DENY",
			"principal", req.SecurityContext.PrincipalID,
			"action", req.Action,
			"resource", req.Resource,
			"reason", result.Reason,
		)
		return result, nil
	}

	if len(requireApprovals) > 0 {
		result := requireApprovals[0].result
		pe.logger.Info("policy evaluation: REQUIRE_APPROVAL",
			"principal", req.SecurityContext.PrincipalID,
			"action", req.Action,
			"resource", req.Resource,
		)
		return result, nil
	}

	rateLimitResult := pe.checkRateLimit(req.SecurityContext.PrincipalID)
	if rateLimitResult != nil {
		pe.logger.Info("policy evaluation: RATE_LIMITED",
			"principal", req.SecurityContext.PrincipalID,
		)
		return rateLimitResult, nil
	}

	if len(allowals) > 0 {
		result := allowals[0].result
		result.Decision = entities.DecisionAllow
		pe.logger.Info("policy evaluation: ALLOW",
			"principal", req.SecurityContext.PrincipalID,
			"action", req.Action,
			"resource", req.Resource,
		)
		return result, nil
	}

	result := entities.NewPolicyEvaluationResult(
		entities.DecisionDeny,
		"no matching allow policy found",
	)
	pe.logger.Info("policy evaluation: DENY (default)",
		"principal", req.SecurityContext.PrincipalID,
		"action", req.Action,
		"resource", req.Resource,
	)
	return result, nil
}

func (pe *PolicyEngine) evaluatePolicy(policy *entities.Policy, req *PolicyEvaluationRequest) *entities.PolicyEvaluationResult {
	for _, stmt := range policy.Statements {
		if pe.statementMatches(stmt, req) {
			if stmt.Effect == entities.EffectDeny {
				result := entities.NewPolicyEvaluationResult(
					entities.DecisionDeny,
					fmt.Sprintf("denied by policy %s", policy.Name),
				)
				result.AddMatchedPolicy(policy)
				return result
			}

			if stmt.Effect == entities.EffectAllow {
				needsApproval := pe.requiresApproval(stmt, req)
				if needsApproval {
					result := entities.NewPolicyEvaluationResult(
						entities.DecisionRequireApproval,
						fmt.Sprintf("policy %s requires approval", policy.Name),
					)
					result.AddMatchedPolicy(policy)
					return result
				}

				result := entities.NewPolicyEvaluationResult(
					entities.DecisionAllow,
					fmt.Sprintf("allowed by policy %s", policy.Name),
				)
				result.AddMatchedPolicy(policy)
				return result
			}
		}
	}

	return nil
}

func (pe *PolicyEngine) statementMatches(stmt *entities.Statement, req *PolicyEvaluationRequest) bool {
	if !pe.resourcesMatch(stmt.Resource, req.Resource) {
		return false
	}

	if !pe.actionsMatch(stmt.Action, req.Action) {
		return false
	}

	if !pe.principalsMatch(stmt.Principal, req.SecurityContext.PrincipalType) {
		return false
	}

	for _, condition := range stmt.Conditions {
		if !pe.evaluateCondition(condition, req) {
			return false
		}
	}

	return true
}

func (pe *PolicyEngine) resourcesMatch(statementResources []entities.Resource, requestResource entities.Resource) bool {
	if len(statementResources) == 0 {
		return true
	}

	for _, r := range statementResources {
		if r == "*" || r == requestResource {
			return true
		}
	}

	return false
}

func (pe *PolicyEngine) actionsMatch(statementActions []entities.Action, requestAction entities.Action) bool {
	if len(statementActions) == 0 {
		return true
	}

	for _, a := range statementActions {
		if a == "*" || a == requestAction {
			return true
		}
	}

	return false
}

func (pe *PolicyEngine) principalsMatch(statementPrincipals []entities.PrincipalType, requestPrincipal entities.PrincipalType) bool {
	if len(statementPrincipals) == 0 {
		return true
	}

	for _, p := range statementPrincipals {
		if p == "*" || p == requestPrincipal {
			return true
		}
	}

	return false
}

func (pe *PolicyEngine) evaluateCondition(cond *entities.Condition, req *PolicyEvaluationRequest) bool {
	switch cond.Type {
	case "time-based":
		return pe.evaluateTimeCondition(cond)
	case "rate-based":
		return pe.evaluateRateCondition(cond)
	case "attribute-based":
		return pe.evaluateAttributeCondition(cond, req)
	default:
		return false
	}
}

func (pe *PolicyEngine) evaluateTimeCondition(cond *entities.Condition) bool {
	now := time.Now()

	if startStr, ok := cond.Meta["start"].(string); ok {
		start, _ := time.Parse(time.RFC3339, startStr)
		if now.Before(start) {
			return false
		}
	}

	if endStr, ok := cond.Meta["end"].(string); ok {
		end, _ := time.Parse(time.RFC3339, endStr)
		if now.After(end) {
			return false
		}
	}

	return true
}

func (pe *PolicyEngine) evaluateRateCondition(cond *entities.Condition) bool {
	return true
}

func (pe *PolicyEngine) evaluateAttributeCondition(cond *entities.Condition, req *PolicyEvaluationRequest) bool {
	if value, ok := req.Attributes[cond.Type]; ok {
		switch cond.Operator {
		case "equals":
			return value == cond.Value
		case "contains":
			return false
		default:
			return false
		}
	}

	return false
}

func (pe *PolicyEngine) requiresApproval(stmt *entities.Statement, req *PolicyEvaluationRequest) bool {
	for _, cond := range stmt.Conditions {
		if cond.Type == "approval-required" {
			return true
		}
	}

	return false
}

func (pe *PolicyEngine) checkRateLimit(principalID string) *entities.PolicyEvaluationResult {
	config, exists := pe.rateLimitConfigs["default"]
	if !exists {
		return nil
	}

	key := principalID + ":default"
	state, exists := pe.rateLimits[key]

	if !exists || state.IsExpired() {
		state = entities.NewRateLimitState(principalID, "default", time.Now().Add(config.Window))
		pe.rateLimits[key] = state
	}

	count := state.Increment()
	if count > config.MaxCount {
		result := entities.NewPolicyEvaluationResult(
			entities.DecisionRateLimited,
			fmt.Sprintf("rate limit exceeded: %d/%d", count-1, config.MaxCount),
		)
		return result
	}

	return nil
}
