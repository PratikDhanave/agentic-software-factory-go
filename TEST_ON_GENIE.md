# 🧪 TEST TRIAGE ON GENIE REPO

## Quick Test Instructions

### 1. Start Services

**Terminal 1: Backend**
```bash
cd ~/go/src/github.com/PratikDhanave/agentic-software-factory-go
export GITHUB_TOKEN=ghp_your_real_token_here
./asf-web localhost:8080
```

**Terminal 2: Frontend**
```bash
cd ~/go/src/github.com/PratikDhanave/agentic-software-factory-go/web-ui
npm run dev
```

### 2. Open Dashboard
```
http://localhost:3000
```

### 3. Create Test Issue in Genie Repo

Go to: https://github.com/PratikDhanave/genie/issues

Click "New Issue" and create:

```
Title: Add caching layer for performance

Description:
# Caching Layer for Performance

## Acceptance Criteria
- Implement LRU cache
- Cache hit/miss metrics
- Thread-safe operations
- 80%+ test coverage

## Context
Current system has slow lookups. Need in-memory cache to improve response times.

## Steps to Implement
1. Design cache interface
2. Implement LRU eviction
3. Add metrics collection
4. Write comprehensive tests
```

### 4. Get Issue Number

```
URL will be: github.com/PratikDhanave/genie/issues/N
Note the issue number N (e.g., 1, 2, etc.)
```

### 5. Test in UI

Fill the form:
```
GitHub Owner:  PratikDhanave
Repository:    genie
Issue Number:  N (from step 4)
Token:         ✓ Auto-loaded
```

Click: **Run Triage**

### 6. See Results

Expected output:
```
Category:     feature
Priority:     medium
Ready:        ✅ Yes
Labels:       ready-to-implement, category-feature, priority-medium
Analysis:     Has clear acceptance criteria, sufficient context, and no blockers
Spec:         [YAML spec generated]
```

---

## 📊 What Gets Posted to GitHub

The issue will automatically receive:

**Comment:**
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

**Labels:**
- `ready-to-implement`
- `category-feature`
- `priority-medium`

---

## 🎯 Next Steps After Triage

### Option 1: Generate Code
```
1. Click "Copy Spec"
2. Scroll to LaunchPad (if full UI)
3. Paste spec into Code Generator
4. Generate code
```

### Option 2: Create PR Manually
```
1. Note the issue number
2. Use spec to guide implementation
3. Create PR linked to issue
```

### Option 3: Override Decision
```
1. Go to GitHub issue
2. Add comment with clarifications
3. Re-run triage
```

---

## 🔍 Test Scenarios

### Scenario 1: Ready Issue ✅
**What to try:**
- Create issue with clear acceptance criteria
- Expected: `ready-to-implement` status
- Verify: Labels applied to GitHub

### Scenario 2: Needs Info ⏳
**What to try:**
```
Title: Improve performance
Body: Things are slow
```
- Expected: `needs-info` status
- Verify: Different labels applied

### Scenario 3: Blocked Issue 🚫
**What to try:**
```
Title: Fix bug (blocked by PR #123)
Body: Cannot start until PR #123 is merged
```
- Expected: `wait-to-implement` status
- Verify: Blocked label applied

---

## 🐛 Troubleshooting

### Token Error
```
Solution: Get fresh token from github.com/settings/tokens
Scopes: repo, write:discussion
```

### Issue Not Found
```
Check:
- Correct repository name
- Correct issue number
- Issue is not archived
```

### Labels Not Applied
```
Check:
- Token has write:discussion scope
- Issue is not locked
- No permission issues
```

---

## 📈 Success Criteria

✅ **Passed if:**
1. Issue is fetched
2. Analysis completes (< 10 seconds)
3. Comment posted to issue
4. Labels applied to issue
5. Results display in UI
6. Spec is generated
7. "Copy Spec" button works

---

## 🎉 Expected Output

**On Dashboard:**
```
Category:     feature
Priority:     medium
Ready:        ✅ Yes
Labels:       ready-to-implement, category-feature, priority-medium
Analysis:     Clear acceptance criteria found
Spec:         [Collapsible YAML]
Actions:      [View on GitHub] [Copy Spec]
```

**On GitHub:**
- Comment posted ✓
- Labels added ✓
- Issue linked to spec ✓

---

## 🚀 Ready to Test?

1. Start both servers (steps 1-2 above)
2. Create issue in genie repo
3. Fill triage form
4. Click "Run Triage"
5. Watch the magic happen! ✨

Let me know the results!
