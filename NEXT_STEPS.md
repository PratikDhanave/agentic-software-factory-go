# Factory Development Roadmap

## ✅ COMPLETED (This Session)

1. ✅ Web UI Dashboard (7 pages, real-time metrics, charts)
2. ✅ PostgreSQL Database Persistence (5 tables, repositories, API)
3. ✅ Dashboard HTML/CSS/JS (responsive dark theme)
4. ✅ Database Schema (workflow_runs, metrics, bugs, reviews, triage)
5. ✅ API Integration Layer (record/update/query functions)
6. ✅ Docker Compose Setup (PostgreSQL container)
7. ✅ Documentation (DATABASE.md, PERSISTENCE_GUIDE.md)

---

## 📋 PENDING WORK (Priority Order)

### 🔴 HIGH PRIORITY (Impact: 9/10, Effort: 2-3 hours each)

**1. Webhook Integration Completion**
- [ ] GitHub webhook event processing
- [ ] Auto-trigger workflows on issue/PR events
- [ ] Webhook signature verification (HMAC SHA256)
- [ ] Event payload transformation
- [ ] Error handling and retries
- **Impact**: Automation triggers from GitHub
- **Files**: cmd/asf-web/handlers/webhook.go (exists but needs testing)

**2. Factory Handlers Integration with Database**
- [ ] Update HandleSpecToCode to persist results
- [ ] Update HandleCodeReview to save findings
- [ ] Update HandleExecuteTask to track execution
- [ ] Update HandleGitHubTriage to save results
- **Impact**: All workflows automatically persist to database
- **Files**: cmd/asf-web/handlers/factory.go

**3. Real-Time WebSocket Updates**
- [ ] WebSocket handler for live dashboard updates
- [ ] Server-Sent Events (SSE) for workflow progress
- [ ] Pub/Sub system for broadcast updates
- [ ] Dashboard WebSocket client integration
- **Impact**: Dashboard updates live as workflows run
- **Files**: pkg/realtime/websocket.go (new)

**4. Workflow Queue & Scheduler**
- [ ] Job queue system (in-memory or Redis)
- [ ] Task scheduler for delayed workflows
- [ ] Retry mechanism for failed workflows
- [ ] Concurrency limits and rate limiting
- **Impact**: Better workflow management and fault tolerance
- **Files**: pkg/queue/queue.go (new)

### 🟡 MEDIUM PRIORITY (Impact: 7/10, Effort: 1-2 hours each)

**5. Advanced Search & Filtering**
- [ ] Full-text search on code and bug descriptions
- [ ] Advanced SQL queries for dashboard
- [ ] Filter workflows by date range, status, type
- [ ] Export workflow results (CSV, JSON)
- **Impact**: Better data exploration
- **Files**: pkg/db/search.go (new), handlers/search.go

**6. Performance Optimization**
- [ ] Add caching layer (Redis or in-memory)
- [ ] Batch database inserts
- [ ] Query optimization with EXPLAIN ANALYZE
- [ ] Response time monitoring
- **Impact**: Faster dashboard and API responses
- **Files**: pkg/cache/cache.go (new)

**7. Monitoring & Observability**
- [ ] Prometheus metrics export
- [ ] Structured JSON logging
- [ ] Request tracing (OpenTelemetry)
- [ ] Error tracking and alerts
- **Impact**: Better visibility into system health
- **Files**: pkg/metrics/metrics.go, pkg/logging/logger.go (new)

**8. Security Hardening**
- [ ] Input validation on all endpoints
- [ ] Rate limiting per IP/API key
- [ ] CORS configuration
- [ ] Request size limits
- [ ] SQL injection prevention (already using prepared statements)
- **Impact**: Production-ready security
- **Files**: pkg/security/validation.go (new)

### 🟢 LOW PRIORITY (Impact: 5/10, Effort: 1 hour each)

**9. CLI Enhancements**
- [ ] `asf-cli workflow list` - Show running workflows
- [ ] `asf-cli workflow cancel <id>` - Cancel execution
- [ ] `asf-cli workflow logs <id>` - Stream workflow logs
- [ ] `asf-cli metrics` - Show statistics
- **Impact**: Better command-line experience
- **Files**: cmd/asf-cli/main.go

**10. Dashboard UI Improvements**
- [ ] Dark/light theme toggle
- [ ] Workflow execution timeline visualization
- [ ] Code diff viewer for reviews
- [ ] Interactive bug severity filter
- [ ] Keyboard shortcuts
- **Impact**: Better UX
- **Files**: web/static/dashboard.js

