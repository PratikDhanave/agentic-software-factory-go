import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useStore } from '../store';

export default function TenantDetail() {
  const { id } = useParams<{ id: string }>();
  const selectedTenant = useStore((state) => state.selectedTenant);
  const loadTenant = useStore((state) => state.loadTenant);

  useEffect(() => {
    if (id) loadTenant(id);
  }, [id, loadTenant]);

  if (!selectedTenant) {
    return <div className="text-center py-12">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-gray-900">{selectedTenant.name}</h1>
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="grid grid-cols-2 gap-6">
          <div>
            <p className="text-gray-600 text-sm">Namespace</p>
            <p className="text-lg font-bold text-gray-900">{selectedTenant.namespace}</p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">Status</p>
            <p className="text-lg font-bold text-green-600">{selectedTenant.status}</p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">Max Agents</p>
            <p className="text-lg font-bold">{selectedTenant.quota_max_agents}</p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">Created</p>
            <p className="text-lg font-bold">{new Date(selectedTenant.created_at).toLocaleDateString()}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
