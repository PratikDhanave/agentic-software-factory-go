package services

import (
	"context"
	"fmt"
)

// LLMProvider interface - model agnostic
type LLMProvider interface {
	// Generate a decision response
	GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error)

	// Analyze code
	AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error)

	// Review specification
	ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error)

	// Triage issue
	TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error)

	// Get provider name
	Name() string
}

// DecisionRequest - provider agnostic format
type DecisionRequest struct {
	IssueTitle       string
	IssueDescription string
	AgentRole        string
	PreviousDecisions []map[string]interface{}
}

// DecisionResponse - provider agnostic format
type DecisionResponse struct {
	Decision   string
	Reasoning  string
	Confidence float64
	Alternatives []string
}

// ProviderConfig holds configuration for any LLM provider
type ProviderConfig struct {
	Provider string            // "claude", "openai", "ollama", "custom"
	APIKey   string
	Model    string
	BaseURL  string            // For custom/local providers
	Params   map[string]interface{} // Provider-specific parameters
}

// NewLLMProvider creates provider based on config
func NewLLMProvider(config ProviderConfig) (LLMProvider, error) {
	switch config.Provider {
	case "claude":
		return NewClaudeProvider(config)
	case "openai":
		return NewOpenAIProvider(config)
	case "ollama":
		return NewOllamaProvider(config)
	case "mock":
		return &MockProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
}

// MockProvider for testing
type MockProvider struct{}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	return &DecisionResponse{
		Decision:   fmt.Sprintf("Mock decision for: %s", req.IssueTitle),
		Reasoning:  "This is a mock response",
		Confidence: 0.85,
		Alternatives: []string{"alternative1", "alternative2"},
	}, nil
}

func (m *MockProvider) AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"complexity": "medium",
		"issues":     []string{},
		"quality":    0.8,
	}, nil
}

func (m *MockProvider) ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"feasible":   true,
		"complexity": "medium",
		"risks":      []string{},
	}, nil
}

func (m *MockProvider) TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"category":        "enhancement",
		"priority":        "medium",
		"suggested_agent": "Developer",
		"effort_estimate": "2-3 days",
	}, nil
}
