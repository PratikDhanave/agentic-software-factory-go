'use client';

import Link from 'next/link';
import { ArrowLeft, Cpu } from 'lucide-react';
import PipelinePlayground from './PipelinePlayground';

export default function AgentStudioClient({ id }: { id: string }) {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        <div className="flex items-center justify-between border-b border-slate-800 pb-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
              <Cpu className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-slate-100 capitalize">Agent Runner: {id}</h1>
              <p className="text-xs text-slate-400">7-Layer Streaming Execution &amp; MCP Tooling</p>
            </div>
          </div>
          <Link
            href="/"
            className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-800 text-xs font-medium text-slate-200"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Control Plane
          </Link>
        </div>

        <PipelinePlayground />
      </div>
    </div>
  );
}
