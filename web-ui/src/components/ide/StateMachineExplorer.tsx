'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { apiClient, StateMachineExport } from '@/lib/api';
import { FRONTEND_MACHINES, MachineExport, statusBadge } from '@/lib/stateMachine';
import {
  ArrowRight,
  Check,
  CheckCircle2,
  Copy,
  Cpu,
  GitCommit,
  Layers,
  MonitorSmartphone,
  Play,
  RotateCcw,
  ShieldAlert,
} from 'lucide-react';

const CATEGORY_LABELS: Record<string, string> = {
  all: 'All Machines',
  assembly: 'Factory Assembly',
  governance: 'Governance & HITL',
  orchestration: 'DAG & Streaming',
  execution: 'Workers & Queues',
  verification: 'Verify & Self-Heal',
  infrastructure: 'Integrations',
  ui: 'Frontend UI',
};

/** Computes BFS depth layers from `initial` so any state machine renders left-to-right cleanly. */
function computeStateLayers(machine: MachineExport): string[][] {
  const depth: Record<string, number> = { [machine.initial]: 0 };
  const queue: string[] = [machine.initial];

  while (queue.length > 0) {
    const cur = queue.shift()!;
    const nexts = machine.transitions[cur] || [];
    for (const nxt of nexts) {
      if (depth[nxt] === undefined) {
        depth[nxt] = depth[cur] + 1;
        queue.push(nxt);
      }
    }
  }

  for (const s of machine.states) {
    if (depth[s] === undefined) depth[s] = 1;
  }

  const maxDepth = Math.max(0, ...Object.values(depth));
  const layers: string[][] = Array.from({ length: maxDepth + 1 }, () => []);
  for (const s of machine.states) {
    layers[depth[s]].push(s);
  }
  return layers.filter((l) => l.length > 0);
}

