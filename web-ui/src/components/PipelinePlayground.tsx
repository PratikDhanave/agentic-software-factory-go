'use client';

import { useCallback, useEffect, useState } from 'react';
import { API_BASE } from '@/lib/api';
import { sevenLayerClientMachine, SevenLayerClientState, statusBadge, useMachine } from '@/lib/stateMachine';
import {
  CheckCircle2,
  Loader2,
  Play,
  RefreshCw,
  Send,
  Server,
  Webhook,
  Zap,
} from 'lucide-react';

export type PlaygroundMode = 'all' | 'mcp' | 'webhooks' | 'seven-layer';

const SEVEN_LAYERS = [
  { id: 'layer-1', num: 1, name: 'Factories-as-Code', desc: 'Parse YAML specification & intent' },
  { id: 'layer-2', num: 2, name: 'Data & Context', desc: 'Load ADRs, codebase context & constraints' },
  { id: 'layer-3', num: 3, name: 'Compute', desc: 'Allocate isolated Go module sandbox' },
  { id: 'layer-4', num: 4, name: 'Inference', desc: 'Synthesize thread-safe Go code & tests' },
  { id: 'layer-5', num: 5, name: 'Improvement', desc: 'Run go build, go test, go vet & Judge' },
  { id: 'layer-6', num: 6, name: 'Orchestration', desc: 'Self-heal & validate state transitions' },
  { id: 'layer-7', num: 7, name: 'Access', desc: 'Package verified artifact & scorecard' },
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
    label: 'tools/list',
    payload: { jsonrpc: '2.0', id: 1, method: 'tools/list', params: {} },
  },
  {
    label: 'list_state_machines (AX)',
    payload: {
      jsonrpc: '2.0',
      id: 2,
      method: 'tools/call',
      params: { name: 'list_state_machines', arguments: {} },
    },
  },
  {
    label: 'inspect_state_machine (AX)',
    payload: {
      jsonrpc: '2.0',
      id: 3,
      method: 'tools/call',
      params: {
        name: 'inspect_state_machine',
        arguments: {
          machine: 'factory-assembly-line',
          state: 'agent_governance',
          next_state: 'workspace_init',
        },
      },
    },
  },
  {
    label: 'list_specs',
    payload: {
      jsonrpc: '2.0',
      id: 4,
      method: 'tools/call',
      params: { name: 'list_specs', arguments: {} },
    },
  },
  {
    label: 'get_factory_status',
    payload: {
      jsonrpc: '2.0',
      id: 5,
      method: 'tools/call',
      params: { name: 'get_factory_status', arguments: {} },
    },
  },
  {
    label: 'resources/list',
    payload: { jsonrpc: '2.0', id: 6, method: 'resources/list', params: {} },
  },
];

