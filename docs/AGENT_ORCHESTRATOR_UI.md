# Agent Orchestrator UI

A comprehensive web interface for launching, monitoring, and coordinating multiple agents in parallel with real-time graph visualization.

## Features

### 🎯 Launch & Control
- **Launch agents** with custom name, type, description, and dependencies
- **Pause/Resume** execution without losing state
- **Stop/Cancel** agents forcefully
- **Delete** completed or failed agents
- **Parent-child relationships** for sequential or parallel execution

### 📊 Real-Time Monitoring
- **Live status updates** via WebSocket
- **Progress tracking** for running agents
- **Duration calculation** for completed tasks
- **Status indicators** (running ⚙️, completed ✅, failed ❌, paused ⏸️)

### 📈 Graph Visualization
- **SVG-based graph** showing agent relationships
- **Auto-layout** with hierarchical positioning
- **Connection labels** (sequential, parallel, conditional)
- **Interactive nodes** - click to select and view details

### 📋 Agent List
- **Table view** of all agent instances
- **Quick status overview** with filtering
- **Individual agent metrics** (progress, duration, type)

## UI Components

### File Structure
```
web-ui/src/
├── app/agents/
│   └── orchestrator/
│       └── page.tsx           # Main page
├── components/
│   └── AgentOrchestrator.tsx   # Main component (650+ lines)
└── lib/
    └── agentOrchestrator.ts    # Backend integration utilities
```

### Component: AgentOrchestrator

**Location:** `web-ui/src/components/AgentOrchestrator.tsx`

Main React component with:
- Agent state management
- Graph visualization
- Modal for agent launch
- Control panel for selected agent
- Agent list table

**Key Props:**
- None required (fully self-contained state)

**Usage:**
```tsx
import AgentOrchestrator from '@/components/AgentOrchestrator';

export default function Page() {
  return <AgentOrchestrator />;
}
```

## Backend Integration

### Required API Endpoints

Your Go backend should implement these endpoints:

#### 1. Launch Agent
```
POST /api/agents/launch
Content-Type: application/json

{
  "name": "Bug Finder",
  "type": "code-reviewer",
  "description": "Find bugs in state machines",
  "parentId": "agent-123",  // optional
  "tags": ["bug", "state"],
  "timeout": 3600
}

Response:
{
  "id": "agent-456",
  "name": "Bug Finder",
  "type": "code-reviewer",
  "status": "running",
  "progress": 0,
  "startTime": "2026-11-18T10:30:00Z"
}
```

#### 2. Get Agent Status
```
GET /api/agents/{agentId}

Response:
{
  "id": "agent-456",
  "name": "Bug Finder",
  "type": "code-reviewer",
  "status": "running|completed|failed|paused",
  "progress": 45,
  "startTime": "2026-11-18T10:30:00Z",
  "endTime": null,
  "output": "Found 10 bugs...",
  "metadata": {}
}
```

#### 3. Get All Agents
```
GET /api/agents

Response:
[
  { "id": "agent-456", ... },
  { "id": "agent-789", ... }
]
```

#### 4. Control Endpoints
```
POST /api/agents/{agentId}/pause
POST /api/agents/{agentId}/resume
POST /api/agents/{agentId}/stop
DELETE /api/agents/{agentId}
```

#### 5. Get Agent Output
```
GET /api/agents/{agentId}/output?limit=100

Response:
[
  "Starting agent...",
  "Processing...",
  "Found bug #1..."
]
```

#### 6. Create Dependency
```
POST /api/agents/dependencies
Content-Type: application/json

{
  "from": "agent-123",
  "to": "agent-456",
  "label": "sequential",
  "dependencyType": "sequential|parallel|conditional"
}

Response:
{
  "from": "agent-123",
  "to": "agent-456",
  "label": "sequential"
}
```

#### 7. Get Agent Graph
```
GET /api/agents/graph

Response:
{
  "agents": [
    { "id": "agent-456", ... },
    { "id": "agent-789", ... }
  ],
  "connections": [
    { "from": "agent-456", "to": "agent-789", "label": "sequential" }
  ]
}
```

### WebSocket Connection

For real-time updates, implement WebSocket at `ws://localhost:3000/ws/agents`:

```go
// Go example
ws.HandleFunc("/ws/agents", func(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    for {
        // Send agent status updates
        update := map[string]interface{}{
            "type": "agent:status_update",
            "agentId": "agent-456",
            "payload": map[string]interface{}{
                "status": "running",
                "progress": 45,
            },
        }
        conn.WriteJSON(update)
    }
})
```

**Message Format:**
```json
{
  "type": "agent:status_update",
  "agentId": "agent-456",
  "payload": {
    "status": "running",
    "progress": 45,
    "output": "Processing...",
    "timestamp": "2026-11-18T10:35:00Z"
  }
}
```

## Frontend Usage

### Using the Orchestrator Utility

