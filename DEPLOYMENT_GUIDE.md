# Factory Deployment Guide

## Quick Start (5 minutes)

### 1. Prerequisites
```bash
# Install Go 1.20+
go version

# Install PostgreSQL (optional but recommended)
postgres --version

# Or use Docker
docker --version
```

### 2. Clone & Build
```bash
cd agentic-software-factory-go
go build -o asf-web ./cmd/asf-web
go build -o asf-cli ./cmd/asf-cli
```

### 3. Configure
```bash
# GitHub webhook secret (generate with: openssl rand -hex 32)
export GITHUB_WEBHOOK_SECRET=your-32-byte-secret

# Queue settings (optional)
export QUEUE_MAX_WORKERS=5
export QUEUE_MAX_RETRIES=3

# Ollama (optional)
export OLLAMA_URL=http://localhost:11434
export OLLAMA_MODEL=mistral
```

### 4. Run
```bash
./asf-web localhost:8080
```

Visit http://localhost:8080 in your browser.

---

## Production Deployment

### Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o asf-web ./cmd/asf-web

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/asf-web /app/
EXPOSE 8080
CMD ["/app/asf-web", "0.0.0.0:8080"]
```

Build and run:
```bash
docker build -t factory:latest .
docker run -p 8080:8080 \
  -e GITHUB_WEBHOOK_SECRET=... \
  -e DB_HOST=postgres \
  -e OLLAMA_URL=http://ollama:11434 \
  factory:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  factory:
    build: .
    ports:
      - "8080:8080"
    environment:
      GITHUB_WEBHOOK_SECRET: ${GITHUB_WEBHOOK_SECRET}
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: asf
      OLLAMA_URL: http://ollama:11434
      OLLAMA_MODEL: mistral
    depends_on:
      - postgres
      - ollama
    networks:
      - factory-net

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: asf
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - factory-net

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ollama_data:/root/.ollama
    networks:
      - factory-net

volumes:
  postgres_data:
  ollama_data:

networks:
  factory-net:
    driver: bridge
```

Run with:
```bash
docker-compose up -d
```

### Kubernetes Deployment

```yaml
# manifests/factory-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: factory
  labels:
    app: factory
spec:
  replicas: 3
  selector:
    matchLabels:
      app: factory
  template:
    metadata:
      labels:
        app: factory
    spec:
      containers:
      - name: factory
        image: factory:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
        env:
        - name: GITHUB_WEBHOOK_SECRET
          valueFrom:
            secretKeyRef:
              name: github-secret
              key: webhook-secret
        - name: DB_HOST
          value: postgres-service
        - name: OLLAMA_URL
          value: http://ollama-service:11434
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
---
apiVersion: v1
kind: Service
metadata:
  name: factory-service
spec:
  selector:
    app: factory
  type: LoadBalancer
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
```

Deploy with:
```bash
kubectl create secret generic github-secret --from-literal=webhook-secret=...
kubectl apply -f manifests/factory-deployment.yaml
```

---

## Configuration

### Environment Variables

**Required:**
```bash
GITHUB_WEBHOOK_SECRET=your-32-byte-hex-secret
```

**Optional - Queue:**
```bash
QUEUE_MAX_WORKERS=5          # Concurrent workflow executors
QUEUE_MAX_RETRIES=3          # Retries on failure
```

**Optional - Database:**
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=asf
DATABASE_URL=postgresql://user:pass@host:5432/asf  # Alternative to individual vars
```

**Optional - Ollama:**
```bash
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=mistral
```

---

## GitHub Webhook Setup

### 1. Configure GitHub Repository
1. Go to Settings → Webhooks → Add webhook
2. **Payload URL**: `https://your-domain.com/webhooks/github`
3. **Content type**: `application/json`
4. **Secret**: (paste the value from GITHUB_WEBHOOK_SECRET)
5. **Events**: Select:
   - Issues
   - Pull requests
   - Pull request reviews
   - Pushes
6. Click "Add webhook"

### 2. Test Webhook
```bash
# GitHub will send a "ping" event automatically
# Check the webhook delivery log in GitHub settings
```

---

## Monitoring & Observability

### Health Check
```bash
curl http://localhost:8080/health
```

### Metrics
```bash
curl http://localhost:8080/api/metrics/detailed | jq
```

### Audit Log
```bash
curl http://localhost:8080/api/audit/log | jq
```

### Real-Time Stream
```bash
curl -N http://localhost:8080/api/stream
```

---

## Performance Tuning

### Queue Tuning
```bash
# For high throughput:
QUEUE_MAX_WORKERS=20        # More concurrent workers
QUEUE_MAX_RETRIES=5         # More retry attempts

# For low latency:
QUEUE_MAX_WORKERS=5         # Conservative resources
QUEUE_MAX_RETRIES=2         # Quick failure feedback
```

