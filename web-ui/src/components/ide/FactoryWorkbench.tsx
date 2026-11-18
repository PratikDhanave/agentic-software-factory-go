'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { API_BASE, apiClient, FactoryPhase, FactoryRunRecord, SpecSummary } from '@/lib/api';
import { useMachine, workbenchActionMachine } from '@/lib/stateMachine';
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
  Sparkles,
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
  { key: 'spec_synthesis', num: '1', label: 'Spec' },
  { key: 'agent_governance', num: '2', label: 'Agent Review' },
  { key: 'workspace_init', num: '3', label: 'Sandbox' },
  { key: 'code_synthesis', num: '4', label: 'Write Code' },
  { key: 'verification_loop', num: '5', label: 'Run Tests' },
  { key: 'judge_evaluation', num: '6', label: 'Quality Score' },
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
}

export default function FactoryWorkbench({
  activePhase = 'walk',
  externalBlueprintTrigger,
  onRunStateChange,
}: Props) {
  const [mode, setMode] = useState<'composer' | 'run'>('composer');
  const [subView, setSubView] = useState<SubView>('dag');

  // Composer state
  const [title, setTitle] = useState(STARTER_BLUEPRINTS[0].title);
  const [description, setDescription] = useState(STARTER_BLUEPRINTS[0].description);
  const [selectedSpecId, setSelectedSpecId] = useState(STARTER_BLUEPRINTS[0].spec_id);
  const [forceApproval, setForceApproval] = useState(false);

  // Data state
  const [specs, setSpecs] = useState<SpecSummary[]>([]);
  const [runs, setRuns] = useState<FactoryRunRecord[]>([]);
  const [activeRun, setActiveRun] = useState<FactoryRunRecord | null>(null);
  const [initialLoaded, setInitialLoaded] = useState(false);

  // Explicit action state machine (eliminates conflicting boolean flags)
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
          // ignore malformed event
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
        subscribeStream(created.id);
        await loadRunsAndSpecs();
        onRunStateChange?.();
      } catch (err) {
        actionMachine.send('failed');
        setError(err instanceof Error ? err.message : 'Could not start factory build');
      }
    },
    [title, description, selectedSpecId, forceApproval, actionMachine, subscribeStream, loadRunsAndSpecs, onRunStateChange]
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
      setNotice(`Package promoted to ${promoted.promoted_dir}`);
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

  return (
    <div className="h-full grid grid-cols-1 lg:grid-cols-12 min-h-[calc(100vh-3rem)]">
      {/* Left Explorer Column: New Build Button + Runs List + Quick Blueprints */}
      <aside className="lg:col-span-3 xl:col-span-3 border-b lg:border-b-0 lg:border-r ide-border ide-canvas flex flex-col">
        <div className="p-3 border-b ide-border flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={() => {
              setMode('composer');
              setError('');
              setNotice('');
            }}
            className={`flex-1 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs font-semibold transition shadow-sm ${
              mode === 'composer'
                ? 'bg-blue-600 text-white'
                : 'ide-surface border ide-border ide-ink hover:border-blue-500/50'
            }`}
          >
            <Plus className="w-3.5 h-3.5" />
            New Factory Build
          </button>

          <button
            type="button"
            onClick={loadRunsAndSpecs}
            className="p-2 rounded-lg ide-surface border ide-border ide-ink-muted hover:ide-ink transition"
            title="Refresh builds"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>

        {/* Recent Builds List */}
        <div className="flex-1 overflow-y-auto p-2.5 space-y-4">
          <div>
            <div className="px-2 pb-1.5 flex items-center justify-between">
              <span className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted">
                Builds ({runs.length})
              </span>
            </div>

            {runs.length === 0 ? (
              <div className="px-3 py-6 rounded-lg border border-dashed ide-border text-center">
                <p className="text-xs font-medium ide-ink-secondary">No builds yet</p>
                <p className="text-[11px] ide-ink-muted mt-1">
                  Pick any blueprint on the right to watch the factory ship verified Go code.
                </p>
              </div>
            ) : (
              <div className="space-y-1">
                {runs.map((r) => {
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
                        const hasCode = (r.payload?.files?.length || 0) > 0;
                        setSubView(hasCode ? 'code' : 'dag');
                      }}
                      className={`w-full text-left px-3 py-2.5 rounded-lg border transition ${
                        selected
                          ? 'ide-surface border-blue-600/60 shadow-sm ring-1 ring-blue-600/15'
                          : 'border-transparent hover:ide-surface hover:ide-border'
                      }`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex items-center gap-2 min-w-0">
                          {r.status === 'completed' && (
                            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                          )}
                          {r.status === 'awaiting_approval' && (
                            <PauseCircle className="w-3.5 h-3.5 text-amber-500 shrink-0" />
                          )}
                          {r.status === 'running' && (
                            <Loader2 className="w-3.5 h-3.5 text-blue-600 animate-spin shrink-0" />
                          )}
                          {r.status === 'failed' && (
                            <XCircle className="w-3.5 h-3.5 text-rose-600 shrink-0" />
                          )}
                          <span className="text-xs font-medium ide-ink truncate">{r.title}</span>
                        </div>

                        {typeof score === 'number' && (
                          <span className="text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 tabular-nums shrink-0">
                            {Math.round(score)}
                          </span>
                        )}
                      </div>

                      <div className="mt-1 pl-5 flex items-center justify-between text-[11px] ide-ink-muted">
                        <span className="font-mono truncate">{r.spec_id || r.id}</span>
                        <span className="capitalize">
                          {r.status === 'awaiting_approval' ? 'Needs Review' : r.status}
                        </span>
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          {/* 1-Click Starter Blueprints in Sidebar */}
          <div className="pt-2 border-t ide-border">
            <div className="px-2 pb-1.5 text-[11px] font-semibold uppercase tracking-wider ide-ink-muted">
              Quick Blueprints
            </div>
            <div className="space-y-1">
              {STARTER_BLUEPRINTS.map((bp) => (
                <button
                  key={bp.spec_id}
                  type="button"
                  onClick={() => {
                    setSelectedSpecId(bp.spec_id);
                    setTitle(bp.title);
                    setDescription(bp.description);
                    setMode('composer');
                  }}
                  className="w-full text-left px-2.5 py-2 rounded-md hover:ide-surface text-xs ide-ink-secondary hover:ide-ink transition flex items-center justify-between gap-2 group"
                >
                  <span className="truncate">{bp.title}</span>
                  <ArrowRight className="w-3 h-3 opacity-0 group-hover:opacity-100 text-blue-600 shrink-0 transition" />
                </button>
              ))}
            </div>
          </div>
        </div>
      </aside>

      {/* Main Workspace Canvas */}
      <section className="lg:col-span-9 xl:col-span-9 ide-surface flex flex-col min-w-0">
        {mode === 'composer' || !activeRun ? (
          /* ================= GUIDED USER-FRIENDLY BUILD COMPOSER ================= */
          <div className="max-w-4xl w-full mx-auto px-6 py-8 space-y-7">
            <div>
              <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-blue-600/10 text-blue-700 dark:text-blue-300 text-xs font-medium mb-3">
                <Sparkles className="w-3.5 h-3.5" />
                Autonomous Go Software Factory
              </div>
              <h1 className="text-2xl font-semibold tracking-tight ide-ink">
                What Go package or service would you like to ship?
              </h1>
              <p className="text-sm ide-ink-secondary mt-1.5">
                Pick a ready-made blueprint below or describe your own requirement. The factory coordinates 10 specialized agents, compiles and race-tests the code in an isolated workspace, and scores quality automatically.
              </p>
            </div>

            {/* 3-Step Visual Explainer */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {[
                {
                  num: '1',
                  title: 'Describe or Pick Spec',
                  desc: 'Start from plain English or a validated YAML specification.',
                },
                {
                  num: '2',
                  title: '10-Agent Plan & Code',
                  desc: 'Architect, Security & Developer synthesize Go source + tests.',
                },
                {
                  num: '3',
                  title: 'Verify & Promote',
                  desc: 'Runs go build, go vet & go test -race before promoting.',
                },
              ].map((s) => (
                <div key={s.num} className="p-3.5 rounded-lg ide-canvas border ide-border">
                  <div className="flex items-center gap-2">
                    <span className="w-5 h-5 rounded-full bg-blue-600 text-white text-[11px] font-semibold flex items-center justify-center">
                      {s.num}
                    </span>
                    <span className="text-xs font-semibold ide-ink">{s.title}</span>
                  </div>
                  <p className="text-xs ide-ink-muted mt-1.5 leading-relaxed">{s.desc}</p>
                </div>
              ))}
            </div>

            {/* Starter Blueprint Cards */}
            <div>
              <div className="flex items-center justify-between mb-2.5">
                <span className="text-xs font-semibold ide-ink">
                  Start from a Proven Blueprint
                </span>
                <span className="text-xs ide-ink-muted">
                  Click a card to load, or &ldquo;Build Now&rdquo; to run immediately
                </span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {STARTER_BLUEPRINTS.map((bp) => {
                  const isChosen = selectedSpecId === bp.spec_id;
                  return (
                    <div
                      key={bp.spec_id}
                      onClick={() => {
                        setSelectedSpecId(bp.spec_id);
                        setTitle(bp.title);
                        setDescription(bp.description);
                      }}
                      className={`p-4 rounded-lg border cursor-pointer transition flex flex-col justify-between gap-3 ${
                        isChosen
                          ? 'border-blue-600 bg-blue-600/[0.03] ring-1 ring-blue-600/20'
                          : 'ide-border ide-surface hover:border-slate-400 dark:hover:border-slate-600'
                      }`}
                    >
                      <div>
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-[11px] font-semibold px-2 py-0.5 rounded ide-subtle ide-ink-secondary">
                            {bp.tag}
                          </span>
                          <span className="text-[11px] font-mono ide-ink-muted">{bp.spec_id}</span>
                        </div>
                        <div className="text-sm font-semibold ide-ink mt-2">{bp.title}</div>
                        <p className="text-xs ide-ink-secondary mt-1">{bp.short}</p>
                      </div>

                      <div className="flex items-center justify-between pt-2 border-t ide-border">
                        <span className="text-[11px] ide-ink-muted">
                          {isChosen ? 'Selected in form below' : 'Click to customize'}
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
                          className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 dark:text-blue-400 hover:underline"
                        >
                          Build Now <ArrowRight className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Clean Custom Build Form */}
            <div className="p-5 rounded-xl ide-canvas border ide-border space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
                <div className="md:col-span-7">
                  <label className="block text-xs font-medium ide-ink mb-1.5">
                    Package or Feature Name
                  </label>
                  <input
                    value={title}
                    onChange={(e) => {
                      setTitle(e.target.value);
                      setSelectedSpecId('');
                    }}
                    placeholder="e.g., Concurrent Token-Bucket Rate Limiter"
                    className="w-full px-3.5 py-2 rounded-lg ide-surface border ide-border text-sm ide-ink focus:outline-none focus:ring-2 focus:ring-blue-600/20 focus:border-blue-600"
                  />
                </div>

                <div className="md:col-span-5">
                  <label className="block text-xs font-medium ide-ink mb-1.5">
                    Specification Source
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
                    className="w-full px-3 py-2 rounded-lg ide-surface border ide-border text-sm ide-ink focus:outline-none focus:ring-2 focus:ring-blue-600/20 focus:border-blue-600"
                  >
                    <option value="">Auto-synthesize spec from description</option>
                    {specs.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.title} ({s.id})
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium ide-ink mb-1.5">
                  Requirements, API Behavior &amp; Test Expectations
                </label>
                <textarea
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  onKeyDown={(e) => {
                    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                      e.preventDefault();
                      startBuild();
                    }
                  }}
                  placeholder="Describe the Go types, methods, concurrency guarantees, and edge cases you want tested..."
                  className="w-full px-3.5 py-2 rounded-lg ide-surface border ide-border text-sm ide-ink focus:outline-none focus:ring-2 focus:ring-blue-600/20 focus:border-blue-600"
                />
              </div>

              {error && (
                <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-xs text-rose-700 dark:text-rose-300">
                  {error}
                </div>
              )}

              <div className="flex flex-wrap items-center justify-between gap-4 pt-1">
                <label className="inline-flex items-center gap-2 text-xs ide-ink-secondary cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={forceApproval || activePhase === 'crawl'}
                    disabled={activePhase === 'crawl'}
                    onChange={(e) => setForceApproval(e.target.checked)}
                    className="rounded border-slate-300 text-blue-600 focus:ring-blue-600"
                  />
                  <span>
                    Pause for my approval after Agent Review{' '}
                    <span className="ide-ink-muted">
                      ({activePhase === 'crawl' ? 'Always on in Supervised mode' : 'Optional checkpoint'})
                    </span>
                  </span>
                </label>

                <button
                  type="button"
                  disabled={starting}
                  onClick={() => startBuild()}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-slate-900 hover:bg-slate-800 dark:bg-blue-600 dark:hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium shadow-sm transition"
                >
                  {starting ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Play className="w-4 h-4 fill-current" />
                  )}
                  {starting ? 'Starting Factory Build...' : 'Start Factory Build'}
                </button>
              </div>
            </div>
          </div>
        ) : (
          /* ================= ACTIVE BUILD SPLIT-PANE WORKBENCH ================= */
          <div className="flex-1 flex flex-col min-w-0">
            {/* Run Top Header + 6-Stage Conveyor */}
            <div className="px-6 py-4 border-b ide-border space-y-4">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="text-lg font-semibold ide-ink">{activeRun.title}</h2>
                    <span
                      className={`text-xs font-medium px-2.5 py-0.5 rounded-full border ${
                        activeRun.status === 'completed'
                          ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/25'
                          : activeRun.status === 'awaiting_approval'
                            ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/25'
                            : activeRun.status === 'failed'
                              ? 'bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/25'
                              : 'bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/25'
                      }`}
                    >
                      {activeRun.status === 'awaiting_approval'
                        ? 'Paused for Approval'
                        : activeRun.status === 'completed'
                          ? 'Verified & Complete'
                          : activeRun.status === 'running'
                            ? 'Building...'
                            : activeRun.status}
                    </span>
                  </div>
                  <div className="flex flex-wrap items-center gap-3 text-xs ide-ink-muted mt-1 font-mono">
                    <span>Run: {activeRun.id}</span>
                    {activeRun.spec_id && <span>• Spec: {activeRun.spec_id}</span>}
                    <span>• Mode: {activeRun.phase_mode}</span>
                  </div>
                </div>

                {/* Primary Run Actions */}
                <div className="flex items-center gap-2">
                  {activeRun.status === 'awaiting_approval' && (
                    <button
                      type="button"
                      disabled={approving}
                      onClick={handleApproveGate}
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-semibold shadow-sm transition"
                    >
                      {approving ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <ShieldCheck className="w-4 h-4" />
                      )}
                      Approve &amp; Continue Build
                    </button>
                  )}

                  {activeRun.status === 'completed' && !activeRun.promoted_dir && (
                    <button
                      type="button"
                      disabled={promoting}
                      onClick={handlePromote}
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold shadow-sm transition"
                    >
                      {promoting ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <FolderOutput className="w-4 h-4" />
                      )}
                      Promote to generated/
                    </button>
                  )}

                  {activeRun.promoted_dir && (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-xs font-mono text-emerald-700 dark:text-emerald-300">
                      <CheckCircle2 className="w-3.5 h-3.5" />
                      {activeRun.promoted_dir.split('/').slice(-2).join('/')}
                    </span>
                  )}
                </div>
              </div>

              {/* Clean 6-Step Progress Bar */}
              <div className="grid grid-cols-2 sm:grid-cols-6 gap-2">
                {PIPELINE_STEPS.map((st, idx) => {
                  const stepNum = idx + 1;
                  const done = activeRun.status === 'completed' || currentStepOrder > stepNum;
                  const current =
                    activeRun.status !== 'completed' && currentStepOrder === stepNum;
                  const pausedHere = current && activeRun.status === 'awaiting_approval';

                  return (
                    <div
                      key={st.key}
                      className={`px-3 py-2 rounded-lg border text-xs flex items-center justify-between gap-2 transition ${
                        pausedHere
                          ? 'bg-amber-500/10 border-amber-500/40 text-amber-800 dark:text-amber-200'
                          : current
                            ? 'bg-blue-600/10 border-blue-600/40 text-blue-800 dark:text-blue-200'
                            : done
                              ? 'bg-emerald-500/[0.06] border-emerald-500/30 ide-ink'
                              : 'ide-canvas ide-border ide-ink-muted'
                      }`}
                    >
                      <span className="font-medium truncate">
                        {st.num}. {st.label}
                      </span>
                      {done && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0" />}
                      {pausedHere && <PauseCircle className="w-3.5 h-3.5 text-amber-500 shrink-0" />}
                      {current && !pausedHere && (
                        <Loader2 className="w-3.5 h-3.5 text-blue-600 animate-spin shrink-0" />
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Contextual Plain-English Helper Banner */}
              {activeRun.status === 'awaiting_approval' && (
                <div className="p-3.5 rounded-lg bg-amber-500/10 border border-amber-500/30 flex flex-wrap items-center justify-between gap-3 text-xs">
                  <div className="text-amber-900 dark:text-amber-200">
                    <span className="font-semibold">Human checkpoint reached:</span> Review the 10-agent architecture &amp; security decisions below, then click{' '}
                    <strong>Approve &amp; Continue Build</strong> to synthesize and test the Go package.
                  </div>
                </div>
              )}

              {notice && (
                <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-xs text-emerald-800 dark:text-emerald-200">
                  {notice}
                </div>
              )}

              {(error || activeRun.error) && (
                <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-xs text-rose-700 dark:text-rose-300">
                  {error || activeRun.error}
                </div>
              )}
            </div>

            {/* Sub-View Workspace Tabs */}
            <div className="px-6 ide-subtle border-b ide-border flex items-center gap-1 overflow-x-auto">
              {[
                {
                  id: 'code' as SubView,
                  label: `Generated Code (${files.length})`,
                  icon: FileCode2,
                },
                {
                  id: 'dag' as SubView,
                  label: `Agent Graph (${decisions.length || 10})`,
                  icon: GitBranch,
                },
                {
                  id: 'terminal' as SubView,
                  label: 'Test & Build Output',
                  icon: Terminal,
                },
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
                    className={`flex items-center gap-2 px-4 py-2.5 text-xs font-medium border-b-2 whitespace-nowrap transition ${
                      active
                        ? 'border-blue-600 text-blue-700 dark:text-blue-400 bg-white/60 dark:bg-white/[0.03]'
                        : 'border-transparent ide-ink-secondary hover:ide-ink'
                    }`}
                  >
                    <Icon className="w-3.5 h-3.5" />
                    {t.label}
                  </button>
                );
              })}
            </div>

            {/* Sub-View Body */}
            <div className="flex-1 p-6 overflow-y-auto space-y-6">
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
                <div className="space-y-5">
                  {/* Verification Gate Summary Cards */}
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <GateCard
                      label="Go Compiler (go build)"
                      passed={verification?.build_passed}
                      detail={
                        verification?.build_passed
                          ? 'Compiled cleanly with zero errors'
                          : verification
                            ? 'Build diagnostics reported'
                            : 'Waiting for Stage 5...'
                      }
                    />
                    <GateCard
                      label="Unit & Race Tests (go test -race)"
                      passed={verification?.tests_passed}
                      detail={
                        verification?.tests_passed
                          ? 'All test cases & race detector passed'
                          : verification
                            ? 'Test failures reported'
                            : 'Waiting for Stage 5...'
                      }
                    />
                    <GateCard
                      label="Static Analysis (go vet)"
                      passed={verification?.lint_passed}
                      detail={
                        verification?.lint_passed
                          ? 'Zero static analysis warnings'
                          : verification
                            ? 'Vet warnings found'
                            : 'Waiting for Stage 5...'
                      }
                    />
                  </div>

                  {/* Terminal Output */}
                  <div className="rounded-lg border ide-border overflow-hidden">
                    <div className="px-4 py-2 ide-subtle border-b ide-border flex items-center justify-between text-xs">
                      <span className="font-semibold ide-ink">
                        Sandbox Test Runner Output
                      </span>
                      <span className="font-mono text-[11px] ide-ink-muted">
                        {activeRun.workspace_dir
                          ? activeRun.workspace_dir.split('/').slice(-3).join('/')
                          : 'isolated-workspace'}
                      </span>
                    </div>
                    <pre className="p-4 bg-slate-950 text-emerald-300 text-xs font-mono overflow-x-auto max-h-80 leading-relaxed">
                      {verification?.raw_test_log ||
                        'Waiting for sandbox compiler and go test -v -race execution...'}
                    </pre>
                  </div>

                  {/* Execution Timeline */}
                  <div className="rounded-lg border ide-border ide-surface overflow-hidden">
                    <div className="px-4 py-2.5 ide-subtle border-b ide-border text-xs font-semibold ide-ink">
                      Stage Execution Timeline
                    </div>
                    <div className="divide-y ide-border">
                      {timeline.map((ev, idx) => (
                        <div
                          key={idx}
                          className="px-4 py-2.5 flex items-center justify-between gap-4 text-xs"
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            <Clock className="w-3.5 h-3.5 ide-ink-muted shrink-0" />
                            <span className="font-mono text-[11px] px-1.5 py-0.5 rounded ide-subtle ide-ink-secondary">
                              {ev.stage}
                            </span>
                            <span className="ide-ink truncate">{ev.message}</span>
                          </div>
                          {typeof ev.duration_ms === 'number' && ev.duration_ms > 0 && (
                            <span className="font-mono text-[11px] ide-ink-muted tabular-nums shrink-0">
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
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
                  {/* Left: 5-Criterion Judge Scorecard */}
                  <div className="lg:col-span-6 p-5 rounded-lg border ide-border ide-surface space-y-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="text-sm font-semibold ide-ink">
                          Engineering Quality Rubric
                        </h3>
                        <p className="text-xs ide-ink-muted mt-0.5">
                          Automated 5-dimension evaluation (pass threshold: 75/100)
                        </p>
                      </div>
                      {evaluation && (
                        <div className="text-right">
                          <div className="text-2xl font-bold text-emerald-600 dark:text-emerald-400 tabular-nums">
                            {Math.round(evaluation.overall_score)}
                            <span className="text-xs font-normal ide-ink-muted">/100</span>
                          </div>
                        </div>
                      )}
                    </div>

                    {evaluation?.scores ? (
                      <div className="space-y-3 pt-2">
                        {Object.entries(evaluation.scores).map(([criterion, val]) => {
                          const num = Math.round(Number(val) || 0);
                          return (
                            <div key={criterion}>
                              <div className="flex items-center justify-between text-xs mb-1">
                                <span className="capitalize font-medium ide-ink-secondary">
                                  {criterion.replace(/_/g, ' ')}
                                </span>
                                <span className="font-mono font-semibold ide-ink tabular-nums">
                                  {num}%
                                </span>
                              </div>
                              <div className="h-2 rounded-full ide-subtle overflow-hidden">
                                <div
                                  className="h-full rounded-full bg-blue-600 transition-all"
                                  style={{ width: `${Math.min(100, num)}%` }}
                                />
                              </div>
                            </div>
                          );
                        })}
                        {evaluation.summary && (
                          <p className="text-xs ide-ink-secondary pt-3 border-t ide-border leading-relaxed">
                            {evaluation.summary}
                          </p>
                        )}
                      </div>
                    ) : (
                      <p className="text-xs ide-ink-muted py-8 text-center">
                        Quality scores appear once Stage 6 completes.
                      </p>
                    )}
                  </div>

                  {/* Right: Specification Summary */}
                  <div className="lg:col-span-6 p-5 rounded-lg border ide-border ide-surface space-y-4">
                    <div>
                      <h3 className="text-sm font-semibold ide-ink">
                        Active Specification ({activeRun.spec_id || 'Synthesized'})
                      </h3>
                      <p className="text-xs ide-ink-secondary mt-1">
                        {specObj?.Intent?.Goal || specObj?.intent?.goal || activeRun.description}
                      </p>
                    </div>

                    {(specObj?.Constraints || specObj?.constraints) && (
                      <div>
                        <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2">
                          Enforced Constraints
                        </div>
                        <div className="space-y-1.5">
                          {(specObj.Constraints || specObj.constraints || []).map((c: any) => (
                            <div
                              key={c.ID || c.id}
                              className="px-3 py-2 rounded-md ide-canvas border ide-border text-xs flex items-center gap-2"
                            >
                              <span className="font-mono text-[10px] uppercase px-1.5 py-0.5 rounded bg-blue-600/10 text-blue-700 dark:text-blue-300 font-semibold">
                                {c.Type || c.type}
                              </span>
                              <span className="ide-ink">{c.Description || c.description}</span>
                            </div>
                          ))}
                        </div>
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
  );
}

function GateCard({
  label,
  passed,
  detail,
}: {
  label: string;
  passed?: boolean;
  detail: string;
}) {
  return (
    <div className="p-4 rounded-lg border ide-border ide-surface">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-semibold ide-ink">{label}</span>
        {passed === true && <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />}
        {passed === false && <XCircle className="w-4 h-4 text-rose-600 shrink-0" />}
        {passed === undefined && <Clock className="w-4 h-4 ide-ink-muted shrink-0" />}
      </div>
      <p className="text-xs ide-ink-secondary mt-1.5">{detail}</p>
    </div>
  );
}
