# Human-in-the-Loop: Decision Points & Feedback Mechanisms

The Automatic Triage Factory implements **human oversight at critical decision points** rather than full automation. This guide explains where humans intervene and how feedback flows back.

---

## Architecture: Human Decision Points

```
┌─────────────────────────────────────────────────────────┐
│  Automatic: Issue Triage                                │
│  ├─ Categorize (feature/bug/refactor)                   │
│  ├─ Assess priority                                     │
│  └─ Determine readiness                                 │
│                                                         │
│  🔴 HUMAN DECISION POINT 1                             │
│  ├─ Review triage analysis                              │
│  ├─ Accept or override labels                           │
│  ├─ Ask clarifying questions if needed                  │
│  └─ Approve or reject "ready-to-implement"              │
└─────────────────────────────────────────────────────────┘
    ↓ (human approves ready-to-implement)
┌─────────────────────────────────────────────────────────┐
│  Automatic: Generate Spec                               │
│  ├─ Parse issue into structured spec                    │
│  ├─ Extract requirements                                │
│  └─ Generate acceptance criteria                        │
│                                                         │
│  🟡 OPTIONAL: Human Review of Spec                     │
│  ├─ Review generated spec                               │
│  ├─ Refine acceptance criteria                          │
│  └─ Adjust constraints/requirements                     │
└─────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────┐
│  Automatic: Generate Code                               │
│  ├─ Create implementation from spec                     │
│  ├─ Add error handling                                  │
│  ├─ Include tests                                       │
│  └─ Score quality                                       │
│                                                         │
│  🔴 HUMAN DECISION POINT 2                             │
│  ├─ Review generated code                               │
│  ├─ Test locally if needed                              │
│  ├─ Request changes or regenerate                       │
│  └─ Approve for PR                                      │
└─────────────────────────────────────────────────────────┘
    ↓ (human approves code)
┌─────────────────────────────────────────────────────────┐
│  Automatic: Create Pull Request                         │
│  ├─ Create branch                                       │
│  ├─ Commit code                                         │
│  ├─ Open PR with link to issue                          │
│  └─ Add code review checklist                           │
│                                                         │
│  🔴 HUMAN DECISION POINT 3                             │
│  ├─ Review PR in GitHub UI                              │
│  ├─ Run tests locally                                   │
│  ├─ Request changes via PR comments                     │
│  └─ Merge or close                                      │
└─────────────────────────────────────────────────────────┘
```

---

## Decision Point 1: Triage Review

### What Humans See

**GitHub Issue Comment (Auto-Posted):**

```
## Automatic Triage Report

**Category:** feature
**Priority:** medium
**Status:** ✅ Ready to Implement

**Analysis:**
Has clear acceptance criteria, sufficient context, and no blockers

**Labels Applied:** ready-to-implement, priority-medium, category-feature

_Analyzed by 7-Layer Automatic Triage Agent_
```

**Labels Applied:**
- `ready-to-implement` — automatic assessment
- `priority-medium` — detected priority
- `category-feature` — detected type

### Human Intervention Options

#### Option A: Accept Triage
```
Human Action: Comment "✅ LGTM" or approve via label
System Response: Proceeds to code generation
```

#### Option B: Override Label
```
Human Action: Remove label, add custom label
Example: Remove "ready-to-implement", add "needs-design-review"
System Response: Workflow stops, issue waits for next step
```

#### Option C: Request Changes
```
Human Action: Comment with clarifications needed
Example: "@factory please clarify: what's the cache eviction policy?"
System Response: 
  1. AI agent reads comment
  2. Agent adds "needs-info" label
  3. Workflow pauses
```

#### Option D: Override Priority
```
Human Action: Add different priority label
Example: Remove "priority-medium", add "priority-critical"
System Response: 
  1. New priority noted in metrics
  2. Sent to faster-track workflow
  3. Escalates in queue
```

### Implementation in Code

