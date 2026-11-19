'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { apiClient, SelfImprovementMetrics } from '@/lib/api';
import { learningLabMachine, useMachine } from '@/lib/stateMachine';
import { BarChart3, Loader2, RefreshCw, Sparkles, Zap } from 'lucide-react';

export default function SelfImprovementDashboard() {
  const [metrics, setMetrics] = useState<SelfImprovementMetrics | null>(null);
  const lab = useMachine(learningLabMachine);
  const [error, setError] = useState<string | null>(null);

  const loadMetrics = useCallback(async () => {
    try {
      const m = await apiClient.getSelfImprovementMetrics();
      setMetrics(m);
      setError(null);
      lab.send('ready');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load learning telemetry');
      lab.send('error');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    loadMetrics();
    const timer = setInterval(loadMetrics, 6000);
    return () => clearInterval(timer);
  }, [loadMetrics]);

  const handleSimulateBatch = async () => {
    if (!lab.send('simulating')) return;
    setError(null);
    try {
      await apiClient.simulateLearningBatch(25);
      await loadMetrics();
    } catch (err) {
      lab.send('error');
      setError(err instanceof Error ? err.message : 'Batch simulation failed');
    }
  };

  const handleTriggerLearning = async () => {
    if (!lab.send('calibrating')) return;
    setError(null);
    try {
      await apiClient.triggerLearning();
      await loadMetrics();
    } catch (err) {
      lab.send('error');
      setError(err instanceof Error ? err.message : 'Pattern calibration failed');
    }
  };

  const agentsEntries = Object.entries(metrics?.Agents || {});
  const patterns = metrics?.Patterns || [];

  return (
    <div className="space-y-6">
      {/* Header & Controls */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight ide-ink">Learning &amp; Usage</h1>
          <p className="text-sm ide-ink-muted mt-0.5">
            SQLite-persisted agent confidence calibration, verified outcome ratios, and discovered execution patterns.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            disabled={lab.current === 'simulating'}
            onClick={handleSimulateBatch}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full ide-surface border ide-border text-xs font-medium ide-ink hover:ide-subtle transition"
          >
            {lab.current === 'simulating' ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Zap className="w-3.5 h-3.5" />
            )}
            Simulate 25 Outcomes
          </button>

          <button
            type="button"
            disabled={lab.current === 'calibrating'}
            onClick={handleTriggerLearning}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full bg-[#181816] dark:bg-white dark:text-black text-white text-xs font-medium transition"
          >
            {lab.current === 'calibrating' ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Sparkles className="w-3.5 h-3.5" />
            )}
            Recalibrate Patterns
          </button>

          <button
            type="button"
            onClick={loadMetrics}
            className="p-2 rounded-full ide-surface border ide-border ide-ink-secondary hover:ide-ink"
            title="Refresh"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {error && (
        <div className="p-3.5 rounded-xl bg-[#F5DCDC] text-[#7D2626] text-xs">{error}</div>
      )}

      {/* KPI Strip */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="ide-surface border ide-border rounded-2xl p-5">
          <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
            CALIBRATED AGENTS
          </div>
          <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
            {agentsEntries.length}
          </div>
          <div className="text-xs ide-ink-muted mt-1">tracked in SQLite WAL</div>
        </div>

        <div className="ide-surface border ide-border rounded-2xl p-5">
          <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
            DISCOVERED PATTERNS
          </div>
          <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
            {patterns.length}
          </div>
          <div className="text-xs ide-ink-muted mt-1">eligible for Walk auto-approval</div>
        </div>

        <div className="ide-surface border ide-border rounded-2xl p-5">
          <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
            TOTAL DECISIONS
          </div>
          <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
            {metrics?.Database?.total_agent_decisions ??
              agentsEntries.reduce((acc, [, m]) => acc + (m.DecisionCount || 0), 0)}
          </div>
          <div className="text-xs ide-ink-muted mt-1">recorded outcomes</div>
        </div>
      </div>

      {/* Agent Calibration Table */}
      <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
        <div className="px-6 py-4 flex items-center justify-between border-b ide-border">
          <div>
            <h3 className="text-base font-semibold ide-ink">Agent Calibration Ledger</h3>
            <p className="text-xs ide-ink-muted mt-0.5">
              Empirical accuracy vs. calibrated confidence across specialized factory roles.
            </p>
          </div>
          <BarChart3 className="w-4 h-4 ide-ink-muted" />
        </div>

        {agentsEntries.length === 0 ? (
          <div className="p-10 text-center text-xs ide-ink-muted">
            Run a Factory Build or click &ldquo;Simulate 25 Outcomes&rdquo; above to populate empirical agent calibration metrics.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                  <th className="py-3 px-6 font-medium">AGENT</th>
                  <th className="py-3 px-6 font-medium">STATUS</th>
                  <th className="py-3 px-6 font-medium">SUCCESS RATE</th>
                  <th className="py-3 px-6 font-medium">CALIBRATED CONFIDENCE</th>
                  <th className="py-3 px-6 font-medium text-right">DECISIONS</th>
                </tr>
              </thead>
              <tbody className="divide-y ide-border text-sm">
                {agentsEntries.map(([role, stat]) => {
                  const sr = Math.round((stat.SuccessRate || 0) * 100);
                  const conf = Math.round((stat.Confidence || 0) * 100);
                  return (
                    <tr key={role} className="hover:bg-black/[0.015] dark:hover:bg-white/[0.02]">
                      <td className="py-3.5 px-6">
                        <div className="font-semibold capitalize ide-ink">Factory {role}</div>
                        <div className="text-xs font-mono ide-ink-muted">factory-{role}</div>
                      </td>
                      <td className="py-3.5 px-6">
                        <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                          <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                          Calibrated
                        </span>
                      </td>
                      <td className="py-3.5 px-6 font-mono text-xs ide-ink tabular-nums">{sr}%</td>
                      <td className="py-3.5 px-6 font-mono text-xs ide-ink tabular-nums">{conf}%</td>
                      <td className="py-3.5 px-6 font-mono text-xs ide-ink-muted text-right tabular-nums">
                        {stat.DecisionCount}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Learned Patterns Table */}
      {patterns.length > 0 && (
        <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b ide-border">
            <h3 className="text-base font-semibold ide-ink">Learned High-Confidence Patterns</h3>
            <p className="text-xs ide-ink-muted mt-0.5">
              Patterns &ge; 85% success rate automatically bypass human gates in Walk mode.
            </p>
          </div>
          <div className="divide-y ide-border">
            {patterns.map((p, idx) => (
              <div
                key={`${p.IssueType}-${p.Agent}-${idx}`}
                className="px-6 py-3.5 flex items-center justify-between text-xs"
              >
                <div>
                  <span className="font-semibold ide-ink">{p.IssueType}</span>
                  <span className="ide-ink-muted mx-2">→</span>
                  <span className="font-mono ide-ink-secondary">factory-{p.Agent}</span>
                </div>
                <div className="flex items-center gap-3 font-mono">
                  <span className="ide-ink-muted">{p.Applied}x applied</span>
                  <span className="kora-pill-healthy px-2.5 py-0.5 rounded-full font-medium">
                    {Math.round((p.SuccessRate || 0) * 100)}% success
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
