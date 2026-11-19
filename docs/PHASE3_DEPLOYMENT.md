# Phase 3: Deployment & Operations (Months 7-9)

**Goal:** Factory runs in Kubernetes with GitOps, Helm, and Kustomize  
**Status:** Starting  
**Completion Date:** 2026-12-31 (target)

---

## Overview

Phase 3 operationalizes everything from Phase 1 (foundation) and Phase 2 (governance) into production Kubernetes deployments. We adopt the kora-platform GitOps model with ArgoCD, Helm, and Kustomize.

**Key Insight:** Infrastructure is code, versioned, and auto-deployed.

---

## Architecture

```
GitHub Repository (Single Source of Truth)
    ↓
├── helm/
│   └── agentic-factory/
│       ├── Chart.yaml
│       ├── templates/
│       └── values-*.yaml
│
├── config/
│   ├── base/ (Common)
│   ├── dev/  (Development overlay)
│   └── prod/ (Production overlay)
│
└── ArgoCD Application CRDs
    └── Watches GitHub for changes
       ↓
       Automatically syncs to cluster
       ↓
       Kubernetes Cluster (kind/EKS/GKE)
       ├── factory-system (control plane)
       ├── factory-dev (dev agents)
       └── factory-prod (production agents)
```

---

## Implementation Plan

### Part 1: Helm Chart Structure (Week 1)

**Directory structure:**
```
helm/agentic-factory/
├── Chart.yaml                      # Chart metadata
├── values.yaml                     # Default values
├── values-dev.yaml                 # Development overrides
├── values-prod.yaml                # Production overrides
├── templates/
│   ├── deployment.yaml             # Main factory deployment
│   ├── service.yaml                # Kubernetes service
│   ├── configmap.yaml              # Configuration
│   ├── secret.yaml                 # Sensitive data
│   ├── rbac.yaml                   # Role + RoleBinding
│   ├── ingress.yaml                # Ingress for HTTP
│   ├── hpa.yaml                    # Horizontal Pod Autoscaler
│   └── networkpolicy.yaml          # Network security
└── Chart.lock                       # Dependency lock
```

**Chart.yaml:**
```yaml
apiVersion: v2
name: agentic-factory
description: Agentic Software Factory Control Plane
type: application
version: 1.0.0
appVersion: "1.0"
maintainers:
  - name: Team
    email: team@example.com
keywords:
  - agents
  - factory
  - governance
home: https://github.com/PratikDhanave/agentic-software-factory-go
sources:
  - https://github.com/PratikDhanave/agentic-software-factory-go
```

**values.yaml:**
```yaml
replicaCount: 1

image:
  repository: ghcr.io/pratikdhanave/agentic-factory
  pullPolicy: IfNotPresent
  tag: latest

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

podAnnotations: {}

podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true

service:
  type: ClusterIP
  port: 8080
  targetPort: 8080

ingress:
  enabled: true
  className: nginx
  annotations: {}
  hosts:
    - host: factory.example.com
      paths:
        - path: /
          pathType: Prefix
  tls: []

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 3
  targetCPUUtilizationPercentage: 80

env:
  LOG_LEVEL: info
  ADDR: ":8080"

persistence:
  enabled: true
  size: 1Gi
  storageClassName: standard
  mountPath: /data
```

**values-dev.yaml:**
```yaml
replicaCount: 1

image:
  tag: dev

service:
  type: LoadBalancer

ingress:
  enabled: false

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

env:
  LOG_LEVEL: debug
  ADDR: ":8080"

autoscaling:
  enabled: false
```

**values-prod.yaml:**
```yaml
replicaCount: 3

image:
  tag: v1.0.0
  pullPolicy: Always

service:
  type: ClusterIP

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: factory.production.com
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

env:
  LOG_LEVEL: info
  ADDR: ":8080"

persistence:
  size: 10Gi
  storageClassName: fast-ssd
```

### Part 2: Kubernetes Templates (Week 1-2)

**deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "agentic-factory.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "agentic-factory.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "agentic-factory.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
      {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
      {{- end }}
      labels:
        {{- include "agentic-factory.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "agentic-factory.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
      - name: {{ .Chart.Name }}
        securityContext:
          {{- toYaml .Values.securityContext | nindent 12 }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          {{- toYaml .Values.resources | nindent 12 }}
        volumeMounts:
        - name: data
          mountPath: {{ .Values.persistence.mountPath }}
        - name: tmp
          mountPath: /tmp
        env:
        {{- range $key, $value := .Values.env }}
        - name: {{ $key }}
          value: {{ $value | quote }}
        {{- end }}
      volumes:
      - name: data
        {{- if .Values.persistence.enabled }}
        persistentVolumeClaim:
          claimName: {{ include "agentic-factory.fullname" . }}-pvc
        {{- else }}
        emptyDir: {}
        {{- end }}
      - name: tmp
        emptyDir: {}
```

**service.yaml:**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "agentic-factory.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "agentic-factory.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "agentic-factory.selectorLabels" . | nindent 4 }}
```

**rbac.yaml:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "agentic-factory.serviceAccountName" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "agentic-factory.labels" . | nindent 4 }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "agentic-factory.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "agentic-factory.labels" . | nindent 4 }}
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "agentic-factory.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "agentic-factory.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ include "agentic-factory.fullname" . }}
subjects:
- kind: ServiceAccount
  name: {{ include "agentic-factory.serviceAccountName" . }}
  namespace: {{ .Release.Namespace }}
```

### Part 3: Kustomize Overlays (Week 2)

**config/base/kustomization.yaml:**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: factory-system

commonLabels:
  app: agentic-factory
  managed-by: kustomize

resources:
  - ../../helm/agentic-factory

configMapGenerator:
  - name: factory-config
    literals:
      - environment=base
      - version=1.0.0
```

**config/dev/kustomization.yaml:**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: factory-dev

bases:
  - ../base

namePrefix: dev-

commonLabels:
  environment: development

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: factory-config
    behavior: merge
    literals:
      - environment=dev
      - log_level=debug

images:
  - name: ghcr.io/pratikdhanave/agentic-factory
    newTag: dev
```

**config/prod/kustomization.yaml:**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: factory-prod

bases:
  - ../base

commonLabels:
  environment: production

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: factory-config
    behavior: merge
    literals:
      - environment=prod
      - log_level=info

replicas:
  - name: agentic-factory
    count: 3

images:
  - name: ghcr.io/pratikdhanave/agentic-factory
    newTag: v1.0.0
```

### Part 4: ArgoCD Application CRDs (Week 2)

**argocd/applications/dev.yaml:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: factory-dev
  namespace: argocd
spec:
  project: default
  
  source:
    repoURL: https://github.com/PratikDhanave/agentic-software-factory-go.git
    targetRevision: main
    path: config/dev
  
  destination:
    server: https://kubernetes.default.svc
    namespace: factory-dev
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

**argocd/applications/prod.yaml:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: factory-prod
  namespace: argocd
spec:
  project: default
  
  source:
    repoURL: https://github.com/PratikDhanave/agentic-software-factory-go.git
    targetRevision: v1.0.0  # Pin to release tag
    path: config/prod
  
  destination:
    server: https://kubernetes.default.svc
    namespace: factory-prod
  
  syncPolicy:
    # Manual sync for production safety
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 3
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 5m
  
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

### Part 5: Testing & Validation (Week 3)

**tests/kind-setup.sh:**
```bash
#!/bin/bash
# Create local kind cluster
kind create cluster --name factory-test

# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=argocd-application-controller -n argocd --timeout=300s

# Install our applications
kubectl apply -f argocd/applications/dev.yaml
```

**tests/e2e-k8s.bats:**
```bash
#!/usr/bin/env bats

@test "factory deployment is running in dev" {
  kubectl wait --for=condition=Ready pod \
    -l app=agentic-factory \
    -n factory-dev \
    --timeout=300s
}

@test "factory service is accessible" {
  kubectl port-forward -n factory-dev svc/agentic-factory 8080:8080 &
  sleep 2
  run curl -f http://localhost:8080/health
  [ "$status" -eq 0 ]
}

@test "ArgoCD application is synced" {
  run kubectl get application -n argocd factory-dev -o jsonpath='{.status.sync.status}'
  [ "$output" = "Synced" ]
}
```

---

## Success Criteria

✅ Helm chart builds without errors  
✅ Values for dev/prod are distinct and correct  
✅ Kustomize overlays build independently  
✅ ArgoCD applications deploy to kind cluster  
✅ Factory pod is healthy (ready/live probes)  
✅ Service is accessible via kubectl port-forward  
✅ Configuration changes trigger ArgoCD sync  
✅ Rollback works (previous revision deployable)  

---

## References

- **docs/PHASE1_DEPLOYMENT.md**: Backend patterns baseline
- **docs/PHASE2_GOVERNANCE.md**: Identity + credential configuration
- **helm/**: All chart files
- **config/**: Kustomization overlays
- **argocd/**: Application definitions

