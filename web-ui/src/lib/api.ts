function normalizeApiBase(raw?: string): string {
  if (!raw) return '';
  return raw.trim().replace(/\/+$/, '').replace(/\/api(\/v1)?$/, '');
}

export const API_BASE = normalizeApiBase(process.env.NEXT_PUBLIC_API_URL);

export type FactoryPhase = 'crawl' | 'walk' | 'run';

export interface AgentCapabilitySummary {
  role: string;
  title: string;
  description: string;
  capabilities: string[];
  decision_authority: string;
  interaction_mode: string;
  domains?: string[];
}

export interface WorkerInfo {
  id: string;
  status: string;
  task_type?: string;
  task_spec?: string;
}

export interface FactoryStatusPayload {
  workers_total: number;
  workers_busy: number;
  workers_idle: number;
  workers: WorkerInfo[];
  tasks_pending: number;
  tasks_completed: number;
  tasks_failed: number;
  max_workers: number;
  llm_provider?: string;
  phase?: FactoryPhase;
}

export interface GatewayProviderStatus {
  id: string;
  name: string;
  model: string;
  configured: boolean;
  active: boolean;
  masked_key?: string;
  env_var: string;
}

export interface SystemOverview {
  status: string;
  uptime: string;
  phase: FactoryPhase;
  llm_provider: string;
  gateway_mode?: string;
  gateway_providers?: GatewayProviderStatus[];
  persistence: string;
  db_path: string;
  specs_count: number;
  agents_count: number;
  agents: AgentCapabilitySummary[];
  pending_approvals: number;
  factory: FactoryStatusPayload;
  database: Record<string, any>;
}

export interface WorkflowDecisionItem {
  id?: string;
  decision_uuid?: string;
  agent_role: string;
  decision: string;
  choice?: string;
  reasoning: string;
  confidence: number;
  alternatives?: string[];
  needs_approval?: boolean;
  approval_status?: string;
}

export interface WorkflowGenerateRequest {
  issue_title: string;
  issue_description: string;
  trigger_type?: string;
  classification?: string;
  approval_required?: boolean;
  auto_execute?: boolean;
}

export interface WorkflowResponse {
  session_id?: string;
  workflow_id: string;
  workflow_name: string;
  status?: string;
  phase?: FactoryPhase;
  classification?: string;
  requires_approval?: boolean;
  decisions: WorkflowDecisionItem[];
  pending_approvals?: string[];
  dag_levels?: string[][];
  jobs?: Record<string, { Task: string; DependsOn: string[]; Timeout: number }>;
  workflow_yaml?: string;
  execution?: any;
}

export interface PendingApproval {
  id?: string;
  session_id: string;
  workflow_id?: string;
  issue_title?: string;
  agent_role: string;
  decision?: string;
  current_choice?: string;
  reasoning?: string;
  confidence: number;
  alternatives?: string[];
  options?: string[];
  requires_action?: boolean;
  status?: string;
  created_at?: string;
}

export interface PendingApprovalsResponse {
  approvals: PendingApproval[];
  pending_approvals?: PendingApproval[];
  total_pending: number;
}

export interface SubmitApprovalPayload {
  approval_id?: string;
  session_id?: string;
  approved: boolean;
  approver?: string;
  comments?: string;
  override?: string;
}

export interface SpecSummary {
  id: string;
  title: string;
  version: string;
  description?: string;
  goal?: string;
  steps_count?: number;
  constraints?: Array<{ id: string; type: string; description: string }>;
  steps?: Array<{ id: string; name: string; description: string; depends_on?: string[] }>;
}

