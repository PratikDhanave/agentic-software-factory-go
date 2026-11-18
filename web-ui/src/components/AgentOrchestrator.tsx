'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Plus, Play, Pause, StopCircle, Trash2, RefreshCw } from 'lucide-react';

type AgentStatus = 'idle' | 'running' | 'completed' | 'failed' | 'paused';
type AgentType = 'general-purpose' | 'code-reviewer' | 'plan' | 'explore' | 'custom';

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
}

interface AgentConnection {
  from: string;
  to: string;
  label?: string;
}

export default function AgentOrchestrator() {
  const [agents, setAgents] = useState<AgentTask[]>([]);
  const [connections, setConnections] = useState<AgentConnection[]>([]);
  const [showLaunchModal, setShowLaunchModal] = useState(false);
  const [newAgentConfig, setNewAgentConfig] = useState({
    name: '',
    type: 'general-purpose' as AgentType,
    description: '',
    parentId: '',
  });
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  // Simulate agent execution
  const simulateAgentExecution = useCallback((agentId: string) => {
    const interval = setInterval(() => {
      setAgents((prev) => {
        const updated = [...prev];
        const agent = updated.find((a) => a.id === agentId);
        if (agent && agent.status === 'running') {
          agent.progress = Math.min(agent.progress + Math.random() * 15, 95);
          if (agent.progress >= 95) {
            agent.status = Math.random() > 0.1 ? 'completed' : 'failed';
            agent.progress = 100;
            agent.endTime = new Date();
            clearInterval(interval);
          }
        }
        return updated;
      });
    }, 1000);
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

    // Add connection if it has a parent
    if (newAgentConfig.parentId) {
      setConnections((prev) => [
        ...prev,
        { from: newAgentConfig.parentId, to: agentId, label: 'depends' },
      ]);
    }

    // Start simulation
    simulateAgentExecution(agentId);

    // Reset form
    setNewAgentConfig({ name: '', type: 'general-purpose', description: '', parentId: '' });
    setShowLaunchModal(false);
  }, [newAgentConfig, simulateAgentExecution]);

  const pauseAgent = (agentId: string) => {
    setAgents((prev) =>
      prev.map((a) =>
        a.id === agentId && a.status === 'running' ? { ...a, status: 'paused' } : a
      )
    );
  };

  const resumeAgent = (agentId: string) => {
    setAgents((prev) =>
      prev.map((a) => {
        if (a.id === agentId && a.status === 'paused') {
          simulateAgentExecution(agentId);
          return { ...a, status: 'running' };
        }
        return a;
      })
    );
  };

  const stopAgent = (agentId: string) => {
    setAgents((prev) =>
      prev.map((a) =>
        a.id === agentId && (a.status === 'running' || a.status === 'paused')
          ? { ...a, status: 'failed', endTime: new Date() }
          : a
      )
    );
  };

  const deleteAgent = (agentId: string) => {
    setAgents((prev) => prev.filter((a) => a.id !== agentId));
    setConnections((prev) => prev.filter((c) => c.from !== agentId && c.to !== agentId));
    setSelectedAgent(null);
  };

  const getStatusColor = (status: AgentStatus) => {
    switch (status) {
      case 'running':
        return 'bg-blue-100 border-blue-400 text-blue-900';
      case 'completed':
        return 'bg-green-100 border-green-400 text-green-900';
      case 'failed':
        return 'bg-red-100 border-red-400 text-red-900';
      case 'paused':
        return 'bg-yellow-100 border-yellow-400 text-yellow-900';
      default:
        return 'bg-gray-100 border-gray-400 text-gray-900';
    }
  };

  const getStatusEmoji = (status: AgentStatus) => {
    switch (status) {
      case 'running':
        return '⚙️';
      case 'completed':
        return '✅';
      case 'failed':
        return '❌';
      case 'paused':
        return '⏸️';
      default:
        return '⏳';
    }
  };

  const calculateDuration = (startTime: Date | null, endTime: Date | null) => {
    if (!startTime) return '';
    const end = endTime || new Date();
    const duration = Math.floor((end.getTime() - startTime.getTime()) / 1000);
    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
  };

  // Graph visualization
  const svgWidth = 1000;
  const svgHeight = 600;
  const nodeRadius = 40;
  const positions: { [key: string]: { x: number; y: number } } = {};

  // Calculate positions (simple grid layout)
  const levels: { [key: string]: number } = {};
  agents.forEach((agent) => {
    if (!agent.parentId) {
      levels[agent.id] = 0;
    }
  });

  agents.forEach((agent) => {
    if (agent.parentId && !(agent.id in levels)) {
      levels[agent.id] = (levels[agent.parentId] || 0) + 1;
    }
  });

  const levelCounts: { [key: number]: number } = {};
  const levelIndices: { [key: string]: number } = {};
  agents.forEach((agent) => {
    const level = levels[agent.id] || 0;
    levelCounts[level] = (levelCounts[level] || 0) + 1;
    levelIndices[agent.id] = levelCounts[level] - 1;
  });

  agents.forEach((agent) => {
    const level = levels[agent.id] || 0;
    const index = levelIndices[agent.id];
    const count = levelCounts[level] || 1;
    positions[agent.id] = {
      x: 100 + (index - count / 2) * 150,
      y: 100 + level * 150,
    };
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">🏭 Agent Orchestrator</h1>
          <p className="text-slate-400">Launch, monitor, and coordinate multiple agents in parallel</p>
        </div>

        {/* Controls */}
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 mb-6 flex justify-between items-center">
          <div className="flex gap-3">
            <button
              onClick={() => setShowLaunchModal(true)}
              className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg transition"
            >
              <Plus size={20} /> Launch Agent
            </button>
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg transition ${
                autoRefresh
                  ? 'bg-blue-600 hover:bg-blue-700 text-white'
                  : 'bg-slate-700 hover:bg-slate-600 text-slate-300'
              }`}
            >
              <RefreshCw size={20} /> Auto-refresh: {autoRefresh ? 'ON' : 'OFF'}
            </button>
          </div>
          <div className="text-slate-400 text-sm">
            {agents.length} agents • {agents.filter((a) => a.status === 'running').length} running •{' '}
            {agents.filter((a) => a.status === 'completed').length} completed
          </div>
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-3 gap-6">
          {/* Graph Visualization */}
          <div className="col-span-2 bg-slate-800 border border-slate-700 rounded-lg p-4 overflow-x-auto">
            <h2 className="text-lg font-semibold text-white mb-4">Agent Graph</h2>
            {agents.length === 0 ? (
              <div className="flex items-center justify-center h-80 text-slate-500">
                <p>No agents launched yet. Click "Launch Agent" to get started.</p>
              </div>
            ) : (
              <svg width={svgWidth} height={svgHeight} className="bg-slate-900 rounded border border-slate-700">
                {/* Draw connections */}
                {connections.map((conn, idx) => {
                  const from = positions[conn.from];
                  const to = positions[conn.to];
                  if (!from || !to) return null;

                  return (
                    <g key={idx}>
                      <line
                        x1={from.x}
                        y1={from.y}
                        x2={to.x}
                        y2={to.y}
                        stroke="#64748b"
                        strokeWidth="2"
                        markerEnd="url(#arrowhead)"
                      />
                      {conn.label && (
                        <text
                          x={(from.x + to.x) / 2}
                          y={(from.y + to.y) / 2 - 5}
                          textAnchor="middle"
                          fill="#94a3b8"
                          fontSize="12"
                        >
                          {conn.label}
                        </text>
                      )}
                    </g>
                  );
                })}

                {/* Arrow marker */}
                <defs>
                  <marker
                    id="arrowhead"
                    markerWidth="10"
                    markerHeight="10"
                    refX="9"
                    refY="3"
                    orient="auto"
                  >
                    <polygon points="0 0, 10 3, 0 6" fill="#64748b" />
                  </marker>
                </defs>

                {/* Draw nodes */}
                {agents.map((agent) => {
                  const pos = positions[agent.id];
                  if (!pos) return null;

                  const isSelected = selectedAgent === agent.id;
                  const statusColor = getStatusColor(agent.status);

                  return (
                    <g
                      key={agent.id}
                      onClick={() => setSelectedAgent(agent.id)}
                      className="cursor-pointer"
                    >
                      {/* Node circle */}
                      <circle
                        cx={pos.x}
                        cy={pos.y}
                        r={nodeRadius}
                        fill={isSelected ? '#3b82f6' : agent.status === 'running' ? '#1e40af' : '#334155'}
                        stroke={isSelected ? '#60a5fa' : '#64748b'}
                        strokeWidth={isSelected ? 3 : 2}
                      />

                      {/* Status emoji */}
                      <text
                        x={pos.x}
                        y={pos.y}
                        textAnchor="middle"
                        dominantBaseline="central"
                        fontSize="20"
                      >
                        {getStatusEmoji(agent.status)}
                      </text>

                      {/* Label */}
                      <text
                        x={pos.x}
                        y={pos.y + nodeRadius + 20}
                        textAnchor="middle"
                        fill="#e2e8f0"
                        fontSize="12"
                        className="font-semibold"
                      >
                        {agent.name.substring(0, 10)}
                      </text>

                      {/* Progress bar */}
                      {agent.status === 'running' && (
                        <rect
                          x={pos.x - 30}
                          y={pos.y + nodeRadius + 35}
                          width={60}
                          height="4"
                          fill="#475569"
                          rx="2"
                        />
                      )}
                      {agent.status === 'running' && (
                        <rect
                          x={pos.x - 30}
                          y={pos.y + nodeRadius + 35}
                          width={(60 * agent.progress) / 100}
                          height="4"
                          fill="#3b82f6"
                          rx="2"
                        />
                      )}
                    </g>
                  );
                })}
              </svg>
            )}
          </div>

          {/* Agent Details Sidebar */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 flex flex-col h-full">
            <h2 className="text-lg font-semibold text-white mb-4">Agent Details</h2>

            {selectedAgent ? (
              (() => {
                const agent = agents.find((a) => a.id === selectedAgent);
                if (!agent) return null;

                return (
                  <div className="space-y-4 flex-1">
                    {/* Agent Info */}
                    <div className="bg-slate-700 p-3 rounded">
                      <p className="text-sm font-semibold text-slate-300">Name</p>
                      <p className="text-white font-mono text-sm mt-1">{agent.name}</p>
                    </div>

                    <div className="bg-slate-700 p-3 rounded">
                      <p className="text-sm font-semibold text-slate-300">Type</p>
                      <p className="text-white font-mono text-sm mt-1">{agent.type}</p>
                    </div>

                    <div className="bg-slate-700 p-3 rounded">
                      <p className="text-sm font-semibold text-slate-300">Status</p>
                      <div className={`mt-1 inline-block px-2 py-1 rounded text-xs font-semibold ${getStatusColor(agent.status)}`}>
                        {getStatusEmoji(agent.status)} {agent.status}
                      </div>
                    </div>

                    <div className="bg-slate-700 p-3 rounded">
                      <p className="text-sm font-semibold text-slate-300">Progress</p>
                      <div className="mt-2">
                        <div className="bg-slate-600 rounded h-2 overflow-hidden">
                          <div
                            className="bg-green-500 h-full transition-all"
                            style={{ width: `${agent.progress}%` }}
                          />
                        </div>
                        <p className="text-xs text-slate-400 mt-1">{Math.round(agent.progress)}%</p>
                      </div>
                    </div>

                    <div className="bg-slate-700 p-3 rounded">
                      <p className="text-sm font-semibold text-slate-300">Duration</p>
                      <p className="text-white font-mono text-sm mt-1">
                        {calculateDuration(agent.startTime, agent.endTime)}
                      </p>
                    </div>

                    {agent.description && (
                      <div className="bg-slate-700 p-3 rounded">
                        <p className="text-sm font-semibold text-slate-300">Description</p>
                        <p className="text-slate-300 text-xs mt-1 line-clamp-3">{agent.description}</p>
                      </div>
                    )}

                    {/* Actions */}
                    <div className="mt-auto pt-4 space-y-2">
                      {agent.status === 'running' && (
                        <>
                          <button
                            onClick={() => pauseAgent(agent.id)}
                            className="w-full flex items-center justify-center gap-2 bg-yellow-600 hover:bg-yellow-700 text-white py-2 rounded text-sm transition"
                          >
                            <Pause size={16} /> Pause
                          </button>
                          <button
                            onClick={() => stopAgent(agent.id)}
                            className="w-full flex items-center justify-center gap-2 bg-red-600 hover:bg-red-700 text-white py-2 rounded text-sm transition"
                          >
                            <StopCircle size={16} /> Stop
                          </button>
                        </>
                      )}
                      {agent.status === 'paused' && (
                        <button
                          onClick={() => resumeAgent(agent.id)}
                          className="w-full flex items-center justify-center gap-2 bg-green-600 hover:bg-green-700 text-white py-2 rounded text-sm transition"
                        >
                          <Play size={16} /> Resume
                        </button>
                      )}
                      <button
                        onClick={() => deleteAgent(agent.id)}
                        className="w-full flex items-center justify-center gap-2 bg-slate-700 hover:bg-slate-600 text-slate-300 py-2 rounded text-sm transition"
                      >
                        <Trash2 size={16} /> Delete
                      </button>
                    </div>
                  </div>
                );
              })()
            ) : (
              <div className="flex items-center justify-center h-full text-slate-500">
                <p className="text-center">Select an agent to view details</p>
              </div>
            )}
          </div>
        </div>

        {/* Agent List */}
        <div className="mt-6 bg-slate-800 border border-slate-700 rounded-lg p-4">
          <h2 className="text-lg font-semibold text-white mb-4">Agent Instances</h2>
          {agents.length === 0 ? (
            <p className="text-slate-500 text-center py-4">No agents launched yet</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-700">
                    <th className="text-left py-2 px-3 text-slate-400 text-sm font-semibold">Agent</th>
                    <th className="text-left py-2 px-3 text-slate-400 text-sm font-semibold">Type</th>
                    <th className="text-left py-2 px-3 text-slate-400 text-sm font-semibold">Status</th>
                    <th className="text-left py-2 px-3 text-slate-400 text-sm font-semibold">Progress</th>
                    <th className="text-left py-2 px-3 text-slate-400 text-sm font-semibold">Duration</th>
                    <th className="text-center py-2 px-3 text-slate-400 text-sm font-semibold">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {agents.map((agent) => (
                    <tr
                      key={agent.id}
                      className="border-b border-slate-700 hover:bg-slate-700 transition cursor-pointer"
                      onClick={() => setSelectedAgent(agent.id)}
                    >
                      <td className="py-3 px-3 text-white font-mono text-sm">{agent.name}</td>
                      <td className="py-3 px-3 text-slate-400 text-sm">{agent.type}</td>
                      <td className="py-3 px-3">
                        <span className={`inline-block px-2 py-1 rounded text-xs font-semibold ${getStatusColor(agent.status)}`}>
                          {getStatusEmoji(agent.status)} {agent.status}
                        </span>
                      </td>
                      <td className="py-3 px-3">
                        <div className="flex items-center gap-2">
                          <div className="bg-slate-700 rounded h-2 w-24">
                            <div
                              className="bg-green-500 h-full transition-all rounded"
                              style={{ width: `${agent.progress}%` }}
                            />
                          </div>
                          <span className="text-slate-400 text-xs">{Math.round(agent.progress)}%</span>
                        </div>
                      </td>
                      <td className="py-3 px-3 text-slate-400 text-sm">
                        {calculateDuration(agent.startTime, agent.endTime)}
                      </td>
                      <td className="py-3 px-3 text-center">
                        {agent.status === 'running' && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              pauseAgent(agent.id);
                            }}
                            className="text-yellow-400 hover:text-yellow-300"
                          >
                            ⏸
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Launch Agent Modal */}
      {showLaunchModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-4">Launch New Agent</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Agent Name</label>
                <input
                  type="text"
                  value={newAgentConfig.name}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, name: e.target.value })}
                  placeholder="e.g., Bug Finder, Code Reviewer"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white placeholder-slate-500 focus:border-blue-500 focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Agent Type</label>
                <select
                  value={newAgentConfig.type}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, type: e.target.value as AgentType })}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white focus:border-blue-500 focus:outline-none"
                >
                  <option value="general-purpose">General Purpose</option>
                  <option value="code-reviewer">Code Reviewer</option>
                  <option value="plan">Planner</option>
                  <option value="explore">Explorer</option>
                  <option value="custom">Custom</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Description (Optional)</label>
                <textarea
                  value={newAgentConfig.description}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, description: e.target.value })}
                  placeholder="What should this agent do?"
                  rows={3}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white placeholder-slate-500 focus:border-blue-500 focus:outline-none resize-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Parent Agent (Optional)</label>
                <select
                  value={newAgentConfig.parentId}
                  onChange={(e) => setNewAgentConfig({ ...newAgentConfig, parentId: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white focus:border-blue-500 focus:outline-none"
                >
                  <option value="">No parent (independent)</option>
                  {agents.filter((a) => a.status !== 'failed').map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowLaunchModal(false)}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded transition"
              >
                Cancel
              </button>
              <button
                onClick={launchAgent}
                className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded transition"
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
