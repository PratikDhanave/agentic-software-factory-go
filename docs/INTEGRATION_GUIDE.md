# Factory Stack Integration Guide

## End-to-End Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       GITHUB EVENTS                             │
│         (PR opened, Issue created, Review submitted)            │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   GITHUB WEBHOOK HANDLER                        │
│  - Verify webhook signature                                     │
│  - Parse event (PR, Issue, Review)                              │
│  - Route to appropriate handler                                 │
│  - Create trigger object                                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
    ┌─────────────┐  ┌──────────────┐  ┌──────────────┐
    │ Code Review │  │ Triage Issue │  │ Merge Check  │
    │   Skill     │  │   Skill      │  │   Skill      │
    └────┬────────┘  └───────┬──────┘  └──────┬───────┘
         │                   │                 │
         └───────────────────┼─────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DECISION GENERATION                           │
│  - Classify issue (auth, perf, security, data)                  │
│  - Select appropriate agents (Architect, Developer, QA, etc.)   │
│  - Apply learned patterns from self-improving service           │
│  - Generate decision with confidence score                      │
│  - Set approval_required based on confidence                    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   APPROVAL WORKFLOW                             │
│  - If confidence < 0.9: Requires human approval                 │
│  - Store decision pending approval                              │
│  - Display in UI for review                                     │
│  - Record human decision (approve/reject)                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SELF-IMPROVING FEEDBACK                       │
│  - Record if decision was approved                              │
│  - Track approval frequency per agent                           │
│  - Collect GitHub outcomes (merged, build passed)               │
│  - Calculate accuracy metrics                                   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   PATTERN LEARNING                              │
│  - Trigger when 100 decisions collected                         │
│  - Discover high-accuracy patterns (>80% success)               │
│  - Update agent specialization (which agent is best at what)    │
│  - Calibrate confidence scores                                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
         ┌───────────────────┴───────────────────┐
         │                                       │
         ▼                                       ▼
    ┌─────────────────┐              ┌──────────────────┐
    │ Higher Accuracy │              │ Lower Approval % │
    │ Decisions       │              │ Future Decisions │
    │ (Phase 2: Walk) │              │ (Crawl→Walk→Run) │
    └─────────────────┘              └──────────────────┘
```

## 1. Setting Up GitHub Webhook Integration

### Step 1: Create Webhook Handler

```go
// In cmd/asf-web/main.go
import (
    "github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/http/github_webhook_handler"
    "github.com/PratikDhanave/agentic-software-factory-go/cmd/asf-web"
)

func main() {
    // Initialize skill registry
    skillRegistry := services.NewSkillRegistry()
    skillRegistry.RegisterSkill(services.NewCodeReviewSkill(llmClient))
    skillRegistry.RegisterSkill(services.NewTriageIssueSkill(llmClient))
    skillRegistry.RegisterSkill(services.NewSecurityReviewSkill(llmClient))
    skillRegistry.RegisterSkill(services.NewReviewSpecSkill(llmClient))
    
    // Initialize integration manager
    integrationMgr := main.NewIntegrationManager(repo, llmClient, skillRegistry)
    
    // Register routes
    mux := http.NewServeMux()
    integrationMgr.RegisterRoutes(mux)
    
    // Start server
    http.ListenAndServe(":8000", mux)
}
```

### Step 2: Configure GitHub Webhook Secret

```bash
# In GitHub repository settings:
# Settings → Webhooks → Add Webhook
# - Payload URL: https://your-domain.com/webhooks/github
# - Content type: application/json
# - Secret: Generate a strong secret
# - Events: Pull requests, Issues, Pull request reviews, Issue comments
# - Active: Yes
```

### Step 3: Set Environment Variables

```bash
export GITHUB_WEBHOOK_SECRET="your-webhook-secret"
export CLAUDE_API_KEY="your-claude-api-key"
export DATABASE_URL="postgresql://user:pass@localhost/factory"
```

## 2. Workflow Generation Flow

### Trigger 1: PR Opened

```mermaid
GitHub Event: PR Opened
    ↓
