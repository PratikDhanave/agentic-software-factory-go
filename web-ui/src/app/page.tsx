import Image from 'next/image'
import { Inter } from 'next/font/google'
import { LaunchPad } from '@/components/LaunchPad'
import { statusBadge } from '@/lib/statusBadge'

const inter = Inter({ subsets: ['latin'], weight: ['400', '500', '600'] })

export default function Home() {
  return (
    <main className="min-h-screen bg-white text-gray-900 antialiased">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <header className="mb-8 border-b border-gray-200 dark:border-gray-700">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tighter text-gray-900 dark:text-gray-100">
                Agentic Software Factory
              </h1>
              <p className="mt-2 text-lg text-gray-600 dark:text-gray-400">
                Production-grade Agentic Software Engineering patterns for Go
              </p>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500 dark:text-gray-400">v1.0.0</span>
              <button className="p-2 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996-.608 2.296-.07 2.572-1.065z"/>
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
              </button>
              <button className="p-2 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.36 0L3.34 16.5c-.77 1.333.192 3 1.732 3z"/>
                </svg>
              </button>
            </div>
          </div>
        </header>

        {/* Feature Launchpad */}
        <LaunchPad />

        {/* Factory Status */}
        <section className="mt-8">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">Factory Status</h2>
          <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
            <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
              <div className="flex items-center gap-4">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Factory Status:</span>
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                  <span className="w-2 h-2 rounded-full bg-green-500 mr-2"></span>
                  Running
                </span>
              </div>
              <div className="flex items-center gap-6 text-sm text-gray-500 dark:text-gray-400">
                <span>Workers: <span className="font-medium text-gray-900 dark:text-gray-100">3/4</span> active</span>
                <span>Queue: <span className="font-medium text-gray-900 dark:text-gray-100">2</span> tasks</span>
                <span>Completed: <span className="font-medium text-gray-900 dark:text-gray-100">47</span></span>
              </div>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
              <WorkerCard number={1} status="busy" currentTask="Code Review - PR #234" />
              <WorkerCard number={2} status="busy" currentTask="Feature Gen - spec-002" />
              <WorkerCard number={3} status="busy" currentTask="Bug Fix - auth.go" />
              <WorkerCard number={4} status="idle" currentTask="Waiting..." />
              <QueueCard count={2} />
            </div>
          </div>
        </section>

        {/* Recent Activity */}
        <section className="mt-8">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">Recent Activity</h2>
          <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
            <ActivityFeed />
          </div>
        </section>
      </div>
    </main>
  )
}

function WorkerCard({ number, status, currentTask }: { number: number; status: 'busy' | 'idle'; currentTask: string }) {
  return (
    <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-2">
        <span className="font-medium text-gray-900 dark:text-gray-100">Worker {number}</span>
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
          status === 'busy' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
        }`}>
          <span className={`w-1.5 h-1.5 rounded-full mr-1 ${number % 2 === 0 ? 'bg-green-500' : 'bg-green-400'}`}></span>
          {status === 'busy' ? 'Busy' : 'Idle'}
        </span>
      </div>
      <p className="text-sm text-gray-500 dark:text-gray-400 truncate">{currentTask}</p>
    </div>
  );
}

function QueueCard({ count }: { count: number }) {
  return (
    <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4 text-center">
      <div className="text-3xl font-bold text-gray-900 dark:text-gray-100">{count}</div>
      <p className="text-sm text-gray-500 dark:text-gray-400">Queued Tasks</p>
      <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">Next: spec-002 review</p>
    </div>
  );
}

function ActivityFeed() {
  const activities = [
    { time: '10:23', status: 'success', message: 'Code Review completed on PR #234', link: '#' },
    { time: '10:15', status: 'running', message: 'Feature "Auth API" generated → PR #235', link: '#' },
    { time: '09:42', status: 'success', message: 'Bug fix auto-merged for null pointer in auth.go', link: '#' },
    { time: '09:15', status: 'success', message: 'Prompt optimization completed (+12% success rate)', link: '#' },
  ];

  return (
    <div className="space-y-3">
      {activities.map((item, i) => (
        <div key={i} className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
          <div className="text-sm font-mono text-gray-500 w-10">{item.time}</div>
          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusBadge[item.status] ?? statusBadge.idle}`}>
            {item.status}
          </span>
          <span className="text-sm text-gray-700 dark:text-gray-300 flex-1">
            {item.link ? (
              <a href={item.link} className="hover:underline">{item.message}</a>
            ) : (
              item.message
            )}
          </span>
        </div>
      ))}
    </div>
  );
}