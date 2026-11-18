package services

import (
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type AgentRegistry struct {
	capabilities map[entities.AgentRole]*entities.AgentCapabilities
}

func NewAgentRegistry() *AgentRegistry {
	registry := &AgentRegistry{
		capabilities: make(map[entities.AgentRole]*entities.AgentCapabilities),
	}
	registry.registerDefaults()
	return registry
}

func (r *AgentRegistry) registerDefaults() {
	r.Register(&entities.AgentCapabilities{
		Role:        entities.ProductAgent,
		Title:       "Product Manager Agent",
		Description: "Understands requirements, prioritizes work, and tracks metrics",
		Capabilities: []string{
			"parse-requirements",
			"prioritize-work",
			"identify-stakeholders",
			"estimate-timelines",
			"track-metrics",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"product", "roadmap", "metrics"},
		DecisionAuthority: entities.Suggestive,
		InteractionMode:   entities.Interactive,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.ArchitectAgent,
		Title:       "System Architect Agent",
		Description: "Makes system design decisions and architecture choices",
		Capabilities: []string{
			"design-architecture",
			"evaluate-tradeoffs",
			"assess-scalability",
			"review-tech-choices",
			"plan-migrations",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"architecture", "systems", "scalability"},
		DecisionAuthority: entities.Authoritative,
		InteractionMode:   entities.Advisory,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.DesignAgent,
		Title:       "Design Agent",
		Description: "Creates architecture and design patterns, reviews design proposals",
		Capabilities: []string{
			"design-patterns",
			"api-design",
			"database-schema",
			"ui-ux-decisions",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"design", "patterns", "api"},
		DecisionAuthority: entities.Suggestive,
		InteractionMode:   entities.Interactive,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.DeveloperAgent,
		Title:       "Developer Agent",
		Description: "Implements changes, writes code, fixes issues",
		Capabilities: []string{
			"implement-features",
			"write-code",
			"fix-bugs",
			"refactor-code",
			"estimate-effort",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"development", "coding", "implementation"},
		DecisionAuthority: entities.Suggestive,
		InteractionMode:   entities.Autonomous,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.SecurityAgent,
		Title:       "Security Agent",
		Description: "Makes security decisions, identifies vulnerabilities",
		Capabilities: []string{
			"threat-analysis",
			"vulnerability-scanning",
			"security-testing",
			"compliance-review",
			"access-control-decisions",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"security", "compliance", "threats"},
		DecisionAuthority: entities.Authoritative,
		InteractionMode:   entities.Advisory,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.QAAgent,
		Title:       "QA Agent",
		Description: "Tests coverage and verifies fixes",
		Capabilities: []string{
			"test-strategy",
			"test-coverage-analysis",
			"verify-fixes",
			"performance-testing",
			"regression-detection",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"qa", "testing", "quality"},
		DecisionAuthority: entities.Suggestive,
		InteractionMode:   entities.Autonomous,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.PerformanceAgent,
		Title:       "Performance Agent",
		Description: "Optimizes performance and identifies bottlenecks",
		Capabilities: []string{
			"profiling",
			"bottleneck-identification",
			"optimization-recommendations",
			"load-testing",
			"capacity-planning",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"performance", "optimization", "metrics"},
		DecisionAuthority: entities.Suggestive,
		InteractionMode:   entities.Advisory,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.DevOpsAgent,
		Title:       "DevOps Agent",
		Description: "Orchestrates deployments and infrastructure decisions",
		Capabilities: []string{
			"deployment-orchestration",
			"infrastructure-decisions",
			"monitoring-setup",
			"rollback-planning",
			"disaster-recovery",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"devops", "infrastructure", "deployment"},
		DecisionAuthority: entities.Authoritative,
		InteractionMode:   entities.Autonomous,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.ComplianceAgent,
		Title:       "Compliance Agent",
		Description: "Ensures compliance and audit trail integrity",
		Capabilities: []string{
			"compliance-review",
			"audit-trail-validation",
			"regulation-enforcement",
			"access-logging",
			"policy-verification",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"compliance", "audit", "regulations"},
		DecisionAuthority: entities.Authoritative,
		InteractionMode:   entities.Advisory,
	})

	r.Register(&entities.AgentCapabilities{
		Role:        entities.CoordinatorAgent,
		Title:       "Coordinator Agent",
		Description: "Orchestrates workflow generation and agent coordination",
		Capabilities: []string{
			"workflow-generation",
			"agent-orchestration",
			"decision-synthesis",
			"conflict-resolution",
			"priority-setting",
		},
		LLMModel:          entities.Claude35Sonnet,
		ContextWindow:     100000,
		ResponseFormat:    "json",
		KnowledgeDomains:  []string{"coordination", "orchestration", "workflows"},
		DecisionAuthority: entities.Authoritative,
		InteractionMode:   entities.Interactive,
	})
}

func (r *AgentRegistry) Register(caps *entities.AgentCapabilities) error {
	if caps.Role == "" {
		return fmt.Errorf("agent role cannot be empty")
	}
	r.capabilities[caps.Role] = caps
	return nil
}

func (r *AgentRegistry) Get(role entities.AgentRole) *entities.AgentCapabilities {
	return r.capabilities[role]
}

func (r *AgentRegistry) GetAll() map[entities.AgentRole]*entities.AgentCapabilities {
	return r.capabilities
}

func (r *AgentRegistry) Has(role entities.AgentRole) bool {
	_, exists := r.capabilities[role]
	return exists
}
