'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  AlertTriangle,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  FolderOutput,
  Loader2,
  Play,
  ShieldAlert,
  Sparkles,
  Terminal,
  XCircle,
} from 'lucide-react';
import { API_BASE, RunRecord, SpecRecord, api } from '@/lib/api';
import { RunStudioUIMachine, useMachine } from '@/lib/stateMachine';
import { CodeViewer } from './CodeViewer';

interface RunsStudioProps {
  runs: RunRecord[];
  specs: SpecRecord[];
  autonomy: 'crawl' | 'walk' | 'run';
  selectedRunId: string | null;
  onSelectRun: (id: string) => void;
  onRefresh: () => void;
  notify: (msg: string) => void;
}

const PIPELINE_STAGES = [
  { id: 'queued', label: 'Queued' },
  { id: 'planning', label: 'Planning' },
  { id: 'awaiting_approval', label: 'Gate' },
  { id: 'synthesizing', label: 'Synthesizing' },
  { id: 'verifying', label: 'go test -race' },
  { id: 'completed', label: 'Verified' },
];

export function RunsStudio({
  runs,
  specs,
  autonomy,
  selectedRunId,
  onSelectRun,
  onRefresh,
  notify,
}: RunsStudioProps) {
  const studioMachine = useMachine(RunStudioUIMachine);
  const [prompt, setPrompt] = useState('');
  const [specId, setSpecId] = useState('');
  const [forceApproval, setForceApproval] = useState(false);
  const [liveRun, setLiveRun] = useState<RunRecord | null>(null);
  const [promoting, setPromoting] = useState(false);

  const activeRun =
    liveRun && liveRun.id === selectedRunId
      ? liveRun
      : runs.find((r) => r.id === selectedRunId) || runs[0] || null;

  // Sync UI state machine & SSE stream when active run changes
  useEffect(() => {
    if (!activeRun) {
      studioMachine.send('idle');
      return;
    }
    const st = activeRun.status;
    if (st === 'completed') {
      studioMachine.send('completed');
      return;
    }
    if (st === 'failed') {
      studioMachine.send('failed');
      return;
    }
    if (st === 'awaiting_approval') {
      studioMachine.send('awaiting_approval');
    } else {
      studioMachine.send('streaming');
    }

    const es = new EventSource(`${API_BASE}/api/runs/${activeRun.id}/stream`);
    es.onmessage = (ev) => {
      try {
        const updated = JSON.parse(ev.data) as RunRecord;
        setLiveRun(updated);
        if (updated.status === 'awaiting_approval') {
          studioMachine.send('awaiting_approval');
        } else if (updated.status === 'completed') {
          studioMachine.send('completed');
          onRefresh();
          es.close();
        } else if (updated.status === 'failed') {
          studioMachine.send('failed');
          onRefresh();
          es.close();
        }
      } catch {
        // ignore malformed frame
      }
    };
    es.onerror = () => {
      es.close();
    };
    return () => es.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeRun?.id, activeRun?.status]);

  const launchRun = useCallback(
    async (customPrompt?: string, customSpecId?: string) => {
      const p = (customPrompt ?? prompt).trim();
      const s = customSpecId ?? specId;
      if (!p && !s) return;
      studioMachine.send('submitting');
      try {
        const created = await api.startRun({
          prompt: p,
          spec_id: s,
          force_approval: forceApproval,
        });
        setPrompt('');
        setLiveRun(created);
        onSelectRun(created.id);
        onRefresh();
        notify(`Launched ${created.id}`);
      } catch (err: unknown) {
        studioMachine.send('failed');
        notify(err instanceof Error ? err.message : 'Failed to start run');
      }
    },
    [forceApproval, notify, onRefresh, onSelectRun, prompt, specId, studioMachine]
  );

  const handleApprove = async () => {
    if (!activeRun) return;
    try {
      const updated = await api.approveRun(activeRun.id);
      setLiveRun(updated);
      studioMachine.send('streaming');
      onRefresh();
      notify(`Approved ${activeRun.id} — synthesizing Go module`);
    } catch (err: unknown) {
      notify(err instanceof Error ? err.message : 'Approval failed');
    }
  };

  const handleReject = async () => {
    if (!activeRun) return;
    try {
      const updated = await api.rejectRun(activeRun.id, 'Rejected by operator in Runs Studio');
      setLiveRun(updated);
      studioMachine.send('failed');
      onRefresh();
      notify(`Rejected ${activeRun.id}`);
    } catch (err: unknown) {
      notify(err instanceof Error ? err.message : 'Rejection failed');
    }
  };

  const handlePromote = async () => {
    if (!activeRun) return;
    setPromoting(true);
    try {
      const updated = await api.promoteRun(activeRun.id);
      setLiveRun(updated);
      onRefresh();
      notify(`Promoted verified package -> ${updated.promoted_dir}/`);
    } catch (err: unknown) {
      notify(err instanceof Error ? err.message : 'Promotion failed');
    } finally {
      setPromoting(false);
    }
  };

  const stageOrder = PIPELINE_STAGES.map((s) => s.id);
  const currentStageIdx = activeRun ? stageOrder.indexOf(activeRun.stage) : -1;

  return (
    <div className="space-y-6">
      {/* Prompt & Blueprint Composer */}
      <div className="rk-card-elevated p-5">
        <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
          <div>
            <h2 className="text-base font-semibold">New Verified Go Build</h2>
            <p className="text-xs text-[var(--text-secondary)]">
              Pick a blueprint or describe any Go package — compiled &amp; verified in an isolated{' '}
              <code className="font-mono">.asf/workspaces/&lt;run-id&gt;</code> sandbox with{' '}
              <code className="font-mono">go test -race</code>.
            </p>
          </div>
          <span className="px-2.5 py-1 rounded-full text-[11px] font-mono bg-[var(--bg-subtle)] text-[var(--text-secondary)]">
            ui-state: {studioMachine.state} · mode: {autonomy}
          </span>
        </div>

        {/* 1-Click Blueprint Pills */}
        {specs.length > 0 && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5 mb-4">
            {specs.map((sp) => (
              <button
                key={sp.id}
                onClick={() => launchRun(sp.title + ' — ' + sp.description, sp.id)}
                className="text-left p-3 rounded-xl border border-[var(--border-hairline)] bg-[var(--bg-card)] hover:border-[var(--border-strong)] transition group"
              >
                <div className="flex items-center justify-between gap-1">
                  <span className="font-mono text-[11px] text-[var(--text-muted)]">{sp.id}</span>
                  <ArrowUpRight className="w-3.5 h-3.5 text-[var(--text-muted)] group-hover:text-[var(--text-primary)]" />
                </div>
                <div className="text-xs font-semibold mt-1 line-clamp-1">{sp.title}</div>
                <div className="text-[11px] text-[var(--text-secondary)] mt-0.5 line-clamp-1">
                  {sp.description}
                </div>
              </button>
            ))}
          </div>
        )}

        <div className="flex flex-col sm:flex-row gap-2.5">
          <select
            value={specId}
            onChange={(e) => setSpecId(e.target.value)}
            className="px-3 py-2.5 rounded-xl text-xs font-mono bg-[var(--bg-card)] border border-[var(--border-hairline)]"
          >
            <option value="">Custom prompt (auto-detect spec)</option>
            {specs.map((sp) => (
              <option key={sp.id} value={sp.id}>
                {sp.id}
              </option>
            ))}
          </select>

          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && launchRun()}
            placeholder="e.g. Build a thread-safe token-bucket rate limiter with burst capacity and race tests..."
            className="flex-1 px-4 py-2.5 rounded-xl text-sm bg-[var(--bg-card)] border border-[var(--border-hairline)] focus:outline-none focus:border-[var(--border-strong)]"
          />

          <label className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl text-xs text-[var(--text-secondary)] bg-[var(--bg-card)] border border-[var(--border-hairline)] cursor-pointer select-none">
            <input
              type="checkbox"
              checked={forceApproval || autonomy === 'crawl'}
              disabled={autonomy === 'crawl'}
              onChange={(e) => setForceApproval(e.target.checked)}
              className="rounded"
            />
            Require gate
          </label>

          <button
            onClick={() => launchRun()}
            disabled={(!prompt.trim() && !specId) || studioMachine.state === 'submitting'}
            className="rk-btn-primary px-5 py-2.5 text-sm inline-flex items-center justify-center gap-2"
          >
            {studioMachine.state === 'submitting' ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Play className="w-4 h-4" />
            )}
            Start Run
          </button>
        </div>
      </div>

      {/* Split Pane: Run History List + Live Build Theater */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <div className="lg:col-span-4 space-y-2">
          <div className="flex items-center justify-between px-1">
            <span className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)]">
              Runs ({runs.length})
            </span>
          </div>

          {runs.length === 0 ? (
            <div className="rk-card p-6 text-center text-xs text-[var(--text-secondary)]">
              No runs yet — click any blueprint card above to watch a live Go build &amp;{' '}
              <code className="font-mono">go test -race</code> verification.
            </div>
          ) : (
            <div className="space-y-2 max-h-[680px] overflow-y-auto pr-1">
              {runs.map((r) => {
                const isSelected = activeRun?.id === r.id;
                return (
                  <button
                    key={r.id}
                    onClick={() => onSelectRun(r.id)}
                    className={`w-full text-left p-3.5 rounded-2xl border transition ${
                      isSelected
                        ? 'bg-[var(--bg-elevated)] border-[var(--border-strong)] shadow-sm'
                        : 'bg-[var(--bg-card)] border-[var(--border-hairline)] hover:bg-[var(--bg-elevated)]'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-mono text-xs font-semibold">{r.id}</span>
                      <RunStatusPill status={r.status} />
                    </div>
                    <div className="text-sm font-medium mt-1 line-clamp-1">{r.title}</div>
                    <div className="flex items-center justify-between text-[11px] text-[var(--text-muted)] font-mono mt-2">
                      <span>{r.spec_id || 'custom-spec'}</span>
                      <span>
                        {r.score > 0 ? `${r.score}/100` : r.stage}
                        {r.promotion_state === 'promoted' ? ' · promoted' : ''}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Live Build Theater */}
        <div className="lg:col-span-8 space-y-4">
          {!activeRun ? (
            <div className="rk-card p-12 text-center space-y-3">
              <Sparkles className="w-7 h-7 mx-auto text-[var(--text-muted)]" />
              <h3 className="text-base font-semibold">Ready for Your First Factory Run</h3>
              <p className="text-sm text-[var(--text-secondary)] max-w-md mx-auto">
                Click any blueprint card above (such as{' '}
                <span className="font-mono">spec-token-bucket</span>) to synthesize multi-file Go
                code and execute <span className="font-mono">go test -race</span> live.
              </p>
            </div>
          ) : (
            <>
              {/* Header + Promote */}
              <div className="rk-card-elevated p-5 space-y-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs px-2 py-0.5 rounded bg-[var(--bg-subtle)]">
                        {activeRun.id}
                      </span>
                      <RunStatusPill status={activeRun.status} />
                      {activeRun.score > 0 && (
                        <span className="rk-pill-healthy px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold">
                          Score {activeRun.score}/100
                        </span>
                      )}
                    </div>
                    <h3 className="text-lg font-semibold mt-1.5">{activeRun.title}</h3>
                    <p className="text-xs font-mono text-[var(--text-muted)] mt-0.5">
                      workspace: {activeRun.workspace_dir}
                      {activeRun.promoted_dir ? ` -> ${activeRun.promoted_dir}/` : ''}
                    </p>
                  </div>

                  {activeRun.status === 'completed' && (
                    <button
                      onClick={handlePromote}
                      disabled={promoting}
                      className="rk-btn-primary px-4 py-2 text-xs inline-flex items-center gap-1.5"
                    >
                      {promoting ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <FolderOutput className="w-3.5 h-3.5" />
                      )}
                      {activeRun.promotion_state === 'promoted'
                        ? `Re-Promote (${activeRun.promoted_dir}/)`
                        : 'Promote to generated/'}
                    </button>
                  )}
                </div>

                {/* 6-Node State Stepper */}
                <div className="grid grid-cols-3 sm:grid-cols-6 gap-2 pt-1">
                  {PIPELINE_STAGES.map((st, idx) => {
                    const isGateSkipped =
                      st.id === 'awaiting_approval' &&
                      activeRun.status !== 'awaiting_approval' &&
                      !activeRun.steps.some((s) => s.stage === 'awaiting_approval');

                    const reached =
                      activeRun.status === 'completed' ||
                      (currentStageIdx >= idx && !isGateSkipped);
                    const isCurrent = activeRun.stage === st.id;

                    return (
                      <div
                        key={st.id}
                        className={`px-2.5 py-2 rounded-xl border text-center text-[11px] font-mono transition ${
                          isCurrent && activeRun.status !== 'completed'
                            ? 'bg-[var(--text-primary)] text-[var(--bg-elevated)] border-[var(--text-primary)] font-semibold'
                            : reached
                              ? 'rk-pill-healthy border-transparent font-medium'
                              : 'bg-[var(--bg-card)] border-[var(--border-hairline)] text-[var(--text-muted)]'
                        }`}
                      >
                        {isGateSkipped ? 'Gate (auto)' : st.label}
                      </div>
                    );
                  })}
                </div>

                {/* Crawl Approval Gate Banner */}
                {activeRun.status === 'awaiting_approval' && (
                  <div className="rk-pill-warn rounded-xl p-4 flex flex-wrap items-center justify-between gap-3 border border-[#D8B982]">
                    <div className="flex items-start gap-2.5">
                      <ShieldAlert className="w-5 h-5 shrink-0 mt-0.5" />
                      <div>
                        <div className="text-sm font-semibold">
                          Human-in-the-Loop Checkpoint (awaiting_approval)
                        </div>
                        <div className="text-xs opacity-90">
                          Specification planning succeeded. Approve to transition{' '}
                          <code className="font-mono">awaiting_approval -&gt; synthesizing</code>.
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={handleReject}
                        className="rk-btn-secondary px-3 py-1.5 text-xs"
                      >
                        Reject
                      </button>
                      <button
                        onClick={handleApprove}
                        className="rk-btn-primary px-3.5 py-1.5 text-xs"
                      >
                        Approve &amp; Synthesize
                      </button>
                    </div>
                  </div>
                )}

                {activeRun.error && (
                  <div className="rk-pill-err rounded-xl p-3.5 text-xs font-mono whitespace-pre-wrap">
                    {activeRun.error}
                  </div>
                )}
              </div>

              {/* Synthesized Multi-File Go Code */}
              <CodeViewer files={activeRun.files || []} />

              {/* Compiler & go test -race Output + Rubric */}
              <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
                <div className="md:col-span-8 rk-card-elevated overflow-hidden">
                  <div className="px-4 py-2.5 border-b border-[var(--border-hairline)] bg-[var(--bg-card)] flex items-center justify-between">
                    <div className="inline-flex items-center gap-2 text-xs font-semibold">
                      <Terminal className="w-3.5 h-3.5" />
                      Sandbox Verification Output
                    </div>
                    <div className="flex items-center gap-2 text-[11px] font-mono">
                      <span className={activeRun.build_passed ? 'text-[#1F5E38]' : 'text-[var(--text-muted)]'}>
                        build:{activeRun.build_passed ? 'ok' : '—'}
                      </span>
                      <span className={activeRun.vet_passed ? 'text-[#1F5E38]' : 'text-[var(--text-muted)]'}>
                        vet:{activeRun.vet_passed ? 'ok' : '—'}
                      </span>
                      <span className={activeRun.tests_passed ? 'text-[#1F5E38]' : 'text-[var(--text-muted)]'}>
                        race:{activeRun.tests_passed ? 'ok' : '—'}
                      </span>
                    </div>
                  </div>
                  <pre className="p-4 text-[11.5px] font-mono leading-relaxed max-h-64 overflow-auto whitespace-pre-wrap text-[var(--text-primary)]">
                    {activeRun.test_output || 'Waiting for sandbox verification stage...'}
                  </pre>
                </div>

                <div className="md:col-span-4 rk-card p-4 space-y-3">
                  <div className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)]">
                    Transition Log &amp; Rubric
                  </div>
                  {activeRun.rubric && Object.keys(activeRun.rubric).length > 0 && (
                    <div className="space-y-1.5 pb-2 border-b border-[var(--border-hairline)] text-xs font-mono">
                      {Object.entries(activeRun.rubric).map(([k, v]) => (
                        <div key={k} className="flex justify-between">
                          <span className="text-[var(--text-secondary)]">{k}</span>
                          <span className="font-semibold">+{v}</span>
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="space-y-2 max-h-44 overflow-y-auto text-xs">
                    {(activeRun.steps || []).map((s, idx) => (
                      <div key={idx} className="flex items-start gap-2">
                        <span className="font-mono text-[10.5px] px-1.5 py-0.5 rounded bg-[var(--bg-subtle)] shrink-0">
                          {s.stage}
                        </span>
                        <span className="text-[var(--text-secondary)] text-[11.5px] leading-snug">
                          {s.message}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

export function RunStatusPill({ status }: { status: string }) {
  if (status === 'completed') {
    return (
      <span className="rk-pill-healthy inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-medium">
        <CheckCircle2 className="w-3 h-3" />
        completed
      </span>
    );
  }
  if (status === 'awaiting_approval') {
    return (
      <span className="rk-pill-warn inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-medium">
        <AlertTriangle className="w-3 h-3" />
        awaiting_approval
      </span>
    );
  }
  if (status === 'failed') {
    return (
      <span className="rk-pill-err inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-medium">
        <XCircle className="w-3 h-3" />
        failed
      </span>
    );
  }
  return (
    <span className="rk-pill-warn inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-medium">
      <Clock className="w-3 h-3 animate-spin" />
      {status}
    </span>
  );
}
