# Self-Improving Agent Architecture

## Overview

Self-improving agents learn from past decisions and outcomes to refine their decision-making over time. The Factory Stack implements this through a **continuous feedback loop** that captures metrics, discovers patterns, and applies learning.

## Architecture Layers

```
┌─────────────────────────────────────────┐
│   Human Review & Override Layer         │
│   (Approvals, Feedback, Corrections)    │
└─────────────┬───────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│   Metric Collection Layer               │
│   - Decision accuracy                   │
│   - Approval rate                       │
│   - Issue type accuracy                 │
│   - Confidence calibration              │
└─────────────┬───────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│   Pattern Learning Layer                │
│   - Issue classification patterns       │
│   - Decision patterns by issue type     │
│   - Agent specialization discovery      │
│   - Confidence calibration curves       │
└─────────────┬───────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│   Decision Improvement Layer            │
│   - Apply learned patterns              │
│   - Adjust confidence scores            │
│   - Route to specialized agents         │
│   - Early approval/rejection            │
└─────────────┬───────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│   Decision Generation Layer             │
│   - LLM-powered decisions               │
│   - Multi-agent coordination            │
│   - Skill selection                     │
└─────────────────────────────────────────┘
```

## 1. Metric Collection

### What to Measure

For each **agent decision**, collect:

```go
type AgentMetric struct {
    AgentRole           string    // "Architect", "Developer", etc.
    ExecutionID         uuid.UUID // Links to workflow execution
    
    // Core metrics
    DecisionCorrect     bool      // Was the decision right? (from approval or outcome)
    Confidence          float64   // Agent's confidence (0-1)
    ApprovalRequired    bool      // Did it need human review?
    ApprovedByHuman     bool      // Did human approve it?
    
    // Context
    IssueType           string    // "auth", "performance", "security", "data"
    IssueComplexity     string    // "simple", "medium", "complex"
    TimeToDecision      int64     // milliseconds
    
    // Learned relationship
    PatternMatches      []string  // Which learned patterns applied?
    
    CreatedAt           time.Time
}
```

### Where to Collect

**During Workflow Generation:**
- Store decision before sending to user
- Track agent confidence score
- Record issue classification

**During Approval:**
- Record human approval/rejection
- Store approver feedback
- Update "DecisionCorrect" if rejection signals error

**After Execution:**
- Record if the fix actually worked (from GitHub webhook)
- Update success metrics
- Trigger pattern learning

## 2. Pattern Discovery

### Three Types of Patterns

#### A. Issue Classification Patterns

```go
type IssueClassificationPattern struct {
    // Recognition
    KeywordSet    []string // ["auth", "oauth", "login", "session"]
    IssueType     string   // "authentication"
    
    // Reliability
    TimesMatched  int
    AccuracyRate  float64  // How often this classification was correct (0-1)
    
    // Application
    Priority      int      // Higher = apply first
    ConfidenceBoost float64 // Add to agent confidence if matched
}
```

**Discovery Process:**
1. Collect 50+ decisions for each agent/issue-type combination
2. For each issue_type group, analyze:
   - What keywords appear in titles/descriptions?
   - Which agent decisions were approved?
   - Which were rejected?
3. Extract discriminative keyword sets
4. Calculate accuracy = approved_decisions / total_decisions
5. Store with priority weighting

#### B. Decision Patterns

```go
type DecisionPattern struct {
    // Context where this decision works
    IssueType       string   // "auth"
    Agent           string   // "Architect"
    ComplexityRange [2]string // ["simple", "medium"]
    KeywordTriggers []string // ["oauth", "session"]
    
    // The decision template
    DecisionTemplate string // "Use OAuth2 with..."
    AlternativeCount int    // How many alternatives to suggest
    
    // Success rate
    Successful      int     // Approved/correct outcomes
    Total           int     // Total times tried
    SuccessRate     float64 // 0-1
    
    // Confidence adjustment
    ConfidenceBase  float64 // Base confidence (0.7-0.95)
    
    CreatedAt       time.Time
    LastUpdated     time.Time
}
```

