# Software Factory UX Plan

## Vision
Create an intuitive, production-ready UX for the Agentic Software Factory that allows one-click launching of agent workflows, similar to GitHub's code review actions.

## Core UX Principles

### 1. **One-Click Launch**
Every factory operation should be launchable with a single click, with smart defaults and sensible fallbacks.

### 2. **Real-Time Feedback**
Users should see immediate visual feedback with progress indicators, logs, and results streaming in real-time.

### 3. **Context-Aware Actions**
Actions should adapt based on the current project state, active spec, and repository context.

### 4. **Error Resilience**
Clear error messages with actionable recovery suggestions, never leaving the user stuck.

### 5. **Accessibility First**
WCAG 2.1 AA compliance, keyboard navigation, screen reader support, high contrast mode.

---

## Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Agentic Software Factory                    👤 User  ⚙️  🌙  🔔  📚         │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │  🎯 ACTIVE SPEC: spec-001 (Auth API)    ▼  [Switch Spec]  [New Spec]  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────────────┤
│  🚀 LAUNCHPAD                                                               │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐              │
│  │ 🔍 CODE REVIEW  │ │ 🐛 BUG FIXER    │ │ ✨ FEATURE GEN │              │
│  │ Analyze PRs     │ │ Auto-fix bugs   │ │ Spec → Code    │              │
│  │ [Launch Review] │ │ [Enable Watch]  │ │ [Generate]     │              │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘              │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐              │
│  │ 🧠 AGENT TUNE   │ │ 🔧 REFACTOR     │ │ 📦 MIGRATE     │              │
│  │ Tune prompts    │ │ Clean code      │ │ Upgrade Go     │              │
│  │ [Run Bench]     │ │ [Clean Code]    │ │ [Migrate]      │              │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘              │
├─────────────────────────────────────────────────────────────────────────────┤
│  📈 RECENT ACTIVITY                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 10:23  ✅ Code Review completed on PR #234                          │   │
│  │ 10:15  🔄 Feature "Auth API" generated → PR #235                   │   │
│  │ 09:42  🐛 Bug fix auto-merged for null pointer in auth.go          │   │
│  │ 09:15  🧠 Prompt optimization completed (+12% success rate)         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────┤
│  🏭 FACTORY STATUS  [Running: 3/4]  [Queue: 2]  [Completed: 47]          │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐            │
│  │ Worker 1 │ │ Worker 2 │ │ Worker 3 │ │ Worker 4 │ │ Queue    │         │
│  │ 🟢 Busy  │ │ 🟢 Busy  │ │ 🟢 Busy  │ │ ⚪ Idle  │ │ 2 tasks  │            │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. LaunchPad Component
```tsx
// web-ui/src/components/LaunchPad.tsx
interface FeatureCardProps {
  icon: ReactNode;
  title: string;
  description: string;
  actionLabel: string;
  endpoint: string;
  color: 'indigo' | 'red' | 'green' | 'purple' | 'orange' | 'cyan';
  specRequired?: boolean;
}

export function FeatureCard({ 
  icon, title, description, actionLabel, endpoint, color, specRequired = true 
}: FeatureCardProps) {
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  
  const handleLaunch = async () => {
    if (!activeSpec && specRequired) {
      toast.error('Please select a spec first');
      return;
    }
    
    setStatus('loading');
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spec_id: activeSpec, priority: 1 }),
      });
      const data = await res.json();
      if (data.task_id) {
        setStatus('success');
        openTaskMonitor(data.task_id);
      }
    } catch {
      setStatus('error');
    }
  };
  
  return (
    <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 hover:shadow-xl transition-shadow">
      <div className="flex items-start gap-3 mb-4">
        <div className={`w-12 h-12 rounded-lg bg-${color}-100 dark:bg-${color}-900/20 flex items-center justify-center`}>
          {icon}
        </div>
        <div className="flex-1">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">{title}</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{description}</p>
        </div>
      </div>
      <Button 
        onClick={handleLaunch} 
        disabled={loading}
        className="w-full"
        variant={status === 'running' ? 'secondary' : 'default'}
      >
        {status === 'loading' && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
        {status === 'success' && <Check className="w-4 h-4 mr-2" />}
        {status === 'idle' && <Play className="w-4 h-4 mr-2" />}
        <span>{status === 'loading' ? 'Launching...' : 
                 status === 'success' ? 'Launched!' : actionLabel}</span>
      </Button>
    </div>
  );
}
```

