# Using Automatic Triage from the Web UI

This guide shows how to use the automatic triage factory directly from the web dashboard.

---

## Quick Start (2 minutes)

### Step 1: Start the Backend

```bash
./asf-web localhost:8080
```

Expected output:
```
ASF Web Server starting on localhost:8080
7-Layer Agent Endpoints:
  POST /api/agents/run              - Execute 7-layer agent (SSE)
  GET  /api/agents/status           - Get result
  GET  /api/agents/metrics          - Get metrics
GitHub Integration Endpoints:
  GET  /api/github/review           - Review PR (SSE streaming)
       ?owner=X&repo=Y&pr=N&token=Z
GitHub Automatic Triage Endpoints:
  GET  /api/github/triage           - Triage issue (SSE streaming)
       ?owner=X&repo=Y&issue=N&token=Z
  GET  /api/github/triage/status    - Get triage result
```

### Step 2: Start the Frontend

```bash
cd web-ui
npm run dev
```

Expected output:
```
  ▲ Next.js 15.0.0
  - Local:        http://localhost:3000
  - Environments: .env.local

 ✓ Ready in 2.3s
```

### Step 3: Open Dashboard

Visit **http://localhost:3000**

You should see:
- Agentic Software Factory header
- LaunchPad section with feature cards
- **NEW: Automatic GitHub Issue Triage section** (below LaunchPad)

---

## UI Components Explained

### The Triage Panel

The **Automatic GitHub Issue Triage** section is located on the homepage below the LaunchPad.

```
┌─────────────────────────────────────────────────────────────┐
│ ⚡ Automatic Triage                                          │
│ Analyze GitHub issues, categorize them, and assess          │
│ readiness for implementation                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ GitHub Owner: [PratikDhanave             ]                   │
│ Repository:   [agentic-software-factory-go]                 │
│ Issue Number: [1                         ]                   │
│                                                              │
│ GitHub Personal Access Token:                                │
│ [•••••••••••••••••••••] [Show] [Load] [Save]                │
│                                                              │
│ [🚀 Run Triage]                                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Input Fields

#### 1. GitHub Owner
**What**: Your GitHub username or organization name
**Example**: `PratikDhanave`
**Where to find**: Your GitHub profile URL: github.com/YOUR_USERNAME

#### 2. Repository
**What**: The repository name (without .git)
**Example**: `agentic-software-factory-go`
**Where to find**: GitHub URL: github.com/owner/REPO_NAME

#### 3. Issue Number
**What**: The issue ID (just the number)
**Example**: `1`, `42`, `101`
**Where to find**: GitHub issue URL: github.com/owner/repo/issues/123

#### 4. GitHub Personal Access Token
**What**: Your authentication token with GitHub API access
**Scopes needed**: `repo`, `write:discussion`
**Where to get**:
1. Go to https://github.com/settings/tokens
2. Click "Generate new token" (classic)
3. Name it: "Agentic Factory"
4. Select scopes:
   - ✅ repo (Full control)
   - ✅ write:discussion (Pull request reviews)
5. Click "Generate token"
6. Copy the token (you won't see it again!)

**Button Options**:
- **Show/Hide**: Toggle password visibility
- **Load**: Load previously saved token from browser storage
- **Save**: Save token locally (encrypted in browser)

---

## Using the Triage Panel

### Example Workflow

#### Step 1: Fill in Repository Details

```
GitHub Owner:    PratikDhanave
Repository:      agentic-software-factory-go
Issue Number:    1
```

#### Step 2: Add GitHub Token

**Option A: Paste Token Directly**
```
1. Copy your personal access token
2. Paste into "GitHub Personal Access Token" field
3. Click "Run Triage"
```

**Option B: Load Previously Saved Token**
```
1. Click "Load" button
2. Token auto-fills if saved before
3. Click "Run Triage"
```

**Option C: Save for Future Use**
```
1. Paste token once
2. Click "Save" button
3. Next time: Click "Load" to auto-fill
```

#### Step 3: Click "Run Triage"

The panel will show:

```
Processing...
████████░░ 55%  (animated progress bar)
```

**Progress stages** (you'll see real-time updates):
- 🔗 10%  - Connecting to GitHub
- 📥 25%  - Fetching issue details
- 🔍 45%  - Analyzing issue content
- 📝 65%  - Generating implementation spec
- 🏷️  80%  - Applying labels to issue
- 💬 90%  - Posting triage comment
- ✅ 100% - Complete

---

## Results Panel

After triage completes, you'll see the **Triage Result** section:

### Result Fields

#### ✅ Category
Detected issue type:
- **feature** - New functionality request
- **bug** - Defect or problem
- **documentation** - Docs/wiki updates
- **refactor** - Code improvement

#### ✅ Priority
Severity level:
- **critical** - Urgent, blocks work
- **high** - Important, needed soon
- **medium** - Standard priority
- **low** - Nice-to-have, non-urgent

#### ✅ Ready to Implement
Status assessment:
- ✅ **Yes** - Clear acceptance criteria, ready to code
- ⏳ **No** - Needs clarification or has blockers

#### ✅ Labels to Apply
GitHub labels that will be added:
- `ready-to-implement` - Passes readiness check
- `priority-{level}` - Priority indicator
- `category-{type}` - Issue category
- `needs-info` - Needs clarification (if not ready)

Example:
```
ready-to-implement, priority-high, category-feature
```

#### ✅ Analysis
Human-readable explanation:
- Why it's ready (or what it needs)
- What acceptance criteria found
- Any blockers identified

Example:
```
"Has clear acceptance criteria, sufficient context, and no blockers"
```

#### 📋 Generated Spec (if Ready)
Collapsible YAML specification for code generation:
- Shows acceptance criteria
- Lists requirements
- Includes constraints
- Click "Show YAML Spec" to expand

---

## Taking Action on Results

### If Issue is Ready to Implement ✅

You have two options:

#### Option 1: Copy Spec and Generate Code

```
1. Click "Copy Spec" button
2. Go to code generation section (different part of UI)
3. Paste the spec
4. Click to generate code
5. Review and create PR
```

#### Option 2: View on GitHub

```
1. Click "View on GitHub" button
2. Browser opens the GitHub issue
3. You'll see:
   - Original issue content
   - Automatic triage comment from the factory
   - Applied labels