```go
// In triage handler, human feedback flows through labels
func HandleTriageStatus(w http.ResponseWriter, r *http.Request) {
    triageID := r.URL.Query().Get("id")
    result := triageStore[triageID]
    
    // Include suggested labels that human can override
    result.SuggestedLabels = []string{"ready-to-implement", "priority-medium"}
    
    // But we also check: did human override?
    actualLabels, _ := getIssueLabels(result.Repo, result.Issue)
    
    if !contains(actualLabels, "ready-to-implement") {
        // Human rejected or is waiting for something
        result.HumanOverride = true
        result.Status = "waiting_for_human_approval"
    }
}
```

---

## Decision Point 2: Code Review

### What Humans See

**Generated Code Posted to PR (or in status endpoint):**

```json
{
  "code": "package cache\nimport \"sync\"\n...",
  "score": 92,
  "issues": [],
  "testsPassed": 10,
  "acceptanceMet": true,
  "generatedAt": "2026-11-16T12:00:00Z"
}
```

**Quality Metrics:**
- Score: 92/100 ✅ Excellent
- Tests: 10 passing ✅
- Acceptance: Met ✅
- Issues: None detected ✅

### Human Intervention Options

#### Option A: Accept Code as-is
```
Human Action: Create PR with generated code
Command: 
  git checkout -b feature/cache
  # paste generated code
  git commit -m "Implement: Issue #5"
  git push origin feature/cache
  gh pr create --title "..." --body "..."
System Response: PR created, ready for review
```

#### Option B: Request Regeneration with Feedback
```
Human Action: Modify spec and regenerate
Steps:
  1. Download generated spec
  2. Edit spec file locally
  3. Add/modify constraints or requirements
  4. Call /api/agents/run again with updated spec
System Response: New code generated with updated spec
```

**Modified Spec Example:**

```yaml
# Before: basic cache
constraints:
  - type: "should"
    description: "Add expiration support"

# After: enhanced cache
constraints:
  - type: "must"
    description: "Implement TTL-based expiration"
  - type: "must"
    description: "Cleanup expired entries automatically"
  - type: "must"
    description: "Performance: evict in < 1ms"
```

#### Option C: Use Code as Baseline for Manual Changes
```
Human Action: Use generated code as starting point
Steps:
  1. Copy generated code to file
  2. Add custom features not in spec
  3. Enhance error handling
  4. Add advanced features
System Response: Foundation is solid, human extends it
```

#### Option D: Reject and Try Alternative Approach
```
Human Action: Discard and try with different spec
Steps:
  1. Modify spec significantly
  2. Change approach/design
  3. Call /api/agents/run with new spec
System Response: Different code generated from new spec
```

### Implementation: Code Regeneration Loop

```go
// Human can request regeneration with updated spec
type RegenerationRequest struct {
    TriageID    string // Original triage result
    UpdatedSpec string // Human-modified spec
    Notes       string // Human feedback/notes
}

func HandleRegenerate(w http.ResponseWriter, r *http.Request) {
    var req RegenerationRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Track that this is a regeneration
    // (enables metrics on iterations needed)
    result := regenerateCode(req.UpdatedSpec)
    result.OriginalTriageID = req.TriageID
    result.IterationNumber = 2  // Second attempt
    result.HumanFeedback = req.Notes
    
    return result
}
```

---

## Decision Point 3: Pull Request Review

### GitHub PR as Human Interface

**Generated PR (Auto-Created):**

```markdown
# Implement: Add thread-safe cache service

Closes #1

## Quality Score: 92%

- ✅ Tests: 10/10 passing
- ✅ Linting: No issues
- ✅ Acceptance criteria: Met
- ✅ Code coverage: 92%

Generated from automatic triage of GitHub issue.
Code analyzed by 7-layer factory.

---

## Checklist for Reviewer
- [ ] Code follows project conventions
- [ ] Tests are comprehensive
- [ ] Error handling is correct
- [ ] Performance is acceptable
- [ ] Documentation is clear
```

### Human Review Feedback Mechanisms

#### Mechanism 1: PR Comments for Issues

```
Human Comment:
@factory why are we using RWMutex instead of regular Mutex? 
A regular Mutex might be simpler here.

System Response:
1. AI reads comment
2. Creates issue on original triage result
3. Suggests spec modification
4. Regenerates code with explanation
```

#### Mechanism 2: Approval Workflow

```
Human Action: Review and Approve

System Response:
1. Sets "ready-to-merge" label
2. Runs final checks
3. Merges when all checks pass
```

