package application

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// SelfImprovingService manages agent learning and continuous improvement
type SelfImprovingService struct {
	repo                domain.WorkflowRepository
	llmClient           services.ClaudeClient
	decisionWindow      int    // Number of decisions before learning triggers
	learningThreshold   float64 // Minimum accuracy to form a pattern (0.8 = 80%)
	confidenceThreshold float64 // Confidence to skip approval (0.9 = 90%)
	mu                  sync.RWMutex
}

// NewSelfImprovingService creates a new self-improving service
func NewSelfImprovingService(repo domain.WorkflowRepository, llmClient services.ClaudeClient) *SelfImprovingService {
	return &SelfImprovingService{
		repo:                repo,
		llmClient:           llmClient,
		decisionWindow:      100,
		learningThreshold:   0.80,
		confidenceThreshold: 0.90,
	}
}

// RecordDecision captures a decision for learning
func (s *SelfImprovingService) RecordDecision(ctx context.Context, decision *domain.AgentDecision, issueType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metric := &domain.AgentMetric{
		AgentRole:        decision.AgentRole,
		ExecutionID:      decision.ExecutionID,
		DecisionCorrect:  false, // Will be updated when approved/rejected
		Confidence:       decision.Confidence,
		ApprovalRequired: decision.RequiresApproval,
		IssueType:        issueType,
		CreatedAt:        time.Now(),
	}

	return s.repo.SaveMetric(ctx, metric)
}

// RecordApproval records human feedback on a decision
func (s *SelfImprovingService) RecordApproval(ctx context.Context, approval *domain.Approval) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update decision correctness based on approval
	decision, _ := s.repo.GetDecision(ctx, approval.DecisionID)
	if decision != nil {
		decision.ApprovalStatus = approval.Status
		if approval.Status == "approved" {
			// Mark decision as correct
			s.repo.MarkDecisionCorrect(ctx, approval.DecisionID, true)
		} else if approval.Status == "rejected" {
			// Mark decision as incorrect and capture feedback
			s.repo.MarkDecisionCorrect(ctx, approval.DecisionID, false)
		}
	}

	// Check if learning should trigger
	recentCount, _ := s.repo.CountRecentDecisions(ctx, decision.AgentRole, s.decisionWindow)
	if recentCount >= s.decisionWindow {
		go s.LearnPatterns(context.Background(), decision.AgentRole)
	}

	return nil
}

