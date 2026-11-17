'use client';

interface ScoreDisplayProps {
  score: number;
}

export function ScoreDisplay({ score }: ScoreDisplayProps) {
  const getScoreColor = (s: number) => {
    if (s >= 85) return 'from-green-500 to-emerald-600';
    if (s >= 75) return 'from-blue-500 to-cyan-600';
    if (s >= 60) return 'from-yellow-500 to-orange-600';
    return 'from-red-500 to-pink-600';
  };

  const getScoreBadge = (s: number) => {
    if (s >= 85) return { text: '✅ Excellent', bg: 'bg-green-100 dark:bg-green-900' };
    if (s >= 75) return { text: '✅ Passed', bg: 'bg-blue-100 dark:bg-blue-900' };
    if (s >= 60) return { text: '⚠️ Acceptable', bg: 'bg-yellow-100 dark:bg-yellow-900' };
    return { text: '❌ Below Threshold', bg: 'bg-red-100 dark:bg-red-900' };
  };

  const badge = getScoreBadge(score);

  return (
    <div className={`bg-gradient-to-br ${getScoreColor(score)} rounded-xl p-6 text-white shadow-lg`}>
      <h2 className="text-sm font-semibold opacity-90 mb-2">Evaluation Score</h2>
      <div className="text-5xl font-bold mb-2">{score.toFixed(1)}</div>
      <div className="text-sm opacity-90 mb-4">/100</div>

      <div className={`${badge.bg} text-gray-900 dark:text-gray-100 px-3 py-1 rounded-full text-sm font-medium inline-block`}>
        {badge.text}
      </div>

      {/* Score breakdown */}
      <div className="mt-4 grid grid-cols-2 gap-2 text-sm">
        <div className="opacity-90">
          <div className="text-xs opacity-75">Tests</div>
          <div className="font-semibold">{Math.round(score * 0.4)}/40</div>
        </div>
        <div className="opacity-90">
          <div className="text-xs opacity-75">Acceptance</div>
          <div className="font-semibold">{Math.round(score * 0.3)}/30</div>
        </div>
        <div className="opacity-90">
          <div className="text-xs opacity-75">Quality</div>
          <div className="font-semibold">{Math.round(score * 0.15)}/15</div>
        </div>
        <div className="opacity-90">
          <div className="text-xs opacity-75">Safety</div>
          <div className="font-semibold">{Math.round(score * 0.15)}/15</div>
        </div>
      </div>
    </div>
  );
}
