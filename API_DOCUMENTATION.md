# Factory API Documentation

## Overview
Complete REST API for the Agentic Software Factory with real-time streaming and comprehensive management endpoints.

## Base URL
```
http://localhost:8080
```

---

## 🔗 Integration Endpoints

### GitHub Webhooks
**POST** `/webhooks/github`

Receives GitHub webhook events and auto-triggers workflows.

**Headers:**
```
X-GitHub-Event: issues|pull_request|pull_request_review|push
X-Hub-Signature-256: sha256=<hmac>
Content-Type: application/json
```

**Example:**
```bash
curl -X POST http://localhost:8080/webhooks/github \
  -H "X-GitHub-Event: issues" \
  -H "X-Hub-Signature-256: sha256=..." \
  -d @payload.json
```

**Response (202 Accepted):**
```json
{
  "status": "accepted",
  "workflow_id": "gh-issue-triage-123456789",
  "type": "issue-triage"
}
```

**Triggered Workflows:**
- `issue-triage` - On issue opened/reopened
- `pr-review` - On PR opened/synchronized
- `pr-feedback` - On review submitted
- `push-analyze` - On push events

---

### Real-Time Stream (Server-Sent Events)
**GET** `/api/stream`

Streams real-time workflow updates via Server-Sent Events.

**Headers:**
```
Accept: text/event-stream
```

**Example:**
```bash
curl -N http://localhost:8080/api/stream
```

**Message Types:**
```json
{
  "type": "workflow-started|workflow-update|workflow-completed|metrics-update",
  "workflow_id": "gh-issue-triage-123456789",
  "status": "running|success|failed",
  "progress": 0-100,
  "data": {},
  "timestamp": "2026-11-17T15:00:00Z"
}
```

**Example Updates:**
```
data: {"type":"workflow-started","workflow_id":"wf-1","status":"running","progress":0}
data: {"type":"workflow-update","workflow_id":"wf-1","status":"running","progress":50}
data: {"type":"workflow-completed","workflow_id":"wf-1","status":"success","progress":100}
```

---

## 📊 Metrics & Monitoring

### Get Detailed Metrics
**GET** `/api/metrics/detailed`

Returns comprehensive system metrics.

**Example:**
```bash
curl http://localhost:8080/api/metrics/detailed
```

**Response:**
```json
{
  "metrics": {
    "workflows": {
      "started": 10,
      "completed": 8,
      "failed": 2,
      "success_rate": 80.0,
      "avg_duration_ms": 2500,
      "avg_code_score": 85
    },
    "bugs": {
      "detected": 15,
      "fixed": 12
    },
    "github": {
      "issues_analyzed": 25,
      "prs_reviewed": 10
    },
    "api": {
      "requests_total": 500,
      "requests_error": 5,
      "error_rate": 1.0,
      "avg_response_time_ms": 125
    },
    "database": {
      "queries_executed": 1000,
      "query_errors": 2,
      "avg_query_time_ms": 50
    }
  },
  "queue": {
    "total": 25,
    "pending": 3,
    "running": 2,
    "success": 15,
    "failed": 5,
    "capacity": 5
  },
  "clients": 3
}
```

---

## 🎯 Job Management

### Get Job Status
**GET** `/api/jobs/status?id=<job-id>`

Returns detailed job status and results.

**Example:**
```bash
curl http://localhost:8080/api/jobs/status?id=gh-issue-triage-123456789
```

**Response:**
```json
{
  "id": "gh-issue-triage-123456789",
  "type": "issue-triage",
  "status": "success",
  "input": {
    "issue_number": 42,
    "repository": "owner/repo"
  },
  "output": {
    "priority": "high",
    "category": "feature",
    "action": "review"
  },
  "error": "",
  "created_at": "2026-11-17T15:00:00Z",
  "started_at": "2026-11-17T15:00:05Z",
  "completed_at": "2026-11-17T15:00:10Z",
  "retries": 0
}
```

### List All Jobs
**GET** `/api/jobs/list`

Returns all jobs grouped by status.

**Example:**
```bash
curl http://localhost:8080/api/jobs/list
```

**Response:**
```json
{
  "pending": [
    {"id": "job-1", "type": "issue-triage", "status": "pending"}
  ],
  "running": [
    {"id": "job-2", "type": "pr-review", "status": "running"}
  ],
  "success": [
    {"id": "job-3", "type": "issue-triage", "status": "success"}
  ],
  "failed": []
}
```

---

## 🔐 Security & Audit

### Get Audit Log
**GET** `/api/audit/log`

Returns recent audit events for compliance tracking.

**Example:**
```bash
curl http://localhost:8080/api/audit/log
```