Webhook Handler: Parse PR details (title, description, changed files)
    ↓
Classify Issue: Extract issue type from PR description
    ↓
Execute Code Review Skill:
    - Input: code, test coverage, complexity
    - Output: Review points, suggested changes, confidence
    ↓
Decision Generation:
    - Agent: "Review" (specialized for code review)
    - Decision: "Approve after addressing X issues"
    - Confidence: 0.87 (from pattern matching)
    ↓
Approval Required?: 0.87 < 0.90 → YES
    ↓
Store for Human Review:
    - Save to approvals table
    - Display in web UI
```

### Trigger 2: Issue Created

```mermaid
GitHub Event: Issue Created
    ↓
Webhook Handler: Parse issue details (title, description, labels)
    ↓
Classify Issue: Extract issue type (auth, performance, security, data)
    ↓
Execute Triage Skill:
    - Input: issue title + body
    - Output: Category, priority, suggested team
    ↓
Decision Generation:
    - Agent: "Triage" (Foreman)
    - Decision: "Route to Team X, Priority Y"
    - Confidence: 0.92 (from learned patterns)
    ↓
Approval Required?: 0.92 >= 0.90 → NO (Phase 2: Walk)
    ↓
Execute Automatically:
    - Assign to suggested team
    - Set priority label
    - Add comment with explanation
```

## 3. Self-Improving Feedback Loop

### Recording Approvals

```go
// When human approves/rejects a decision:
func (s *webServer) handleApprovalSubmit(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ApprovalID uuid.UUID
        Status     string // "approved" or "rejected"
        Comments   string
    }
    
    json.NewDecoder(r.Body).Decode(&req)
    
    // Record the approval
    s.selfImprovingService.RecordApproval(ctx, &domain.Approval{
        ID:                req.ApprovalID,
        Status:           req.Status,
        ApprovalComments: req.Comments,
    })
    
    // If 100 decisions collected, trigger learning
    if recentCount >= 100 {
        go s.selfImprovingService.LearnPatterns(ctx, agentRole)
    }
}
```

### Recording Outcomes from GitHub

```go
// When PR is merged or issue is resolved:
func (h *IntegrationManager) handlePullRequestEvent(ctx context.Context, payload []byte) {
    // Check if PR was merged
    if event.Action == "closed" && event.PullRequest.Merged {
        // Success! The decision led to good code
        s.selfImprovingService.RecordOutcome(ctx, executionID, true)
    }
}

// From CI/CD webhook:
func (im *IntegrationManager) handleWorkflowOutcome(w http.ResponseWriter, r *http.Request) {
    // POST /webhooks/outcome
    // {
    //   "execution_id": "uuid",
    //   "pr_number": 123,
    //   "merged": true,
    //   "build_passed": true
    // }
    
    success := req.Merged && req.BuildPassed
    s.selfImprovingService.RecordOutcome(ctx, req.ExecutionID, success)
}
```

## 4. Pattern Learning Cycle

### Every 100 Decisions:

```go
// Automatic trigger
func (s *SelfImprovingService) LearnPatterns(ctx context.Context, agentRole string) {
    // 1. Fetch last 100 decisions
    decisions, _ := s.repo.GetRecentDecisions(ctx, agentRole, 100)
    
    // 2. Group by issue type
    grouped := s.groupDecisionsByIssueType(decisions)
    
    // 3. For each issue type, calculate success rate
    for issueType, typedecisions := range grouped {
        accuracy := s.calculateAccuracy(typedecisions)
        
        // If accuracy >= 80%, create pattern
        if accuracy >= 0.80 {
            pattern := &domain.LearnedPattern{
                AgentRole:       agentRole,
                IssueType:       issueType,
                SuccessRate:     accuracy,
                ConfidenceBase:  accuracy * 0.95,
                TimesApplied:    len(typedecisions),
            }
            s.repo.SavePattern(ctx, pattern)
        }
    }
    
    // 4. Update agent specialization
    s.UpdateAgentSpecialization(ctx)
    
    // 5. Calibrate confidence
    s.CalibrateConfidence(ctx, agentRole)
}
```

### Pattern Application in Decision Generation

```go
func (s *Service) GenerateDecision(ctx context.Context, issue *Issue, agent string) {
    // 1. Classify issue
    issueType := s.classifyIssue(issue.Title, issue.Body)
    
    // 2. Find matching patterns
    patterns, _ := s.selfImprovingService.repo.FindPatterns(ctx, agent, issueType)
    
    // 3. Apply best pattern
    decision := s.generateDecision(agent, issue)
    
    for _, pattern := range patterns {
        if pattern.SuccessRate >= 0.90 {
            // Highly reliable pattern
            decision.Confidence = 0.98
            decision.RequiresApproval = false
            break
        } else if pattern.SuccessRate >= 0.80 {
            // Good pattern
            decision.Confidence = pattern.ConfidenceBase
            decision.RequiresApproval = decision.Confidence < 0.90
            break
        }
    }
    
    return decision
}
```

## 5. Crawl-Walk-Run Phases

### Phase 1: Crawl (Initial, MVP)

**Goal:** Manual workflow generation with approval

```yaml
Features:
  - Receive GitHub events
  - Generate agent decisions
  - All decisions require approval (confidence_threshold = 0.95)
  - Collect metrics for learning
  - Store approvals for analysis

