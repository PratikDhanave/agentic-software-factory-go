'use client';

import { useState, useEffect, useCallback } from 'react';
import { apiClient, PendingApproval } from '@/lib/api';
import { approvalsInboxMachine, useMachine } from '@/lib/stateMachine';
import { CheckCircle2, Clock, RefreshCw, ShieldCheck, XCircle } from 'lucide-react';

interface Props {
  onApprovalResolved?: () => void;
}

export default function ApprovalsPanel({ onApprovalResolved }: Props) {
  const [approvals, setApprovals] = useState<PendingApproval[]>([]);
  const inbox = useMachine(approvalsInboxMachine);
  const [errorMessage, setErrorMessage] = useState('');
  const [banner, setBanner] = useState('');
  const [activeKey, setActiveKey] = useState<string | null>(null);
  const [overrideSelection, setOverrideSelection] = useState<Record<string, string>>({});

  const fetchApprovals = useCallback(async () => {
    try {
      const response = await apiClient.getPendingApprovals();
      const list = response.approvals || response.pending_approvals || [];
      setApprovals(list);
      setErrorMessage('');
      inbox.send(list.length > 0 ? 'ready' : 'empty');
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Unable to load approvals');
      inbox.send('error');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    fetchApprovals();
    const interval = setInterval(fetchApprovals, 5000);
    return () => clearInterval(interval);
  }, [fetchApprovals]);

  const handleDecision = async (item: PendingApproval, approved: boolean) => {
    const key = item.id || `${item.session_id}:${item.agent_role}`;
    setActiveKey(key);
    inbox.send('submitting');
    setBanner('');
    setErrorMessage('');

    try {
      const chosenOverride = overrideSelection[key];
      const res = await apiClient.submitApproval({
        approval_id: item.id,
        session_id: item.session_id,
        approved,
        override: approved && chosenOverride ? chosenOverride : undefined,
      });
      setBanner(res.message || (approved ? 'Approved and resumed execution.' : 'Decision rejected.'));
      await fetchApprovals();
      onApprovalResolved?.();
    } catch (err) {
      inbox.send('error');
      setErrorMessage(err instanceof Error ? err.message : 'Submission failed');
    } finally {
      setActiveKey(null);
    }
  };

  return (
    <div className="ide-surface border ide-border rounded-xl p-6 space-y-5 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b ide-border">
        <div className="flex items-center gap-2.5">
          <ShieldCheck className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <div>
            <h2 className="text-lg font-semibold ide-ink">
              Pending Approvals ({approvals.length})
            </h2>
            <p className="text-xs ide-ink-secondary">
              Human-in-the-loop review queue for Supervised (Crawl) and low-confidence agent decisions.
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-[11px] font-mono px-2 py-0.5 rounded ide-subtle ide-ink-muted">
            state: {inbox.current}
          </span>
          <button
            type="button"
            onClick={fetchApprovals}
            className="p-2 rounded-lg ide-canvas border ide-border ide-ink-secondary hover:ide-ink transition"
            title="Refresh approvals"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {banner && (
        <div className="p-3 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/30 text-xs text-emerald-800 dark:text-emerald-200">
          {banner}
        </div>
      )}

      {errorMessage && (
        <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30 text-xs text-rose-700 dark:text-rose-300">
          {errorMessage}
        </div>
      )}

      {inbox.current === 'loading' && approvals.length === 0 ? (
        <div className="py-10 text-center text-xs ide-ink-muted">
          Loading pending approvals...
        </div>
      ) : approvals.length === 0 ? (
        <div className="py-12 text-center rounded-lg ide-canvas border border-dashed ide-border">
          <CheckCircle2 className="w-8 h-8 text-emerald-600 mx-auto mb-2" />
          <p className="text-sm font-medium ide-ink">No pending approvals</p>
          <p className="text-xs ide-ink-muted mt-1 max-w-md mx-auto">
            All governance checkpoints are clear. Switch Autonomy Mode to Supervised (Crawl) at the top right to pause every build for review.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {approvals.map((item, idx) => {
            const key = item.id || `${item.session_id}:${item.agent_role}:${idx}`;
            const confPct = Math.round((item.confidence ?? 0) * 100);
            const options = item.options || item.alternatives || [];
            const proposed = item.current_choice || item.decision || '';
            const busy = activeKey === key;

            return (
              <div
                key={key}
                className="p-4 rounded-xl ide-canvas border ide-border space-y-3"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold ide-ink">{item.agent_role}</span>
                      <span className="text-xs font-mono px-2 py-0.5 rounded bg-amber-50 text-amber-800 border border-amber-200 dark:bg-amber-500/15 dark:text-amber-300 dark:border-amber-500/30">
                        {confPct}% confidence
                      </span>
                    </div>
                    {item.issue_title && (
                      <div className="text-xs font-medium ide-ink-secondary mt-1">
                        {item.issue_title}
                      </div>
                    )}
                    <div className="text-[11px] font-mono ide-ink-muted mt-0.5 flex items-center gap-1.5">
                      <Clock className="w-3 h-3" />
                      Session: {item.session_id}
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => handleDecision(item, true)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-xs font-semibold transition"
                    >
                      <CheckCircle2 className="w-3.5 h-3.5" />
                      Approve
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => handleDecision(item, false)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg ide-surface hover:bg-rose-50 dark:hover:bg-rose-500/10 border ide-border text-rose-600 dark:text-rose-400 text-xs font-semibold transition"
                    >
                      <XCircle className="w-3.5 h-3.5" />
                      Reject
                    </button>
                  </div>
                </div>

                {proposed && (
                  <div className="text-xs ide-ink">
                    <span className="ide-ink-muted">Proposed strategy: </span>
                    <strong className="font-medium">{proposed}</strong>
                  </div>
                )}

                {item.reasoning && (
                  <p className="text-xs ide-ink-secondary leading-relaxed">{item.reasoning}</p>
                )}

                {options.length > 0 && (
                  <div className="pt-2 border-t ide-border flex flex-wrap items-center gap-2">
                    <span className="text-[11px] ide-ink-muted">Optional override:</span>
                    {options.map((opt) => {
                      const activeOpt = (overrideSelection[key] ?? proposed) === opt;
                      return (
                        <button
                          key={opt}
                          type="button"
                          onClick={() =>
                            setOverrideSelection((prev) => ({ ...prev, [key]: opt }))
                          }
                          className={`text-xs px-2.5 py-1 rounded-md border transition ${
                            activeOpt
                              ? 'bg-blue-600/10 border-blue-600 text-blue-700 dark:text-blue-300 font-medium'
                              : 'ide-surface ide-border ide-ink-secondary hover:ide-ink'
                          }`}
                        >
                          {opt}
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
