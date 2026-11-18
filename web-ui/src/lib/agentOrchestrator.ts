/**
 * Agent Orchestrator utilities for launching and monitoring agents
 * Supports WebSocket connection for real-time status updates
 */

export interface AgentConfig {
  name: string;
  type: AgentType;
  description?: string;
  parentId?: string;
  tags?: string[];
  timeout?: number;
}

export interface AgentStatus {
  id: string;
  name: string;
  type: AgentType;
  status: 'idle' | 'queued' | 'running' | 'completed' | 'failed' | 'paused' | 'cancelled';
  progress: number;
  startTime?: Date;
  endTime?: Date;
  output?: string;
  error?: string;
  parentId?: string;
  childIds?: string[];
  metadata?: Record<string, any>;
}

export interface AgentConnection {
  from: string;
  to: string;
  label?: string;
  dependencyType?: 'sequential' | 'parallel' | 'conditional';
}

export type AgentType =
  | 'general-purpose'
  | 'code-reviewer'
  | 'plan'
  | 'explore'
  | 'custom'
  | 'bug-finder'
  | 'code-generator'
  | 'tester'
  | 'architect';

export class AgentOrchestrator {
  private baseUrl: string;
  private wsUrl: string;
  private ws: WebSocket | null = null;
  private messageHandlers: Map<string, (data: any) => void> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(baseUrl: string = 'http://localhost:3000', wsUrl: string = 'ws://localhost:3000') {
    this.baseUrl = baseUrl;
    this.wsUrl = wsUrl;
  }

  /**
   * Connect to the agent orchestrator via WebSocket
   */
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(`${this.wsUrl}/ws/agents`);

        this.ws.onopen = () => {
          console.log('Connected to agent orchestrator');
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            const { type, agentId, payload } = data;

            // Handle generic updates
            if (this.messageHandlers.has(type)) {
              this.messageHandlers.get(type)?.(payload);
            }

            // Handle agent-specific updates
            if (agentId && this.messageHandlers.has(`agent:${agentId}`)) {
              this.messageHandlers.get(`agent:${agentId}`)?.(payload);
            }
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };

        this.ws.onclose = () => {
          console.log('Disconnected from agent orchestrator');
          this.attemptReconnect();
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Attempt to reconnect to WebSocket
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      console.log(`Reconnecting in ${delay}ms... (attempt ${this.reconnectAttempts})`);

      setTimeout(() => {
        this.connect().catch((error) => {
          console.error('Reconnection failed:', error);
        });
      }, delay);
    }
  }

  /**
   * Subscribe to agent events
   */
  subscribe(eventType: string, handler: (data: any) => void): () => void {
    this.messageHandlers.set(eventType, handler);

    // Return unsubscribe function
    return () => {
      this.messageHandlers.delete(eventType);
    };
  }

  /**
   * Launch a new agent
   */
  async launchAgent(config: AgentConfig): Promise<AgentStatus> {
    const response = await fetch(`${this.baseUrl}/api/agents/launch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });

    if (!response.ok) {
      throw new Error(`Failed to launch agent: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Get agent status
   */
  async getAgentStatus(agentId: string): Promise<AgentStatus> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/status`);

    if (!response.ok) {
      throw new Error(`Failed to get agent status: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Get all agents
   */
  async getAllAgents(): Promise<AgentStatus[]> {
    const response = await fetch(`${this.baseUrl}/api/agents`);

    if (!response.ok) {
      throw new Error(`Failed to get agents: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Pause agent execution
   */
  async pauseAgent(agentId: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/pause`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to pause agent: ${response.statusText}`);
    }
  }

  /**
   * Resume agent execution
   */
  async resumeAgent(agentId: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/resume`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to resume agent: ${response.statusText}`);
    }
  }

  /**
   * Stop agent execution
   */
  async stopAgent(agentId: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/stop`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to stop agent: ${response.statusText}`);
    }
  }

  /**
   * Delete agent
   */
  async deleteAgent(agentId: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to delete agent: ${response.statusText}`);
    }
  }

  /**
   * Get agent output/logs
   */
  async getAgentOutput(agentId: string, limit: number = 100): Promise<string[]> {
    const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/output?limit=${limit}`);

    if (!response.ok) {
      throw new Error(`Failed to get agent output: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Launch multiple agents in parallel
   */
  async launchMultipleAgents(configs: AgentConfig[]): Promise<AgentStatus[]> {
    return Promise.all(configs.map((config) => this.launchAgent(config)));
  }

  /**
   * Wait for agent to complete
   */
  async waitForAgent(agentId: string, timeoutMs: number = 300000): Promise<AgentStatus> {
    const startTime = Date.now();

    return new Promise((resolve, reject) => {
      const checkStatus = async () => {
        try {
          const status = await this.getAgentStatus(agentId);

          if (status.status === 'completed' || status.status === 'failed' || status.status === 'cancelled') {
            resolve(status);
            return;
          }

          if (Date.now() - startTime > timeoutMs) {
            reject(new Error(`Agent ${agentId} did not complete within ${timeoutMs}ms`));
            return;
          }

          setTimeout(checkStatus, 1000);
        } catch (error) {
          reject(error);
        }
      };

      checkStatus();
    });
  }

  /**
   * Create a dependency between two agents
   */
  async createDependency(
    parentId: string,
    childId: string,
    type: 'sequential' | 'parallel' | 'conditional' = 'sequential',
    label?: string
  ): Promise<AgentConnection> {
    const response = await fetch(`${this.baseUrl}/api/agents/dependencies`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        from: parentId,
        to: childId,
        label,
        dependencyType: type,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to create dependency: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Get agent graph (all agents and connections)
   */
  async getAgentGraph(): Promise<{ agents: AgentStatus[]; connections: AgentConnection[] }> {
    const response = await fetch(`${this.baseUrl}/api/agents/graph`);

    if (!response.ok) {
      throw new Error(`Failed to get agent graph: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Disconnect WebSocket
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Export singleton instance
export const orchestrator = new AgentOrchestrator();
