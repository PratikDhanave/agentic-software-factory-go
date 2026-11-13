# Next UI Phase: Launching Agent Factory New Features from UI

## Vision
Transform the dashboard into a command center for launching sophisticated agent factory workflows with one-click simplicity, inspired by GitHub's code review actions, Vercel's deployments, and Linear's issue management.

---

## New Features to Launch from UI

### 1. **GitHub Code Review Automation** 🔍
Launch automated code review agents on any PR.

| Trigger | Action | Result |
|---------|--------|--------|
| Button: "🔍 Start Code Review" | Spawns review agents | Annotated PR with findings |
| Button: "🤖 Auto-Fix" | Spawns fix agents | Creates fix commit |
| Button: "📊 Merge Risk Score" | Runs risk analysis | Shows risk dashboard |

**UI Location**: Spec detail page + PR integration page

---

### 2. **Proactive Bug Fix Loop** 🐛
Based on Warp's "Proactive Bug Fixes" pattern — auto-detect errors and generate fixes.

| Trigger | Action | Result |
|---------|--------|--------|
| Button: "🛡️ Enable Bug Watcher" | Monitors Sentry/Logs | Auto-PRs for crashes |
| Button: "🔄 Auto-Retry Failed Tests" | Spawns fix agents | Green CI pipeline |
| Toggle: "Auto-Merge Safe Fixes" | Auto-merge green PRs | Hands-off maintenance |

**UI Location**: Factory dashboard → Monitoring tab

---

### 3. **Feature Generation Pipeline** ✨
One-click feature development from spec to deployed code.

| Trigger | Action | Result |
|---------|--------|--------|
| Button: "🚀 Generate Feature" | Full pipeline: Spec → Code → Test → PR | Ready-to-review PR |
| Button: "📝 Generate Tests Only" | Spawns test agents | Test coverage report |
| Button: "📚 Generate Docs" | Spawns doc agents | README + API docs |

**UI Location**: Specs page → Individual spec detail

---

### 4. **Agent Self-Improvement Loop** 📈
Based on Warp's "Proactive Agent Self-Improvement" — agents review their own performance.

| Trigger | Action | Result |
|---------|--------|--------|
| Button: "🧠 Analyze Agent Performance" | Runs eval benchmarks | Performance report |
| Button: "🔧 Optimize Prompts" | Auto-tunes prompts | Better success rate |
| Button: "📊 Compare Models" | Runs model benchmarks | Model recommendation |

**UI Location**: Evaluation dashboard

---

### 5. **Migration & Refactoring Tools** 🔧
One-click modernization and cleanup.

| Trigger | Action | Result |
|---------|--------|--------|
| Button: "📦 Migrate to Go 1.23" | Spawns migration agents | Updated codebase |
| Button: "🧹 Clean Architecture" | Spawns refactor agents | Clean code PR |
| Button: "⚡ Optimize Performance" | Spawns perf agents | Benchmark + PR |

**UI Location**: Factory dashboard → Tools tab

---

## UI Layout: Feature Launch Dashboard

```
┌─────────────────────────────────────────────────────────────────────┐
│  Agentic Software Factory                    👤 User  ⚙️  🌙      │
├─────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Context     │  │  Specs       │  │  Verify      │  🚀 LAUNCH  │
│  │  Eng.        │  │  Eng.        │  │              │  Factory    │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
├─────────────────────────────────────────────────────────────────────┤
│  📊 FEATURE LAUNCHPAD                                              │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐          │
│  │ 🔍 CODE REVIEW │ │ 🐛 BUG FIXER   │ │ ✨ FEATURE GEN │          │
│  │ Review PRs     │ │ Auto-fix bugs  │ │ Spec → Code    │          │
│  │ [Start Review] │ │ [Enable Watch] │ │ [Generate]     │          │
│  └────────────────┘ └────────────────┘ └────────────────┘          │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐          │
│  │ 🧠 AGENT TUNE  │ │ 🔧 REFACTOR    │ │ 📦 MIGRATE     │          │
│  │ Tune prompts   │ │ Clean code     │ │ Upgrade Go     │          │
│  │ [Run Bench]    │ │ [Clean Code]   │ │ [Migrate]      │          │
│  └────────────────┘ └────────────────┘ └────────────────┘          │
├─────────────────────────────────────────────────────────────────────┤
│  📈 RECENT ACTIVITY                                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ 10:23  ✅ Code Review completed on PR #234                    │  │
│  │ 10:15  🔄 Feature "Auth API" generated → PR #235             │  │
│  │ 09:42  🐛 Bug fix auto-merged for null pointer in auth.go    │  │
│  │ 09:15  🧠 Prompt optimization completed (+12% success rate)   │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Architecture

### API Endpoints Needed

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/factory/launch` | POST | Generic task launch (existing) |
| `/api/factory/launch/review` | POST | Launch code review on PR |
| `/api/factory/launch/bugfix` | POST | Enable bug watcher |
| `/api/factory/launch/feature` | POST | Generate feature from spec |
| `/api/factory/launch/eval` | POST | Run agent benchmarks |
| `/api/factory/launch/refactor` | POST | Run refactoring |
| `/api/factory/launch/migrate` | POST | Run migration |
| `/api/github/pr/{id}/review` | POST | Trigger review on specific PR |
| `/api/tasks/{id}/stream` | WS | Real-time status (existing) |
| `/api/activity` | GET | Recent activity feed |

