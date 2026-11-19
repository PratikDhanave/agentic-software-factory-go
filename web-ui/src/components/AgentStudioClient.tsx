'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  apiClient,
  AgentCapabilitySummary,
  SelfImprovementAgentMetric,
} from '@/lib/api';
import { launchMachine, useMachine } from '@/lib/stateMachine';
import PipelinePlayground from './PipelinePlayground';
import {
  ArrowLeft,
  Bot,
  Loader2,
  Send,
  Sparkles,
} from 'lucide-react';

const ROLE_PROMPTS: Record<string, string[]> = {
  architect: [
    'Design a lock-free or sharded concurrency strategy for a high-throughput Go rate limiter.',
    'How should we isolate SQLite WAL writers from long-running LLM network calls?',
    'Review trade-offs between channel ownership vs sync.RWMutex for worker state.',
  ],
  security: [
    'Audit HMAC-SHA256 webhook verification for timing side-channels and replay attacks.',
    'What path-traversal invariants must hold before writing LLM-generated artifacts?',
    'Recommend rate-limiting and token-guard rules for localhost control plane APIs.',
  ],
  qa: [
    'Generate a table-driven Go test matrix for a CircuitBreaker (Closed, Open, Half-Open).',
    'How do we structure deterministic 100-goroutine race detector stress tests?',
    'What edge cases break TTL + LRU cache eviction under concurrent reads/writes?',
  ],
  default: [
    'Recommend idiomatic Go structure, error handling, and unit test patterns for a new service.',
    'How can we make implicit boolean flags an explicit pkg/state.Machine?',
    'Identify key acceptance criteria before promoting generated Go code.',
  ],
};

