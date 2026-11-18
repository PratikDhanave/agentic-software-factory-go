'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Explicit State Machine Engine for UX, DX, and AX — mirroring Go's `pkg/state`.
 * Every lifecycle is declared as pure data: states, initial state, allowed transitions,
 * and plain-English descriptions. Invalid transitions are rejected deterministically
 * instead of silently creating impossible boolean combinations.
 */
export interface MachineDef<S extends string> {
  name: string;
  category?: 'ui' | 'assembly' | 'governance' | 'orchestration' | 'execution' | 'verification' | 'infrastructure';
  description?: string;
  initial: S;
  transitions: Record<S, readonly S[]>;
  stateDescriptions?: Partial<Record<S, string>>;
  /** Optional timed auto-transitions: after entering `from`, transition to `to` after `ms`. */
  after?: Partial<Record<S, { to: S; ms: number }>>;
}

export interface MachineExport {
  name: string;
  category?: string;
  description?: string;
  initial: string;
  states: string[];
  terminal: string[];
  transitions: Record<string, string[]>;
  state_descriptions?: Record<string, string>;
  mermaid?: string;
  source?: 'backend' | 'frontend';
}

export function canTransition<S extends string>(def: MachineDef<S>, from: S, to: S): boolean {
  return (def.transitions[from] ?? []).includes(to);
}

export function allowedNext<S extends string>(def: MachineDef<S>, from: S): readonly S[] {
  return def.transitions[from] ?? [];
}

export function isTerminal<S extends string>(def: MachineDef<S>, s: S): boolean {
  return (def.transitions[s] ?? []).length === 0;
}

export function exportMachineDef<S extends string>(def: MachineDef<S>): MachineExport {
  const seen = new Set<string>([def.initial]);
  const transitionsCopy: Record<string, string[]> = {};

  for (const [from, targets] of Object.entries(def.transitions) as Array<[S, readonly S[]]>) {
    seen.add(from);
    transitionsCopy[from] = [...targets];
    for (const t of targets) seen.add(t);
  }

  const states = Array.from(seen).sort();
  const terminal = states.filter((s) => (transitionsCopy[s] ?? []).length === 0);

  const lines = ['stateDiagram-v2', `    [*] --> ${def.initial}`];
  for (const from of Object.keys(transitionsCopy).sort()) {
    for (const to of transitionsCopy[from]) {
      lines.push(`    ${from} --> ${to}`);
    }
  }
  for (const term of terminal) {
    lines.push(`    ${term} --> [*]`);
  }

  return {
    name: def.name,
    category: def.category || 'ui',
    description: def.description,
    initial: def.initial,
    states,
    terminal,
    transitions: transitionsCopy,
    state_descriptions: def.stateDescriptions as Record<string, string> | undefined,
    mermaid: lines.join('\n'),
    source: 'frontend',
  };
}

export interface TransitionRecord<S extends string> {
  from: S;
  to: S;
  at: number;
}

/**
 * React hook driving a component through an explicit MachineDef.
 */
export function useMachine<S extends string>(def: MachineDef<S>) {
  const [current, setCurrent] = useState<S>(def.initial);
  const [history, setHistory] = useState<TransitionRecord<S>[]>([]);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const send = useCallback(
    (to: S): boolean => {
      let applied = false;
      setCurrent((from) => {
        if (from === to) {
          applied = true;
          return from;
        }
        if (!canTransition(def, from, to)) {
          if (process.env.NODE_ENV !== 'production') {
            console.warn(`${def.name}: invalid transition ${from} -> ${to}`);
          }
          return from;
        }
        applied = true;
        setHistory((prev) => [...prev.slice(-19), { from, to, at: Date.now() }]);
        return to;
      });
      return applied;
    },
    [def]
  );

  const reset = useCallback(() => {
    setCurrent(def.initial);
    setHistory([]);
  }, [def.initial]);

  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    const auto = def.after?.[current];
    if (auto) {
      timer.current = setTimeout(() => send(auto.to), auto.ms);
    }
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [current, def, send]);

  return {
    current,
    send,
    reset,
    can: useCallback((to: S) => canTransition(def, current, to), [def, current]),
    allowed: allowedNext(def, current),
    isTerminal: isTerminal(def, current),
    history,
  };
}

// ============================================================================
// EXPLICIT FRONTEND UI STATE MACHINES
// ============================================================================

/** 1. Launch-button lifecycle, shared by LaunchPad cards. */
export type LaunchState = 'idle' | 'loading' | 'success' | 'error';

