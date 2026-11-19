'use client';

import React, { useEffect, useState } from 'react';
import { Bot, Loader2, Send, Sparkles, X } from 'lucide-react';
import { AgentRecord, api } from '@/lib/api';
import { AgentDrawerUIMachine, useMachine } from '@/lib/stateMachine';

interface AgentDrawerProps {
  agent: AgentRecord | null;
  onClose: () => void;
  onLaunchRun: (prompt: string, agentSlug: string) => void;
}

interface Turn {
  role: 'user' | 'agent';
  text: string;
  meta?: string;
}

export function AgentDrawer({ agent, onClose, onLaunchRun }: AgentDrawerProps) {
  const machine = useMachine(AgentDrawerUIMachine);
  const [input, setInput] = useState('');
  const [turns, setTurns] = useState<Turn[]>([]);
  const [errMsg, setErrMsg] = useState('');

  useEffect(() => {
    if (agent) {
      machine.send('ready');
      setTurns([
        {
          role: 'agent',
          text: `${agent.description}\n\nAsk me to review concurrency invariants, edge cases, or package design — or launch a full sandbox run directly.`,
          meta: `${agent.model} · ${agent.authority}`,
        },
      ]);
      setErrMsg('');
    } else {
      machine.send('closed');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agent?.slug]);

  if (!agent) return null;

  const quickPrompts = [
    `What concurrency invariants should a Go ${agent.role === 'security' ? 'HMAC webhook verifier' : 'token-bucket rate limiter'} enforce?`,
    'Design table-driven -race stress tests with deterministic clock injection.',
  ];

  const handleConsult = async (promptText?: string) => {
    const q = (promptText ?? input).trim();
    if (!q || machine.state === 'consulting') return;
    setInput('');
    setErrMsg('');
    setTurns((prev) => [...prev, { role: 'user', text: q }]);
    machine.send('consulting');

    try {
      const res = await api.consultAgent(agent.slug, q);
      setTurns((prev) => [
        ...prev,
        {
          role: 'agent',
          text: res.reply,
          meta: `${res.provider}/${res.model} · ${res.tokens} tok · ${res.duration_ms}ms`,
        },
      ]);
      machine.send('ready');
    } catch (err: unknown) {
      setErrMsg(err instanceof Error ? err.message : 'Consultation failed');
      machine.send('error');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/25 backdrop-blur-[1px]">
      <div className="w-full max-w-lg h-full bg-[var(--bg-card)] border-l border-[var(--border-hairline)] flex flex-col shadow-2xl">
        <div className="p-5 border-b border-[var(--border-hairline)] flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <div className="w-9 h-9 rounded-xl bg-[var(--bg-subtle)] flex items-center justify-center">
              <Bot className="w-4 h-4 text-[var(--text-primary)]" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-base">{agent.name}</h3>
                <span className="rk-pill-healthy px-2 py-0.5 rounded-full text-[11px] font-medium">
                  ● {agent.status}
                </span>
              </div>
              <p className="text-xs font-mono text-[var(--text-muted)] mt-0.5">
                {agent.slug} · state: {machine.state}
              </p>
            </div>
          </div>
          <button
            onClick={() => {
              machine.send('closed');
              onClose();
            }}
            className="p-1.5 rounded-lg hover:bg-[var(--bg-subtle)] text-[var(--text-secondary)]"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5 space-y-3">
          {turns.map((t, idx) => (
            <div
              key={idx}
              className={`rounded-xl p-3.5 text-sm whitespace-pre-wrap ${
                t.role === 'user'
                  ? 'ml-8 bg-[var(--text-primary)] text-[var(--bg-elevated)]'
                  : 'mr-4 rk-card-elevated text-[var(--text-primary)]'
              }`}
            >
              <div>{t.text}</div>
              {t.meta && (
                <div className="mt-2 pt-2 border-t border-[var(--border-hairline)] text-[11px] font-mono text-[var(--text-muted)]">
                  {t.meta}
                </div>
              )}
            </div>
          ))}

          {machine.state === 'consulting' && (
            <div className="inline-flex items-center gap-2 text-xs text-[var(--text-secondary)] px-3 py-2 rk-card-elevated">
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
              Consulting {agent.name}...
            </div>
          )}

          {errMsg && (
            <div className="rk-pill-err rounded-xl p-3 text-xs font-mono">{errMsg}</div>
          )}

          <div className="pt-2 space-y-1.5">
            <div className="text-[11px] uppercase tracking-wider text-[var(--text-muted)] font-medium">
              Quick Consultations
            </div>
            {quickPrompts.map((qp) => (
              <button
                key={qp}
                onClick={() => handleConsult(qp)}
                className="w-full text-left text-xs px-3 py-2 rounded-lg border border-[var(--border-hairline)] bg-[var(--bg-elevated)] hover:bg-[var(--bg-subtle)] transition"
              >
                {qp}
              </button>
            ))}
          </div>
        </div>

        <div className="p-4 border-t border-[var(--border-hairline)] bg-[var(--bg-elevated)] space-y-2.5">
          <div className="flex gap-2">
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleConsult()}
              placeholder={`Ask ${agent.name}...`}
              className="flex-1 px-3.5 py-2 rounded-xl text-sm bg-[var(--bg-card)] border border-[var(--border-hairline)] focus:outline-none focus:border-[var(--border-strong)]"
            />
            <button
              onClick={() => handleConsult()}
              disabled={!input.trim() || machine.state === 'consulting'}
              className="rk-btn-primary px-3.5 py-2 inline-flex items-center gap-1.5 text-sm"
            >
              <Send className="w-3.5 h-3.5" />
            </button>
          </div>
          <button
            onClick={() => {
              const task =
                input.trim() ||
                'Implement a thread-safe token-bucket rate limiter with burst capacity and race tests';
              onClose();
              onLaunchRun(task, agent.slug);
            }}
            className="rk-btn-secondary w-full py-2 text-xs inline-flex items-center justify-center gap-1.5"
          >
            <Sparkles className="w-3.5 h-3.5" />
            Dispatch Full Sandbox Build with {agent.name}
          </button>
        </div>
      </div>
    </div>
  );
}
