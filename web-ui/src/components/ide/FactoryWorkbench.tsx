'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { API_BASE, apiClient, FactoryPhase, FactoryRunRecord, SpecSummary } from '@/lib/api';
import { statusBadge, useMachine, workbenchActionMachine } from '@/lib/stateMachine';
import CodeSyntaxView, { WorkspaceFileItem } from './CodeSyntaxView';
import AgentDagCanvas from './AgentDagCanvas';
import {
  ArrowRight,
  Award,
  CheckCircle2,
  Clock,
  FileCode2,
  FolderOutput,
  GitBranch,
  Loader2,
  PauseCircle,
  Play,
  Plus,
  RefreshCw,
  ShieldCheck,
  Terminal,
  XCircle,
} from 'lucide-react';

export const STARTER_BLUEPRINTS = [
  {
    spec_id: 'spec-token-bucket',
    tag: 'Concurrency',
    title: 'Concurrent Token-Bucket Rate Limiter',
    short: 'Burst-capable rate limiter tested with 100 concurrent goroutines.',
    description:
      'Implement a thread-safe Go token-bucket rate limiter supporting burst capacity, continuous token refill, Allow/AllowN methods, and 100-goroutine race tests.',
  },
  {
    spec_id: 'spec-circuit-breaker',
    tag: 'Resilience',
    title: 'State-Machine Circuit Breaker',
    short: 'Closed, Open, and Half-Open fault-tolerance state machine.',
    description:
      'Implement a resilient Go CircuitBreaker with closed, open, and half-open recovery transitions, configurable failure thresholds, and table-driven unit tests.',
  },
  {
    spec_id: 'spec-ttl-lru-cache',
    tag: 'Generics',
    title: 'Generic Thread-Safe TTL LRU Cache',
    short: 'Type-safe TTLCache[K, V] with O(1) eviction and expiration.',
    description:
      'Implement a generic TTLCache[K comparable, V any] with O(1) doubly-linked LRU eviction, lazy TTL expiration, and concurrent stress tests.',
  },
  {
    spec_id: 'spec-webhook-verifier',
    tag: 'Security',
    title: 'HMAC-SHA256 Webhook Verifier',
    short: 'Constant-time signature verification + replay protection.',
    description:
      'Implement constant-time HMAC-SHA256 webhook signature verification with timestamp tolerance and sliding-window delivery ID replay protection.',
  },
];

const PIPELINE_STEPS = [
  { key: 'spec_synthesis', num: '01', label: 'Spec' },
  { key: 'agent_governance', num: '02', label: 'Agent Review' },
  { key: 'workspace_init', num: '03', label: 'Sandbox' },
  { key: 'code_synthesis', num: '04', label: 'Write Code' },
  { key: 'verification_loop', num: '05', label: 'Race Tests' },
  { key: 'judge_evaluation', num: '06', label: 'Quality Gate' },
];

const STAGE_INDEX: Record<string, number> = {
  queued: 0,
  spec_synthesis: 1,
  agent_governance: 2,
  awaiting_approval: 2,
  workspace_init: 3,
  code_synthesis: 4,
  verification_loop: 5,
  judge_evaluation: 6,
  completed: 7,
  failed: -1,
};

type SubView = 'code' | 'dag' | 'terminal' | 'quality';

interface Props {
  activePhase?: FactoryPhase;
  externalBlueprintTrigger?: { spec_id: string; ts: number } | null;
  onRunStateChange?: () => void;
  onNotify?: (message: string) => void;
}