export const launchMachine: MachineDef<LaunchState> = {
  name: 'ui-launch-action',
  category: 'ui',
  description: 'Single-action button lifecycle with automatic recovery from success and error states.',
  initial: 'idle',
  transitions: {
    idle: ['loading'],
    loading: ['success', 'error'],
    success: ['idle'],
    error: ['idle', 'loading'],
  },
  stateDescriptions: {
    idle: 'Ready to trigger task.',
    loading: 'Submitting request to backend.',
    success: 'Accepted; auto-resets after confirmation.',
    error: 'Request failed; retryable immediately.',
  },
  after: {
    success: { to: 'idle', ms: 2500 },
    error: { to: 'idle', ms: 4000 },
  },
};

/** 2. Control Plane telemetry connection health machine (replaces `boolean | null`). */
export type ConnectionHealthState = 'connecting' | 'online' | 'offline' | 'reconnecting';

export const connectionHealthMachine: MachineDef<ConnectionHealthState> = {
  name: 'ui-connection-health',
  category: 'ui',
  description: 'Tracks live telemetry connectivity between the browser IDE and the Go control plane.',
  initial: 'connecting',
  transitions: {
    connecting: ['online', 'offline'],
    online: ['offline', 'reconnecting'],
    offline: ['reconnecting', 'online'],
    reconnecting: ['online', 'offline'],
  },
  stateDescriptions: {
    connecting: 'Performing initial handshake with /api/system/overview.',
    online: 'Live telemetry connected to SQLite and worker pool.',
    offline: 'Backend unreachable; displaying startup helper command.',
    reconnecting: 'Polling to restore connection.',
  },
};

/** 3. Autonomy mode switcher lifecycle. */
export type PhaseSwitchState = 'idle' | 'switching' | 'error';

export const phaseSwitchMachine: MachineDef<PhaseSwitchState> = {
  name: 'ui-autonomy-switch',
  category: 'ui',
  description: 'Serializes Crawl / Walk / Run autonomy phase updates to prevent concurrent race clicks.',
  initial: 'idle',
  transitions: {
    idle: ['switching'],
    switching: ['idle', 'error'],
    error: ['idle', 'switching'],
  },
  after: {
    error: { to: 'idle', ms: 3000 },
  },
};

/** 4. Factory Workbench mutation lifecycle (replaces `starting` + `approving` + `promoting` booleans). */
export type WorkbenchActionState =
  | 'idle'
  | 'starting_build'
  | 'approving_gate'
  | 'promoting_package'
  | 'succeeded'
  | 'failed';

export const workbenchActionMachine: MachineDef<WorkbenchActionState> = {
  name: 'ui-workbench-command',
  category: 'ui',
  description: 'Ensures mutually exclusive Factory Workbench commands (start build, approve gate, promote package).',
  initial: 'idle',
  transitions: {
    idle: ['starting_build', 'approving_gate', 'promoting_package'],
    starting_build: ['succeeded', 'failed'],
    approving_gate: ['succeeded', 'failed'],
    promoting_package: ['succeeded', 'failed'],
    succeeded: ['idle', 'starting_build', 'approving_gate', 'promoting_package'],
    failed: ['idle', 'starting_build', 'approving_gate', 'promoting_package'],
  },
  stateDescriptions: {
    idle: 'Ready for operator command.',
    starting_build: 'Creating FactoryRun and opening SSE stream.',
    approving_gate: 'Submitting human approval and resuming Stages 3-6.',
    promoting_package: 'Exporting verified Go source files into generated/<slug>/.',
    succeeded: 'Command completed cleanly.',
    failed: 'Command returned an error.',
  },
  after: {
    succeeded: { to: 'idle', ms: 3500 },
  },
};

/** 5. Multi-Agent Workflow Studio lifecycle (replaces `loading` + `executing` + `result` + `error`). */
export type WorkflowStudioState =
  | 'idle'
  | 'validation_error'
  | 'coordinating'
  | 'awaiting_gate'
  | 'planned'
  | 'executing_dag'
  | 'executed'
  | 'error';

export const workflowStudioMachine: MachineDef<WorkflowStudioState> = {
  name: 'ui-workflow-studio',
  category: 'ui',
  description: 'Drives 10-agent DAG synthesis, human governance pause, and manual DAG execution.',
  initial: 'idle',
  transitions: {
    idle: ['coordinating', 'validation_error'],
    validation_error: ['idle', 'coordinating', 'validation_error'],
    coordinating: ['awaiting_gate', 'planned', 'executed', 'error'],
    awaiting_gate: ['coordinating', 'executing_dag', 'idle'],
    planned: ['executing_dag', 'coordinating', 'idle'],
    executing_dag: ['executed', 'error'],
    executed: ['coordinating', 'executing_dag', 'idle'],
    error: ['coordinating', 'executing_dag', 'idle'],
  },
  stateDescriptions: {
    idle: 'Waiting for engineering issue title and acceptance criteria.',
    validation_error: 'Form missing required title or description.',
    coordinating: '10 specialized agents synthesizing decisions and topological DAG.',
    awaiting_gate: 'Blocked on Human-in-the-Loop approval gate.',
    planned: 'DAG synthesized and ready to execute.',
    executing_dag: 'Running topological DAG levels.',
    executed: 'DAG execution finished with node outputs.',
    error: 'API error occurred; retryable.',
  },
};

