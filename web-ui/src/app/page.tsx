'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiClient, FactoryPhase, FactoryTask, SystemOverview } from '@/lib/api';
import WorkflowGenerator from '@/components/WorkflowGenerator';
import ApprovalsPanel from '@/components/ApprovalsPanel';
import WorkspaceRunner from '@/components/WorkspaceRunner';
import PipelinePlayground from '@/components/PipelinePlayground';
import SelfImprovementDashboard from '@/components/SelfImprovementDashboard';
import FactoryAssemblyLine from '@/components/FactoryAssemblyLine';
import {
  Activity,
  ArrowRight,
  Bot,
  Brain,
  CheckCircle2,
  Cpu,
  Database,
  FolderGit2,
  GitBranch,
  Layers,
  PlugZap,
  RefreshCw,
  Rocket,
  ShieldCheck,
  Sparkles,
  Terminal,
} from 'lucide-react';

type TabKey = 'assembly' | 'overview' | 'studio' | 'workspaces' | 'pipelines' | 'approvals' | 'learning';

export default function ControlPlanePage() {
  const [activeTab, setActiveTab] = useState<TabKey>('assembly');
  const [overview, setOverview] = useState<SystemOverview | null>(null);
  const [recentTasks, setRecentTasks] = useState<FactoryTask[]>([]);
  const [recentSessions, setRecentSessions] = useState<any[]>([]);
  const [connected, setConnected] = useState<boolean | null>(null);
  const [switchingPhase, setSwitchingPhase] = useState(false);

  const refreshTelemetry = useCallback(async () => {
    try {
      const [ov, tks, sess] = await Promise.all([
        apiClient.getSystemOverview(),
        apiClient.getTasks().catch(() => ({ tasks: [] })),
        apiClient.getSessions().catch(() => ({ sessions: [] })),
      ]);
      setOverview(ov);
      setRecentTasks((tks.tasks || []).slice(0, 6));
      setRecentSessions((sess.sessions || []).slice(0, 5));
      setConnected(true);
    } catch {
      setConnected(false);
    }
  }, []);

  useEffect(() => {
    refreshTelemetry();
    const interval = setInterval(refreshTelemetry, 4000);
    return () => clearInterval(interval);
  }, [refreshTelemetry]);

  const handlePhaseChange = async (phase: FactoryPhase) => {
    setSwitchingPhase(true);
    try {
      await apiClient.setSystemPhase(phase);
      await refreshTelemetry();
    } finally {
      setSwitchingPhase(false);
    }
  };

  const pendingCount = overview?.pending_approvals || 0;
  const activePhase: FactoryPhase = overview?.phase || 'walk';

  const navItems: Array<{ id: TabKey; label: string; icon: any; badge?: number }> = [
    { id: 'assembly', label: 'Factory Assembly Line', icon: Rocket },
    { id: 'overview', label: 'Command Center', icon: Layers },
    { id: 'studio', label: 'Multi-Agent Studio', icon: GitBranch },
    { id: 'workspaces', label: 'Specs & Workspaces', icon: FolderGit2 },
    { id: 'pipelines', label: '7-Layer, Webhooks & MCP', icon: PlugZap },
    { id: 'approvals', label: 'Governance Gate', icon: ShieldCheck, badge: pendingCount },
    { id: 'learning', label: 'Self-Improvement', icon: Brain },
  ];

  const pipelineStages = [
    { step: '01', name: 'Intake', metric: `${overview?.specs_count ?? 0} specs`, tab: 'pipelines' as TabKey },
    { step: '02', name: '10-Agent DAG', metric: `${overview?.database?.total_sessions ?? 0} sessions`, tab: 'studio' as TabKey },
    { step: '03', name: 'HITL Gate', metric: `${pendingCount} pending`, tab: 'approvals' as TabKey },
    { step: '04', name: 'Workspace & Judge', metric: `${overview?.factory?.tasks_completed ?? 0} verified`, tab: 'workspaces' as TabKey },
    { step: '05', name: 'SQLite Learning', metric: `${overview?.database?.discovered_patterns ?? 0} patterns`, tab: 'learning' as TabKey },
  ];

  return (
    <div className="min-h-screen bg-slate-950 bg-factory-grid text-slate-100 flex flex-col">
      {/* Top Sticky Control Plane Header */}
      <header className="sticky top-0 z-30 border-b border-slate-800/90 bg-slate-950/90 backdrop-blur-md">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 via-purple-500 to-emerald-500 flex items-center justify-center shadow-lg shadow-indigo-950/60">
              <Cpu className="w-5 h-5 text-white" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold tracking-tight text-slate-100">
                  Agentic Software Factory
                </span>
                <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300 border border-indigo-500/30">
                  Control Plane v2.0
                </span>
              </div>
              <p className="text-[11px] text-slate-400 hidden sm:block">
                Multi-Provider LLM Gateway • Isolated Workspace Verification • SQLite WAL
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <div className="hidden lg:flex items-center gap-2.5 px-3.5 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
              <span
                className={`w-2 h-2 rounded-full ${
                  connected === true
                    ? 'bg-emerald-400 animate-pulse'
                    : connected === false
                      ? 'bg-red-400'
                      : 'bg-slate-500'
                }`}
              />
              <span className="text-slate-300 font-medium">
                {connected === true ? 'Live' : connected === false ? 'Offline' : 'Connecting'}
              </span>
              {overview?.llm_provider && (
                <>
                  <span className="text-slate-700">|</span>
                  <span className="font-mono text-indigo-300">provider:{overview.llm_provider}</span>
                </>
              )}
              <span className="text-slate-700">|</span>
              <span className="font-mono text-emerald-300">sqlite3:wal</span>
            </div>

            {/* Crawl / Walk / Run Maturity Switcher */}
            <div className="flex items-center bg-slate-900 p-1 rounded-xl border border-slate-800 text-xs">
              {(['crawl', 'walk', 'run'] as FactoryPhase[]).map((p) => (
                <button
                  key={p}
                  disabled={switchingPhase}
                  onClick={() => handlePhaseChange(p)}
                  className={`px-2.5 py-1 rounded-lg font-semibold capitalize transition ${
                    activePhase === p
                      ? 'bg-indigo-600 text-white shadow'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                  title={
                    p === 'crawl'
                      ? 'Crawl: Require human approval on every agent decision'
                      : p === 'walk'
                        ? 'Walk: Auto-approve >=90% confidence & learned patterns'
                        : 'Run: Autonomous execution for >=70% confidence'
                  }
                >
                  {p}
                </button>
              ))}
            </div>

            <button
              onClick={refreshTelemetry}
              className="p-2 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300"
              title="Refresh Telemetry"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Workspace Navigation Tabs */}
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center gap-1 overflow-x-auto border-t border-slate-900 scrollbar-hide">
          {navItems.map((tab) => {
            const Icon = tab.icon;
            const active = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-3 text-xs sm:text-sm font-medium border-b-2 whitespace-nowrap transition ${
                  active
                    ? 'border-indigo-500 text-white'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                <Icon className={`w-4 h-4 ${active ? 'text-indigo-400' : 'text-slate-500'}`} />
                {tab.label}
                {typeof tab.badge === 'number' && tab.badge > 0 && (
                  <span className="ml-1 px-2 py-0.5 text-[11px] font-bold rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                    {tab.badge}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </header>

      {/* Offline Helper Banner */}
      {connected === false && (
        <div className="bg-amber-950/70 border-b border-amber-800/60 px-4 py-3">
          <div className="max-w-7xl mx-auto flex flex-wrap items-center justify-between gap-2 text-xs text-amber-200">
            <span>
              Go backend not detected on <code className="font-mono">http://localhost:8080</code>. Start it to enable live
              orchestration:
            </span>
            <code className="px-2.5 py-1 rounded bg-slate-950 border border-amber-700/50 font-mono text-amber-300">
              go run -buildvcs=false ./cmd/asf-web
            </code>
          </div>
        </div>
      )}

      {/* Main Workspace Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'overview' && (
          <div className="space-y-7">
            {/* Interactive End-to-End Closed-Loop Pipeline Strip */}
            <div className="bg-slate-900/85 border border-slate-800 rounded-2xl p-5 shadow-xl">
              <div className="flex items-center justify-between mb-3">
                <span className="text-[11px] font-semibold uppercase tracking-widest text-indigo-400">
                  Closed-Loop Factory Architecture — Click Any Stage
                </span>
                <span className="text-xs font-mono text-slate-500">
                  Active Maturity: {activePhase.toUpperCase()}
                </span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-5 gap-2.5">
                {pipelineStages.map((st, idx) => (
                  <button
                    key={st.step}
                    onClick={() => setActiveTab(st.tab)}
                    className="group text-left p-3.5 rounded-xl bg-slate-950 hover:bg-indigo-950/30 border border-slate-800 hover:border-indigo-500/40 transition flex items-center justify-between gap-2"
                  >
                    <div>
                      <div className="text-[10px] font-mono text-indigo-400">STAGE {st.step}</div>
                      <div className="text-xs font-semibold text-slate-100 mt-0.5">{st.name}</div>
                      <div className="text-[11px] text-slate-400 mt-0.5 font-mono">{st.metric}</div>
                    </div>
                    {idx < pipelineStages.length - 1 && (
                      <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-indigo-400 transition hidden sm:block" />
                    )}
                  </button>
                ))}
              </div>
            </div>

            {/* KPI Cards */}
            <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
              <KpiCard
                label="Worker Pool"
                value={`${overview?.factory?.workers_busy ?? 0}/${overview?.factory?.workers_total ?? 4}`}
                sub={`${overview?.factory?.tasks_completed ?? 0} tasks completed`}
                icon={Cpu}
                accent="text-indigo-400"
              />
              <KpiCard
                label="Specialized Agents"
                value={overview?.agents_count ?? 10}
                sub="RBAC & policy governed"
                icon={Bot}
                accent="text-emerald-400"
              />
              <KpiCard
                label="DAG Executions"
                value={overview?.database?.total_executions ?? 0}
                sub={`${overview?.database?.total_workflows ?? 0} workflow definitions`}
                icon={GitBranch}
                accent="text-cyan-400"
              />
              <KpiCard
                label="Pending Gates"
                value={pendingCount}
                sub={`Phase: ${activePhase.toUpperCase()}`}
                icon={ShieldCheck}
                accent={pendingCount > 0 ? 'text-amber-400' : 'text-emerald-400'}
              />
              <KpiCard
                label="Learned Patterns"
                value={overview?.database?.discovered_patterns ?? 0}
                sub={`${overview?.database?.total_agent_decisions ?? 0} SQLite decisions`}
                icon={Database}
                accent="text-purple-400"
              />
            </div>

            {/* Quick Launch Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
              <ActionPanel
                title="Synthesize Multi-Agent DAG"
                description="Coordinate Product, Architect, Developer, QA, and Security agents into a topological execution graph."
                cta="Open Multi-Agent Studio"
                icon={Sparkles}
                onClick={() => setActiveTab('studio')}
              />
              <ActionPanel
                title="Isolated Workspace Spec Runner"
                description="Materialize multi-file Go artifacts in isolated workspaces with compiler self-healing and LLM-as-a-Judge scoring."
                cta="Launch Spec Studio"
                icon={Terminal}
                onClick={() => setActiveTab('workspaces')}
              />
              <ActionPanel
                title="7-Layer Stream, Webhooks & MCP"
                description="Stream real-time 7-layer SSE runs, simulate GitHub webhook deliveries, or invoke MCP JSON-RPC 2.0 tools."
                cta="Open Live Playground"
                icon={PlugZap}
                onClick={() => setActiveTab('pipelines')}
              />
            </div>

            {/* Worker Telemetry + Recent Runs + Sessions */}
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
              <div className="lg:col-span-4 bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300 flex items-center gap-2">
                    <Activity className="w-4 h-4 text-emerald-400" />
                    Worker Pool
                  </h3>
                  <span className="text-xs font-mono text-slate-500">Uptime: {overview?.uptime ?? '—'}</span>
                </div>
                <div className="space-y-2.5">
                  {(overview?.factory?.workers || []).map((w) => {
                    const busy = w.status === 'busy';
                    return (
                      <div key={w.id} className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
                        <div>
                          <div className="text-xs font-semibold text-slate-200">{w.id}</div>
                          <div className="text-[11px] text-slate-500 mt-0.5">
                            {busy && w.task_type ? `${w.task_type} (${w.task_spec || ''})` : 'Idle • awaiting task lease'}
                          </div>
                        </div>
                        <span
                          className={`inline-flex items-center gap-1.5 text-[11px] px-2.5 py-0.5 rounded-full ${
                            busy ? 'bg-indigo-500/20 text-indigo-300' : 'bg-emerald-500/10 text-emerald-300'
                          }`}
                        >
                          <span className={`w-1.5 h-1.5 rounded-full ${busy ? 'bg-indigo-400' : 'bg-emerald-400'}`} />
                          {w.status}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="lg:col-span-4 bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                    Recent Workspace Tasks
                  </h3>
                  <button onClick={() => setActiveTab('workspaces')} className="text-xs text-indigo-400 hover:text-indigo-300">
                    Inspect &rarr;
                  </button>
                </div>

                {recentTasks.length === 0 ? (
                  <div className="py-10 text-center text-xs text-slate-500 border border-dashed border-slate-800 rounded-xl">
                    No workspace tasks yet.
                  </div>
                ) : (
                  <div className="space-y-2.5">
                    {recentTasks.map((t) => (
                      <div
                        key={t.id}
                        onClick={() => setActiveTab('workspaces')}
                        className="p-3 rounded-xl bg-slate-950 border border-slate-800 hover:border-slate-700 cursor-pointer flex items-center justify-between gap-2 transition"
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <CheckCircle2
                            className={`w-4 h-4 flex-shrink-0 ${
                              t.status === 'completed'
                                ? 'text-emerald-400'
                                : t.status === 'failed'
                                  ? 'text-red-400'
                                  : 'text-indigo-400'
                            }`}
                          />
                          <div className="min-w-0">
                            <div className="text-xs font-semibold text-slate-200 truncate">
                              {t.type.toUpperCase()} · {t.spec_id}
                            </div>
                            <div className="text-[11px] font-mono text-slate-500 truncate">{t.id}</div>
                          </div>
                        </div>
                        <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-slate-900 text-slate-300">
                          {t.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="lg:col-span-4 bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                    Coordination Sessions
                  </h3>
                  <button onClick={() => setActiveTab('studio')} className="text-xs text-indigo-400 hover:text-indigo-300">
                    New &rarr;
                  </button>
                </div>

                {recentSessions.length === 0 ? (
                  <div className="py-10 text-center text-xs text-slate-500 border border-dashed border-slate-800 rounded-xl">
                    No coordination sessions yet.
                  </div>
                ) : (
                  <div className="space-y-2.5">
                    {recentSessions.map((s: any) => (
                      <div
                        key={s.ID}
                        onClick={() => setActiveTab(s.Status === 'waiting_approval' ? 'approvals' : 'studio')}
                        className="p-3 rounded-xl bg-slate-950 border border-slate-800 hover:border-slate-700 cursor-pointer transition"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-xs font-semibold text-slate-200 truncate">
                            {s.TriggerEvent?.title || s.ID}
                          </span>
                          <span
                            className={`text-[11px] px-2 py-0.5 rounded-full ${
                              s.Status === 'waiting_approval'
                                ? 'bg-amber-500/15 text-amber-300'
                                : 'bg-emerald-500/15 text-emerald-300'
                            }`}
                          >
                            {s.Status}
                          </span>
                        </div>
                        <div className="text-[11px] font-mono text-slate-500 mt-1 truncate">{s.ID}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* 10-Agent Specialized Fleet */}
            {overview?.agents && overview.agents.length > 0 && (
              <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                    Registered 10-Agent Specialized Fleet
                  </h3>
                  <span className="text-xs text-slate-500">Click any agent to coordinate a workflow</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
                  {overview.agents.map((ag) => (
                    <button
                      key={ag.role}
                      onClick={() => setActiveTab('studio')}
                      className="text-left p-3.5 rounded-xl bg-slate-950 hover:bg-slate-900 border border-slate-800 hover:border-indigo-500/40 transition"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs font-bold uppercase text-indigo-300">{ag.role}</span>
                        <span className="text-[10px] px-2 py-0.5 rounded bg-slate-900 text-slate-400 border border-slate-800">
                          {ag.decision_authority}
                        </span>
                      </div>
                      <div className="text-xs font-medium text-slate-200 mt-1">{ag.title}</div>
                      <p className="text-[11px] text-slate-500 mt-1 line-clamp-2">{ag.description}</p>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'assembly' && <FactoryAssemblyLine onRunStateChange={refreshTelemetry} />}
        {activeTab === 'studio' && <WorkflowGenerator onApprovalQueued={refreshTelemetry} />}
        {activeTab === 'workspaces' && <WorkspaceRunner />}
        {activeTab === 'pipelines' && <PipelinePlayground />}
        {activeTab === 'approvals' && <ApprovalsPanel onApprovalResolved={refreshTelemetry} />}
        {activeTab === 'learning' && <SelfImprovementDashboard />}
      </main>
    </div>
  );
}

function KpiCard({
  label,
  value,
  sub,
  icon: Icon,
  accent,
}: {
  label: string;
  value: string | number;
  sub: string;
  icon: any;
  accent: string;
}) {
  return (
    <div className="bg-slate-900/85 border border-slate-800 rounded-2xl p-5 shadow-lg">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-slate-400">{label}</span>
        <Icon className={`w-4 h-4 ${accent}`} />
      </div>
      <div className="text-2xl font-bold text-slate-100 mt-2">{value}</div>
      <div className="text-[11px] text-slate-500 mt-1 truncate">{sub}</div>
    </div>
  );
}

function ActionPanel({
  title,
  description,
  cta,
  icon: Icon,
  onClick,
}: {
  title: string;
  description: string;
  cta: string;
  icon: any;
  onClick: () => void;
}) {
  return (
    <div className="bg-slate-900/85 border border-slate-800 hover:border-indigo-500/40 rounded-2xl p-6 flex flex-col justify-between transition shadow-lg">
      <div>
        <div className="w-9 h-9 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 flex items-center justify-center mb-4">
          <Icon className="w-5 h-5" />
        </div>
        <h3 className="text-base font-semibold text-slate-100">{title}</h3>
        <p className="text-xs text-slate-400 mt-2 leading-relaxed">{description}</p>
      </div>
      <button
        onClick={onClick}
        className="mt-5 w-full py-2.5 px-4 rounded-xl bg-slate-800 hover:bg-indigo-600 text-slate-100 hover:text-white text-xs font-semibold transition"
      >
        {cta} &rarr;
      </button>
    </div>
  );
}
