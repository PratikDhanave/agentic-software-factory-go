import React, { useEffect } from 'react';
import { useStore } from '../store';
import { Building2, Users, Network, Lock, TrendingUp } from 'lucide-react';
import StatCard from '../components/StatCard';
import ActivityList from '../components/ActivityList';

export default function Dashboard() {
  const stats = useStore((state) => state.stats);
  const loadStats = useStore((state) => state.loadStats);
  const loadTenants = useStore((state) => state.loadTenants);
  const tenants = useStore((state) => state.tenants);

  useEffect(() => {
    loadStats();
    loadTenants();
  }, [loadStats, loadTenants]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-600 mt-2">System overview and recent activity</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Tenants"
          value={stats.tenants}
          icon={<Building2 className="text-blue-600" size={24} />}
          trend="+12% from last week"
          color="blue"
        />
        <StatCard
          title="Agents"
          value={stats.agents}
          icon={<Users className="text-green-600" size={24} />}
          trend="+5 this week"
          color="green"
        />
        <StatCard
          title="Services"
          value={stats.services}
          icon={<Network className="text-purple-600" size={24} />}
          trend="All healthy"
          color="purple"
        />
        <StatCard
          title="Credentials"
          value={stats.credentials}
          icon={<Lock className="text-orange-600" size={24} />}
          trend="0 revoked"
          color="orange"
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Tenants */}
        <div className="lg:col-span-2 bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Recent Tenants</h2>
          <div className="space-y-3">
            {tenants.slice(0, 5).map((tenant) => (
              <div
                key={tenant.id}
                className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 border border-gray-100"
              >
                <div>
                  <h3 className="font-medium text-gray-900">{tenant.name}</h3>
                  <p className="text-sm text-gray-600">{tenant.namespace}</p>
                </div>
                <div className="flex items-center gap-4">
                  <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                    tenant.status === 'active'
                      ? 'bg-green-100 text-green-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {tenant.status}
                  </span>
                  <span className="text-sm text-gray-600">
                    {tenant.quota_max_agents} agents
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* System Health */}
        <div className="bg-white rounded-lg shadow-sm p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">System Health</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-gray-700">Control Plane</span>
              <span className="text-green-600 font-bold">✓ Healthy</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-700">Discovery Registry</span>
              <span className="text-green-600 font-bold">✓ Healthy</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-700">Credential Broker</span>
              <span className="text-green-600 font-bold">✓ Healthy</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-700">Policy Engine</span>
              <span className="text-green-600 font-bold">✓ Healthy</span>
            </div>
            <div className="pt-4 border-t text-sm text-gray-600">
              Last updated: Just now
            </div>
          </div>
        </div>
      </div>

      {/* Activity */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold text-gray-900">Recent Activity</h2>
          <TrendingUp className="text-blue-600" size={20} />
        </div>
        <ActivityList />
      </div>
    </div>
  );
}