4. Can manually start implementation
```

### If Issue Needs More Info ⏳

The triage result shows:
- What's missing (acceptance criteria, context, etc.)
- Labels applied: `needs-info`, `wait-to-implement`
- You need to:
  1. Comment on GitHub issue with clarifications
  2. Add more details/acceptance criteria
  3. Re-run triage when ready

---

## Full Workflow: From Issue to PR

### Scenario: Building a Cache Service

#### Step 1: Create GitHub Issue
```
Title: "Add thread-safe cache service"

Body:
# Cache Service

## Acceptance Criteria
- Thread-safe operations
- Lock/Unlock with defer pattern
- Get, Set, Delete methods
- Full test coverage

## Context
Needed for request memoization
```

#### Step 2: Run Triage in UI
```
1. Go to http://localhost:3000
2. Scroll to "Automatic GitHub Issue Triage"
3. Fill in:
   - Owner: your_username
   - Repo: your_repo
   - Issue: 1 (or whatever number)
   - Token: your_token
4. Click "Run Triage"
5. Wait for completion (5-10 seconds)
```

#### Step 3: Review Triage Result
```
Category: feature
Priority: medium
Ready to Implement: ✅ Yes

Analysis: "Has clear acceptance criteria, sufficient context, and no blockers"

Labels to Apply:
- ready-to-implement
- priority-medium
- category-feature
```

#### Step 4: Copy Generated Spec
```
1. Click "Copy Spec" button
2. Spec copied to clipboard
3. Scroll up to "LaunchPad"
```

#### Step 5: Generate Code
```
1. Find "Code Generator" card in LaunchPad
2. Or go to /api/agents/run endpoint
3. Paste the spec
4. Click to generate
5. Wait for code generation (10-15 seconds)
```

#### Step 6: Review Generated Code
```
Shows:
- Generated service code
- Test code
- Quality score (e.g., 92%)
- Any issues detected
```

#### Step 7: Create PR
```
1. If code looks good, create branch locally:
   git checkout -b cache-service
