'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  apiClient,
  AgentCapabilitySummary,
  FactoryPhase,
  FactoryRunRecord,
  FactoryTask,
  GatewayProviderStatus,
  SystemOverview,
} from '@/lib/api';
import {
  connectionHealthMachine,
  FRONTEND_MACHINES,
  launchMachine,
  phaseSwitchMachine,
  statusBadge,
  useMachine,
} from '@/lib/stateMachine';
import FactoryWorkbench, { STARTER_BLUEPRINTS } from '@/components/ide/FactoryWorkbench';
import StateMachineExplorer from '@/components/ide/StateMachineExplorer';
import CommandPalette, { PaletteAction } from '@/components/ide/CommandPalette';
import AgentInspectorDrawer from '@/components/ide/AgentInspectorDrawer';
import WorkflowGenerator from '@/components/WorkflowGenerator';
import ApprovalsPanel from '@/components/ApprovalsPanel';
import WorkspaceRunner from '@/components/WorkspaceRunner';
import PipelinePlayground from '@/components/PipelinePlayground';
import SelfImprovementDashboard from '@/components/SelfImprovementDashboard';
import {
  Activity,
  ArrowRight,
  BarChart3,
  Bot,
  CheckCircle2,
  ChevronRight,
  Command,
  FileCode2,
  GitFork,
  KeyRound,
  Layers,
  LayoutGrid,
  Loader2,
  Menu,
  Moon,
  Network,
  Play,
  Plus,
  Server,
  ShieldCheck,
  Sparkles,
  Sun,
  User,
  Wrench,
  X,
  Zap,
} from 'lucide-react';

type NavKey =
  | 'dashboard'
  | 'runs'
  | 'agents'
  | 'connections'
  | 'mcp'
  | 'environments'
  | 'specs'
  | 'triggers'
  | 'approvals'
  | 'state-machines'
  | 'models'
  | 'learning'
  | 'logs'
  | 'api-keys'
  | 'setup';

const FALLBACK_AGENTS: AgentCapabilitySummary[] = [
  {
    role: 'architect',
    title: 'Factory System Architect',
    description: 'Defines Go package boundaries, lock discipline, and concurrency topologies.',
    capabilities: ['architecture', 'concurrency', 'interfaces'],
    decision_authority: 'authoritative',
    interaction_mode: 'interactive',
  },
  {
    role: 'developer',
    title: 'Factory Go Developer',
    description: 'Synthesizes idiomatic multi-file Go implementations inside isolated sandboxes.',
    capabilities: ['codegen', 'refactoring', 'go-modules'],
    decision_authority: 'authoritative',
    interaction_mode: 'autonomous',
  },
  {
    role: 'security',
    title: 'Factory Security Auditor',
    description: 'Enforces memory bounds, HMAC constant-time checks, and path-traversal guards.',
    capabilities: ['threat-modeling', 'rbac', 'cryptography'],
    decision_authority: 'authoritative',
    interaction_mode: 'interactive',
  },
  {
    role: 'qa',
    title: 'Factory SpecVer & Race Tester',
    description: 'Authors table-driven unit tests and executes go test -race verification.',
    capabilities: ['unit-testing', 'race-detector', 'tdd'],
    decision_authority: 'authoritative',
    interaction_mode: 'autonomous',
  },
  {
    role: 'performance',
    title: 'Factory SRE & Performance',
    description: 'Audits allocation profiles, O(1) data structures, and throughput.',
    capabilities: ['benchmarking', 'profiling', 'latency'],
    decision_authority: 'suggestive',
    interaction_mode: 'advisory',
  },
  {
    role: 'product',
    title: 'Factory Product Strategist',
    description: 'Translates plain-English intent into testable specification invariants.',
    capabilities: ['requirements', 'acceptance-criteria'],
    decision_authority: 'suggestive',
    interaction_mode: 'interactive',
  },
  {
    role: 'design',
    title: 'Factory API Designer',
    description: 'Shapes exported Go types, constructors, and ergonomic error contracts.',
    capabilities: ['api-design', 'ergonomics'],
    decision_authority: 'suggestive',
    interaction_mode: 'advisory',
  },
  {
    role: 'compliance',
    title: 'Factory ADR Compliance',
    description: 'Verifies MUST and MUST_NOT architectural decision constraints.',
    capabilities: ['adr-audit', 'policy-gate'],
    decision_authority: 'authoritative',
    interaction_mode: 'autonomous',
  },
  {
    role: 'devops',
    title: 'Factory Build & Sandbox Engineer',
    description: 'Manages isolated Go workspaces, static analysis, and artifact promotion.',
    capabilities: ['workspaces', 'ci-cd', 'promotion'],
    decision_authority: 'suggestive',
    interaction_mode: 'autonomous',
  },
  {
    role: 'coordinator',
    title: 'Factory DAG Coordinator',
    description: 'Orchestrates level-parallel multi-agent execution and HITL checkpoints.',
    capabilities: ['dag-planning', 'consensus'],
    decision_authority: 'authoritative',
    interaction_mode: 'autonomous',
  },
];

const SECTION_TITLES: Record<NavKey, string> = {
  dashboard: 'Dashboard',
  runs: 'Factory Runs',
  agents: 'Agents',
  connections: 'Connections',
  mcp: 'MCP Servers',
  environments: 'Environments',
  specs: 'Specifications',
  triggers: 'Triggers',
  approvals: 'Approvals',
  'state-machines': 'State Machines',
  models: 'Models',
  learning: 'My Usage',
  logs: 'Logs',
  'api-keys': 'API Keys',
  setup: 'Setup',
};

function formatModelForAgent(provider: string | undefined, role: string): string {
  const p = (provider || 'mock').toLowerCase();
  if (p.includes('gemini') || p.includes('google')) {
    return ['architect', 'developer', 'security', 'coordinator'].includes(role)
      ? 'google/gemini-2.5-pro'
      : 'google/gemini-2.5-flash';
  }
  if (p.includes('anthropic') || p.includes('claude')) {
    return ['architect', 'developer', 'security', 'coordinator'].includes(role)
      ? 'anthropic/claude-sonnet-4'
      : 'anthropic/claude-haiku-3-5';
  }
  if (p.includes('openai')) return 'openai/gpt-4o';
  if (p.includes('ollama')) return 'ollama/qwen2.5-coder';
  return 'google/gemini-ready (offline-synth)';
}