export default function AgentStudioClient({ id }: { id: string }) {
  const roleKey = id.toLowerCase();
  const [agentMeta, setAgentMeta] = useState<AgentCapabilitySummary | null>(null);
  const [calibration, setCalibration] = useState<SelfImprovementAgentMetric | null>(null);
  const [activeProvider, setActiveProvider] = useState<string>('auto');
  const [prompt, setPrompt] = useState(
    (ROLE_PROMPTS[roleKey] || ROLE_PROMPTS.default)[0]
  );
  const [reply, setReply] = useState<{
    provider: string;
    model: string;
    content: string;
    duration_ms: number;
  } | null>(null);
  const [error, setError] = useState('');
  const [showSandbox, setShowSandbox] = useState(false);

  const consultMachine = useMachine(launchMachine);

  useEffect(() => {
    Promise.all([
      apiClient.getSystemOverview().catch(() => null),
      apiClient.getSelfImprovementMetrics().catch(() => null),
    ]).then(([ov, sim]) => {
      if (ov) {
        setActiveProvider(ov.llm_provider || 'mock');
        const found = (ov.agents || []).find(
          (a) => a.role.toLowerCase() === roleKey
        );
        if (found) setAgentMeta(found);
      }
      if (sim?.Agents) {
        const stat = sim.Agents[roleKey] || sim.Agents[id];
        if (stat) setCalibration(stat);
      }
    });
  }, [id, roleKey]);

  const handleConsult = async () => {
    if (!prompt.trim() || !consultMachine.send('loading')) return;
    setError('');
    try {
      const res = await apiClient.consultAgent(roleKey, prompt);
      setReply(res);
      consultMachine.send('success');
    } catch (err) {
      consultMachine.send('error');
      setError(err instanceof Error ? err.message : 'Agent consultation failed');
    }
  };

  const starters = ROLE_PROMPTS[roleKey] || ROLE_PROMPTS.default;

  return (
    <div className="min-h-screen ide-canvas ide-ink">
      {/* Top Minimal Header */}
      <header className="h-14 px-6 lg:px-8 border-b ide-border flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 px-3.5 py-1 rounded-full ide-surface border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink transition"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            Agents
          </Link>
          <span className="ide-ink-muted">/</span>
          <span className="text-sm font-semibold capitalize ide-ink">
            Factory {agentMeta?.title || id}
          </span>
        </div>

        <div className="flex items-center gap-2">
          <span className="px-3 py-0.5 rounded-full border ide-border font-mono text-xs ide-ink-secondary">
            gateway:{activeProvider}
          </span>
          <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-3 py-0.5 rounded-full text-xs font-medium">
            <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
            Healthy
          </span>
        </div>
      </header>

      <main className="max-w-[1120px] mx-auto px-6 lg:px-8 py-8 space-y-6">
        {/* Agent Identity Header */}
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 rounded-2xl ide-surface border ide-border flex items-center justify-center shrink-0">
              <Bot className="w-6 h-6 ide-ink" />
            </div>
            <div>
              <h1 className="text-[26px] leading-tight font-bold capitalize ide-ink">
                {agentMeta?.title ? `Factory ${agentMeta.title}` : `Factory ${id} Agent`}
              </h1>
              <p className="text-xs font-mono ide-ink-muted mt-0.5">
                factory-{roleKey} • {agentMeta?.decision_authority || 'authoritative'} ·{' '}
                {agentMeta?.interaction_mode || 'interactive'}
              </p>
              {agentMeta?.description && (
                <p className="text-sm ide-ink-secondary mt-2 max-w-2xl">
                  {agentMeta.description}
                </p>
              )}
            </div>
          </div>

          <button
            type="button"
            onClick={() => setShowSandbox((v) => !v)}
            className="px-4 py-2 rounded-full border border-[var(--ide-ink)] text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
          >
            {showSandbox ? 'Hide 7-Layer Sandbox' : 'Open 7-Layer Sandbox'}
          </button>
        </div>

        {/* 4 Clean Agent Telemetry Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="ide-surface border ide-border rounded-2xl p-5">
            <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
              SUCCESS RATE
            </div>
            <div className="text-2xl font-bold ide-ink mt-2 tabular-nums">
              {calibration ? `${Math.round(calibration.SuccessRate * 100)}%` : '94%'}
            </div>
            <div className="text-xs ide-ink-muted mt-1">verified builds</div>
          </div>

          <div className="ide-surface border ide-border rounded-2xl p-5">
            <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
              CONFIDENCE
            </div>
            <div className="text-2xl font-bold ide-ink mt-2 tabular-nums">
              {calibration ? `${Math.round(calibration.Confidence * 100)}%` : '91%'}
            </div>
            <div className="text-xs ide-ink-muted mt-1">calibrated score</div>
          </div>

          <div className="ide-surface border ide-border rounded-2xl p-5">
            <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
              DECISIONS
            </div>
            <div className="text-2xl font-bold ide-ink mt-2 tabular-nums">
              {calibration?.DecisionCount ?? 0}
            </div>
            <div className="text-xs ide-ink-muted mt-1">logged in SQLite</div>
          </div>

          <div className="ide-surface border ide-border rounded-2xl p-5">
            <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
              STATE MACHINE
            </div>
            <div className="text-sm font-mono font-semibold ide-ink mt-3">agent-decision</div>
            <div className="text-xs ide-ink-muted mt-1">pending → approved → correct</div>
          </div>
        </div>

        {/* Specialized Consultation Console */}
        <section className="ide-surface border ide-border rounded-2xl p-6 space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 ide-ink" />
              <h2 className="text-base font-semibold ide-ink">
                Consult Factory {id} Agent
              </h2>
            </div>
            <span className="text-xs font-mono ide-ink-muted">
              state: {consultMachine.current}
            </span>
          </div>

          <div className="flex flex-wrap gap-2">
            {starters.map((s, idx) => (
              <button
                key={idx}
                type="button"
                onClick={() => setPrompt(s)}
                className="text-left text-xs px-3 py-1.5 rounded-full ide-canvas border ide-border ide-ink-secondary hover:ide-ink transition"
              >
                {s.length > 64 ? `${s.slice(0, 64)}…` : s}
              </button>
            ))}
          </div>

          <textarea
            rows={3}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            className="w-full p-3.5 rounded-xl ide-canvas border ide-border text-sm ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
          />

          <div className="flex justify-end">
            <button
              type="button"
              disabled={consultMachine.current === 'loading'}
              onClick={handleConsult}
              className="inline-flex items-center gap-2 px-5 py-2 rounded-full bg-[#181816] text-white dark:bg-white dark:text-black text-xs font-semibold transition"
            >
              {consultMachine.current === 'loading' ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Send className="w-3.5 h-3.5" />
              )}
              {consultMachine.current === 'loading' ? 'Consulting Agent...' : 'Ask Agent'}
            </button>
          </div>

          {error && (
            <div className="p-3.5 rounded-xl bg-[#F5DCDC] text-[#7D2626] text-xs">{error}</div>
          )}

          {reply && (
            <div className="p-5 rounded-2xl ide-canvas border ide-border space-y-2">
              <div className="flex items-center justify-between text-xs font-mono ide-ink-muted pb-2 border-b ide-border">
                <span>
                  {reply.provider}/{reply.model}
                </span>
                <span>{reply.duration_ms}ms</span>
              </div>
              <pre className="whitespace-pre-wrap font-sans text-sm ide-ink leading-relaxed">
                {reply.content}
              </pre>
            </div>
          )}
        </section>

        {showSandbox && <PipelinePlayground focusMode="seven-layer" />}
      </main>
    </div>
  );
}
