'use client';

import { useState, useCallback } from 'react';

export interface AgentResult {
  id: string;
  generatedCode: string;
  score: number;
  testsPassed: number;
  acceptanceMet: boolean;
  safetyViolations: string[];
  executionTime: number;
  createdAt: string;
}

export interface ExecutionStatus {
  executionId: string;
  status: string;
  progress: number;
  message: string;
  result?: AgentResult;
  error?: string;
}

export function useAgentExecution(agentId: string) {
  const [executing, setExecuting] = useState(false);
  const [status, setStatus] = useState<ExecutionStatus | null>(null);
  const [result, setResult] = useState<AgentResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const execute = useCallback(
    async (lldYaml: string) => {
      setExecuting(true);
      setStatus(null);
      setResult(null);
      setError(null);

      try {
        const response = await fetch(`/api/agents/run`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ agentId, lldYaml }),
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error('Response body is not readable');
        }

        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });

          // Process complete lines
          const lines = buffer.split('\n');
          buffer = lines.pop() || ''; // Keep incomplete line in buffer

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              try {
                const data = JSON.parse(line.slice(6)) as ExecutionStatus;
                setStatus(data);

                if (data.status === 'done' && data.result) {
                  setResult(data.result);
                } else if (data.status === 'error' && data.error) {
                  setError(data.error);
                }
              } catch (e) {
                console.error('Failed to parse SSE data:', e);
              }
            }
          }
        }
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Unknown error';
        setError(errorMsg);
        console.error('Execution error:', err);
      } finally {
        setExecuting(false);
      }
    },
    [agentId]
  );

  return { execute, status, result, error, executing };
}
