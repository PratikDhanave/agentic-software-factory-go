import axios, { AxiosInstance } from 'axios';

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  status: number;
}

export interface Tenant {
  id: string;
  name: string;
  namespace: string;
  created_at: string;
  status: 'active' | 'inactive' | 'deleted';
  quota_max_agents: number;
}

export interface Agent {
  id: string;
  name: string;
  type: string;
  public_key?: string;
  status: string;
  created_at: string;
}

export interface Credential {
  id: string;
  agent_id: string;
  subject: string;
  actions: string[];
  issued_at: string;
  expires_at: string;
  revoked: boolean;
}

export interface ServiceInstance {
  id: string;
  tenant_id: string;
  agent_id: string;
  name: string;
  address: string;
  port: number;
  health: 'healthy' | 'unhealthy' | 'draining';
  created_at: string;
}

export interface PolicyRule {
  subject: string;
  action: string;
  effect: 'allow' | 'deny';
}

export class FactoryClient {
  private api: AxiosInstance;

  constructor(baseURL: string = 'http://localhost:8080/api') {
    this.api = axios.create({
      baseURL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request tracing
    this.api.interceptors.request.use((config) => {
      config.headers['X-Request-ID'] = crypto.randomUUID();
      return config;
    });
  }

  // Tenants
  async listTenants(): Promise<Tenant[]> {
    const res = await this.api.get('/tenants');
    return res.data || [];
  }

  async getTenant(id: string): Promise<Tenant> {
    const res = await this.api.get(`/tenants/${id}`);
    return res.data;
  }

  async createTenant(name: string, metadata?: Record<string, string>): Promise<Tenant> {
    const res = await this.api.post('/tenants', { name, metadata });
    return res.data;
  }

  async updateTenant(id: string, metadata: Record<string, string>): Promise<Tenant> {
    const res = await this.api.put(`/tenants/${id}`, { metadata });
    return res.data;
  }

  async deleteTenant(id: string): Promise<void> {
    await this.api.delete(`/tenants/${id}`);
  }

  // Agents
  async listAgents(tenantId: string): Promise<Agent[]> {
    const res = await this.api.get(`/tenants/${tenantId}/agents`);
    return res.data || [];
  }

  async getAgent(tenantId: string, agentId: string): Promise<Agent> {
    const res = await this.api.get(`/tenants/${tenantId}/agents/${agentId}`);
    return res.data;
  }

  // Credentials
  async listCredentials(agentId: string): Promise<Credential[]> {
    const res = await this.api.get(`/agents/${agentId}/credentials`);
    return res.data || [];
  }

  async getCredential(credentialId: string): Promise<Credential> {
    const res = await this.api.get(`/credentials/${credentialId}`);
    return res.data;
  }

  async revokeCredential(credentialId: string): Promise<void> {
    await this.api.post(`/credentials/${credentialId}/revoke`);
  }

  // Services
  async listServices(tenantId: string): Promise<ServiceInstance[]> {
    const res = await this.api.get(`/tenants/${tenantId}/services`);
    return res.data || [];
  }

  async getService(serviceId: string): Promise<ServiceInstance> {
    const res = await this.api.get(`/services/${serviceId}`);
    return res.data;
  }

  async updateServiceHealth(serviceId: string, healthy: boolean): Promise<void> {
    await this.api.patch(`/services/${serviceId}/health`, { healthy });
  }

  // Policies
  async getPolicies(tenantId: string): Promise<PolicyRule[]> {
    const res = await this.api.get(`/tenants/${tenantId}/policies`);
    return res.data || [];
  }

  async createPolicy(tenantId: string, rules: PolicyRule[]): Promise<PolicyRule[]> {
    const res = await this.api.post(`/tenants/${tenantId}/policies`, { rules });
    return res.data;
  }

  // Health
  async getHealth(): Promise<{ status: string; timestamp: string }> {
    const res = await this.api.get('/health');
    return res.data;
  }

  // Stats
  async getStats(): Promise<{
    tenants: number;
    agents: number;
    services: number;
    credentials: number;
  }> {
    const res = await this.api.get('/stats');
    return res.data || { tenants: 0, agents: 0, services: 0, credentials: 0 };
  }
}

export const client = new FactoryClient();