#### Mechanism 3: Request Changes

```
GitHub UI: Click "Request Changes"
Comment: "Please add benchmarks for concurrent access"

System Response:
1. AI adds benchmarks to generated code
2. Creates new commit
3. Pushes to PR
4. Requests human re-review
```

### Implementation: PR Comment Parsing

```go
// System reads PR comments and acts on them
func HandlePRCommentWebhook(w http.ResponseWriter, r *http.Request) {
    var payload GitHubPRCommentWebhook
    json.NewDecoder(r.Body).Decode(&payload)
    
    comment := payload.Comment.Body
    
    // Detect human feedback patterns
    if strings.Contains(comment, "regenerate with") {
        // Extract modified spec from comment
        spec := extractSpec(comment)
        result := regenerateCode(spec)
        
        // Post new code as comment
        postComment(result.Code, "Regenerated based on your feedback")
    }
    
    if strings.Contains(comment, "add benchmarks") {
        // Regenerate with benchmark constraint
        spec := updateSpec(constraints{"must": "include benchmarks"})
        result := regenerateCode(spec)
        
        // Push to PR
        pushToPR(payload.PullRequest.HeadRef, result.Code)
    }
}
```

---

## Feedback Loop Mechanisms

### 1. Label-Based Control

**Humans control workflow via labels:**

```
Labels                          → Action
─────────────────────────────────────────────
needs-info                      → Stop, wait for clarification
ready-to-implement              → Proceed to generation
blocked                         → Pause all workflows
approved-for-generation         → Force code generation
approved-for-merge              → Auto-merge after checks
needs-redesign                  → Regenerate with new spec
```

### 2. Comment-Based Commands

**Humans give commands via comments:**

```
Comment                                 → Action
──────────────────────────────────────────────────────
@factory regenerate                     → Run code generation
@factory update spec: <yaml>            → Use custom spec
@factory add benchmarks                 → Modify and regenerate
@factory use approach: <description>    → New strategy
@factory approve                        → Mark ready
```

### 3. Direct API for Manual Workflows

**Humans can call API endpoints directly:**

```bash
# Regenerate with custom spec
curl -X POST http://localhost:8080/api/agents/run \
  -d '{
    "agentId": "CodeGen",
    "lldYaml": "... custom spec ...",
    "linkedToIssue": 5,
    "linkedToTriage": "abc123"
  }'

# Override triage decision
curl -X POST http://localhost:8080/api/github/triage/override \
  -d '{
    "triageId": "abc123",
    "newCategory": "refactor",
    "newPriority": "critical",
    "notes": "Human override for urgent fix"
  }'
```

---

## Metrics: Tracking Human Feedback

### Feedback Loop Metrics

```bash
curl http://localhost:8080/api/agents/metrics | jq .
```

**Example Response:**

```json
{
  "total_runs": 15,
  "average_score": 88.5,
  "human_approvals": 12,
  "human_rejections": 2,
  "human_modifications": 3,
  "regenerations": 2,
  "average_feedback_iterations": 1.3,
  "learned_patterns": {
    "cache-implementation": 5,
    "concurrency-patterns": 4,
    "error-handling-style": 6
  }
}
```

### Tracking Modifications

```go
type FeedbackMetrics struct {
    TriageID              string        // Original triage
    OriginalCategory      string        // auto-detected
    HumanOverrideCategory string        // human changed to
    
    OriginalScore         float64       // auto-generated score
    FinalScore            float64       // after human review
    
    RegenerationCount     int           // iterations needed
    ApprovedAfterN        int           // iterations before approval
    
    TimeToApproval        time.Duration // how long human took
    HumanNotes            string        // feedback text
}
```

---

## Example: Full Human-in-the-Loop Session

### Session: Building a Cache Service

**Step 1: Triage Auto-Posted**

```
Issue #5: "Add thread-safe cache service"

Automatic Triage Result:
✅ ready-to-implement
Category: feature
Priority: medium
Labels: ready-to-implement, priority-medium, category-feature
```

**Step 2: Human Reviews Triage**