Approval Required: 100% of decisions
LLM Integration: Mock (heuristic-based)
Self-Improvement: Learning data collected
Database: In-memory or file-based
Timeline: Week 1-2 ✅ COMPLETE
```

### Phase 2: Walk (Learning Enabled)

**Goal:** Self-improvement with graduated autonomy

```yaml
Triggered When: 100+ decisions with outcomes recorded

Features:
  - Pattern discovery on decision data
  - Confidence calibration from approvals
  - High-confidence decisions skip approval
  - Agent specialization tracking
  - Apply learned patterns

Approval Required: Decisions with confidence < 0.90
LLM Integration: Real Claude API
Self-Improvement: Patterns discovered every 100 decisions
Database: PostgreSQL with schema
Timeline: Week 3-4

Implementation:
  1. Connect real Claude API
  2. Implement pattern discovery
  3. Enable approval skipping
  4. Build self-improvement dashboard
```

### Phase 3: Run (Fully Autonomous)

**Goal:** High-confidence autonomous execution

```yaml
Triggered When: Average decision accuracy > 90%

Features:
  - Autonomous execution of decisions
  - LLM as judge validates other agents
  - Multi-skill composition (combine skills)
  - GitHub automation (create PRs, merge)
  - Proactive agent recommendations
  - Full audit trail and rollback

Approval Required: Only for 95%+ confidence decisions
LLM Integration: Claude with tool use for GitHub API
Self-Improvement: Real-time feedback
Database: PostgreSQL with advanced analytics
Timeline: Week 5-8

New Skills:
  - create-pr: Generate and push code changes
  - merge-pr: Merge approved PRs
  - create-issue: Create bugs from logs
  - pr-walkthrough: Explain PR changes
  - performance-analyze: Profile code
  - security-scan: Run SAST tools
```

## 6. Database Schema Quick Reference

### Core Tables for Self-Improvement

```sql
-- Decisions and their outcomes
agent_decisions (ID, ExecutionID, AgentRole, Decision, Confidence, ApprovalStatus)

-- Human approvals
approvals (ID, DecisionID, Status, ApproverID, ApprovalComments)

-- Metrics for each decision
agent_metrics (ID, AgentRole, DecisionCorrect, Confidence, IssueType, CreatedAt)

-- Discovered patterns
learned_patterns (ID, AgentRole, IssueType, SuccessRate, ConfidenceBase, TimesApplied)

-- Confidence calibration
confidence_calibration (AgentRole, ConfidenceBin, Calibration, SampleCount)

