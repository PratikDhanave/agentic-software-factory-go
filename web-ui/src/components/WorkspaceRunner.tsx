'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiClient, FactoryTask, SpecSummary } from '@/lib/api';
import { specHarnessMachine, statusBadge, useMachine } from '@/lib/stateMachine';
import {
  CheckCircle2,
  FolderGit2,
  Loader2,
  Play,
  Plus,
  RefreshCw,
  ShieldCheck,
  Terminal,
  X,
} from 'lucide-react';

export default function WorkspaceRunner({
  onLaunchInWorkbench,
}: {
  onLaunchInWorkbench?: (specId: string) => void;
}) {
  const [specs, setSpecs] = useState<SpecSummary[]>([]);
  const [selectedSpec, setSelectedSpec] = useState<string>('');
  const [specDetail, setSpecDetail] = useState<{ spec: any; prompt: string } | null>(null);
  const [tasks, setTasks] = useState<FactoryTask[]>([]);
  const [activeTask, setActiveTask] = useState<FactoryTask | null>(null);
  const [repoVerifyResult, setRepoVerifyResult] = useState<any>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const [newSpecId, setNewSpecId] = useState('spec-worker-pool');
  const [newSpecTitle, setNewSpecTitle] = useState('Bounded Worker Pool');
  const [newSpecGoal, setNewSpecGoal] = useState(
    'Implement a context-aware Go worker pool with graceful shutdown and concurrent unit tests'
  );

  const harness = useMachine(specHarnessMachine);
  const [error, setError] = useState('');

  const loadSpecsAndTasks = useCallback(async () => {
    try {
      const [specData, taskData] = await Promise.all([apiClient.getSpecs(), apiClient.getTasks()]);
      const list = specData.specs || [];
      setSpecs(list);
      if (!selectedSpec && list.length > 0) {
        setSelectedSpec(list[0].id);
      }
      const recent = taskData.tasks || [];
      setTasks(recent);
      if (activeTask) {
        const updated = recent.find((t) => t.id === activeTask.id);
        if (updated) setActiveTask(updated);
      } else if (recent.length > 0) {
        setActiveTask(recent[0]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load specifications');
    }
  }, [selectedSpec, activeTask]);

  useEffect(() => {
    loadSpecsAndTasks();
    const interval = setInterval(loadSpecsAndTasks, 3500);
    return () => clearInterval(interval);
  }, [loadSpecsAndTasks]);

  useEffect(() => {
    if (!selectedSpec) return;
    apiClient
      .getSpecDetail(selectedSpec)
      .then(setSpecDetail)
      .catch(() => setSpecDetail(null));
  }, [selectedSpec]);

  const handleLaunchWorkerTask = async (action: string) => {
    if (!selectedSpec || !harness.send('launching_task')) return;
    setError('');
    try {
      const res = await apiClient.launchTask(action, selectedSpec);
      await loadSpecsAndTasks();
      const created = await apiClient.getTask(res.task_id);
      setActiveTask(created);
      harness.send('succeeded');
    } catch (err) {
      harness.send('error');
      setError(err instanceof Error ? err.message : 'Task launch failed');
    }
  };

  const handleCreateSpec = async () => {
    if (!newSpecId.trim() || !newSpecTitle.trim() || !newSpecGoal.trim()) return;
    if (!harness.send('creating_spec')) return;
    setError('');
    try {
      await apiClient.createSpec({
        id: newSpecId.trim(),
        version: '1.0',
        title: newSpecTitle.trim(),
        description: newSpecGoal.trim(),
        intent: {
          goal: newSpecGoal.trim(),
          context: 'Created in ASF Blueprints Studio',
          success_criteria: ['Compiles cleanly with go build', 'Passes go test -race'],
        },
        constraints: [{ id: 'c1', type: 'must_not', description: 'panic(' }],
        steps: [
          {
            id: 'step1',
            name: 'Implement & Test',
            description: 'Synthesize Go source and table-driven tests',
          },
        ],
      });
      setShowCreateModal(false);
      setSelectedSpec(newSpecId.trim());
      harness.send('succeeded');
      await loadSpecsAndTasks();
    } catch (err) {
      harness.send('error');
      setError(err instanceof Error ? err.message : 'Failed to save specification');
    }
  };

  const handleVerifyRepo = async () => {
    if (!harness.send('verifying_repo')) return;
    setError('');
    try {
      const res = await apiClient.runVerification();
      setRepoVerifyResult(res);
      harness.send('succeeded');
    } catch (err) {
      harness.send('error');
      setError(err instanceof Error ? err.message : 'Repository verification failed');
    }
  };

  const activeSpecObj = specs.find((s) => s.id === selectedSpec);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold ide-ink">
            Blueprints &amp; Specifications ({specs.length})
          </h1>
          <p className="text-sm ide-ink-secondary mt-1">
            Inspect machine-readable YAML specifications in <code className="font-mono">specs/</code>, author new blueprints, or dispatch worker tasks directly into isolated sandboxes.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={handleVerifyRepo}
            disabled={harness.current === 'verifying_repo'}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg ide-surface border ide-border text-xs font-medium ide-ink hover:border-blue-500/50 transition"
          >
            {harness.current === 'verifying_repo' ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin text-blue-600" />
            ) : (
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
            )}
            Verify Repository
          </button>

          <button
            type="button"
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-slate-900 hover:bg-slate-800 dark:bg-blue-600 text-white text-xs font-semibold transition"
          >
            <Plus className="w-3.5 h-3.5" />
            New Specification
          </button>
        </div>
      </div>

      {error && (
        <div className="p-3.5 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30 text-xs text-rose-700 dark:text-rose-300">
          {error}
        </div>
      )}

      {repoVerifyResult && (
        <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/30 flex items-center justify-between text-xs">
          <div className="flex items-center gap-2 text-emerald-800 dark:text-emerald-200">
            <CheckCircle2 className="w-4 h-4 text-emerald-600" />
            <span>
              Repository verification completed:{' '}
              <strong>{repoVerifyResult.success ? 'ALL GATES PASSED' : 'ISSUES REPORTED'}</strong>
            </span>
          </div>
          <button
            type="button"
            onClick={() => setRepoVerifyResult(null)}
            className="ide-ink-muted hover:ide-ink"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Create Spec Drawer/Card */}
      {showCreateModal && (
        <div className="ide-surface border border-blue-500/40 rounded-xl p-5 space-y-4 shadow-md">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold ide-ink">Create New YAML Specification</h3>
            <button
              type="button"
              onClick={() => setShowCreateModal(false)}
              className="p-1 rounded ide-ink-muted hover:ide-ink"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium ide-ink mb-1">Spec ID</label>
              <input
                value={newSpecId}
                onChange={(e) => setNewSpecId(e.target.value)}
                className="w-full px-3 py-2 rounded-lg ide-canvas border ide-border text-xs font-mono ide-ink"
              />
            </div>
            <div>
              <label className="block text-xs font-medium ide-ink mb-1">Title</label>
              <input
                value={newSpecTitle}
                onChange={(e) => setNewSpecTitle(e.target.value)}
                className="w-full px-3 py-2 rounded-lg ide-canvas border ide-border text-xs ide-ink"
              />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium ide-ink mb-1">Engineering Goal</label>
            <textarea
              rows={2}
              value={newSpecGoal}
              onChange={(e) => setNewSpecGoal(e.target.value)}
              className="w-full px-3 py-2 rounded-lg ide-canvas border ide-border text-xs ide-ink"
            />
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShowCreateModal(false)}
              className="px-3.5 py-1.5 rounded-lg ide-canvas border ide-border text-xs ide-ink-secondary"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleCreateSpec}
              className="px-4 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-semibold"
            >
              Save to specs/
            </button>
          </div>
        </div>
      )}

      {/* 3-Pane Spec Explorer */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Specs List */}
        <div className="lg:col-span-4 ide-surface border ide-border rounded-xl overflow-hidden">
          <div className="px-4 py-3 ide-subtle border-b ide-border flex items-center justify-between">
            <span className="text-xs font-semibold ide-ink">specs/*.yaml</span>
            <button
              type="button"
              onClick={loadSpecsAndTasks}
              className="ide-ink-muted hover:ide-ink"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
          </div>
          <div className="divide-y ide-border">
            {specs.map((sp) => {
              const active = sp.id === selectedSpec;
              return (
                <button
                  key={sp.id}
                  type="button"
                  onClick={() => setSelectedSpec(sp.id)}
                  className={`w-full text-left px-4 py-3 transition ${
                    active
                      ? 'bg-blue-600/[0.06] border-l-2 border-l-blue-600'
                      : 'hover:bg-black/[0.02] dark:hover:bg-white/[0.02]'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-semibold ide-ink truncate">{sp.title}</span>
                    <span className="text-[10px] font-mono ide-ink-muted">v{sp.version}</span>
                  </div>
                  <div className="text-[11px] font-mono ide-ink-muted mt-0.5">{sp.id}</div>
                </button>
              );
            })}
          </div>
        </div>

        {/* Right: Spec Inspector & Worker Dispatch */}
        <div className="lg:col-span-8 space-y-5">
          {activeSpecObj && (
            <div className="ide-surface border ide-border rounded-xl p-5 space-y-4">
              <div className="flex flex-wrap items-start justify-between gap-3 pb-4 border-b ide-border">
                <div>
                  <div className="flex items-center gap-2">
                    <FolderGit2 className="w-4 h-4 text-blue-600" />
                    <h2 className="text-base font-semibold ide-ink">{activeSpecObj.title}</h2>
                  </div>
                  <p className="text-xs ide-ink-secondary mt-1">
                    {activeSpecObj.description || activeSpecObj.goal}
                  </p>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {onLaunchInWorkbench && (
                    <button
                      type="button"
                      onClick={() => onLaunchInWorkbench(activeSpecObj.id)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold transition"
                    >
                      <Play className="w-3.5 h-3.5 fill-current" />
                      Build in Studio
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={() => handleLaunchWorkerTask('generate')}
                    disabled={harness.current === 'launching_task'}
                    className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg ide-canvas border ide-border text-xs font-medium ide-ink hover:border-blue-500/40 transition"
                  >
                    <Terminal className="w-3.5 h-3.5" />
                    Dispatch Worker Task
                  </button>
                </div>
              </div>

              {activeSpecObj.constraints && activeSpecObj.constraints.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2">
                    Specification Invariants
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {activeSpecObj.constraints.map((c) => (
                      <div
                        key={c.id}
                        className="px-3 py-2 rounded-lg ide-canvas border ide-border text-xs flex items-center gap-2"
                      >
                        <span className="font-mono text-[10px] uppercase px-1.5 py-0.5 rounded bg-blue-600/10 text-blue-700 dark:text-blue-300 font-semibold">
                          {c.type}
                        </span>
                        <span className="ide-ink truncate">{c.description}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {specDetail?.prompt && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider ide-ink-muted mb-2">
                    Compiled Agent Prompt Preview
                  </div>
                  <pre className="p-3.5 rounded-lg ide-canvas border ide-border text-xs font-mono ide-ink-secondary overflow-x-auto max-h-48 leading-relaxed">
                    {specDetail.prompt}
                  </pre>
                </div>
              )}
            </div>
          )}

          {/* Worker Pool Tasks */}
          <div className="ide-surface border ide-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 ide-subtle border-b ide-border flex items-center justify-between">
              <span className="text-xs font-semibold ide-ink">
                Worker Pool Tasks ({tasks.length})
              </span>
              <span className="text-[11px] font-mono ide-ink-muted">
                machine: task (pending → running → completed)
              </span>
            </div>
            {tasks.length === 0 ? (
              <div className="p-8 text-center text-xs ide-ink-muted">
                No background worker tasks dispatched yet.
              </div>
            ) : (
              <div className="divide-y ide-border">
                {tasks.slice(0, 8).map((t) => (
                  <div
                    key={t.id}
                    onClick={() => setActiveTask(t)}
                    className="px-4 py-3 flex items-center justify-between gap-3 text-xs cursor-pointer hover:bg-black/[0.02] dark:hover:bg-white/[0.02]"
                  >
                    <div>
                      <span className="font-semibold uppercase ide-ink">{t.type}</span>
                      <span className="ide-ink-muted mx-1.5">•</span>
                      <span className="font-mono ide-ink-secondary">{t.spec_id}</span>
                      <div className="text-[11px] font-mono ide-ink-muted mt-0.5">{t.id}</div>
                    </div>
                    <span
                      className={`px-2.5 py-0.5 rounded-full text-[11px] font-mono font-medium ${
                        statusBadge[t.status] || 'ide-subtle ide-ink-secondary'
                      }`}
                    >
                      {t.status}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
