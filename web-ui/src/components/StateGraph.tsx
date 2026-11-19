'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Play, RotateCcw, ShieldAlert, ShieldCheck } from 'lucide-react';
import { MachineExport } from '@/lib/stateMachine';
import { api } from '@/lib/api';

interface StateGraphProps {
  machine: MachineExport;
}

interface PositionedNode {
  state: string;
  x: number;
  y: number;
  col: number;
}

export function StateGraph({ machine }: StateGraphProps) {
  const [simState, setSimState] = useState<string>(machine.initial);
  const [simFeedback, setSimFeedback] = useState<{
    valid: boolean;
    message: string;
  } | null>(null);
  const [copiedMermaid, setCopiedMermaid] = useState(false);

  useEffect(() => {
    setSimState(machine.initial);
    setSimFeedback(null);
  }, [machine.name, machine.initial]);

  // Compute deterministic left-to-right BFS columns from initial state
  const layout = useMemo(() => {
    const depth: Record<string, number> = { [machine.initial]: 0 };
    const queue: string[] = [machine.initial];
    while (queue.length > 0) {
      const cur = queue.shift()!;
      const nexts = machine.transitions[cur] || [];
      for (const nx of nexts) {
        if (depth[nx] === undefined) {
          depth[nx] = depth[cur] + 1;
          queue.push(nx);
        }
      }
    }
    for (const s of machine.states) {
      if (depth[s] === undefined) depth[s] = 0;
    }

    const byCol: Record<number, string[]> = {};
    let maxCol = 0;
    for (const s of machine.states) {
      const c = depth[s];
      if (c > maxCol) maxCol = c;
      if (!byCol[c]) byCol[c] = [];
      byCol[c].push(s);
    }

    const colSpacing = 195;
    const rowSpacing = 86;
    const width = Math.max(660, (maxCol + 1) * colSpacing + 120);
    let maxRows = 1;
    Object.values(byCol).forEach((arr) => {
      if (arr.length > maxRows) maxRows = arr.length;
    });
    const height = Math.max(250, maxRows * rowSpacing + 110);

    const nodes: Record<string, PositionedNode> = {};
    for (let c = 0; c <= maxCol; c++) {
      const colStates = byCol[c] || [];
      const totalH = (colStates.length - 1) * rowSpacing;
      const startY = height / 2 - totalH / 2;
      colStates.forEach((st, idx) => {
        nodes[st] = {
          state: st,
          col: c,
          x: 105 + c * colSpacing,
          y: startY + idx * rowSpacing,
        };
      });
    }

    const edges: { from: string; to: string; backEdge: boolean }[] = [];
    for (const [from, tos] of Object.entries(machine.transitions)) {
      for (const to of tos || []) {
        if (nodes[from] && nodes[to]) {
          edges.push({
            from,
            to,
            backEdge: nodes[to].col <= nodes[from].col,
          });
        }
      }
    }

    return { nodes, edges, width, height };
  }, [machine]);

  const allowedNext = machine.transitions[simState] || [];

  const attemptTransition = async (target: string) => {
    if (machine.origin === 'frontend') {
      const ok = allowedNext.includes(target);
      if (ok) {
        setSimState(target);
        setSimFeedback({
          valid: true,
          message: `Transitioned ${simState} -> ${target}`,
        });
      } else {
        setSimFeedback({
          valid: false,
          message: `${machine.name}: illegal transition "${simState}" -> "${target}" (allowed: [${allowedNext.join(', ')}])`,
        });
      }
      return;
    }

    try {
      const res = await api.validateTransition(machine.name, simState, target);
      if (res.valid) {
        setSimState(target);
        setSimFeedback({
          valid: true,
          message: `Validated by Go engine: ${simState} -> ${target}`,
        });
      } else {
        setSimFeedback({
          valid: false,
          message: res.reason || `Illegal transition ${simState} -> ${target}`,
        });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Validation failed';
      setSimFeedback({ valid: false, message: msg });
    }
  };

  const copyMermaid = async () => {
    await navigator.clipboard.writeText(machine.mermaid);
    setCopiedMermaid(true);
    setTimeout(() => setCopiedMermaid(false), 1600);
  };

  return (
    <div className="space-y-4">
      <div className="rk-card-elevated p-5">
        <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-lg font-semibold font-mono">{machine.name}</h3>
              <span className="px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-[var(--bg-subtle)] text-[var(--text-secondary)]">
                {machine.category}
              </span>
              <span className="rk-pill-healthy px-2.5 py-0.5 rounded-full text-[11px] font-medium">
                {machine.origin === 'frontend' ? 'Frontend TS' : 'Go Engine'}
              </span>
            </div>
            <p className="text-sm text-[var(--text-secondary)] mt-1">
              {machine.description}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                setSimState(machine.initial);
                setSimFeedback(null);
              }}
              className="rk-btn-secondary inline-flex items-center gap-1.5 px-3 py-1.5 text-xs"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              Reset ({machine.initial})
            </button>
            <button
              onClick={copyMermaid}
              className="rk-btn-secondary inline-flex items-center gap-1.5 px-3 py-1.5 text-xs"
            >
              {copiedMermaid ? (
                <Check className="w-3.5 h-3.5 text-[#1F5E38]" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
              {copiedMermaid ? 'Copied Mermaid' : 'Copy Mermaid'}
            </button>
          </div>
        </div>

        {/* Interactive SVG Directed Graph */}
        <div className="overflow-x-auto rounded-xl border border-[var(--border-hairline)] bg-[var(--bg-card)] p-3">
          <svg
            width={layout.width}
            height={layout.height}
            viewBox={`0 0 ${layout.width} ${layout.height}`}
            className="mx-auto block select-none"
          >
            <defs>
              <marker
                id="arrow-muted"
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth="6"
                markerHeight="6"
                orient="auto-start-reverse"
              >
                <path d="M 0 1 L 10 5 L 0 9 z" fill="#8A877D" />
              </marker>
              <marker
                id="arrow-active"
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth="6.5"
                markerHeight="6.5"
                orient="auto-start-reverse"
              >
                <path d="M 0 1 L 10 5 L 0 9 z" fill="#1F5E38" />
              </marker>
            </defs>

            {layout.edges.map((e) => {
              const a = layout.nodes[e.from];
              const b = layout.nodes[e.to];
              const isActiveEdge = e.from === simState;
              const nodeHalfW = 66;

              let d = '';
              if (e.backEdge) {
                const startX = a.x;
                const startY = a.y - 22;
                const endX = b.x;
                const endY = b.y - 22;
                const arcY = Math.min(startY, endY) - 44;
                d = `M ${startX} ${startY} C ${startX} ${arcY}, ${endX} ${arcY}, ${endX} ${endY}`;
              } else {
                const startX = a.x + nodeHalfW;
                const startY = a.y;
                const endX = b.x - nodeHalfW;
                const endY = b.y;
                const midX = (startX + endX) / 2;
                d = `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
              }

              return (
                <path
                  key={`${e.from}->${e.to}`}
                  d={d}
                  fill="none"
                  stroke={isActiveEdge ? '#1F5E38' : '#B8B5AA'}
                  strokeWidth={isActiveEdge ? 2.2 : 1.4}
                  strokeDasharray={e.backEdge ? '5 4' : undefined}
                  markerEnd={isActiveEdge ? 'url(#arrow-active)' : 'url(#arrow-muted)'}
                />
              );
            })}

            {Object.values(layout.nodes).map((n) => {
              const isCurrent = n.state === simState;
              const isLegalNext = allowedNext.includes(n.state);
              const isTerminal = machine.terminal.includes(n.state);

              let fill = '#FFFFFF';
              let stroke = '#DFDDD3';
              let textFill = '#171715';
              if (isCurrent) {
                fill = '#171715';
                stroke = '#171715';
                textFill = '#FAF9F5';
              } else if (isLegalNext) {
                fill = '#DCE9DF';
                stroke = '#1F5E38';
                textFill = '#1F5E38';
              } else if (isTerminal) {
                fill = '#F1F0EA';
                stroke = '#8A877D';
              }

              return (
                <g
                  key={n.state}
                  onClick={() => attemptTransition(n.state)}
                  className="cursor-pointer"
                >
                  <rect
                    x={n.x - 66}
                    y={n.y - 21}
                    width={132}
                    height={42}
                    rx={11}
                    fill={fill}
                    stroke={stroke}
                    strokeWidth={isCurrent || isLegalNext ? 2 : 1.2}
                  />
                  <text
                    x={n.x}
                    y={n.y + 4}
                    textAnchor="middle"
                    fill={textFill}
                    fontSize="11.5"
                    fontFamily="JetBrains Mono, ui-monospace, monospace"
                    fontWeight={isCurrent || isLegalNext ? 600 : 500}
                  >
                    {n.state}
                  </text>
                </g>
              );
            })}
          </svg>
        </div>

        {/* Interactive Simulator Strip */}
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3 pt-3 border-t border-[var(--border-hairline)]">
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <span className="text-[var(--text-secondary)]">Current State:</span>
            <span className="px-2.5 py-1 rounded-md font-mono font-semibold bg-[var(--text-primary)] text-[var(--bg-elevated)]">
              {simState}
            </span>
            <span className="text-[var(--text-muted)] ml-1">
              Click any node above to test transition guards:
            </span>
            {allowedNext.length === 0 ? (
              <span className="px-2 py-0.5 rounded bg-[var(--bg-subtle)] text-[var(--text-secondary)] font-mono">
                [terminal state]
              </span>
            ) : (
              allowedNext.map((nx) => (
                <button
                  key={nx}
                  onClick={() => attemptTransition(nx)}
                  className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md font-mono rk-pill-healthy hover:opacity-85 transition"
                >
                  <Play className="w-3 h-3" />
                  {nx}
                </button>
              ))
            )}
          </div>

          {simFeedback && (
            <div
              className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-mono ${
                simFeedback.valid ? 'rk-pill-healthy' : 'rk-pill-err'
              }`}
            >
              {simFeedback.valid ? (
                <ShieldCheck className="w-3.5 h-3.5" />
              ) : (
                <ShieldAlert className="w-3.5 h-3.5" />
              )}
              <span>{simFeedback.message}</span>
            </div>
          )}
        </div>
      </div>

      {/* State Reference Table */}
      {machine.state_descriptions && (
        <div className="rk-card p-4">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-3">
            State Invariants & Allowed Transitions
          </h4>
          <div className="divide-y divide-[var(--border-hairline)] text-sm">
            {machine.states.map((st) => (
              <div
                key={st}
                className="py-2.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2"
              >
                <div className="flex items-center gap-2.5">
                  <span className="font-mono text-xs font-semibold px-2 py-0.5 rounded bg-[var(--bg-subtle)]">
                    {st}
                  </span>
                  <span className="text-[var(--text-secondary)] text-xs">
                    {machine.state_descriptions?.[st] || ''}
                  </span>
                </div>
                <div className="font-mono text-[11px] text-[var(--text-muted)]">
                  {(machine.transitions[st] || []).length > 0
                    ? `-> ${(machine.transitions[st] || []).join(', ')}`
                    : 'terminal'}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
