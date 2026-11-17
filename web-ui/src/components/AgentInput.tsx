'use client';

interface AgentInputProps {
  value: string;
  onChange: (value: string) => void;
  onRun: () => void;
  disabled?: boolean;
}

export function AgentInput({ value, onChange, onRun, disabled }: AgentInputProps) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 h-full flex flex-col">
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">LLD Input</h2>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder="Paste Low-Level Design YAML..."
        className="flex-1 p-4 bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-gray-100 rounded-lg border border-gray-300 dark:border-gray-600 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none resize-none"
      />
      <button
        onClick={onRun}
        disabled={disabled || !value.trim()}
        className="w-full mt-4 px-4 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
      >
        {disabled ? 'Running...' : 'Run Agent'}
      </button>
    </div>
  );
}
