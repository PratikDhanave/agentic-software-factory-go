# Phase 2: Walk Implementation Guide

## Overview

Phase 2 enables self-improvement and graduated autonomy. Agents learn from approval feedback, discover high-confidence patterns, and high-confidence (90%+) decisions skip human approval.

## Implementation Checklist

### ✅ Completed (Just Implemented)

- [x] Real Claude API integration (`internal/domain/services/claude_api_real.go`)
- [x] PostgreSQL repository layer (`internal/adapters/persistence/postgres_repository.go`)
- [x] Self-improvement metrics dashboard (`web-ui/src/components/SelfImprovementDashboard.tsx`)
- [x] Domain types for metrics (`internal/domain/types.go`)
- [x] Self-improving service (`internal/application/self_improving_service.go`)
- [x] Integration manager (`cmd/asf-web/integration.go`)

### 🚀 Ready to Complete (This Week)

- [ ] Set up PostgreSQL database
- [ ] Create `.env` configuration
- [ ] Wire real Claude API client into main server
- [ ] Implement approval skipping for high-confidence decisions
- [ ] Deploy self-improvement dashboard
- [ ] Test full feedback loop with real GitHub events
- [ ] Enable pattern learning after 100 decisions

## Step-by-Step Setup

### 1. PostgreSQL Database Setup

```bash
# Start PostgreSQL (if using Docker)
docker run -d \
  --name factory-db \
  -e POSTGRES_DB=factory \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15

# Or use existing PostgreSQL
psql -U postgres -c "CREATE DATABASE factory;"

# Apply schema
psql -U postgres -d factory -f internal/adapters/persistence/postgres_schema.sql
```

### 2. Environment Configuration

Create `.env` file:

```bash
# Claude API
CLAUDE_API_KEY="sk-ant-..."
CLAUDE_MODEL="claude-3-5-sonnet-20241022"

# GitHub
GITHUB_WEBHOOK_SECRET="whsec_your_secret_here"
GITHUB_TOKEN="ghp_your_token_here"

# Database
DATABASE_URL="postgresql://postgres:password@localhost:5432/factory"

# Server
PORT="8000"
LOG_LEVEL="debug"

# Phase configuration
FACTORY_PHASE="walk"  # crawl, walk, or run
APPROVAL_THRESHOLD="0.90"  # Approve decisions above this confidence
LEARNING_WINDOW="100"  # Trigger learning after N decisions
```

### 3. Update Main Server

Modify `cmd/asf-web/main.go`:

```go
package main

import (
	"database/sql"
	"os"
	
	_ "github.com/lib/pq"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

func main() {
	// 1. Connect to PostgreSQL
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// 2. Initialize real Claude client
	llmClient := services.NewRealClaudeAPIClient(
		os.Getenv("CLAUDE_API_KEY"),
		os.Getenv("CLAUDE_MODEL"),
	)

	// 3. Initialize repository
	repo := persistence.NewPostgresRepository(db)

	// 4. Initialize skill registry
	skillRegistry := services.NewSkillRegistry()
	skillRegistry.RegisterSkill(services.NewCodeReviewSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewTriageIssueSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewSecurityReviewSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewReviewSpecSkill(llmClient))

	// 5. Initialize self-improving service
	selfImprovingService := application.NewSelfImprovingService(repo, llmClient)

	// 6. Initialize integration manager
	integrationMgr := NewIntegrationManager(repo, llmClient, skillRegistry)
	integrationMgr.selfImprovingService = selfImprovingService

	// 7. Register routes
	mux := http.NewServeMux()
	integrationMgr.RegisterRoutes(mux)
	
	// Add self-improvement dashboard endpoint
	mux.HandleFunc("GET /dashboard/self-improvement", func(w http.ResponseWriter, r *http.Request) {
		// Serve React component or redirect to web-ui
		http.ServeFile(w, r, "web-ui/src/components/SelfImprovementDashboard.tsx")
	})

	// 8. Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	
	log.Printf("Server starting on :%s (Phase: Walk)\n", port)
	http.ListenAndServe(":"+port, mux)
}
```