**Discovery Process:**
1. Group approved decisions by: agent + issue_type + complexity
2. For each group with 10+ decisions:
   - Find common decision phrases (NLP clustering)
   - Calculate success_rate = approved / total
   - Set confidence_base = success_rate * 0.95 (confidence cap)
3. Store high-confidence patterns (success_rate > 0.8)

#### C. Agent Specialization Patterns

```go
type AgentSpecializationPattern struct {
    Agent           string
    SpecialtyType   string   // "authentication", "performance", "security"
    SuccessRate     float64  // How good is this agent at this type?
    AverageConfidence float64
    DecisionCount   int
    Priority        float64  // 0-1, computed as success_rate * decision_count / max_count
}
```

**Discovery Process:**
1. For each agent, calculate success rates per issue_type
2. Rank issue_types by success_rate
3. Mark top 2-3 as specialties
4. Use to route future decisions

## 3. Continuous Learning Loop

### Every 100 Decisions:

```go
func (s *LearningSupervisor) LearnFromRecentDecisions(ctx context.Context, agentRole string) error {
    // 1. Fetch recent 100 decisions for this agent
    decisions, err := s.repo.GetRecentDecisions(ctx, agentRole, 100)
    
    // 2. For each issue_type, analyze patterns
    for issueType, decisions := range groupByIssueType(decisions) {
        // 3. Discover patterns
        patterns := s.discoverPatterns(decisions, issueType)
        
        // 4. Update learned_patterns table
        for _, pattern := range patterns {
            if pattern.SuccessRate > 0.8 {
                s.repo.SavePattern(ctx, pattern)
            }
        }
    }
    
    // 5. Update agent metrics (success_rate, accuracy trend)
    metrics := s.computeAgentMetrics(agentRole, decisions)
    s.repo.UpdateAgentMetrics(ctx, agentRole, metrics)
    
    return nil
}
```

### Every 1000 Decisions:

```go
func (s *LearningSupervisor) UpdateAgentRouting(ctx context.Context) error {
    // 1. For each agent, discover specialties
    for _, agent := range s.agents {
        specialties := s.analyzeSpecializations(ctx, agent)
        
        // 2. Update routing decisions
        s.updateAgentRouter(agent, specialties)
    }
    return nil
}
```

## 4. Using Learned Patterns in Decision Generation

### During Workflow Generation:

```go
func (s *Service) GenerateDecision(ctx context.Context, issue *Issue, agent string) (*Decision, error) {
    // 1. Classify the issue
    issueType, keywords := s.classifyIssue(issue.Title, issue.Body)
    
    // 2. Find matching learned patterns
    patterns, err := s.repo.FindPatterns(ctx, agent, issueType, keywords)
    
    // 3. Apply pattern to boost confidence and guide decision
    decision := s.generateDecision(agent, issue)
    
    for _, pattern := range patterns {
        if pattern.SuccessRate > 0.9 {
            // Pattern is highly reliable
            decision.Confidence = min(0.98, pattern.ConfidenceBase)
            decision.Decision = pattern.DecisionTemplate
            decision.ApprovalRequired = false // Skip approval for high-confidence patterns
            break
        } else if pattern.SuccessRate > 0.8 {
            // Pattern is good but not great
            decision.Confidence = pattern.ConfidenceBase
            decision.ApprovalRequired = decision.Confidence < 0.9
            break
        }
    }
    
    return decision, nil
}
```

## 5. Feedback Integration

### From Human Approvals:

