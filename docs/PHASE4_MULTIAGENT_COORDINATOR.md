# Phase 4: Multi-Agent Coordinator Design
## Intelligent Workflow Generation & Decision Making

---

## Overview

**Phase 4 Goal**: Build the AI-DLC layer that generates and coordinates multi-agent workflows

**Scope**:
- Multi-agent orchestration framework
- Workflow generation from requirements
- Knowledge graph integration
- Agent role system with LLM-driven decision making
- Interactive prompting & collaboration
- State tracking across agent interactions

**Integration Points**:
- Input: Trigger events (GitHub issues, manual requests)
- Output: Workflow YAML definitions → Phase 1-3 DAG Engine
- Observability: Logging & tracing from Phase 3
- Persistence: PostgreSQL from Phase 3

---

## Architecture: 5 Core Components

```
┌─────────────────────────────────────────────────────────┐
│          Agent Coordinator (Phase 4)                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 1. AGENT ROLE SYSTEM                           │   │
│  │  - Define agent archetypes                      │   │
│  │  - Capability mapping                           │   │
│  │  - LLM model selection per role                 │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 2. WORKFLOW GENERATOR                          │   │
│  │  - Parse requirements                           │   │
│  │  - Generate YAML workflow definitions           │   │
│  │  - Validate workflow structure                  │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 3. DECISION ENGINE                             │   │
│  │  - Agent decides next steps                     │   │
│  │  - Prioritize work items                        │   │
│  │  - Choose execution strategy                    │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 4. KNOWLEDGE GRAPH                             │   │
│  │  - Store decisions & outcomes                   │   │
│  │  - Track team expertise                         │   │
│  │  - Learn from past executions                   │   │
│  └─────────────────────────────────────────────────┘   │
│                           ↓                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 5. INTERACTION MANAGER                         │   │
│  │  - Human-in-the-loop prompts                    │   │
│  │  - Agent collaboration                          │   │
│  │  - Session state management                     │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
└─────────────────────────────────────────────────────────┘
              ↓
    Phase 1-3 DAG Engine
```

---

## Component 1: Agent Role System

### Core Agent Types

```go
type AgentRole string

const (
    // Strategic agents
    ProductAgent    AgentRole = "product"      // Understand requirements
    ArchitectAgent  AgentRole = "architect"    // System design decisions
    
    // Implementation agents
    DesignAgent     AgentRole = "design"       // Architecture & design patterns
    DeveloperAgent  AgentRole = "developer"    // Code implementation
    SecurityAgent   AgentRole = "security"     // Security decisions & scanning
    
    // Quality agents
    QAAgent         AgentRole = "qa"           // Test strategy & verification
    PerformanceAgent AgentRole = "performance" // Performance optimization
    
    // Operations agents
    DevOpsAgent     AgentRole = "devops"       // Deployment & infrastructure
    ComplianceAgent AgentRole = "compliance"   // Compliance & audit
    
    // Coordination agents
    CoordinatorAgent AgentRole = "coordinator" // Workflow generation
)
```

### Agent Capabilities Mapping

```go
type AgentCapabilities struct {
    Role              AgentRole
    Title             string
    Description       string
    Capabilities      []string    // What this agent can do
    LLMModel          string      // Claude 3.5 Sonnet, etc
    ContextWindow     int         // tokens
    ResponseFormat    string      // JSON, text, structured
    KnowledgeDomains  []string    // e.g., ["security", "authentication"]
    DecisionAuthority string      // Can make decisions or only suggest?
    InteractionMode   string      // "autonomous", "interactive", "advisory"
}

// Example: Product Agent
var ProductAgentCaps = AgentCapabilities{
    Role: ProductAgent,
    Title: "Product Manager Agent",
    Description: "Understands requirements, prioritizes work, tracks metrics",
    Capabilities: []string{
        "parse-requirements",
        "prioritize-work",
        "identify-stakeholders",
        "suggest-timelines",
    },
    LLMModel: "claude-3-5-sonnet",
    ContextWindow: 100000,
    ResponseFormat: "json",
    KnowledgeDomains: []string{"product", "roadmap", "metrics"},
    DecisionAuthority: "suggest",
    InteractionMode: "interactive",
}
```

---

## Component 2: Workflow Generator

### Input: Trigger Event

```go
type TriggerEvent struct {
    Type       string                 // "github.issue", "manual", "schedule"
    Source     string                 // "github", "api", "scheduled_task"
    Data       map[string]interface{} // Raw event data
    Context    ExecutionContext       // Who, when, budget
}

// Example GitHub Issue Trigger
{
  "type": "github.issue",
  "source": "github",
  "data": {
    "action": "opened",
    "issue": {
      "number": 42,
      "title": "Fix authentication bug",
      "body": "Users cannot login with OAuth...",
      "labels": ["bug", "critical", "auth"],
      "assignee": null
    },
    "repository": "myorg/myrepo"
  },
  "context": {
    "user": "developer@acme.com",
    "budget": 1000,  // tokens
    "urgency": "high"
  }
}
```