-- Agent specialization
agent_specialization (Agent, SpecialtyType, SuccessRate, DecisionCount)
```

## 7. Monitoring & Dashboards

### Metrics to Track

```go
type SelfImprovementMetrics struct {
    // Agent performance
    AgentSuccessRates    map[string]float64 // "Architect": 0.87
    AgentSpecializations map[string][]string
    
    // Learning progress
    PatternsDiscovered      int
    HighConfidencePatterns  int
    PatternAccuracy         float64
    
    // Feedback loop health
    ApprovalRate            float64 // % requiring approval
    HumanAcceptanceRate     float64 // % of approvals accepted
    AvgDecisionConfidence   float64
    
    // Trend
    SuccessRateTrend        []float64 // Last 30 days
    ConfidenceTrend         []float64
    PatternUtilization      int // % using learned patterns
}
```

### Dashboard Endpoints

```
GET /metrics/self-improvement → Returns SelfImprovementMetrics
GET /metrics/agents/:role → Agent-specific metrics
GET /patterns → List discovered patterns
GET /specializations → Agent specialization matrix
GET /dashboard/trends → Trend data for charts
```

## 8. Testing the Integration

### Unit Tests

```bash
# Self-improving service tests
go test ./internal/application/... -v

# GitHub webhook handler tests
go test ./internal/adapters/http/... -v

# Skill execution tests
go test ./internal/domain/services/... -v
```

### Integration Tests

```bash
# 1. Send test GitHub webhook
curl -X POST http://localhost:8000/webhooks/github \
  -H "X-Hub-Signature-256: sha256=..." \
  -d @test-pr-opened.json

# 2. Approve a decision
curl -X POST http://localhost:8000/api/approvals \
  -d '{"approval_id": "uuid", "status": "approved"}'

# 3. Record outcome
curl -X POST http://localhost:8000/webhooks/outcome \
  -d '{"execution_id": "uuid", "merged": true, "build_passed": true}'

# 4. Check metrics
curl http://localhost:8000/metrics/self-improvement
```

### Manual Testing Checklist

- [ ] GitHub webhook signature verification works
- [ ] PR opened event triggers code review
- [ ] Issue created event triggers triage
- [ ] Decisions are stored with correct confidence
- [ ] Approvals update decision status
- [ ] Patterns are discovered after 100 decisions
- [ ] High-confidence decisions skip approval (Phase 2)
- [ ] Self-improvement metrics dashboard loads
- [ ] Agent specialization updates correctly
- [ ] Confidence calibration improves over time

## 9. Known Limitations & Future Work

### Current Limitations (Phase 1: Crawl)

- [ ] Decisions require manual classification of issue type
- [ ] Learning disabled (Phase 2 feature)
- [ ] No GitHub action execution
- [ ] MockClaudeClient used (needs real Claude API)
- [ ] In-memory storage (needs PostgreSQL migration)
- [ ] No LLM-as-judge validation

### Phase 2 Blockers

- [ ] Real Claude API integration
- [ ] PostgreSQL schema migration
- [ ] Pattern discovery algorithm refinement
- [ ] Confidence calibration curves
- [ ] Self-improvement metrics collection

### Phase 3 Blockers

- [ ] GitHub API for PR creation/merge
- [ ] Code change validation
- [ ] Automated testing infrastructure
- [ ] Rollback mechanisms
- [ ] Comprehensive audit logging

## 10. Next Steps

1. **This Week:**
   - Integrate real Claude API
   - Set up PostgreSQL database
   - Deploy to staging environment
   - Test with real GitHub webhooks

2. **Next Week:**
   - Implement pattern discovery
   - Enable approval skipping for high-confidence decisions
   - Build self-improvement dashboard

3. **Week 3+:**
   - Add more skills (security-scan, performance-analyze)
   - Implement LLM-as-judge
   - Enable autonomous execution
   - Set up monitoring and alerting

---

## Warp Reference Architecture

This implementation is based on Warp's Factory Stack patterns:
- **Crawl-Walk-Run:** Progressive autonomy through proven patterns
- **Self-Improving Agents:** Learning from approval feedback
- **LLM as Judge:** Quality validation at each step
- **Skill-Based Architecture:** Reusable, composable capabilities
- **GitHub-Native:** Event-driven automation

See: https://www.warp.dev/blog/the-factory-stack
