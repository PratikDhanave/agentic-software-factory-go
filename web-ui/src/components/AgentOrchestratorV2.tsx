'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Plus, Play, Pause, StopCircle, Trash2, RefreshCw, ChevronRight, Zap, TrendingUp, Clock, AlertCircle } from 'lucide-react';

type AgentStatus = 'idle' | 'running' | 'completed' | 'failed' | 'paused';
type AgentType = 'general-purpose' | 'code-reviewer' | 'plan' | 'explore' | 'custom' | 'bug-finder' | 'code-generator';

interface AgentTask {
  id: string;
  name: string;
  type: AgentType;
  description: string;
  status: AgentStatus;
  progress: number;
  startTime: Date | null;
  endTime: Date | null;
  output?: string;
  error?: string;
  parentId?: string;
  metadata?: Record<string, any>;
}

interface AgentConnection {
  from: string;
  to: string;
  label?: string;
}

const StatusBadge: React.FC<{ status: AgentStatus; progress?: number }> = ({ status, progress = 0 }) => {
  const configs = {
    idle: { bg: 'bg-slate-500', text: 'Idle', icon: '⏳' },
    running: { bg: 'bg-blue-500', text: 'Running', icon: '⚙️' },
    completed: { bg: 'bg-emerald-500', text: 'Complete', icon: '✓' },
    failed: { bg: 'bg-red-500', text: 'Failed', icon: '✕' },
    paused: { bg: 'bg-amber-500', text: 'Paused', icon: '⏸' },
  };

  const config = configs[status];

  return (
    <div className="flex items-center gap-2">
      <div className={`${config.bg} rounded-full w-3 h-3 relative`}>
        {status === 'running' && <div className={`${config.bg} rounded-full w-3 h-3 absolute animate-pulse`} />}
      </div>
      <span className="text-sm font-medium text-slate-300">{config.text}</span>
      {status === 'running' && <span className="text-xs text-slate-400">{progress}%</span>}
    </div>
  );
};

