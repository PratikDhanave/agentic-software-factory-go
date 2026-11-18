package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type LLMRequest struct {
	Model          string
	Prompt         string
	SystemPrompt   string
	Temperature    float64
	MaxTokens      int
	ResponseSchema interface{}
	JsonMode       bool
}

type LLMResponse struct {
	Content  string
	Tokens   int
	Provider string
	Model    string
	Error    error
}

type LLMClient interface {
	Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

// LLMProvider is the unified multi-provider gateway contract across Anthropic,
// OpenAI-compatible endpoints, local Ollama, and offline deterministic mocks.
type LLMProvider interface {
	LLMClient
	Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	ProviderName() string
	HealthCheck(ctx context.Context) error
}

// ExtractJSONBlock strips optional Markdown code fences and isolates the outer
// JSON object or array from an LLM completion string.
func ExtractJSONBlock(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.Contains(s, "```") {
		parts := strings.Split(s, "```")
		for i := 1; i < len(parts); i += 2 {
			block := strings.TrimSpace(parts[i])
			if strings.HasPrefix(strings.ToLower(block), "json") {
				block = strings.TrimSpace(block[4:])
			}
			if strings.HasPrefix(block, "{") || strings.HasPrefix(block, "[") {
				return block
			}
		}
	}

	firstObj := strings.IndexByte(s, '{')
	lastObj := strings.LastIndexByte(s, '}')
	firstArr := strings.IndexByte(s, '[')
	lastArr := strings.LastIndexByte(s, ']')

	if firstObj >= 0 && lastObj > firstObj && (firstArr < 0 || firstObj < firstArr) {
		return s[firstObj : lastObj+1]
	}
	if firstArr >= 0 && lastArr > firstArr {
		return s[firstArr : lastArr+1]
	}
	return s
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
	cleanJSON := ExtractJSONBlock(jsonStr)
	if err := json.Unmarshal([]byte(cleanJSON), &data); err != nil {
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
