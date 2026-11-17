# ASF Database Persistence

The Agentic Software Factory now supports PostgreSQL for persistent storage of workflow results, code metrics, bug detections, and triage results.

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Start PostgreSQL
docker compose up -d postgres

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=asf

# Start the web server
./asf-web localhost:8080
```

### Option 2: Manual PostgreSQL Setup

```bash
# Create PostgreSQL user and database
createuser -P postgres  # password: postgres
createdb -O postgres asf

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=asf

# Start the web server
./asf-web localhost:8080
```

## Database Schema

### workflow_runs
Stores information about each workflow execution.

```sql
CREATE TABLE workflow_runs (
  id TEXT PRIMARY KEY,
  type VARCHAR(50) NOT NULL,              -- 'spec-to-code', 'code-review', etc
  status VARCHAR(50) NOT NULL,            -- 'pending', 'running', 'success', 'failed'
  input JSONB,                            -- Workflow input parameters
  output JSONB,                           -- Workflow output results
  error_message TEXT,                     -- Error details if failed
  started_at TIMESTAMP WITH TIME ZONE,
  completed_at TIMESTAMP WITH TIME ZONE,
  duration_ms INTEGER,                    -- Total execution time
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
```

### code_metrics
Stores code quality metrics for generated code.

```sql
CREATE TABLE code_metrics (
  id TEXT PRIMARY KEY,
  workflow_run_id TEXT REFERENCES workflow_runs(id),
  language VARCHAR(50),
  lines_of_code INTEGER,
  code_score INTEGER,                     -- 0-100
  bugs_found INTEGER,
  bugs_fixed INTEGER,
  test_coverage DECIMAL(5, 2),            -- Percentage
  complexity_score INTEGER,
  maintainability_score INTEGER,
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
```

### review_findings
Stores code review results.

```sql
CREATE TABLE review_findings (
  id TEXT PRIMARY KEY,
  workflow_run_id TEXT REFERENCES workflow_runs(id),
  file_name VARCHAR(255),
  score INTEGER,                          -- 0-100
  summary TEXT,
  approved BOOLEAN,
  findings JSONB,                         -- Detailed review findings
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
```

### bug_detections
Stores detected bugs and their fixes.

```sql
CREATE TABLE bug_detections (
  id TEXT PRIMARY KEY,
  workflow_run_id TEXT REFERENCES workflow_runs(id),
  severity VARCHAR(20),                   -- 'critical', 'high', 'medium', 'low'
  bug_type VARCHAR(100),                  -- 'null-pointer', 'race-condition', etc
  location VARCHAR(255),                  -- Line number or function name
  description TEXT,
  impact TEXT,
  fix_suggestion TEXT,
  confidence INTEGER,                     -- 0-100
  fixed BOOLEAN,
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
```

### triage_results
Stores GitHub issue triage analysis results.

```sql
CREATE TABLE triage_results (
  id TEXT PRIMARY KEY,
  issue_number INTEGER,
  repository VARCHAR(255),
  priority VARCHAR(20),                   -- 'critical', 'high', 'medium', 'low'
  category VARCHAR(50),                   -- 'feature', 'bug', 'docs', 'refactor'
  title VARCHAR(255),
  description TEXT,
  suggested_action VARCHAR(255),
  analyzed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
```

## API Endpoints

### Database Statistics
```
GET /api/db/stats
```
Returns comprehensive dashboard statistics including workflow counts, bug statistics, triage results, and average code scores.

### Workflow History
```
GET /api/db/workflows?limit=20&offset=0
```
Retrieves historical workflow data with pagination.

### Workflow Details
```
GET /api/db/workflow/details?id=<workflow-id>
```
Returns detailed information about a specific workflow including associated metrics and bugs.

### Database Health
```
GET /api/db/health
```
Returns database connection status.

### Search Workflows
```
GET /api/db/search?type=spec-to-code&limit=20&offset=0
```
Search for workflows by type with pagination.

## Usage Examples

### Record a Workflow
```go
workflowID, err := database.RecordWorkflow("spec-to-code", "pending", inputData, nil)
// ... execute workflow ...
err = database.UpdateWorkflow(workflowID, "success", outputData, "")
```

### Record Code Metrics
```go
metrics := map[string]interface{}{
  "lines_of_code": 150,
  "code_score": 85,
  "bugs_found": 2,
  "bugs_fixed": 1,
  "test_coverage": 92.5,
}
err := database.RecordCodeMetrics(workflowID, "go", metrics)
```

### Record a Bug
```go
bug := map[string]interface{}{
  "severity": "high",
  "type": "null-pointer",
  "location": "line 42",
  "description": "Potential null pointer dereference",
  "impact": "Runtime crash",
  "fix_suggestion": "Add nil check",
  "confidence": 90,
}
err := database.RecordBug(workflowID, bug)
```

### Record Triage Result
```go
err := database.RecordTriageResult(
  123,                      // issue number
  "PratikDhanave/factory",  // repository
  "high",                   // priority
  "feature",                // category
  "Add database support",   // title
  "Database persistence...", // description
  "Implement database layer", // action
)
```

## Dashboard Integration

The web UI dashboard now displays:
- **Real-time metrics** from the database
- **Workflow history** with status and duration
- **Code quality trends** from metrics
- **Bug statistics** with severity breakdown
- **Triage results** by priority

Enable persistence by setting environment variables before starting the server.

## Notes

- Migrations run automatically on startup
- Database is optional; the factory works without it
- All data is stored in PostgreSQL; no in-memory data loss
- Indexes are created for optimal query performance
- Tables include timestamps for tracking changes

## Troubleshooting

### "Connection refused"
Make sure PostgreSQL is running:
```bash
docker compose ps
```

### "Database does not exist"
Create the database:
```bash
psql -U postgres -c "CREATE DATABASE asf;"
```

### "Permission denied"
Check database credentials in environment variables

### "Table already exists"
This is normal - migrations check for existing tables