### Workflow Generation Process

```
1. ANALYSIS
   Trigger Event → CoordinatorAgent
   
   "Given this GitHub issue:
   - Issue #42: Fix authentication bug
   - Critical severity
   - OAuth failure
   
   What workflow should we run?"
   
2. AGENT CONSULTATION
   CoordinatorAgent → Consults:
   - ProductAgent: "What's the priority?"
   - ArchitectAgent: "What's the design impact?"
   - SecurityAgent: "What's the security risk?"
   
3. WORKFLOW GENERATION
   CoordinatorAgent generates YAML:
   ```yaml
   name: "critical-auth-bug-fix"
   triggers: ["github.issue"]
   
   jobs:
     analyze:
       task: "analyze-authentication"
       agent: "security"
       input: "{{ trigger.issue }}"
     
     design_fix:
       task: "design-fix"
       agent: "architect"
       depends_on: ["analyze"]
       input: "{{ jobs.analyze.output }}"
     
     implement:
       task: "implement-fix"
       agent: "developer"
       depends_on: ["design_fix"]
       input: "{{ jobs.design_fix.output }}"
     
     test:
       task: "run-security-tests"
       agent: "qa"
       depends_on: ["implement"]
       input: "{{ jobs.implement.output }}"
     
     deploy:
       task: "deploy-fix"
       agent: "devops"
       depends_on: ["test"]
       input: "{{ jobs.test.output }}"
   ```

4. VALIDATION
   - DAGBuilder validates YAML
   - Checks for cycles
   - Verifies agent capabilities

5. EXECUTION
   → Phase 1-3 DAG Engine executes workflow
```

### Implementation

```go
type WorkflowGenerator struct {
    agentFactory     *AgentFactory
    knowledgeGraph   *KnowledgeGraph
    templates        *WorkflowTemplates
    logger           Logger
}

func (wg *WorkflowGenerator) Generate(ctx context.Context, event TriggerEvent) (*WorkflowDefinition, error) {
    // 1. Classify the event
    classification := wg.classifyEvent(event)
    
    // 2. Select workflow template
    template := wg.templates.Find(classification)
    
    // 3. Consult agents for customization
    decisions := wg.consultAgents(ctx, event, template)
    
    // 4. Generate YAML
    workflow := wg.render(template, decisions)
    
    // 5. Validate
    if err := workflow.Validate(); err != nil {
        return nil, err
    }
    
    return workflow, nil
}

func (wg *WorkflowGenerator) consultAgents(ctx context.Context, event TriggerEvent, template *WorkflowTemplate) map[string]interface{} {
    decisions := make(map[string]interface{})
    
    // For each agent type in template
    for _, agentRole := range template.RequiredAgents {
        agent := wg.agentFactory.Create(agentRole)
        
        decision, err := agent.Decide(ctx, DecisionRequest{
            Event:    event,
            Template: template,
            Context:  decisions,  // Prior decisions for context
        })
        
        decisions[string(agentRole)] = decision
    }
    
    return decisions
}
```

---

## Component 3: Decision Engine

### Agent Decision Framework

```go
type DecisionRequest struct {
    Event    TriggerEvent
    Template *WorkflowTemplate
    Context  map[string]interface{} // Prior decisions
    Query    string                   // Specific question
}

type Decision struct {
    Choice      string                 // The decision made
    Reasoning   string                 // Why this choice
    Confidence  float64                // 0.0-1.0 confidence
    Alternatives []string              // Other considered options
    Impact      map[string]interface{} // Expected impact
}

// Example: SecurityAgent decides on security testing level
func (sa *SecurityAgent) Decide(ctx context.Context, req DecisionRequest) (*Decision, error) {
    prompt := fmt.Sprintf(`
    You are a Security Decision Agent.
    
    Event: %s
    Current workflow template: %s
    Prior decisions: %v
    
    Decision: What security testing level should we apply?
    - Options: basic, standard, intensive
    - Return JSON with choice, reasoning, confidence (0-1), alternatives, and expected impact
    `, req.Event, req.Template, req.Context)
    
    response, err := sa.llm.Call(ctx, prompt)
    // Parse JSON response into Decision struct
    
    return parseDecision(response), nil
}
```

### Interactive Decision Making

