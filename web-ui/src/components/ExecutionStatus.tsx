'use client';

interface ExecutionStatusProps {
  status: string;
  progress: number;
  message: string;
}

const STEP_LABELS: Record<string, { label: string; color: string }> = {
  initializing: { label: 'Initializing', color: 'bg-gray-400' },
  'llm-initializing': { label: 'Loading LLM', color: 'bg-yellow-500' },
  'llm-generating': { label: 'Generating Code', color: 'bg-blue-500' },
  evaluating: { label: 'Evaluating', color: 'bg-purple-500' },
  done: { label: 'Complete', color: 'bg-green-500' },
  error: { label: 'Error', color: 'bg-red-500' },
};

export function ExecutionStatus({ status, progress, message }: ExecutionStatusProps) {
  const stepInfo = STEP_LABELS[status] || { label: status, color: 'bg-gray-400' };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        {stepInfo.label}
      </h2>

      {/* Progress bar */}
      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 mb-4 overflow-hidden">
        <div
          className={`${stepInfo.color} h-3 rounded-full transition-all duration-500`}
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* Progress text */}
      <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
        {progress}% • {message}
      </p>

      {/* Status indicators */}
      <div className="space-y-2">
        {[
          { key: 'initializing', label: 'Initializing' },
          { key: 'llm-initializing', label: 'Loading LLM' },
          { key: 'llm-generating', label: 'Generating' },
          { key: 'evaluating', label: 'Evaluating' },
          { key: 'done', label: 'Done' },
        ].map((step) => {
          const isActive = step.key === status;
          const isDone =
            ['initializing', 'llm-initializing', 'llm-generating', 'evaluating'].includes(
              step.key
            ) &&
            (['llm-initializing', 'llm-generating', 'evaluating', 'done'].includes(status) ||
              (status === 'done' && step.key !== 'error'));

          return (
            <div key={step.key} className="flex items-center gap-3">
              <div
                className={`w-3 h-3 rounded-full ${
                  isDone || isActive ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              />
              <span
                className={`text-sm ${
                  isDone || isActive
                    ? 'text-gray-900 dark:text-gray-100 font-medium'
                    : 'text-gray-500 dark:text-gray-400'
                }`}
              >
                {step.label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
