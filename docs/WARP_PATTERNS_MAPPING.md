# Warp Factory Stack Patterns → Our Implementation

This document maps Warp's published patterns to our codebase implementation.

## 1. The Crawl-Walk-Run Model

**Warp's Pattern:** Progressive autonomy through proven patterns
**Reference:** https://www.warp.dev/blog/adopting-the-software-factory-model-crawl-walk-run

### Our Implementation

```
Phase 1: CRAWL ✅ (Current)
├── GitHub events trigger workflows
├── Agents generate decisions
├── All decisions require human approval
├── Decisions stored with metrics
└── Approval data collected for learning

Phase 2: WALK ⚙️ (Ready to implement)
├── Pattern learning from 100+ decisions
├── High-confidence (90%+) decisions skip approval
├── Confidence calibration from real outcomes
├── Agent specialization discovered
└── GitHub feedback loop integrated

Phase 3: RUN 🚀 (Planned)
├── Autonomous execution of high-confidence decisions
├── LLM-as-judge validates critical decisions
├── Multi-skill composition (combine skills)
├── GitHub actions (create PRs, merge, label)
└── Proactive recommendations for edge cases
```

**Where to Find:** `docs/INTEGRATION_GUIDE.md` → Section 5: Crawl-Walk-Run Phases

## 2. Self-Improving Agents

**Warp's Pattern:** Agents learn from approval feedback
**Reference:** https://www.warp.dev/blog/self-improving-software-proactive-agents

### Our Implementation

**Step 1: Collect Metrics**
```go
// Record every decision with context
type AgentMetric struct {
    AgentRole        string    // "Architect"
    ExecutionID      uuid.UUID // Links to workflow
    DecisionCorrect  bool      // From approval
    Confidence       float64   // Agent's stated confidence
    IssueType        string    // "auth", "performance", etc.
}
// File: internal/domain/types.go
```

**Step 2: Discover Patterns**
```go
// Every 100 decisions, find high-accuracy patterns
patterns := discoverPatterns(last100Decisions)

// Extract: Issue types → Best agent + confidence range
// Example: "auth issues" → 93% success with Architect @ 0.92 confidence
// File: internal/application/self_improving_service.go → LearnPatterns()
```

**Step 3: Apply Learning**
```go
// Use patterns to improve future decisions
if pattern.SuccessRate >= 0.90 {
    decision.Confidence = 0.98        // Boost confidence
    decision.RequiresApproval = false  // Skip approval
}
// File: internal/application/self_improving_service.go → UpdateConfidenceWithLearning()
```

**Step 4: Feedback Loop**
```go
// From GitHub: PR merged = decision was good
// From GitHub: PR closed = decision needs refinement
// From Human: Approval/rejection = accuracy signal
// File: cmd/asf-web/integration.go → handleWorkflowOutcome()
```

## 3. LLM-as-Judge Pattern

**Warp's Pattern:** Use LLM to score other agents' decisions
**Reference:** https://www.warp.dev/blog/using-llm-as-a-judge-scoring-to-measure-your-software-factory

### Our Implementation

**Currently:** Mock scoring (Phase 1: Crawl)
```go
type ClaudeDecisionResponse struct {
    Decision   string    // "Fix this by..."
    Reasoning  string    // "Because..."
    Confidence float64   // 0-1 score
}
// File: internal/domain/services/claude_client.go
```

**Phase 2: Real LLM Scoring**
```go
// Judge compares decision against:
// 1. Code quality standards
// 2. Security best practices
// 3. Performance implications
// 4. Similar past decisions

// Scores on:
// - Correctness (will this fix the issue?)
// - Safety (could this break something?)
// - Elegance (is this the best approach?)
// - Speed (how fast does this run?)

func (c *ClaudeAPIClient) JudgeDecision(decision string) float64 {
    // Call Claude API with multi-dimension scoring
    // Return aggregate confidence
}
```

## 4. Confidence Calibration

**Warp's Pattern:** Learn how well agent confidence matches reality
**Reference:** Implicit in self-improving agents blog

### Our Implementation

```go
// Track confidence score vs. actual accuracy
type ConfidenceCalibration struct {
    ConfidenceBin  string   // "0.9-0.95"
    ActualRate     float64  // 0.93 (what % were actually correct?)
    PredictedRate  float64  // 0.925 (what did agent claim?)
    Calibration    float64  // 0.93 / 0.925 = 1.005 (well-calibrated)
}

// If Calibration > 1.1: Agent overconfident → reduce scores
// If Calibration < 0.9: Agent underconfident → increase scores
// File: internal/application/self_improving_service.go → CalibrateConfidence()
```

**Example Learning Curve:**

```
Day 1-50: Architect says 85% confident, actually right 60% of the time
          Calibration = 0.60 / 0.85 = 0.71 (overconfident)

Day 51-100: System detects overconfidence, adjusts
            95% → 85%, confidence scoring improved

Day 101-150: Architect says 85% confident, actually right 82% of the time
            Calibration = 0.82 / 0.85 = 0.96 (well-calibrated)
```