```typescript
import { orchestrator } from '@/lib/agentOrchestrator';

// Connect
await orchestrator.connect();

// Subscribe to updates
const unsubscribe = orchestrator.subscribe('agent:status_update', (data) => {
  console.log('Agent update:', data);
});

// Launch agent
const agent = await orchestrator.launchAgent({
  name: 'Bug Finder',
  type: 'code-reviewer',
  description: 'Find bugs',
});

// Wait for completion
const result = await orchestrator.waitForAgent(agent.id);

// Control agents
await orchestrator.pauseAgent(agent.id);
await orchestrator.resumeAgent(agent.id);
await orchestrator.stopAgent(agent.id);

// Cleanup
unsubscribe();
orchestrator.disconnect();
```

### Launch Multiple Agents in Parallel

```typescript
const agents = await orchestrator.launchMultipleAgents([
  { name: 'Bug Finder', type: 'code-reviewer' },
  { name: 'Code Generator', type: 'code-generator' },
  { name: 'Tester', type: 'tester' },
]);

// Wait for all to complete
await Promise.all(agents.map(a => orchestrator.waitForAgent(a.id)));
```

### Create Agent Pipeline

```typescript
// Launch parent agent
const parent = await orchestrator.launchAgent({
  name: 'Analyzer',
  type: 'plan',
});

// Wait for completion
await orchestrator.waitForAgent(parent.id);

// Launch child agents when parent completes
const child1 = await orchestrator.launchAgent({
  name: 'Bug Finder',
  type: 'code-reviewer',
  parentId: parent.id,
});

const child2 = await orchestrator.launchAgent({
  name: 'Code Generator',
  type: 'code-generator',
  parentId: parent.id,
});
```

## Status Types

```typescript
type AgentStatus =
  | 'idle'       // Not yet started
  | 'queued'     // Waiting to run
  | 'running'    // Currently executing
  | 'completed'  // Finished successfully
  | 'failed'     // Execution failed
  | 'paused'     // Paused by user
  | 'cancelled'  // Cancelled by user
```

## Agent Types

```typescript
type AgentType =
  | 'general-purpose'    // General task agent
  | 'code-reviewer'      // Code review specialization
  | 'plan'               // Planning & architecture
  | 'explore'            // Code exploration
  | 'custom'             // Custom agent
  | 'bug-finder'         // Bug detection
  | 'code-generator'     // Code generation
  | 'tester'             // Testing specialization
  | 'architect'          // Architecture design
```

## Styling

The component uses:
- **Tailwind CSS** for responsive design
- **Dark theme** (slate-800/900 background)
- **Lucide icons** for UI elements
- **SVG** for graph visualization

Customize by editing CSS classes in `AgentOrchestrator.tsx`

## Example Implementation

### Go Backend Handler

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/PratikDhanave/agentic-software-factory-go/pkg/agents"
)

type LaunchAgentRequest struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    Description string   `json:"description"`
    ParentID    string   `json:"parentId"`
    Tags        []string `json:"tags"`
    Timeout     int      `json:"timeout"`
}

func (h *Handler) LaunchAgent(w http.ResponseWriter, r *http.Request) {
    var req LaunchAgentRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Create agent instance
    agent, err := h.orchestrator.LaunchAgent(r.Context(), agents.AgentConfig{
        Name:        req.Name,
        Type:        req.Type,
        Description: req.Description,
        ParentID:    req.ParentID,
        Tags:        req.Tags,
        Timeout:     req.Timeout,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(agent)
}

func (h *Handler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
    agentID := r.PathValue("agentId")

    agent, err := h.orchestrator.GetAgent(r.Context(), agentID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(agent)
}
```

## Next Steps

1. **Implement backend endpoints** following the API specification
2. **Add WebSocket support** for real-time updates
3. **Connect to your agent execution system** (DAG executor, etc.)
4. **Add authentication** if needed
5. **Implement agent persistence** (database storage)
6. **Add filtering/search** to agent list
7. **Export agent logs/results**

## Performance Considerations

- **Graph rendering:** Currently supports ~50 agents before performance degradation
- **WebSocket updates:** Rate-limit to 100-500ms intervals to prevent UI thrashing
- **Agent list:** Paginate after 100 agents
- **SVG positioning:** Use memoization to avoid recalculating positions on every render

## Browser Compatibility

- Chrome/Edge: ✅ Full support
- Firefox: ✅ Full support
- Safari: ✅ Full support
- Mobile browsers: ⚠️ Responsive but may have layout issues with large graphs

## Troubleshooting

**Graph not updating:**
- Check WebSocket connection in browser DevTools
- Verify backend is sending status updates
- Check browser console for errors

**Agents not launching:**
- Verify backend endpoints are implemented
- Check CORS configuration if running on different domain
- Look for 404/500 errors in browser Network tab

**Performance issues:**
- Reduce number of agents displayed
- Increase WebSocket update interval
- Use virtual scrolling for agent list

## Related Documentation

- [DAG Workflow Engine](DAG_WORKFLOW_ENGINE.md)
- [State Machines](EXPLICIT_STATE_MACHINES.md)
- [Multi-Agent Coordination](PHASE4_MULTIAGENT_COORDINATOR.md)