export default function ControlPlanePage() {
  const [activeNav, setActiveNav] = useState<NavKey>('dashboard');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [isDark, setIsDark] = useState(false);
  const [blueprintTrigger, setBlueprintTrigger] = useState<{ spec_id: string; ts: number } | null>(
    null
  );
  const [inspectedAgent, setInspectedAgent] = useState<AgentCapabilitySummary | null>(null);
  const [toastMsg, setToastMsg] = useState<string | null>(null);

  const notify = useCallback((msg: string) => {
    setToastMsg(msg);
  }, []);

  useEffect(() => {
    if (!toastMsg) return;
    const t = setTimeout(() => setToastMsg(null), 3200);
    return () => clearTimeout(t);
  }, [toastMsg]);

  // Explicit UI state machines
  const connMachine = useMachine(connectionHealthMachine);
  const phaseMachine = useMachine(phaseSwitchMachine);
  const keySaveMachine = useMachine(launchMachine);

  // Live Control Plane telemetry
  const [overview, setOverview] = useState<SystemOverview | null>(null);
  const [runs, setRuns] = useState<FactoryRunRecord[]>([]);
  const [tasks, setTasks] = useState<FactoryTask[]>([]);
  const [sessions, setSessions] = useState<any[]>([]);
  const [backendMachineCount, setBackendMachineCount] = useState<number>(20);

  // Runtime Model Gateway & API Keys form state
  const [preferredMode, setPreferredMode] = useState<string>('auto');
  const [geminiKey, setGeminiKey] = useState('');
  const [geminiModel, setGeminiModel] = useState('gemini-2.5-flash');
  const [anthropicKey, setAnthropicKey] = useState('');
  const [openaiKey, setOpenaiKey] = useState('');
  const [ollamaUrl, setOllamaUrl] = useState('http://127.0.0.1:11434');
  const [gatewayNotice, setGatewayNotice] = useState('');

  useEffect(() => {
    setIsDark(document.documentElement.classList.contains('dark'));
  }, []);

  const toggleTheme = useCallback(() => {
    setIsDark((prev) => {
      const next = !prev;
      if (next) {
        document.documentElement.classList.add('dark');
        try {
          localStorage.setItem('asf-theme', 'dark');
        } catch {}
      } else {
        document.documentElement.classList.remove('dark');
        try {
          localStorage.setItem('asf-theme', 'light');
        } catch {}
      }
      return next;
    });
  }, []);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const refreshTelemetry = useCallback(async () => {
    try {
      const [ov, runsResp, tasksResp, sessResp, smResp] = await Promise.all([
        apiClient.getSystemOverview(),
        apiClient.listFactoryRuns().catch(() => ({ runs: [] })),
        apiClient.getTasks().catch(() => ({ tasks: [] })),
        apiClient.getSessions().catch(() => ({ sessions: [] })),
        apiClient.getStateMachines().catch(() => ({ count: 20, machines: [] })),
      ]);
      setOverview(ov);
      if (ov.gateway_mode) setPreferredMode(ov.gateway_mode);
      setRuns(runsResp.runs || []);
      setTasks(tasksResp.tasks || []);
      setSessions(sessResp.sessions || []);
      if (smResp.machines && smResp.machines.length > 0) {
        setBackendMachineCount(smResp.machines.length);
      }
      connMachine.send('online');
    } catch {
      connMachine.send('offline');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    refreshTelemetry();
    const timer = setInterval(refreshTelemetry, 4000);
    return () => clearInterval(timer);
  }, [refreshTelemetry]);

  const handlePhaseChange = async (phase: FactoryPhase) => {
    if (!phaseMachine.send('switching')) return;
    try {
      await apiClient.setSystemPhase(phase);
      phaseMachine.send('idle');
      notify(`Autonomy mode updated to ${phase.toUpperCase()}`);
      await refreshTelemetry();
    } catch {
      phaseMachine.send('error');
    }
  };

  const handleSaveGateway = async (overrideProvider?: string) => {
    if (!keySaveMachine.send('loading')) return;
    setGatewayNotice('');
    try {
      const res = await apiClient.updateGatewayConfig({
        preferred_provider: overrideProvider || preferredMode,
        gemini_api_key: geminiKey.trim() || undefined,
        gemini_model: geminiModel.trim() || undefined,
        anthropic_api_key: anthropicKey.trim() || undefined,
        openai_api_key: openaiKey.trim() || undefined,
        ollama_url: ollamaUrl.trim() || undefined,
      });
      setGatewayNotice(res.message || `Active provider updated to ${res.active_provider}`);
      setGeminiKey('');
      setAnthropicKey('');
      setOpenaiKey('');
      keySaveMachine.send('success');
      await refreshTelemetry();
    } catch (err) {
      keySaveMachine.send('error');
      setGatewayNotice(err instanceof Error ? err.message : 'Failed to update gateway');
    }
  };

  const launchBlueprintInRuns = useCallback((specId: string) => {
    setActiveNav('runs');
    setBlueprintTrigger({ spec_id: specId, ts: Date.now() });
  }, []);

  const agentsList: AgentCapabilitySummary[] =
    overview?.agents && overview.agents.length > 0 ? overview.agents : FALLBACK_AGENTS;
  const activeRunsCount = runs.filter(
    (r) => r.status === 'running' || r.status === 'awaiting_approval'
  ).length;
  const specsCount = overview?.specs_count ?? 4;
  const totalStateMachines = backendMachineCount + FRONTEND_MACHINES.length;
  const pendingApprovals = overview?.pending_approvals ?? 0;
  const activePhase: FactoryPhase = overview?.phase || 'walk';

  const gatewayProviders: GatewayProviderStatus[] = overview?.gateway_providers || [
    {
      id: 'gemini',
      name: 'Google Gemini',
      model: 'gemini-2.5-flash',
      configured: false,
      active: false,
      env_var: 'GEMINI_API_KEY',
    },
    {
      id: 'anthropic',
      name: 'Anthropic Claude',
      model: 'claude-sonnet-4-20250514',
      configured: false,
      active: false,
      env_var: 'ANTHROPIC_API_KEY',
    },
    {
      id: 'openai',
      name: 'OpenAI Compatible',
      model: 'gpt-4o',
      configured: false,
      active: false,
      env_var: 'OPENAI_API_KEY',
    },
    {
      id: 'ollama',
      name: 'Local Ollama',
      model: 'qwen2.5-coder',
      configured: false,
      active: false,
      env_var: 'OLLAMA_URL',
    },
    {
      id: 'mock',
      name: 'Deterministic Go Synthesizer',
      model: 'spec-aware-go-v2',
      configured: true,
      active: true,
      env_var: 'Built-in Offline',
    },
  ];

  const paletteActions: PaletteAction[] = useMemo(
    () => [
      ...STARTER_BLUEPRINTS.map((bp) => ({
        id: `build-${bp.spec_id}`,
        group: 'Build',
        label: `Ship: ${bp.title}`,
        hint: bp.spec_id,
        onSelect: () => launchBlueprintInRuns(bp.spec_id),
      })),
      {
        id: 'nav-dashboard',
        group: 'Control Plane',
        label: 'Dashboard',
        onSelect: () => setActiveNav('dashboard'),
      },
      {
        id: 'nav-runs',
        group: 'Control Plane',
        label: 'Factory Runs (Autonomous Build Studio)',
        onSelect: () => setActiveNav('runs'),
      },
      {
        id: 'nav-states',
        group: 'Control Plane',
        label: `State Machines (${totalStateMachines} Explicit SVG Graphs)`,
        onSelect: () => setActiveNav('state-machines'),
      },
      {
        id: 'nav-keys',
        group: 'Model Gateway',
        label: 'Configure Google Gemini / Claude / OpenAI API Keys',
        onSelect: () => setActiveNav('api-keys'),
      },
      {
        id: 'nav-models',
        group: 'Model Gateway',
        label: 'Switch Active LLM Provider',
        onSelect: () => setActiveNav('models'),
      },
      {
        id: 'theme-toggle',
        group: 'Appearance',
        label: isDark ? 'Switch to Warm Light Studio' : 'Switch to Dark Mode',
        onSelect: toggleTheme,
      },
    ],
    [totalStateMachines, isDark, launchBlueprintInRuns, toggleTheme]
  );

  const controlPlaneNav: Array<{ id: NavKey; label: string; icon: any; badge?: number }> = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutGrid },
    { id: 'runs', label: 'Factory Runs', icon: Play, badge: activeRunsCount || undefined },
    { id: 'agents', label: 'Agents', icon: Bot },
    { id: 'connections', label: 'Connections', icon: Network },
    { id: 'mcp', label: 'MCP Servers', icon: Server },
    { id: 'environments', label: 'Environments', icon: Layers },
    { id: 'specs', label: 'Specifications', icon: FileCode2 },
    { id: 'triggers', label: 'Triggers', icon: Zap },
    { id: 'approvals', label: 'Approvals', icon: ShieldCheck, badge: pendingApprovals || undefined },
    { id: 'state-machines', label: 'State Machines', icon: GitFork },
  ];

  const modelGatewayNav: Array<{ id: NavKey; label: string; icon: any }> = [
    { id: 'models', label: 'Models', icon: Sparkles },
    { id: 'learning', label: 'My Usage', icon: BarChart3 },
    { id: 'logs', label: 'Logs', icon: Activity },
    { id: 'api-keys', label: 'API Keys', icon: KeyRound },
    { id: 'setup', label: 'Setup', icon: Wrench },
  ];

  const renderSidebarContent = () => (
    <div className="h-full flex flex-col justify-between select-none">
      <div>
        {/* Brand Header */}
        <div className="h-14 px-5 border-b ide-border flex items-center justify-between">
          <button
            type="button"
            onClick={() => setActiveNav('dashboard')}
            className="flex items-center gap-2.5 text-left"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" className="shrink-0">
              <path d="M12 2a10 10 0 0 1 10 10H2A10 10 0 0 1 12 2Z" fill="currentColor" />
              <path
                d="M2 12h20a10 10 0 0 1-20 0Z"
                fill="currentColor"
                fillOpacity="0.32"
              />
            </svg>
            <span className="text-[19px] tracking-tight ide-ink">
              <span className="font-normal">asf</span>
              <span className="font-bold">factory</span>
            </span>
          </button>

          <button
            type="button"
            onClick={() => setMobileMenuOpen(false)}
            className="lg:hidden p-1 ide-ink-muted"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Sidebar Sections */}
        <div className="px-3 py-4 space-y-5 overflow-y-auto max-h-[calc(100vh-7.5rem)]">
          <div>
            <div className="px-3.5 mb-1.5 text-[11px] font-medium uppercase tracking-[0.08em] ide-ink-muted">
              CONTROL PLANE
            </div>
            <nav className="space-y-0.5">
              {controlPlaneNav.map((item) => {
                const Icon = item.icon;
                const active = activeNav === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => {
                      setActiveNav(item.id);
                      setMobileMenuOpen(false);
                    }}
                    className={`w-full flex items-center justify-between px-3.5 py-2 rounded-[12px] text-[13.5px] transition ${
                      active
                        ? 'kora-nav-active'
                        : 'ide-ink-secondary hover:ide-ink hover:bg-black/[0.03] dark:hover:bg-white/[0.04] border border-transparent'
                    }`}
                  >
                    <span className="flex items-center gap-2.5 truncate">
                      <Icon className="w-4 h-4 shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </span>
                    {typeof item.badge === 'number' && item.badge > 0 && (
                      <span className="text-[11px] font-mono px-2 py-0.2 rounded-full bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3]">
                        {item.badge}
                      </span>
                    )}
                  </button>
                );
              })}
            </nav>
          </div>

          <div>
            <div className="px-3.5 mb-1.5 text-[11px] font-medium uppercase tracking-[0.08em] ide-ink-muted">
              MODEL GATEWAY
            </div>
            <nav className="space-y-0.5">
              {modelGatewayNav.map((item) => {
                const Icon = item.icon;
                const active = activeNav === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => {
                      setActiveNav(item.id);
                      setMobileMenuOpen(false);
                    }}
                    className={`w-full flex items-center gap-2.5 px-3.5 py-2 rounded-[12px] text-[13.5px] transition ${
                      active
                        ? 'kora-nav-active'
                        : 'ide-ink-secondary hover:ide-ink hover:bg-black/[0.03] dark:hover:bg-white/[0.04] border border-transparent'
                    }`}
                  >
                    <Icon className="w-4 h-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </button>
                );
              })}
            </nav>
          </div>
        </div>
      </div>

      {/* Bottom Operator Footer */}
      <button
        type="button"
        onClick={() => setActiveNav('setup')}
        className="p-4 border-t ide-border flex items-center justify-between gap-2 text-left hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition"
      >
        <div className="flex items-center gap-2.5 min-w-0">
          <div className="w-8 h-8 rounded-full ide-subtle border ide-border flex items-center justify-center shrink-0">
            <User className="w-4 h-4 ide-ink-secondary" />
          </div>
          <div className="min-w-0">
            <div className="text-xs font-semibold ide-ink truncate">Pratik Dhanave</div>
            <div className="text-[11px] ide-ink-muted truncate font-mono">
              {connMachine.current === 'online'
                ? `${overview?.llm_provider || 'mock'} • sqlite-wal`
                : connMachine.current}
            </div>
          </div>
        </div>
        <ChevronRight className="w-4 h-4 ide-ink-muted shrink-0" />
      </button>
    </div>
  );

  return (
    <div className="min-h-screen ide-canvas ide-ink flex">
      {/* Desktop Sidebar */}
      <aside className="hidden lg:block w-[244px] shrink-0 border-r ide-border ide-sidebar sticky top-0 h-screen">
        {renderSidebarContent()}
      </aside>

      {/* Mobile Drawer */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-50 lg:hidden bg-black/30 backdrop-blur-[1px]"
          onClick={() => setMobileMenuOpen(false)}
        >
          <div
            className="w-[250px] h-full ide-sidebar border-r ide-border"
            onClick={(e) => e.stopPropagation()}
          >
            {renderSidebarContent()}
          </div>
        </div>
      )}

      {/* Main Column */}
      <div className="flex-1 min-w-0 flex flex-col">
        {/* Top Header */}
        <header className="h-14 px-6 lg:px-8 border-b ide-border ide-sidebar sticky top-0 z-20 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setMobileMenuOpen(true)}
              className="lg:hidden p-1.5 rounded-lg border ide-border ide-ink-secondary"
            >
              <Menu className="w-4 h-4" />
            </button>
            <span className="text-sm font-semibold ide-ink">
              {SECTION_TITLES[activeNav]}
            </span>
          </div>

          <div className="flex items-center gap-2.5">
            <button
              type="button"
              onClick={() => setPaletteOpen(true)}
              className="hidden sm:inline-flex items-center gap-2 px-3 py-1 rounded-full ide-surface border ide-border text-xs ide-ink-muted hover:ide-ink transition"
              title="Command Palette (⌘K)"
            >
              <span>Search</span>
              <span className="inline-flex items-center gap-0.5 font-mono text-[10px] px-1.5 py-0.2 rounded ide-subtle ide-ink-secondary">
                <Command className="w-2.5 h-2.5" />K
              </span>
            </button>

            <div className="hidden md:flex items-center p-0.5 rounded-full ide-surface border ide-border text-[11px]">
              {(['crawl', 'walk', 'run'] as FactoryPhase[]).map((p) => (
                <button
                  key={p}
                  type="button"
                  disabled={phaseMachine.current === 'switching'}
                  onClick={() => handlePhaseChange(p)}
                  className={`px-2.5 py-0.5 rounded-full capitalize font-medium transition ${
                    activePhase === p
                      ? 'bg-[#181816] text-white dark:bg-white dark:text-black'
                      : 'ide-ink-muted hover:ide-ink'
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>

            <span className="px-3 py-0.5 rounded-full border ide-border-strong text-xs font-mono ide-ink">
              v2.0.0
            </span>

            <button
              type="button"
              onClick={toggleTheme}
              className="p-1.5 rounded-full hover:bg-black/5 dark:hover:bg-white/10 ide-ink-secondary hover:ide-ink transition"
              title={isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
            >
              {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
            </button>
          </div>
        </header>

        {connMachine.current === 'offline' && (
          <div className="px-6 lg:px-8 py-2.5 bg-[#F5E8CE] border-b border-[#E6D0A3] flex flex-wrap items-center justify-between gap-2 text-xs text-[#7A5213]">
            <span>
              Control plane server not detected on <code className="font-mono">localhost:8080</code>. Start it to run live sandbox builds:
            </span>
            <code className="px-2.5 py-0.5 rounded-full bg-white/80 border border-[#E6D0A3] font-mono">
              ./bin/asf-web
            </code>
          </div>
        )}

        {/* ================= DASHBOARD ================= */}
        {activeNav === 'dashboard' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1260px] w-full space-y-6">
            <div className="flex flex-wrap items-end justify-between gap-4">
              <div>
                <h1 className="text-[28px] leading-tight font-bold tracking-tight ide-ink">
                  Dashboard
                </h1>
                <p className="text-sm ide-ink-muted mt-1">
                  Real-time overview of the Agentic Software Factory control plane.
                </p>
              </div>

              <button
                type="button"
                onClick={() => setActiveNav('runs')}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[#181816] hover:opacity-90 dark:bg-white dark:text-black text-white text-xs font-medium transition"
              >
                <Plus className="w-3.5 h-3.5" />
                New Factory Run
              </button>
            </div>

            {/* 4 Clean KPI Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <OverviewStatCard
                label="AGENT TYPES"
                value={agentsList.length}
                sub="registered"
                subTone="green"
                icon={Bot}
                onClick={() => setActiveNav('agents')}
              />
              <OverviewStatCard
                label="ACTIVE RUNS"
                value={activeRunsCount}
                sub={`${runs.length} total`}
                subTone="ochre"
                icon={Activity}
                onClick={() => setActiveNav('runs')}
              />
              <OverviewStatCard
                label="SPECIFICATIONS"
                value={specsCount}
                sub="configured"
                subTone="muted"
                icon={FileCode2}
                onClick={() => setActiveNav('specs')}
              />
              <OverviewStatCard
                label="STATE MACHINES"
                value={totalStateMachines}
                sub="explicit"
                subTone="muted"
                icon={GitFork}
                onClick={() => setActiveNav('state-machines')}
              />
            </div>

            {/* Agent Types Table Card */}
            <section className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <div className="px-6 py-5 flex items-center justify-between gap-4">
                <div>
                  <h2 className="text-base font-semibold ide-ink">Agent Types</h2>
                  <p className="text-xs ide-ink-muted mt-0.5">
                    All agent types registered with the control plane.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setActiveNav('agents')}
                  className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-full border border-[var(--ide-ink)] text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
                >
                  View all <ChevronRight className="w-3.5 h-3.5" />
                </button>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-y ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                      <th className="py-2.5 px-6 font-medium">NAME</th>
                      <th className="py-2.5 px-6 font-medium">STATUS</th>
                      <th className="py-2.5 px-6 font-medium">MODEL</th>
                      <th className="py-2.5 px-6 font-medium">AUTHORITY</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y ide-border text-sm">
                    {agentsList.slice(0, 6).map((ag) => (
                      <tr
                        key={ag.role}
                        onClick={() => setInspectedAgent(ag)}
                        className="hover:bg-black/[0.018] dark:hover:bg-white/[0.02] cursor-pointer transition"
                      >
                        <td className="py-3.5 px-6">
                          <div className="font-semibold text-[13.5px] ide-ink">
                            {ag.title.startsWith('Factory') ? ag.title : `Factory ${ag.title}`}
                          </div>
                          <div className="text-xs font-mono ide-ink-muted mt-0.5">
                            factory-{ag.role}
                          </div>
                        </td>
                        <td className="py-3.5 px-6">
                          <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                            <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45] dark:bg-emerald-400" />
                            Healthy
                          </span>
                        </td>
                        <td className="py-3.5 px-6 font-mono text-xs ide-ink-secondary">
                          {formatModelForAgent(overview?.llm_provider, ag.role)}
                        </td>
                        <td className="py-3.5 px-6 text-xs ide-ink-muted capitalize">
                          {ag.decision_authority} · {ag.interaction_mode}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="px-6 py-3.5 border-t ide-border flex items-center justify-between text-xs ide-ink-muted">
                <span>
                  Showing {Math.min(6, agentsList.length)} of {agentsList.length} agent types
                </span>
                <button
                  type="button"
                  onClick={() => setActiveNav('agents')}
                  className="inline-flex items-center gap-1 font-semibold ide-ink hover:underline"
                >
                  View all <ChevronRight className="w-3.5 h-3.5" />
                </button>
              </div>
            </section>

            {/* Recent Factory Runs OR Quick-Start Blueprints */}
            <section className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <div className="px-6 py-5 flex items-center justify-between gap-4">
                <div>
                  <h2 className="text-base font-semibold ide-ink">
                    {runs.length > 0 ? 'Recent Factory Runs' : 'Quick-Start Blueprints'}
                  </h2>
                  <p className="text-xs ide-ink-muted mt-0.5">
                    {runs.length > 0
                      ? 'Autonomous Go package builds verified with go build, go vet, and go test -race.'
                      : 'Ship a complete, race-tested Go package in one click.'}
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setActiveNav('runs')}
                  className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-full border border-[var(--ide-ink)] text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
                >
                  Open Studio <ChevronRight className="w-3.5 h-3.5" />
                </button>
              </div>

              {runs.length > 0 ? (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-y ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                        <th className="py-2.5 px-6 font-medium">BUILD</th>
                        <th className="py-2.5 px-6 font-medium">STATUS</th>
                        <th className="py-2.5 px-6 font-medium">STAGE</th>
                        <th className="py-2.5 px-6 font-medium text-right">QUALITY SCORE</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y ide-border text-sm">
                      {runs.slice(0, 5).map((r) => {
                        const score = r.payload?.evaluation?.overall_score;
                        return (
                          <tr
                            key={r.id}
                            onClick={() => setActiveNav('runs')}
                            className="hover:bg-black/[0.018] dark:hover:bg-white/[0.02] cursor-pointer transition"
                          >
                            <td className="py-3.5 px-6">
                              <div className="font-semibold text-[13.5px] ide-ink">{r.title}</div>
                              <div className="text-xs font-mono ide-ink-muted mt-0.5">
                                {r.id} • {r.spec_id || 'synthesized'}
                              </div>
                            </td>
                            <td className="py-3.5 px-6">
                              <span
                                className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${
                                  statusBadge[r.status] || statusBadge.idle
                                }`}
                              >
                                {r.status === 'awaiting_approval' ? 'Needs Approval' : r.status}
                              </span>
                            </td>
                            <td className="py-3.5 px-6 font-mono text-xs ide-ink-secondary">
                              {r.stage}
                            </td>
                            <td className="py-3.5 px-6 font-mono text-xs ide-ink text-right tabular-nums">
                              {typeof score === 'number' ? `${Math.round(score)} / 100` : '—'}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="px-6 pb-6 pt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3 border-t ide-border">
                  {STARTER_BLUEPRINTS.map((bp) => (
                    <button
                      key={bp.spec_id}
                      type="button"
                      onClick={() => launchBlueprintInRuns(bp.spec_id)}
                      className="text-left p-4 rounded-2xl ide-canvas border ide-border hover:ide-border-strong transition flex flex-col justify-between gap-3 group"
                    >
                      <div>
                        <div className="text-[11px] font-mono ide-ink-muted">{bp.spec_id}</div>
                        <div className="text-xs font-semibold ide-ink mt-1">{bp.title}</div>
                        <p className="text-[11px] ide-ink-muted mt-1 leading-relaxed">{bp.short}</p>
                      </div>
                      <span className="inline-flex items-center gap-1 text-xs font-medium ide-ink group-hover:translate-x-0.5 transition">
                        Build now <ArrowRight className="w-3.5 h-3.5" />
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </section>
          </main>
        )}

        {/* ================= FACTORY RUNS ================= */}
        {activeNav === 'runs' && (
          <FactoryWorkbench
            activePhase={activePhase}
            externalBlueprintTrigger={blueprintTrigger}
            onRunStateChange={refreshTelemetry}
            onNotify={notify}
          />
        )}

        {/* ================= AGENTS ================= */}
        {activeNav === 'agents' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1260px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Agents</h1>
              <p className="text-sm ide-ink-muted mt-1">
                All {agentsList.length} specialized agents registered with the control plane. Click any agent to consult directly or synthesize a multi-agent DAG below.
              </p>
            </div>

            <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                    <th className="py-3 px-6 font-medium">NAME</th>
                    <th className="py-3 px-6 font-medium">STATUS</th>
                    <th className="py-3 px-6 font-medium">MODEL</th>
                    <th className="py-3 px-6 font-medium">CAPABILITIES</th>
                    <th className="py-3 px-6 font-medium text-right">STUDIO</th>
                  </tr>
                </thead>
                <tbody className="divide-y ide-border text-sm">
                  {agentsList.map((ag) => (
                    <tr key={ag.role} className="hover:bg-black/[0.015] dark:hover:bg-white/[0.02]">
                      <td className="py-3.5 px-6">
                        <div className="font-semibold text-[13.5px] ide-ink">
                          {ag.title.startsWith('Factory') ? ag.title : `Factory ${ag.title}`}
                        </div>
                        <div className="text-xs font-mono ide-ink-muted">factory-{ag.role}</div>
                      </td>
                      <td className="py-3.5 px-6">
                        <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                          <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                          Healthy
                        </span>
                      </td>
                      <td className="py-3.5 px-6 font-mono text-xs ide-ink-secondary">
                        {formatModelForAgent(overview?.llm_provider, ag.role)}
                      </td>
                      <td className="py-3.5 px-6 text-xs ide-ink-muted">
                        {(ag.capabilities || []).slice(0, 3).join(', ')}
                      </td>
                      <td className="py-3.5 px-6 text-right">
                        <div className="inline-flex items-center gap-2 justify-end">
                          <button
                            type="button"
                            onClick={() => setInspectedAgent(ag)}
                            className="inline-flex items-center gap-1 px-3.5 py-1 rounded-full border border-[var(--ide-ink)] text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
                          >
                            Consult <ChevronRight className="w-3 h-3" />
                          </button>
                          <Link
                            href={`/agents/${ag.role}`}
                            className="text-xs ide-ink-muted hover:ide-ink underline"
                          >
                            Page
                          </Link>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <WorkflowGenerator onApprovalQueued={refreshTelemetry} />
          </main>
        )}

        {/* ================= CONNECTIONS ================= */}
        {activeNav === 'connections' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Connections</h1>
              <p className="text-sm ide-ink-muted mt-1">
                External repository, issue tracker, and protocol adapters governed by the explicit `integration-connection` state machine.
              </p>
            </div>

            <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                    <th className="py-3 px-6 font-medium">ADAPTER</th>
                    <th className="py-3 px-6 font-medium">STATE</th>
                    <th className="py-3 px-6 font-medium">ENDPOINT / CONFIG</th>
                    <th className="py-3 px-6 font-medium text-right">ACTION</th>
                  </tr>
                </thead>
                <tbody className="divide-y ide-border text-sm">
                  <tr>
                    <td className="py-4 px-6">
                      <div className="font-semibold ide-ink">GitHub Webhooks &amp; PR Triage</div>
                      <div className="text-xs font-mono ide-ink-muted">github-webhook-adapter</div>
                    </td>
                    <td className="py-4 px-6">
                      <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                        connected
                      </span>
                    </td>
                    <td className="py-4 px-6 font-mono text-xs ide-ink-secondary">
                      POST /webhooks/github (HMAC-SHA256)
                    </td>
                    <td className="py-4 px-6 text-right">
                      <button
                        type="button"
                        onClick={() => setActiveNav('triggers')}
                        className="px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink"
                      >
                        Test Trigger
                      </button>
                    </td>
                  </tr>
                  <tr>
                    <td className="py-4 px-6">
                      <div className="font-semibold ide-ink">Model Context Protocol (MCP)</div>
                      <div className="text-xs font-mono ide-ink-muted">mcp-jsonrpc-adapter</div>
                    </td>
                    <td className="py-4 px-6">
                      <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                        connected
                      </span>
                    </td>
                    <td className="py-4 px-6 font-mono text-xs ide-ink-secondary">
                      POST /api/mcp/factory (9 tools · 3 resources)
                    </td>
                    <td className="py-4 px-6 text-right">
                      <button
                        type="button"
                        onClick={() => setActiveNav('mcp')}
                        className="px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink"
                      >
                        Open Console
                      </button>
                    </td>
                  </tr>
                  <tr>
                    <td className="py-4 px-6">
                      <div className="font-semibold ide-ink">SQLite WAL Persistence</div>
                      <div className="text-xs font-mono ide-ink-muted">sqlite3-wal-store</div>
                    </td>
                    <td className="py-4 px-6">
                      <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                        connected
                      </span>
                    </td>
                    <td className="py-4 px-6 font-mono text-xs ide-ink-secondary">
                      {overview?.db_path || '.asf/factory.db'}
                    </td>
                    <td className="py-4 px-6 text-right">
                      <button
                        type="button"
                        onClick={() => setActiveNav('logs')}
                        className="px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink"
                      >
                        View Ledger
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </main>
        )}

        {/* ================= SPECIFICATIONS ================= */}
        {activeNav === 'specs' && (
          <WorkspaceRunner onLaunchInWorkbench={launchBlueprintInRuns} />
        )}

        {/* ================= APPROVALS ================= */}
        {activeNav === 'approvals' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1100px] w-full">
            <ApprovalsPanel onApprovalResolved={refreshTelemetry} />
          </main>
        )}

        {/* ================= MCP SERVERS ================= */}
        {activeNav === 'mcp' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">MCP Servers</h1>
              <p className="text-sm ide-ink-muted mt-1">
                Built-in Model Context Protocol JSON-RPC 2.0 server exposing specs, verification, and all explicit state machines.
              </p>
            </div>
            <PipelinePlayground focusMode="mcp" />
          </main>
        )}

        {/* ================= ENVIRONMENTS ================= */}
        {activeNav === 'environments' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Environments</h1>
              <p className="text-sm ide-ink-muted mt-1">
                Isolated Go module sandboxes (<code className="font-mono">.asf/workspaces/*</code>) and promoted deliverables (<code className="font-mono">generated/*</code>).
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="ide-surface border ide-border rounded-2xl p-5">
                <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
                  WORKER POOL
                </div>
                <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
                  {overview?.factory?.workers_busy ?? 0}/{overview?.factory?.workers_total ?? 4}
                </div>
                <div className="text-xs ide-ink-muted mt-1">busy / configured workers</div>
              </div>
              <div className="ide-surface border ide-border rounded-2xl p-5">
                <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
                  ISOLATED SANDBOXES
                </div>
                <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
                  {runs.filter((r) => Boolean(r.workspace_dir)).length}
                </div>
                <div className="text-xs ide-ink-muted mt-1">provisioned workspaces</div>
              </div>
              <div className="ide-surface border ide-border rounded-2xl p-5">
                <div className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
                  PROMOTED PACKAGES
                </div>
                <div className="text-3xl font-bold ide-ink mt-2 tabular-nums">
                  {runs.filter((r) => Boolean(r.promoted_dir)).length}
                </div>
                <div className="text-xs ide-ink-muted mt-1">exported to generated/</div>
              </div>
            </div>

            <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <div className="px-6 py-4 border-b ide-border">
                <h3 className="text-base font-semibold ide-ink">Workspace &amp; Promotion Ledger</h3>
              </div>
              {runs.length === 0 ? (
                <div className="p-10 text-center text-xs ide-ink-muted">
                  No isolated sandboxes provisioned yet. Launch a run in Factory Runs to create one.
                </div>
              ) : (
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                      <th className="py-2.5 px-6 font-medium">RUN</th>
                      <th className="py-2.5 px-6 font-medium">SANDBOX PATH</th>
                      <th className="py-2.5 px-6 font-medium">PROMOTED PATH</th>
                      <th className="py-2.5 px-6 font-medium">STATUS</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y ide-border text-xs">
                    {runs.map((r) => (
                      <tr key={r.id}>
                        <td className="py-3 px-6 font-medium ide-ink">{r.title}</td>
                        <td className="py-3 px-6 font-mono ide-ink-muted">
                          {r.workspace_dir ? r.workspace_dir.split('/').slice(-3).join('/') : '—'}
                        </td>
                        <td className="py-3 px-6 font-mono ide-ink-secondary">
                          {r.promoted_dir || 'unpromoted'}
                        </td>
                        <td className="py-3 px-6">
                          <span className={`px-2.5 py-0.5 rounded-full font-medium ${statusBadge[r.status] || statusBadge.idle}`}>
                            {r.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </main>
        )}

        {/* ================= TRIGGERS ================= */}
        {activeNav === 'triggers' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Triggers</h1>
              <p className="text-sm ide-ink-muted mt-1">
                GitHub webhook listeners, `@factory` issue comment triggers, and queued background jobs.
              </p>
            </div>
            <PipelinePlayground focusMode="webhooks" />
          </main>
        )}

        {/* ================= STATE MACHINES ================= */}
        {activeNav === 'state-machines' && <StateMachineExplorer />}

        {/* ================= MODEL GATEWAY: MODELS ================= */}
        {activeNav === 'models' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div className="flex flex-wrap items-end justify-between gap-4">
              <div>
                <h1 className="text-[28px] font-bold tracking-tight ide-ink">Models</h1>
                <p className="text-sm ide-ink-muted mt-1">
                  Unified Model Gateway supporting Google Gemini, Anthropic Claude, OpenAI, Local Ollama, and offline deterministic Go synthesis.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setActiveNav('api-keys')}
                className="px-4 py-2 rounded-full bg-[#181816] text-white dark:bg-white dark:text-black text-xs font-semibold transition"
              >
                Configure API Keys
              </button>
            </div>

            {gatewayNotice && (
              <div className="px-4 py-3 rounded-xl kora-pill-healthy text-xs">{gatewayNotice}</div>
            )}

            <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
              <div className="px-6 py-4 border-b ide-border flex items-center justify-between">
                <div>
                  <h3 className="text-base font-semibold ide-ink">Gateway Routing Table</h3>
                  <p className="text-xs ide-ink-muted mt-0.5">
                    Active primary provider:{' '}
                    <code className="font-mono font-semibold ide-ink">
                      {overview?.llm_provider || 'mock'}
                    </code>
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => handleSaveGateway('auto')}
                  className="px-3.5 py-1 rounded-full border ide-border text-xs font-medium ide-ink-secondary hover:ide-ink"
                >
                  Auto-Route
                </button>
              </div>

              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b ide-border text-[11px] uppercase tracking-[0.06em] ide-ink-muted">
                    <th className="py-3 px-6 font-medium">PROVIDER</th>
                    <th className="py-3 px-6 font-medium">STATUS</th>
                    <th className="py-3 px-6 font-medium">DEFAULT MODEL</th>
                    <th className="py-3 px-6 font-medium">CREDENTIALS</th>
                    <th className="py-3 px-6 font-medium text-right">ROUTING</th>
                  </tr>
                </thead>
                <tbody className="divide-y ide-border text-sm">
                  {gatewayProviders.map((prov) => (
                    <tr key={prov.id}>
                      <td className="py-4 px-6">
                        <div className="font-semibold text-[13.5px] ide-ink">{prov.name}</div>
                        <div className="text-xs font-mono ide-ink-muted">provider:{prov.id}</div>
                      </td>
                      <td className="py-4 px-6">
                        {prov.active ? (
                          <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
                            <span className="w-1.5 h-1.5 rounded-full bg-[#2F7A45]" />
                            Active
                          </span>
                        ) : prov.configured ? (
                          <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusBadge.running}`}>
                            Ready
                          </span>
                        ) : (
                          <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusBadge.idle}`}>
                            No Key
                          </span>
                        )}
                      </td>
                      <td className="py-4 px-6 font-mono text-xs ide-ink-secondary">
                        {prov.model}
                      </td>
                      <td className="py-4 px-6 font-mono text-xs ide-ink-muted">
                        {prov.masked_key || prov.env_var}
                      </td>
                      <td className="py-4 px-6 text-right">
                        {prov.configured ? (
                          <button
                            type="button"
                            disabled={prov.active}
                            onClick={() => handleSaveGateway(prov.id)}
                            className="px-3.5 py-1 rounded-full border border-[var(--ide-ink)] disabled:opacity-40 text-xs font-medium ide-ink hover:bg-[var(--ide-ink)] hover:text-[var(--ide-surface)] transition"
                          >
                            {prov.active ? 'Selected' : 'Use Provider'}
                          </button>
                        ) : (
                          <button
                            type="button"
                            onClick={() => setActiveNav('api-keys')}
                            className="px-3.5 py-1 rounded-full border ide-border text-xs ide-ink-muted hover:ide-ink transition"
                          >
                            Add Key
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </main>
        )}

        {/* ================= MODEL GATEWAY: MY USAGE ================= */}
        {activeNav === 'learning' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full">
            <SelfImprovementDashboard />
          </main>
        )}

        {/* ================= MODEL GATEWAY: LOGS ================= */}
        {activeNav === 'logs' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1200px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Logs</h1>
              <p className="text-sm ide-ink-muted mt-1">
                Persisted SQLite WAL telemetry across coordination sessions and worker-pool tasks ({overview?.db_path || '.asf/factory.db'}).
              </p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
                <div className="px-6 py-4 border-b ide-border">
                  <h3 className="text-base font-semibold ide-ink">
                    Coordination Sessions ({sessions.length})
                  </h3>
                </div>
                {sessions.length === 0 ? (
                  <div className="p-8 text-center text-xs ide-ink-muted">
                    No coordination sessions recorded yet.
                  </div>
                ) : (
                  <div className="divide-y ide-border">
                    {sessions.slice(0, 10).map((s: any) => (
                      <div key={s.ID} className="px-6 py-3.5 flex items-center justify-between text-xs">
                        <div className="min-w-0 pr-3">
                          <div className="font-semibold ide-ink truncate">
                            {s.TriggerEvent?.title || s.ID}
                          </div>
                          <div className="font-mono text-[11px] ide-ink-muted truncate">{s.ID}</div>
                        </div>
                        <span className={`px-2.5 py-0.5 rounded-full font-medium shrink-0 ${statusBadge[s.Status] || statusBadge.idle}`}>
                          {s.Status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="ide-surface border ide-border rounded-2xl overflow-hidden">
                <div className="px-6 py-4 border-b ide-border">
                  <h3 className="text-base font-semibold ide-ink">
                    Worker Tasks ({tasks.length})
                  </h3>
                </div>
                {tasks.length === 0 ? (
                  <div className="p-8 text-center text-xs ide-ink-muted">
                    No worker tasks recorded yet.
                  </div>
                ) : (
                  <div className="divide-y ide-border">
                    {tasks.slice(0, 10).map((t) => (
                      <div key={t.id} className="px-6 py-3.5 flex items-center justify-between text-xs">
                        <div>
                          <div className="font-semibold ide-ink">
                            {t.type.toUpperCase()} · {t.spec_id}
                          </div>
                          <div className="font-mono text-[11px] ide-ink-muted">{t.id}</div>
                        </div>
                        <span className={`px-2.5 py-0.5 rounded-full font-medium ${statusBadge[t.status] || statusBadge.idle}`}>
                          {t.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </main>
        )}

        {/* ================= MODEL GATEWAY: API KEYS ================= */}
        {activeNav === 'api-keys' && (
          <main className="px-6 lg:px-8 py-7 max-w-[920px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">API Keys</h1>
              <p className="text-sm ide-ink-muted mt-1">
                Hot-reload Google Gemini, Anthropic Claude, OpenAI, or Local Ollama credentials into the running gateway without restarting the server.
              </p>
            </div>

            {gatewayNotice && (
              <div className="px-4 py-3 rounded-xl kora-pill-healthy text-xs">{gatewayNotice}</div>
            )}

            <div className="ide-surface border ide-border rounded-2xl p-6 space-y-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Google Gemini API Key (`GEMINI_API_KEY`)
                  </label>
                  <input
                    type="password"
                    value={geminiKey}
                    onChange={(e) => setGeminiKey(e.target.value)}
                    placeholder="AIzaSy..."
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm font-mono ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
                  />
                </div>

                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Gemini Default Model
                  </label>
                  <select
                    value={geminiModel}
                    onChange={(e) => setGeminiModel(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm font-mono ide-ink focus:outline-none"
                  >
                    <option value="gemini-2.5-flash">gemini-2.5-flash</option>
                    <option value="gemini-2.5-pro">gemini-2.5-pro</option>
                    <option value="gemini-2.0-flash">gemini-2.0-flash</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Anthropic API Key (`ANTHROPIC_API_KEY`)
                  </label>
                  <input
                    type="password"
                    value={anthropicKey}
                    onChange={(e) => setAnthropicKey(e.target.value)}
                    placeholder="sk-ant-..."
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm font-mono ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
                  />
                </div>

                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    OpenAI API Key (`OPENAI_API_KEY`)
                  </label>
                  <input
                    type="password"
                    value={openaiKey}
                    onChange={(e) => setOpenaiKey(e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm font-mono ide-ink focus:outline-none focus:border-[var(--ide-ink)]"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Local Ollama Host (`OLLAMA_URL`)
                  </label>
                  <input
                    value={ollamaUrl}
                    onChange={(e) => setOllamaUrl(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm font-mono ide-ink focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium mb-1.5">
                    Preferred Routing Mode
                  </label>
                  <select
                    value={preferredMode}
                    onChange={(e) => setPreferredMode(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-xl ide-canvas border ide-border text-sm ide-ink focus:outline-none"
                  >
                    <option value="auto">Auto (Gemini → Claude → OpenAI → Ollama → Offline)</option>
                    <option value="gemini">Google Gemini</option>
                    <option value="anthropic">Anthropic Claude</option>
                    <option value="openai">OpenAI Compatible</option>
                    <option value="ollama">Local Ollama</option>
                    <option value="mock">Deterministic Offline Synthesizer</option>
                  </select>
                </div>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  type="button"
                  disabled={keySaveMachine.current === 'loading'}
                  onClick={() => handleSaveGateway()}
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-[#181816] text-white dark:bg-white dark:text-black text-xs font-semibold transition"
                >
                  {keySaveMachine.current === 'loading' && (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  )}
                  Save &amp; Hot-Reload Gateway
                </button>
              </div>
            </div>
          </main>
        )}

        {/* ================= MODEL GATEWAY: SETUP ================= */}
        {activeNav === 'setup' && (
          <main className="px-6 lg:px-8 py-7 max-w-[1000px] w-full space-y-6">
            <div>
              <h1 className="text-[28px] font-bold tracking-tight ide-ink">Setup</h1>
              <p className="text-sm ide-ink-muted mt-1">
                Control plane readiness checks and CLI reference for local and CI execution.
              </p>
            </div>

            <div className="ide-surface border ide-border rounded-2xl divide-y ide-border">
              {[
                {
                  title: 'Control Plane HTTP & SSE Server',
                  detail: 'Serving bundled UI and REST/SSE APIs on http://localhost:8080',
                  ok: connMachine.current === 'online',
                },
                {
                  title: 'SQLite WAL Unified Store',
                  detail: overview?.db_path || '.asf/factory.db',
                  ok: connMachine.current === 'online',
                },
                {
                  title: 'Explicit State Machine Registry',
                  detail: `${totalStateMachines} declarative backend & UI machines validated`,
                  ok: true,
                },
                {
                  title: 'Model Gateway & Deterministic Fallback',
                  detail: `Active provider: ${overview?.llm_provider || 'mock'} (zero-key ready)`,
                  ok: true,
                },
              ].map((row) => (
                <div key={row.title} className="px-6 py-4 flex items-center justify-between gap-4">
                  <div>
                    <div className="text-sm font-semibold ide-ink">{row.title}</div>
                    <div className="text-xs font-mono ide-ink-muted mt-0.5">{row.detail}</div>
                  </div>
                  <span className="kora-pill-healthy inline-flex items-center gap-1.5 px-3 py-0.5 rounded-full text-xs font-medium">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    {row.ok ? 'Ready' : 'Pending'}
                  </span>
                </div>
              ))}
            </div>

            <div className="ide-surface border ide-border rounded-2xl p-6 space-y-3">
              <h3 className="text-base font-semibold ide-ink">CLI Quick Reference</h3>
              <pre className="p-4 rounded-xl bg-[#181816] text-[#F3F2ED] text-xs font-mono overflow-x-auto leading-relaxed">{`# 1. Start the unified Control Plane & UI
./bin/asf-web

# 2. Run an autonomous 6-stage build from the CLI and promote to generated/
./bin/asf build -spec spec-circuit-breaker -promote

# 3. Export all 20 explicit backend state machines (JSON or Markdown/Mermaid)
./bin/asf factory states --json
./bin/asf factory states --markdown`}</pre>
            </div>
          </main>
        )}
      </div>

      {/* Slide-Over Specialized Agent Consultation Drawer */}
      <AgentInspectorDrawer
        agent={inspectedAgent}
        modelLabel={
          inspectedAgent
            ? formatModelForAgent(overview?.llm_provider, inspectedAgent.role)
            : 'google/gemini-2.5-flash'
        }
        onClose={() => setInspectedAgent(null)}
        onLaunchRun={() => setActiveNav('runs')}
      />

      {/* Editorial Toast Notification */}
      {toastMsg && (
        <div className="fixed bottom-5 right-5 z-50 px-4 py-2.5 rounded-full bg-[#171715] text-white dark:bg-white dark:text-[#171715] text-xs font-medium shadow-xl flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#4ADE80]" />
          <span>{toastMsg}</span>
        </div>
      )}

      {/* Global ⌘K Command Palette */}
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        actions={paletteActions}
      />
    </div>
  );
}

function OverviewStatCard({
  label,
  value,
  sub,
  subTone,
  icon: Icon,
  onClick,
}: {
  label: string;
  value: number;
  sub: string;
  subTone: 'green' | 'ochre' | 'muted';
  icon: any;
  onClick: () => void;
}) {
  const subCls =
    subTone === 'green'
      ? 'text-[#3D6E4C] dark:text-emerald-400'
      : subTone === 'ochre'
        ? 'text-[#9C6B19] dark:text-amber-400'
        : 'ide-ink-muted';

  return (
    <button
      type="button"
      onClick={onClick}
      className="text-left ide-surface border ide-border hover:ide-border-strong rounded-2xl p-5 flex flex-col justify-between min-h-[146px] transition group"
    >
      <div className="w-full flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-[0.06em] ide-ink-muted font-medium">
          {label}
        </span>
        <Icon className="w-4 h-4 ide-ink-muted" />
      </div>

      <div className="my-2">
        <div className="text-[30px] leading-none font-bold ide-ink tabular-nums">{value}</div>
        <div className={`text-xs mt-1.5 ${subCls}`}>{sub}</div>
      </div>

      <div className="w-full flex justify-end">
        <span className="text-xs ide-ink-muted group-hover:ide-ink transition inline-flex items-center gap-1">
          View all <ArrowRight className="w-3 h-3" />
        </span>
      </div>
    </button>
  );
}
