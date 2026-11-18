/**
 * Shared semantic mapping from any explicit backend or frontend state to
 * soft sage / amber / slate / rose pills matching the warm editorial control plane.
 */
export const statusBadge: Record<string, string> = {
  // Healthy / Terminal positive states (soft sage pill)
  healthy: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  success: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  completed: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  passed: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  succeeded: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  promoted: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  approved: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  correct: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  online: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  connected: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  done: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',
  active: 'bg-[#DCE9DF] text-[#285936] border border-[#C3DAC8] dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30',

  // Active / In-progress states (soft slate-blue pill)
  running: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  executing: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  coordinating: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  synthesizing: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  verifying: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  loading: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',
  busy: 'bg-[#DCE6F5] text-[#1E4785] border border-[#C2D4EE] dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30',

  // Gate / Human review / Queued states (soft warm ochre pill)
  pending: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  queued: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  awaiting_approval: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  waiting_approval: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  awaiting_gate: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  retrying: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  repairing: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',
  degraded: 'bg-[#F5E8CE] text-[#7A5213] border border-[#E6D0A3] dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30',

  // Error / Rejected states (soft terracotta-rose pill)
  failed: 'bg-[#F5DCDC] text-[#7D2626] border border-[#E8BEBE] dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30',
  error: 'bg-[#F5DCDC] text-[#7D2626] border border-[#E8BEBE] dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30',
  rejected: 'bg-[#F5DCDC] text-[#7D2626] border border-[#E8BEBE] dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30',
  exhausted: 'bg-[#F5DCDC] text-[#7D2626] border border-[#E8BEBE] dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30',
  offline: 'bg-[#F5DCDC] text-[#7D2626] border border-[#E8BEBE] dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30',

  // Neutral / Idle states
  idle: 'bg-[#ECEAE2] text-[#575650] border border-[#DCD8CC] dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700',
  disabled: 'bg-[#ECEAE2] text-[#78766E] border border-[#DCD8CC] dark:bg-zinc-800 dark:text-zinc-400 dark:border-zinc-700',
};
