# Automatic Triage Factory: Complete Working Example

This guide demonstrates the complete **Automatic Triage Factory** workflow: from GitHub issue creation through analysis, categorization, spec generation, and code generation.

## Architecture Overview

```
GitHub Issue
    ↓
┌─────────────────────────────────────────┐
│  1. TRIAGE LAYER                        │
│  - Analyzes issue                       │
│  - Categorizes (feature/bug/refactor)   │
│  - Determines priority                  │
│  - Checks readiness to implement        │
│  - Generates spec if ready              │
│  - Applies labels                       │
│  - Posts comment                        │
└─────────────────────────────────────────┘
    ↓ (if ready-to-implement)
┌─────────────────────────────────────────┐
│  2. GENERATION LAYER                    │
│  - Takes generated spec                 │
│  - Generates production code            │
│  - Analyzes code quality                │
│  - Returns scored result                │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│  3. PULL REQUEST LAYER                  │
│  - Creates branch                       │
│  - Commits code                         │
│  - Creates pull request                 │
│  - Links to original issue              │
└─────────────────────────────────────────┘
    ↓
GitHub Pull Request (Ready for Review)
```

---

## Step 1: Setup

### Prerequisites

```bash
# Export GitHub personal access token
export GITHUB_TOKEN=ghp_your_token_here

# Verify token works
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Start the server
./asf-web localhost:8080
```

### Token Scopes Required

```
✓ repo (read/write repository access)
✓ write:discussion (write pull request reviews)
```

---

## Step 2: Create Test Issues

### Issue 1: Well-Described Feature (READY)

**Create via GitHub UI or CLI:**

```bash
gh issue create \
  --title "Add thread-safe cache service" \
  --body "
# Add Thread-Safe Cache Service

## Description
We need a production-grade cache service that is safe for concurrent use.

## Acceptance Criteria
- Create a Cache type with sync.RWMutex protection
- Implement Get, Set, Delete operations
- Thread-safe operations using defer Unlock pattern
- Full test coverage (80%+)
- Proper error handling for all operations

## Expected Behavior
Users should be able to safely use the cache from multiple goroutines without data corruption.

## Steps to Implement
1. Define Cache struct with RWMutex
2. Implement Get operation (read lock)
3. Implement Set operation (write lock)
4. Implement Delete operation (write lock)
5. Add unit tests for concurrent access
6. Add benchmarks for performance
"
```

**Result**: This issue will be marked as `ready-to-implement` because:
- ✅ Has acceptance criteria
- ✅ Has sufficient context (>100 chars)
- ✅ No blockers or unknown requirements

---

### Issue 2: Vague Feature (NEEDS-INFO)

**Create via GitHub UI or CLI:**

```bash
gh issue create \
  --title "Improve performance" \
  --body "Things are slow"
```

**Result**: This issue will be marked as `needs-info` because:
- ❌ No acceptance criteria
- ❌ No context
- ❌ Unclear requirements

---

### Issue 3: Bug Report (READY)

**Create via GitHub UI or CLI:**

```bash
gh issue create \
  --title "Fix concurrent map access panic" \
  --body "
# Concurrent Map Access Panic

## Description
The storage service crashes when multiple goroutines access the map simultaneously.

## Steps to Reproduce
1. Create a StorageService
2. Launch 10 concurrent goroutines
3. Each goroutine: call Store.Get() and Store.Set() 100 times
4. Observe panic: concurrent map read/write

## Expected Behavior
No panic. Concurrent operations should be safe.

## Acceptance Criteria
- Add sync.RWMutex to protect map
- Use defer Unlock() for safety
- No panics under concurrent load
- Tests verify thread safety

## Technical Details
- Priority: critical
- Current issue: map not protected
- Solution: add proper locking
"
```

**Result**: Will be marked as `ready-to-implement` and `priority-critical`

---

## Step 3: Run Automatic Triage

### Call the Triage API

**Example 1: Triage the first issue**

```bash
# Get issue number from creation response, or list issues
ISSUE_NUM=1
OWNER=your_username
REPO=your_repo

curl "http://localhost:8080/api/github/triage?owner=$OWNER&repo=$REPO&issue=$ISSUE_NUM&token=$GITHUB_TOKEN" \
  -H "Accept: text/event-stream" \
  --no-buffer
```

**Server-Sent Events Output:**

