'use client';

import { useState } from 'react';
import { AgentInput } from '@/components/AgentInput';
import { ExecutionStatus } from '@/components/ExecutionStatus';
import { ScoreDisplay } from '@/components/ScoreDisplay';
import { CodeViewer } from '@/components/CodeViewer';
import { useAgentExecution } from '@/lib/hooks/useAgentExecution';

interface PageProps {
  params: {
    id: string;
  };
}

export default function AgentPage({ params }: PageProps) {
  const { id } = params;
  const [lldYaml, setLldYaml] = useState(`# Low-Level Design: User Management Service
name: User Management
description: Implement user CRUD operations with validation

requirements:
  - Create, read, update, delete users
  - Validate email and password
  - Handle concurrent access safely
  - Return proper error messages

acceptanceCriteria:
  - Code compiles without errors
  - Unit tests pass
  - Thread-safe operations
  - Proper error handling
`);

  const { execute, status, result, error, executing } = useAgentExecution(id);

  const handleRun = () => {
    if (!lldYaml.trim()) {
      alert('Please enter LLD YAML');
      return;
    }
    execute(lldYaml);
  };

  return (
    <main className="min-h-screen bg-white text-gray-900 dark:bg-gray-900 dark:text-gray-100 antialiased">
      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Header */}
        <header className="mb-8 border-b border-gray-200 dark:border-gray-700 pb-6">
          <div className="flex flex-col gap-2 mb-4">
            <h1 className="text-4xl font-bold tracking-tighter">Agent {id}</h1>
            <p className="text-lg text-gray-600 dark:text-gray-400">Code Generation with Self-Evolution</p>
          </div>
          <p className="text-gray-700 dark:text-gray-300">
            Generate production-ready code using an 8-layer self-evolving agent that learns from each execution
          </p>
        </header>

        {/* Main Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left: Input */}
          <div className="lg:col-span-1">
            <AgentInput
              value={lldYaml}
              onChange={setLldYaml}
              onRun={handleRun}
              disabled={executing}
            />
          </div>

          {/* Right: Output */}
          <div className="lg:col-span-2 space-y-6">
            {/* Error Display */}
            {error && (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded-xl p-6">
                <h3 className="font-semibold text-red-900 dark:text-red-200 mb-2">Error</h3>
                <p className="text-red-800 dark:text-red-300 text-sm">{error}</p>
              </div>
            )}

            {/* Status Display */}
            {status && status.status !== 'done' && (
              <ExecutionStatus
                status={status.status}
                progress={status.progress}
                message={status.message}
              />
            )}

            {/* Results Display */}
            {result && (
              <>
                <ScoreDisplay score={result.score} />
                <CodeViewer code={result.generatedCode} />

                {/* Metadata */}
                <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
                  <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-4">Execution Details</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <div className="text-sm text-gray-600 dark:text-gray-400">Tests Passed</div>
                      <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                        {result.testsPassed}/10
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-600 dark:text-gray-400">Execution Time</div>
                      <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                        {result.executionTime}ms
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-600 dark:text-gray-400">Acceptance</div>
                      <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                        {result.acceptanceMet ? '✓ Met' : '✗ Not Met'}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-gray-600 dark:text-gray-400">Safety Violations</div>
                      <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                        {result.safetyViolations.length === 0 ? '✓ None' : result.safetyViolations.length}
                      </div>
                    </div>
                  </div>

                  {result.safetyViolations.length > 0 && (
                    <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                      <div className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
                        Safety Violations:
                      </div>
                      <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
                        {result.safetyViolations.map((v, i) => (
                          <li key={i} className="flex items-start gap-2">
                            <span className="text-red-500 mt-1">•</span>
                            <span>{v}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
                    Result ID: {result.id}
                  </div>
                </div>
              </>
            )}

            {/* Initial State Message */}
            {!status && !result && !error && (
              <div className="bg-gray-50 dark:bg-gray-800/50 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-600 p-12 text-center">
                <div className="text-5xl mb-4">⚙️</div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                  Ready to Generate
                </h3>
                <p className="text-gray-600 dark:text-gray-400">
                  Enter your specification and click "Run Agent" to generate production-ready code
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <footer className="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 text-center text-sm text-gray-600 dark:text-gray-400">
          <p>Agent Factory • 8-Layer Self-Evolving Agent • Powered by Factory Stack</p>
        </footer>
      </div>
    </main>
  );
}
