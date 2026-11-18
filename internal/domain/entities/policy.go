package entities

import (
	"time"

	"github.com/google/uuid"
)

type PolicyEffect string

const (
	EffectAllow PolicyEffect = "allow"
	EffectDeny  PolicyEffect = "deny"
)

type Condition struct {
	Type     string        // time-based, rate-based, attribute-based
	Operator string        // equals, contains, in, greater_than, less_than
	Value    string
	Meta     map[string]interface{}
}

type Statement struct {
	ID         string
	Effect     PolicyEffect
	Resource   []Resource
	Action     []Action
	Principal  []PrincipalType
	Conditions []*Condition
}

type Policy struct {
	ID          string
	Name        string
	Description string
	Statements  []*Statement
	Priority    int
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   string
}

type PolicyDecision string

const (
	DecisionAllow            PolicyDecision = "allow"
	DecisionDeny             PolicyDecision = "deny"
	DecisionRequireApproval  PolicyDecision = "require_approval"
	DecisionRateLimited      PolicyDecision = "rate_limited"
)

type PolicyEvaluationResult struct {
	Decision         PolicyDecision
	Reason           string
	MatchedPolicies  []*Policy
	RequiredApprovers []string
	CacheExpiry      time.Duration
	Timestamp        time.Time
}

type RateLimitConfig struct {
	Window   time.Duration
	MaxCount int
	Metric   string
}

type RateLimitState struct {
	PrincipalID string
	Metric      string
	Count       int
	ResetAt     time.Time
}

func NewPolicy(name, description string) *Policy {
	return &Policy{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Statements:  []*Statement{},
		Priority:    0,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func NewStatement(effect PolicyEffect) *Statement {
	return &Statement{
		ID:         uuid.New().String(),
		Effect:     effect,
		Resource:   []Resource{},
		Action:     []Action{},
		Principal:  []PrincipalType{},
		Conditions: []*Condition{},
	}
}

func (s *Statement) AddResource(r Resource) *Statement {
	s.Resource = append(s.Resource, r)
	return s
}

func (s *Statement) AddAction(a Action) *Statement {
	s.Action = append(s.Action, a)
	return s
}

func (s *Statement) AddPrincipal(p PrincipalType) *Statement {
	s.Principal = append(s.Principal, p)
	return s
}

func (s *Statement) AddCondition(c *Condition) *Statement {
	s.Conditions = append(s.Conditions, c)
	return s
}

func (p *Policy) AddStatement(s *Statement) *Policy {
	p.Statements = append(p.Statements, s)
	p.UpdatedAt = time.Now()
	return p
}

func NewCondition(condType, operator, value string) *Condition {
	return &Condition{
		Type:     condType,
		Operator: operator,
		Value:    value,
		Meta:     make(map[string]interface{}),
	}
}

func (c *Condition) SetMeta(key string, value interface{}) *Condition {
	c.Meta[key] = value
	return c
}

type PolicySet struct {
	ID       string
	Name     string
	Policies []*Policy
}

func NewPolicySet(name string) *PolicySet {
	return &PolicySet{
		ID:       uuid.New().String(),
		Name:     name,
		Policies: []*Policy{},
	}
}

func (ps *PolicySet) AddPolicy(p *Policy) *PolicySet {
	ps.Policies = append(ps.Policies, p)
	return ps
}

func NewPolicyEvaluationResult(decision PolicyDecision, reason string) *PolicyEvaluationResult {
	return &PolicyEvaluationResult{
		Decision:        decision,
		Reason:          reason,
		MatchedPolicies: []*Policy{},
		CacheExpiry:     5 * time.Minute,
		Timestamp:       time.Now(),
	}
}

func (per *PolicyEvaluationResult) AddMatchedPolicy(p *Policy) *PolicyEvaluationResult {
	per.MatchedPolicies = append(per.MatchedPolicies, p)
	return per
}

func (per *PolicyEvaluationResult) AddRequiredApprover(approver string) *PolicyEvaluationResult {
	per.RequiredApprovers = append(per.RequiredApprovers, approver)
	return per
}

func NewRateLimitState(principalID string, metric string, resetAt time.Time) *RateLimitState {
	return &RateLimitState{
		PrincipalID: principalID,
		Metric:      metric,
		Count:       0,
		ResetAt:     resetAt,
	}
}

func (rls *RateLimitState) IsExpired() bool {
	return time.Now().After(rls.ResetAt)
}

func (rls *RateLimitState) Increment() int {
	rls.Count++
	return rls.Count
}

func (rls *RateLimitState) Reset() {
	rls.Count = 0
	rls.ResetAt = time.Now().Add(1 * time.Minute)
}