export interface FactoryTask {
  id: string;
  type: string;
  spec_id: string;
  assigned_to?: string;
  status: string;
  terminal: boolean;
  retries: number;
  result?: any;
  error?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface SelfImprovementAgentMetric {
  SuccessRate: number;
  Confidence: number;
  DecisionCount: number;
}

export interface SelfImprovementPattern {
  IssueType: string;
  Agent: string;
  SuccessRate: number;
  Applied: number;
}

export interface SelfImprovementMetrics {
  Agents: Record<string, SelfImprovementAgentMetric>;
  Patterns: SelfImprovementPattern[];
  Timestamp: string;
  Phase?: FactoryPhase;
  Database?: Record<string, any>;
}

export interface StateMachineExport {
  name: string;
  category?: string;
  description?: string;
  initial: string;
  states: string[];
  terminal?: string[];
  transitions: Record<string, string[]>;
  state_descriptions?: Record<string, string>;
  mermaid?: string;
}

export interface StateMachineValidationResult {
  machine: string;
  from: string;
  to?: string;
  allowed?: boolean;
  error?: string;
  inspection: {
    machine: string;
    category?: string;
    state: string;
    description?: string;
    is_valid: boolean;
    is_initial: boolean;
    is_terminal: boolean;
    allowed_next: string[];
  };
}

export interface FactoryRunRecord {
  id: string;
  title: string;
  description: string;
  spec_id: string;
  session_id: string;
  phase_mode: string;
  stage: string;
  status: string;
  workspace_dir: string;
  promoted_dir?: string;
  payload: Record<string, any>;
  error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

class APIClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl = API_BASE;
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...((init?.headers as Record<string, string>) || {}),
    };
    const res = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers,
    });
    if (!res.ok) {
      let msg = `API error (${res.status})`;
      try {
        const body = await res.json();
        if (body?.error) msg = body.error;
      } catch {
        // ignore parse error
      }
      throw new Error(msg);
    }
    return res.json();
  }

  async getSystemOverview(): Promise<SystemOverview> {
    return this.request<SystemOverview>('/api/system/overview');
  }

  async setSystemPhase(phase: FactoryPhase): Promise<{ phase: FactoryPhase; message: string }> {
    return this.request('/api/system/phase', {
      method: 'POST',
      body: JSON.stringify({ phase }),
    });
  }

  async generateWorkflow(req: WorkflowGenerateRequest): Promise<WorkflowResponse> {
    return this.request<WorkflowResponse>('/api/v1/workflows/generate', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async executeWorkflow(workflowId: string, sessionId?: string): Promise<any> {
    return this.request('/api/v1/workflows/execute', {
      method: 'POST',
      body: JSON.stringify({ workflow_id: workflowId, session_id: sessionId }),
    });
  }

  async getSessions(): Promise<{ sessions: any[] }> {
    return this.request('/api/v1/sessions');
  }

  async getPendingApprovals(): Promise<PendingApprovalsResponse> {
    return this.request<PendingApprovalsResponse>('/api/v1/approvals/pending');
  }

  async submitApproval(
    payloadOrId: SubmitApprovalPayload | string,
    approved?: boolean,
    override?: string
  ): Promise<{ session_id: string; status: string; message: string; execution?: any }> {
    const payload: SubmitApprovalPayload =
      typeof payloadOrId === 'string'
        ? { approval_id: payloadOrId, approved: Boolean(approved), override }
        : payloadOrId;

    return this.request('/api/v1/approvals/submit', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getSpecs(): Promise<{ specs: SpecSummary[] }> {
    return this.request('/api/specs');
  }

  async getSpecDetail(id: string): Promise<{ spec: any; prompt: string }> {
    return this.request(`/api/specs/${encodeURIComponent(id)}`);
  }

  async createSpec(specPayload: Record<string, any>): Promise<{ spec: any; file: string }> {
    return this.request('/api/specs', {
      method: 'POST',
      body: JSON.stringify(specPayload),
    });
  }

  async getFactoryStatus(): Promise<FactoryStatusPayload> {
    return this.request('/api/factory/status');
  }

  async launchTask(action: string, specId: string, priority = 1): Promise<{ task_id: string; status: string }> {
    return this.request(`/api/factory/launch/${encodeURIComponent(action)}`, {
      method: 'POST',
      body: JSON.stringify({ spec_id: specId, priority }),
    });
  }

  async getTasks(): Promise<{ tasks: FactoryTask[] }> {
    return this.request('/api/tasks');
  }

  async getTask(id: string): Promise<FactoryTask> {
    return this.request(`/api/tasks/${encodeURIComponent(id)}`);
  }

  async runVerification(): Promise<{ success: boolean; verification: any; error: string }> {
    return this.request('/api/factory/verify', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async getSelfImprovementMetrics(): Promise<SelfImprovementMetrics> {
    return this.request('/api/v1/metrics/self-improvement');
  }

  async simulateLearningBatch(count = 25): Promise<{ generated: number; patterns_found: number }> {
    return this.request('/api/v1/learning/simulate', {
      method: 'POST',
      body: JSON.stringify({ count }),
    });
  }

  async triggerLearning(): Promise<{ success: boolean; patterns_found: number }> {
    return this.request('/api/v1/learning/learn', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async getStateMachines(category?: string): Promise<{ count?: number; machines: StateMachineExport[] }> {
    const q = category ? `?category=${encodeURIComponent(category)}` : '';
    return this.request(`/api/state-machines${q}`);
  }

  async validateStateTransition(
    machine: string,
    from: string,
    to?: string
  ): Promise<StateMachineValidationResult> {
    return this.request('/api/state-machines/validate', {
      method: 'POST',
      body: JSON.stringify({ machine, from, to }),
    });
  }

  async listFactoryRuns(): Promise<{ runs: FactoryRunRecord[] }> {
    return this.request('/api/v1/factory/runs');
  }

  async getFactoryRun(id: string): Promise<FactoryRunRecord> {
    return this.request(`/api/v1/factory/runs/${encodeURIComponent(id)}`);
  }

  async startFactoryRun(payload: {
    title?: string;
    description?: string;
    spec_id?: string;
    constraints?: string[];
    force_approval?: boolean;
    sync?: boolean;
  }): Promise<FactoryRunRecord> {
    return this.request('/api/v1/factory/runs', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveFactoryRun(id: string, approver = 'human-operator'): Promise<FactoryRunRecord> {
    return this.request(`/api/v1/factory/runs/${encodeURIComponent(id)}/approve`, {
      method: 'POST',
      body: JSON.stringify({ approver }),
    });
  }

  async promoteFactoryRun(id: string, targetSubdir = ''): Promise<FactoryRunRecord> {
    return this.request(`/api/v1/factory/runs/${encodeURIComponent(id)}/promote`, {
      method: 'POST',
      body: JSON.stringify({ target_subdir: targetSubdir }),
    });
  }

  async getGatewayConfig(): Promise<{
    preferred_provider: string;
    active_provider: string;
    providers: GatewayProviderStatus[];
  }> {
    return this.request('/api/v1/gateway/config');
  }

  async updateGatewayConfig(payload: {
    preferred_provider?: string;
    gemini_api_key?: string;
    gemini_model?: string;
    anthropic_api_key?: string;
    anthropic_model?: string;
    openai_api_key?: string;
    openai_model?: string;
    ollama_url?: string;
    ollama_model?: string;
  }): Promise<{
    preferred_provider: string;
    active_provider: string;
    providers: GatewayProviderStatus[];
    message?: string;
  }> {
    return this.request('/api/v1/gateway/config', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async consultAgent(
    role: string,
    prompt: string
  ): Promise<{
    role: string;
    provider: string;
    model: string;
    content: string;
    tokens: number;
    duration_ms: number;
  }> {
    return this.request('/api/v1/agents/consult', {
      method: 'POST',
      body: JSON.stringify({ role, prompt }),
    });
  }
}

export const apiClient = new APIClient();