```
Human Action: 
💬 Comment: "Actually this is higher priority - we need this for the Q4 release.
            Also we need TTL support for expiration."

System Action:
✅ Detects priority-bump mention
✅ Adds constraint: "TTL expiration support"
✅ Changes label to priority-high
```

**Step 3: Code Generated with TTL**

```
Automatic Generation:
✅ Code generated with TTL support
✅ Score: 88/100 (slightly lower due to complexity)
✅ Tests: 12 passing
```

**Step 4: Human Reviews Code**

```
Human Action:
💬 Comment: "@factory can we use time.AfterFunc instead of manual cleanup?
            Seems simpler and more efficient."

System Action:
✅ Parses suggestion
✅ Creates new spec constraint: "Use time.AfterFunc for cleanup"
✅ Regenerates code
```

**Step 5: Regenerated Code**

```
Automatic Regeneration:
✅ Code updated with time.AfterFunc
✅ Score: 91/100 (human suggestion improved quality)
✅ New approach adds simplicity
```

**Step 6: Human Approves**

```
Human Action:
✅ Approves PR in GitHub

System Action:
✅ Sets ready-to-merge label
✅ Merges PR
✅ Closes issue #5
```

**Metrics Recorded:**

```json
{
  "triageId": "abc123",
  "iterations": 2,
  "timeToApproval": "45 minutes",
  "humanFeedback": [
    "priority changed: medium → high",
    "constraint added: TTL support",
    "optimization suggested: time.AfterFunc"
  ],
  "scoreProgression": [88, 91],
  "finalScore": 91,
  "approved": true
}
```

---

## Best Practices: When to Override

### ✅ Good Times to Override Triage Decision

1. **You have domain knowledge the issue author didn't mention**
   ```
   Auto: medium priority
   Human: "This blocks the auth refactor, make it critical"
   Result: Issue gets expedited
   ```

2. **You see a better approach**
   ```
   Auto: Use approach X
   Human: "Actually approach Y is better because..."
   Result: Regenerate with new spec
   ```

3. **You need to add constraints the spec missed**
   ```
   Auto: Basic cache
   Human: "Need benchmarking and async support"
   Result: Enhanced spec, regenerate
   ```

### ❌ When NOT to Override

1. **The agent already got it right**
   - Don't second-guess working code
   - Trust the quality score

2. **You're just bikeshedding**
   - "I prefer this naming" → Don't regenerate
   - Accept reasonable choices

3. **The spec was clear and complete**
   - Agent should follow it exactly
   - Don't add new requirements mid-stream

---

## Configuration: Tuning Human Control

### Control Level Settings

```yaml
# config.yaml
automation:
  # How much does human need to actively approve?
  requireHumanApprovalFor:
    - triage_decision          # Always show to human
    - code_generation          # Always show to human
    - pull_request_creation    # Proceed without approval
    - merging                  # Always require approval
  
  # Confidence thresholds where human review kicks in
  thresholds:
    triage_score_min: 0.75      # If <75% confident, flag for human
    code_score_min: 0.85        # If code score <85%, request review
    auto_merge_score: 0.95      # Auto-merge only if score >95%
  
  # How much human feedback loops back?
  feedback_loops:
    parse_pr_comments: true     # Read PR comments for feedback
    parse_labels: true          # Act on label changes
    parse_issue_comments: true  # React to issue discussion
  
  # Escalation: when to loop in humans
  escalation:
    on_low_confidence: true     # Confidence < threshold
    on_regeneration_loop: true  # If need to regenerate
    on_test_failure: true       # If tests fail
    on_scoring_drop: true       # If quality score drops
```

---

## Summary

**Human-in-the-loop is implemented through:**

| Mechanism | Where | How |
|-----------|-------|-----|
| **Triage Review** | GitHub issue | Comments, labels, approvals |
| **Spec Modification** | PR/comments | Request changes, regenerate |
| **Code Review** | GitHub PR | Approve, request changes, comment |
| **Feedback Loop** | Comments + Labels | System reads and acts on them |
| **Direct Control** | API | Call endpoints directly |
| **Metrics** | Dashboard | Track feedback iterations |

**Key Principle:** Automation handles repetitive tasks (categorization, code generation), but humans make judgment calls (priority, design decisions, final approval).

The system is **designed for human collaboration, not replacement.**
