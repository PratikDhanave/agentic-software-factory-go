import React, { useEffect, useState } from 'react';
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface AgentMetrics {
  SuccessRate: number;
  Confidence: number;
  DecisionCount: number;
}

interface PatternMetrics {
  IssueType: string;
  Agent: string;
  SuccessRate: number;
  Applied: number;
}

interface SelfImprovementMetrics {
  Agents: { [key: string]: AgentMetrics };
  Patterns: PatternMetrics[];
  Timestamp: string;
}

export const SelfImprovementDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<SelfImprovementMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/metrics/self-improvement`
        );
        if (!response.ok) throw new Error('Failed to fetch metrics');
        const data = await response.json();
        setMetrics(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        setMetrics(null);
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div className="p-8 text-gray-400">Loading metrics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;
  if (!metrics) return <div className="p-8 text-gray-400">No metrics available</div>;

  // Prepare agent data for charts
  const agentData = Object.entries(metrics.Agents).map(([agent, data]) => ({
    agent,
    successRate: (data.SuccessRate * 100).toFixed(1),
    confidence: (data.Confidence * 100).toFixed(1),
    decisions: data.DecisionCount,
  }));

  // Prepare pattern data
  const topPatterns = metrics.Patterns.slice(0, 10).map((p) => ({
    name: `${p.Agent} → ${p.IssueType}`,
    successRate: (p.SuccessRate * 100).toFixed(1),
    applied: p.Applied,
  }));

  return (
    <div className="bg-gradient-to-br from-slate-900 to-purple-900 min-h-screen p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-12">
          <h1 className="text-4xl font-bold text-white mb-2">Self-Improvement Dashboard</h1>
          <p className="text-gray-300">Phase 2: Walk - Agent learning and pattern discovery</p>
          <p className="text-gray-400 text-sm mt-2">Updated: {new Date(metrics.Timestamp).toLocaleString()}</p>
        </div>

        {/* Metrics Grid */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-12">
          {/* Total Agents */}
          <MetricCard
            title="Active Agents"
            value={Object.keys(metrics.Agents).length}
            subtitle="Learning from decisions"
            icon="🤖"
          />

          {/* Total Patterns */}
          <MetricCard
            title="Patterns Discovered"
            value={metrics.Patterns.length}
            subtitle="High-confidence patterns"
            icon="🔍"
          />

          {/* Average Success Rate */}
          <MetricCard
            title="Avg Success Rate"
            value={`${(
              Object.values(metrics.Agents).reduce((sum, a) => sum + a.SuccessRate, 0) /
              Object.keys(metrics.Agents).length *
              100
            ).toFixed(1)}%`}
            subtitle="Across all agents"
            icon="✅"
          />

          {/* Total Decisions */}
          <MetricCard
            title="Total Decisions"
            value={Object.values(metrics.Agents).reduce((sum, a) => sum + a.DecisionCount, 0)}
            subtitle="Learning dataset"
            icon="📊"
          />
        </div>

        {/* Agent Success Rates */}
        <div className="bg-slate-800 rounded-lg p-6 mb-8 border border-purple-500">
          <h2 className="text-2xl font-bold text-white mb-6">Agent Success Rates</h2>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={agentData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis dataKey="agent" stroke="#888" />
              <YAxis stroke="#888" />
              <Tooltip
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #666' }}
                labelStyle={{ color: '#fff' }}
              />
              <Legend />
              <Bar dataKey="successRate" fill="#10b981" name="Success Rate %" />
              <Bar dataKey="confidence" fill="#6366f1" name="Avg Confidence %" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Top Patterns */}
        <div className="bg-slate-800 rounded-lg p-6 mb-8 border border-purple-500">
          <h2 className="text-2xl font-bold text-white mb-6">Top Discovered Patterns</h2>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={topPatterns} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#444" />
              <XAxis type="number" stroke="#888" />
              <YAxis dataKey="name" type="category" width={180} stroke="#888" fontSize={12} />
              <Tooltip
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #666' }}
                labelStyle={{ color: '#fff' }}
              />
              <Bar dataKey="successRate" fill="#8b5cf6" name="Success Rate %" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Detailed Agent Metrics Table */}
        <div className="bg-slate-800 rounded-lg p-6 border border-purple-500 overflow-x-auto">
          <h2 className="text-2xl font-bold text-white mb-6">Detailed Agent Metrics</h2>
          <table className="w-full text-left text-gray-300">
            <thead className="border-b border-purple-500">
              <tr>
                <th className="px-4 py-3 font-semibold">Agent</th>
                <th className="px-4 py-3 font-semibold">Success Rate</th>
                <th className="px-4 py-3 font-semibold">Avg Confidence</th>
                <th className="px-4 py-3 font-semibold">Decisions</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody>
              {agentData.map((agent, i) => (
                <tr key={i} className="border-b border-slate-700 hover:bg-slate-700">
                  <td className="px-4 py-3 font-medium text-white">{agent.agent}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-24 bg-slate-600 rounded h-2">
                        <div
                          className="bg-green-500 h-2 rounded"
                          style={{ width: `${parseFloat(agent.successRate)}%` }}
                        />
                      </div>
                      <span>{agent.successRate}%</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-24 bg-slate-600 rounded h-2">
                        <div
                          className="bg-indigo-500 h-2 rounded"
                          style={{ width: `${parseFloat(agent.confidence)}%` }}
                        />
                      </div>
                      <span>{agent.confidence}%</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">{agent.decisions}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-3 py-1 rounded-full text-sm font-medium ${
                        parseFloat(agent.successRate) >= 80
                          ? 'bg-green-900 text-green-200'
                          : parseFloat(agent.successRate) >= 70
                          ? 'bg-yellow-900 text-yellow-200'
                          : 'bg-red-900 text-red-200'
                      }`}
                    >
                      {parseFloat(agent.successRate) >= 80
                        ? 'Specialized'
                        : parseFloat(agent.successRate) >= 70
                        ? 'Learning'
                        : 'Improving'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Phase Status */}
        <div className="mt-12 bg-slate-800 rounded-lg p-6 border border-purple-500">
          <h3 className="text-xl font-bold text-white mb-4">Phase Status</h3>
          <PhaseIndicator
            phase="Crawl"
            completed
            description="Manual approvals, metric collection"
            status="complete"
          />
          <PhaseIndicator
            phase="Walk"
            completed={metrics.Patterns.length > 0}
            description="Pattern learning, approval skipping for 90%+ confidence"
            status={metrics.Patterns.length > 0 ? 'active' : 'pending'}
          />
          <PhaseIndicator
            phase="Run"
            completed={false}
            description="Autonomous execution, LLM-as-judge, full GitHub automation"
            status="pending"
          />
        </div>
      </div>
    </div>
  );
};

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle: string;
  icon: string;
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, subtitle, icon }) => (
  <div className="bg-slate-800 rounded-lg p-6 border border-purple-500 hover:border-purple-400 transition">
    <div className="text-4xl mb-2">{icon}</div>
    <h3 className="text-gray-400 text-sm font-medium">{title}</h3>
    <div className="text-3xl font-bold text-white mt-2">{value}</div>
    <p className="text-gray-500 text-xs mt-2">{subtitle}</p>
  </div>
);

interface PhaseIndicatorProps {
  phase: string;
  completed: boolean;
  description: string;
  status: 'complete' | 'active' | 'pending';
}

const PhaseIndicator: React.FC<PhaseIndicatorProps> = ({ phase, completed, description, status }) => (
  <div className="mb-4 pb-4 border-b border-slate-700 last:border-0">
    <div className="flex items-center gap-4">
      <div className="text-2xl">
        {status === 'complete' ? '✅' : status === 'active' ? '⚙️' : '⏳'}
      </div>
      <div className="flex-1">
        <h4 className="text-white font-semibold">Phase {phase}</h4>
        <p className="text-gray-400 text-sm">{description}</p>
      </div>
      <span
        className={`px-3 py-1 rounded-full text-sm font-medium ${
          status === 'complete'
            ? 'bg-green-900 text-green-200'
            : status === 'active'
            ? 'bg-blue-900 text-blue-200'
            : 'bg-slate-700 text-gray-300'
        }`}
      >
        {status === 'complete' ? 'Complete' : status === 'active' ? 'Active' : 'Pending'}
      </span>
    </div>
  </div>
);