### 4. Install Dependencies

```bash
# Go dependencies
go get github.com/lib/pq  # PostgreSQL driver
go get github.com/google/uuid

# Node dependencies (for dashboard)
cd web-ui
npm install recharts  # Charts library
npm install
```

### 5. Update Web UI

Add self-improvement dashboard to main page (`web-ui/src/app/page.tsx`):

```tsx
import { SelfImprovementDashboard } from '../components/SelfImprovementDashboard';

export default function Home() {
  return (
    <main>
      {/* Existing Factory Stack Control Plane */}
      <FactoryStackDashboard />
      
      {/* New: Self-Improvement Dashboard */}
      <SelfImprovementDashboard />
    </main>
  );
}
```

### 6. Update Approval Workflow

Modify approval handling in `cmd/asf-web/main.go`:

```go
func (s *webServer) handleWorkflowGenerate(w http.ResponseWriter, r *http.Request) {
	// ... existing code ...
	
	for _, decision := range decisions {
		// Phase 2: Check if approval is needed
		phase := os.Getenv("FACTORY_PHASE")
		if phase == "walk" {
			threshold, _ := strconv.ParseFloat(os.Getenv("APPROVAL_THRESHOLD"), 64)
			if decision.Confidence >= threshold {
				// High-confidence decision - skip approval
				decision.RequiresApproval = false
				// Record approval as "auto-approved"
				s.selfImprovingService.RecordApproval(ctx, &domain.Approval{
					Status: "approved",
					DecisionID: decision.ID,
				})
				continue
			}
		}
		
		// Otherwise, requires approval
		decision.RequiresApproval = true
	}
	
	// ... rest of function ...
}
```

### 7. Configure GitHub Webhook

Update webhook URL in GitHub repository settings:

```
Settings → Webhooks → Edit
- Payload URL: https://your-domain.com/webhooks/github
- Content type: application/json
- Secret: (use GITHUB_WEBHOOK_SECRET from .env)
- Events: Pull requests, Issues, Pull request reviews
- Active: ✓
```

### 8. Test Full Loop

#### Test 1: GitHub Event → Decision → Database

```bash
# 1. Create a test issue
curl -X POST https://api.github.com/repos/YOUR_REPO/issues \
  -H "Authorization: token $GITHUB_TOKEN" \
  -d '{"title": "Test auth issue", "body": "OAuth2 flow broken"}'

# 2. Webhook handler should:
# - Receive GitHub event
# - Trigger triage skill
# - Generate decision using real Claude API
# - Save decision to PostgreSQL

# 3. Check database
psql factory -c "SELECT * FROM agent_decisions ORDER BY created_at DESC LIMIT 1;"
```

#### Test 2: Approval Recording

```bash
# Approve a decision
curl -X POST http://localhost:8000/api/approvals \
  -H "Content-Type: application/json" \
  -d '{
    "approval_id": "uuid-from-decision",
    "status": "approved",
    "comments": "Good decision"
  }'

# Check metrics were recorded
psql factory -c "SELECT * FROM agent_metrics ORDER BY created_at DESC LIMIT 5;"
```

#### Test 3: Pattern Learning

```bash
# After 100 approvals, check if patterns were discovered
psql factory -c "SELECT * FROM learned_patterns WHERE success_rate > 0.80;"

# Verify pattern application in next decision
# - Decision confidence should be boosted if pattern matches
# - Approval requirement should be waived if confidence >= 0.90
```

#### Test 4: Dashboard Metrics

```bash
# Access dashboard
curl http://localhost:8000/metrics/self-improvement | jq .

# Should show:
# {
#   "Agents": {
#     "Architect": { "SuccessRate": 0.92, "Confidence": 0.89, "DecisionCount": 42 },
#     ...
#   },
#   "Patterns": [
#     { "IssueType": "auth", "Agent": "Architect", "SuccessRate": 0.95, "Applied": 5 }
#   ]
# }
```

## Approval Skipping Logic