2. Copy generated code to file
3. Commit and push
4. Create pull request on GitHub
5. Link to original issue #1
```

#### Step 8: GitHub Workflow Complete
```
✅ Issue is triaged
✅ Code is generated and reviewed
✅ PR is ready for team review
✅ Labeled for easy tracking
```

---

## Tips & Tricks

### 1. Save Your Token
```
First time:
1. Paste token
2. Click "Save"
3. Stored locally in browser

Later:
1. Just click "Load"
2. Token auto-fills
3. No need to paste again
```

### 2. Batch Multiple Issues
```
1. Run triage on issue #1
2. Note results, click "View on GitHub"
3. Back to dashboard
4. Change issue number to #2
5. Click "Run Triage" again
6. Results appear below (old results scroll away)
```

### 3. Check Results Offline
```
After triage completes:
1. Open browser developer tools (F12)
2. Go to Application → Local Storage
3. See saved triage results as JSON
4. Can reference later
```

### 4. Share Results
```
Generated spec can be shared:
1. Copy spec to clipboard
2. Paste in Slack, email, etc.
3. Anyone can regenerate code from it
4. Great for collaboration
```

---

## Troubleshooting

### Error: "HTTP 401"
**Problem**: GitHub token is invalid or expired
**Solution**:
1. Generate new token at https://github.com/settings/tokens
2. Make sure scopes include `repo` and `write:discussion`
3. Copy new token (don't paste the old one!)
4. Try again

### Error: "404: Issue not found"
**Problem**: Wrong issue number or repository
**Solution**:
1. Double-check GitHub owner name (case-sensitive!)
2. Verify repository name
3. Confirm issue number exists on GitHub
4. Try accessing directly: github.com/owner/repo/issues/123

### Progress Bar Stuck at 50%
**Problem**: Network issue or slow connection
**Solution**:
1. Check your internet connection
2. Wait a few more seconds
3. If still stuck, refresh page and try again
4. Check browser console (F12) for errors

### Token Not Saving
**Problem**: Browser local storage disabled
**Solution**:
1. Check if your browser allows local storage
2. Or manually copy-paste token each time
3. Consider using password manager instead

---

## API Behind the Scenes

The UI calls these backend endpoints:

### Triage Endpoint
```
GET /api/github/triage?owner=X&repo=Y&issue=N&token=Z

Returns: Server-Sent Events (streaming)
Progress: 0% → 100% in real-time
Final: Triage result as JSON
```

### Get Status
```
GET /api/github/triage/status?id=ABC123

Returns: Complete triage result including:
- Category, priority, readiness
- Labels to apply
- Generated spec (if ready)
- GitHub issue details
```

---

## Next Steps

After triaging from the UI:

### If Ready to Implement:
1. ✅ Copy generated spec
2. 📝 Use LaunchPad to generate code
3. 👀 Review generated code
4. 📤 Create pull request
5. 🔀 Merge after review

### If Needs Info:
1. ❓ Go to GitHub issue
2. 💬 Add clarifying comments
3. 📝 Add acceptance criteria
4. ↩️ Re-run triage from UI
5. ✅ Once ready, follow steps above

---

## Summary

**Quick Reference:**

| Action | Steps | Time |
|--------|-------|------|
| Run triage | Fill form + click button | 10-15 sec |
| Review results | Scroll down to results | 5 sec |
| Copy spec | Click "Copy Spec" button | 1 sec |
| Generate code | Paste in LaunchPad | 10-15 sec |
| Create PR | Use generated code | 2-5 min |

**Pro Tip**: The entire workflow (triage → code → PR) takes about **5-10 minutes** for typical issues!

Happy building! 🚀
