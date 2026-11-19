'use client';

import React, { useMemo, useState } from 'react';
import { CheckCircle2, Clock, Loader2, PauseCircle, ShieldAlert, Sparkles } from 'lucide-react';

interface DecisionNodeData {
  agent_role: string;
  decision?: string;
  choice?: string;
  reasoning?: string;
  confidence?: number;
  alternatives?: string[];
  needs_approval?: boolean;
}

interface DagNodeSpec {
  id: string;
  role: string;
  title: string;
  stageLabel: string;
  col: number;
  row: number;
  summary: string;
}

const DAG_NODES: DagNodeSpec[] = [
  {
    id: 'spec',
    role: 'spec',
    title: 'Spec Synthesizer',
    stageLabel: 'Intake',
    col: 0,
    row: 1,
    summary: 'Transforms plain-English requirements into structured Go invariants and acceptance steps.',
  },
  {
    id: 'product',
    role: 'product',
    title: 'Product Agent',
    stageLabel: 'Scope',
    col: 1,
    row: 0,
    summary: 'Validates user-facing API ergonomics, edge cases, and acceptance criteria.',
  },
  {
    id: 'architect',
    role: 'architect',
    title: 'Architect Agent',
    stageLabel: 'Design',
    col: 1,
    row: 1,
    summary: 'Defines concurrency primitives, lock discipline, and package boundaries.',
  },
  {
    id: 'security',
    role: 'security',
    title: 'Security Agent',
    stageLabel: 'Policy',
    col: 1,
    row: 2,
    summary: 'Audits memory bounds, replay safety, input validation, and RBAC rules.',
  },
  {
    id: 'design',
    role: 'design',
    title: 'API Design Agent',
    stageLabel: 'Contracts',
    col: 2,
    row: 0.5,
    summary: 'Refines exported Go types, constructor signatures, and error contracts.',
  },
  {
    id: 'developer',
    role: 'developer',
    title: 'Developer Agent',
    stageLabel: 'CodeGen',
    col: 2,
    row: 1.5,
    summary: 'Synthesizes idiomatic multi-file Go source code inside the isolated sandbox.',
  },
  {
    id: 'qa',
    role: 'qa',
    title: 'QA & Race Tester',
    stageLabel: 'Verify',
    col: 3,
    row: 0,
    summary: 'Runs table-driven unit tests and multi-goroutine go test -race checks.',
  },
  {
    id: 'performance',
    role: 'performance',
    title: 'Performance Agent',
    stageLabel: 'Verify',
    col: 3,
    row: 1,
    summary: 'Verifies lock contention, O(1) hot paths, and allocation behavior.',
  },
  {
    id: 'compliance',
    role: 'compliance',
    title: 'Compliance Agent',
    stageLabel: 'Verify',
    col: 3,
    row: 2,
    summary: 'Checks ADR constraints (no panics, clean go vet, deterministic behavior).',
  },
  {
    id: 'judge',
    role: 'judge',
    title: 'Judge & Delivery',
    stageLabel: 'Release',
    col: 4,
    row: 1,
    summary: 'Scores 5 engineering rubrics and unlocks promotion to generated/.',
  },
];

const DAG_EDGES: Array<[string, string]> = [
  ['spec', 'product'],
  ['spec', 'architect'],
  ['spec', 'security'],
  ['product', 'design'],
  ['architect', 'design'],
  ['architect', 'developer'],
  ['security', 'developer'],
  ['design', 'qa'],
  ['developer', 'qa'],
  ['developer', 'performance'],
  ['security', 'compliance'],
  ['developer', 'compliance'],
  ['qa', 'judge'],
  ['performance', 'judge'],
  ['compliance', 'judge'],
];

const NODE_W = 166;
const NODE_H = 66;
const COL_X = [24, 232, 440, 648, 856];
const ROW_Y = (row: number) => 28 + row * 94;

function computeNodeState(
  node: DagNodeSpec,
  runStage: string,
  runStatus: string,
  decision?: DecisionNodeData
): 'done' | 'active' | 'gate' | 'idle' {
  if (!runStage) return 'idle';
  if (runStatus === 'completed') return 'done';

  if (decision?.needs_approval && runStatus === 'awaiting_approval') {
    return 'gate';
  }

  const stageRank: Record<string, number> = {
    queued: 0,
    spec_synthesis: 0,
    agent_governance: 1,
    awaiting_approval: 1,
    workspace_init: 2,
    code_synthesis: 2,
    verification_loop: 3,
    judge_evaluation: 4,
    completed: 5,
  };
  const currentCol = stageRank[runStage] ?? 0;

  if (node.col < currentCol) return 'done';
  if (node.col === currentCol) {
    if (runStatus === 'awaiting_approval') return 'gate';
    if (runStatus === 'running') return 'active';
  }
  return 'idle';
}