// RecordOutcome records the actual outcome (from GitHub/execution)
func (s *SelfImprovingService) RecordOutcome(ctx context.Context, executionID string, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	execution, _ := s.repo.GetExecution(ctx, executionID)
	if execution == nil {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// Update all decisions in this execution
	for _, decision := range execution.Decisions {
		s.repo.MarkDecisionCorrect(ctx, decision.ID, success)
	}

	return nil
}

// LearnPatterns discovers and stores patterns from recent decisions
func (s *SelfImprovingService) LearnPatterns(ctx context.Context, agentRole string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Fetch recent decisions
	decisions, err := s.repo.GetRecentDecisions(ctx, agentRole, s.decisionWindow)
	if err != nil || len(decisions) < 20 {
		return fmt.Errorf("insufficient data for learning: %w", err)
	}

	// 2. Group by issue type
	grouped := s.groupDecisionsByIssueType(decisions)

	// 3. For each issue type, discover patterns
	for issueType, typeDecisions := range grouped {
		accuracy := s.calculateAccuracy(typeDecisions)
		if accuracy < s.learningThreshold {
			continue // Not reliable enough to learn
		}

		// Find common decision patterns
		patterns := s.extractPatterns(typeDecisions, issueType)
		for _, pattern := range patterns {
			pattern.AgentRole = agentRole
			s.repo.SavePattern(ctx, pattern)
		}

		// Update agent specialization
		s.updateAgentSpecialty(ctx, agentRole, issueType, accuracy, len(typeDecisions))
	}

	return nil
}

// UpdateConfidenceWithLearning applies learned patterns to adjust confidence
func (s *SelfImprovingService) UpdateConfidenceWithLearning(ctx context.Context, decision *domain.AgentDecision, issueType string, keywords []string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Find matching patterns
	patterns, _ := s.repo.FindPatterns(ctx, decision.AgentRole, issueType)

	// 2. Apply most reliable pattern
	for _, pattern := range patterns {
		if pattern.SuccessRate >= 0.90 {
			// High-confidence pattern
			decision.Confidence = 0.98
			decision.RequiresApproval = false
			break
		} else if pattern.SuccessRate >= s.learningThreshold {
			// Moderate-confidence pattern
			decision.Confidence = pattern.ConfidenceBase
			decision.RequiresApproval = decision.Confidence < s.confidenceThreshold
			break
		}
	}

	// 3. Apply confidence calibration
	cal, _ := s.repo.GetCalibration(ctx, decision.AgentRole, decision.Confidence)
	if cal != nil && cal.Calibration > 1.1 {
		// Agent is overconfident
		decision.Confidence *= (1.0 / cal.Calibration)
	} else if cal != nil && cal.Calibration < 0.9 {
		// Agent is underconfident (conservative correction)
		decision.Confidence *= (1.0 / cal.Calibration * 0.9)
	}

	return nil
}

// CalibrateConfidence learns the accuracy of confidence scores
func (s *SelfImprovingService) CalibrateConfidence(ctx context.Context, agentRole string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get recent decisions with outcomes
	decisions, _ := s.repo.GetRecentDecisions(ctx, agentRole, 500)
	if len(decisions) < 100 {
		return fmt.Errorf("insufficient data for calibration")
	}

	// Group by confidence bins
	bins := map[string][]*domain.AgentDecision{}
	for _, d := range decisions {
		bin := s.confidenceBin(d.Confidence)
		bins[bin] = append(bins[bin], d)
	}

	// For each bin, calculate calibration
	for binName, binDecisions := range bins {
		if len(binDecisions) < 20 {
			continue
		}

		correct := 0
		for _, d := range binDecisions {
			if d.ApprovalStatus == "approved" || d.ApprovalStatus == "correct" {
				correct++
			}
		}

		actualRate := float64(correct) / float64(len(binDecisions))
		predictedRate := s.binMidpoint(binName)
		calibration := actualRate / predictedRate

		s.repo.SaveCalibration(ctx, &domain.ConfidenceCalibration{
			AgentRole:     agentRole,
			ConfidenceBin: binName,
			Calibration:   calibration,
			SampleCount:   len(binDecisions),
			ActualRate:    actualRate,
		})
	}

	return nil
}

// UpdateAgentSpecialization determines which agent is best at which task type
func (s *SelfImprovingService) UpdateAgentSpecialization(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agents := []string{"Architect", "Developer", "QA", "Security", "Performance"}
	issueTypes := []string{"auth", "performance", "security", "data"}

	for _, agent := range agents {
		for _, issueType := range issueTypes {
			decisions, _ := s.repo.GetDecisionsByAgentAndType(ctx, agent, issueType, 100)
			if len(decisions) < 10 {
				continue
			}

			accuracy := s.calculateAccuracy(decisions)
			avgConfidence := s.calculateAvgConfidence(decisions)

			s.repo.SaveSpecialization(ctx, &domain.AgentSpecialization{
				Agent:             agent,
				SpecialtyType:     issueType,
				SuccessRate:       accuracy,
				AverageConfidence: avgConfidence,
				DecisionCount:     len(decisions),
			})
		}
	}

	return nil
}

// Helper methods

func (s *SelfImprovingService) groupDecisionsByIssueType(decisions []*domain.AgentDecision) map[string][]*domain.AgentDecision {
	grouped := make(map[string][]*domain.AgentDecision)
	for _, d := range decisions {
		if d.Reasoning != "" {
			// Extract issue type from decision reasoning
			issueType := s.extractIssueType(d.Reasoning)
			grouped[issueType] = append(grouped[issueType], d)
		}
	}
	return grouped
}

func (s *SelfImprovingService) calculateAccuracy(decisions []*domain.AgentDecision) float64 {
	if len(decisions) == 0 {
		return 0
	}

	correct := 0
	for _, d := range decisions {
		if d.ApprovalStatus == "approved" {
			correct++
		}
	}

	return float64(correct) / float64(len(decisions))
}

func (s *SelfImprovingService) calculateAvgConfidence(decisions []*domain.AgentDecision) float64 {
	if len(decisions) == 0 {
		return 0
	}

	sum := 0.0
	for _, d := range decisions {
		sum += d.Confidence
	}

	return sum / float64(len(decisions))
}

func (s *SelfImprovingService) extractPatterns(decisions []*domain.AgentDecision, issueType string) []*domain.LearnedPattern {
	var patterns []*domain.LearnedPattern

	// Group successful decisions
	successful := 0
	for _, d := range decisions {
		if d.ApprovalStatus == "approved" {
			successful++
		}
	}

	if len(decisions) > 0 {
		successRate := float64(successful) / float64(len(decisions))

		pattern := &domain.LearnedPattern{
			PatternType:     "issue_classification",
			IssueType:       issueType,
			SuccessRate:     successRate,
			TimesApplied:    len(decisions),
			ConfidenceBase:  successRate * 0.95, // Conservative
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if successRate >= s.learningThreshold {
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

func (s *SelfImprovingService) extractIssueType(reasoning string) string {
	// Extract issue type from reasoning
	// In production, use NLP/ML, for now use simple heuristics
	keywords := map[string][]string{
		"auth":        {"oauth", "login", "session", "jwt", "authentication"},
		"performance": {"slow", "latency", "speed", "timeout", "queue"},
		"security":    {"vulnerability", "exploit", "xss", "sql injection", "breach"},
		"data":        {"database", "migration", "schema", "corrupt", "loss"},
	}

	for issueType, kws := range keywords {
		for _, kw := range kws {
			if contains(reasoning, kw) {
				return issueType
			}
		}
	}

	return "general"
}

func (s *SelfImprovingService) confidenceBin(confidence float64) string {
	switch {
	case confidence < 0.70:
		return "0.60-0.70"
	case confidence < 0.75:
		return "0.70-0.75"
	case confidence < 0.80:
		return "0.75-0.80"
	case confidence < 0.85:
		return "0.80-0.85"
	case confidence < 0.90:
		return "0.85-0.90"
	case confidence < 0.95:
		return "0.90-0.95"
	default:
		return "0.95-1.00"
	}
}

func (s *SelfImprovingService) binMidpoint(bin string) float64 {
	midpoints := map[string]float64{
		"0.60-0.70": 0.65,
		"0.70-0.75": 0.725,
		"0.75-0.80": 0.775,
		"0.80-0.85": 0.825,
		"0.85-0.90": 0.875,
		"0.90-0.95": 0.925,
		"0.95-1.00": 0.975,
	}
	return midpoints[bin]
}

func (s *SelfImprovingService) updateAgentSpecialty(ctx context.Context, agentRole, issueType string, accuracy float64, decisionCount int) error {
	spec := &domain.AgentSpecialization{
		Agent:         agentRole,
		SpecialtyType: issueType,
		SuccessRate:   accuracy,
		DecisionCount: decisionCount,
	}

	return s.repo.SaveSpecialization(ctx, spec)
}

// SelfImprovementMetrics returns current metrics for the dashboard
func (s *SelfImprovingService) GetMetrics(ctx context.Context) (*SelfImprovementMetricsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := &SelfImprovementMetricsResponse{
		Agents:      make(map[string]*AgentMetrics),
		Patterns:    []PatternMetrics{},
		Timestamp:   time.Now(),
	}

	agents := []string{"Architect", "Developer", "QA", "Security", "Performance"}

	// Get metrics for each agent
	for _, agent := range agents {
		decisions, _ := s.repo.GetRecentDecisions(ctx, agent, 100)
		if len(decisions) == 0 {
			continue
		}

		accuracy := s.calculateAccuracy(decisions)
		avgConfidence := s.calculateAvgConfidence(decisions)

		metrics.Agents[agent] = &AgentMetrics{
			SuccessRate:   accuracy,
			Confidence:    avgConfidence,
			DecisionCount: len(decisions),
		}
	}

	// Get top patterns
	patterns, _ := s.repo.GetTopPatterns(ctx, 10)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].SuccessRate > patterns[j].SuccessRate
	})

	for _, p := range patterns {
		metrics.Patterns = append(metrics.Patterns, PatternMetrics{
			IssueType:   p.IssueType,
			Agent:       p.AgentRole,
			SuccessRate: p.SuccessRate,
			Applied:     p.TimesApplied,
		})
	}

	return metrics, nil
}

// Response types
type SelfImprovementMetricsResponse struct {
	Agents    map[string]*AgentMetrics
	Patterns  []PatternMetrics
	Timestamp time.Time
}

type AgentMetrics struct {
	SuccessRate   float64
	Confidence    float64
	DecisionCount int
}

type PatternMetrics struct {
	IssueType   string
	Agent       string
	SuccessRate float64
	Applied     int
}

// Utility
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0))
}
