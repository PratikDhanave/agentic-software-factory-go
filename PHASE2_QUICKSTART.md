# Phase 2: Quick Start (5 Minutes)

## Status: ✅ Ready to Run

- PostgreSQL: Connected ✓
- Database Schema: Applied ✓
- Phase 2 Server: Compiled ✓
- Configuration: Ready ✓

## Step 1: Configure Claude API Key

Edit `.env` and add your Claude API key:

```bash
# Get your API key from:
# https://console.anthropic.com/account/keys

# Edit .env
CLAUDE_API_KEY=sk-ant-YOUR_KEY_HERE
```

## Step 2: Start the Phase 2 Server

```bash
./bin/phase2-server
```

You should see:

```
✓ PostgreSQL connected
✓ Claude API client initialized
✓ PostgreSQL repository initialized
✓ Skill system initialized
✓ Self-improving service initialized

============================================================
⚙️  Phase 2: Walk - Self-Improving Agents
============================================================

Configuration:
  Phase: walk
  Approval Threshold: 0.90
  Learning Window: 100 decisions

Components Ready:
  ✓ PostgreSQL Database
  ✓ Claude API (Real)
  ✓ Self-Improving Service
  ✓ GitHub Integration

Test Workflow:
  1. POST /test/decision  → Generate decision using Claude
  2. POST /test/approve   → Approve last decision
  3. POST /test/learn     → Trigger pattern learning
  4. GET  /metrics/self-improvement → View metrics

============================================================
```

## Step 3: Test the Full Workflow

### 3.1 Generate a Decision (Uses Real Claude API)

```bash
curl -X POST http://localhost:8000/test/decision
```

**Response:**
```json
{
  "decision": "Implement refresh token rotation with 30-minute intervals",
  "confidence": 0.92,
  "decision_id": "uuid-...",
  "execution_id": "uuid-...",
  "requires_approval": false
}
```

✅ **What happened:**
- Claude analyzed the OAuth2 session timeout issue
- Generated a specific recommendation
- Saved to PostgreSQL with 92% confidence
- High confidence (0.92 >= 0.90) → auto-approved ✓

### 3.2 Check Metrics

```bash
curl http://localhost:8000/metrics/self-improvement | jq .
```

**Response:**
```json
{
  "Agents": {
    "Architect": {
      "SuccessRate": 0,
      "Confidence": 0.92,
      "DecisionCount": 1
    }
  },
  "Patterns": [],
  "Timestamp": "2024-11-18T..."
}
```

✅ **What happened:**
- One decision recorded in PostgreSQL
- Metrics dashboard shows 1 decision from Architect
- Patterns empty (need 100+ decisions before pattern discovery)

### 3.3 Generate 10 More Decisions (Build Dataset)

```bash
for i in {1..10}; do
  echo "Decision $i..."
  curl -s -X POST http://localhost:8000/test/decision | jq '.confidence'
done
```

✅ **What happened:**
- Each decision saved to `agent_decisions` table
- Approval auto-recorded for high-confidence
- Building dataset for pattern discovery

### 3.4 Check Database

```bash
psql factory -c "SELECT agent_role, confidence, approval_status, created_at FROM agent_decisions ORDER BY created_at DESC LIMIT 5;"
```

**Output:**
```
 agent_role | confidence | approval_status |        created_at
 Architect  |       0.92 | pending         | 2024-11-18 14:08:23
 Architect  |       0.88 | pending         | 2024-11-18 14:08:19
 ...
```

✅ **What happened:**
- Decisions persisted to PostgreSQL
- Ready for learning once 100 decisions accumulated

## Step 4: Monitor Real Metrics

```bash
# Terminal 1: Watch metrics update
watch -n 2 'curl -s http://localhost:8000/metrics/self-improvement | jq ".Agents.Architect"'
```

You'll see:
```json
{
  "SuccessRate": 0,
  "Confidence": 0.90,
  "DecisionCount": 11
}
```

## Understanding Phase 2 Workflow

