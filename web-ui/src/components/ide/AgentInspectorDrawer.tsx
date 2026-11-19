'use client';

import React, { useEffect, useState } from 'react';
import { AgentCapabilitySummary, apiClient } from '@/lib/api';
import { launchMachine, useMachine } from '@/lib/stateMachine';
import { ArrowRight, Bot, Loader2, Send, Sparkles, X } from 'lucide-react';

const ROLE_STARTERS: Record<string, string[]> = {
  architect: [
    'Design a lock-free or sharded concurrency strategy for a Go rate limiter.',
    'How should we isolate SQLite WAL writers from long-running LLM calls?',
  ],
  developer: [
    'Show an idiomatic Go constructor with functional options and context cancellation.',
    'Structure a generic TTL + LRU cache in Go 1.23 with zero external dependencies.',
  ],
  security: [
    'Audit HMAC-SHA256 webhook signature checks for timing attacks and replay protection.',
    'List path-boundary checks needed before writing generated Go files to disk.',
  ],
  qa: [
    'Design a 100-goroutine race-detector stress test for a CircuitBreaker.',
    'Create a table-driven test template covering boundary and error cases.',
  ],
  default: [
    'Recommend idiomatic Go structure, explicit state machines, and race-safe testing.',
    'What invariants should this agent enforce before code promotion?',
  ],
};

interface MessageItem {
  role: 'user' | 'agent';
  text: string;
  meta?: string;
}

interface Props {
  agent: AgentCapabilitySummary | null;
  modelLabel: string;
  onClose: () => void;
  onLaunchRun: () => void;
}

export default function AgentInspectorDrawer({
  agent,
  modelLabel,
  onClose,
  onLaunchRun,
}: Props) {
  const [prompt, setPrompt] = useState('');
  const [messages, setMessages] = useState<MessageItem[]>([]);
  const consult = useMachine(launchMachine);

  useEffect(() => {
    if (!agent) return;
    const starters = ROLE_STARTERS[agent.role.toLowerCase()] || ROLE_STARTERS.default;
    setPrompt(starters[0]);
    setMessages([]);
  }, [agent]);

  useEffect(() => {
    if (!agent) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [agent, onClose]);

  if (!agent) return null;

  const starters = ROLE_STARTERS[agent.role.toLowerCase()] || ROLE_STARTERS.default;

  const handleSend = async () => {
    const q = prompt.trim();
    if (!q || !consult.send('loading')) return;
    setMessages((prev) => [...prev, { role: 'user', text: q }]);
    setPrompt('');

    try {
      const res = await apiClient.consultAgent(agent.role, q);
      setMessages((prev) => [
        ...prev,
        {
          role: 'agent',
          text: res.content,
          meta: `${res.provider}/${res.model} • ${res.duration_ms}ms`,
        },
      ]);
      consult.send('success');
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          role: 'agent',
          text: err instanceof Error ? err.message : 'Consultation request failed.',
          meta: 'error',
        },
      ]);
      consult.send('error');
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 bg-black/25 backdrop-blur-[1px] flex justify-end"
      onClick={onClose}
    >
      <aside
        className="w-full max-w-xl h-full ide-surface border-l ide-border shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Drawer Header */}
        <div className="px-6 py-5 border-b ide-border flex items-start justify-between gap-4">
          <div className="flex items-start gap-3.5">
            <div className="w-10 h-10 rounded-2xl ide-card border ide-border flex items-center justify-center shrink-0">
              <Bot className="w-5 h-5 ide-ink" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold ide-ink">
                  {agent.title.startsWith('Factory') ? agent.title : `Factory ${agent.title}`}
                </h2>
                <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                  Healthy
                </span>
              </div>
              <div className="text-xs font-mono ide-ink-muted mt-0.5">
                factory-{agent.role} • {modelLabel}
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/10 ide-ink-muted hover:ide-ink transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Agent Metadata & Capabilities */}
        <div className="px-6 py-4 border-b ide-border ide-canvas space-y-3">
          <p className="text-xs ide-ink-secondary leading-relaxed">{agent.description}</p>

          <div className="flex flex-wrap items-center gap-2 text-[11px]">
            <span className="px-2.5 py-0.5 rounded-md ide-card border ide-border font-mono ide-ink-secondary">
              authority: {agent.decision_authority}
            </span>
            <span className="px-2.5 py-0.5 rounded-md ide-card border ide-border font-mono ide-ink-secondary">
              mode: {agent.interaction_mode}
            </span>
            {(agent.capabilities || []).map((cap) => (
              <span
                key={cap}
                className="px-2.5 py-0.5 rounded-md ide-subtle ide-ink-secondary"
              >
                {cap}
              </span>
            ))}
          </div>

          <div className="pt-1 flex items-center justify-between">
            <span className="text-[11px] font-mono ide-ink-muted">
              machine: agent-decision (pending → approved → correct)
            </span>
            <button
              type="button"
              onClick={() => {
                onClose();
                onLaunchRun();
              }}
              className="inline-flex items-center gap-1 text-xs font-semibold ide-ink hover:underline"
            >
              Start Factory Run <ArrowRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Consultation Thread */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {messages.length === 0 ? (
            <div className="space-y-3">
              <div className="flex items-center gap-1.5 text-xs font-semibold ide-ink">
                <Sparkles className="w-3.5 h-3.5" />
                Ask {agent.title}
              </div>
              <p className="text-xs ide-ink-muted">
                Choose a suggested engineering question below or type your own prompt to consult this specialized agent directly through the Model Gateway:
              </p>
              <div className="space-y-2 pt-1">
                {starters.map((s, idx) => (
                  <button
                    key={idx}
                    type="button"
                    onClick={() => setPrompt(s)}
                    className="w-full text-left p-3.5 rounded-xl ide-card border ide-border hover:ide-border-strong text-xs ide-ink transition"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            messages.map((m, idx) => (
              <div
                key={idx}
                className={`p-4 rounded-2xl text-xs leading-relaxed ${
                  m.role === 'user'
                    ? 'bg-[#171715] text-white dark:bg-white dark:text-[#171715] ml-8'
                    : 'ide-card border ide-border ide-ink mr-4'
                }`}
              >
                <pre className="whitespace-pre-wrap font-sans">{m.text}</pre>
                {m.meta && (
                  <div className="mt-2 pt-2 border-t ide-border text-[10px] font-mono ide-ink-muted">
                    {m.meta}
                  </div>
                )}
              </div>
            ))
          )}
        </div>

        {/* Composer Footer */}
        <div className="p-4 border-t ide-border ide-card flex items-end gap-2">
          <textarea
            rows={2}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={(e) => {
              if ((e.metaKey || e.ctrlKey || !e.shiftKey) && e.key === 'Enter') {
                e.preventDefault();
                handleSend();
              }
            }}
            placeholder={`Ask Factory ${agent.role} an engineering question...`}
            className="flex-1 px-3.5 py-2 rounded-xl ide-canvas border ide-border text-xs ide-ink focus:outline-none focus:border-[var(--ide-ink)] resize-none"
          />
          <button
            type="button"
            disabled={consult.current === 'loading' || !prompt.trim()}
            onClick={handleSend}
            className="px-4 py-2.5 rounded-xl bg-[#171715] text-white dark:bg-white dark:text-black disabled:opacity-40 text-xs font-semibold inline-flex items-center gap-1.5 transition"
          >
            {consult.current === 'loading' ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Send className="w-3.5 h-3.5" />
            )}
            Send
          </button>
        </div>
      </aside>
    </div>
  );
}