---

### Frontend Components

#### 1. **FeatureCard** Component
```tsx
interface FeatureCardProps {
  icon: ReactNode;
  title: string;
  description: string;
  actionLabel: string;
  onLaunch: () => void;
  loading?: boolean;
  status?: 'idle' | 'running' | 'success' | 'error';
}

export function FeatureCard({ icon, title, description, actionLabel, onLaunch, loading, status }: FeatureCardProps) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 hover:shadow-xl transition-shadow">
      <div className="flex items-start gap-3 mb-4">
        <div className="w-12 h-12 rounded-lg bg-indigo-100 dark:bg-indigo-900/20 flex items-center justify-center">
          {icon}
        </div>
        <div className="flex-1">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">{title}</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{description}</p>
        </div>
      </div>
      <Button 
        onClick={onLaunch} 
        disabled={loading}
        className="w-full"
        variant={status === 'running' ? 'secondary' : 'default'}
      >
        {loading ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            Running...
          </>
        ) : status === 'success' ? (
          <>
            <Check className="w-4 h-4 mr-2" />
            Completed
          </>
        ) : (
          actionLabel
        )}
      </Button>
    </div>
  );
}
```

#### 2. **ActivityFeed** Component
```tsx
interface ActivityItem {
  time: string;
  status: 'success' | 'running' | 'failed';
  message: string;
  link?: string;
}

export function ActivityFeed({ items }: { items: ActivityItem[] }) {
  return (
    <div className="space-y-3">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
          <div className="text-sm font-mono text-gray-500 w-10">{item.time}</div>
          <Badge 
            variant={item.status === 'success' ? 'success' : 
                     item.status === 'running' ? 'default' : 'destructive'}
            className="text-xs"
          >
            {item.status}
          </Badge>
          <span className="text-sm text-gray-700 dark:text-gray-300 flex-1">
            {item.link ? (
              <a href={item.link} className="hover:underline">{item.message}</a>
            ) : (
              item.message
            )}
          </span>
        </div>
      ))}
    </div>
  );
}
```

