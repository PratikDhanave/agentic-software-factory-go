package services

import (
	"context"
	"fmt"
	"sync"
)

// Skill represents a reusable capability that agents can execute
type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	SupportedAgents() []string
}

// BaseSkill provides common skill functionality
type BaseSkill struct {
	name             string
	description      string
	supportedAgents  []string
}

func (s *BaseSkill) Name() string {
	return s.name
}

func (s *BaseSkill) Description() string {
	return s.description
}

func (s *BaseSkill) SupportedAgents() []string {
	return s.supportedAgents
}

// CodeReviewSkill reviews code changes
type CodeReviewSkill struct {
	BaseSkill
	llmClient ClaudeClient
}

func NewCodeReviewSkill(llmClient ClaudeClient) *CodeReviewSkill {
	return &CodeReviewSkill{
		BaseSkill: BaseSkill{
			name:        "code-review",
			description: "Automated code review and quality assessment",
			supportedAgents: []string{"Review", "QA"},
		},
		llmClient: llmClient,
	}
}

func (s *CodeReviewSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	code, ok := input["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code parameter required")
	}

	// Use LLM to review code
	analysis, err := s.llmClient.AnalyzeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"analysis":       analysis,
		"issues":         []string{},
		"recommendations": []string{},
	}, nil
}

// TriageIssueSkill triages GitHub issues
type TriageIssueSkill struct {
	BaseSkill
	llmClient ClaudeClient
}

func NewTriageIssueSkill(llmClient ClaudeClient) *TriageIssueSkill {
	return &TriageIssueSkill{
		BaseSkill: BaseSkill{
			name:        "triage-issue",
			description: "Automatically triage and route GitHub issues",
			supportedAgents: []string{"Triage", "Foreman"},
		},
		llmClient: llmClient,
	}
}

func (s *TriageIssueSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	issue, ok := input["issue"].(string)
	if !ok {
		return nil, fmt.Errorf("issue parameter required")
	}

	// Use LLM to triage
	triage, err := s.llmClient.TriageIssue(ctx, issue)
	if err != nil {
		return nil, err
	}

	return triage, nil
}

// SecurityReviewSkill performs security reviews
type SecurityReviewSkill struct {
	BaseSkill
	llmClient ClaudeClient
}

func NewSecurityReviewSkill(llmClient ClaudeClient) *SecurityReviewSkill {
	return &SecurityReviewSkill{
		BaseSkill: BaseSkill{
			name:        "security-review",
			description: "Security-focused code and specification review",
			supportedAgents: []string{"Security", "Review"},
		},
		llmClient: llmClient,
	}
}

func (s *SecurityReviewSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	code, ok := input["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code parameter required")
	}

	// Security-focused analysis
	analysis, err := s.llmClient.AnalyzeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"vulnerabilities": []string{},
		"severity":        "low",
		"analysis":        analysis,
		"recommendations": []string{},
	}, nil
}

// ReviewSpecSkill reviews specifications
type ReviewSpecSkill struct {
	BaseSkill
	llmClient ClaudeClient
}

func NewReviewSpecSkill(llmClient ClaudeClient) *ReviewSpecSkill {
	return &ReviewSpecSkill{
		BaseSkill: BaseSkill{
			name:        "review-spec",
			description: "Review and validate specifications",
			supportedAgents: []string{"Spec", "Architect"},
		},
		llmClient: llmClient,
	}
}

func (s *ReviewSpecSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	spec, ok := input["spec"].(string)
	if !ok {
		return nil, fmt.Errorf("spec parameter required")
	}

	review, err := s.llmClient.ReviewSpecification(ctx, spec)
	if err != nil {
		return nil, err
	}

	return review, nil
}

// SkillRegistry manages available skills
type SkillRegistry struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

// NewSkillRegistry creates a new skill registry
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]Skill),
	}
}

// RegisterSkill registers a new skill
func (r *SkillRegistry) RegisterSkill(skill Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skill.Name()]; exists {
		return fmt.Errorf("skill %s already registered", skill.Name())
	}

	r.skills[skill.Name()] = skill
	return nil
}

// GetSkill retrieves a skill by name
func (r *SkillRegistry) GetSkill(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill %s not found", name)
	}

	return skill, nil
}

// GetSkillsForAgent gets all skills available to an agent
func (r *SkillRegistry) GetSkillsForAgent(agentRole string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Skill
	for _, skill := range r.skills {
		for _, supportedAgent := range skill.SupportedAgents() {
			if supportedAgent == agentRole {
				result = append(result, skill)
				break
			}
		}
	}

	return result
}

// ListSkills returns all registered skills
func (r *SkillRegistry) ListSkills() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []map[string]interface{}
	for _, skill := range r.skills {
		result = append(result, map[string]interface{}{
			"name":              skill.Name(),
			"description":       skill.Description(),
			"supported_agents": skill.SupportedAgents(),
		})
	}

	return result
}

// ExecuteSkill executes a skill with the given input
func (r *SkillRegistry) ExecuteSkill(ctx context.Context, skillName string, input map[string]interface{}) (map[string]interface{}, error) {
	skill, err := r.GetSkill(skillName)
	if err != nil {
		return nil, err
	}

	return skill.Execute(ctx, input)
}
