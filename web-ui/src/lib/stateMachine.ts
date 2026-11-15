'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Minimal explicit state machine, mirroring Go's pkg/state.
 * Transitions are data; invalid transitions are rejected (and logged in dev)
 * instead of silently corrupting component state.
 */
export interface MachineDef<S extends string> {
  name: string;
  initial: S;
  transitions: Record<S, readonly S[]>;
  /** Auto-transitions: after entering `from`, move to `to` after `ms`. */
  after?: Partial<Record<S, { to: S; ms: number }>>;
}

export function canTransition<S extends string>(def: MachineDef<S>, from: S, to: S): boolean {
  return (def.transitions[from] ?? []).includes(to);
}

export function isTerminal<S extends string>(def: MachineDef<S>, s: S): boolean {
  return (def.transitions[s] ?? []).length === 0;
}

/**
 * React hook driving a component through a machine definition.
 * `send(to)` applies a transition if legal and returns whether it was applied.
 * Timed auto-transitions declared in `def.after` fire automatically.
 */
export function useMachine<S extends string>(def: MachineDef<S>) {
  const [current, setCurrent] = useState<S>(def.initial);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const send = useCallback(
    (to: S): boolean => {
      let applied = false;
      setCurrent(from => {
        if (!canTransition(def, from, to)) {
          if (process.env.NODE_ENV !== 'production') {
            console.warn(`${def.name}: invalid transition ${from} -> ${to}`);
          }
          return from;
        }
        applied = true;
        return to;
      });
      return applied;
    },
    [def],
  );

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

  return { current, send };
}

/** Launch-button lifecycle, shared by every LaunchPad feature card. */
export type LaunchState = 'idle' | 'loading' | 'success' | 'error';

export const launchMachine: MachineDef<LaunchState> = {
  name: 'launch',
  initial: 'idle',
  transitions: {
    idle: ['loading'],
    loading: ['success', 'error'],
    success: ['idle'], // auto-reset so the button is reusable
    error: ['idle', 'loading'], // retry directly, or recover to idle
  },
  after: {
    success: { to: 'idle', ms: 2500 },
    error: { to: 'idle', ms: 4000 },
  },
};

export { statusBadge } from './statusBadge';
