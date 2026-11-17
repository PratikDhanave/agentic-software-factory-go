# ASF Persistence & Dashboard Guide

## 🎯 Overview

The Agentic Software Factory now has two major features:

1. **Web UI Dashboard** - Real-time visualization of factory operations
2. **PostgreSQL Persistence** - Durable storage of all workflow results

## 📊 Web UI Dashboard

### Access
```
http://localhost:8080
```

### Pages

**Dashboard**
- Key metrics: Total issues, ready count, active workflows, success rate
- Issue distribution chart (feature/bug/docs/refactor)
- Workflow status trend (last 7 days)
- Recent workflows table with progress
- Priority issues list (critical/high)

**Workflows**
- Browse all workflow executions
- Create new workflows
- Filter by status
- View detailed results

**Triage**
- GitHub issue analysis results
- Priority breakdown
- Suggested actions
- Category distribution

**Reviews**
- Code review findings
- Approval status
- Score tracking
- Detailed recommendations

**Bugs**
- Detected bugs with severity
- Fix suggestions
- Confidence ratings
- Statistics by severity

**Docs**
- Generated documentation
- README, API docs, examples
- Contributing guidelines
- Searchable library

**Settings**
- System health indicators
- Database status
- Component connectivity
- Configuration options

## 💾 Database Persistence

### Setup Options

**Option 1: Docker (Recommended)**
```bash
docker compose up -d postgres
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=asf
./asf-web localhost:8080
```

**Option 2: Manual PostgreSQL**
```bash
# Install PostgreSQL
createdb -O postgres asf

# Set environment
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=asf
./asf-web localhost:8080
```

**Option 3: No Database (Development)**
```bash
./asf-web localhost:8080
# Factory works without persistence; data is lost on restart
```

### Data Stored

The database automatically persists:

**Workflow Executions**
- Workflow ID, type, status
- Input parameters (JSON)
- Output results (JSON)
- Start/end times
- Duration and error messages

**Code Quality Metrics**
- Lines of code
- Code score (0-100)
- Bugs found and fixed
- Test coverage percentage
- Complexity and maintainability scores

**Code Review Findings**
- File names
- Review scores
- Findings (JSON)
- Approval status
- Review summary

**Bug Detections**
- Bug severity (critical/high/medium/low)
- Bug type (null-pointer, race-condition, etc)
- Location in code
- Impact and fix suggestions
- Confidence rating
- Fix status

**Issue Triage Results**
- GitHub issue number and repository
- Priority and category
- Issue title and description
- Suggested action
- Analysis timestamp

## 🔄 Data Flow

```
Factory Workflow
    ↓
RecordWorkflow()
    ↓
Database Storage
    ↓
UpdateWorkflow()
    ↓
RecordMetrics/Bugs/Review()
    ↓
Persistent Storage
    ↓
API Endpoints
    ↓
Web Dashboard
```

## 🌐 API Integration

### Automatic Recording

The handlers automatically persist data:

```go
// In factory handlers
workflowID, _ := database.RecordWorkflow("spec-to-code", "pending", input, nil)
// ... execute workflow ...
database.UpdateWorkflow(workflowID, "success", output, "")
database.RecordCodeMetrics(workflowID, "go", metrics)
database.RecordBug(workflowID, bugData)
```

### Dashboard API Endpoints

**Get Statistics**
```bash
curl http://localhost:8080/api/db/stats | jq
```

**Get Workflow History**
```bash
curl 'http://localhost:8080/api/db/workflows?limit=20&offset=0' | jq
```

**Get Workflow Details**
```bash
curl 'http://localhost:8080/api/db/workflow/details?id=<workflow-id>' | jq
```

**Check Database Health**
```bash
curl http://localhost:8080/api/db/health | jq
```

**Search Workflows**
```bash
curl 'http://localhost:8080/api/db/search?type=spec-to-code' | jq
```

## 📈 Dashboard Metrics

The dashboard pulls from the database:

- **Total Issues**: Count from triage_results (last 7 days)
- **Ready to Build**: Issues with status "ready"
- **Active Workflows**: Count of running workflow_runs
- **Success Rate**: success_count / total_count × 100%

**Charts:**
- Issue distribution (feature 45, bug 32, docs 22, refactor 28)
- Workflow status (success/failed by day)

**Tables:**
- Recent workflows (last 5 from database)
- Priority issues (critical 12, high 34)

## 🛠️ Configuration

### Environment Variables

```bash
# Database connection
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=asf

# Or use complete DSN
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/asf

# Server
GITHUB_TOKEN=your-token
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=mistral
```

### Docker Compose

```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: asf
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
```

## 📊 Query Examples

### Get Success Rate
```sql
SELECT 
  COUNT(*) as total,
  COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
  ROUND(100.0 * COUNT(CASE WHEN status = 'success' THEN 1 END) / COUNT(*), 2) as success_rate
FROM workflow_runs
WHERE created_at > NOW() - INTERVAL '24 hours';
```

### Get Bug Statistics
```sql
SELECT 
  severity,
  COUNT(*) as count,
  COUNT(CASE WHEN fixed = true THEN 1 END) as fixed
FROM bug_detections
GROUP BY severity
ORDER BY 
  CASE severity 
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    ELSE 4
  END;
```

### Get Code Quality Trends
```sql
SELECT 
  DATE_TRUNC('day', cm.created_at) as day,
  ROUND(AVG(cm.code_score), 2) as avg_score,
  ROUND(AVG(cm.test_coverage), 2) as avg_coverage,
  SUM(cm.bugs_found) as bugs_found,
  SUM(cm.bugs_fixed) as bugs_fixed
FROM code_metrics cm
GROUP BY DATE_TRUNC('day', cm.created_at)
ORDER BY day DESC;
```

## 🚀 Production Deployment

### With Database

```bash
# 1. Start PostgreSQL
docker compose up -d postgres

# 2. Set environment
export DB_HOST=prod-db.example.com
export DB_PORT=5432
export DB_USER=asf-user
export DB_PASSWORD=secure-password
export DB_NAME=asf

# 3. Start server
./asf-web 0.0.0.0:8080
```

### Without Database

```bash
# Server works fine without database
./asf-web 0.0.0.0:8080
# Dashboard shows real-time metrics but no history
```

## ✅ Verification

**Check Dashboard**
- Open http://localhost:8080 in browser
- Should see metrics, charts, workflows, issues

**Check Database**
```bash
# Should return healthy
curl http://localhost:8080/api/db/health

# Should return statistics
curl http://localhost:8080/api/db/stats | jq '.workflow'

# Should list recent workflows
curl http://localhost:8080/api/db/workflows | jq '.workflows | length'
```

**Direct PostgreSQL**
```bash
psql -h localhost -U postgres -d asf

# Should see tables
\dt

# Should see data
SELECT COUNT(*) FROM workflow_runs;
SELECT COUNT(*) FROM bug_detections;
SELECT COUNT(*) FROM triage_results;
```

## 📝 Notes

- Database is **optional** - factory runs without it
- Migrations run **automatically** on startup
- All timestamps are **timezone-aware**
- Data includes **foreign key constraints**
- **Cascading deletes** clean up related data
- Indexes optimize **frequently-queried columns**
- Pagination supports **large result sets**

## 🐛 Troubleshooting

**"Database not initialized"** - Skip DB env vars or start database
**"Connection refused"** - Check PostgreSQL is running
**"Table already exists"** - Normal; migrations check for existing tables
**Dashboard shows "0" for metrics** - Wait for first workflows to complete
**Can't see historical data** - Ensure workflows ran with DB enabled

---

Both features work together to create a complete observability and persistence layer for the factory.