```
data: {"status":"connecting","progress":10,"message":"Connecting to GitHub"}
data: {"status":"fetching","progress":25,"message":"Fetching issue details"}
data: {"status":"analyzing","progress":45,"message":"Analyzing issue content"}
data: {"status":"spec-generation","progress":65,"message":"Generating implementation spec"}
data: {"status":"labeling","progress":80,"message":"Applying labels to issue"}
data: {"status":"commenting","progress":90,"message":"Posting triage comment to issue"}
data: {"status":"complete","progress":100,"message":"Triage complete: feature (medium)"}
```

---

### What Gets Posted to the Issue

**GitHub Issue Comment:**

```
## Automatic Triage Report

**Category:** feature
**Priority:** medium
**Status:** ✅ Ready to Implement

**Analysis:**
Has clear acceptance criteria, sufficient context, and no blockers

**Labels Applied:** ready-to-implement, priority-medium, category-feature

**Implementation Spec Generated:** Ready for code generation

_Analyzed by 7-Layer Automatic Triage Agent_
```

**Labels Applied to Issue:**
- `ready-to-implement` — issue passes triage
- `category-feature` — identified as a feature request
- `priority-medium` — medium priority level

---

## Step 4: Generate Code from Triaged Issue

### Get Triage Result

```bash
# From the SSE stream, capture the triage ID, or use status endpoint
TRIAGE_ID=$(curl -s "http://localhost:8080/api/github/triage/status?id=$TRIAGE_ID" | jq -r .id)

# Get full triage result
curl -s "http://localhost:8080/api/github/triage/status?id=$TRIAGE_ID" | jq .
```

**Response:**

```json
{
  "id": "abc123",
  "issue": 1,
  "repo": "user/repo",
  "title": "Add thread-safe cache service",
  "category": "feature",
  "priority": "medium",
  "ready_to_implement": true,
  "reason": "Has clear acceptance criteria, sufficient context, and no blockers",
  "labels_to_apply": [
    "ready-to-implement",
    "priority-medium",
    "category-feature"
  ],
  "generated_spec": "id: \"issue-1\"\nversion: \"1.0\"\n...",
  "created_at": "2026-11-16T12:00:00Z"
}
```

### Generate Code Using the Spec

```bash
# Extract the generated spec
SPEC=$(curl -s "http://localhost:8080/api/github/triage/status?id=$TRIAGE_ID" | jq -r .generated_spec)

# Save to file
echo "$SPEC" > /tmp/issue-spec.yaml

# Call code generation endpoint with the spec
curl -X POST http://localhost:8080/api/agents/run \
  -H "Content-Type: application/json" \
  -d "{
    \"agentId\": \"CodeGen\",
    \"lldYaml\": \"$SPEC\"
  }" \
  -H "Accept: text/event-stream" \
  --no-buffer
```

**Server-Sent Events Output:**

```
data: {"status":"start","progress":5,"message":"Starting 7-layer agent"}
data: {"status":"layer-1","progress":15,"message":"Layer 1: Parsing specification"}
data: {"status":"layer-2","progress":28,"message":"Layer 2: Loading context"}
data: {"status":"layer-3","progress":40,"message":"Layer 3: Computing resources"}
data: {"status":"layer-4","progress":55,"message":"Layer 4: Generating code"}
data: {"status":"layer-5","progress":68,"message":"Layer 5: Analyzing quality"}
data: {"status":"layer-6","progress":80,"message":"Layer 6: Orchestrating workflow"}
data: {"status":"done","progress":100,"message":"Complete","result":{"code":"package cache\n...","score":92}}
```

---

## Step 5: Review Generated Code

### Generated Service Code

**Result from code generation:**