interface Props {
  stage?: string;
  status?: string;
  decisions?: DecisionNodeData[];
  judgeScore?: number;
  specId?: string;
}

export default function AgentDagCanvas({
  stage = '',
  status = '',
  decisions = [],
  judgeScore,
  specId,
}: Props) {
  const [selectedNodeId, setSelectedNodeId] = useState<string>('architect');

  const decisionByRole = useMemo(() => {
    const map: Record<string, DecisionNodeData> = {};
    for (const d of decisions) {
      if (d?.agent_role) {
        map[d.agent_role.toLowerCase()] = d;
      }
    }
    return map;
  }, [decisions]);

  const selectedNode = DAG_NODES.find((n) => n.id === selectedNodeId) || DAG_NODES[2];
  const selectedDecision = decisionByRole[selectedNode.role];

  const nodeMap = useMemo(() => {
    const m: Record<string, DagNodeSpec> = {};
    for (const n of DAG_NODES) m[n.id] = n;
    return m;
  }, []);

  return (
    <div className="space-y-4">
      {/* Interactive Vector DAG Canvas */}
      <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
        <div className="px-5 py-3 border-b ide-border ide-canvas flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold ide-ink">
              10-Agent Execution Topology
            </span>
            <span className="text-[11px] ide-ink-muted">
              Click any agent node to inspect its decision &amp; reasoning
            </span>
          </div>
          <div className="flex items-center gap-4 text-[11px] ide-ink-secondary">
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-[#2F7A45]" /> Verified
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-[#181816]" /> Running
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-[#B87B19]" /> Needs Review
            </span>
          </div>
        </div>

        <div className="bg-dag-dot-grid overflow-x-auto p-3">
          <div className="relative min-w-[1046px] h-[296px]">
            {/* Bezier Edges Layer */}
            <svg
              className="absolute inset-0 w-full h-full pointer-events-none"
              viewBox="0 0 1046 296"
              fill="none"
            >
              {DAG_EDGES.map(([fromId, toId]) => {
                const a = nodeMap[fromId];
                const b = nodeMap[toId];
                const x1 = COL_X[a.col] + NODE_W;
                const y1 = ROW_Y(a.row) + NODE_H / 2;
                const x2 = COL_X[b.col];
                const y2 = ROW_Y(b.row) + NODE_H / 2;
                const cx = (x1 + x2) / 2;

                const fromState = computeNodeState(a, stage, status, decisionByRole[a.role]);
                const toState = computeNodeState(b, stage, status, decisionByRole[b.role]);
                const isDone = fromState === 'done' && (toState === 'done' || toState === 'active');
                const isActive = toState === 'active';

                return (
                  <path
                    key={`${fromId}-${toId}`}
                    d={`M ${x1} ${y1} C ${cx} ${y1}, ${cx} ${y2}, ${x2} ${y2}`}
                    stroke={
                      isActive
                        ? '#2563eb'
                        : isDone
                          ? '#10b981'
                          : 'var(--ide-border-strong)'
                    }
                    strokeWidth={isActive || isDone ? 1.75 : 1.25}
                    strokeDasharray={isActive ? '5 4' : undefined}
                  />
                );
              })}
            </svg>

            {/* Interactive Agent Nodes */}
            {DAG_NODES.map((node) => {
              const dec = decisionByRole[node.role];
              const state = computeNodeState(node, stage, status, dec);
              const isSelected = node.id === selectedNode.id;
              const confPct =
                typeof dec?.confidence === 'number'
                  ? `${Math.round(dec.confidence * 100)}%`
                  : node.id === 'judge' && typeof judgeScore === 'number'
                    ? `${Math.round(judgeScore)}/100`
                    : node.id === 'spec' && specId
                      ? specId.replace(/^spec-/, '')
                      : null;

              const stateBorder =
                state === 'active'
                  ? 'border-[#181816] ring-2 ring-black/10'
                  : state === 'gate'
                    ? 'border-[#B87B19] ring-2 ring-[#B87B19]/15'
                    : state === 'done'
                      ? 'border-[#285936]/60'
                      : 'ide-border';

              return (
                <button
                  key={node.id}
                  type="button"
                  onClick={() => setSelectedNodeId(node.id)}
                  style={{
                    left: `${COL_X[node.col]}px`,
                    top: `${ROW_Y(node.row)}px`,
                    width: `${NODE_W}px`,
                    height: `${NODE_H}px`,
                  }}
                  className={`absolute text-left rounded-xl ide-card border px-3 py-2 shadow-sm transition hover:shadow-md flex flex-col justify-between ${stateBorder} ${
                    isSelected ? 'ring-2 ring-[var(--ide-ink)] border-[var(--ide-ink)]' : ''
                  }`}
                >
                  <div className="flex items-center justify-between gap-1.5">
                    <span className="text-[10px] font-semibold uppercase tracking-wider ide-ink-muted">
                      {node.stageLabel}
                    </span>
                    {state === 'done' && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0" />}
                    {state === 'active' && <Loader2 className="w-3.5 h-3.5 text-blue-600 animate-spin shrink-0" />}
                    {state === 'gate' && <PauseCircle className="w-3.5 h-3.5 text-amber-500 shrink-0" />}
                    {state === 'idle' && <Clock className="w-3 h-3 ide-ink-muted shrink-0" />}
                  </div>

                  <div className="text-xs font-semibold ide-ink truncate">{node.title}</div>

                  <div className="flex items-center justify-between gap-1 text-[10px]">
                    <span className="ide-ink-muted truncate">
                      {dec ? dec.choice || dec.decision || 'Evaluated' : 'Ready'}
                    </span>
                    {confPct && (
                      <span className="font-mono font-medium px-1.5 py-0.2 rounded ide-subtle ide-ink-secondary tabular-nums shrink-0">
                        {confPct}
                      </span>
                    )}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Selected Agent Plain-English Inspector Card */}
      <div className="ide-canvas border ide-border rounded-2xl p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 ide-ink" />
              <h4 className="text-sm font-semibold ide-ink">{selectedNode.title}</h4>
              <span className="text-[11px] px-2.5 py-0.5 rounded-full ide-subtle ide-ink-secondary font-medium">
                Stage: {selectedNode.stageLabel}
              </span>
            </div>
            <p className="text-xs ide-ink-secondary mt-1">{selectedNode.summary}</p>
          </div>

          {selectedDecision && typeof selectedDecision.confidence === 'number' && (
            <div className="flex items-center gap-2">
              {selectedDecision.needs_approval && (
                <span className="inline-flex items-center gap-1 text-xs px-2.5 py-0.5 rounded-full bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] font-medium">
                  <ShieldAlert className="w-3.5 h-3.5" />
                  Requires Human Review
                </span>
              )}
              <span className="kora-pill-healthy text-xs font-mono font-medium px-2.5 py-0.5 rounded-full tabular-nums">
                {Math.round(selectedDecision.confidence * 100)}% confidence
              </span>
            </div>
          )}
        </div>

        {selectedDecision ? (
          <div className="mt-3 pt-3 border-t ide-border grid grid-cols-1 md:grid-cols-12 gap-4 text-xs">
            <div className="md:col-span-5">
              <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-1">
                Selected Strategy
              </div>
              <div className="font-medium ide-ink">
                {selectedDecision.choice || selectedDecision.decision}
              </div>
              {selectedDecision.alternatives && selectedDecision.alternatives.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {selectedDecision.alternatives.map((alt) => (
                    <span
                      key={alt}
                      className="text-[11px] px-2 py-0.5 rounded ide-subtle ide-ink-secondary border ide-border"
                    >
                      Alt: {alt}
                    </span>
                  ))}
                </div>
              )}
            </div>
            <div className="md:col-span-7">
              <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-1">
                Agent Reasoning
              </div>
              <p className="ide-ink-secondary leading-relaxed">
                {selectedDecision.reasoning || 'Policy and architecture checks passed for this build.'}
              </p>
            </div>
          </div>
        ) : (
          <div className="mt-3 pt-3 border-t ide-border text-xs ide-ink-muted">
            Start or select a Factory Build to inspect live decisions recorded by {selectedNode.title}.
          </div>
        )}
      </div>
    </div>
  );
}
