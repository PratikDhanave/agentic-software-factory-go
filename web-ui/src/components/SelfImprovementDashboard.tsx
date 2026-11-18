'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { apiClient, SelfImprovementMetrics, StateMachineExport } from '@/lib/api';
import {
  Activity,
  Brain,
  GitCommit,
  Loader,
  RefreshCw,
  Sparkles,
  TrendingUp,
  Zap,
} from 'lucide-react';

export const SelfImprovementDashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<SelfImprovementMetrics | null>(null);
  const [machines, setMachines] = useState<StateMachineExport[]>([]);
  const [loading, setLoading] = useState(true);
  const [simulating, setSimulating] = useState(false);
  const [learning, setLearning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    try {
      const [m, sm] = await Promise.all([
        apiClient.getSelfImprovementMetrics(),
        apiClient.getStateMachines().catch(() => ({ machines: [] })),
      ]);
      setMetrics(m);
      setMachines(sm.machines || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load self-improvement metrics');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
    const interval = setInterval(fetchAll, 10000);
    return () => clearInterval(interval);
  }, [fetchAll]);

  const handleSimulate = async (count: number) => {
    setSimulating(true);
    try {
      await apiClient.simulateLearningBatch(count);
      await fetchAll();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Simulation failed');
    } finally {
      setSimulating(false);
    }
  };

  const handleLearn = async () => {
    setLearning(true);
    try {
      await apiClient.triggerLearning();
      await fetchAll();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Pattern discovery failed');
    } finally {
      setLearning(false);
    }
  };

  if (loading) {
    return (
      <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-12 flex items-center justify-center gap-3 text-slate-400">
        <Loader className="w-5 h-5 animate-spin text-indigo-400" />
        Loading self-improvement &amp; lifecycle telemetry...
      </div>
    );
  }

  const agentEntries = Object.entries(metrics?.Agents || {});
  const totalDecisions = agentEntries.reduce((acc, [, a]) => acc + (a.DecisionCount || 0), 0);
  const avgSuccessRate =
    agentEntries.length > 0
      ? (agentEntries.reduce((acc, [, a]) => acc + (a.SuccessRate || 0), 0) / agentEntries.length) * 100
      : 0;
  const patterns = metrics?.Patterns || [];

  return (
    <div className="space-y-6">
      {/* Header & Controls */}
      <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-purple-500/10 border border-purple-500/20 text-purple-400">
              <Brain className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-semibold text-slate-100">
                Continuous Learning &amp; Declared State Machines
              </h2>
              <p className="text-xs text-slate-400">
                Tracks per-agent accuracy in SQLite, calibrates confidence bins, and promotes high-accuracy patterns
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            <button
              onClick={() => handleSimulate(30)}
              disabled={simulating}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-slate-800 text-white text-xs font-medium transition"
            >
              {simulating ? <Loader className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
              {simulating ? 'Recording...' : 'Simulate +30 Outcomes'}
            </button>
            <button
              onClick={handleLearn}
              disabled={learning}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-purple-300 border border-purple-500/30 text-xs font-medium transition"
            >
              {learning ? <Loader className="w-3.5 h-3.5 animate-spin" /> : <Sparkles className="w-3.5 h-3.5" />}
              Discover Patterns
            </button>
            <button
              onClick={fetchAll}
              className="p-2 rounded-xl bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-300"
              title="Refresh"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>

        {error && (
          <div className="mt-4 p-3.5 rounded-xl bg-red-950/50 border border-red-800/60 text-xs text-red-200">
            {error}
          </div>
        )}

        {/* KPI Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mt-6">
          <div className="p-4 rounded-xl bg-slate-950/90 border border-slate-800">
            <div className="text-xs text-slate-400">Evaluated Agents</div>
            <div className="text-2xl font-bold text-slate-100 mt-1">{agentEntries.length}</div>
            <div className="text-[11px] text-slate-500 mt-1">Specialized roles tracked</div>
          </div>
          <div className="p-4 rounded-xl bg-slate-950/90 border border-slate-800">
            <div className="text-xs text-slate-400">Mean Accuracy</div>
            <div className="text-2xl font-bold text-emerald-400 mt-1">{avgSuccessRate.toFixed(1)}%</div>
            <div className="text-[11px] text-slate-500 mt-1">Human + outcome verified</div>
          </div>
          <div className="p-4 rounded-xl bg-slate-950/90 border border-slate-800">
            <div className="text-xs text-slate-400">Discovered Patterns</div>
            <div className="text-2xl font-bold text-purple-400 mt-1">{patterns.length}</div>
            <div className="text-[11px] text-slate-500 mt-1">&ge;80% reliability threshold</div>
          </div>
          <div className="p-4 rounded-xl bg-slate-950/90 border border-slate-800">
            <div className="text-xs text-slate-400">Recorded Decisions</div>
            <div className="text-2xl font-bold text-indigo-400 mt-1">{totalDecisions}</div>
            <div className="text-[11px] text-slate-500 mt-1">SQLite learning corpus</div>
          </div>
        </div>
      </div>

      {/* Native Responsive SVG + Bar Visualizations */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
          <div className="flex items-center justify-between mb-5">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300 flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-emerald-400" />
              Agent Accuracy vs. Confidence Calibration
            </h3>
          </div>

          {agentEntries.length === 0 ? (
            <div className="py-12 text-center text-xs text-slate-500">
              No agent learning records yet. Click &ldquo;Simulate +30 Outcomes&rdquo; or generate a workflow to populate metrics.
            </div>
          ) : (
            <div className="space-y-4">
              {agentEntries.map(([agent, data]) => {
                const acc = Math.min(100, Math.max(0, data.SuccessRate * 100));
                const conf = Math.min(100, Math.max(0, data.Confidence * 100));
                return (
                  <div key={agent} className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-semibold text-slate-100">{agent}</span>
                      <span className="text-slate-400 font-mono">{data.DecisionCount} decisions</span>
                    </div>
                    <div className="space-y-1.5">
                      <div className="flex items-center gap-3 text-[11px]">
                        <span className="w-20 text-slate-400">Accuracy</span>
                        <div className="flex-1 h-2 bg-slate-800 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-emerald-500 rounded-full transition-all duration-500"
                            style={{ width: `${acc}%` }}
                          />
                        </div>
                        <span className="w-12 text-right font-mono text-emerald-300">{acc.toFixed(1)}%</span>
                      </div>
                      <div className="flex items-center gap-3 text-[11px]">
                        <span className="w-20 text-slate-400">Confidence</span>
                        <div className="flex-1 h-2 bg-slate-800 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-indigo-500 rounded-full transition-all duration-500"
                            style={{ width: `${conf}%` }}
                          />
                        </div>
                        <span className="w-12 text-right font-mono text-indigo-300">{conf.toFixed(1)}%</span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
          <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300 flex items-center gap-2 mb-5">
            <Sparkles className="w-4 h-4 text-purple-400" />
            High-Confidence Learned Patterns
          </h3>

          {patterns.length === 0 ? (
            <div className="py-12 text-center text-xs text-slate-500">
              Patterns crystallize once an agent accumulates &ge;20 verified decisions at &ge;80% accuracy.
            </div>
          ) : (
            <div className="space-y-3">
              {patterns.map((p, idx) => {
                const pct = Math.min(100, Math.max(0, p.SuccessRate * 100));
                return (
                  <div key={idx} className="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
                    <div className="flex items-center justify-between text-xs mb-2">
                      <span className="font-semibold text-slate-200">
                        {p.Agent} &rarr; <span className="text-purple-300">{p.IssueType}</span>
                      </span>
                      <span className="font-mono text-emerald-400">{pct.toFixed(1)}% success</span>
                    </div>
                    <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
                      <div className="h-full bg-purple-500 rounded-full" style={{ width: `${pct}%` }} />
                    </div>
                    <div className="text-[11px] text-slate-500 mt-1.5">
                      Applied across {p.Applied} historical decisions • eligible for Walk-phase auto-approval
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Declared State Machines Inspector */}
      {machines.length > 0 && (
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
          <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300 flex items-center gap-2 mb-4">
            <Activity className="w-4 h-4 text-indigo-400" />
            Declared Factory State Machines ({machines.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {machines.map((m) => (
              <div key={m.name} className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-slate-100 capitalize">{m.name}</span>
                  <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300">
                    init: {m.initial}
                  </span>
                </div>
                <div className="space-y-1.5 text-xs font-mono">
                  {Object.entries(m.transitions || {}).map(([from, tos]) => (
                    <div key={from} className="flex items-center gap-1.5 text-slate-400">
                      <GitCommit className="w-3 h-3 text-slate-600 flex-shrink-0" />
                      <span className="text-slate-200">{from}</span>
                      <span>&rarr;</span>
                      <span className="text-emerald-400 truncate">{(tos || []).join(' | ')}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* SQLite WAL Persistence Table Explorer */}
      {metrics?.Database?.tables && (
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
          <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300">
              SQLite WAL Persistence Tables
            </h3>
            <span className="text-xs font-mono text-slate-400">{metrics.Database.path}</span>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {Object.entries(metrics.Database.tables as Record<string, number>).map(([tbl, count]) => (
              <div key={tbl} className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
                <span className="text-xs font-mono text-slate-300 truncate">{tbl}</span>
                <span className="text-sm font-bold text-indigo-400 ml-2">{count}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default SelfImprovementDashboard;