#### 3. **Launchpad** Page Component
```tsx
export function Launchpad() {
  const [activeSpec, setActiveSpec] = useState('spec-001');
  const [loadingFeature, setLoadingFeature] = useState<string | null>(null);

  const features = [
    {
      id: 'review',
      icon: <Search className="w-5 h-5 text-indigo-600" />,
      title: 'Code Review',
      description: 'Automated PR review with findings & fixes',
      actionLabel: 'Start Review',
      endpoint: '/api/factory/launch/review',
    },
    {
      id: 'bugfix',
      icon: <Bug className="w-5 h-5 text-red-600" />,
      title: 'Bug Fixer',
      description: 'Auto-detect & fix bugs from logs/Sentry',
      actionLabel: 'Enable Watch',
      endpoint: '/api/factory/launch/bugfix',
    },
    {
      id: 'feature',
      icon: <Sparkles className="w-5 h-5 text-green-600" />,
      title: 'Feature Gen',
      description: 'Spec → Code → Test → PR pipeline',
      actionLabel: 'Generate',
      endpoint: '/api/factory/launch/feature',
    },
    {
      id: 'eval',
      icon: <Brain className="w-5 h-5 text-purple-600" />,
      title: 'Agent Tune',
      description: 'Benchmark & optimize agent prompts',
      actionLabel: 'Run Benchmark',
      endpoint: '/api/factory/launch/eval',
    },
    {
      id: 'refactor',
      icon: <Wrench className="w-5 h-5 text-orange-600" />,
      title: 'Refactor',
      description: 'Clean architecture & optimize code',
      actionLabel: 'Clean Code',
      endpoint: '/api/factory/launch/refactor',
    },
    {
      id: 'migrate',
      icon: <Package className="w-5 h-5 text-cyan-600" />,
      title: 'Migrate',
      description: 'Upgrade Go version & dependencies',
      actionLabel: 'Migrate',
      endpoint: '/api/factory/launch/migrate',
    },
  ];

  const handleLaunch = async (feature: typeof features[0]) => {
    setLoadingFeature(feature.id);
    try {
      const res = await fetch(feature.endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spec_id: activeSpec, priority: 1 }),
      });
      const data = await res.json();
      if (data.task_id) {
        // Open task monitor modal
        openTaskMonitor(data.task_id);
      }
    } catch (e) {
      toast.error('Failed to launch');
    } finally {
      setLoadingFeature(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Spec Selector */}
      <div className="flex items-center gap-4">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Active Spec:</label>
        <select 
          value={activeSpec}
          onChange={(e) => setActiveSpec(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800"
        >
          <option value="spec-001">spec-001: Example Specification</option>
          <option value="spec-002">spec-002: API Generation</option>
          <option value="spec-003">spec-003: Test Generation</option>
        </select>
      </div>

      {/* Feature Grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map(feature => (
          <FeatureCard
            key={feature.id}
            icon={feature.icon}
            title={feature.title}
            description={feature.description}
            actionLabel={feature.actionLabel}
            loading={loadingFeature === feature.id}
            onLaunch={() => handleLaunch(feature)}
          />
        ))}
      </div>
    </div>
  );
}
```

---

## Integration with GitHub

### PR Review Button
Add a "🤖 AI Review" button to PR pages via GitHub App integration:

```typescript
// GitHub App webhook handler
async function handlePREvent(payload: GitHubPREvent) {
  if (payload.action === 'opened' || payload.action === 'synchronize') {
    // Check if repo has auto-review enabled
    const config = await getRepoConfig(payload.repository.id);
    if (config.auto_review) {
      await launchReview({
        repo: payload.repository.full_name,
        pr_number: payload.number,
        base_sha: payload.pull_request.base.sha,
        head_sha: payload.pull_request.head.sha,
      });
    }
  }
}
```

### GitHub App Manifest
```yaml
name: "ASF Agent Factory"
url: "https://your-domain.com"
hook_attributes:
  url: "https://your-domain.com/api/github/webhook"
permissions:
  contents: read
  pull_requests: write
  issues: write
  metadata: read
events:
  - pull_request
  - pull_request_review
  - push
```

---

## Next Implementation Steps

### Phase 1: Core Launchpad (Week 1)
1. ✅ `/api/factory/launch` endpoint
2. ✅ WebSocket task streaming
3. Create `Launchpad` page in Next.js
4. Add `FeatureCard` components
5. Connect to existing `/api/factory/launch`

### Phase 2: Feature Endpoints (Week 2)
1. Add specific launch endpoints (`/review`, `/bugfix`, `/feature`, etc.)
2. Implement GitHub webhook handler for auto-review
3. Add GitHub App configuration

### Phase 3: Activity & Monitoring (Week 3)
1. Create `/api/activity` endpoint
2. Build `ActivityFeed` component
3. Add WebSocket for real-time activity updates

### Phase 4: Polish (Week 4)
1. Add toast notifications
2. Add loading skeletons
3. Add error boundaries
4. Add keyboard shortcuts (Cmd+K to launch)
5. Add dark mode persistence

---

## Acceptance Criteria

- [ ] User sees "Feature Launchpad" on dashboard
- [ ] Spec selector works to choose target
- [ ] Each feature card shows correct status
- [ ] Clicking "Start Review" queues review task
- [ ] Real-time WebSocket updates show progress
- [ ] Modal opens with task logs on launch
- [ ] Activity feed shows recent launches
- [ ] GitHub PR auto-review works when configured
- [ ] All 6 feature types launch successfully
- [ ] Dark mode works across all components

---

## Dependencies

```bash
cd web-ui
npm install lucide-react clsx tailwind-merge @headlessui/react
npm install -D @types/ws ws
```

---

This plan transforms the UI from a static dashboard into an active **command center** for the agent factory — one click launches sophisticated multi-agent workflows just like GitHub's code review buttons trigger CI/CD pipelines.