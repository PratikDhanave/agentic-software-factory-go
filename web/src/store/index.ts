import { create } from 'zustand';
import { client, Tenant, Agent, Credential, ServiceInstance, PolicyRule } from '../api/client';

interface UIState {
  // Loading states
  loading: boolean;
  error: string | null;

  // Data
  tenants: Tenant[];
  selectedTenant: Tenant | null;
  agents: Agent[];
  credentials: Credential[];
  services: ServiceInstance[];
  policies: PolicyRule[];

  // Stats
  stats: {
    tenants: number;
    agents: number;
    services: number;
    credentials: number;
  };

  // Actions
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  loadTenants: () => Promise<void>;
  loadTenant: (id: string) => Promise<void>;
  createTenant: (name: string) => Promise<void>;
  deleteTenant: (id: string) => Promise<void>;
  loadAgents: (tenantId: string) => Promise<void>;
  loadCredentials: (agentId: string) => Promise<void>;
  revokeCredential: (credentialId: string) => Promise<void>;
  loadServices: (tenantId: string) => Promise<void>;
  updateServiceHealth: (serviceId: string, healthy: boolean) => Promise<void>;
  loadPolicies: (tenantId: string) => Promise<void>;
  createPolicy: (tenantId: string, rules: PolicyRule[]) => Promise<void>;
  loadStats: () => Promise<void>;
}

export const useStore = create<UIState>((set) => ({
  loading: false,
  error: null,
  tenants: [],
  selectedTenant: null,
  agents: [],
  credentials: [],
  services: [],
  policies: [],
  stats: {
    tenants: 0,
    agents: 0,
    services: 0,
    credentials: 0,
  },

  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),

  loadTenants: async () => {
    set({ loading: true, error: null });
    try {
      const tenants = await client.listTenants();
      set({ tenants, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load tenants',
        loading: false,
      });
    }
  },

  loadTenant: async (id) => {
    set({ loading: true, error: null });
    try {
      const tenant = await client.getTenant(id);
      set({ selectedTenant: tenant, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load tenant',
        loading: false,
      });
    }
  },

  createTenant: async (name) => {
    set({ loading: true, error: null });
    try {
      const newTenant = await client.createTenant(name);
      set((state) => ({
        tenants: [...state.tenants, newTenant],
        loading: false,
      }));
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to create tenant',
        loading: false,
      });
    }
  },

  deleteTenant: async (id) => {
    set({ loading: true, error: null });
    try {
      await client.deleteTenant(id);
      set((state) => ({
        tenants: state.tenants.filter((t) => t.id !== id),
        loading: false,
      }));
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to delete tenant',
        loading: false,
      });
    }
  },

  loadAgents: async (tenantId) => {
    set({ loading: true, error: null });
    try {
      const agents = await client.listAgents(tenantId);
      set({ agents, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load agents',
        loading: false,
      });
    }
  },

  loadCredentials: async (agentId) => {
    set({ loading: true, error: null });
    try {
      const credentials = await client.listCredentials(agentId);
      set({ credentials, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load credentials',
        loading: false,
      });
    }
  },

  revokeCredential: async (credentialId) => {
    set({ loading: true, error: null });
    try {
      await client.revokeCredential(credentialId);
      set((state) => ({
        credentials: state.credentials.map((c) =>
          c.id === credentialId ? { ...c, revoked: true } : c
        ),
        loading: false,
      }));
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to revoke credential',
        loading: false,
      });
    }
  },

  loadServices: async (tenantId) => {
    set({ loading: true, error: null });
    try {
      const services = await client.listServices(tenantId);
      set({ services, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load services',
        loading: false,
      });
    }
  },

  updateServiceHealth: async (serviceId, healthy) => {
    set({ loading: true, error: null });
    try {
      await client.updateServiceHealth(serviceId, healthy);
      set((state) => ({
        services: state.services.map((s) =>
          s.id === serviceId
            ? { ...s, health: healthy ? 'healthy' : 'unhealthy' }
            : s
        ),
        loading: false,
      }));
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to update service health',
        loading: false,
      });
    }
  },

  loadPolicies: async (tenantId) => {
    set({ loading: true, error: null });
    try {
      const policies = await client.getPolicies(tenantId);
      set({ policies, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load policies',
        loading: false,
      });
    }
  },

  createPolicy: async (tenantId, rules) => {
    set({ loading: true, error: null });
    try {
      const policies = await client.createPolicy(tenantId, rules);
      set({ policies, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to create policy',
        loading: false,
      });
    }
  },

  loadStats: async () => {
    set({ loading: true, error: null });
    try {
      const stats = await client.getStats();
      set({ stats, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Failed to load stats',
        loading: false,
      });
    }
  },
}));