**Response:**
```json
{
  "events": [
    {
      "timestamp": "2026-11-17T15:00:00Z",
      "event_type": "WEBHOOK",
      "actor": "192.168.1.1",
      "resource": "owner/repo",
      "action": "post",
      "result": "success",
      "details": "Workflow gh-issue-triage-123456789 submitted"
    },
    {
      "timestamp": "2026-11-17T14:59:00Z",
      "event_type": "ACCESS",
      "actor": "user@example.com",
      "resource": "api",
      "action": "query",
      "result": "success",
      "details": "Retrieved metrics"
    }
  ],
  "count": 2
}
```

**Event Types:**
- `WEBHOOK` - GitHub webhook processed
- `ACCESS` - API endpoint accessed
- `SECURITY` - Security event (rate limit, invalid signature)
- `ERROR` - System error
- `RATE_LIMIT` - Rate limit exceeded
- `AUDIT` - Audit trail

---

## 📈 Dashboard APIs

### Get Dashboard
**GET** `/api/dashboard`

Returns metrics for the dashboard.

**Response:**
```json
{
  "triage": {
    "total_analyzed": 127,
    "ready_count": 89,
    "not_ready_count": 38,
    "by_priority": {
      "critical": 12,
      "high": 34,
      "medium": 56,
      "low": 25
    },
    "by_category": {
      "feature": 45,
      "bug": 32,
      "documentation": 22,
      "refactor": 28
    }
  },
  "workflow_summary": {
    "total": 100,
    "success": 85,
    "failed": 15
  },
  "active_workflows": 3
}
```

---

## 🗄️ Database APIs

### Get Database Stats
**GET** `/api/db/stats`

Returns historical statistics from database.

**Response:**
```json
{
  "workflow": {
    "total": 1000,
    "success": 850,
    "failed": 150,
    "running": 0,
    "avg_duration": 2500
  },
  "bugs": {
    "total": 350,
    "fixed": 300,
    "critical": 25
  },
  "triage": {
    "total": 500,
    "by_priority": {
      "critical": 50,
      "high": 150,
      "medium": 200,
      "low": 100
    }
  },
  "avg_code_score": 82.5
}
```

### Get Workflow History
**GET** `/api/db/workflows?limit=20&offset=0`

Returns paginated workflow history.

**Response:**
```json
{
  "workflows": [
    {
      "id": "wf-1",
      "type": "spec-to-code",
      "status": "success",
      "duration_ms": 5000,
      "created_at": "2026-11-17T15:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

---

## ⚙️ Configuration

### Environment Variables
```bash
# GitHub Webhook
GITHUB_WEBHOOK_SECRET=your-secret-key

# Queue
QUEUE_MAX_WORKERS=5
QUEUE_MAX_RETRIES=3

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=asf

# Ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=mistral
```

---

## 🛡️ Rate Limiting

**Default:** 100 requests per second per IP

**When Exceeded:** Returns `429 Too Many Requests`

**Retry-After:** Included in response headers

---

## 📝 Error Responses

### 400 Bad Request
```json
{
  "error": "Missing required parameter: id"
}
```

### 401 Unauthorized
```json
{
  "error": "Invalid webhook signature"
}
```

### 429 Too Many Requests
```json
{
  "error": "Rate limit exceeded"
}
```

### 500 Internal Server Error
```json
{
  "error": "Failed to process workflow"
}
```

---

## 🔄 Workflow State Transitions

```
pending → running → success/failed → archived

Retries:
running → failed → pending (retry) → running → success/failed
```

---

## 📊 Metrics Collection

**Automatically Tracked:**
- Workflow duration
- Success/failure rates
- Bug detection counts
- Code quality scores
- API response times
- Database query performance
- Rate limit violations
- Security events

---

## 🧪 Testing the Integration

### Test Webhook
```bash
curl -X POST http://localhost:8080/webhooks/github \
  -H "X-GitHub-Event: issues" \
  -H "X-Hub-Signature-256: sha256=$(echo -n '{...}' | openssl dgst -sha256 -hmac 'secret')" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "issue": {"number": 1, "title": "Test", "body": "Description"},
    "repository": {"full_name": "owner/repo", "owner": {"login": "owner"}},
    "sender": {"login": "user"}
  }'
```

### Test Real-Time Stream
```bash
curl -N http://localhost:8080/api/stream
```

### Test Metrics
```bash
curl http://localhost:8080/api/metrics/detailed | jq
```

---

## 🚀 Deployment Checklist

- [ ] Set `GITHUB_WEBHOOK_SECRET` environment variable
- [ ] Configure queue workers and retries
- [ ] Set up database (optional)
- [ ] Configure rate limiting per environment
- [ ] Enable HTTPS for webhook endpoint
- [ ] Set up audit log monitoring
- [ ] Configure realtime streaming for dashboard
- [ ] Load test with concurrent workflows
- [ ] Set up monitoring/alerting