```go
func (s *Service) ProcessApproval(ctx context.Context, approval *Approval) error {
    // 1. Update decision as correct/incorrect
    if approval.Status == "approved" {
        s.repo.MarkDecisionCorrect(ctx, approval.DecisionID, true)
    } else if approval.Status == "rejected" {
        s.repo.MarkDecisionCorrect(ctx, approval.DecisionID, false)
        s.captureRejectionFeedback(ctx, approval)
    }
    
    // 2. If rejection, store feedback
    if approval.ApprovalComments != "" {
        s.repo.SaveFeedback(ctx, &Feedback{
            DecisionID: approval.DecisionID,
            Content:   approval.ApprovalComments,
            Type:      "human_correction",
        })
    }
    
    // 3. Trigger learning if threshold reached
    recentCount, err := s.repo.CountRecentDecisions(ctx, approval.AgentRole)
    if recentCount >= 100 {
        go s.supervisor.LearnFromRecentDecisions(ctx, approval.AgentRole)
    }
    
    return nil
}
```

### From GitHub Outcomes:

```go
func (s *Service) ProcessGitHubOutcome(ctx context.Context, prURL string, merged bool, buildPassed bool) error {
    // 1. Find the workflow execution that generated the fix
    exec, err := s.repo.FindExecutionByPRURL(ctx, prURL)
    
    // 2. Mark decision as successful if:
    //    - PR merged (code approved by humans)
    //    - Build passed (automated tests passed)
    //    - No new issues filed against it (from monitoring)
    successful := merged && buildPassed
    
    s.repo.MarkExecutionSuccessful(ctx, exec.ID, successful)
    
    // 3. Update agent metrics
    for _, decision := range exec.Decisions {
        s.repo.RecordMetric(ctx, &AgentMetric{
            AgentRole:        decision.AgentRole,
            ExecutionID:      exec.ID,
            DecisionCorrect:  successful,
            ApprovalRequired: decision.RequiresApproval,
            ApprovedByHuman:  decision.Approved,
        })
    }
    
    return nil
}
```

## 6. Confidence Calibration

### Problem: Agents Miscalibrate Confidence

If an agent says "95% confident" but is wrong 20% of the time, the approval system approves bad decisions.

### Solution: Track Calibration Curve

```go
type ConfidenceCalibration struct {
    Agent         string
    ConfidenceBin string // "0.9-0.95", "0.95-0.99", etc.
    
    TotalDecisions int
    CorrectCount   int
    ActualRate     float64 // Empirical correctness rate
    
    PredictedRate  float64 // What agent claimed (mean of bin)
    Calibration    float64 // ActualRate / PredictedRate
}
```

### Learning:

```go
func (s *LearningSupervisor) CalibrateConfidence(ctx context.Context, agent string) error {
    // 1. Group decisions by confidence range
    decisions, _ := s.repo.GetRecentDecisions(ctx, agent, 1000)
    
    for _, bin := range confidenceBins { // ["0.8-0.85", "0.85-0.9", ...]
        inBin := filterByConfidenceRange(decisions, bin)
        if len(inBin) < 20 { continue } // Not enough samples
        
        correct := countCorrect(inBin)
        actualRate := float64(correct) / float64(len(inBin))
        predictedRate := (bin.Min + bin.Max) / 2
        
        calibration := actualRate / predictedRate
        s.repo.SaveCalibration(ctx, &ConfidenceCalibration{
            Agent:        agent,
            ConfidenceBin: bin.Name,
            Calibration: calibration,
        })
    }
    
    return nil
}
```

### Applying Calibration:

```go
func (s *Service) ApplyCalibration(ctx context.Context, decision *Decision) {
    cal, _ := s.repo.GetCalibration(ctx, decision.AgentRole, decision.Confidence)
    if cal != nil && cal.Calibration > 1.1 {
        // Agent is overconfident; reduce confidence
        decision.Confidence *= (1.0 / cal.Calibration)
    } else if cal != nil && cal.Calibration < 0.9 {
        // Agent is underconfident; increase confidence slightly
        decision.Confidence *= (1.0 / cal.Calibration * 0.9) // Conservative
    }
}
```

