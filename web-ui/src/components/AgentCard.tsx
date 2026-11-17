'use client';

import Link from 'next/link';

interface AgentCardProps {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
}

const colorMap: Record<string, { bg: string; border: string; icon: string }> = {
  blue: {
    bg: 'bg-blue-50 dark:bg-blue-900/20 hover:bg-blue-100 dark:hover:bg-blue-900/40',
    border: 'border-blue-200 dark:border-blue-800',
    icon: 'text-blue-600 dark:text-blue-400',
  },
  green: {
    bg: 'bg-green-50 dark:bg-green-900/20 hover:bg-green-100 dark:hover:bg-green-900/40',
    border: 'border-green-200 dark:border-green-800',
    icon: 'text-green-600 dark:text-green-400',
  },
  purple: {
    bg: 'bg-purple-50 dark:bg-purple-900/20 hover:bg-purple-100 dark:hover:bg-purple-900/40',
    border: 'border-purple-200 dark:border-purple-800',
    icon: 'text-purple-600 dark:text-purple-400',
  },
  amber: {
    bg: 'bg-amber-50 dark:bg-amber-900/20 hover:bg-amber-100 dark:hover:bg-amber-900/40',
    border: 'border-amber-200 dark:border-amber-800',
    icon: 'text-amber-600 dark:text-amber-400',
  },
};

export function AgentCard({ id, name, description, icon, color }: AgentCardProps) {
  const colors = colorMap[color] || colorMap.blue;

  return (
    <Link href={`/agents/${id}`}>
      <div
        className={`group block h-full ${colors.bg} border-2 ${colors.border} rounded-xl p-6 transition-all duration-300 cursor-pointer`}
      >
        <div className="flex items-start gap-4 mb-4">
          <div className={`text-4xl`}>{icon}</div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Agent {id}
            </h3>
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{name}</p>
          </div>
        </div>

        <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">{description}</p>

        <div className="inline-flex items-center gap-2 px-3 py-2 bg-white/50 dark:bg-gray-800/50 rounded-lg text-sm font-medium text-gray-900 dark:text-gray-100 group-hover:bg-white dark:group-hover:bg-gray-800 transition-colors">
          Open Agent
          <svg
            className="w-4 h-4 group-hover:translate-x-1 transition-transform"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5l7 7-7 7"
            />
          </svg>
        </div>
      </div>
    </Link>
  );
}