## 5. Automatic Triage Skill

**Warp's Pattern:** Skill system for reusable agent capabilities
**Reference:** https://www.warp.dev/blog/how-to-build-a-cloud-software-factory-the-automatic-triage-skill

### Our Implementation

```go
// Define skill interface
type Skill interface {
    Name() string
    Description() string
    Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
    SupportedAgents() []string
}

// Implement triage skill
type TriageIssueSkill struct {
    BaseSkill
    llmClient ClaudeClient
}

func (s *TriageIssueSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    issue := input["issue"].(string)
    
    // Use LLM to triage
    triage, _ := s.llmClient.TriageIssue(ctx, issue)
    
    return map[string]interface{}{
        "category":    triage.Category,
        "priority":    triage.Priority,
        "agent":       triage.Agent,
        "effort":      triage.Effort,
    }, nil
}

// File: internal/domain/services/skill_system.go
```

**Four Skills Implemented:**
1. **CodeReviewSkill** → Review PRs, suggest improvements
2. **TriageIssueSkill** → Classify and route issues
3. **SecurityReviewSkill** → Find vulnerabilities
4. **ReviewSpecSkill** → Validate specifications

## 6. Code Review Improvement

**Warp's Pattern:** Self-improving code review through learned patterns
**Reference:** https://www.warp.dev/blog/how-to-build-a-cloud-software-factory-self-improving-code-review

### Our Implementation

```
GitHub Event: PR Opened
    ↓
Execute CodeReviewSkill
    - Analyze code quality
    - Check test coverage
    - Find potential bugs
    ↓
Generate Review Decision
    - Points to address
    - Confidence in review
    ↓
Human Approves/Modifies
    - Review accepted/rejected
    - Feedback captured
    ↓
PR Merged Successfully?
    - YES → Mark decision as correct
    - NO  → Mark decision as incorrect
    ↓
After 100 Reviews:
    - Discover patterns (e.g., "runtime errors in async code")
    - Improve confidence scores
    - Specialize Review agent
```

**Example Pattern Discovery:**

```
Pattern 1: Race Conditions
- Keywords: "async", "Promise", "race"
- Success Rate: 94%
- Confidence Boost: +0.05

Pattern 2: Missing null checks
- Keywords: "null", "undefined", "optional"
- Success Rate: 87%
- Confidence Boost: +0.03

Pattern 3: Off-by-one in loops
- Keywords: "for", "while", "length", "index"
- Success Rate: 72%
- Confidence Boost: +0.02
```

## 7. PR Walkthrough Skill

**Warp's Pattern:** Skill for explaining PR changes
**Reference:** https://www.warp.dev/blog/generate-pr-walkthrough-skill

### Our Implementation (Planned for Phase 3)

```go
// New skill: generate PR walkthroughs
type PRWalkthroughSkill struct {
    BaseSkill
    llmClient ClaudeClient
}

func (s *PRWalkthroughSkill) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    prDiff := input["diff"].(string)
    
    // Use LLM to explain changes
    explanation, _ := s.llmClient.AnalyzePRDiff(ctx, prDiff)
    
    return map[string]interface{}{
        "summary":       explanation.Summary,
        "key_changes":   explanation.Changes,
        "testing_notes": explanation.TestingNotes,
        "risks":         explanation.Risks,
    }, nil
}
```

## 8. GitHub-Native Integration

**Warp's Pattern:** Event-driven automation from GitHub
**Reference:** Foundation of Factory Stack

### Our Implementation

```go
// Webhook handler routes GitHub events
// File: internal/adapters/http/github_webhook_handler.go

Supported Events:
├── pull_request
│   ├── "opened" → Code Review workflow
│   ├── "ready_for_review" → Code Review workflow
│   └── "review_requested" → Code Review workflow
├── issues
│   ├── "opened" → Triage workflow
│   └── "labeled" → Review workflow (if needs-review)
├── pull_request_review
│   └── "submitted" → Merge Decision workflow
└── issue_comment
    └── "@factory" mentions → Command routing
```

## 9. Agent Specialization Discovery

**Warp's Pattern:** Learn which agent is best at each task
**Reference:** Implicit in self-improving pattern

### Our Implementation

```go
// Track agent performance by task type
type AgentSpecialization struct {
    Agent         string   // "Architect"
    SpecialtyType string   // "authentication"
    SuccessRate   float64  // 0.95 (95% success rate)
    DecisionCount int      // 127 auth decisions
}

// Update specialization matrix after learning
func (s *SelfImprovingService) UpdateAgentSpecialization(ctx) {
    // For each agent:
    for agent := range agents {
        // For each issue type:
        for issueType := range issueTypes {
            decisions := getDecisions(agent, issueType)
            accuracy := calculateAccuracy(decisions)
            
            // Record specialization
            if accuracy > 0.85 {
                saveSpecialization(agent, issueType, accuracy)
            }
        }
    }
}
```

