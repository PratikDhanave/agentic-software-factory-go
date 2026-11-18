'use client';

import React, { useEffect, useState } from 'react';
import { apiClient, Metrics, DatabaseStats } from '@/lib/api';
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { AlertCircle, Loader } from 'lucide-react';

export default function Phase2Dashboard() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [dbStats, setDbStats] = useState<DatabaseStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [generatingCount, setGeneratingCount] = useState(0);

  const fetchData = async () => {
    try {
      const [metricsData, statsData] = await Promise.all([
        apiClient.getMetrics(),
        apiClient.getDatabaseStats(),
      ]);
      setMetrics(metricsData);
      setDbStats(statsData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleGenerateDecisions = async (count: number) => {
    setGeneratingCount(count);
    try {
      await apiClient.generateDecisions(count);
      await new Promise(resolve => setTimeout(resolve, 1000));
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate decisions');
    } finally {
      setGeneratingCount(0);
    }
  };

  const handleApproveLast = async () => {
    try {
      await apiClient.approveLast();
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve');
    }
  };

  const handleTriggerLearning = async () => {
    try {
      await apiClient.triggerLearning();
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to trigger learning');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-slate-900 to-purple-900">
        <div className="text-center">
          <Loader className="w-8 h-8 animate-spin text-purple-400 mx-auto mb-4" />
          <p className="text-gray-300">Loading Phase 2 metrics...</p>
        </div>
      </div>
    );
  }

  const agentArray = metrics ? Object.entries(metrics.Agents).map(([agent, data]) => ({
    agent,
    decisions: data.total_decisions,
    approved: data.approved,
    rejected: data.rejected,
    success_rate: (data.success_rate * 100).toFixed(1),
    confidence: (data.confidence_avg * 100).toFixed(1),
  })) : [];

  return (
    <div className="bg-gradient-to-br from-slate-900 to-purple-900 min-h-screen p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-white mb-2">Phase 2: Walk Control Plane</h1>
          <p className="text-gray-300">Self-improving agent factory with pattern learning</p>
        </div>

        {/* Error display */}
        {error && (
          <div className="flex items-start gap-3 p-4 bg-red-50 border border-red-200 rounded-lg mb-8">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        {/* Quick stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <StatCard label="Total Decisions" value={dbStats?.decisions || 0} icon="📊" />
          <StatCard label="Approvals" value={dbStats?.approvals || 0} icon="✅" />
          <StatCard label="Patterns Found" value={dbStats?.patterns || 0} icon="🔍" />
          <StatCard label="Agents Active" value={Object.keys(metrics?.Agents || {}).length} icon="🤖" />
        </div>

        {/* Controls */}
        <div className="bg-slate-800 rounded-lg p-6 mb-8 border border-purple-500">
          <h2 className="text-xl font-bold text-white mb-4">Controls</h2>
          <div className="flex flex-wrap gap-3">
            <button
              onClick={() => handleGenerateDecisions(10)}
              disabled={generatingCount > 0}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-500 text-white rounded font-medium"
            >
              {generatingCount > 0 ? `Generating ${generatingCount}...` : 'Generate 10 Decisions'}
            </button>
            <button
              onClick={() => handleGenerateDecisions(100)}
              disabled={generatingCount > 0}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-500 text-white rounded font-medium"
            >
              Generate 100 Decisions
            </button>
            <button
              onClick={handleApproveLast}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded font-medium"
            >
              Approve Last
            </button>
            <button
              onClick={handleTriggerLearning}
              className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded font-medium"
            >
              Trigger Learning
            </button>
            <button
              onClick={fetchData}
              className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded font-medium"
            >
              Refresh
            </button>
          </div>
        </div>

        {/* Agent Metrics Chart */}
        {agentArray.length > 0 && (
          <div className="bg-slate-800 rounded-lg p-6 mb-8 border border-purple-500">
            <h2 className="text-2xl font-bold text-white mb-6">Agent Performance</h2>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={agentArray}>
                <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                <XAxis dataKey="agent" stroke="#888" />
                <YAxis stroke="#888" />
                <Tooltip contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #666' }} />
                <Legend />
                <Bar dataKey="decisions" fill="#6366f1" name="Decisions" />
                <Bar dataKey="approved" fill="#10b981" name="Approved" />
                <Bar dataKey="rejected" fill="#ef4444" name="Rejected" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Patterns */}
        {metrics && metrics.Patterns.length > 0 && (
          <div className="bg-slate-800 rounded-lg p-6 mb-8 border border-purple-500">
            <h2 className="text-2xl font-bold text-white mb-6">Discovered Patterns</h2>
            <div className="space-y-3">
              {metrics.Patterns.slice(0, 10).map((pattern, i) => (
                <div key={i} className="flex items-center justify-between bg-slate-700 p-3 rounded">
                  <span className="text-gray-200">{pattern.description}</span>
                  <div className="flex gap-4">
                    <span className="text-purple-400">Freq: {pattern.frequency}</span>
                    <span className="text-green-400">Conf: {(pattern.confidence * 100).toFixed(1)}%</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {!metrics || Object.keys(metrics.Agents).length === 0 && (
          <div className="bg-slate-800 rounded-lg p-12 border border-purple-500 text-center">
            <p className="text-gray-400">No agents or metrics yet. Click "Generate Decisions" to start.</p>
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({ label, value, icon }: { label: string; value: number | string; icon: string }) {
  return (
    <div className="bg-slate-800 rounded-lg p-6 border border-purple-500">
      <div className="text-3xl mb-2">{icon}</div>
      <div className="text-2xl font-bold text-white">{value}</div>
      <div className="text-sm text-gray-400">{label}</div>
    </div>
  );
}