export default function FactoryWorkbench({
  activePhase = 'walk',
  externalBlueprintTrigger,
  onRunStateChange,
  onNotify,
}: Props) {
  const [mode, setMode] = useState<'composer' | 'run'>('composer');
  const [subView, setSubView] = useState<SubView>('dag');
  const [runFilter, setRunFilter] = useState<'all' | 'completed' | 'paused'>('all');

  const [title, setTitle] = useState(STARTER_BLUEPRINTS[0].title);
  const [description, setDescription] = useState(STARTER_BLUEPRINTS[0].description);
  const [selectedSpecId, setSelectedSpecId] = useState(STARTER_BLUEPRINTS[0].spec_id);
  const [forceApproval, setForceApproval] = useState(false);

  const [specs, setSpecs] = useState<SpecSummary[]>([]);
  const [runs, setRuns] = useState<FactoryRunRecord[]>([]);
  const [activeRun, setActiveRun] = useState<FactoryRunRecord | null>(null);
  const [initialLoaded, setInitialLoaded] = useState(false);

  const actionMachine = useMachine(workbenchActionMachine);
  const starting = actionMachine.current === 'starting_build';
  const approving = actionMachine.current === 'approving_gate';
  const promoting = actionMachine.current === 'promoting_package';
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const loadRunsAndSpecs = useCallback(async () => {
    try {
      const [runsResp, specsResp] = await Promise.all([
        apiClient.listFactoryRuns(),
        apiClient.getSpecs().catch(() => ({ specs: [] })),
      ]);
      const list = runsResp.runs || [];
      setRuns(list);
      setSpecs(specsResp.specs || []);

      setActiveRun((prev) => {
        if (prev) {
          const updated = list.find((r) => r.id === prev.id);
          return updated || prev;
        }
        return list[0] || null;
      });

      if (!initialLoaded) {
        setInitialLoaded(true);
        if (list.length > 0) {
          setMode('run');
          const first = list[0];
          const hasFiles = (first.payload?.files?.length || 0) > 0;
          setSubView(hasFiles ? 'code' : 'dag');
        }
      }
    } catch {
      // ignore background poll error
    }
  }, [initialLoaded]);

  useEffect(() => {
    loadRunsAndSpecs();
    const timer = setInterval(loadRunsAndSpecs, 2500);
    return () => clearInterval(timer);
  }, [loadRunsAndSpecs]);

  const subscribeStream = useCallback(
    (runId: string) => {
      const es = new EventSource(`${API_BASE}/api/v1/factory/runs/${encodeURIComponent(runId)}/stream`);
      es.onmessage = (ev) => {
        try {
          const updated: FactoryRunRecord = JSON.parse(ev.data);
          setActiveRun(updated);
          if ((updated.payload?.files?.length || 0) > 0 && updated.status === 'completed') {
            setSubView('code');
          }
          if (
            updated.status === 'completed' ||
            updated.status === 'failed' ||
            updated.status === 'awaiting_approval'
          ) {
            es.close();
            loadRunsAndSpecs();
            onRunStateChange?.();
          }
        } catch {
          // ignore
        }
      };
      es.onerror = () => es.close();
    },
    [loadRunsAndSpecs, onRunStateChange]
  );

  const startBuild = useCallback(
    async (override?: { title: string; description: string; spec_id?: string }) => {
      const finalTitle = (override?.title ?? title).trim();
      const finalDesc = (override?.description ?? description).trim() || finalTitle;
      const finalSpec = override ? override.spec_id : selectedSpecId || undefined;

      if (!finalTitle && !finalSpec) {
        setError('Please enter a package title or choose a starter blueprint.');
        return;
      }

      if (!actionMachine.send('starting_build')) return;
      setError('');
      setNotice('');
      try {
        const created = await apiClient.startFactoryRun({
          title: finalTitle,
          description: finalDesc,
          spec_id: finalSpec,
          force_approval: forceApproval || undefined,
        });
        setActiveRun(created);
        setMode('run');
        setSubView('dag');
        actionMachine.send('succeeded');
        onNotify?.(`Started build: ${created.title}`);
        subscribeStream(created.id);
        await loadRunsAndSpecs();
        onRunStateChange?.();
      } catch (err) {
        actionMachine.send('failed');
        setError(err instanceof Error ? err.message : 'Could not start factory build');
      }
    },
    [title, description, selectedSpecId, forceApproval, actionMachine, onNotify, subscribeStream, loadRunsAndSpecs, onRunStateChange]
  );

  useEffect(() => {
    if (!externalBlueprintTrigger) return;
    const bp = STARTER_BLUEPRINTS.find((b) => b.spec_id === externalBlueprintTrigger.spec_id);
    if (bp) {
      setTitle(bp.title);
      setDescription(bp.description);
      setSelectedSpecId(bp.spec_id);
      startBuild({ title: bp.title, description: bp.description, spec_id: bp.spec_id });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [externalBlueprintTrigger?.ts]);

  const handleApproveGate = async () => {
    if (!activeRun || !actionMachine.send('approving_gate')) return;
    setError('');
    setNotice('');
    try {
      const resumed = await apiClient.approveFactoryRun(activeRun.id);
      setActiveRun(resumed);
      actionMachine.send('succeeded');
      setNotice('Approval recorded — resuming sandbox synthesis and verification...');
      onNotify?.('Approved checkpoint — resuming sandbox build');
      subscribeStream(resumed.id);
      onRunStateChange?.();
    } catch (err) {
      actionMachine.send('failed');
      setError(err instanceof Error ? err.message : 'Approval failed');
    }
  };

  const handlePromote = async () => {
    if (!activeRun || !actionMachine.send('promoting_package')) return;
    setError('');
    setNotice('');
    try {
      const promoted = await apiClient.promoteFactoryRun(activeRun.id);
      setActiveRun(promoted);
      actionMachine.send('succeeded');
      setNotice(`Promoted verified Go package to ${promoted.promoted_dir}`);
      onNotify?.(`Promoted package to ${promoted.promoted_dir}`);
      await loadRunsAndSpecs();
    } catch (err) {
      actionMachine.send('failed');
      setError(err instanceof Error ? err.message : 'Failed to promote package');
    }
  };

  const files: WorkspaceFileItem[] = activeRun?.payload?.files || [];
  const decisions: any[] = activeRun?.payload?.decisions || [];
  const timeline: Array<{
    stage: string;
    status: string;
    message: string;
    duration_ms?: number;
    timestamp: string;
  }> = activeRun?.payload?.timeline || [];
  const verification = activeRun?.payload?.verification;
  const evaluation = activeRun?.payload?.evaluation;
  const specObj = activeRun?.payload?.spec;
  const currentStepOrder = activeRun ? STAGE_INDEX[activeRun.stage] ?? 0 : 0;

  const filteredRuns = runs.filter((r) => {
    if (runFilter === 'completed') return r.status === 'completed';
    if (runFilter === 'paused') return r.status === 'awaiting_approval';
    return true;
  });

  // Extract individual `--- PASS: TestXxx` lines from go test -v output
  const passedTestNames: string[] = React.useMemo(() => {
    const raw = String(verification?.raw_test_log || verification?.test_output || '');
    const matches = raw.match(/---\s+PASS:\s+([A-Za-z0-9_]+)/g) || [];
    return matches.map((m) => m.replace(/---\s+PASS:\s+/, '').trim());
  }, [verification]);

  return (
    <div className="px-6 lg:px-8 py-7 max-w-[1280px] w-full space-y-6">
      {/* Top Studio Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[28px] leading-tight font-bold tracking-tight ide-ink">
            Factory Runs
          </h1>
          <p className="text-sm ide-ink-muted mt-1">
            Autonomous 6-stage Go assembly line — Spec synthesis, 10-agent review, isolated `go test -race`, and Judge scoring.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => {
              setMode('composer');
              setError('');
              setNotice('');
            }}
            className={`inline-flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium transition ${
              mode === 'composer'
                ? 'bg-[#181816] text-white dark:bg-white dark:text-black'
                : 'ide-surface border ide-border-strong ide-ink hover:ide-subtle'
            }`}
          >
            <Plus className="w-3.5 h-3.5" />
            New Run
          </button>

          <button
            type="button"
            onClick={loadRunsAndSpecs}
            className="p-2 rounded-full ide-surface border ide-border ide-ink-secondary hover:ide-ink transition"
            title="Refresh runs"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Main Split Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        {/* Left Runs Ledger */}
        <aside className="lg:col-span-3 ide-surface border ide-border rounded-2xl overflow-hidden flex flex-col max-h-[760px]">
          <div className="px-4 py-3 border-b ide-border space-y-2.5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold ide-ink">Runs ({filteredRuns.length})</span>
              <span className="text-[10px] font-mono ide-ink-muted">sqlite-wal</span>
            </div>
            <div className="grid grid-cols-3 gap-1 p-0.5 rounded-lg ide-canvas border ide-border text-[11px]">
              {(
                [
                  ['all', 'All'],
                  ['completed', 'Passed'],
                  ['paused', 'Paused'],
                ] as const
              ).map(([k, lbl]) => (
                <button
                  key={k}
                  type="button"
                  onClick={() => setRunFilter(k)}
                  className={`py-1 rounded-md font-medium transition ${
                    runFilter === k ? 'ide-card ide-ink shadow-sm' : 'ide-ink-muted hover:ide-ink'
                  }`}
                >
                  {lbl}
                </button>
              ))}
            </div>
          </div>

          {filteredRuns.length === 0 ? (
            <div className="p-6 text-center text-xs ide-ink-muted">
              No matching runs. Choose any blueprint on the right to start a verified build.
            </div>
          ) : (
            <div className="divide-y ide-border overflow-y-auto flex-1">
              {filteredRuns.map((r) => {
                const selected = mode === 'run' && activeRun?.id === r.id;
                const score = r.payload?.evaluation?.overall_score;
                return (
                  <button
                    key={r.id}
                    type="button"
                    onClick={() => {
                      setActiveRun(r);
                      setMode('run');
                      setError('');
                      setNotice('');
                      setSubView((r.payload?.files?.length || 0) > 0 ? 'code' : 'dag');
                    }}
                    className={`w-full text-left px-4 py-3.5 transition ${
                      selected ? 'ide-subtle' : 'hover:bg-black/[0.018] dark:hover:bg-white/[0.02]'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        {r.status === 'completed' && (
                          <CheckCircle2 className="w-3.5 h-3.5 text-[#285936] shrink-0" />
                        )}
                        {r.status === 'awaiting_approval' && (
                          <PauseCircle className="w-3.5 h-3.5 text-[#9C6B19] shrink-0" />
                        )}
                        {r.status === 'running' && (
                          <Loader2 className="w-3.5 h-3.5 animate-spin ide-ink shrink-0" />
                        )}
                        {r.status === 'failed' && (
                          <XCircle className="w-3.5 h-3.5 text-[#7D2626] shrink-0" />
                        )}
                        <span className="text-xs font-semibold ide-ink truncate">{r.title}</span>
                      </div>
                      {typeof score === 'number' && (
                        <span className="kora-pill-healthy text-[10px] font-mono px-2 py-0.2 rounded-full shrink-0">
                          {Math.round(score)}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 pl-5 flex items-center justify-between text-[11px] ide-ink-muted">
                      <span className="font-mono truncate">{r.id}</span>
                      <span className="capitalize">
                        {r.status === 'awaiting_approval' ? 'Paused' : r.status}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </aside>

        {/* Right Composer OR Active Run Inspector */}
        <section className="lg:col-span-9 space-y-5 min-w-0">
          {mode === 'composer' || !activeRun ? (
            <div className="ide-surface border ide-border rounded-2xl p-6 space-y-6">
              <div>
                <h2 className="text-lg font-semibold ide-ink">
                  Ship a Verified Go Package
                </h2>
                <p className="text-xs ide-ink-muted mt-0.5">
                  Select a starter blueprint or describe your custom Go requirement below.
                </p>
              </div>

              {/* 4 Editorial Blueprint Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                {STARTER_BLUEPRINTS.map((bp) => {
                  const chosen = selectedSpecId === bp.spec_id;
                  return (
                    <div
                      key={bp.spec_id}
                      onClick={() => {
                        setSelectedSpecId(bp.spec_id);
                        setTitle(bp.title);
                        setDescription(bp.description);
                      }}
                      className={`p-4 rounded-2xl border cursor-pointer transition flex flex-col justify-between gap-3 ${
                        chosen
                          ? 'ide-card border-[var(--ide-ink)]'
                          : 'ide-canvas ide-border hover:ide-border-strong'
                      }`}
                    >
                      <div>
                        <div className="flex items-center justify-between text-[11px]">
                          <span className="px-2.5 py-0.5 rounded-full ide-subtle ide-ink-secondary font-medium">
                            {bp.tag}
                          </span>
                          <span className="font-mono ide-ink-muted">{bp.spec_id}</span>
                        </div>
                        <div className="text-sm font-semibold ide-ink mt-2.5">{bp.title}</div>
                        <p className="text-xs ide-ink-muted mt-1 leading-relaxed">{bp.short}</p>
                      </div>

                      <div className="flex items-center justify-between pt-2.5 border-t ide-border">
                        <span className="text-[11px] ide-ink-muted">
                          {chosen ? 'Loaded below' : 'Click to customize'}
                        </span>
                        <button
                          type="button"
                          disabled={starting}
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedSpecId(bp.spec_id);
                            setTitle(bp.title);
                            setDescription(bp.description);
                            startBuild({
                              title: bp.title,
                              description: bp.description,
                              spec_id: bp.spec_id,
                            });
                          }}
                          className="inline-flex items-center gap-1 text-xs font-semibold ide-ink hover:underline"
                        >
                          Build now <ArrowRight className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Custom Requirement Inputs */}
              <div className="p-5 rounded-2xl ide-canvas border ide-border space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
                  <div className="md:col-span-7">
                    <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                      Package / Feature Title
                    </label>
                    <input
                      value={title}
                      onChange={(e) => {
                        setTitle(e.target.value);
                        setSelectedSpecId('');
                      }}
                      placeholder="e.g., Concurrent Token-Bucket Rate Limiter"
                      className="w-full px-3.5 py-2 rounded-xl ide-surface border ide-border text-sm ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
                    />
                  </div>

                  <div className="md:col-span-5">
                    <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                      Specification Blueprint
                    </label>
                    <select
                      value={selectedSpecId}
                      onChange={(e) => {
                        const id = e.target.value;
                        setSelectedSpecId(id);
                        const found = specs.find((s) => s.id === id);
                        if (found) {
                          setTitle(found.title);
                          if (found.description) setDescription(found.description);
                        }
                      }}
                      className="w-full px-3.5 py-2 rounded-xl ide-surface border ide-border text-sm ide-ink focus:outline-none"
                    >
                      <option value="">Synthesize spec from description</option>
                      {specs.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.title} ({s.id})
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Requirements &amp; Concurrency Invariants
                  </label>
                  <textarea
                    rows={3}
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="Describe expected Go types, edge cases, and concurrency guarantees..."
                    className="w-full px-3.5 py-2 rounded-xl ide-surface border ide-border text-sm ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
                  />
                </div>

                {error && (
                  <div className="p-3 rounded-xl text-xs bg-[#F5DCDC] text-[#7D2626]">{error}</div>
                )}

                <div className="flex flex-wrap items-center justify-between gap-4 pt-1">
                  <label className="inline-flex items-center gap-2 text-xs ide-ink-secondary cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={forceApproval || activePhase === 'crawl'}
                      disabled={activePhase === 'crawl'}
                      onChange={(e) => setForceApproval(e.target.checked)}
                      className="rounded border-stone-400"
                    />
                    <span>Pause for human approval after 10-agent review</span>
                  </label>

                  <button
                    type="button"
                    disabled={starting}
                    onClick={() => startBuild()}
                    className="inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-[#181816] hover:opacity-90 dark:bg-white dark:text-black disabled:opacity-50 text-white text-xs font-semibold transition"
                  >
                    {starting ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Play className="w-3.5 h-3.5 fill-current" />
                    )}
                    {starting ? 'Starting Run...' : 'Start Factory Run'}
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
              {/* Run Header + 6-Stage Stepper */}
              <div className="p-6 border-b ide-border space-y-4">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <div className="flex flex-wrap items-center gap-2.5">
                      <h2 className="text-lg font-bold ide-ink">{activeRun.title}</h2>
                      <span
                        className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          statusBadge[activeRun.status] || statusBadge.idle
                        }`}
                      >
                        {activeRun.status === 'awaiting_approval'
                          ? 'Needs Approval'
                          : activeRun.status}
                      </span>
                    </div>
                    <div className="text-xs font-mono ide-ink-muted mt-1">
                      {activeRun.id} • {activeRun.spec_id || 'synthesized'} • mode:{activeRun.phase_mode}
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    {activeRun.status === 'awaiting_approval' && (
                      <button
                        type="button"
                        disabled={approving}
                        onClick={handleApproveGate}
                        className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#181816] text-white dark:bg-white dark:text-black text-xs font-semibold transition"
                      >
                        {approving ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <ShieldCheck className="w-3.5 h-3.5" />
                        )}
                        Approve &amp; Continue
                      </button>
                    )}

                    {activeRun.status === 'completed' && !activeRun.promoted_dir && (
                      <button
                        type="button"
                        disabled={promoting}
                        onClick={handlePromote}
                        className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#181816] text-white dark:bg-white dark:text-black text-xs font-semibold transition"
                      >
                        {promoting ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <FolderOutput className="w-3.5 h-3.5" />
                        )}
                        Promote to generated/
                      </button>
                    )}

                    {activeRun.promoted_dir && (
                      <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-mono">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        {activeRun.promoted_dir}
                      </span>
                    )}
                  </div>
                </div>

                {/* 6-Stage Stepper */}
                <div className="grid grid-cols-2 sm:grid-cols-6 gap-2">
                  {PIPELINE_STEPS.map((st, idx) => {
                    const stepNum = idx + 1;
                    const done = activeRun.status === 'completed' || currentStepOrder > stepNum;
                    const current = activeRun.status !== 'completed' && currentStepOrder === stepNum;
                    const paused = current && activeRun.status === 'awaiting_approval';

                    return (
                      <div
                        key={st.key}
                        className={`px-3 py-2 rounded-xl border text-xs flex items-center justify-between gap-1.5 ${
                          paused
                            ? statusBadge.awaiting_approval
                            : done
                              ? 'kora-pill-healthy'
                              : current
                                ? statusBadge.running
                                : 'ide-canvas ide-border ide-ink-muted'
                        }`}
                      >
                        <span className="font-medium truncate">
                          {st.num} {st.label}
                        </span>
                        {done && <CheckCircle2 className="w-3.5 h-3.5 shrink-0" />}
                        {paused && <PauseCircle className="w-3.5 h-3.5 shrink-0" />}
                        {current && !paused && <Loader2 className="w-3.5 h-3.5 animate-spin shrink-0" />}
                      </div>
                    );
                  })}
                </div>

                {notice && (
                  <div className="px-4 py-2.5 rounded-xl kora-pill-healthy text-xs">{notice}</div>
                )}
                {(error || activeRun.error) && (
                  <div className="px-4 py-2.5 rounded-xl bg-[#F5DCDC] text-[#7D2626] text-xs">
                    {error || activeRun.error}
                  </div>
                )}
              </div>

              {/* Pill Sub-View Navigation */}
              <div className="px-6 py-3 ide-canvas border-b ide-border flex flex-wrap items-center gap-2">
                {[
                  { id: 'code' as SubView, label: `Generated Code (${files.length})`, icon: FileCode2 },
                  { id: 'dag' as SubView, label: `Agent Graph (${decisions.length || 10})`, icon: GitBranch },
                  { id: 'terminal' as SubView, label: 'Test & Build Output', icon: Terminal },
                  {
                    id: 'quality' as SubView,
                    label: evaluation
                      ? `Quality & Spec (${Math.round(evaluation.overall_score)}/100)`
                      : 'Quality & Spec',
                    icon: Award,
                  },
                ].map((t) => {
                  const Icon = t.icon;
                  const active = subView === t.id;
                  return (
                    <button
                      key={t.id}
                      type="button"
                      onClick={() => setSubView(t.id)}
                      className={`inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-medium transition ${
                        active
                          ? 'kora-nav-active'
                          : 'ide-surface border ide-border ide-ink-secondary hover:ide-ink'
                      }`}
                    >
                      <Icon className="w-3.5 h-3.5" />
                      {t.label}
                    </button>
                  );
                })}
              </div>

              {/* Sub-View Canvas */}
              <div className="p-6 space-y-5">
                {subView === 'code' && (
                  <CodeSyntaxView
                    files={files}
                    workspaceDir={activeRun.workspace_dir}
                    promotedDir={activeRun.promoted_dir}
                  />
                )}

                {subView === 'dag' && (
                  <AgentDagCanvas
                    stage={activeRun.stage}
                    status={activeRun.status}
                    decisions={decisions}
                    judgeScore={evaluation?.overall_score}
                    specId={activeRun.spec_id}
                  />
                )}

                {subView === 'terminal' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                      <GateCard label="go build" passed={verification?.build_passed} />
                      <GateCard label="go test -race" passed={verification?.tests_passed} />
                      <GateCard label="go vet" passed={verification?.lint_passed} />
                    </div>

                    {passedTestNames.length > 0 && (
                      <div className="p-4 rounded-2xl ide-canvas border ide-border space-y-2">
                        <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
                          Verified Test Cases ({passedTestNames.length})
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {passedTestNames.map((fn) => (
                            <span
                              key={fn}
                              className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono"
                            >
                              <CheckCircle2 className="w-3 h-3" />
                              {fn}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    <pre className="p-4 rounded-2xl bg-[#181816] text-[#DCE9DF] text-xs font-mono overflow-x-auto max-h-80 leading-relaxed">
                      {verification?.raw_test_log ||
                        verification?.test_output ||
                        'Waiting for Stage 5 isolated workspace verification...'}
                    </pre>

                    <div className="rounded-2xl border ide-border overflow-hidden">
                      <div className="px-4 py-2.5 ide-canvas border-b ide-border text-xs font-semibold ide-ink">
                        Stage Timeline
                      </div>
                      <div className="divide-y ide-border">
                        {timeline.map((ev, i) => (
                          <div key={i} className="px-4 py-2.5 flex items-center justify-between text-xs">
                            <div className="flex items-center gap-2 min-w-0">
                              <Clock className="w-3.5 h-3.5 ide-ink-muted shrink-0" />
                              <span className="font-mono text-[11px] ide-ink-muted">{ev.stage}</span>
                              <span className="ide-ink truncate">{ev.message}</span>
                            </div>
                            {typeof ev.duration_ms === 'number' && ev.duration_ms > 0 && (
                              <span className="font-mono text-[11px] ide-ink-muted shrink-0">
                                {ev.duration_ms}ms
                              </span>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {subView === 'quality' && (
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
                    <div className="p-5 rounded-2xl ide-canvas border ide-border space-y-4">
                      <div className="flex items-center justify-between">
                        <div>
                          <h3 className="text-sm font-semibold ide-ink">Judge Quality Rubric</h3>
                          <p className="text-xs ide-ink-muted">Pass threshold: 75 / 100</p>
                        </div>
                        {evaluation && (
                          <span className="text-2xl font-bold ide-ink tabular-nums">
                            {Math.round(evaluation.overall_score)}/100
                          </span>
                        )}
                      </div>
                      {evaluation?.scores ? (
                        <div className="space-y-3">
                          {Object.entries(evaluation.scores).map(([k, val]) => {
                            const num = Math.round(Number(val) || 0);
                            return (
                              <div key={k}>
                                <div className="flex justify-between text-xs mb-1">
                                  <span className="capitalize ide-ink-secondary">
                                    {k.replace(/_/g, ' ')}
                                  </span>
                                  <span className="font-mono font-semibold ide-ink">{num}%</span>
                                </div>
                                <div className="h-2 rounded-full ide-subtle overflow-hidden">
                                  <div
                                    className="h-full rounded-full bg-[#285936]"
                                    style={{ width: `${Math.min(100, num)}%` }}
                                  />
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <p className="text-xs ide-ink-muted py-6">Awaiting Stage 6 evaluation...</p>
                      )}
                    </div>

                    <div className="p-5 rounded-2xl ide-canvas border ide-border space-y-3">
                      <h3 className="text-sm font-semibold ide-ink">
                        Specification ({activeRun.spec_id || 'Synthesized'})
                      </h3>
                      <p className="text-xs ide-ink-secondary leading-relaxed">
                        {specObj?.Intent?.Goal || specObj?.intent?.goal || activeRun.description}
                      </p>
                      {(specObj?.Constraints || specObj?.constraints) && (
                        <div className="space-y-1.5 pt-2">
                          {(specObj.Constraints || specObj.constraints || []).map((c: any) => (
                            <div
                              key={c.ID || c.id}
                              className="px-3 py-2 rounded-xl ide-surface border ide-border text-xs flex items-center gap-2"
                            >
                              <span className="font-mono text-[10px] uppercase px-2 py-0.5 rounded-full ide-subtle ide-ink-secondary">
                                {c.Type || c.type}
                              </span>
                              <span className="ide-ink">{c.Description || c.description}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function GateCard({ label, passed }: { label: string; passed?: boolean }) {
  return (
    <div className="p-4 rounded-2xl ide-canvas border ide-border flex items-center justify-between">
      <span className="text-xs font-mono font-semibold ide-ink">{label}</span>
      {passed === true && (
        <span className="kora-pill-healthy px-2.5 py-0.5 rounded-full text-xs font-medium">
          Passed
        </span>
      )}
      {passed === false && (
        <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusBadge.failed}`}>
          Failed
        </span>
      )}
      {passed === undefined && (
        <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusBadge.idle}`}>
          Pending
        </span>
      )}
    </div>
  );
}
