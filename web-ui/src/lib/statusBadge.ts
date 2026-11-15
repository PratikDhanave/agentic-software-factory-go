/**
 * One shared mapping from any lifecycle state to badge styling, usable from
 * both server and client components — add states here, not inline in JSX.
 * State names match the machines declared in Go (`asf factory states`).
 */
export const statusBadge: Record<string, string> = {
  success: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  completed: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  running: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  loading: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  error: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  idle: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
};