## 7. Integration with GitHub Events

### Automatic Feedback from GitHub:

```go
func (h *GitHubWebhookHandler) handlePRMerged(ctx context.Context, event *GitHubWebhookEvent) {
    // PR was merged - this means:
    // 1. The fix was approved by code reviewers
    // 2. CI/CD passed
    // 3. The decision was good!
    
    workflow, _ := h.repo.FindWorkflowByPRNumber(ctx, event.PullRequest.Number)
    for _, decision := range workflow.Decisions {
        h.learner.RecordSuccess(ctx, decision.ID)
    }
}

func (h *GitHubWebhookHandler) handlePRClosed(ctx context.Context, event *GitHubWebhookEvent) {
    // PR was closed without merging - decision wasn't good
    workflow, _ := h.repo.FindWorkflowByPRNumber(ctx, event.PullRequest.Number)
    for _, decision := range workflow.Decisions {
        h.learner.RecordFailure(ctx, decision.ID)
    }
}

func (h *GitHubWebhookHandler) handleIssueReopened(ctx context.Context, event *GitHubWebhookEvent) {
    // Issue was reopened after being marked done - decision was wrong
    // Find related decisions and mark as failures
    decisions, _ := h.repo.FindDecisionsForIssue(ctx, event.Issue.Number)
    for _, decision := range decisions {
        h.learner.RecordFailure(ctx, decision.ID)
    }
}
```

## 8. Monitoring Self-Improvement

### Dashboard Metrics:

```go
type SelfImprovementMetrics struct {
    // Agent Performance Over Time
    AgentSuccessRates    map[string]float64 // "Architect": 0.87
    AgentSpecializations map[string][]string // "Developer": ["auth", "performance"]
    
    // Learning Progress
    PatternsDiscovered   int
    HighConfidencePatterns int
    PatternAccuracy      float64
    
    // Feedback Loop Health
    ApprovalRate         float64 // % of decisions requiring approval
    HumanAcceptanceRate  float64 // % of approvals accepted
    
    // Trend
    SuccessRateTrend     []float64 // Last 30 days
    ConfidenceTrend      []float64
    PatternUtilization   int // % of decisions using learned patterns
}
```

## 9. Implementation Roadmap

### Week 1: Metric Collection ✅
- Save metrics to agent_metrics table
- Track during approval flow
- Record human feedback

### Week 2: Pattern Discovery
- Implement pattern detection algorithms
- Store patterns in learned_patterns table
- Validate accuracy

### Week 3: Learning Integration
- Apply patterns to boost confidence
- Skip approvals for high-confidence patterns
- Implement confidence calibration

### Week 4: Feedback Loop
- Wire GitHub outcomes to metrics
- Auto-mark successes/failures
- Trigger learning cycles

### Week 5: Monitoring
- Dashboard for self-improvement metrics
- Trends over time
- Alert on degradation

## Example: Architect Agent Learning Authentication Issues

### Day 1-50: Initial Decisions
- Receives auth issues mixed with other types
- Struggles to route correctly
- Accuracy: 65%, confidence: 0.85

### Day 51-100: Pattern Emerges
- Discovers keywords: ["oauth", "login", "session", "jwt"]
- 90% of issues with these keywords → auth type
- Learns: When these keywords present, confidence should be 0.95

### Day 101-150: Applied Learning
- Recognizes auth issues faster
- Routes correctly 92% of time
- Uses learned patterns for 40% of decisions
- Accuracy: 92%, confidence calibrated to 0.88

### Day 151+: Specialization
- Becomes the auth expert
- Other agents defer to Architect for auth issues
- Accuracy: 95%, confidence: 0.92

---

## Next Steps

1. **Wire GitHub webhook handler** into main server
2. **Implement metric collection** in approval flow
3. **Build pattern discovery** scheduler
4. **Connect feedback loop** to database
5. **Create self-improvement dashboard** in web UI