```
1. GitHub Event (Webhook)
   ↓
2. Skill Execution (Real Claude API)
   ↓
3. Decision Generation with Confidence Score
   ↓
4. Auto-Approval? (If confidence >= 0.90)
   ├─ YES → Skip human approval, record as "auto-approved"
   └─ NO  → Require human review
   ↓
5. Store in PostgreSQL
   ├─ agent_decisions table
   ├─ agent_metrics table
   └─ approval records
   ↓
6. Collect Outcomes
   ├─ From GitHub (PR merged?)
   ├─ From human approvals
   └─ From skill execution
   ↓
7. After 100 Decisions → Pattern Learning
   ├─ Discover high-accuracy patterns
   ├─ Boost confidence for future decisions
   ├─ Update specialization matrix
   └─ Improve calibration curves
   ↓
8. Next Decisions Use Learned Patterns
   └─ Higher confidence → More auto-approval
```

## Expected Metrics After Each Stage

| Stage | Decision Count | Patterns | Approval Rate | Note |
|-------|----------------|----------|---------------|------|
| Start | 0 | 0 | 100% | Baseline |
| After 10 | 10 | 0 | High (unless 90%+ confident) | Building dataset |
| After 50 | 50 | 0 | Medium | Need 100 for learning |
| After 100 | 100 | 2-3 | 70-80% | Patterns discovered! |
| After 200 | 200 | 5-7 | 60-70% | Specialization emerges |

## Troubleshooting

### "CLAUDE_API_KEY must be set in .env"

```bash
# Edit .env
CLAUDE_API_KEY=sk-ant-YOUR_REAL_KEY_HERE

# Get key from: https://console.anthropic.com/account/keys
```

### "database connection failed"

```bash
# Check PostgreSQL is running
brew services list | grep postgres

# Should show: postgresql@16 started

# If not:
brew services start postgresql@16
```

### "No recent decisions found"

```bash
# Generate at least one decision first
curl -X POST http://localhost:8000/test/decision

# Then check
curl http://localhost:8000/metrics/self-improvement | jq .
```

### Metrics not updating

```bash
# Check database has decisions
psql factory -c "SELECT COUNT(*) FROM agent_decisions;"

# Check server logs for errors
# (Server output shows any database errors)
```

## Next: Integrate with Real GitHub

Once Phase 2 is working locally:

1. **Configure GitHub Webhook**
   - Settings → Webhooks → Add Webhook
   - URL: `https://your-domain.com/webhooks/github`
   - Events: Pull requests, Issues
   - Secret: `GITHUB_WEBHOOK_SECRET` from .env

2. **Create Issues/PRs**
   - New GitHub issue → triggers triage workflow
   - New PR → triggers code review
   - Webhook sends decision to your server
   - Automatic approval if confident

3. **Monitor Dashboard**
   - View self-improvement metrics
   - See approval rate decrease
   - Patterns discovered after 100 decisions

## Success Criteria for Phase 2

✅ Decisions saved to PostgreSQL  
✅ Real Claude API generating decisions  
✅ High-confidence decisions auto-approve  
✅ Metrics dashboard working  
✅ Database growing with decisions  

Once these work:
- **Week 1-2:** Collect 100+ real decisions
- **Week 3:** Patterns discovered
- **Week 4:** 70%+ approval skipping
- **Then:** Ready for Phase 3 (LLM-as-Judge)

---

## Commands Reference

```bash
# Start server
./bin/phase2-server

# Generate decision
curl -X POST http://localhost:8000/test/decision

# Approve last decision
curl -X POST http://localhost:8000/test/approve

# Trigger learning manually
curl -X POST http://localhost:8000/test/learn

# View metrics
curl http://localhost:8000/metrics/self-improvement | jq .

# Check database
psql factory -c "SELECT COUNT(*) FROM agent_decisions;"
psql factory -c "SELECT agent_role, confidence FROM agent_decisions LIMIT 10;"

# Monitor metrics live
watch -n 2 'curl -s http://localhost:8000/metrics/self-improvement | jq .'
```

---

**Phase 2 is live. Good luck! 🚀**
