package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/platform/credentials"
	"github.com/google/uuid"
)

// Policy defines a set of rules for access control.
type Policy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rules       []Rule    `json:"rules"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Enabled     bool      `json:"enabled"`
}

// Rule defines a single access control rule.
type Rule struct {
	ID         string        `json:"id"`
	Subject    string        `json:"subject"` // e.g., "code:*", "*"
	Actions    []string      `json:"actions"` // e.g., ["read", "write"]
	Effect     string        `json:"effect"`  // "allow" or "deny"
	Conditions []Condition   `json:"conditions,omitempty"`
}

// Condition is a constraint on when a rule applies.
type Condition struct {
	Field    string      `json:"field"`    // e.g., "agent.type"
	Operator string      `json:"operator"` // "equals", "in", "contains"
	Value    interface{} `json:"value"`
}

// EvaluationRequest is used to evaluate a policy decision.
type EvaluationRequest struct {
	AgentID    string
	Credential *credentials.Credential
	Subject    string
	Action     string
	Context    map[string]interface{} // Additional context (agent.type, time, etc.)
}

// EvaluationResult is the outcome of a policy evaluation.
type EvaluationResult struct {
	Allowed    bool
	Reason     string
	MatchedRule *Rule
	Subject    string
	Action     string
	Timestamp  time.Time
}

// Engine evaluates policies.
type Engine struct {
	policies []*Policy
	broker   *credentials.Broker
}

// NewEngine creates a new policy engine.
func NewEngine(broker *credentials.Broker) *Engine {
	return &Engine{
		policies: make([]*Policy, 0),
		broker:   broker,
	}
}

// AddPolicy registers a new policy.
func (e *Engine) AddPolicy(ctx context.Context, policy *Policy) error {
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now().UTC()
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = time.Now().UTC()
	}

	e.policies = append(e.policies, policy)
	return nil
}

// Evaluate determines if an action is allowed by policy.
func (e *Engine) Evaluate(ctx context.Context, req *EvaluationRequest) *EvaluationResult {
	result := &EvaluationResult{
		Allowed:   false,
		Reason:    "default deny",
		Subject:   req.Subject,
		Action:    req.Action,
		Timestamp: time.Now().UTC(),
	}

	// 1. Verify credential is valid
	if req.Credential == nil {
		result.Reason = "no credential provided"
		return result
	}

	valid, reason, err := e.broker.Verify(ctx, req.Credential)
	if err != nil {
		result.Reason = fmt.Sprintf("credential verification error: %v", err)
		return result
	}

	if !valid {
		result.Reason = fmt.Sprintf("credential invalid: %s", reason)
		return result
	}

	// 2. Evaluate policies (in order)
	for _, policy := range e.policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			// Check if rule applies to this subject/action
			if !e.ruleMatches(&rule, req) {
				continue
			}

			// Check if conditions are met
			if !e.conditionsMet(&rule, req) {
				continue
			}

			result.MatchedRule = &rule
			result.Allowed = rule.Effect == "allow"
			result.Reason = fmt.Sprintf("matched policy %s rule %s", policy.Name, rule.ID)
			return result
		}
	}

	// No matching rule found - deny by default
	result.Reason = "no matching policy rule"
	return result
}

// ruleMatches checks if a rule applies to the request.
func (e *Engine) ruleMatches(rule *Rule, req *EvaluationRequest) bool {
	// Check subject
	if rule.Subject != "*" && !e.subjectMatches(rule.Subject, req.Subject) {
		return false
	}

	// Check action
	actionMatches := false
	for _, ruleAction := range rule.Actions {
		if ruleAction == "*" || ruleAction == req.Action {
			actionMatches = true
			break
		}
	}

	return actionMatches
}

// subjectMatches checks if a subject pattern matches a request subject.
func (e *Engine) subjectMatches(pattern string, subject string) bool {
	if pattern == "*" {
		return true
	}

	// Support wildcard patterns like "code:*"
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(subject, prefix+":")
	}

	return pattern == subject
}

// conditionsMet checks if all conditions for a rule are satisfied.
func (e *Engine) conditionsMet(rule *Rule, req *EvaluationRequest) bool {
	if len(rule.Conditions) == 0 {
		return true
	}

	for _, cond := range rule.Conditions {
		if !e.conditionMet(&cond, req) {
			return false
		}
	}

	return true
}

// conditionMet checks a single condition.
func (e *Engine) conditionMet(cond *Condition, req *EvaluationRequest) bool {
	var fieldValue interface{}

	// Resolve field from context
	switch cond.Field {
	case "agent.type":
		// Would need to get agent type from identity system
		// For now, check context
		if v, ok := req.Context["agent.type"]; ok {
			fieldValue = v
		}
	case "credential.subject":
		fieldValue = req.Credential.Subject
	case "time.hour":
		fieldValue = time.Now().Hour()
	default:
		// Try to get from context
		if v, ok := req.Context[cond.Field]; ok {
			fieldValue = v
		}
	}

	if fieldValue == nil {
		return false
	}

	// Evaluate condition
	switch cond.Operator {
	case "equals":
		return fieldValue == cond.Value
	case "in":
		// Check if fieldValue is in cond.Value (assumed to be a slice)
		if values, ok := cond.Value.([]interface{}); ok {
			for _, v := range values {
				if fieldValue == v {
					return true
				}
			}
		}
		return false
	case "contains":
		// Check if fieldValue contains cond.Value
		if str, ok := fieldValue.(string); ok {
			return strings.Contains(str, cond.Value.(string))
		}
		return false
	case "gt":
		// Greater than - for numeric comparisons
		if fv, ok := fieldValue.(int); ok {
			if cv, ok := cond.Value.(int); ok {
				return fv > cv
			}
		}
		return false
	case "lt":
		// Less than - for numeric comparisons
		if fv, ok := fieldValue.(int); ok {
			if cv, ok := cond.Value.(int); ok {
				return fv < cv
			}
		}
		return false
	default:
		return false
	}
}

// DefaultPolicy returns a default permissive policy for development.
func DefaultPolicy() *Policy {
	return &Policy{
		ID:          uuid.New().String(),
		Name:        "default",
		Description: "Default development policy - allow all",
		Rules: []Rule{
			{
				ID:      uuid.New().String(),
				Subject: "*",
				Actions: []string{"*"},
				Effect:  "allow",
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Enabled:   true,
	}
}

// RestrictivePolicy returns a restrictive policy for production.
func RestrictivePolicy() *Policy {
	return &Policy{
		ID:          uuid.New().String(),
		Name:        "restrictive",
		Description: "Restrictive policy - deny by default",
		Rules: []Rule{
			{
				ID:      uuid.New().String(),
				Subject: "code:analyze",
				Actions: []string{"read", "execute"},
				Effect:  "allow",
			},
			{
				ID:      uuid.New().String(),
				Subject: "workflow:*",
				Actions: []string{"read"},
				Effect:  "allow",
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Enabled:   true,
	}
}
