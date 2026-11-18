'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  CheckCircle2,
  Code2,
  Cpu,
  GitPullRequest,
  Layers,
  Loader,
  Play,
  PlugZap,
  RefreshCw,
  Send,
  ShieldAlert,
  Terminal,
  Webhook,
} from 'lucide-react';

type SubTab = 'seven-layer' | 'webhooks' | 'mcp';

const SEVEN_LAYERS = [
  { id: 'layer-1', num: 1, name: 'Factories-as-Code', desc: 'Parse YAML specification & intent' },
  { id: 'layer-2', num: 2, name: 'Data & Context', desc: 'Load ADRs, codebase context & constraints' },
  { id: 'layer-3', num: 3, name: 'Compute', desc: 'Allocate isolated execution sandbox' },
  { id: 'layer-4', num: 4, name: 'Inference', desc: 'Synthesize thread-safe Go code & tests' },
  { id: 'layer-5', num: 5, name: 'Improvement', desc: 'Score quality & audit safety invariants' },
  { id: 'layer-6', num: 6, name: 'Orchestration', desc: 'Coordinate workflow state transitions' },
  { id: 'layer-7', num: 7, name: 'Access', desc: 'Publish verified artifact & telemetry' },
];

const DEFAULT_LLD = `name: OrderLedgerService
description: Thread-safe financial order ledger with idempotency keys
requirements:
  - Create, read, update, and cancel orders atomically
  - Prevent duplicate order IDs and race conditions
  - Include unit tests covering happy path and error branches
`;

const MCP_PRESETS = [
  {
    label: 'Initialize MCP',
    payload: { jsonrpc: '2.0', id: 1, method: 'initialize', params: {} },
  },
  {
    label: 'List Tools',
    payload: { jsonrpc: '2.0', id: 2, method: 'tools/list', params: {} },
  },
  {
    label: 'Call: list_specs',
    payload: {
      jsonrpc: '2.0',
      id: 3,
      method: 'tools/call',
      params: { name: 'list_specs', arguments: {} },
    },
  },
  {
    label: 'Call: get_factory_status',
    payload: {
      jsonrpc: '2.0',
      id: 4,
      method: 'tools/call',
      params: { name: 'get_factory_status', arguments: {} },
    },
  },
  {
    label: 'Call: list_agents',
    payload: {
      jsonrpc: '2.0',
      id: 5,
      method: 'tools/call',
      params: { name: 'list_agents', arguments: {} },
    },
  },
  {
    label: 'List Resources',
    payload: { jsonrpc: '2.0', id: 6, method: 'resources/list', params: {} },
  },
];

