'use client';

import Link from 'next/link';
import { ArrowLeft, Bot } from 'lucide-react';
import PipelinePlayground from './PipelinePlayground';

export default function AgentStudioClient({ id }: { id: string }) {
  return (
    <div className="min-h-screen ide-canvas ide-ink">
      <header className="h-14 px-8 border-b ide-border flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink transition"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            Control Plane
          </Link>
          <span className="ide-ink-muted">/</span>
          <span className="text-sm font-semibold capitalize ide-ink">Factory {id}</span>
        </div>
        <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-3 py-0.5 rounded-full text-xs font-medium">
          <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
          Healthy
        </span>
      </header>

      <main className="max-w-5xl mx-auto px-8 py-8 space-y-6">
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-2xl ide-surface border ide-border flex items-center justify-center">
            <Bot className="w-5 h-5 ide-ink" />
          </div>
          <div>
            <h1 className="text-2xl font-bold capitalize ide-ink">Factory {id} Agent</h1>
            <p className="text-sm ide-ink-muted font-mono">factory-{id.toLowerCase()}</p>
          </div>
        </div>

        <PipelinePlayground defaultTab="seven-layer" />
      </main>
    </div>
  );
}
