'use client';

import { useState, useEffect } from 'react';
import { apiClient, PendingApproval } from '@/lib/api';
import { CheckCircle, XCircle, AlertCircle, Loader, RefreshCw } from 'lucide-react';

export default function ApprovalsPanel() {
  const [approvals, setApprovals] = useState<PendingApproval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [processing, setProcessing] = useState<string | null>(null);
  const [overrideId, setOverrideId] = useState<string | null>(null);
  const [overrideValue, setOverrideValue] = useState('');

  useEffect(() => {
    loadApprovals();
    const interval = setInterval(loadApprovals, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadApprovals = async () => {
    try {
      const data = await apiClient.getPendingApprovals();
      setApprovals(data);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load approvals');
    } finally {
      setLoading(false);
    }
  };

  const handleApproval = async (id: string, approved: boolean, override?: string) => {
    setProcessing(id);
    try {
      await apiClient.submitApproval(id, approved, override);
      setApprovals(approvals.filter((a) => a.id !== id));
      setOverrideId(null);
      setOverrideValue('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit approval');
    } finally {
      setProcessing(null);
    }
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex items-center justify-center gap-2 py-12">
          <Loader className="w-5 h-5 animate-spin text-blue-600" />
          <p className="text-gray-600">Loading pending approvals...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">
          Pending Approvals {approvals.length > 0 && `(${approvals.length})`}
        </h2>
        <button
          onClick={loadApprovals}
          className="p-2 hover:bg-gray-100 rounded-lg transition"
          title="Refresh"
        >
          <RefreshCw className="w-5 h-5 text-gray-600" />
        </button>
      </div>

      {error && (
        <div className="flex items-start gap-3 p-4 bg-red-50 border border-red-200 rounded-lg mb-4">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
          <p className="text-sm text-red-700">{error}</p>
        </div>
      )}

      {approvals.length === 0 ? (
        <div className="text-center py-12 text-gray-600">
          <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-4" />
          <p>No pending approvals</p>
        </div>
      ) : (
        <div className="space-y-4">
          {approvals.map((approval) => (
            <div
              key={approval.id}
              className="p-4 border border-yellow-200 bg-yellow-50 rounded-lg"
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-gray-900">
                    {approval.agent_role}
                  </h3>
                  <p className="text-sm text-gray-600 mt-1">{approval.decision}</p>
                  <div className="flex items-center gap-4 mt-3">
                    <span className="text-xs font-semibold px-2 py-1 bg-blue-100 text-blue-800 rounded">
                      {Math.round(approval.confidence * 100)}% confidence
                    </span>
                    <span className="text-xs text-gray-500">
                      {new Date(approval.created_at).toLocaleTimeString()}
                    </span>
                  </div>
                </div>
              </div>

              {overrideId === approval.id && (
                <div className="mb-4 p-3 bg-white rounded border border-gray-300">
                  <input
                    type="text"
                    placeholder="Enter override reason..."
                    value={overrideValue}
                    onChange={(e) => setOverrideValue(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              )}

              <div className="flex gap-3">
                <button
                  onClick={() => handleApproval(approval.id, true)}
                  disabled={processing === approval.id}
                  className="flex-1 flex items-center justify-center gap-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white font-medium py-2 rounded-lg transition text-sm"
                >
                  {processing === approval.id ? (
                    <Loader className="w-4 h-4 animate-spin" />
                  ) : (
                    <CheckCircle className="w-4 h-4" />
                  )}
                  Approve
                </button>
                <button
                  onClick={() => handleApproval(approval.id, false)}
                  disabled={processing === approval.id}
                  className="flex-1 bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white font-medium py-2 rounded-lg transition text-sm flex items-center justify-center gap-2"
                >
                  {processing === approval.id ? (
                    <Loader className="w-4 h-4 animate-spin" />
                  ) : (
                    <XCircle className="w-4 h-4" />
                  )}
                  Reject
                </button>
                {overrideId === approval.id ? (
                  <button
                    onClick={() =>
                      handleApproval(approval.id, true, overrideValue)
                    }
                    className="flex-1 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 rounded-lg transition text-sm"
                  >
                    Override
                  </button>
                ) : (
                  <button
                    onClick={() => setOverrideId(approval.id)}
                    className="flex-1 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 rounded-lg transition text-sm"
                  >
                    Override
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
