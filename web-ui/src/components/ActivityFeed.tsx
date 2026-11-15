'use client';

import { useCallback, useEffect, useState } from 'react';
import { statusBadge } from '@/lib/statusBadge';

interface TaskEntry {
  id: string;
  type: string;
  spec_id: string;
  status: string;
  terminal: boolean;
  retries: number;
  error: string;
  created_at: string;
  completed_at: string | null;
}

const POLL_MS = 3000;

function formatTime(iso: string | null): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function describe(t: TaskEntry): string {
  const what = `${t.type} · ${t.spec_id}`;
  switch (t.status) {
    case 'completed':
      return `${what} completed${t.retries > 0 ? ` after ${t.retries} retr${t.retries === 1 ? 'y' : 'ies'}` : ''}`;
    case 'failed':
      return `${what} failed${t.error ? `: ${t.error}` : ''}`;
    case 'running':
      return `${what} running…`;
    default:
      return `${what} queued`;
  }
}

export function ActivityFeed() {
  const [tasks, setTasks] = useState<TaskEntry[] | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await fetch('/api/tasks');
      if (!res.ok) throw new Error(`${res.status}`);
      const data = await res.json();
      setTasks(data.tasks ?? []);
    } catch {
      setTasks(null);
    }
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, POLL_MS);
    window.addEventListener('taskLaunched', refresh);
    return () => {
      clearInterval(interval);
      window.removeEventListener('taskLaunched', refresh);
    };
  }, [refresh]);

  if (tasks === null) {
    return (
      <p className="text-sm text-gray-500 dark:text-gray-400">
        No activity available — the factory API is unreachable.
      </p>
    );
  }

  if (tasks.length === 0) {
    return (
      <p className="text-sm text-gray-500 dark:text-gray-400">
        No tasks yet — launch one from the cards above.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {tasks.map(t => (
        <div key={t.id} className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
          <div className="text-sm font-mono text-gray-500 w-12 shrink-0">{formatTime(t.completed_at ?? t.created_at)}</div>
          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium shrink-0 ${statusBadge[t.status] ?? statusBadge.idle}`}>
            {t.status}
          </span>
          <span className="text-sm text-gray-700 dark:text-gray-300 flex-1 truncate" title={t.id}>
            {describe(t)}
          </span>
        </div>
      ))}
    </div>
  );
}
