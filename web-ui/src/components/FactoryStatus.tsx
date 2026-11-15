'use client';

import { useCallback, useEffect, useState } from 'react';
import { MachineDef, useMachine } from '@/lib/stateMachine';
import { statusBadge } from '@/lib/statusBadge';

interface WorkerStatus {
  id: string;
  status: string;
  task_type?: string;
  task_spec?: string;
}

interface FactoryStatusData {
  workers_total: number;
  workers_busy: number;
  workers_idle: number;
  workers: WorkerStatus[];
  tasks_pending: number;
  tasks_completed: number;
  tasks_failed: number;
  max_workers: number;
}

/** Connection lifecycle of the dashboard's link to the factory API. */
type ConnState = 'connecting' | 'live' | 'unreachable';

const connectionMachine: MachineDef<ConnState> = {
  name: 'factory-connection',
  initial: 'connecting',
  transitions: {
    connecting: ['live', 'unreachable'],
    live: ['unreachable'],
    unreachable: ['live'],
  },
};

const POLL_MS = 3000;

export function FactoryStatus() {
  const { current: conn, send } = useMachine(connectionMachine);
  const [data, setData] = useState<FactoryStatusData | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await fetch('/api/factory/status');
      if (!res.ok) throw new Error(`${res.status}`);
      setData(await res.json());
      if (conn !== 'live') send('live');
    } catch {
      if (conn !== 'unreachable') send('unreachable');
    }
  }, [conn, send]);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, POLL_MS);
    // Refresh immediately when a LaunchPad card submits a task.
    window.addEventListener('taskLaunched', refresh);
    return () => {
      clearInterval(interval);
      window.removeEventListener('taskLaunched', refresh);
    };
  }, [refresh]);

  const pill =
    conn === 'live'
      ? { label: 'Running', cls: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400', dot: 'bg-green-500' }
      : conn === 'connecting'
        ? { label: 'Connecting…', cls: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300', dot: 'bg-gray-400' }
        : { label: 'Unreachable', cls: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400', dot: 'bg-red-500' };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <div className="flex items-center gap-4">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Factory Status:</span>
          <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${pill.cls}`}>
            <span className={`w-2 h-2 rounded-full mr-2 ${pill.dot}`}></span>
            {pill.label}
          </span>
        </div>
        {data && (
          <div className="flex items-center gap-6 text-sm text-gray-500 dark:text-gray-400">
            <span>Workers: <span className="font-medium text-gray-900 dark:text-gray-100">{data.workers_busy}/{data.workers_total}</span> busy</span>
            <span>Queue: <span className="font-medium text-gray-900 dark:text-gray-100">{data.tasks_pending}</span> tasks</span>
            <span>Completed: <span className="font-medium text-gray-900 dark:text-gray-100">{data.tasks_completed}</span></span>
            {data.tasks_failed > 0 && (
              <span>Failed: <span className="font-medium text-red-600 dark:text-red-400">{data.tasks_failed}</span></span>
            )}
          </div>
        )}
      </div>

      {conn === 'unreachable' && (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Cannot reach the factory API. Start it with <code className="font-mono bg-gray-100 dark:bg-gray-700 px-1 rounded">go run ./cmd/asf-web</code> and it will reconnect automatically.
        </p>
      )}

      {data && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
          {data.workers.map(w => (
            <WorkerCard key={w.id} worker={w} />
          ))}
          <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4 text-center">
            <div className="text-3xl font-bold text-gray-900 dark:text-gray-100">{data.tasks_pending}</div>
            <p className="text-sm text-gray-500 dark:text-gray-400">Queued Tasks</p>
          </div>
        </div>
      )}
    </div>
  );
}

function WorkerCard({ worker }: { worker: WorkerStatus }) {
  const busy = worker.status === 'busy';
  const label = worker.id.replace('worker-', 'Worker ');
  const task = busy && worker.task_type
    ? `${worker.task_type}${worker.task_spec ? ` — ${worker.task_spec}` : ''}`
    : 'Waiting…';

  return (
    <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-2">
        <span className="font-medium text-gray-900 dark:text-gray-100">{label}</span>
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusBadge[worker.status] ?? statusBadge.idle}`}>
          <span className={`w-1.5 h-1.5 rounded-full mr-1 ${busy ? 'bg-green-500' : 'bg-gray-400'}`}></span>
          {worker.status}
        </span>
      </div>
      <p className="text-sm text-gray-500 dark:text-gray-400 truncate">{task}</p>
    </div>
  );
}