export default function StateMachineExplorer() {
  const [backendMachines, setBackendMachines] = useState<MachineExport[]>([]);
  const [category, setCategory] = useState<string>('all');
  const [selectedName, setSelectedName] = useState<string>('factory-assembly-line');
  const [simState, setSimState] = useState<string>('');
  const [simHistory, setSimHistory] = useState<string[]>([]);
  const [simFeedback, setSimFeedback] = useState<{ ok: boolean; msg: string } | null>(null);
  const [exportTab, setExportTab] = useState<'simulator' | 'mermaid' | 'json'>('simulator');
  const [copied, setCopied] = useState(false);

  const loadBackend = useCallback(async () => {
    try {
      const res = await apiClient.getStateMachines();
      const list: MachineExport[] = (res.machines || []).map((m: StateMachineExport) => ({
        ...m,
        terminal:
          m.terminal ||
          m.states.filter((s) => !(m.transitions?.[s] && m.transitions[s].length > 0)),
        source: 'backend',
      }));
      setBackendMachines(list);
    } catch {
      // keep frontend machines visible even if backend is offline
    }
  }, []);

  useEffect(() => {
    loadBackend();
  }, [loadBackend]);

  const allMachines = useMemo(() => {
    return [...backendMachines, ...FRONTEND_MACHINES];
  }, [backendMachines]);

  const filteredMachines = useMemo(() => {
    if (category === 'all') return allMachines;
    return allMachines.filter((m) => (m.category || 'ui') === category);
  }, [allMachines, category]);

  const activeMachine = useMemo(() => {
    return (
      filteredMachines.find((m) => m.name === selectedName) ||
      allMachines.find((m) => m.name === selectedName) ||
      filteredMachines[0] ||
      FRONTEND_MACHINES[0]
    );
  }, [filteredMachines, allMachines, selectedName]);

  useEffect(() => {
    if (activeMachine) {
      setSimState(activeMachine.initial);
      setSimHistory([activeMachine.initial]);
      setSimFeedback(null);
    }
  }, [activeMachine]);

  const layers = useMemo(() => {
    return activeMachine ? computeStateLayers(activeMachine) : [];
  }, [activeMachine]);

  const allowedTargets = useMemo(() => {
    if (!activeMachine || !simState) return [];
    return activeMachine.transitions[simState] || [];
  }, [activeMachine, simState]);

  const handleSimulateStep = async (target: string) => {
    if (!activeMachine) return;
    const isAllowedLocal = (activeMachine.transitions[simState] || []).includes(target);

    if (activeMachine.source === 'backend') {
      try {
        const check = await apiClient.validateStateTransition(activeMachine.name, simState, target);
        if (!check.allowed) {
          setSimFeedback({
            ok: false,
            msg: check.error || `Transition "${simState}" → "${target}" is disallowed.`,
          });
          return;
        }
      } catch {
        // fallback to local transition table if offline
      }
    }

    if (!isAllowedLocal) {
      setSimFeedback({
        ok: false,
        msg: `Blocked: "${simState}" cannot transition directly to "${target}". Allowed from "${simState}": [${
          allowedTargets.join(', ') || 'none (terminal)'
        }].`,
      });
      return;
    }

    setSimState(target);
    setSimHistory((prev) => [...prev, target]);
    setSimFeedback({
      ok: true,
      msg: `Transitioned "${simState}" → "${target}"`,
    });
  };

  const resetSimulator = () => {
    if (!activeMachine) return;
    setSimState(activeMachine.initial);
    setSimHistory([activeMachine.initial]);
    setSimFeedback(null);
  };

  const copyExport = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Page Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold ide-ink">
            Explicit State Machines ({allMachines.length})
          </h1>
          <p className="text-sm ide-ink-secondary mt-1">
            Every backend entity, assembly pipeline, verification loop, and UI interaction is governed by a declarative state machine — eliminating implicit boolean flags and exposing formal contracts to users (UX), developers (DX), and MCP agents (AX).
          </p>
        </div>

        <div className="flex items-center gap-2 text-xs">
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg ide-surface border ide-border ide-ink-secondary font-medium">
            <Cpu className="w-3.5 h-3.5 text-blue-600" />
            {backendMachines.length} Go Backend
          </span>
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg ide-surface border ide-border ide-ink-secondary font-medium">
            <MonitorSmartphone className="w-3.5 h-3.5 text-emerald-600" />
            {FRONTEND_MACHINES.length} React UI
          </span>
        </div>
      </div>

      {/* Category Filter Pills */}
      <div className="flex flex-wrap items-center gap-1.5">
        {Object.entries(CATEGORY_LABELS).map(([key, label]) => {
          const active = category === key;
          const count =
            key === 'all'
              ? allMachines.length
              : allMachines.filter((m) => (m.category || 'ui') === key).length;
          return (
            <button
              key={key}
              type="button"
              onClick={() => setCategory(key)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition flex items-center gap-1.5 ${
                active
                  ? 'bg-slate-900 text-white dark:bg-blue-600'
                  : 'ide-surface border ide-border ide-ink-secondary hover:ide-ink'
              }`}
            >
              <span>{label}</span>
              <span
                className={`text-[10px] font-mono px-1.5 py-0.2 rounded-full ${
                  active ? 'bg-white/20 text-white' : 'ide-subtle ide-ink-muted'
                }`}
              >
                {count}
              </span>
            </button>
          );
        })}
      </div>

      {/* 2-Column Split Explorer */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Catalog */}
        <aside className="lg:col-span-4 ide-surface border ide-border rounded-xl overflow-hidden flex flex-col max-h-[680px]">
          <div className="px-4 py-3 border-b ide-border ide-subtle flex items-center justify-between">
            <span className="text-xs font-semibold ide-ink">
              Machine Registry ({filteredMachines.length})
            </span>
          </div>
          <div className="divide-y ide-border overflow-y-auto flex-1">
            {filteredMachines.map((m) => {
              const selected = activeMachine?.name === m.name;
              return (
                <button
                  key={m.name}
                  type="button"
                  onClick={() => setSelectedName(m.name)}
                  className={`w-full text-left px-4 py-3 transition ${
                    selected
                      ? 'bg-blue-600/[0.06] border-l-2 border-l-blue-600'
                      : 'hover:bg-black/[0.02] dark:hover:bg-white/[0.02]'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-mono font-semibold ide-ink truncate">
                      {m.name}
                    </span>
                    <span
                      className={`text-[10px] font-medium px-2 py-0.5 rounded-full ${
                        m.source === 'backend'
                          ? 'bg-blue-50 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300'
                          : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                      }`}
                    >
                      {m.source === 'backend' ? 'Go' : 'UI'}
                    </span>
                  </div>
                  {m.description && (
                    <p className="text-[11px] ide-ink-muted mt-1 line-clamp-2">{m.description}</p>
                  )}
                  <div className="mt-1.5 flex items-center gap-2 text-[10px] font-mono ide-ink-muted">
                    <span>{m.states.length} states</span>
                    <span>•</span>
                    <span>init: {m.initial}</span>
                  </div>
                </button>
              );
            })}
          </div>
        </aside>

        {/* Right Interactive Diagram & Simulator */}
        {activeMachine && (
          <section className="lg:col-span-8 ide-surface border ide-border rounded-xl overflow-hidden flex flex-col">
            <div className="p-5 border-b ide-border space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-2.5">
                  <Layers className="w-4 h-4 text-blue-600" />
                  <h2 className="text-base font-mono font-semibold ide-ink">
                    {activeMachine.name}
                  </h2>
                  {activeMachine.category && (
                    <span className="text-[11px] px-2 py-0.5 rounded ide-subtle ide-ink-secondary capitalize">
                      {activeMachine.category}
                    </span>
                  )}
                </div>

                {/* Mode Switcher */}
                <div className="flex items-center gap-1 p-1 rounded-lg ide-subtle border ide-border text-xs">
                  {(
                    [
                      ['simulator', 'Interactive Simulator'],
                      ['mermaid', 'Mermaid Diagram'],
                      ['json', 'AX JSON Schema'],
                    ] as const
                  ).map(([id, label]) => (
                    <button
                      key={id}
                      type="button"
                      onClick={() => setExportTab(id)}
                      className={`px-2.5 py-1 rounded-md font-medium transition ${
                        exportTab === id
                          ? 'ide-surface ide-ink shadow-sm'
                          : 'ide-ink-muted hover:ide-ink'
                      }`}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </div>

              {activeMachine.description && (
                <p className="text-xs ide-ink-secondary">{activeMachine.description}</p>
              )}
            </div>

            {exportTab === 'simulator' && (
              <div className="p-5 space-y-5">
                {/* Simulator Status Bar */}
                <div className="p-3.5 rounded-lg ide-canvas border ide-border flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-2 text-xs">
                    <span className="font-medium ide-ink-secondary">Current State:</span>
                    <span
                      className={`font-mono font-semibold px-2.5 py-0.5 rounded-md ${
                        statusBadge[simState] ||
                        'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-500/15 dark:text-blue-300'
                      }`}
                    >
                      {simState}
                    </span>
                    {activeMachine.terminal.includes(simState) && (
                      <span className="text-[11px] font-medium text-emerald-700 dark:text-emerald-300">
                        (Terminal State)
                      </span>
                    )}
                  </div>

                  <button
                    type="button"
                    onClick={resetSimulator}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md ide-surface border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink transition"
                  >
                    <RotateCcw className="w-3.5 h-3.5" />
                    Reset to {activeMachine.initial}
                  </button>
                </div>

                {/* Layered Interactive State Graph */}
                <div className="bg-dag-dot-grid border ide-border rounded-lg p-5 overflow-x-auto">
                  <div className="text-[11px] ide-ink-muted mb-3">
                    Click any state node below to test a transition from{' '}
                    <code className="font-mono ide-ink font-semibold">{simState}</code>:
                  </div>
                  <div className="flex items-stretch gap-5 min-w-max">
                    {layers.map((layerStates, layerIdx) => (
                      <React.Fragment key={layerIdx}>
                        <div className="flex flex-col justify-center gap-2.5 min-w-[164px]">
                          <div className="text-[10px] font-semibold uppercase tracking-wider ide-ink-muted text-center">
                            {layerIdx === 0 ? 'Initial' : `Stage ${layerIdx}`}
                          </div>
                          {layerStates.map((st) => {
                            const isCurrent = st === simState;
                            const isAllowed = allowedTargets.includes(st);
                            const isTerm = activeMachine.terminal.includes(st);
                            const desc = activeMachine.state_descriptions?.[st];

                            return (
                              <button
                                key={st}
                                type="button"
                                onClick={() => handleSimulateStep(st)}
                                title={desc || st}
                                className={`text-left px-3.5 py-2.5 rounded-lg border transition shadow-sm ${
                                  isCurrent
                                    ? 'bg-blue-600 text-white border-blue-600 ring-4 ring-blue-600/15'
                                    : isAllowed
                                      ? 'ide-surface border-emerald-500 hover:border-emerald-600 ring-1 ring-emerald-500/20'
                                      : 'ide-surface ide-border opacity-75 hover:opacity-100'
                                }`}
                              >
                                <div className="flex items-center justify-between gap-2">
                                  <span className="font-mono text-xs font-semibold truncate">
                                    {st}
                                  </span>
                                  {isCurrent && <Play className="w-3 h-3 fill-current shrink-0" />}
                                  {!isCurrent && isAllowed && (
                                    <ArrowRight className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                                  )}
                                  {!isCurrent && !isAllowed && isTerm && (
                                    <CheckCircle2 className="w-3.5 h-3.5 ide-ink-muted shrink-0" />
                                  )}
                                </div>
                                <div
                                  className={`text-[10px] mt-1 line-clamp-2 ${
                                    isCurrent ? 'text-blue-100' : 'ide-ink-muted'
                                  }`}
                                >
                                  {isCurrent
                                    ? 'Active state'
                                    : isAllowed
                                      ? 'Click to transition →'
                                      : isTerm
                                        ? 'Terminal'
                                        : 'Guarded'}
                                </div>
                              </button>
                            );
                          })}
                        </div>
                        {layerIdx < layers.length - 1 && (
                          <div className="flex items-center justify-center ide-ink-muted">
                            <ArrowRight className="w-4 h-4" />
                          </div>
                        )}
                      </React.Fragment>
                    ))}
                  </div>
                </div>

                {/* Validation Feedback Banner */}
                {simFeedback && (
                  <div
                    className={`p-3 rounded-lg border text-xs flex items-center gap-2 ${
                      simFeedback.ok
                        ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-800 dark:text-emerald-200'
                        : 'bg-rose-500/10 border-rose-500/30 text-rose-800 dark:text-rose-200'
                    }`}
                  >
                    {simFeedback.ok ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                    ) : (
                      <ShieldAlert className="w-4 h-4 text-rose-600 shrink-0" />
                    )}
                    <span>{simFeedback.msg}</span>
                  </div>
                )}

                {/* State Descriptions & Trace */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="p-4 rounded-lg ide-canvas border ide-border">
                    <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2">
                      Allowed Next Transitions from &ldquo;{simState}&rdquo;
                    </div>
                    {allowedTargets.length === 0 ? (
                      <p className="text-xs ide-ink-muted">
                        None — <code className="font-mono">{simState}</code> is a terminal state.
                      </p>
                    ) : (
                      <div className="flex flex-wrap gap-2">
                        {allowedTargets.map((nxt) => (
                          <button
                            key={nxt}
                            type="button"
                            onClick={() => handleSimulateStep(nxt)}
                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md ide-surface border border-emerald-500/40 text-xs font-mono font-medium ide-ink hover:bg-emerald-500/10 transition"
                          >
                            <span>{nxt}</span>
                            <ArrowRight className="w-3 h-3 text-emerald-600" />
                          </button>
                        ))}
                      </div>
                    )}
                    {activeMachine.state_descriptions?.[simState] && (
                      <p className="text-xs ide-ink-secondary mt-3 pt-3 border-t ide-border">
                        {activeMachine.state_descriptions[simState]}
                      </p>
                    )}
                  </div>

                  <div className="p-4 rounded-lg ide-canvas border ide-border">
                    <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2">
                      Simulated Transition Trace ({simHistory.length})
                    </div>
                    <div className="flex flex-wrap items-center gap-1.5 text-xs font-mono">
                      {simHistory.map((step, i) => (
                        <React.Fragment key={`${step}-${i}`}>
                          <span className="px-2 py-0.5 rounded ide-surface border ide-border ide-ink">
                            {step}
                          </span>
                          {i < simHistory.length - 1 && (
                            <GitCommit className="w-3.5 h-3.5 ide-ink-muted" />
                          )}
                        </React.Fragment>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {exportTab === 'mermaid' && (
              <div className="p-5 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs ide-ink-secondary">
                    Mermaid <code className="font-mono">stateDiagram-v2</code> (embeddable in ADRs &amp; docs)
                  </span>
                  <button
                    type="button"
                    onClick={() => copyExport(activeMachine.mermaid || '')}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md ide-surface border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink"
                  >
                    {copied ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                    {copied ? 'Copied' : 'Copy Mermaid'}
                  </button>
                </div>
                <pre className="p-4 rounded-lg bg-slate-950 text-slate-100 text-xs font-mono overflow-x-auto">
                  {activeMachine.mermaid}
                </pre>
              </div>
            )}

            {exportTab === 'json' && (
              <div className="p-5 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs ide-ink-secondary">
                    MCP Agent Contract (<code className="font-mono">list_state_machines</code> /{' '}
                    <code className="font-mono">factory://state-machines</code>)
                  </span>
                  <button
                    type="button"
                    onClick={() => copyExport(JSON.stringify(activeMachine, null, 2))}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md ide-surface border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink"
                  >
                    {copied ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                    {copied ? 'Copied' : 'Copy JSON'}
                  </button>
                </div>
                <pre className="p-4 rounded-lg bg-slate-950 text-emerald-300 text-xs font-mono overflow-x-auto max-h-[460px]">
                  {JSON.stringify(activeMachine, null, 2)}
                </pre>
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