```go
type InteractiveDecision struct {
    Agent      AgentRole
    Question   string
    Options    []string
    Current    string
    Reasoning  string
    
    // For human intervention
    NeedsApproval bool
    Approver      string
}

// When agent confidence < threshold, ask human
if decision.Confidence < 0.7 {
    interactiveDecision := InteractiveDecision{
        Agent:         sa.Role,
        Question:      "What security level?",
        Options:       []string{"basic", "standard", "intensive"},
        Current:       decision.Choice,
        Reasoning:     decision.Reasoning,
        NeedsApproval: true,
        Approver:      "security-lead",
    }
    
    // Ask human via API/UI
    approval := promptUser(interactiveDecision)
    
    // Update decision if human disagrees
    if approval.Decision != decision.Choice {
        decision.Choice = approval.Decision
        decision.Reasoning = approval.Reasoning
    }
}
```

---

## Component 4: Knowledge Graph

### Knowledge Model

```go
type KnowledgeNode struct {
    ID         string              // Unique ID
    Type       string              // "decision", "pattern", "expertise", "outcome"
    Agent      AgentRole           // Which agent created this
    Content    string              // The knowledge
    Timestamp  time.Time
    Metadata   map[string]interface{}
}

type KnowledgeEdge struct {
    From       string              // Source node ID
    To         string              // Target node ID
    Relation   string              // "informs", "refutes", "extends", "depends_on"
    Strength   float64             // 0.0-1.0 edge weight
}

type KnowledgeGraph struct {
    Nodes map[string]*KnowledgeNode
    Edges []KnowledgeEdge
    db    GraphDatabase
}
```

### Examples of Knowledge

```
Node: "auth-oauth-vulnerable-pattern"
Type: "pattern"
Agent: SecurityAgent
Content: "OAuth token refresh timing issues cause race conditions"
Timestamp: 2026-11-18

Node: "team-alice-expertise-oauth"
Type: "expertise"
Agent: ProductAgent
Content: "Alice has deep OAuth expertise, should lead auth decisions"
Timestamp: 2026-11-10

Edge: "auth-oauth-vulnerable-pattern" → "implement-oauth-fix"
Relation: "informs"
Strength: 0.95

Node: "oauth-fix-attempt-1-failed"
Type: "outcome"
Agent: DeveloperAgent
Content: "First OAuth fix attempt failed security tests - race condition still present"
Timestamp: 2026-11-18

Edge: "oauth-fix-attempt-1-failed" → "auth-oauth-vulnerable-pattern"
Relation: "confirms"
Strength: 0.98
```

### Learning Loop

```go
func (kg *KnowledgeGraph) LearnFromExecution(execution *WorkflowExecution) error {
    // 1. Extract decisions made
    decisions := extractDecisions(execution)
    
    // 2. Extract outcomes
    outcomes := extractOutcomes(execution)
    
    // 3. For each decision + outcome
    for decision, outcome := range outcomes {
        // 4. Create/update knowledge nodes
        decisionNode := kg.createNode(KnowledgeNode{
            Type:    "decision",
            Agent:   decision.Agent,
            Content: decision.Choice,
        })
        
        outcomeNode := kg.createNode(KnowledgeNode{
            Type:    "outcome",
            Agent:   "system",
            Content: outcome.Result,
        })
        
        // 5. Create edge: decision → outcome
        // Calculate strength based on correlation
        strength := calculateCorrelation(decision, outcome)
        
        kg.addEdge(decisionNode.ID, outcomeNode.ID, "leads_to", strength)
    }
    
    return nil
}
```

### Querying Knowledge

```go
// When making a new decision, query relevant knowledge
func (kg *KnowledgeGraph) QueryRelevant(agent AgentRole, query string) []*KnowledgeNode {
    // Find nodes relevant to agent's role + query
    // Weight by recency, strength of connections, outcome success rate
    
    // Returns ranked list of relevant knowledge
}

// Example: Before SecurityAgent decides on testing level
relatedKnowledge := kg.QueryRelevant(SecurityAgent, "OAuth security testing")
// Returns: [
//   {pattern: "race condition in token refresh", strength: 0.98},
//   {pattern: "timing-based OAuth attacks", strength: 0.92},
//   {expertise: "Alice knows OAuth best", strength: 0.95},
// ]
```

---

## Component 5: Interaction Manager

### Session State

```go
type CoordinationSession struct {
    ID                 string
    TriggerEvent       TriggerEvent
    Workflow           *WorkflowDefinition
    Agents             []*AgentExecution
    Decisions          map[string]*Decision
    HumanInteractions  []InteractiveDecision
    Status             SessionStatus
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type SessionStatus string
const (
    SessionPending     SessionStatus = "pending"      // Waiting for user to start
    SessionCoordinating SessionStatus = "coordinating" // Agents deciding
    SessionWaitingApproval SessionStatus = "waiting"  // Need human approval
    SessionExecuting   SessionStatus = "executing"    // DAG engine running
    SessionCompleted   SessionStatus = "completed"
    SessionFailed      SessionStatus = "failed"
)
```

