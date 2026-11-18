'use client';

import { useState } from 'react';
import { apiClient, WorkflowResponse } from '@/lib/api';
import { useMachine, workflowStudioMachine } from '@/lib/stateMachine';
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  GitBranch,
  Loader2,
  Play,
  ShieldAlert,
  Sparkles,
} from 'lucide-react';

const PRESETS = [
  {
    label: 'OAuth2 PKCE Hardening',
    title: 'Harden OAuth2 PKCE token refresh and session rotation',
    description:
      'Add cryptographic PKCE verification, sliding refresh token rotation, replay detection, and unit tests for concurrent auth requests.',
  },
  {
    label: 'Worker Pool Backpressure',
    title: 'Fix worker queue latency spike under burst load',
    description:
      'Profile dispatcher lock contention, add bounded ring-buffer backpressure metrics, and benchmark throughput under 500 RPS.',
  },
  {
    label: 'SQLite Audit Ledger',
    title: 'Implement tamper-evident SQLite audit log migration',
    description:
      'Design WAL-safe audit log schema with SHA-256 hash chaining, retention pruning, and automated rollback verification.',
  },
];

interface Props {
  onApprovalQueued?: () => void;
}

export default function WorkflowGenerator({ onApprovalQueued }: Props) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const studio = useMachine(workflowStudioMachine);
  const [result, setResult] = useState<WorkflowResponse | null>(null);
  const [execResult, setExecResult] = useState<any>(null);
  const [errorMessage, setErrorMessage] = useState('');

  const isCoordinating = studio.current === 'coordinating';
  const isExecuting = studio.current === 'executing_dag';

  const handleGenerate = async () => {
    if (!title.trim() || !description.trim()) {
      studio.send('validation_error');
      setErrorMessage('Please fill in both title and description');
      return;
    }

    if (!studio.send('coordinating')) return;
    setErrorMessage('');
    setResult(null);
    setExecResult(null);

    try {
      const response = await apiClient.generateWorkflow({
        issue_title: title,
        issue_description: description,
      });
      setResult(response);
      if (response.execution) {
        setExecResult(response.execution);
        studio.send('executed');
      } else if (response.requires_approval) {
        studio.send('awaiting_gate');
        onApprovalQueued?.();
      } else {
        studio.send('planned');
      }
    } catch (err) {
      studio.send('error');
      setErrorMessage(err instanceof Error ? err.message : 'Failed to generate workflow');
    }
  };

  const handleManualExecute = async () => {
    if (!result?.workflow_id || !studio.send('executing_dag')) return;
    setErrorMessage('');
    try {
      const exec = await apiClient.executeWorkflow(result.workflow_id, result.session_id);
      setExecResult(exec);
      studio.send('executed');
    } catch (err) {
      studio.send('error');
      setErrorMessage(err instanceof Error ? err.message : 'Failed to execute workflow');
    }
  };

  return (
    <div className="space-y-6">
      <div className="ide-surface border ide-border rounded-xl p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-4 mb-5">
          <div>
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h2 className="text-lg font-semibold ide-ink">Generate Workflow</h2>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded ide-subtle ide-ink-muted">
                state: {studio.current}
              </span>
            </div>
            <p className="text-xs ide-ink-secondary mt-1">
              Coordinate Product, Architect, Security, Design, Developer, and QA agents into a level-parallel execution DAG.
            </p>
          </div>

          <div className="flex flex-wrap gap-1.5">
            {PRESETS.map((preset) => (
              <button
                key={preset.label}
                type="button"
                onClick={() => {
                  setTitle(preset.title);
                  setDescription(preset.description);
                  setErrorMessage('');
                }}
                className="text-xs px-3 py-1.5 rounded-lg ide-canvas hover:ide-subtle ide-ink-secondary hover:ide-ink border ide-border transition"
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-xs font-medium ide-ink mb-1.5">
              Issue Title
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g., Fix authentication bug in OAuth flow"
              className="w-full px-3.5 py-2 ide-surface border ide-border rounded-lg text-sm ide-ink placeholder:ide-ink-muted focus:outline-none focus:ring-2 focus:ring-blue-600/20 focus:border-blue-600"
            />
          </div>

          <div>
            <label className="block text-xs font-medium ide-ink mb-1.5">
              Issue Description &amp; Acceptance Criteria
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe the issue in detail..."
              rows={3}
              className="w-full px-3.5 py-2 ide-surface border ide-border rounded-lg text-sm ide-ink placeholder:ide-ink-muted focus:outline-none focus:ring-2 focus:ring-blue-600/20 focus:border-blue-600"
            />
          </div>

          {errorMessage && (
            <div className="flex items-start gap-2.5 p-3.5 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30 rounded-lg">
              <AlertCircle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
              <p className="text-xs text-rose-700 dark:text-rose-300">{errorMessage}</p>
            </div>
          )}

          <div className="flex justify-end">
            <button
              type="button"
              onClick={handleGenerate}
              disabled={isCoordinating}
              aria-label="Generate Workflow"
              className="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-lg bg-slate-900 hover:bg-slate-800 dark:bg-blue-600 dark:hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium transition shadow-sm"
            >
              {isCoordinating ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <GitBranch className="w-4 h-4" />
              )}
              {isCoordinating ? 'Generating...' : 'Generate Workflow'}
            </button>
          </div>
        </div>
      </div>

      {result && (
        <div className="ide-surface border ide-border rounded-xl p-6 space-y-5 shadow-sm">
          <div className="flex flex-wrap items-start justify-between gap-4 pb-4 border-b ide-border">
            <div className="flex items-start gap-3">
              {result.requires_approval ? (
                <ShieldAlert className="w-5 h-5 text-amber-500 shrink-0 mt-0.5" />
              ) : (
                <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5" />
              )}
              <div>
                <h3 className="text-base font-semibold ide-ink">
                  {result.requires_approval
                    ? 'Workflow Paused at Governance Gate'
                    : 'Workflow Generated'}
                </h3>
                <p className="text-xs font-mono ide-ink-muted mt-0.5">
                  ID: {result.workflow_id}
                  {result.session_id ? ` • Session: ${result.session_id}` : ''}
                </p>
              </div>
            </div>

            {!result.requires_approval && !execResult && (
              <button
                type="button"
                onClick={handleManualExecute}
                disabled={isExecuting}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold transition"
              >
                {isExecuting ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Play className="w-3.5 h-3.5 fill-current" />
                )}
                {isExecuting ? 'Executing DAG...' : 'Execute DAG Now'}
              </button>
            )}
          </div>

          {result.dag_levels && result.dag_levels.length > 0 && (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2.5">
                Topological Execution Levels
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {result.dag_levels.map((lvl, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <div className="px-3 py-2 rounded-lg ide-canvas border ide-border">
                      <div className="text-[10px] font-mono ide-ink-muted">Level {idx + 1}</div>
                      <div className="flex flex-wrap gap-1.5 mt-1">
                        {lvl.map((jobId) => (
                          <span
                            key={jobId}
                            className="text-xs font-mono px-2 py-0.5 rounded ide-surface border ide-border ide-ink"
                          >
                            {jobId}
                          </span>
                        ))}
                      </div>
                    </div>
                    {idx < result.dag_levels!.length - 1 && (
                      <ArrowRight className="w-4 h-4 ide-ink-muted" />
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {result.decisions && result.decisions.length > 0 && (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2.5">
                Specialized Agent Decisions ({result.decisions.length})
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {result.decisions.map((dec, idx) => {
                  const pct = Math.round((dec.confidence ?? 0) * 100);
                  return (
                    <div
                      key={`${dec.agent_role}-${idx}`}
                      className="p-4 rounded-lg ide-canvas border ide-border space-y-1.5"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs font-semibold capitalize ide-ink">
                          {dec.agent_role}
                        </span>
                        <span className="text-[11px] font-mono font-semibold px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:border-emerald-500/30">
                          {pct}% confidence
                        </span>
                      </div>
                      <div className="text-xs font-medium ide-ink">
                        {dec.choice || dec.decision}
                      </div>
                      {dec.reasoning && (
                        <p className="text-xs ide-ink-secondary leading-relaxed">{dec.reasoning}</p>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {execResult && (
            <div className="p-4 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/30 text-xs text-emerald-800 dark:text-emerald-200">
              DAG execution status:{' '}
              <strong className="font-mono">{execResult.status || 'completed'}</strong>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