### 2. TaskMonitor Component (Real-time Modal)
```tsx
// web-ui/src/components/TaskMonitor.tsx
export function TaskMonitor({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const [status, setStatus] = useState<'queued' | 'running' | 'completed' | 'failed'>('queued');
  const [logs, setLogs] = useState<string[]>([]);
  const [result, setResult] = useState<any>(null);

  useEffect(() => {
    const ws = new WebSocket(`${window.location.origin}/api/tasks/${taskId}/stream`);
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.log) setLogs(prev => [...prev, data.log]);
      if (data.status) {
        setStatus(data.status);
        if (data.status === 'completed' || data.status === 'failed') {
          setResult(data);
        }
      }
    };
    return () => ws.close();
  }, [taskId]);

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold">Factory Task</h3>
          <Badge variant={status === 'completed' ? 'success' : status === 'failed' ? 'destructive' : 'default'}>
            {status}
          </Badge>
        </div>
        <div className="font-mono text-sm max-h-96 overflow-auto bg-gray-100 dark:bg-gray-900 p-4 rounded">
          {logs.map((log, i) => <div key={i}>{log}</div>)}
        </div>
        {result && status === 'completed' && (
          <Button onClick={() => onClose()} className="w-full mt-4">
            View Results
          </Button>
        )}
      </div>
    </div>
  );
}
```

### 3. SpecSelector Component
```tsx
// web-ui/src/components/SpecSelector.tsx
export function SpecSelector({ activeSpec, onChange }: { activeSpec: string; onChange: (spec: string) => void }) {
  const [specs, setSpecs] = useState<Spec[]>([]);

  useEffect(() => {
    fetch('/api/specs').then(r => r.json()).then(data => {
      setSpecs(data.specs || []);
    });
  }, []);

  return (
    <div className="flex items-center gap-4">
      <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Active Spec:</label>
      <select 
        value={activeSpec}
        onChange={(e) => onChange(e.target.value)}
        className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
      >
        {specs.map(spec => (
          <option key={spec.id} value={spec.id}>
            {spec.id}: {spec.title}
          </option>
        ))}
      </select>
      <Button variant="outline" size="sm" onClick={() => openSpecEditor()}>
        <Plus className="w-4 h-4 mr-2" />
        New Spec
      </Button>
    </div>
  );
}
```

---

## API Endpoints Required

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/factory/launch` | POST | Generic task launch |
| `/api/factory/launch/review` | POST | Launch code review on PR |
| `/api/factory/launch/bugfix` | POST | Enable bug watcher |
| `/api/factory/launch/feature` | POST | Generate feature from spec |
| `/api/factory/launch/eval` | POST | Run agent benchmarks |
| `/api/factory/launch/refactor` | POST | Run refactoring |
| `/api/factory/launch/migrate` | POST | Run migration |
| `/api/tasks/{id}/stream` | WS | Real-time status |
| `/api/specs` | GET | List specs |
| `/api/activity` | GET | Recent activity feed |
| `/api/factory/status` | GET | Factory status |

---

## Backend Implementation (Go)

### 1. Launch Endpoint (cmd/asf-web/main.go)
```go
mux.HandleFunc("/api/factory/launch", w.handleFactoryLaunch)