### Interactive Collaboration

```
Workflow Generation → Coordination → Approval → Execution

Step 1: AI-DLC generates workflow
"Recommended workflow for auth bug:
 - Security analysis (SecurityAgent)
 - Architecture design (ArchitectAgent)
 - Implementation (DeveloperAgent)"

Step 2: Human reviews proposed workflow
✓ Approve
✗ Modify
✗ Reject

Step 3: If modify, interactive decision
"Agent confidence on testing level: 65%
 Recommendation: standard
 Your preference? basic / standard / intensive"

Step 4: Updated workflow → DAG Engine execution
```

---

## Data Flow: Complete Example

```
1. GitHub Issue Created
   "Fix authentication bug" → WebhookHandler
   
2. Coordinator Consultation
   Event → CoordinatorAgent
   
   "What workflow?"
   ↓ (consults agents)
   
   ProductAgent: "Critical priority"
   ArchitectAgent: "Design review needed"
   SecurityAgent: "Security testing required"
   DeveloperAgent: "3-hour estimate"
   
3. Workflow Generation
   YAML generated:
   ```
   jobs:
     security_analysis:
       agent: security
     design_review:
       agent: architect
       depends_on: [security_analysis]
     implement:
       agent: developer
       depends_on: [design_review]
     security_tests:
       agent: qa
       depends_on: [implement]
     deploy:
       agent: devops
       depends_on: [security_tests]
   ```

4. Approval Gate
   Workflow presented to approver (product lead)
   "5-job workflow, 4-hour estimate, critical urgency"
   → Approve / Modify

5. Knowledge Integration
   Prior knowledge added to agent prompts:
   - "Previous OAuth fix failed due to: X"
   - "Team expertise: Alice led last auth effort"
   - "Pattern: timing-based race conditions common"

6. Execution (Phase 1-3 DAG Engine)
   Level 0: [security_analysis]
   Level 1: [design_review]
   Level 2: [implement]
   Level 3: [security_tests]
   Level 4: [deploy]

7. Learning Update
   Outcomes recorded → Knowledge graph updated
   Future workflows benefit from this experience
```

---

## Implementation Roadmap: Phase 4

### Phase 4.1: Foundations (Weeks 1-2)
- [ ] AgentRole system & capabilities mapping
- [ ] AgentFactory for creating agent instances
- [ ] LLM integration layer (Claude API calls)
- [ ] Basic decision interface

### Phase 4.2: Workflow Generation (Weeks 3-4)
- [ ] Workflow templates library
- [ ] Coordinator agent implementation
- [ ] YAML generation from decisions
- [ ] DAG validation integration (Phase 1)

### Phase 4.3: Knowledge System (Weeks 5-6)
- [ ] Knowledge graph data model
- [ ] Node/edge creation & queries
- [ ] Learning loop from executions
- [ ] Relevance ranking algorithm

### Phase 4.4: Interaction & UI (Weeks 7-8)
- [ ] Interactive decision prompts
- [ ] Session state management
- [ ] API endpoints for coordination
- [ ] WebUI for workflow approval

### Phase 4.5: Integration & Polish (Weeks 9-10)
- [ ] End-to-end integration tests
- [ ] Performance optimization
- [ ] Observability instrumentation
- [ ] Documentation & examples

---

## Success Criteria

**Functionality**
- ✅ Generate workflows from trigger events
- ✅ Coordinate multi-agent decisions
- ✅ Support interactive approval gates
- ✅ Learn from past executions

**Performance**
- ✅ Workflow generation < 500ms
- ✅ Support 10+ concurrent coordinations
- ✅ Knowledge graph queries < 50ms

**Quality**
- ✅ Agent decisions logged & auditable
- ✅ 95%+ workflow approval rate
- ✅ Zero hallucinations in YAML generation

---

## Integration Points

**Input From**:
- Trigger events (GitHub, API, schedules)
- Human approvals (interactive decisions)
- Kora control plane (identity, policy)

**Output To**:
- Phase 1-3 DAG Engine (YAML workflows)
- Observability (logging, tracing)
- Persistence (PostgreSQL)

**Reads From**:
- Knowledge graph (prior patterns, expertise)
- Agent capabilities (what agents can do)
- Workflow templates (common patterns)

---

## Next Steps

1. Create Agent abstraction & LLM integration layer
2. Implement Workflow Generator with agent consultation
3. Build Knowledge Graph with learning loop
4. Add interactive decision UI
5. Integration tests with Phase 1-3 components

This completes your **4-layer Factory Stack** →

Layer 1: Integration ✅ (Phase 3)
Layer 2: Execution ✅ (Phase 1-3 DAG Engine)
Layer 3: Coordination ⟵ (Phase 4 NOW)
Layer 4: Governance → (Phase 5: Kora)