### Phase 1: Crawl (100% Approval Required)

```
Decision confidence: 0.87
↓
Requires approval? YES (all decisions approved by human)
↓
Wait for human review
```

### Phase 2: Walk (90%+ Skips Approval)

```
Decision confidence: 0.87
↓
Requires approval? YES (0.87 < 0.90)
↓
Wait for human review

---

Decision confidence: 0.93 (from learned pattern)
↓
Requires approval? NO (0.93 >= 0.90)
↓
Auto-approve and execute immediately
↓
Record as "auto-approved" for learning
↓
Collect GitHub outcome for feedback
```

## Learning Trigger Points

### Every 100 Decisions for Each Agent

```
Decision 1-99: Collect metrics, store in database
Decision 100: Trigger learning

Learning process:
1. Fetch recent 100 decisions for agent
2. Group by issue type
3. For each type with 10+ decisions:
   - Calculate success rate (approved / total)
   - If success_rate >= 80%:
     - Create pattern
     - Set confidence_base = success_rate * 0.95
   - Save to learned_patterns table
4. Update agent specialization matrix
5. Run confidence calibration
6. Reset decision counter
```

## Monitoring & Validation

### KPIs to Track

```
Phase 1 → Phase 2 Transition:

Approval Rate
- Week 1: 100% (Crawl)
- Week 2: 95% (baseline)
- Week 3: 85% (patterns emerging)
- Week 4: 75% (high-confidence decisions skipping)

Success Rate
- Week 1: Baseline (heuristic)
- Week 2: 85% (from Claude API)
- Week 3: 88% (learning)
- Week 4: 92% (patterns optimized)

Pattern Coverage
- Week 1: 0 patterns
- Week 2: 0 patterns (need 100 decisions)
- Week 3: 2-3 patterns discovered
- Week 4: 5-7 patterns

Confidence Calibration
- Week 1: Miscalibrated (agent overconfident)
- Week 2: Detecting calibration error
- Week 3: Applying corrections
- Week 4: Well-calibrated (Calibration ~1.0)
```

### Dashboard Signals

```
✅ Ready for Phase 3 when:
- Average success rate >= 92%
- Pattern discovery enabled and working
- 50%+ decisions skipping approval
- Confidence calibration ratio ~1.0
- No human rejections of auto-approved decisions
```

## Troubleshooting

### Issue: Decisions not being saved to database

```bash
# Check database connection
psql factory -c "SELECT 1;"

# Check for table creation errors
psql factory -c "\dt"  # Should see agent_decisions, approvals, etc.

# Check application logs
grep -i "error\|failed" logs/app.log
```

### Issue: Claude API errors

```bash
# Validate API key
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $CLAUDE_API_KEY" \
  -H "content-type: application/json" \
  -d '{"model": "claude-3-5-sonnet-20241022", "max_tokens": 100, "messages": [{"role": "user", "content": "hi"}]}'

# Check API quota
# Log into Anthropic console: https://console.anthropic.com/
```

### Issue: Patterns not discovered after 100 decisions

```bash
# Check if learning was triggered
psql factory -c "SELECT COUNT(*) FROM agent_decisions;"

# Manually trigger learning
curl -X POST http://localhost:8000/api/admin/learn \
  -d '{"agent": "Architect"}'

# Check for patterns
psql factory -c "SELECT * FROM learned_patterns;"
```

## Next Steps (Phase 3: Run)

Once Phase 2 metrics look good (92%+ success, 50%+ auto-approval):

1. **LLM-as-Judge**: Deploy second Claude model to validate critical decisions
2. **GitHub Actions**: Implement skill to create/merge PRs automatically
3. **Autonomous Execution**: Enable high-confidence decisions to execute without approval
4. **Proactive Agents**: Add recommendation system for preventive fixes

---

## References

- PostgreSQL setup: https://www.postgresql.org/docs/15/
- Anthropic API: https://docs.anthropic.com/en/docs/about-claude/models/overview
- Recharts (dashboard): https://recharts.org/

