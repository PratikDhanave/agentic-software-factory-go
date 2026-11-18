const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

// Phase 2 Types
export interface AgentDecision {
  id: string;
  agent_role: string;
  decision: string;
  confidence: number;
  reasoning: string;
  requires_approval: boolean;
  approval_status: string;
  created_at: string;
}

export interface Metrics {
  Agents: Record<string, {
    total_decisions: number;
    approved: number;
    rejected: number;
    success_rate: number;
    confidence_avg: number;
  }>;
  Patterns: Array<{
    pattern_id: string;
    description: string;
    frequency: number;
    confidence: number;
  }>;
}

export interface SimulateRequest {
  count: number;
}

export interface SimulateResponse {
  decisions: AgentDecision[];
  total_generated: number;
}

export interface DatabaseStats {
  decisions: number;
  approvals: number;
  patterns: number;
  metrics: number;
}

class APIClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl = API_BASE;
  }

  async generateDecisions(count: number): Promise<SimulateResponse> {
    const response = await fetch(`${this.baseUrl}/simulate/batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count }),
    });

    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async getMetrics(): Promise<Metrics> {
    const response = await fetch(`${this.baseUrl}/metrics`);
    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async approveLast(): Promise<{ success: boolean; message: string }> {
    const response = await fetch(`${this.baseUrl}/metrics/approve-last`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async triggerLearning(): Promise<{ success: boolean; patterns_found: number }> {
    const response = await fetch(`${this.baseUrl}/metrics/learn`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }

  async getDatabaseStats(): Promise<DatabaseStats> {
    const response = await fetch(`${this.baseUrl}/metrics/database`);
    if (!response.ok) throw new Error(`API error: ${response.statusText}`);
    return response.json();
  }
}

export const apiClient = new APIClient();