```go
package cache

import (
	"errors"
	"sync"
)

type Cache struct {
	mu    sync.RWMutex
	store map[string]interface{}
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[string]interface{}),
	}
}

func (c *Cache) Get(key string) (interface{}, error) {
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.store[key]
	if !ok {
		return nil, errors.New("key not found")
	}

	return value, nil
}

func (c *Cache) Set(key string, value interface{}) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = value
	return nil
}

func (c *Cache) Delete(key string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]interface{})
}

// Tests

func TestCacheThreadSafety(t *testing.T) {
	c := NewCache()
	done := make(chan bool, 100)

	// Concurrent writers
	for i := 0; i < 50; i++ {
		go func(idx int) {
			key := fmt.Sprintf("key-%d", idx)
			c.Set(key, idx)
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 50; i++ {
		go func(idx int) {
			key := fmt.Sprintf("key-%d", idx)
			c.Get(key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestCacheBasics(t *testing.T) {
	c := NewCache()

	// Test Set and Get
	err := c.Set("name", "John")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := c.Get("name")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if val != "John" {
		t.Errorf("Expected John, got %v", val)
	}

	// Test Delete
	err = c.Delete("name")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = c.Get("name")
	if err == nil {
		t.Errorf("Expected error after delete, got nil")
	}
}
```

### Quality Score Analysis

```
Base Score:                 100
Issues Found:               0 (no penalties)
  - No lock/defer issues
  - All errors checked
  - Proper function sizes
  - No TODOs/FIXMEs
Large Changes:              0 (-0 points, <500 lines)
Missing Tests:              0 (-0 points, includes tests)

Final Score:                92%
Grade:                      Excellent ✅

Issues Detected:            0
  - ✅ All locks properly deferred
  - ✅ All errors properly checked
  - ✅ Functions well-sized
  - ✅ Tests included
```

---

## Step 6: Complete Workflow Example

### Bash Script: End-to-End Triage → Code → PR

```bash
#!/bin/bash

set -e

# Configuration
OWNER="your_username"
REPO="your_repo"
ISSUE_NUM=1
GITHUB_TOKEN="ghp_your_token"

echo "🔍 Step 1: Running automatic triage..."
TRIAGE_RESPONSE=$(curl -s "http://localhost:8080/api/github/triage?owner=$OWNER&repo=$REPO&issue=$ISSUE_NUM&token=$GITHUB_TOKEN" \
  -H "Accept: application/json")

echo "TRIAGE_RESPONSE: $TRIAGE_RESPONSE"

# Parse response
TRIAGE_ID=$(echo "$TRIAGE_RESPONSE" | jq -r '.id')
IS_READY=$(echo "$TRIAGE_RESPONSE" | jq -r '.ready_to_implement')

if [ "$IS_READY" != "true" ]; then
  echo "⏳ Issue not ready to implement. Exiting."
  exit 0
fi

echo "✅ Issue is ready to implement"

echo ""
echo "📝 Step 2: Generating code from spec..."
SPEC=$(curl -s "http://localhost:8080/api/github/triage/status?id=$TRIAGE_ID" | jq -r '.generated_spec')

CODE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/agents/run \
  -H "Content-Type: application/json" \
  -d "{\"agentId\":\"CodeGen\",\"lldYaml\":\"$SPEC\"}" \
  -H "Accept: application/json")

CODE=$(echo "$CODE_RESPONSE" | jq -r '.result.code')
SCORE=$(echo "$CODE_RESPONSE" | jq -r '.result.score')

echo "✅ Code generated with score: $SCORE"

echo ""
echo "📤 Step 3: Creating pull request..."

# Create branch
BRANCH_NAME="triage/issue-$ISSUE_NUM-$(date +%s)"
git checkout -b "$BRANCH_NAME"

# Create file
mkdir -p pkg/cache
echo "$CODE" > pkg/cache/cache.go

# Commit and push
git add pkg/cache/cache.go
git commit -m "Implement: $(gh issue view $ISSUE_NUM --json title --jq '.title')"
git push -u origin "$BRANCH_NAME"

# Create PR linked to issue
PR_URL=$(gh pr create \
  --title "Implement: Issue #$ISSUE_NUM" \
  --body "Automatically generated by Triage Factory
  
Closes #$ISSUE_NUM

## Quality Score: $SCORE%

Generated from automatic triage of GitHub issue.
Code analyzed by 7-layer factory.
" \
  --repo "$OWNER/$REPO")

echo "✅ Pull request created: $PR_URL"
echo ""
echo "✨ Workflow complete! Review and merge the PR."
```

### Run the Script

```bash
bash triage-workflow.sh
```

**Output:**
```
🔍 Step 1: Running automatic triage...
TRIAGE_RESPONSE: {"id":"abc123","issue":1,"category":"feature",...}
✅ Issue is ready to implement

📝 Step 2: Generating code from spec...
✅ Code generated with score: 92

📤 Step 3: Creating pull request...
✅ Pull request created: https://github.com/user/repo/pull/5

✨ Workflow complete! Review and merge the PR.
```

