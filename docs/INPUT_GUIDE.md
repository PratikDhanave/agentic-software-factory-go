# Input Guide - What You Need to Provide

This guide explains what input you need to provide to trigger the end-to-end workflow.

---

## 1. GitHub Webhook (Primary Input)

The main way to trigger the system is by having GitHub send a webhook.

### Setup Required

```bash
# 1. Go to your GitHub repository
# 2. Settings → Webhooks → Add webhook

# Configure:
Payload URL: http://your-server:8080/webhook
Content type: application/json
Events: Select individual events:
  ✓ Issues
  ✓ Pull requests
  ✓ Pushes
Secret: (leave empty for testing)
```

### What GitHub Sends

When you trigger an event (open issue, create PR, push code), GitHub automatically sends:

```json
POST /webhook
Content-Type: application/json
X-GitHub-Event: issues
X-GitHub-Delivery: 12345-67890-abcdef

{
  "action": "opened",
  "issue": {
    "number": 42,
    "title": "Fix authentication bug",
    "body": "Users cannot login with OAuth",
    "user": {
      "login": "developer"
    }
  },
  "repository": {
    "name": "myrepo",
    "full_name": "myorg/myrepo"
  }
}
```

**You don't need to create this JSON manually** - GitHub sends it automatically.

---

## 2. Manual Testing (Simulate Webhook)

To test without GitHub, simulate the webhook locally:

### Using curl

```bash
# Test webhook endpoint
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: issues" \
  -d '{
    "action": "opened",
    "issue": {
      "number": 42,
      "title": "Fix authentication bug",
      "body": "Users cannot login with OAuth"
    },
    "repository": {
      "name": "myrepo",
      "full_name": "myorg/myrepo"
    }
  }'
```

### Expected Response

```json
HTTP/1.1 202 Accepted
Content-Type: application/json

{
  "status": "accepted",
  "job_id": "gh-webhook-1734594600000000000",
  "job_type": "issue-triage"
}
```

---

## 3. Query Job Status (Secondary Input)

After a job is created, you can query its status:

### Using curl

```bash
# Query job status
curl "http://localhost:8080/api/jobs/status?id=gh-webhook-1734594600000000000"
```

### Expected Response

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "gh-webhook-1734594600000000000",
  "type": "issue-triage",
  "status": "created",
  "priority": 5,
  "input": {
    "action": "opened",
    "issue": 42,
    "repo": "myorg/myrepo",
    "title": "Fix authentication bug",
    "body": "Users cannot login with OAuth"
  },
  "created_at": "2026-11-18T10:30:01Z",
  "updated_at": "2026-11-18T10:30:01Z"
}
```

---

## 4. Webhook Events You Can Trigger

### Issues Event

```bash
# User creates/updates/closes an issue
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: issues" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "issue": {
      "number": 1,
      "title": "Bug: Login broken",
      "body": "OAuth not working"
    },
    "repository": {
      "full_name": "myorg/myrepo"
    }
  }'
```

### Pull Request Event

```bash
# User creates/updates a PR
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: pull_request" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "pull_request": {
      "number": 123,
      "title": "Feature: Add OAuth",
      "body": "Implements OAuth login"
    },
    "repository": {
      "full_name": "myorg/myrepo"
    }
  }'
```

### Push Event

```bash
# User pushes code
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: push" \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "refs/heads/main",
    "commits": [
      {
        "message": "Fix: authentication flow",
        "url": "https://github.com/myorg/myrepo/commit/abc123"
      }
    ],
    "repository": {
      "full_name": "myorg/myrepo"
    }
  }'
```

---

## 5. Full Example: Create and Query

### Step 1: Create a job via webhook

```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: issues" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "issue": {
      "number": 42,
      "title": "Authentication failing",
      "body": "OAuth tokens not working"
    },
    "repository": {
      "full_name": "myorg/myrepo"
    }
  }'

# Response:
# {
#   "status": "accepted",
#   "job_id": "gh-webhook-1734594600000000000",
#   "job_type": "issue-triage"
# }
```

### Step 2: Query job status

```bash
# Copy the job_id from response above
JOB_ID="gh-webhook-1734594600000000000"

curl "http://localhost:8080/api/jobs/status?id=$JOB_ID"