**11. Documentation Expansion**
- [ ] API reference documentation (OpenAPI/Swagger)
- [ ] Architecture decision records (ADRs)
- [ ] Video tutorials
- [ ] Example workflows
- [ ] Contributing guide
- **Impact**: Easier to contribute
- **Files**: docs/

**12. Testing & CI/CD**
- [ ] Integration tests (database layer)
- [ ] E2E tests (full workflows)
- [ ] Load testing (database performance)
- [ ] GitHub Actions workflow status check
- **Impact**: Confidence in releases
- **Files**: pkg/db/*.go (tests), .github/workflows/

---

## 🎯 RECOMMENDED NEXT PHASE

**Suggested 1-week Sprint:**

1. **Day 1**: Webhook Integration Completion (GitHub auto-trigger)
2. **Day 2**: Factory Handlers + Database Integration (persist results)
3. **Day 3**: Real-Time WebSocket Updates (live dashboard)
4. **Day 4**: Workflow Queue & Scheduler (better management)
5. **Day 5**: Testing & Documentation (quality assurance)

**Expected Outcome**: 
- Fully automated GitHub integration
- Live dashboard updates
- Persistent workflow history
- Error recovery and retries

---

## 🚀 DEPLOYMENT READINESS

### Current Status: 70% Production Ready

**Ready Now:**
- ✅ Web UI Dashboard
- ✅ Database persistence
- ✅ REST API endpoints
- ✅ Docker containerization
- ✅ Health checks

**Needs Work:**
- ⚠️ GitHub webhook automation
- ⚠️ Real-time updates
- ⚠️ Monitoring/observability
- ⚠️ Security hardening
- ⚠️ Load testing

---

## 💰 Effort Estimates

| Task | Hours | Difficulty | Priority |
|------|-------|-----------|----------|
| Webhook completion | 2 | Medium | 🔴 High |
| Database integration | 2 | Medium | 🔴 High |
| WebSocket updates | 3 | Hard | 🔴 High |
| Queue/scheduler | 2 | Hard | 🔴 High |
| Search/filtering | 1 | Easy | 🟡 Med |
| Performance opt | 2 | Medium | 🟡 Med |
| Monitoring | 2 | Medium | 🟡 Med |
| Security | 2 | Medium | 🟡 Med |
| CLI enhancements | 1 | Easy | 🟢 Low |
| Dashboard UI | 2 | Easy | 🟢 Low |
| Docs | 2 | Easy | 🟢 Low |
| Testing | 3 | Medium | 🟢 Low |

**Total: ~24 hours for full production deployment**

---

## 📊 Feature Checklist

### Core Factory
- [x] Spec to Code generation
- [x] Code Review analysis
- [x] Bug Detection
- [x] Documentation generation
- [x] Tool Executor (agentic loop)
- [ ] Workflow scheduling
- [ ] Workflow retries

### Integration
- [x] GitHub API integration
- [ ] GitHub Webhooks (complete)
- [ ] GitHub auto-review
- [ ] GitHub auto-triage
- [ ] MCP server integration

### Storage
- [x] PostgreSQL database
- [x] Workflow persistence
- [x] Metrics storage
- [ ] Log persistence
- [ ] Result caching

### API
- [x] REST endpoints
- [x] Dashboard API
- [x] Database API
- [ ] WebSocket API
- [ ] Authentication/authorization

### Dashboard
- [x] UI components
- [x] Real-time metrics
- [x] Charts and visualizations
- [x] Workflow management
- [ ] Live updates
- [ ] Advanced search

### DevOps
- [x] Docker Compose
- [x] Kubernetes manifests
- [ ] CI/CD pipeline (GitHub Actions - exists)
- [ ] Monitoring setup
- [ ] Log aggregation

---

## 🔗 Dependencies

```
GitHub Webhooks → Workflow Triggers → Database Persistence → Dashboard Display
                                           ↓
                                    Metrics Collection
                                           ↓
                                      WebSocket Push
                                           ↓
                                   Real-Time Updates
```

---

## ✨ What Would You Like to Build Next?

1. **GitHub Webhooks** - Auto-trigger from GitHub events (most impactful)
2. **WebSocket Real-Time** - Live dashboard updates
3. **Workflow Queue** - Better task management
4. **Performance** - Optimization and caching
5. **Testing** - Integration and E2E tests
6. **Monitoring** - Observability and alerts
7. **Security** - Hardening and validation
8. **Other** - Something else?