export default function AgentOrchestratorV2() {
  const [agents, setAgents] = useState<AgentTask[]>([]);
  const [connections, setConnections] = useState<AgentConnection[]>([]);
  const [showLaunchModal, setShowLaunchModal] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [view, setView] = useState<'grid' | 'graph' | 'timeline'>('grid');
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const [newAgentConfig, setNewAgentConfig] = useState({
    name: '',
    type: 'general-purpose' as AgentType,
    description: '',
    parentId: '',
  });

  // Simulate agent execution
  const simulateAgentExecution = useCallback((agentId: string) => {
    const interval = setInterval(() => {
      setAgents((prev) => {
        const updated = [...prev];
        const agent = updated.find((a) => a.id === agentId);
        if (agent && agent.status === 'running') {
          agent.progress = Math.min(agent.progress + Math.random() * 20, 95);
          if (agent.progress >= 95) {
            agent.status = Math.random() > 0.1 ? 'completed' : 'failed';
            agent.progress = 100;
            agent.endTime = new Date();
            clearInterval(interval);
          }
        }
        return updated;
      });
    }, 1500);
  }, []);

  const launchAgent = useCallback(() => {
    if (!newAgentConfig.name) {
      alert('Agent name is required');
      return;
    }

    const agentId = `agent-${Date.now()}`;
    const newAgent: AgentTask = {
      id: agentId,
      name: newAgentConfig.name,
      type: newAgentConfig.type,
      description: newAgentConfig.description,
      status: 'running',
      progress: 0,
      startTime: new Date(),
      endTime: null,
      parentId: newAgentConfig.parentId || undefined,
    };

    setAgents((prev) => [...prev, newAgent]);

    if (newAgentConfig.parentId) {
      setConnections((prev) => [
        ...prev,
        { from: newAgentConfig.parentId, to: agentId, label: 'sequential' },
      ]);
    }

    simulateAgentExecution(agentId);
    setNewAgentConfig({ name: '', type: 'general-purpose', description: '', parentId: '' });
    setShowLaunchModal(false);
  }, [newAgentConfig, simulateAgentExecution]);

  const stats = {
    total: agents.length,
    running: agents.filter((a) => a.status === 'running').length,
    completed: agents.filter((a) => a.status === 'completed').length,
    failed: agents.filter((a) => a.status === 'failed').length,
  };

  // Grid View
  const GridView = () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {agents.map((agent) => (
        <div
          key={agent.id}
          onClick={() => setSelectedAgent(agent.id)}
          className={`group cursor-pointer bg-gradient-to-br from-slate-800 to-slate-900 border rounded-xl p-5 transition-all hover:shadow-lg hover:shadow-blue-500/20 ${
            selectedAgent === agent.id ? 'border-blue-500 shadow-lg shadow-blue-500/30' : 'border-slate-700'
          }`}
        >
          <div className="flex justify-between items-start mb-3">
            <div>
              <h3 className="font-semibold text-white text-lg group-hover:text-blue-400 transition">{agent.name}</h3>
              <p className="text-xs text-slate-400 mt-1">{agent.type}</p>
            </div>
            <StatusBadge status={agent.status} progress={agent.progress} />
          </div>

          {agent.description && (
            <p className="text-sm text-slate-400 mb-4 line-clamp-2">{agent.description}</p>
          )}

          {/* Progress bar */}
          {agent.status === 'running' && (
            <div className="mb-4">
              <div className="flex justify-between items-center mb-2">
                <span className="text-xs text-slate-500">Progress</span>
                <span className="text-xs font-semibold text-blue-400">{Math.round(agent.progress)}%</span>
              </div>
              <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-blue-500 to-cyan-500 transition-all duration-300"
                  style={{ width: `${agent.progress}%` }}
                />
              </div>
            </div>
          )}

          {/* Quick stats */}
          <div className="flex justify-between text-xs text-slate-400 pt-3 border-t border-slate-700">
            {agent.startTime && (
              <span>
                {Math.floor((new Date().getTime() - agent.startTime.getTime()) / 1000)}s elapsed
              </span>
            )}
            <span className={agent.status === 'failed' ? 'text-red-400' : 'text-emerald-400'}>
              {agent.status === 'completed' ? '✓ Done' : agent.status === 'failed' ? '✕ Error' : agent.status}
            </span>
          </div>
        </div>
      ))}
    </div>
  );

  // Timeline View
  const TimelineView = () => (
    <div className="space-y-6">
      {agents.map((agent, idx) => (
        <div
          key={agent.id}
          onClick={() => setSelectedAgent(agent.id)}
          className={`group cursor-pointer flex gap-4 p-4 rounded-xl border transition-all ${
            selectedAgent === agent.id
              ? 'bg-blue-500/10 border-blue-500'
              : 'bg-slate-800/50 border-slate-700 hover:border-slate-600'
          }`}
        >
          {/* Timeline dot */}
          <div className="flex flex-col items-center pt-1">
            <div
              className={`w-4 h-4 rounded-full border-2 ${
                agent.status === 'completed'
                  ? 'bg-emerald-500 border-emerald-400'
                  : agent.status === 'failed'
                    ? 'bg-red-500 border-red-400'
                    : agent.status === 'running'
                      ? 'bg-blue-500 border-blue-400 animate-pulse'
                      : 'bg-slate-600 border-slate-500'
              }`}
            />
            {idx < agents.length - 1 && <div className="w-0.5 h-12 bg-slate-700 mt-2" />}
          </div>

          {/* Timeline content */}
          <div className="flex-1 min-w-0">
            <div className="flex justify-between items-start mb-2">
              <div>
                <h4 className="font-semibold text-white group-hover:text-blue-400 transition truncate">{agent.name}</h4>
                <p className="text-sm text-slate-400">{agent.type}</p>
              </div>
              <StatusBadge status={agent.status} progress={agent.progress} />
            </div>

            {agent.status === 'running' && (
              <div className="mt-3">
                <div className="h-1 bg-slate-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all"
                    style={{ width: `${agent.progress}%` }}
                  />
                </div>
              </div>
            )}

            {agent.description && (
              <p className="text-sm text-slate-500 mt-2 line-clamp-1">{agent.description}</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Header */}
      <div className="sticky top-0 z-40 backdrop-blur-xl bg-slate-900/50 border-b border-slate-800">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-400 via-cyan-400 to-blue-500 bg-clip-text text-transparent">
                Agent Orchestrator
              </h1>
              <p className="text-slate-400 text-sm mt-1">Coordinate and monitor multi-agent workflows</p>
            </div>
            <button
              onClick={() => setShowLaunchModal(true)}
              className="flex items-center gap-2 bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-700 hover:to-cyan-700 text-white px-5 py-2.5 rounded-lg font-medium transition-all shadow-lg shadow-blue-500/30 hover:shadow-blue-500/50"
            >
              <Plus size={20} /> Launch Agent
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Stats Cards */}
        <div className="grid grid-cols-4 gap-4 mb-8">
          {[
            { label: 'Total', value: stats.total, icon: Zap, color: 'from-blue-500 to-cyan-500' },
            { label: 'Running', value: stats.running, icon: TrendingUp, color: 'from-emerald-500 to-green-500' },
            { label: 'Completed', value: stats.completed, icon: Clock, color: 'from-purple-500 to-pink-500' },
            { label: 'Failed', value: stats.failed, icon: AlertCircle, color: 'from-red-500 to-orange-500' },
          ].map((stat) => (
            <div key={stat.label} className="bg-gradient-to-br from-slate-800 to-slate-900 border border-slate-700 rounded-xl p-5 hover:border-slate-600 transition-all">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-slate-400 text-sm font-medium">{stat.label}</p>
                  <p className={`text-3xl font-bold bg-gradient-to-r ${stat.color} bg-clip-text text-transparent mt-2`}>
                    {stat.value}
                  </p>
                </div>
                <div className={`bg-gradient-to-br ${stat.color} p-3 rounded-lg opacity-20`}>
                  <stat.icon size={24} className="text-white" />
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* View Selector */}
        <div className="flex gap-2 mb-6">
          {['grid', 'timeline', 'graph'].map((v) => (
            <button
              key={v}
              onClick={() => setView(v as any)}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                view === v
                  ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/30'
                  : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-slate-300'
              }`}
            >
              {v.charAt(0).toUpperCase() + v.slice(1)}
            </button>
          ))}
        </div>

        {/* Content */}
        {agents.length === 0 ? (
          <div className="text-center py-16">
            <div className="text-6xl mb-4">🏭</div>
            <h3 className="text-xl font-semibold text-white mb-2">No agents launched yet</h3>
            <p className="text-slate-400 mb-6">Create your first agent to get started</p>
            <button
              onClick={() => setShowLaunchModal(true)}
              className="inline-flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg font-medium transition-all"
            >
              <Plus size={20} /> Launch First Agent
            </button>
          </div>
        ) : view === 'grid' ? (
          <GridView />
        ) : (
          <TimelineView />
        )}
      </div>

      {/* Detail Sidebar */}
      {selectedAgent && (
        <div className="fixed right-0 top-0 h-full w-96 bg-slate-900 border-l border-slate-800 shadow-2xl shadow-black/50 backdrop-blur-xl overflow-y-auto">
          {(() => {
            const agent = agents.find((a) => a.id === selectedAgent);
            if (!agent) return null;

            return (
              <div className="p-6">
                {/* Close button */}
                <button
                  onClick={() => setSelectedAgent(null)}
                  className="absolute top-4 right-4 text-slate-400 hover:text-white transition"
                >
                  ✕
                </button>

                <div className="space-y-6">
                  {/* Header */}
                  <div>
                    <StatusBadge status={agent.status} progress={agent.progress} />
                    <h2 className="text-2xl font-bold text-white mt-3">{agent.name}</h2>
                    <p className="text-slate-400 text-sm mt-1">{agent.type}</p>
                  </div>

                  {/* Description */}
                  {agent.description && (
                    <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4">
                      <p className="text-sm text-slate-300">{agent.description}</p>
                    </div>
                  )}

                  {/* Progress */}
                  {agent.status === 'running' && (
                    <div>
                      <label className="text-sm font-medium text-slate-300 block mb-3">
                        Progress: {Math.round(agent.progress)}%
                      </label>
                      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-blue-500 to-cyan-500 transition-all"
                          style={{ width: `${agent.progress}%` }}
                        />
                      </div>
                    </div>
                  )}

                  {/* Stats */}
                  <div className="grid grid-cols-2 gap-3">
                    <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-3">
                      <p className="text-xs text-slate-400 mb-1">Started</p>
                      <p className="text-sm font-mono text-slate-300">
                        {agent.startTime?.toLocaleTimeString() || '-'}
                      </p>
                    </div>
                    <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-3">
                      <p className="text-xs text-slate-400 mb-1">Duration</p>
                      <p className="text-sm font-mono text-slate-300">
                        {agent.startTime
                          ? `${Math.floor((new Date().getTime() - agent.startTime.getTime()) / 1000)}s`
                          : '-'}
                      </p>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="space-y-2 pt-4 border-t border-slate-700">
                    {agent.status === 'running' && (
                      <>
                        <button className="w-full flex items-center justify-center gap-2 bg-amber-600 hover:bg-amber-700 text-white py-2 rounded-lg text-sm transition font-medium">
                          <Pause size={16} /> Pause
                        </button>
                        <button className="w-full flex items-center justify-center gap-2 bg-red-600 hover:bg-red-700 text-white py-2 rounded-lg text-sm transition font-medium">
                          <StopCircle size={16} /> Stop
                        </button>
                      </>
                    )}
                    {agent.status === 'paused' && (
                      <button className="w-full flex items-center justify-center gap-2 bg-emerald-600 hover:bg-emerald-700 text-white py-2 rounded-lg text-sm transition font-medium">
                        <Play size={16} /> Resume
                      </button>
                    )}
                    <button className="w-full flex items-center justify-center gap-2 bg-slate-700 hover:bg-slate-600 text-slate-300 py-2 rounded-lg text-sm transition font-medium">
                      <Trash2 size={16} /> Delete
                    </button>
                  </div>
                </div>
              </div>
            );
          })()}
        </div>
      )}

      {/* Launch Modal */}
      {showLaunchModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-gradient-to-br from-slate-800 to-slate-900 border border-slate-700 rounded-2xl p-8 max-w-md w-full shadow-2xl shadow-black/50">
            <h3 className="text-2xl font-bold text-white mb-6">Launch New Agent</h3>

            <div className="space-y-5">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">Agent Name *</label>
                <input
                  type="text"
                  value={newAgentConfig.name}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, name: e.target.value })}
                  placeholder="e.g., Bug Finder"
                  className="w-full px-4 py-2.5 bg-slate-700/50 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">Agent Type</label>
                <select
                  value={newAgentConfig.type}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, type: e.target.value as AgentType })}
                  className="w-full px-4 py-2.5 bg-slate-700/50 border border-slate-600 rounded-lg text-white focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition"
                >
                  <option value="general-purpose">General Purpose</option>
                  <option value="code-reviewer">Code Reviewer</option>
                  <option value="plan">Planner</option>
                  <option value="explore">Explorer</option>
                  <option value="bug-finder">Bug Finder</option>
                  <option value="code-generator">Code Generator</option>
                  <option value="custom">Custom</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">Description</label>
                <textarea
                  value={newAgentConfig.description}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, description: e.target.value })}
                  placeholder="What should this agent do?"
                  rows={3}
                  className="w-full px-4 py-2.5 bg-slate-700/50 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition resize-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">Parent Agent (Optional)</label>
                <select
                  value={newAgentConfig.parentId}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, parentId: e.target.value })}
                  className="w-full px-4 py-2.5 bg-slate-700/50 border border-slate-600 rounded-lg text-white focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition"
                >
                  <option value="">Independent agent</option>
                  {agents
                    .filter((a) => a.status !== 'failed')
                    .map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name}
                      </option>
                    ))}
                </select>
              </div>
            </div>

            <div className="flex gap-3 mt-8">
              <button
                onClick={() => setShowLaunchModal(false)}
                className="flex-1 px-4 py-2.5 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition font-medium"
              >
                Cancel
              </button>
              <button
                onClick={launchAgent}
                className="flex-1 px-4 py-2.5 bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-700 hover:to-cyan-700 text-white rounded-lg transition font-medium shadow-lg shadow-blue-500/30"
              >
                Launch Agent
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