# Response:
# {
#   "id": "gh-webhook-1734594600000000000",
#   "type": "issue-triage",
#   "status": "created",
#   ...
# }
```

### Step 3: Check the logs

```bash
# Terminal where server is running should show:
# 2026-11-18 10:30:01.001 [INFO] Webhook received | event_type=issues action=opened
# 2026-11-18 10:30:01.002 [INFO] use case started | operation=submit-job ...
# 2026-11-18 10:30:01.018 [INFO] use case completed in 17ms | operation=submit-job
```

---

## 6. Input Parameters Explained

### Webhook Headers Required

```
X-GitHub-Event: issues              ← Event type (issues, pull_request, push)
Content-Type: application/json      ← JSON format
```

### JSON Payload Structure

#### For Issues

```json
{
  "action": "opened",                    // opened, closed, edited, reopened
  "issue": {
    "number": 42,                        // Issue number
    "title": "Bug title",                // Issue title
    "body": "Issue description",         // Issue description
    "user": {
      "login": "developer"               // Who created it
    }
  },
  "repository": {
    "full_name": "owner/repo"            // Full repo name
  }
}
```

#### For Pull Requests

```json
{
  "action": "opened",                    // opened, synchronize, closed
  "pull_request": {
    "number": 123,                       // PR number
    "title": "Feature title",            // PR title
    "body": "Description"                // PR description
  },
  "repository": {
    "full_name": "owner/repo"            // Full repo name
  }
}
```

#### For Push

```json
{
  "ref": "refs/heads/main",              // Branch ref
  "commits": [
    {
      "message": "Commit message",       // Commit message
      "url": "https://..."               // Commit URL
    }
  ],
  "repository": {
    "full_name": "owner/repo"            // Full repo name
  }
}
```

---

## 7. Query Parameters for Status Endpoint

### Get Job Status

```bash
GET /api/jobs/status?id=JOB_ID
```

**Parameters:**
- `id` (required): The job ID returned from webhook response

**Example:**
```bash
curl "http://localhost:8080/api/jobs/status?id=gh-webhook-1734594600000000000"
```

---

## 8. Environment Configuration

Before running, set environment variables:

```bash
# Log level (optional, defaults to "info")
export LOG_LEVEL=debug

# PostgreSQL connection (optional for production)
export DATABASE_URL=postgres://user:pass@localhost:5432/agentic_factory

# Server configuration (optional)
export PORT=8080
export HOST=localhost
```

---

## 9. What Happens Internally

When you provide input (webhook), the system:

```
Input (Webhook)
  ↓
1. HTTP Handler receives webhook
   - Parses JSON payload
   - Validates headers
   
2. Webhook Parser
   - Extracts event data
   - Determines workflow type
   
3. Job Factory
   - Creates Job entity
   - Assigns unique ID
   - Sets priority
   
4. Use Case Observer
   - Wraps execution
   - Starts tracing
   - Logs operation start
   
5. Business Logic (SubmitJobUseCase)
   - Validates job
   - Saves to database
   - Publishes events
   
6. Observability
   - Logs completion
   - Records duration
   - Ends span
   
7. HTTP Response
   - Returns 202 Accepted
   - Includes job ID

Output (202 Response with Job ID)
```

---

## 10. Testing Checklist

✅ **Before Testing**
```bash
# Start server
go run ./cmd/asf-web/main.go localhost:8080

# In another terminal, verify it's running
curl http://localhost:8080/health  # (if endpoint exists)
```

✅ **Test Webhook**
```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: issues" \
  -H "Content-Type: application/json" \
  -d '{"action":"opened","issue":{"number":1,"title":"Test"},"repository":{"full_name":"test/repo"}}'
```

✅ **Check Response**
```
Expected: HTTP 202 Accepted
Expected body: {"status":"accepted","job_id":"...","job_type":"issue-triage"}
```

✅ **Check Logs**
```
Expected: Multiple [INFO] log entries
Expected: [TRACE] span output showing duration
```

✅ **Query Job**
```bash
curl "http://localhost:8080/api/jobs/status?id=YOUR_JOB_ID"
```

Expected: Job details returned

---

## 11. Common Input Scenarios

### Scenario 1: Simple Issue Report
```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: issues" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "issue": {
      "number": 1,
      "title": "Quick bug",
      "body": "Found a bug"
    },
    "repository": {
      "full_name": "user/project"
    }
  }'
```

### Scenario 2: Detailed Feature Request
```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: issues" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "issue": {
      "number": 42,
      "title": "Feature: Dark mode support",
      "body": "Users want dark mode in the UI.\n\n## Requirements\n- Support system theme\n- Add toggle switch\n- Persist preference"
    },
    "repository": {
      "full_name": "company/app"
    }
  }'
```

### Scenario 3: PR Review
```bash
curl -X POST http://localhost:8080/webhook \
  -H "X-GitHub-Event: pull_request" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "pull_request": {
      "number": 123,
      "title": "Fix: Handle null references",
      "body": "Fixes #456 by adding null checks"
    },
    "repository": {
      "full_name": "company/app"
    }
  }'
```

---

## 12. Troubleshooting

### No Response from Webhook

**Problem**: Server doesn't respond to webhook

**Solution**:
```bash
# Check server is running
ps aux | grep asf-web

# Check port is open
netstat -tlnp | grep 8080

# Try direct call
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Invalid Event Type

**Problem**: "unknown event type" error

**Solution**:
```bash
# Make sure header is set
-H "X-GitHub-Event: issues"  # Must be one of: issues, pull_request, push

# Check JSON is valid
echo '{"action":"opened","issue":{"number":1},"repository":{"full_name":"a/b"}}' | jq .
```

### Job Not Saving

**Problem**: Job created but not persisting

**Solution**:
```bash
# Check database connection
echo "SELECT 1;" | psql $DATABASE_URL

# Check logs for database errors
# Server logs should show [ERROR] if database fails
```

---

## Summary

**You provide:**
- ✅ GitHub webhook events (automatically sent by GitHub)
- ✅ Or manually craft webhook JSON for testing
- ✅ Query parameters to retrieve job status

**System provides:**
- ✅ Automatic logging at every step
- ✅ Automatic tracing with duration
- ✅ Job persistence to database
- ✅ Event publishing
- ✅ HTTP responses with job IDs

**To test**: Just send one curl command with the webhook JSON, and the entire end-to-end flow executes automatically!
