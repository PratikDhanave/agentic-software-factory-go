# 📋 PARAMETERS GUIDE

## UI Form Parameters

### Required Fields (3)

```
1. GitHub Owner
   - What: Your GitHub username or organization
   - Example: PratikDhanave
   - Required: YES
   - Format: text

2. Repository
   - What: Repository name (without .git)
   - Example: agentic-software-factory-go
   - Required: YES
   - Format: text

3. Issue Number
   - What: The issue ID from GitHub
   - Example: 1
   - Required: YES
   - Format: number
```

### Auto-Loaded Parameter

```
4. GitHub Token
   - What: Personal access token from GitHub
   - Example: ghp_ABC123...
   - Required: YES
   - Where: GITHUB_TOKEN environment variable
   - Format: auto-loaded
   - Visibility: Hidden (not shown in UI)
```

---

## API Query Parameters

### GET /api/github/triage

```
owner=VALUE
  - GitHub username or organization
  - Example: owner=PratikDhanave
  - Required: YES

repo=VALUE
  - Repository name
  - Example: repo=genie
  - Required: YES

issue=VALUE
  - Issue number
  - Example: issue=5
  - Required: YES

token=VALUE
  - GitHub personal access token
  - Example: token=ghp_ABC123...
  - Required: YES (if not loaded from env)
```

### Full URL Example

```
GET /api/github/triage?owner=PratikDhanave&repo=genie&issue=1&token=ghp_ABC123...
```

### cURL Example

```bash
curl "http://localhost:8080/api/github/triage?owner=PratikDhanave&repo=genie&issue=1&token=ghp_ABC123"
```

---

## Environment Variables

### Required

```
GITHUB_TOKEN
  - Description: GitHub personal access token
  - Required: YES
  - Set with: export GITHUB_TOKEN=ghp_...
  - Scopes: repo, write:discussion
  - How to get: github.com/settings/tokens
```

### Optional

```
PORT
  - Description: Backend server port (default: 8080)
  - Default: 8080
  - Example: export PORT=3000

HOST
  - Description: Backend server host (default: localhost)
  - Default: localhost
  - Example: export HOST=0.0.0.0
```

---

## API Response Parameters

### Success Response (SSE Stream)

```json
{
  "status": "connecting",
  "progress": 10,
  "message": "Connecting to GitHub"
}
```

```json
{
  "status": "fetching",
  "progress": 25,
  "message": "Fetching issue details"
}
```

```json
{
  "status": "analyzing",
  "progress": 45,
  "message": "Analyzing issue content"
}
```

```json
{
  "status": "spec-generation",
  "progress": 65,
  "message": "Generating implementation spec"
}
```

```json
{
  "status": "labeling",
  "progress": 80,
  "message": "Applying labels to issue"
}
```

```json
{
  "status": "commenting",
  "progress": 90,
  "message": "Posting triage comment to issue"
}
```

```json
{
  "status": "complete",
  "progress": 100,
  "message": "Triage complete: feature (medium)"
}
```

---

## Response Body Parameters

### Triage Result Object

```json
{
  "id": "abc123-def456",
  "issue": 1,
  "repo": "PratikDhanave/genie",
  "title": "Add caching layer",
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

### Parameter Descriptions

```
id
  - Type: string
  - What: Unique triage result ID
  - Example: abc123-def456

issue
  - Type: number
  - What: Issue number from GitHub
  - Example: 1

repo
  - Type: string
  - What: Full repository path
  - Example: PratikDhanave/genie

title
  - Type: string
  - What: Issue title from GitHub
  - Example: Add caching layer

category
  - Type: string (enum)
  - What: Detected issue type
  - Values: feature, bug, refactor, documentation

priority
  - Type: string (enum)
  - What: Detected priority level
  - Values: critical, high, medium, low

ready_to_implement
  - Type: boolean
  - What: Is issue ready to code?
  - Values: true, false

reason
  - Type: string
  - What: Explanation of readiness
  - Example: Has clear acceptance criteria...

labels_to_apply
  - Type: array of strings
  - What: Labels to add to GitHub issue
  - Example: [ready-to-implement, category-feature]

generated_spec
  - Type: string (YAML)
  - What: Generated implementation specification
  - Used for: Code generation

created_at
  - Type: string (ISO 8601)
  - What: When triage was completed
  - Example: 2026-11-16T12:00:00Z
