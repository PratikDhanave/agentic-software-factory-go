const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export interface WorkflowGenerationRequest {
  issue_title: string;
  issue_description: string;
  issue_url?: string;
}

export interface Decision {
  agent_role: string;
  decision: string;
  confidence: number;
  reasoning: string;
}

export interface WorkflowResponse {
  workflow_id: string;
  status: string;
  decisions: Decision[];
  workflow_yaml: string;
  timestamp: string;
}

export interface PendingApproval {
  id: string;
  workflow_id: string;
  agent_role: string;
  decision: string;
  confidence: number;
  created_at: string;
}

export interface ApprovalResponse {
  pending_approvals: PendingApproval[];
}

export interface SessionStatus {
  session_id: string;
  status: string;
  workflow_id: string;
  created_at: string;
  updated_at: string;
  decisions_count: number;
  approvals_pending: number;
}

export interface AuditEntry {
  id: string;
  principal: string;
  action: string;
  resource: string;
  resource_id: string;
  status: string;
  timestamp: string;
  error?: string;
}

class APIClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl = API_BASE;
  }

  async generateWorkflow(req: WorkflowGenerationRequest): Promise<WorkflowResponse> {
    const response = await fetch(`${this.baseUrl}/workflows/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });

    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async getPendingApprovals(): Promise<PendingApproval[]> {
    const response = await fetch(`${this.baseUrl}/approvals/pending`);
    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    const data = await response.json();
    return data.pending_approvals || [];
  }

  async submitApproval(
    approval_id: string,
    approved: boolean,
    override?: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await fetch(`${this.baseUrl}/approvals/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approval_id, approved, override }),
    });

    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async getSessionStatus(session_id: string): Promise<SessionStatus> {
    const response = await fetch(`${this.baseUrl}/sessions/status?session_id=${session_id}`);
    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async getAuditTrail(): Promise<AuditEntry[]> {
    const response = await fetch(`${this.baseUrl}/audit/trail`);
    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    const data = await response.json();
    return data.entries || [];
  }
}

export const apiClient = new APIClient();
