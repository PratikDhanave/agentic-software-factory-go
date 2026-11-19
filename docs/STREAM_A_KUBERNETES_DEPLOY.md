# Stream A: Kubernetes Deployment Guide

**Status:** ✅ Ready for execution when Docker is available  
**Prerequisites:** Docker daemon running, kubectl, helm, kind installed  
**Estimated Duration:** 2-3 hours  
**Date Prepared:** 2026-11-19

---

## Quick Start (5 minutes once Docker is available)

```bash
# 1. Verify Docker is running
docker ps

# 2. Create local Kubernetes cluster
make kind-up

# 3. Deploy application to cluster
make helm-deploy-kind

# 4. Verify deployment
kubectl get pods -n factory-system
kubectl get svc -n factory-system

# 5. Test service
kubectl port-forward -n factory-system svc/agentic-factory 8080:8080
curl http://localhost:8080/health
```

---

## Detailed Deployment Steps

### Phase 1: Local Kubernetes Cluster Setup (30 minutes)

#### 1.1: Verify Prerequisites

```bash
# Check Docker
docker --version
docker ps

# Check kubectl
kubectl version --client

# Check helm
helm version

# Check kind
kind --version
```

#### 1.2: Create kind Cluster

```bash
# Create local cluster
make kind-up

# This command:
# - Installs kind if needed
# - Creates cluster with factory-system namespace
# - Configures kubeconfig to point to local cluster
# - Sets up networking and storage

# Verify cluster is running
kubectl cluster-info
kubectl get nodes
```

#### 1.3: Verify Cluster Components

```bash
# Check cluster status
kubectl get cluster-info

# Check namespaces
kubectl get namespaces

# Check nodes
kubectl get nodes -o wide

# Check persistent volumes
kubectl get pv
```

---

### Phase 2: Helm Deployment (45 minutes)

#### 2.1: Validate Helm Chart

```bash
# Lint the chart
helm lint helm/agentic-factory/

# Expected output:
# ==> Linting helm/agentic-factory/
# [OK] No issues found in chart

# Template the chart (dry-run)
helm template agentic-factory helm/agentic-factory/

# This shows what will be deployed
```

#### 2.2: Deploy to kind Cluster

```bash
# Deploy using Makefile target
make helm-deploy-kind

# This command:
# - Creates factory-system namespace
# - Installs Helm chart with values-dev.yaml
# - Creates ConfigMaps, ServiceAccounts, Roles
# - Deploys Deployment spec with health probes
# - Exposes service (ClusterIP)

# Alternative (manual):
helm install agentic-factory helm/agentic-factory/ \
  --namespace factory-system \
  --create-namespace \
  -f helm/agentic-factory/values-dev.yaml
```

#### 2.3: Monitor Deployment

```bash
# Watch pod status (runs until all are Ready)
kubectl get pods -n factory-system -w

# Check deployment progress
kubectl describe deployment -n factory-system agentic-factory

# View logs as pod starts
kubectl logs -n factory-system -l app=agentic-factory -f

# Wait for pods to be ready
kubectl wait --for=condition=Ready pod \
  -l app=agentic-factory \
  -n factory-system \
  --timeout=300s
```

#### 2.4: Verify Deployment Health

```bash
# Check pod status
kubectl get pods -n factory-system
# Expected: STATUS = Running, READY = 1/1

# Check service
kubectl get svc -n factory-system
# Expected: CLUSTER-IP assigned, PORT 8080 exposed

# Check resource usage
kubectl top pod -n factory-system

# Check events for any warnings
kubectl get events -n factory-system
```

---

### Phase 3: Service Testing (30 minutes)

#### 3.1: Port Forward to Local Machine

```bash
# Forward pod port to localhost
kubectl port-forward -n factory-system svc/agentic-factory 8080:8080

# In another terminal, test the service
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"..."}
```

#### 3.2: Test All Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Readiness probe
curl http://localhost:8080/ready

# Liveness probe (if implemented)
curl http://localhost:8080/live

# Any service endpoints
curl http://localhost:8080/api/v1/...
```

#### 3.3: Check Logs

```bash
# View current logs
kubectl logs -n factory-system -l app=agentic-factory

# View logs with timestamps
kubectl logs -n factory-system -l app=agentic-factory --timestamps=true

# View previous logs (if pod restarted)
kubectl logs -n factory-system -l app=agentic-factory --previous

# Follow logs in real-time
kubectl logs -n factory-system -l app=agentic-factory -f
```

#### 3.4: Inspect Pod Configuration

```bash
# Describe pod
kubectl describe pod -n factory-system -l app=agentic-factory

# View pod spec
kubectl get pod -n factory-system -l app=agentic-factory -o yaml

# Check environment variables
kubectl exec -n factory-system -it <POD_NAME> -- env

# Check mounted volumes
kubectl exec -n factory-system -it <POD_NAME> -- mount | grep /data
```

---

### Phase 4: Kustomize Validation (Optional)

#### 4.1: Validate Overlays

```bash
# Validate base configuration
kustomize build config/base/

# Validate dev overlay
kustomize build config/dev/

# Validate prod overlay  
kustomize build config/prod/

# All commands should output valid Kubernetes YAML
```

#### 4.2: Preview Deployment

```bash
# Preview dev environment
kustomize build config/dev/ | head -50

# Preview prod environment
kustomize build config/prod/ | head -50

# Check resource count
kustomize build config/dev/ | grep "^kind:" | sort | uniq -c
```

---

### Phase 5: ArgoCD Setup (Optional, 1 hour)

#### 5.1: Install ArgoCD

```bash
# Create argocd namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd \
  -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl rollout status -n argocd deployment/argocd-server