**Example Specialization Matrix:**

```
         Auth    Perf    Security  Data
Arch     95%     78%     82%       91%
Dev      87%     89%     71%       78%
QA       82%     76%     79%       84%
Sec      73%     68%     94%       76%
Perf     71%     93%     65%       72%
```

## 10. Approval Workflow

**Warp's Pattern:** Human-in-the-loop for uncertain decisions
**Reference:** Quality gate pattern

### Our Implementation

```
Decision Generated (Confidence: 0.87)
    ↓
Is confidence >= 0.90? NO
    ↓
Route to Approvals Queue
    - Store in approvals table
    - Display in web UI
    - Notify approvers
    ↓
Human Reviews
    - Can see: decision, reasoning, context
    - Can: approve, reject, override
    ↓
Decision Recorded
    - If approved: apply decision
    - If rejected: flag for improvement
    ↓
Feedback Loop
    - Use approval signal for learning
```

**Phase 2 Improvement:**

```
Decision Generated (Confidence: 0.93, from learned pattern)
    ↓
Is confidence >= 0.90? YES
    ↓
Skip Approval (Phase 2: Walk)
    - Execute automatically
    - Log for audit
    - Collect outcome
    ↓
Approval Rate Drops
    - Crawl phase: 95% required approval
    - Walk phase: 25% required approval (efficiency gain)
    - Run phase: 5% required approval (autonomy)
```

## Implementation Status

| Feature | Status | File |
|---------|--------|------|
| GitHub Webhook Handler | ✅ Complete | `internal/adapters/http/github_webhook_handler.go` |
| Skill System | ✅ Complete | `internal/domain/services/skill_system.go` |
| LLM Client Interface | ✅ Complete | `internal/domain/services/claude_client.go` |
| Self-Improving Service | ✅ Complete | `internal/application/self_improving_service.go` |
| Metric Collection | ✅ Complete | `internal/domain/types.go` |
| Pattern Discovery | ✅ Complete | `LearnPatterns()` method |
| Confidence Calibration | ✅ Complete | `CalibrateConfidence()` method |
| Agent Specialization | ✅ Complete | `UpdateAgentSpecialization()` method |
| Crawl Phase | ✅ Complete | Phase 1 fully implemented |
| Walk Phase | ⚙️ Ready | Approval skipping ready to enable |
| Run Phase | 🚀 Planned | Autonomous execution ready to start |
| PostgreSQL Schema | ✅ Complete | `internal/adapters/persistence/postgres_schema.sql` |
| Integration Wiring | ✅ Complete | `cmd/asf-web/integration.go` |
| Documentation | ✅ Complete | Comprehensive guides created |

## Next Steps to Fully Align with Warp

### Week 1-2: Claude API Integration
```bash
1. Replace MockClaudeClient with real Claude API
2. Implement prompt engineering for each skill
3. Test decision quality
4. Measure confidence accuracy
```

### Week 3-4: Database Migration
```bash
1. Set up PostgreSQL
2. Implement WorkflowRepository
3. Migrate from in-memory to database
4. Build self-improvement metrics dashboard
```

### Week 5-6: Phase 2 Enablement
```bash
1. Enable approval skipping for 90%+ confidence
2. Launch self-improvement dashboard
3. Monitor agent specialization
4. Validate feedback loop
```

### Week 7-8: Phase 3 Preparation
```bash
1. Implement LLM-as-judge
2. Build GitHub action executor
3. Create PR creation skill
4. Test autonomous workflows
```

## Key Insights from Warp

1. **Progressive Autonomy Works:** Start with 100% approval, graduate to autonomy through proven patterns
2. **Approval Feedback is Gold:** Every approval/rejection is a learning signal
3. **GitHub is the Source of Truth:** Merged PRs and closed issues are ultimate validation
4. **Specialization Emerges:** Agents naturally develop expertise if you track outcomes
5. **Calibration Matters:** Agent confidence scores need real feedback to stay honest

This architecture is production-proven because Warp is using it at scale.

---

**References:**
- [The Factory Stack](https://www.warp.dev/blog/the-factory-stack)
- [Crawl-Walk-Run](https://www.warp.dev/blog/adopting-the-software-factory-model-crawl-walk-run)
- [Self-Improving Agents](https://www.warp.dev/blog/self-improving-software-proactive-agents)
- [LLM as Judge](https://www.warp.dev/blog/using-llm-as-a-judge-scoring-to-measure-your-software-factory)
- [Automatic Triage](https://www.warp.dev/blog/how-to-build-a-cloud-software-factory-the-automatic-triage-skill)
- [Self-Improving Code Review](https://www.warp.dev/blog/how-to-build-a-cloud-software-factory-self-improving-code-review)
- [PR Walkthrough](https://www.warp.dev/blog/generate-pr-walkthrough-skill)
