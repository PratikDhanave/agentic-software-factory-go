'use client';

import { useState } from 'react';

const colors = {
  indigo: { bg: 'bg-indigo-100 dark:bg-indigo-900/20', text: 'text-indigo-600 dark:text-indigo-400' },
  red: { bg: 'bg-red-100 dark:bg-red-900/20', text: 'text-red-600 dark:text-red-400' },
  green: { bg: 'bg-green-100 dark:bg-green-900/20', text: 'text-green-600 dark:text-green-400' },
  purple: { bg: 'bg-purple-100 dark:bg-purple-900/20', text: 'text-purple-600 dark:text-purple-400' },
  orange: { bg: 'bg-orange-100 dark:bg-orange-900/20', text: 'text-orange-600 dark:text-orange-400' },
  cyan: { bg: 'bg-cyan-100 dark:bg-cyan-900/20', text: 'text-cyan-600 dark:text-cyan-400' },
};

export function FeatureCard({
  icon, title, description, actionLabel, endpoint, color, specRequired = true
}: FeatureCardProps) {
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');

  const handleLaunch = async () => {
    if (!localStorage.getItem('activeSpec')) {
      alert('Please select a spec first');
      return;
    }
    setStatus('loading');
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spec_id: localStorage.getItem('activeSpec') || 'spec-001', priority: 1 }),
      });
      const data = await res.json();
      if (data.task_id) {
        setStatus('success');
        localStorage.setItem('activeTaskId', data.task_id);
        window.dispatchEvent(new CustomEvent('taskLaunched', { detail: { taskId: data.task_id } }));
      } else {
        setStatus('error');
      }
    } catch (e) {
      console.error('Launch error:', e);
      setStatus('error');
    }
  };

  const c = colors[color];

  return (
    <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 hover:shadow-xl transition-shadow duration-300 dark:hover:shadow-gray-800/50">
      <div className="flex items-start gap-3 mb-4">
        <div className={`w-8 h-8 rounded-lg ${c.bg} flex items-center justify-center`}>
          <span className={`${c.text} text-sm`}>🤖</span>
        </div>
        <div>
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{description}</p>
        </div>
      </div>
      <button
        onClick={handleLaunch}
        disabled={status === 'loading'}
        className="w-full px-4 py-2 rounded-lg font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
      >
        {status === 'loading' && <span className="animate-spin mr-2">•</span>}
        {status === 'success' && <span className="text-green-600">Launched!</span>}
        {status === 'error' && <span className="text-red-600">Failed - Try Again</span>}
        {status === 'idle' && <span>{actionLabel}</span>}
      </button>
    </div>
  );
}

export function LaunchPad() {
  const features = [
    {
      id: 'review',
      icon: '🤖',
      title: 'Code Review',
      description: 'Automated PR review with findings & fixes',
      actionLabel: 'Start Review',
      endpoint: '/api/factory/launch/review',
      color: 'indigo' as const,
    },
    {
      id: 'bugfix',
      icon: '🐛',
      title: 'Bug Fixer',
      description: 'Auto-detect & fix bugs from logs/Sentry',
      actionLabel: 'Enable Watch',
      endpoint: '/api/factory/launch/bugfix',
      color: 'red' as const,
    },
    {
      id: 'feature',
      icon: '🤖',
      title: 'Feature Gen',
      description: 'Spec → Code → Test → PR pipeline',
      actionLabel: 'Generate',
      endpoint: '/api/factory/launch/feature',
      color: 'green' as const,
    },
    {
      id: 'eval',
      icon: '🤖',
      title: 'Agent Tune',
      description: 'Benchmark & optimize agent prompts',
      actionLabel: 'Run Benchmark',
      endpoint: '/api/factory/launch/eval',
      color: 'purple' as const,
    },
    {
      id: 'refactor',
      icon: '🤖',
      title: 'Refactor',
      description: 'Clean architecture & optimize code',
      actionLabel: 'Clean Code',
      endpoint: '/api/factory/launch/refactor',
      color: 'orange' as const,
    },
    {
      id: 'migrate',
      icon: '🤖',
      title: 'Migrate',
      description: 'Upgrade Go version & dependencies',
      actionLabel: 'Migrate',
      endpoint: '/api/factory/launch/migrate',
      color: 'cyan' as const,
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Active Spec:</label>
        <select
          defaultValue="spec-001"
          onChange={(e) => localStorage.setItem('activeSpec', e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          <option value="spec-001">spec-001: Auth API</option>
          <option value="spec-002">spec-002: API Generation</option>
          <option value="spec-003">spec-003: Test Generation</option>
        </select>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map(feature => (
          <FeatureCard
            key={feature.id}
            icon={feature.icon}
            title={feature.title}
            description={feature.description}
            actionLabel={feature.actionLabel}
            endpoint={feature.endpoint}
            color={feature.color}
          />
        ))}
      </div>
    </div>
  );
}

export interface FeatureCardProps {
  icon: string;
  title: string;
  description: string;
  actionLabel: string;
  endpoint: string;
  color: 'indigo' | 'red' | 'green' | 'purple' | 'orange' | 'cyan';
  specRequired?: boolean;
}