func (w *webServer) handleFactoryLaunch(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.writeJSON(map[string]string{"error": "method not allowed"})
        return
    }

    var req struct {
        SpecID     string                 `json:"spec_id"`
        TaskType   string                 `json:"task_type"` // generate, verify, review, refactor
        Parameters map[string]interface{} `json:"parameters"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.writeJSON(map[string]string{"error": "invalid request"})
        return
    }

    if req.SpecID == "" {
        w.writeJSON(map[string]string{"error": "spec_id is required"})
        return
    }

    task := &factory.Task{
        Type:       req.TaskType,
        SpecID:     req.SpecID,
        Priority:   1,
        Parameters: req.Parameters,
    }

    if err := w.factory.SubmitTask(task); err != nil {
        w.writeJSON(map[string]string{"error": err.Error()})
        return
    }

    w.writeJSON(map[string]interface{}{
        "task_id": task.ID,
        "status":  "queued",
        "message": "Factory task launched successfully",
    })
}
```

### 2. WebSocket Streaming (cmd/asf-web/main.go)
```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

mux.HandleFunc("/api/tasks/", w.handleTaskStream)

func (w *webServer) handleTaskStream(w http.ResponseWriter, r *http.Request) {
    taskID := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
    taskID = strings.TrimSuffix(taskID, "/stream")

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        task, ok := w.factory.GetTaskResult(taskID)
        if !ok {
            continue
        }

        err := conn.WriteJSON(map[string]interface{}{
            "status": task.Status,
            "log": fmt.Sprintf("[%s] Task %s: %s", time.Now().Format("15:04:05"), taskID, task.Status),
        })

        if task.Status == "completed" || task.Status == "failed" {
            conn.WriteJSON(map[string]interface{}{
                "status": task.Status,
                "result": task.Result,
                "error": task.Error,
            })
            break
        }
    }
}
```

---

## Spec File Format (YAML)

```yaml
# specs/spec-001.yaml
id: "spec-001"
version: "1.0"
title: "Auth API"
description: "REST API for authentication with JWT tokens"
intent:
  goal: "Implement a complete auth API with login, register, refresh, logout"
  context: "Go microservice using Gin framework"
  success_criteria:
    - "All endpoints return proper HTTP status codes"
    - "JWT tokens properly signed and validated"
    - "Rate limiting on auth endpoints"
    - "100% test coverage for auth logic"
  anti_goals:
    - "No session cookies"
    - "No deprecated crypto"
constraints:
  - id: "c1"
    type: "must"
    description: "Use Gin framework for HTTP"
  - id: "c2"
    type: "must"
    description: "Use golang-jwt for JWT handling"
  - id: "c3"
    type: "must_not"
    description: "No hardcoded secrets"
inputs:
  db_driver:
    type: "string"
    description: "Database driver (postgres/mysql/sqlite)"
    required: true
    default: "postgres"
  jwt_secret:
    type: "string"
    description: "JWT signing secret"
    required: true
outputs:
  api_code:
    type: "string"
    description: "Generated API code"
  tests:
    type: "string"
    description: "Generated test files"
steps:
  - id: "step1"
    name: "Generate Models"
    description: "Create User, Session, Token models"
    tool: "codegen"
    depends_on: []
  - id: "step2"
    name: "Generate Handlers"
    description: "Create login, register, refresh, logout handlers"
    tool: "codegen"
    depends_on: ["step1"]
  - id: "step3"
    name: "Generate Tests"
    description: "Unit and integration tests for auth endpoints"
    tool: "codegen"
    depends_on: ["step2"]
  - id: "step4"
    name: "Run Verification"
    description: "Build, test, lint the generated code"
    tool: "verify"
    depends_on: ["step3"]
```

---

## Implementation Checklist

### Phase 1: Core Launchpad (Week 1)
- [ ] Add `/api/factory/launch` endpoint
- [ ] Add WebSocket `/api/tasks/{id}/stream`
- [ ] Create `LaunchPad` page in Next.js
- [ ] Add `FeatureCard`, `SpecSelector` components
- [ ] Connect to existing `/api/factory/launch`

### Phase 2: Feature Endpoints (Week 2)
- [ ] Add specific launch endpoints (`/review`, `/bugfix`, `/feature`, etc.)
- [ ] Implement GitHub webhook handler for auto-review
- [ ] Add GitHub App configuration

### Phase 3: Activity & Monitoring (Week 3)
- [ ] Create `/api/activity` endpoint
- [ ] Build `ActivityFeed` component
- [ ] Add WebSocket for real-time activity updates

### Phase 4: Polish (Week 4)
- [ ] Add toast notifications
- [ ] Add loading skeletons
- [ ] Add error boundaries
- [ ] Add keyboard shortcuts (Cmd+K to launch)
- [ ] Add dark mode persistence

---

## Dependencies

```bash
cd web-ui
npm install lucide-react clsx tailwind-merge @headlessui/react
npm install -D @types/ws ws
```

---

## Success Metrics

- [ ] User can launch any factory task with 1 click
- [ ] Real-time WebSocket updates show progress
- [ ] Modal opens with task logs on launch
- [ ] Activity feed shows recent launches
- [ ] All 6 feature types launch successfully
- [ ] Dark mode works across all components
- [ ] Keyboard accessible (Tab, Enter, Escape)
- [ ] Screen reader compatible