export default function PipelinePlayground() {
  const [subTab, setSubTab] = useState<SubTab>('seven-layer');

  // 7-Layer SSE state
  const [lldYaml, setLldYaml] = useState(DEFAULT_LLD);
  const [streaming, setStreaming] = useState(false);
  const [progress, setProgress] = useState(0);
  const [currentLayer, setCurrentLayer] = useState('');
  const [streamMessage, setStreamMessage] = useState('');
  const [agentResult, setAgentResult] = useState<any>(null);
  const [streamError, setStreamError] = useState('');

  // GitHub Webhook Simulator state
  const [webhookSending, setWebhookSending] = useState(false);
  const [webhookResponse, setWebhookResponse] = useState<any>(null);
  const [queuedJobs, setQueuedJobs] = useState<any[]>([]);

  // MCP JSON-RPC Playground state
  const [mcpRequest, setMcpRequest] = useState(JSON.stringify(MCP_PRESETS[1].payload, null, 2));
  const [mcpResponse, setMcpResponse] = useState<string>('');
  const [mcpSending, setMcpSending] = useState(false);

  const loadWebhookJobs = useCallback(async () => {
    try {
      const res = await fetch('/api/jobs/list');
      if (res.ok) {
        const data = await res.json();
        setQueuedJobs(data.jobs || []);
      }
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    if (subTab === 'webhooks') {
      loadWebhookJobs();
    }
  }, [subTab, loadWebhookJobs]);

  const runSevenLayerStream = async () => {
    if (!lldYaml.trim()) return;
    setStreaming(true);
    setProgress(5);
    setCurrentLayer('start');
    setStreamMessage('Initializing 7-layer pipeline...');
    setAgentResult(null);
    setStreamError('');

    try {
      const res = await fetch('/api/agents/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agentId: 'CodeGen', lldYaml }),
      });
      if (!res.ok || !res.body) {
        throw new Error(`HTTP ${res.status}`);
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          try {
            const msg = JSON.parse(line.slice(6));
            if (msg.status === 'error') {
              setStreamError(msg.error || 'Agent execution failed');
              continue;
            }
            setCurrentLayer(msg.status || '');
            if (typeof msg.progress === 'number') setProgress(msg.progress);
            if (msg.message) setStreamMessage(msg.message);
            if (msg.result) setAgentResult(msg.result);
          } catch {
            // ignore partial chunk
          }
        }
      }
    } catch (err) {
      setStreamError(err instanceof Error ? err.message : 'Streaming failed');
    } finally {
      setStreaming(false);
    }
  };

  const sendSimulatedWebhook = async (eventType: string, payload: Record<string, any>) => {
    setWebhookSending(true);
    setWebhookResponse(null);
    try {
      const res = await fetch('/webhooks/github', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-GitHub-Event': eventType,
          'X-GitHub-Delivery': `sim-${Date.now()}`,
        },
        body: JSON.stringify(payload),
      });
      const text = await res.text();
      let parsed: any = text;
      try {
        parsed = JSON.parse(text);
      } catch {
        // plain text response
      }
      setWebhookResponse({ status: res.status, event: eventType, body: parsed });
      setTimeout(loadWebhookJobs, 300);
    } catch (err) {
      setWebhookResponse({ error: err instanceof Error ? err.message : 'Webhook failed' });
    } finally {
      setWebhookSending(false);
    }
  };

  const sendMcpRpc = async (overridePayload?: any) => {
    setMcpSending(true);
    try {
      const bodyStr = overridePayload ? JSON.stringify(overridePayload, null, 2) : mcpRequest;
      if (overridePayload) setMcpRequest(bodyStr);
      const res = await fetch('/api/mcp/factory', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: bodyStr,
      });
      const data = await res.json();
      setMcpResponse(JSON.stringify(data, null, 2));
    } catch (err) {
      setMcpResponse(JSON.stringify({ error: err instanceof Error ? err.message : 'RPC failed' }, null, 2));
    } finally {
      setMcpSending(false);
    }
  };

  const activeLayerNumber = (() => {
    if (currentLayer === 'done') return 8;
    const match = currentLayer.match(/layer-(\d+)/);
    return match ? Number(match[1]) : 0;
  })();

  return (
    <div className="space-y-6">
      {/* Sub-Navigation Pills */}
      <div className="flex flex-wrap items-center gap-2 bg-slate-900/80 border border-slate-800 p-1.5 rounded-2xl w-fit">
        {[
          { id: 'seven-layer' as SubTab, label: '7-Layer SSE Streaming Agent', icon: Layers },
          { id: 'webhooks' as SubTab, label: 'GitHub Webhook Simulator', icon: Webhook },
          { id: 'mcp' as SubTab, label: 'MCP JSON-RPC 2.0 Console', icon: PlugZap },
        ].map((item) => {
          const Icon = item.icon;
          const active = subTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setSubTab(item.id)}
              className={`inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition ${
                active
                  ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-950/50'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Icon className="w-4 h-4" />
              {item.label}
            </button>
          );
        })}
      </div>

      {/* 1. 7-Layer SSE Streaming Agent */}
      {subTab === 'seven-layer' && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <div className="lg:col-span-5 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-indigo-500/10 border border-indigo-500/20 text-indigo-400">
                <Cpu className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-slate-100">7-Layer Factory Stack Runner</h2>
                <p className="text-xs text-slate-400">Real-time Server-Sent Events (SSE) execution stream</p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
                Low-Level Design Specification (YAML)
              </label>
              <textarea
                value={lldYaml}
                onChange={(e) => setLldYaml(e.target.value)}
                rows={10}
                className="w-full p-3.5 bg-slate-950 border border-slate-800 rounded-xl font-mono text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>

            <button
              onClick={runSevenLayerStream}
              disabled={streaming}
              className="w-full py-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-slate-800 text-white font-semibold text-sm flex items-center justify-center gap-2 transition"
            >
              {streaming ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
              {streaming ? 'Streaming 7-Layer Pipeline...' : 'Stream 7-Layer Agent'}
            </button>
          </div>

          <div className="lg:col-span-7 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 space-y-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                Live Layer Progression
              </span>
              <span className="text-xs font-mono text-indigo-300">
                {streamMessage || 'Idle — ready to stream'} ({progress}%)
              </span>
            </div>

            <div className="w-full h-2 bg-slate-950 rounded-full overflow-hidden border border-slate-800">
              <div
                className="h-full bg-gradient-to-r from-indigo-500 to-emerald-400 transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              {SEVEN_LAYERS.map((layer) => {
                const done = activeLayerNumber > layer.num;
                const active = activeLayerNumber === layer.num;
                return (
                  <div
                    key={layer.id}
                    className={`p-3 rounded-xl border transition flex items-start gap-3 ${
                      done
                        ? 'bg-emerald-950/20 border-emerald-500/30'
                        : active
                          ? 'bg-indigo-950/40 border-indigo-500/60'
                          : 'bg-slate-950/70 border-slate-800/80'
                    }`}
                  >
                    <div
                      className={`w-6 h-6 rounded-full text-xs font-bold flex items-center justify-center flex-shrink-0 ${
                        done
                          ? 'bg-emerald-500 text-slate-950'
                          : active
                            ? 'bg-indigo-500 text-white animate-pulse'
                            : 'bg-slate-800 text-slate-400'
                      }`}
                    >
                      {done ? '✓' : layer.num}
                    </div>
                    <div className="min-w-0">
                      <div className="text-xs font-semibold text-slate-100">{layer.name}</div>
                      <div className="text-[11px] text-slate-400 truncate">{layer.desc}</div>
                    </div>
                  </div>
                );
              })}
            </div>

            {streamError && (
              <div className="p-3.5 rounded-xl bg-red-950/50 border border-red-800/60 text-xs text-red-200">
                {streamError}
              </div>
            )}

            {agentResult && (
              <div className="space-y-4 pt-2 border-t border-slate-800">
                <div className="grid grid-cols-3 gap-3">
                  <div className="p-3 rounded-xl bg-slate-950 border border-slate-800">
                    <div className="text-[11px] text-slate-400">Quality Score</div>
                    <div className="text-xl font-bold text-emerald-400 mt-0.5">{agentResult.score}/100</div>
                  </div>
                  <div className="p-3 rounded-xl bg-slate-950 border border-slate-800">
                    <div className="text-[11px] text-slate-400">Tests Synthesized</div>
                    <div className="text-xl font-bold text-indigo-400 mt-0.5">{agentResult.testsPassed}</div>
                  </div>
                  <div className="p-3 rounded-xl bg-slate-950 border border-slate-800">
                    <div className="text-[11px] text-slate-400">Safety Audit</div>
                    <div className="text-sm font-semibold text-emerald-300 mt-1 flex items-center gap-1.5">
                      {(agentResult.safetyViolations || []).length === 0 ? (
                        <>
                          <CheckCircle2 className="w-4 h-4" /> Clean
                        </>
                      ) : (
                        <>
                          <ShieldAlert className="w-4 h-4 text-amber-400" />
                          {agentResult.safetyViolations.length} warning(s)
                        </>
                      )}
                    </div>
                  </div>
                </div>

                <div className="rounded-xl border border-slate-800 bg-slate-950 overflow-hidden">
                  <div className="px-4 py-2 bg-slate-900 border-b border-slate-800 flex items-center justify-between text-xs text-slate-300">
                    <span className="flex items-center gap-2 font-medium">
                      <Code2 className="w-4 h-4 text-indigo-400" />
                      Synthesized Go Service &amp; Unit Tests
                    </span>
                    <span className="font-mono text-[11px] text-slate-500">{agentResult.id}</span>
                  </div>
                  <pre className="p-4 text-xs font-mono text-slate-300 overflow-x-auto max-h-72">
                    {agentResult.generatedCode}
                  </pre>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 2. GitHub Webhook Simulator */}
      {subTab === 'webhooks' && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <div className="lg:col-span-6 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
                <GitPullRequest className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-slate-100">GitHub Event Intake Simulator</h2>
                <p className="text-xs text-slate-400">
                  Dispatch realistic GitHub webhook events directly to <code className="text-slate-300">/webhooks/github</code>
                </p>
              </div>
            </div>

            <div className="space-y-3">
              <button
                disabled={webhookSending}
                onClick={() =>
                  sendSimulatedWebhook('issues', {
                    action: 'opened',
                    issue: {
                      id: 901,
                      number: 42,
                      title: 'Add rate limiting middleware to public API endpoints',
                      body: 'Protect endpoints with token-bucket rate limiting and unit tests.',
                      user: { login: 'octocat' },
                      labels: [{ name: 'enhancement' }],
                    },
                    repository: { name: 'agentic-software-factory-go', owner: { login: 'PratikDhanave' } },
                  })
                }
                className="w-full text-left p-4 rounded-xl bg-slate-950 hover:bg-slate-900 border border-slate-800 hover:border-indigo-500/40 transition flex items-center justify-between"
              >
                <div>
                  <div className="text-sm font-semibold text-slate-100">Simulate issues.opened (#42)</div>
                  <div className="text-xs text-slate-400 mt-0.5">
                    Triggers automated Issue Triage &amp; SQLite decision recording
                  </div>
                </div>
                <Send className="w-4 h-4 text-indigo-400" />
              </button>

              <button
                disabled={webhookSending}
                onClick={() =>
                  sendSimulatedWebhook('pull_request', {
                    action: 'opened',
                    number: 108,
                    pull_request: {
                      id: 5001,
                      number: 108,
                      title: 'feat(sqlite): enable WAL concurrency and busy_timeout',
                      body: 'Switches embedded store to WAL journal mode.',
                      draft: false,
                      user: { login: 'PratikDhanave' },
                      head: { ref: 'feat/sqlite-wal' },
                    },
                    repository: { name: 'agentic-software-factory-go', owner: { login: 'PratikDhanave' } },
                  })
                }
                className="w-full text-left p-4 rounded-xl bg-slate-950 hover:bg-slate-900 border border-slate-800 hover:border-emerald-500/40 transition flex items-center justify-between"
              >
                <div>
                  <div className="text-sm font-semibold text-slate-100">Simulate pull_request.opened (#108)</div>
                  <div className="text-xs text-slate-400 mt-0.5">
                    Enqueues automated PR Code Review &amp; quality scoring job
                  </div>
                </div>
                <Send className="w-4 h-4 text-emerald-400" />
              </button>
            </div>

            {webhookResponse && (
              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800">
                <div className="text-xs font-semibold text-emerald-400 mb-2">
                  Webhook Response (HTTP {webhookResponse.status})
                </div>
                <pre className="text-xs font-mono text-slate-300 overflow-x-auto">
                  {JSON.stringify(webhookResponse.body, null, 2)}
                </pre>
              </div>
            )}
          </div>

          <div className="lg:col-span-6 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300">
                Webhook Worker Queue ({queuedJobs.length})
              </h3>
              <button
                onClick={loadWebhookJobs}
                className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </button>
            </div>

            {queuedJobs.length === 0 ? (
              <p className="text-xs text-slate-500 py-12 text-center border border-dashed border-slate-800 rounded-xl">
                Click either webhook simulator button on the left to enqueue and process a live GitHub event.
              </p>
            ) : (
              <div className="space-y-2.5 max-h-96 overflow-y-auto">
                {queuedJobs.map((job: any) => (
                  <div key={job.id} className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-xs space-y-1.5">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-slate-200">{job.type}</span>
                      <span className="px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 font-mono">
                        {job.status}
                      </span>
                    </div>
                    <div className="font-mono text-[11px] text-slate-500">{job.id}</div>
                    {job.output && (
                      <pre className="text-[11px] font-mono text-slate-300 bg-slate-900 p-2 rounded overflow-x-auto">
                        {JSON.stringify(job.output, null, 2)}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* 3. MCP JSON-RPC 2.0 Console */}
      {subTab === 'mcp' && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <div className="lg:col-span-6 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-purple-500/10 border border-purple-500/20 text-purple-400">
                <Terminal className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-slate-100">Model Context Protocol (MCP) Console</h2>
                <p className="text-xs text-slate-400">
                  Invoke headless JSON-RPC 2.0 factory tools at <code className="text-slate-300">/api/mcp/factory</code>
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              {MCP_PRESETS.map((p) => (
                <button
                  key={p.label}
                  onClick={() => sendMcpRpc(p.payload)}
                  className="text-xs px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
                >
                  {p.label}
                </button>
              ))}
            </div>

            <textarea
              value={mcpRequest}
              onChange={(e) => setMcpRequest(e.target.value)}
              rows={9}
              className="w-full p-3.5 bg-slate-950 border border-slate-800 rounded-xl font-mono text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />

            <button
              onClick={() => sendMcpRpc()}
              disabled={mcpSending}
              className="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold flex items-center justify-center gap-2 transition"
            >
              {mcpSending ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
              Execute JSON-RPC Call
            </button>
          </div>

          <div className="lg:col-span-6 bg-slate-900/80 border border-slate-800 rounded-2xl p-6 flex flex-col">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-3">
              JSON-RPC 2.0 Response Payload
            </h3>
            <pre className="flex-1 p-4 rounded-xl bg-slate-950 border border-slate-800 font-mono text-xs text-emerald-300 overflow-auto min-h-[280px]">
              {mcpResponse || '// Select any preset or click "Execute JSON-RPC Call" to inspect the live MCP response.'}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