/** 6. Governance Approvals Inbox lifecycle. */
export type ApprovalsInboxState = 'loading' | 'empty' | 'ready' | 'submitting' | 'error';

export const approvalsInboxMachine: MachineDef<ApprovalsInboxState> = {
  name: 'ui-approvals-inbox',
  category: 'ui',
  description: 'Governs fetching pending HITL gates and submitting human Approve/Reject decisions.',
  initial: 'loading',
  transitions: {
    loading: ['empty', 'ready', 'error'],
    empty: ['loading', 'ready', 'error'],
    ready: ['submitting', 'empty', 'loading', 'error'],
    submitting: ['ready', 'empty', 'error'],
    error: ['loading', 'ready', 'empty'],
  },
};

/** 7. Spec Studio & Isolated Harness lifecycle. */
export type SpecHarnessState =
  | 'idle'
  | 'creating_spec'
  | 'launching_task'
  | 'verifying_repo'
  | 'succeeded'
  | 'error';

export const specHarnessMachine: MachineDef<SpecHarnessState> = {
  name: 'ui-spec-harness',
  category: 'ui',
  description: 'Coordinates YAML spec authoring, isolated worker task dispatch, and repository verification.',
  initial: 'idle',
  transitions: {
    idle: ['creating_spec', 'launching_task', 'verifying_repo'],
    creating_spec: ['succeeded', 'error'],
    launching_task: ['succeeded', 'error'],
    verifying_repo: ['succeeded', 'error'],
    succeeded: ['idle', 'creating_spec', 'launching_task', 'verifying_repo'],
    error: ['idle', 'creating_spec', 'launching_task', 'verifying_repo'],
  },
  after: {
    succeeded: { to: 'idle', ms: 2000 },
  },
};

/** 8. 7-Layer SSE Client Stream lifecycle (mirrors Go's `seven-layer-stream`). */
export type SevenLayerClientState =
  | 'idle'
  | 'start'
  | 'layer-1'
  | 'layer-2'
  | 'layer-3'
  | 'layer-4'
  | 'layer-5'
  | 'layer-6'
  | 'layer-7'
  | 'done'
  | 'error';

export const sevenLayerClientMachine: MachineDef<SevenLayerClientState> = {
  name: 'ui-seven-layer-stream',
  category: 'ui',
  description: 'Client-side mirror of the 7-layer SSE pipeline ensuring ordered progressive rendering.',
  initial: 'idle',
  transitions: {
    idle: ['start', 'layer-1', 'error'],
    start: ['layer-1', 'error'],
    'layer-1': ['layer-2', 'error'],
    'layer-2': ['layer-3', 'error'],
    'layer-3': ['layer-4', 'error'],
    'layer-4': ['layer-5', 'error'],
    'layer-5': ['layer-6', 'error'],
    'layer-6': ['layer-7', 'error'],
    'layer-7': ['done', 'error'],
    done: ['idle', 'start'],
    error: ['idle', 'start'],
  },
};

/** 9. Continuous Learning Lab lifecycle. */
export type LearningLabState = 'loading' | 'ready' | 'simulating' | 'calibrating' | 'error';

export const learningLabMachine: MachineDef<LearningLabState> = {
  name: 'ui-learning-lab',
  category: 'ui',
  description: 'Governs SQLite learning telemetry fetch, synthetic batch simulation, and pattern discovery.',
  initial: 'loading',
  transitions: {
    loading: ['ready', 'error'],
    ready: ['simulating', 'calibrating', 'loading', 'error'],
    simulating: ['ready', 'error'],
    calibrating: ['ready', 'error'],
    error: ['loading', 'ready', 'simulating', 'calibrating'],
  },
};

export const FRONTEND_MACHINES: MachineExport[] = [
  exportMachineDef(connectionHealthMachine),
  exportMachineDef(phaseSwitchMachine),
  exportMachineDef(workbenchActionMachine),
  exportMachineDef(workflowStudioMachine),
  exportMachineDef(approvalsInboxMachine),
  exportMachineDef(specHarnessMachine),
  exportMachineDef(sevenLayerClientMachine),
  exportMachineDef(learningLabMachine),
  exportMachineDef(launchMachine),
];

export { statusBadge } from './statusBadge';