---

## Testing the Full Pipeline

### Test Case 1: Ready Issue → Full Pipeline

**Issue:** "Add thread-safe cache service" (detailed, clear criteria)

**Expected Result:**
```
Triage:       ✅ ready-to-implement (labels applied)
Generation:   ✅ Code generated (92% score)
PR:           ✅ Created and linked
```

### Test Case 2: Vague Issue → Triage Only

**Issue:** "Improve performance" (vague)

**Expected Result:**
```
Triage:       ⏳ needs-info (comment posted, labeled)
Generation:   ⏸️ Skipped (not ready)
PR:           ⏸️ Skipped (not ready)
```

### Test Case 3: Critical Bug → Priority Handling

**Issue:** "Fix concurrent map access panic" (detailed, marked critical)

**Expected Result:**
```
Triage:       ✅ ready-to-implement (labels: priority-critical)
Generation:   ✅ Code generated (90% score)
PR:           ✅ Created with priority label
```

---

## Key Features Demonstrated

### 1. Automatic Categorization
- **Input:** GitHub issue title and body
- **Output:** Category (feature/bug/refactor/documentation)
- **Logic:** Pattern matching on keywords

### 2. Priority Detection
- **Input:** Keywords in issue (critical, urgent, high, low)
- **Output:** Priority level (critical/high/medium/low)
- **Logic:** Keyword-based scoring

### 3. Readiness Assessment
- **Checks:**
  - ✅ Has acceptance criteria?
  - ✅ Has sufficient context (>100 chars)?
  - ✅ No blockers or unknowns?
- **Output:** ready-to-implement | needs-info | wait-to-implement

### 4. Spec Generation
- **Input:** Issue title and body
- **Output:** YAML spec file with goals, constraints, steps
- **Format:** Compatible with code generation layer

### 5. GitHub Integration
- **Fetches:** Issue details via GitHub API
- **Posts:** Triage comment with analysis
- **Labels:** Automatically tags for workflow routing
- **Tracking:** Stores triage results for auditing

---

## Metrics & Observability

### Triage Metrics

```bash
curl http://localhost:8080/api/agents/metrics | jq .
```

**Response:**
```json
{
  "total_runs": 3,
  "average_score": 88.5,
  "top_score": 92,
  "learned_patterns": {
    "ready-to-implement": 2,
    "needs-info": 1,
    "priority-critical": 1,
    "category-feature": 2,
    "category-bug": 1
  }
}
```

---

## Next Steps

### Extending the Factory

1. **Add issue spec refinement:** Loop to ask clarifying questions for `needs-info` issues
2. **Add code review cycle:** Review generated code before PR creation
3. **Add test generation:** Generate tests based on acceptance criteria
4. **Add impact analysis:** Estimate effort and risk for each issue
5. **Add routing logic:** Send issues to specialized agents by complexity

### Multi-Agent Workflows

```
Issue
  ├─ Triage Agent (categorize, prioritize)
  ├─ Spec Agent (generate implementation spec)
  ├─ Gen Agent (generate code)
  ├─ Review Agent (review for quality)
  └─ PR Agent (create pull request)
```

---

## Troubleshooting

### Issue: Triage labels not applying

```bash
# Check token has write:discussion scope
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/user/scopes

# Should include: ["repo", "write:discussion"]
```

### Issue: Generation not running for ready issues

```bash
# Verify issue is marked as ready
curl -s "http://localhost:8080/api/github/triage/status?id=$TRIAGE_ID" | jq .ready_to_implement

# Should return: true
```

### Issue: Code generation timeout

```bash
# Check server logs
tail -f asf-web.log

# Increase timeout if needed
export TIMEOUT=60s
```

---

## Summary

The **Automatic Triage Factory** demonstrates a complete real-world workflow:

```
GitHub Issue
    ↓ (automatic triage)
Labeled & Categorized
    ↓ (if ready)
Spec Generated
    ↓ (code generation)
Production Code with Tests
    ↓ (quality scored)
Pull Request Created
    ↓ (ready for review)
Human Review & Merge
```

This is the Warp Cloud Factory approach, but implemented with our 7-layer architecture, AST-based analysis, and real GitHub integration.

**Next:** Try it with your own GitHub repository!
