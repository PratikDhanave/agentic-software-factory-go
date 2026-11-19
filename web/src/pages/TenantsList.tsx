import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useStore } from '../store';
import { Plus, Trash2, Users, Lock } from 'lucide-react';

export default function TenantsList() {
  const tenants = useStore((state) => state.tenants);
  const loadTenants = useStore((state) => state.loadTenants);
  const deleteTenant = useStore((state) => state.deleteTenant);
  const createTenant = useStore((state) => state.createTenant);
  const [newTenantName, setNewTenantName] = useState('');

  useEffect(() => {
    loadTenants();
  }, [loadTenants]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newTenantName.trim()) {
      await createTenant(newTenantName);
      setNewTenantName('');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Tenants</h1>
        <p className="text-gray-600 mt-2">Manage isolated organization namespaces</p>
      </div>

      {/* Create Form */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <form onSubmit={handleCreate} className="flex gap-3">
          <input
            type="text"
            placeholder="Enter tenant name..."
            value={newTenantName}
            onChange={(e) => setNewTenantName(e.target.value)}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
          >
            <Plus size={20} />
            Create Tenant
          </button>
        </form>
      </div>

      {/* Tenants Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {tenants.map((tenant) => (
          <Link key={tenant.id} to={`/tenants/${tenant.id}`}>
            <div className="bg-white rounded-lg shadow-sm p-6 hover:shadow-md transition-shadow cursor-pointer">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-bold text-gray-900">{tenant.name}</h3>
                  <p className="text-sm text-gray-600">{tenant.namespace}</p>
                </div>
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    deleteTenant(tenant.id);
                  }}
                  className="p-2 text-red-600 hover:bg-red-50 rounded-lg"
                >
                  <Trash2 size={20} />
                </button>
              </div>

              <div className="space-y-3 pt-4 border-t">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-gray-600">
                    <Users size={16} />
                    <span className="text-sm">Max Agents</span>
                  </div>
                  <span className="font-bold">{tenant.quota_max_agents}</span>
                </div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-gray-600">
                    <Lock size={16} />
                    <span className="text-sm">Status</span>
                  </div>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    tenant.status === 'active'
                      ? 'bg-green-100 text-green-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {tenant.status}
                  </span>
                </div>
              </div>
            </div>
          </Link>
        ))}
      </div>

      {tenants.length === 0 && (
        <div className="text-center py-12 bg-white rounded-lg">
          <p className="text-gray-600">No tenants yet. Create one to get started.</p>
        </div>
      )}
    </div>
  );
}
