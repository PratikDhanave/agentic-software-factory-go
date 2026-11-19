'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { apiClient, StateMachineExport } from '@/lib/api';
import { FRONTEND_MACHINES, MachineExport, statusBadge } from '@/lib/stateMachine';
import {
  Check,
  CheckCircle2,
  Copy,
  Cpu,
  GitCommit,
  GitFork,
  MonitorSmartphone,
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

interface PositionedState {
  state: string;
  col: number;
  row: number;
  x: number;
  y: number;
}

const NODE_W = 164;
const NODE_H = 56;
const COL_GAP = 228;
const ROW_GAP = 82;

function layoutStateGraph(machine: MachineExport): {
  nodes: PositionedState[];
  byState: Record<string, PositionedState>;
  width: number;
  height: number;
} {
  const depth: Record<string, number> = { [machine.initial]: 0 };
  const queue: string[] = [machine.initial];

  while (queue.length > 0) {
    const cur = queue.shift()!;
    for (const nxt of machine.transitions[cur] || []) {
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
  const cols: string[][] = Array.from({ length: maxDepth + 1 }, () => []);
  for (const s of machine.states) {
    cols[depth[s]].push(s);
  }
  const nonEmptyCols = cols.filter((c) => c.length > 0);
  const maxRows = Math.max(1, ...nonEmptyCols.map((c) => c.length));
  const totalHeight = Math.max(240, maxRows * ROW_GAP + 84);
  const totalWidth = Math.max(540, nonEmptyCols.length * COL_GAP + 36);

  const nodes: PositionedState[] = [];
  const byState: Record<string, PositionedState> = {};

  nonEmptyCols.forEach((colStates, colIdx) => {
    const colHeight = colStates.length * NODE_H + (colStates.length - 1) * (ROW_GAP - NODE_H);
    const startY = Math.max(28, (totalHeight - 44 - colHeight) / 2);
    colStates.forEach((st, rowIdx) => {
      const pos: PositionedState = {
        state: st,
        col: colIdx,
        row: rowIdx,
        x: 28 + colIdx * COL_GAP,
        y: startY + rowIdx * ROW_GAP,
      };
      nodes.push(pos);
      byState[st] = pos;
    });
  });

  return { nodes, byState, width: totalWidth, height: totalHeight };
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
      // keep frontend machines visible even offline
    }
  }, []);

  useEffect(() => {
    loadBackend();
  }, [loadBackend]);

  const allMachines = useMemo(
    () => [...backendMachines, ...FRONTEND_MACHINES],
    [backendMachines]
  );

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

  const graph = useMemo(
    () => (activeMachine ? layoutStateGraph(activeMachine) : null),
    [activeMachine]
  );

  const allowedTargets = useMemo(() => {
    if (!activeMachine || !simState) return [];
    return activeMachine.transitions[simState] || [];
  }, [activeMachine, simState]);

  const handleSimulateStep = async (target: string) => {
    if (!activeMachine) return;
    if (target === simState) return;
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
        // fallback to local transition table
      }
    }

    if (!isAllowedLocal) {
      setSimFeedback({
        ok: false,
        msg: `Guarded: "${simState}" cannot jump directly to "${target}". Legal next states: [${
          allowedTargets.join(', ') || 'none — terminal state'
        }].`,
      });
      return;
    }

    setSimState(target);
    setSimHistory((prev) => [...prev, target]);
    setSimFeedback({
      ok: true,
      msg: `Transition validated: ${simState} → ${target}`,
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
    <div className="px-6 lg:px-8 py-7 max-w-[1260px] w-full space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[28px] leading-tight font-bold tracking-tight ide-ink">
            State Machines
          </h1>
          <p className="text-sm ide-ink-muted mt-1">
            {allMachines.length} explicit finite state machines governing assembly stages, multi-agent governance, self-healing retries, and UI lifecycles.
          </p>
        </div>

        <div className="flex items-center gap-2 text-xs">
          <span className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full ide-surface border ide-border ide-ink-secondary font-medium">
            <Cpu className="w-3.5 h-3.5" />
            {backendMachines.length} Go Backend
          </span>
          <span className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full ide-surface border ide-border ide-ink-secondary font-medium">
            <MonitorSmartphone className="w-3.5 h-3.5" />
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
              className={`px-3.5 py-1.5 rounded-full text-xs font-medium transition flex items-center gap-1.5 ${
                active
                  ? 'kora-nav-active'
                  : 'ide-surface border ide-border ide-ink-secondary hover:ide-ink'
              }`}
            >
              <span>{label}</span>
              <span className="text-[10px] font-mono ide-ink-muted">{count}</span>
            </button>
          );
        })}
      </div>

      {/* Split Explorer */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        {/* Left Registry List */}
        <aside className="lg:col-span-4 ide-surface border ide-border rounded-2xl overflow-hidden flex flex-col max-h-[680px]">
          <div className="px-5 py-3.5 border-b ide-border flex items-center justify-between">
            <span className="text-xs font-semibold ide-ink">
              Registered Machines ({filteredMachines.length})
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
                  className={`w-full text-left px-5 py-3.5 transition ${
                    selected
                      ? 'ide-subtle'
                      : 'hover:bg-black/[0.018] dark:hover:bg-white/[0.02]'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-mono font-semibold ide-ink truncate">
                      {m.name}
                    </span>
                    <span
                      className={`text-[10px] font-medium px-2 py-0.5 rounded-full ${
                        m.source === 'backend' ? 'kora-pill-healthy' : statusBadge.idle
                      }`}
                    >
                      {m.source === 'backend' ? 'Go' : 'UI'}
                    </span>
                  </div>
                  {m.description && (
                    <p className="text-[11px] ide-ink-muted mt-1 line-clamp-2">{m.description}</p>
                  )}
                </button>
              );
            })}
          </div>
        </aside>

        {/* Right Interactive SVG State Graph */}
        {activeMachine && graph && (
          <section className="lg:col-span-8 ide-surface border ide-border rounded-2xl overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b ide-border space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <GitFork className="w-4 h-4 ide-ink" />
                  <h2 className="text-base font-mono font-semibold ide-ink">
                    {activeMachine.name}
                  </h2>
                  {activeMachine.category && (
                    <span className="text-[11px] px-2.5 py-0.5 rounded-full ide-subtle ide-ink-secondary capitalize">
                      {activeMachine.category}
                    </span>
                  )}
                </div>

                <div className="flex items-center gap-1 p-1 rounded-full ide-canvas border ide-border text-xs">
                  {(
                    [
                      ['simulator', 'SVG Simulator'],
                      ['mermaid', 'Mermaid'],
                      ['json', 'MCP JSON'],
                    ] as const
                  ).map(([id, label]) => (
                    <button
                      key={id}
                      type="button"
                      onClick={() => setExportTab(id)}
                      className={`px-3 py-1 rounded-full font-medium transition ${
                        exportTab === id
                          ? 'bg-[#181816] text-white dark:bg-white dark:text-black'
                          : 'ide-ink-muted hover:ide-ink'
                      }`}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </div>
              {activeMachine.description && (
                <p className="text-xs ide-ink-muted">{activeMachine.description}</p>
              )}
            </div>

            {exportTab === 'simulator' && (
              <div className="p-6 space-y-5">
                {/* Active Simulator Bar */}
                <div className="px-4 py-3 rounded-xl ide-canvas border ide-border flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-2 text-xs">
                    <span className="ide-ink-muted">Current State:</span>
                    <span
                      className={`font-mono font-semibold px-3 py-0.5 rounded-full ${
                        statusBadge[simState] || statusBadge.running
                      }`}
                    >
                      {simState}
                    </span>
                    {activeMachine.terminal.includes(simState) && (
                      <span className="text-[11px] ide-ink-muted">(terminal)</span>
                    )}
                  </div>

                  <button
                    type="button"
                    onClick={resetSimulator}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1 rounded-full border ide-border ide-surface text-xs font-medium ide-ink-secondary hover:ide-ink transition"
                  >
                    <RotateCcw className="w-3 h-3" />
                    Reset to {activeMachine.initial}
                  </button>
                </div>

                {/* True Interactive SVG Directed State Graph (Forward + Retry Loopback Curves) */}
                <div className="bg-dag-dot-grid border ide-border rounded-2xl p-4 overflow-x-auto">
                  <div
                    className="relative"
                    style={{ width: `${graph.width}px`, height: `${graph.height}px` }}
                  >
                    <svg
                      className="absolute inset-0 w-full h-full pointer-events-none"
                      viewBox={`0 0 ${graph.width} ${graph.height}`}
                      fill="none"
                    >
                      <defs>
                        <marker
                          id="sm-arrow-neutral"
                          markerWidth="8"
                          markerHeight="8"
                          refX="7"
                          refY="4"
                          orient="auto"
                        >
                          <path d="M0,1 L7,4 L0,7 Z" fill="#A8A499" />
                        </marker>
                        <marker
                          id="sm-arrow-legal"
                          markerWidth="8"
                          markerHeight="8"
                          refX="7"
                          refY="4"
                          orient="auto"
                        >
                          <path d="M0,1 L7,4 L0,7 Z" fill="#285936" />
                        </marker>
                      </defs>

                      {Object.entries(activeMachine.transitions).flatMap(([from, tos]) =>
                        tos.map((to) => {
                          const a = graph.byState[from];
                          const b = graph.byState[to];
                          if (!a || !b || from === to) return null;

                          const isLegalOut = from === simState;
                          const stroke = isLegalOut ? '#285936' : '#C5C1B5';
                          const marker = isLegalOut
                            ? 'url(#sm-arrow-legal)'
                            : 'url(#sm-arrow-neutral)';

                          // Forward or vertical transition
                          if (b.col > a.col) {
                            const x1 = a.x + NODE_W;
                            const y1 = a.y + NODE_H / 2;
                            const x2 = b.x - 4;
                            const y2 = b.y + NODE_H / 2;
                            const cx = (x1 + x2) / 2;
                            return (
                              <path
                                key={`${from}->${to}`}
                                d={`M ${x1} ${y1} C ${cx} ${y1}, ${cx} ${y2}, ${x2} ${y2}`}
                                stroke={stroke}
                                strokeWidth={isLegalOut ? 2 : 1.25}
                                markerEnd={marker}
                              />
                            );
                          }

                          // Same-column transition
                          if (b.col === a.col) {
                            const x1 = a.x + NODE_W;
                            const y1 = a.y + NODE_H / 2;
                            const x2 = b.x + NODE_W;
                            const y2 = b.y + NODE_H / 2;
                            return (
                              <path
                                key={`${from}->${to}`}
                                d={`M ${x1} ${y1} C ${x1 + 36} ${y1}, ${x2 + 36} ${y2}, ${x2} ${y2}`}
                                stroke={stroke}
                                strokeWidth={isLegalOut ? 2 : 1.25}
                                strokeDasharray="4 3"
                                markerEnd={marker}
                              />
                            );
                          }

                          // Backward retry / self-healing loopback arc sweeping cleanly underneath
                          const x1 = a.x + NODE_W / 2;
                          const y1 = a.y + NODE_H;
                          const x2 = b.x + NODE_W / 2;
                          const y2 = b.y + NODE_H + 4;
                          const arcY = Math.min(graph.height - 10, Math.max(y1, y2) + 42);
                          return (
                            <path
                              key={`${from}->${to}`}
                              d={`M ${x1} ${y1} C ${x1} ${arcY}, ${x2} ${arcY}, ${x2} ${y2}`}
                              stroke={stroke}
                              strokeWidth={isLegalOut ? 2 : 1.25}
                              strokeDasharray="5 3"
                              markerEnd={marker}
                            />
                          );
                        })
                      )}
                    </svg>

                    {/* Clickable State Nodes */}
                    {graph.nodes.map((node) => {
                      const isCurrent = node.state === simState;
                      const isAllowed = allowedTargets.includes(node.state);
                      const isTerm = activeMachine.terminal.includes(node.state);

                      return (
                        <button
                          key={node.state}
                          type="button"
                          onClick={() => handleSimulateStep(node.state)}
                          title={activeMachine.state_descriptions?.[node.state] || node.state}
                          style={{
                            left: `${node.x}px`,
                            top: `${node.y}px`,
                            width: `${NODE_W}px`,
                            height: `${NODE_H}px`,
                          }}
                          className={`absolute text-left px-3.5 py-2 rounded-xl border transition flex flex-col justify-between ${
                            isCurrent
                              ? 'bg-[#181816] text-white border-[#181816] dark:bg-white dark:text-[#181816] shadow-md'
                              : isAllowed
                                ? 'ide-card border-[#285936] ring-2 ring-[#285936]/15 hover:bg-[#DCE9DF]/30'
                                : 'ide-card ide-border opacity-85 hover:opacity-100'
                          }`}
                        >
                          <div className="flex items-center justify-between gap-1.5">
                            <span className="font-mono text-xs font-semibold truncate">
                              {node.state}
                            </span>
                            {isTerm && !isCurrent && (
                              <CheckCircle2 className="w-3.5 h-3.5 ide-ink-muted shrink-0" />
                            )}
                          </div>
                          <span
                            className={`text-[10px] truncate ${
                              isCurrent
                                ? 'opacity-80'
                                : isAllowed
                                  ? 'text-[#285936] dark:text-emerald-300 font-medium'
                                  : 'ide-ink-muted'
                            }`}
                          >
                            {isCurrent
                              ? '● Active'
                              : isAllowed
                                ? 'Click to step →'
                                : isTerm
                                  ? 'Terminal'
                                  : node.state === activeMachine.initial
                                    ? 'Initial'
                                    : 'Guarded'}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {simFeedback && (
                  <div
                    className={`px-4 py-3 rounded-xl text-xs flex items-center gap-2 ${
                      simFeedback.ok ? 'kora-pill-healthy' : statusBadge.failed
                    }`}
                  >
                    {simFeedback.ok ? (
                      <CheckCircle2 className="w-4 h-4 shrink-0" />
                    ) : (
                      <ShieldAlert className="w-4 h-4 shrink-0" />
                    )}
                    <span>{simFeedback.msg}</span>
                  </div>
                )}

                {/* State Detail & Trace */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="p-4 rounded-xl ide-canvas border ide-border">
                    <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                      State Semantics
                    </div>
                    <p className="text-xs ide-ink-secondary leading-relaxed">
                      {activeMachine.state_descriptions?.[simState] ||
                        `State "${simState}" has ${allowedTargets.length} outgoing legal transition(s).`}
                    </p>
                  </div>

                  <div className="p-4 rounded-xl ide-canvas border ide-border">
                    <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                      Transition Trace ({simHistory.length})
                    </div>
                    <div className="flex flex-wrap items-center gap-1.5 text-xs font-mono">
                      {simHistory.map((step, i) => (
                        <React.Fragment key={`${step}-${i}`}>
                          <span className="px-2 py-0.5 rounded-md ide-surface border ide-border ide-ink">
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
              <div className="p-6 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs ide-ink-muted">Mermaid `stateDiagram-v2`</span>
                  <button
                    type="button"
                    onClick={() => copyExport(activeMachine.mermaid || '')}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink"
                  >
                    {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <pre className="p-4 rounded-xl bg-[#181816] text-[#F3F2ED] text-xs font-mono overflow-x-auto">
                  {activeMachine.mermaid}
                </pre>
              </div>
            )}

            {exportTab === 'json' && (
              <div className="p-6 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs ide-ink-muted">
                    AX Contract (`list_state_machines` / `factory://state-machines`)
                  </span>
                  <button
                    type="button"
                    onClick={() => copyExport(JSON.stringify(activeMachine, null, 2))}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink"
                  >
                    {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                    {copied ? 'Copied' : 'Copy JSON'}
                  </button>
                </div>
                <pre className="p-4 rounded-xl bg-[#181816] text-[#DCE9DF] text-xs font-mono overflow-x-auto max-h-[440px]">
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
