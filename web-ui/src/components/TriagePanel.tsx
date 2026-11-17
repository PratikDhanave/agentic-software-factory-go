'use client';

import { useState, useEffect } from 'react';
import { AlertCircle, Check, Loader, ExternalLink, Copy, Zap } from 'lucide-react';

interface TriageResult {
  id: string;
  issue: number;
  repo: string;
  title: string;
  category: string;
  priority: string;
  ready_to_implement: boolean;
  reason: string;
  labels_to_apply: string[];
  generated_spec?: string;
  created_at: string;
}

interface TriageStatus {
  status: string;
  progress: number;
  message: string;
}

const COLORS = {
  feature: 'bg-blue-50 text-blue-700 border-blue-200',
  bug: 'bg-red-50 text-red-700 border-red-200',
  refactor: 'bg-purple-50 text-purple-700 border-purple-200',
  documentation: 'bg-amber-50 text-amber-700 border-amber-200',
};

const PRIORITY_COLORS = {
  critical: 'bg-red-50 text-red-700 border-red-200',
  high: 'bg-orange-50 text-orange-700 border-orange-200',
  medium: 'bg-blue-50 text-blue-700 border-blue-200',
  low: 'bg-gray-50 text-gray-700 border-gray-200',
};

export default function TriagePanel() {
  const [owner, setOwner] = useState('');
  const [repo, setRepo] = useState('');
  const [issue, setIssue] = useState('');
  const [token, setToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<TriageResult | null>(null);
  const [error, setError] = useState('');
  const [showSpec, setShowSpec] = useState(false);
  const [copied, setCopied] = useState(false);

  // Load token from localStorage on mount
  useEffect(() => {
    const stored = localStorage.getItem('github_token');
    if (stored) {
      setToken(stored);
    }
  }, []);

  const handleRunTriage = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!owner || !repo || !issue) {
      setError('Owner, Repository, and Issue Number are required');
      return;
    }

    if (!token) {
      setError('GitHub token not loaded. Set GITHUB_TOKEN environment variable.');
      return;
    }

    setError('');
    setLoading(true);
    setProgress(0);
    setResult(null);

    try {
      const url = new URL('/api/github/triage', window.location.origin);
      url.searchParams.append('owner', owner);
      url.searchParams.append('repo', repo);
      url.searchParams.append('issue', issue);
      url.searchParams.append('token', token);

      const response = await fetch(url.toString());

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let buffer = '';
      let lastResult: TriageResult | null = null;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));

              if (data.status === 'error') {
                throw new Error(data.message);
              }

              setProgress(data.progress);

              // Check if this is a final result (contains triage data)
              if (data.id) {
                lastResult = data;
              }
            } catch (e) {
              // JSON parse errors are ok for SSE
            }
          }
        }
      }

      if (lastResult) {
        setResult(lastResult);
      }
      setProgress(100);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setProgress(0);
    } finally {
      setLoading(false);
    }
  };

  const handleCopySpec = async () => {
    if (result?.generated_spec) {
      await navigator.clipboard.writeText(result.generated_spec);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border-b border-gray-200 px-6 py-6">
        <div className="flex items-center gap-3 mb-2">
          <Zap className="w-5 h-5 text-blue-600" />
          <h3 className="text-lg font-semibold text-gray-900">GitHub Issue Triage</h3>
        </div>
        <p className="text-sm text-gray-600">
          Analyze and categorize GitHub issues automatically
        </p>
      </div>

      {/* Content */}
      <div className="p-6">
        <form onSubmit={handleRunTriage} className="space-y-6">
          {/* Input Section */}
          <div className="space-y-4">
            {/* Row 1: Owner, Repo, Issue */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  GitHub Owner
                </label>
                <input
                  type="text"
                  value={owner}
                  onChange={(e) => setOwner(e.target.value)}
                  placeholder="username"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Repository
                </label>
                <input
                  type="text"
                  value={repo}
                  onChange={(e) => setRepo(e.target.value)}
                  placeholder="repo-name"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Issue Number
                </label>
                <input
                  type="number"
                  value={issue}
                  onChange={(e) => setIssue(e.target.value)}
                  placeholder="1"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  required
                />
              </div>
            </div>

            {/* Token Status & Input */}
            <div>
              {token ? (
                <div className="flex items-center justify-between gap-2 p-3 bg-green-50 border border-green-200 rounded-lg">
                  <div className="flex items-center gap-2">
                    <Check className="w-4 h-4 text-green-600" />
                    <span className="text-sm text-green-700 font-medium">✓ GitHub token loaded</span>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setToken('');
                      localStorage.removeItem('github_token');
                    }}
                    className="text-xs text-green-600 hover:text-green-700 font-medium underline"
                  >
                    Clear
                  </button>
                </div>
              ) : (
                <div className="space-y-2">
                  <label className="block text-sm font-medium text-gray-700">
                    GitHub Token (Personal Access Token)
                  </label>
                  <input
                    type="password"
                    placeholder="ghp_..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    onChange={(e) => {
                      const newToken = e.target.value;
                      setToken(newToken);
                      if (newToken) {
                        localStorage.setItem('github_token', newToken);
                      }
                    }}
                  />
                  <p className="text-xs text-gray-500">
                    Get token at: <a href="https://github.com/settings/tokens" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">github.com/settings/tokens</a>
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Error Message */}
          {error && (
            <div className="flex gap-3 p-3 bg-red-50 border border-red-200 rounded-lg">
              <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {/* Progress Bar */}
          {loading && (
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium text-gray-700">Analyzing...</span>
                <span className="text-sm font-semibold text-blue-600">{progress}%</span>
              </div>
              <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className="h-full bg-blue-600 transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}

          {/* Submit Button */}
          <button
            type="submit"
            disabled={loading}
            className="w-full px-4 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-medium rounded-lg transition flex items-center justify-center gap-2"
          >
            {loading ? (
              <>
                <Loader className="w-4 h-4 animate-spin" />
                Analyzing...
              </>
            ) : (
              <>
                <Zap className="w-4 h-4" />
                Run Triage
              </>
            )}
          </button>
        </form>

        {/* Results Section */}
        {result && (
          <div className="mt-8 pt-8 border-t border-gray-200 space-y-6">
            {/* Results Header */}
            <div className="flex items-center gap-2">
              <Check className="w-5 h-5 text-green-600" />
              <h4 className="font-semibold text-gray-900">Analysis Results</h4>
            </div>

            {/* Results Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Category Badge */}
              <div className={`p-4 rounded-lg border ${COLORS[result.category as keyof typeof COLORS]}`}>
                <p className="text-xs font-semibold uppercase tracking-wide opacity-75 mb-1">
                  Category
                </p>
                <p className="text-sm font-semibold capitalize">{result.category}</p>
              </div>

              {/* Priority Badge */}
              <div className={`p-4 rounded-lg border ${PRIORITY_COLORS[result.priority as keyof typeof PRIORITY_COLORS]}`}>
                <p className="text-xs font-semibold uppercase tracking-wide opacity-75 mb-1">
                  Priority
                </p>
                <p className="text-sm font-semibold capitalize">{result.priority}</p>
              </div>

              {/* Ready Status */}
              <div className={`p-4 rounded-lg border ${result.ready_to_implement ? 'bg-green-50 text-green-700 border-green-200' : 'bg-amber-50 text-amber-700 border-amber-200'}`}>
                <p className="text-xs font-semibold uppercase tracking-wide opacity-75 mb-1">
                  Ready to Implement
                </p>
                <div className="flex items-center gap-2">
                  {result.ready_to_implement ? (
                    <>
                      <Check className="w-4 h-4" />
                      <span className="text-sm font-semibold">Yes</span>
                    </>
                  ) : (
                    <>
                      <AlertCircle className="w-4 h-4" />
                      <span className="text-sm font-semibold">No</span>
                    </>
                  )}
                </div>
              </div>

              {/* Labels */}
              <div className="p-4 rounded-lg border border-gray-200 bg-gray-50">
                <p className="text-xs font-semibold uppercase tracking-wide text-gray-600 opacity-75 mb-2">
                  Labels to Apply
                </p>
                <div className="flex flex-wrap gap-2">
                  {result.labels_to_apply.map((label) => (
                    <span
                      key={label}
                      className="px-2.5 py-1 bg-white border border-gray-300 text-gray-700 text-xs font-medium rounded"
                    >
                      {label}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            {/* Analysis Reason */}
            <div className="p-4 rounded-lg border border-gray-200 bg-gray-50">
              <p className="text-xs font-semibold uppercase tracking-wide text-gray-600 opacity-75 mb-2">
                Analysis
              </p>
              <p className="text-sm text-gray-700">{result.reason}</p>
            </div>

            {/* Generated Spec */}
            {result.generated_spec && (
              <div className="p-4 rounded-lg border border-gray-200 bg-gray-50">
                <button
                  onClick={() => setShowSpec(!showSpec)}
                  className="flex items-center justify-between w-full mb-3"
                >
                  <p className="text-xs font-semibold uppercase tracking-wide text-gray-600 opacity-75">
                    Generated Implementation Spec
                  </p>
                  <span className="text-xs text-blue-600 font-medium hover:underline">
                    {showSpec ? 'Hide' : 'Show'}
                  </span>
                </button>
                {showSpec && (
                  <pre className="p-3 bg-white rounded border border-gray-300 text-xs text-gray-700 overflow-x-auto max-h-48 overflow-y-auto">
                    {result.generated_spec}
                  </pre>
                )}
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex gap-3 pt-2">
              <a
                href={`https://github.com/${result.repo}/issues/${result.issue}`}
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-900 rounded-lg text-sm font-medium transition"
              >
                <ExternalLink className="w-4 h-4" />
                View on GitHub
              </a>
              {result.ready_to_implement && result.generated_spec && (
                <button
                  onClick={handleCopySpec}
                  className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition"
                >
                  <Copy className="w-4 h-4" />
                  {copied ? 'Copied!' : 'Copy Spec'}
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
