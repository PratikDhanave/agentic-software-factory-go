package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type LLMRequest struct {
	Model            string
	Prompt           string
	SystemPrompt     string
	Temperature      float64
	MaxTokens        int
	ResponseSchema   interface{}
	JsonMode         bool
}

type LLMResponse struct {
	Content string
	Tokens  int
	Error   error
}

type LLMClient interface {
	Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

type AgentRequest struct {
	AgentRole    entities.AgentRole
	Event        map[string]interface{}
	PriorContext map[string]interface{}
	Query        string
}

type AgentResponse struct {
	Decision *entities.Decision
	Error    error
	RawText  string
}

type AgentInterface interface {
	Decide(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
	GetCapabilities() *entities.AgentCapabilities
	GetRole() entities.AgentRole
}

type BaseAgent struct {
	role         entities.AgentRole
	capabilities *entities.AgentCapabilities
	llmClient    LLMClient
	logger       Logger
}

func NewBaseAgent(
	role entities.AgentRole,
	caps *entities.AgentCapabilities,
	llmClient LLMClient,
	logger Logger,
) *BaseAgent {
	return &BaseAgent{
		role:         role,
		capabilities: caps,
		llmClient:    llmClient,
		logger:       logger,
	}
}

func (a *BaseAgent) GetRole() entities.AgentRole {
	return a.role
}

func (a *BaseAgent) GetCapabilities() *entities.AgentCapabilities {
	return a.capabilities
}

func (a *BaseAgent) Decide(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	systemPrompt := a.buildSystemPrompt()
	userPrompt := a.buildUserPrompt(req)

	llmReq := &LLMRequest{
		Model:        string(a.capabilities.LLMModel),
		SystemPrompt: systemPrompt,
		Prompt:       userPrompt,
		Temperature:  0.7,
		MaxTokens:    2000,
		JsonMode:     true,
	}

	resp, err := a.llmClient.Call(ctx, llmReq)
	if err != nil {
		a.logger.Error("LLM call failed", err, "agent_role", a.role)
		return nil, err
	}

	a.logger.Debug("LLM response received", "agent_role", a.role, "tokens", resp.Tokens)

	decision := a.parseDecision(resp.Content)

	return &AgentResponse{
		Decision: decision,
		RawText:  resp.Content,
	}, nil
}

func (a *BaseAgent) buildSystemPrompt() string {
	return `You are a ` + a.capabilities.Title + `.

Role: ` + string(a.role) + `
Description: ` + a.capabilities.Description + `
Domains: ` + joinStrings(a.capabilities.KnowledgeDomains) + `
Authority: ` + string(a.capabilities.DecisionAuthority) + `

Respond ONLY with valid JSON in this format:
{
  "choice": "the decision made",
  "reasoning": "why this choice",
  "confidence": 0.85,
  "alternatives": ["option1", "option2"],
  "impact": {"key": "value"}
}`
}

func (a *BaseAgent) buildUserPrompt(req *AgentRequest) string {
	eventJSON, _ := json.MarshalIndent(req.Event, "", "  ")
	contextJSON, _ := json.MarshalIndent(req.PriorContext, "", "  ")

	return `Event:
` + string(eventJSON) + `

Prior Context:
` + string(contextJSON) + `

Decision Required: ` + req.Query
}

func (a *BaseAgent) parseDecision(jsonStr string) *entities.Decision {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		a.logger.Error("Failed to parse decision JSON", err, "agent_role", a.role)
		return &entities.Decision{
			AgentRole:  a.role,
			Choice:     "default",
			Confidence: 0.0,
		}
	}

	confidence := 0.5
	if conf, ok := data["confidence"].(float64); ok {
		confidence = conf
	}

	alternatives := []string{}
	if alts, ok := data["alternatives"].([]interface{}); ok {
		for _, alt := range alts {
			if str, ok := alt.(string); ok {
				alternatives = append(alternatives, str)
			}
		}
	}

	impact := make(map[string]interface{})
	if imp, ok := data["impact"].(map[string]interface{}); ok {
		impact = imp
	}

	return &entities.Decision{
		AgentRole:      a.role,
		Choice:         toString(data["choice"]),
		Reasoning:      toString(data["reasoning"]),
		Confidence:     confidence,
		Alternatives:   alternatives,
		Impact:         impact,
		NeedsApproval:  confidence < 0.7,
		CreatedAt:      timeNow(),
	}
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func toString(v interface{}) string {
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

func timeNow() time.Time {
	return time.Now()
}