```

---

## Status Values (Progress)

```
Status         Progress  Meaning
─────────────────────────────────
connecting      10%      Connecting to GitHub API
fetching        25%      Fetching issue details
analyzing       45%      Analyzing content
spec-generation 65%      Generating YAML spec
labeling        80%      Applying labels
commenting      90%      Posting comment
complete       100%      Done!
error            0%      Error occurred
```

---

## Category Values

```
feature
  - New functionality request
  - Detected keywords: create, add, implement, new

bug
  - Defect or problem
  - Detected keywords: bug, error, crash, fix, broken

refactor
  - Code improvement
  - Detected keywords: refactor, improve, clean, optimize

documentation
  - Docs/wiki updates
  - Detected keywords: doc, documentation, readme, wiki
```

---

## Priority Values

```
critical
  - Urgent, blocks work
  - Detected keywords: critical, urgent, blocker, blocking

high
  - Important, needed soon
  - Detected keywords: high priority, important, asap

medium
  - Standard priority (default)
  - No priority keywords found

low
  - Nice-to-have, non-urgent
  - Detected keywords: low priority, nice-to-have, later
```

---

## Label Values

```
ready-to-implement
  - Applied when: Issue passes readiness check
  - Removed when: Issue needs more info

category-feature
  - Applied when: Category detected as feature
  - Variants: category-bug, category-refactor, category-documentation

priority-critical
  - Applied when: Priority detected as critical
  - Variants: priority-high, priority-medium, priority-low

needs-info
  - Applied when: Issue not ready (needs clarification)

wait-to-implement
  - Applied when: Issue blocked or has dependencies
```

---

## Query Parameter Validation

```
owner
  - Must be: non-empty string
  - Max length: 255
  - Valid: alphanumeric, hyphens

repo
  - Must be: non-empty string
  - Max length: 255
  - Valid: alphanumeric, hyphens, underscores, dots

issue
  - Must be: positive integer
  - Range: 1 - 999999
  - Example: 1, 42, 100

token
  - Must be: valid GitHub token
  - Format: ghp_[alphanumeric]
  - Scopes: repo, write:discussion
```

---

## Error Parameters

```
HTTP 400 Bad Request
  - Cause: Missing required parameters
  - Response: {"error": "owner required"}

HTTP 401 Unauthorized
  - Cause: Invalid GitHub token
  - Response: {"error": "Invalid token"}

HTTP 404 Not Found
  - Cause: Issue doesn't exist
  - Response: {"error": "Issue not found"}

HTTP 500 Internal Server Error
  - Cause: Server error
  - Response: {"error": "..."}
```

---

## Example Workflows

### Via UI

```
1. Set environment: export GITHUB_TOKEN=ghp_...
2. Open: http://localhost:3000
3. Fill form:
   - Owner: PratikDhanave
   - Repository: genie
   - Issue Number: 1
   - Token: Auto-loaded
4. Click: Run Triage
5. Results appear with parameters
```

### Via API (cURL)

```bash
curl "http://localhost:8080/api/github/triage?owner=PratikDhanave&repo=genie&issue=1&token=ghp_ABC123"
```

### Via Environment

```bash
export GITHUB_TOKEN=ghp_ABC123
export PORT=8080
export HOST=localhost
./asf-web $HOST:$PORT
```

---

## Common Parameter Mistakes

```
❌ Missing owner
   Fix: Add GitHub username

❌ Invalid token
   Fix: Generate new token at github.com/settings/tokens

❌ Issue number as text
   Fix: Use number only (1, not "1" in URL)

❌ Full GitHub URL as repo
   Fix: Use repo name only (genie, not github.com/PratikDhanave/genie)

❌ Token with spaces
   Fix: Copy token exactly as shown
```

---

## Testing Parameters

### Test 1: Valid Parameters
```
owner: PratikDhanave
repo: genie
issue: 1
token: ghp_valid_token
Expected: Success
```

### Test 2: Invalid Token
```
owner: PratikDhanave
repo: genie
issue: 1
token: invalid
Expected: Error 401
```

### Test 3: Issue Not Found
```
owner: PratikDhanave
repo: genie
issue: 99999
token: ghp_valid_token
Expected: Error 404
```

---

## Summary

**UI Form:** 3 fields (owner, repo, issue) + auto-loaded token
**API Query:** 4 parameters (owner, repo, issue, token)
**Response:** JSON object with triage results
**Status:** SSE stream showing progress (0-100%)