export default function PipelinePlayground({ defaultTab = 'seven-layer', focusMode = 'all' }: { defaultTab?: 'seven-layer' | 'webhooks' | 'mcp'; focusMode?: PlaygroundMode }) {
  const [subTab, setSubTab] = useState<'seven-layer' | 'webhooks' | 'mcp'>(
    focusMode === 'mcp' ? 'mcp' : focusMode === 'webhooks' ? 'webhooks' : defaultTab
  );

  // Explicit 7-layer SSE state machine
  const streamMachine = useMachine(sevenLayerClientMachine);
  const [lldYaml, setLldYaml] = useState(DEFAULT_LLD);
  const [progress, setProgress] = useState(0);
  const [streamMessage, setStreamMessage] = useState('');
  const [agentResult, setAgentResult] = useState<any>(null);
  const [streamError, setStreamError] = useState('');

  // Webhook simulator
  const [webhookSending, setWebhookSending] = useState(false);
  const [webhookResponse, setWebhookResponse] = useState<any>(null);
  const [queuedJobs, setQueuedJobs] = useState<any[]>([]);

  // MCP JSON-RPC console
  const [mcpRequest, setMcpRequest] = useState(JSON.stringify(MCP_PRESETS[1].payload, null, 2));
  const [mcpResponse, setMcpResponse] = useState<string>('');
  const [mcpSending, setMcpSending] = useState(false);

  useEffect(() => {
    if (focusMode === 'mcp') setSubTab('mcp');
    else if (focusMode === 'webhooks') setSubTab('webhooks');
    else if (focusMode === 'seven-layer') setSubTab('seven-layer');
  }, [focusMode]);

  const loadWebhookJobs = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/jobs/list`);
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

  const isStreaming =
    streamMachine.current !== 'idle' &&
    streamMachine.current !== 'done' &&
    streamMachine.current !== 'error';

  const runSevenLayerStream = async () => {
    if (!lldYaml.trim() || isStreaming) return;
    streamMachine.reset();
    streamMachine.send('start');
    setProgress(5);
    setStreamMessage('Initializing 7-layer streaming pipeline...');
    setAgentResult(null);
    setStreamError('');

    try {
      const res = await fetch(`${API_BASE}/api/agents/run`, {
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
              streamMachine.send('error');
              setStreamError(msg.error || 'Agent execution failed');
              continue;
            }
            if (msg.status) {
              streamMachine.send(msg.status as SevenLayerClientState);
            }
            if (typeof msg.progress === 'number') setProgress(msg.progress);
            if (msg.message) setStreamMessage(msg.message);
            if (msg.result) setAgentResult(msg.result);
          } catch {
            // ignore partial chunk
          }
        }
      }
    } catch (err) {
      streamMachine.send('error');
      setStreamError(err instanceof Error ? err.message : 'Streaming failed');
    }
  };

  const sendSimulatedWebhook = async (eventType: string, payload: Record<string, any>) => {
    setWebhookSending(true);
    setWebhookResponse(null);
    try {
      const res = await fetch(`${API_BASE}/webhooks/github`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-GitHub-Event': eventType,
        },
        body: JSON.stringify(payload),
      });
      const body = await res.json().catch(() => ({}));
      setWebhookResponse({ status: res.status, body });
      setTimeout(loadWebhookJobs, 300);
    } catch (err) {
      setWebhookResponse({
        status: 0,
        body: { error: err instanceof Error ? err.message : 'Webhook request failed' },
      });
    } finally {
      setWebhookSending(false);
    }
  };

  const sendMcpRpc = async () => {
    setMcpSending(true);
    setMcpResponse('');
    try {
      const parsed = JSON.parse(mcpRequest);
      const res = await fetch(`${API_BASE}/api/mcp/factory`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(parsed),
      });
      const data = await res.json();
      setMcpResponse(JSON.stringify(data, null, 2));
    } catch (err) {
      setMcpResponse(
        JSON.stringify({ error: err instanceof Error ? err.message : 'Invalid JSON-RPC request' }, null, 2)
      );
    } finally {
      setMcpSending(false);
    }
  };

  return (
    <div className="space-y-6">
      {focusMode === 'all' && (
        <div className="flex flex-wrap items-center gap-2">
          {[
            { id: 'mcp' as const, label: 'MCP Server (JSON-RPC 2.0)', icon: Server },
            { id: 'webhooks' as const, label: 'GitHub Webhook Triggers', icon: Webhook },
            { id: 'seven-layer' as const, label: '7-Layer SSE Stream', icon: Zap },
          ].map((t) => {
            const Icon = t.icon;
            const active = subTab === t.id;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => setSubTab(t.id)}
                className={`inline-flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium transition ${
                  active
                    ? 'kora-nav-active'
                    : 'ide-surface border ide-border ide-ink-secondary hover:ide-ink'
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                {t.label}
              </button>
            );
          })}
        </div>
      )}

      {subTab === 'mcp' && (
        <div className="ide-surface border ide-border rounded-2xl p-6 space-y-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h3 className="text-base font-semibold ide-ink">
                Factory MCP Server (`/api/mcp/factory`)
              </h3>
              <p className="text-xs ide-ink-muted mt-0.5">
                Exposes specs, tasks, verification, and explicit state machines (`list_state_machines`, `inspect_state_machine`) over JSON-RPC 2.0 for AI agents.
              </p>
            </div>
            <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
              Mounted &amp; Healthy
            </span>
          </div>

          <div className="flex flex-wrap gap-1.5">
            {MCP_PRESETS.map((p) => (
              <button
                key={p.label}
                type="button"
                onClick={() => setMcpRequest(JSON.stringify(p.payload, null, 2))}
                className="px-3 py-1.5 rounded-full ide-canvas border ide-border text-xs font-mono ide-ink-secondary hover:ide-ink transition"
              >
                {p.label}
              </button>
            ))}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-[11px] uppercase tracking-wider font-medium ide-ink-muted">
                  JSON-RPC Request
                </span>
                <button
                  type="button"
                  disabled={mcpSending}
                  onClick={sendMcpRpc}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1 rounded-full border border-[var(--ide-ink)] text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
                >
                  {mcpSending ? <Loader2 className="w-3 h-3 animate-spin" /> : <Send className="w-3 h-3" />}
                  Invoke MCP
                </button>
              </div>
              <textarea
                rows={12}
                value={mcpRequest}
                onChange={(e) => setMcpRequest(e.target.value)}
                className="w-full p-3.5 rounded-xl ide-canvas border ide-border font-mono text-xs ide-ink focus:outline-none"
              />
            </div>

            <div>
              <div className="text-[11px] uppercase tracking-wider font-medium ide-ink-muted mb-1.5">
                JSON-RPC Response
              </div>
              <pre className="p-3.5 rounded-xl bg-[#181816] text-[#DCE9DF] font-mono text-xs overflow-auto h-[274px]">
                {mcpResponse || '// Click "Invoke MCP" to inspect the live JSON-RPC 2.0 response'}
              </pre>
            </div>
          </div>
        </div>
      )}

      {subTab === 'webhooks' && (
        <div className="ide-surface border ide-border rounded-2xl p-6 space-y-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h3 className="text-base font-semibold ide-ink">
                Event Triggers &amp; GitHub Webhooks (`/webhooks/github`)
              </h3>
              <p className="text-xs ide-ink-muted mt-0.5">
                Simulate inbound GitHub issue, pull_request, and `@factory` comment triggers feeding the explicit `queue-job` state machine.
              </p>
            </div>
            <button
              type="button"
              onClick={loadWebhookJobs}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full border ide-border text-xs ide-ink-secondary hover:ide-ink"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              Refresh Queue
            </button>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <button
              type="button"
              disabled={webhookSending}
              onClick={() =>
                sendSimulatedWebhook('issues', {
                  action: 'opened',
                  issue: {
                    number: 104,
                    title: 'Add concurrent token-bucket rate limiter',
                    body: 'Implement thread-safe limiter with unit tests.',
                    labels: [{ name: 'factory' }],
                  },
                  repository: { full_name: 'org/agentic-software-factory-go' },
                })
              }
              className="text-left p-4 rounded-xl ide-canvas border ide-border hover:ide-border-strong transition"
            >
              <div className="text-xs font-semibold ide-ink">Simulate `issues.opened`</div>
              <p className="text-[11px] ide-ink-muted mt-1">
                Enqueues automated issue triage &amp; code synthesis job.
              </p>
            </button>

            <button
              type="button"
              disabled={webhookSending}
              onClick={() =>
                sendSimulatedWebhook('pull_request', {
                  action: 'opened',
                  number: 42,
                  pull_request: {
                    number: 42,
                    title: 'Refactor SQLite WAL persistence',
                    body: 'Consolidate repositories onto single WAL handle.',
                  },
                  repository: { full_name: 'org/agentic-software-factory-go' },
                })
              }
              className="text-left p-4 rounded-xl ide-canvas border ide-border hover:ide-border-strong transition"
            >
              <div className="text-xs font-semibold ide-ink">Simulate `pull_request.opened`</div>
              <p className="text-[11px] ide-ink-muted mt-1">
                Enqueues automated multi-agent PR review job.
              </p>
            </button>

            <button
              type="button"
              disabled={webhookSending}
              onClick={() =>
                sendSimulatedWebhook('issue_comment', {
                  action: 'created',
                  issue: { number: 104, title: 'Rate limiter' },
                  comment: { body: '@factory please add race detector benchmarks' },
                  repository: { full_name: 'org/agentic-software-factory-go' },
                })
              }
              className="text-left p-4 rounded-xl ide-canvas border ide-border hover:ide-border-strong transition"
            >
              <div className="text-xs font-semibold ide-ink">Simulate `@factory` Comment</div>
              <p className="text-[11px] ide-ink-muted mt-1">
                Triggers command dispatch from issue comment mention.
              </p>
            </button>
          </div>

          {webhookResponse && (
            <div className="p-3.5 rounded-xl ide-canvas border ide-border text-xs font-mono ide-ink">
              HTTP {webhookResponse.status}: {JSON.stringify(webhookResponse.body)}
            </div>
          )}

          <div className="border ide-border rounded-xl overflow-hidden">
            <div className="px-4 py-2.5 ide-subtle border-b ide-border flex items-center justify-between text-xs">
              <span className="font-semibold ide-ink">Queued Trigger Jobs ({queuedJobs.length})</span>
              <span className="font-mono ide-ink-muted">machine: queue-job</span>
            </div>
            {queuedJobs.length === 0 ? (
              <div className="p-8 text-center text-xs ide-ink-muted">
                Click any trigger card above to dispatch a simulated GitHub event.
              </div>
            ) : (
              <div className="divide-y ide-border">
                {queuedJobs.slice(0, 8).map((j) => (
                  <div key={j.id} className="px-4 py-3 flex items-center justify-between text-xs">
                    <div>
                      <span className="font-semibold ide-ink">{j.type}</span>
                      <div className="font-mono text-[11px] ide-ink-muted">{j.id}</div>
                    </div>
                    <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusBadge[j.status] || ''}`}>
                      {j.status}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {subTab === 'seven-layer' && (
        <div className="ide-surface border ide-border rounded-2xl p-6 space-y-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold ide-ink">
                7-Layer Streaming Execution (`seven-layer-stream`)
              </h3>
              <p className="text-xs ide-ink-muted mt-0.5">
                Streams each architectural layer over SSE while validating transitions via `SevenLayerMachine`.
              </p>
            </div>
            <span className={`px-3 py-0.5 rounded-full text-xs font-mono ${statusBadge[streamMachine.current] || statusBadge.idle}`}>
              state: {streamMachine.current} ({progress}%)
            </span>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
            <div className="lg:col-span-5 space-y-3">
              <textarea
                rows={7}
                value={lldYaml}
                onChange={(e) => setLldYaml(e.target.value)}
                className="w-full p-3.5 rounded-xl ide-canvas border ide-border font-mono text-xs ide-ink focus:outline-none"
              />
              <button
                type="button"
                disabled={isStreaming}
                onClick={runSevenLayerStream}
                className="w-full py-2.5 px-4 rounded-xl bg-[#181816] hover:opacity-90 dark:bg-white dark:text-black text-white text-xs font-semibold flex items-center justify-center gap-2 transition"
              >
                {isStreaming ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5 fill-current" />}
                {isStreaming ? streamMessage || 'Streaming...' : 'Run 7-Layer Stream'}
              </button>
              {streamError && (
                <div className="p-3 rounded-lg bg-[#F5DCDC] text-[#7D2626] text-xs">{streamError}</div>
              )}
            </div>

            <div className="lg:col-span-7 space-y-2">
              {SEVEN_LAYERS.map((layer) => {
                const currentNum = streamMachine.current.startsWith('layer-')
                  ? Number(streamMachine.current.replace('layer-', ''))
                  : streamMachine.current === 'done'
                    ? 8
                    : 0;
                const done = currentNum > layer.num;
                const active = currentNum === layer.num;
                return (
                  <div
                    key={layer.id}
                    className={`px-4 py-2.5 rounded-xl border flex items-center justify-between text-xs ${
                      done
                        ? 'bg-[#DCE9DF]/40 border-[#C3DAC8]'
                        : active
                          ? 'bg-[#DCE6F5]/60 border-[#9AB8E3]'
                          : 'ide-canvas ide-border'
                    }`}
                  >
                    <div>
                      <span className="font-semibold ide-ink">
                        L{layer.num} · {layer.name}
                      </span>
                      <span className="ide-ink-muted ml-2">{layer.desc}</span>
                    </div>
                    {done && <CheckCircle2 className="w-4 h-4 text-[#285936]" />}
                    {active && <Loader2 className="w-4 h-4 animate-spin text-[#1E4785]" />}
                  </div>
                );
              })}
            </div>
          </div>

          {agentResult?.generatedCode && (
            <pre className="p-4 rounded-xl bg-[#181816] text-[#F3F2ED] font-mono text-xs overflow-x-auto max-h-72">
              {agentResult.generatedCode}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