### Rate Limiting
Default: 100 requests/second per IP

Increase if behind load balancer:
- Configure trusted proxies
- Monitor 429 responses in audit log

### Database Optimization
```sql
-- Check query performance
EXPLAIN ANALYZE SELECT * FROM workflow_runs WHERE status = 'running';

-- Analyze indexes
ANALYZE workflow_runs;
```

---

## Scaling Considerations

### Horizontal Scaling
1. Run multiple factory instances behind a load balancer
2. All instances share the same PostgreSQL database
3. Webhook secret must be identical across instances
4. Realtime hub broadcasts to all connected clients (per instance)

### Database Connection Pool
```go
// In database.go
conn.SetMaxOpenConns(25)      // Adjust per workload
conn.SetMaxIdleConns(5)        // Keep warm connections
conn.SetConnMaxLifetime(5 * time.Minute)
```

### Queue Sizing
- Monitor queue depth: `/api/jobs/list`
- If backlog grows, increase QUEUE_MAX_WORKERS
- Monitor success rate in `/api/metrics/detailed`

---

## Troubleshooting

### Webhook Not Triggering
1. Check GitHub webhook delivery logs
2. Verify GITHUB_WEBHOOK_SECRET matches
3. Check firewall rules (port 8080 accessible)
4. Verify event types are selected in GitHub

### High Error Rate
1. Check `/api/audit/log` for error details
2. Monitor queue with `/api/jobs/list`
3. Check Ollama connectivity if LLM features enabled
4. Review database performance

### Database Connection Issues
1. Verify DB credentials
2. Check PostgreSQL is running: `psql -c "SELECT 1"`
3. Verify network connectivity
4. Check connection pool settings

### High Memory Usage
1. Reduce METRICS_MAX_HISTORY (default 1000)
2. Reduce AUDIT_LOG_SIZE (default 500)
3. Monitor active websocket connections
4. Consider horizontal scaling

---

## Backup & Recovery

### Database Backup
```bash
# Daily backups
pg_dump asf > asf-$(date +%Y%m%d).sql

# With compression
pg_dump --compress=9 asf > asf-$(date +%Y%m%d).sql.gz
```

### Restore
```bash
psql asf < asf-20261117.sql
```

### Disaster Recovery
1. Redeploy factory binary
2. Restore database from backup
3. Verify webhook delivery logs for missed events
4. Manually retrigger critical workflows

---

## Monitoring Setup

### Prometheus Integration (Future)
```yaml
# Metrics endpoint (to be implemented)
GET /metrics

# Format: Prometheus text format
workflow_duration_ms_sum{type="issue-triage"} 500000
workflow_duration_ms_count{type="issue-triage"} 100
```

### Log Aggregation
All logs go to stdout → capture with:
- Docker logs: `docker logs container_name`
- Kubernetes logs: `kubectl logs pod_name`
- ELK stack: forward stdout to Logstash
- CloudWatch: forward stdout to CloudWatch

---

## Security Checklist

- [ ] Set strong GITHUB_WEBHOOK_SECRET (32+ bytes hex)
- [ ] Enable HTTPS for webhook endpoint (reverse proxy)
- [ ] Set up firewall rules (only allow GitHub IPs)
- [ ] Configure database credentials securely
- [ ] Enable audit logging and monitor for anomalies
- [ ] Rate limiting configured appropriately
- [ ] Regular database backups enabled
- [ ] Webhook delivery logs monitored
- [ ] Network segmentation (factory ← Postgres ← Ollama)

---

## Rollout Strategy

### Canary Deployment
1. Deploy new version to 10% of instances
2. Monitor metrics for 15 minutes
3. If healthy, deploy to 50% of instances
4. If still healthy, deploy to 100%
5. Keep old version running for quick rollback

### Blue-Green Deployment
1. Deploy new version to blue environment
2. Run smoke tests
3. Switch load balancer to blue
4. Keep green running for rollback

### Rollback
```bash
# Kubernetes rollback
kubectl rollout undo deployment/factory

# Docker: restart with previous image
docker pull factory:v1.0.0
docker run -p 8080:8080 factory:v1.0.0
```

---

## Maintenance

### Weekly Tasks
- Review audit log for security events
- Check queue depth and success rate
- Monitor error rate in metrics
- Verify webhook deliveries in GitHub

### Monthly Tasks
- Database optimization and VACUUM
- Review and clean old workflow history
- Update dependencies and security patches
- Capacity planning based on trends

### Quarterly Tasks
- Full backup restore test
- Disaster recovery drill
- Performance benchmarking
- Security audit