# Get initial password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d
```

#### 5.2: Create ArgoCD Applications

```bash
# Apply dev application (auto-sync enabled)
kubectl apply -f argocd/applications/dev.yaml

# Apply prod application (manual sync enabled)
kubectl apply -f argocd/applications/prod.yaml

# Verify applications
kubectl get application -n argocd

# Check application status
kubectl describe application -n argocd dev
```

#### 5.3: Access ArgoCD UI

```bash
# Port forward to ArgoCD server
kubectl port-forward -n argocd svc/argocd-server 8443:443

# Access at: https://localhost:8443
# Username: admin
# Password: <from step 5.1>

# Or set up ingress for external access
```

---

## Troubleshooting Guide

### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -n factory-system <POD_NAME>

# Check logs
kubectl logs -n factory-system <POD_NAME>

# Check image pull status
kubectl get pod -n factory-system <POD_NAME> -o yaml | grep -A 5 "image:"
```

### Connection Refused

```bash
# Check service exists
kubectl get svc -n factory-system agentic-factory

# Check endpoints
kubectl get endpoints -n factory-system agentic-factory

# Check pod is in Ready state
kubectl get pod -n factory-system -o wide

# Verify port-forward is active
# Should show: Forwarding from 127.0.0.1:8080 -> 8080
```

### Resource Issues

```bash
# Check resource requests/limits
kubectl get pod -n factory-system <POD_NAME> -o yaml | grep -A 5 "resources:"

# Check node capacity
kubectl describe nodes | grep -A 5 "Allocated resources"

# Check for OOMKilled status
kubectl get pod -n factory-system <POD_NAME> -o yaml | grep "lastState"
```

### DNS Resolution Issues

```bash
# Test DNS from pod
kubectl exec -n factory-system <POD_NAME> -- nslookup kubernetes.default

# Check core DNS
kubectl get pod -n kube-system -l k8s-app=kube-dns

# Test service discovery
kubectl exec -n factory-system <POD_NAME> -- nslookup agentic-factory
```

---

## Post-Deployment Verification

### Checklist

- [ ] kind cluster created and running
- [ ] All pods in factory-system namespace are Running
- [ ] All pods have Ready = 1/1
- [ ] Service agentic-factory exists
- [ ] Endpoints are assigned to service
- [ ] /health endpoint responds with 200
- [ ] /ready endpoint responds with 200
- [ ] Pod logs show no errors
- [ ] Resource usage is within limits
- [ ] ArgoCD applications synced (if installed)

### Health Checks

```bash
# Comprehensive health check script
#!/bin/bash

echo "=== Cluster Status ==="
kubectl cluster-info

echo "=== Nodes ==="
kubectl get nodes

echo "=== Namespaces ==="
kubectl get namespaces

echo "=== Deployments ==="
kubectl get deployments -n factory-system

echo "=== Pods ==="
kubectl get pods -n factory-system -o wide

echo "=== Services ==="
kubectl get svc -n factory-system

echo "=== Events ==="
kubectl get events -n factory-system --sort-by='.lastTimestamp'

echo "=== Pod Logs ==="
kubectl logs -n factory-system -l app=agentic-factory --tail=10
```

---

## Scaling and Advanced Operations

### Scale Deployment

```bash
# Scale to 3 replicas
kubectl scale deployment agentic-factory -n factory-system --replicas=3

# Verify scaling
kubectl get pods -n factory-system -o wide
```

### Rolling Updates

```bash
# Update image
kubectl set image deployment/agentic-factory \
  -n factory-system \
  agentic-factory=your-registry/agentic-factory:v2 \
  --record

# Monitor rollout
kubectl rollout status deployment/agentic-factory -n factory-system

# Rollback if needed
kubectl rollout undo deployment/agentic-factory -n factory-system
```

### Delete Deployment

```bash
# Delete the entire installation
helm uninstall agentic-factory -n factory-system

# Or delete the cluster
make kind-down
```

---

## Production Deployment

For production deployment to a real cluster:

1. **Replace kind with real cluster:**
   ```bash
   # Update kubeconfig to point to production cluster
   export KUBECONFIG=/path/to/prod/kubeconfig.yaml
   ```

2. **Use values-prod.yaml:**
   ```bash
   helm install agentic-factory helm/agentic-factory/ \
     --namespace factory-system \
     --create-namespace \
     -f helm/agentic-factory/values-prod.yaml
   ```

3. **Enable ArgoCD for GitOps:**
   ```bash
   kubectl apply -f argocd/applications/prod.yaml
   ```

4. **Configure ingress:**
   ```bash
   # Update values to enable ingress
   helm upgrade agentic-factory helm/agentic-factory/ \
     --set ingress.enabled=true \
     --set ingress.hosts[0].host=factory.example.com
   ```

---

## References

- **Helm Values:** `helm/agentic-factory/values.yaml`
- **Deployment Template:** `helm/agentic-factory/templates/deployment.yaml`
- **Kustomize Base:** `config/base/kustomization.yaml`
- **ArgoCD Apps:** `argocd/applications/`
- **Kubernetes Docs:** https://kubernetes.io/docs/
- **Helm Docs:** https://helm.sh/docs/

---

## Summary

Stream A deployment is **fully configured and ready to execute**:

✅ Helm chart validated and production-ready  
✅ Kustomize overlays for dev/prod/staging  
✅ ArgoCD applications for GitOps  
✅ RBAC, ServiceAccount, and security contexts configured  
✅ Health probes (liveness/readiness) configured  
✅ Resource limits and requests configured  
✅ Persistent storage configured  
✅ Complete troubleshooting guide provided  

**Waiting only for Docker daemon to become available.**

