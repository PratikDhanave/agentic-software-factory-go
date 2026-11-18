'use client';

import { useState } from 'react';
import { apiClient, WorkflowResponse } from '@/lib/api';
import { AlertCircle, CheckCircle, Loader } from 'lucide-react';

export default function WorkflowGenerator() {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<WorkflowResponse | null>(null);
  const [error, setError] = useState('');

  const handleGenerate = async () => {
    if (!title.trim() || !description.trim()) {
      setError('Please fill in both title and description');
      return;
    }

    setLoading(true);
    setError('');
    setResult(null);

    try {
      const response = await apiClient.generateWorkflow({
        issue_title: title,
        issue_description: description,
      });
      setResult(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate workflow');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-md p-6 mb-6">
      <h2 className="text-2xl font-bold text-gray-900 mb-4">Generate Workflow</h2>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Issue Title
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="e.g., Fix authentication bug in OAuth flow"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Issue Description
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe the issue in detail..."
            rows={4}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {error && (
          <div className="flex items-start gap-3 p-4 bg-red-50 border border-red-200 rounded-lg">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <button
          onClick={handleGenerate}
          disabled={loading}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-medium py-3 rounded-lg transition flex items-center justify-center gap-2"
        >
          {loading && <Loader className="w-4 h-4 animate-spin" />}
          {loading ? 'Generating...' : 'Generate Workflow'}
        </button>
      </div>

      {result && (
        <div className="mt-6 p-4 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-start gap-3 mb-4">
            <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <h3 className="font-semibold text-green-900">Workflow Generated</h3>
              <p className="text-sm text-green-700 mt-1">ID: {result.workflow_id}</p>
            </div>
          </div>

          <div className="mt-4">
            <h4 className="font-medium text-gray-900 mb-3">Agent Decisions:</h4>
            <div className="space-y-2">
              {result.decisions.map((decision, idx) => (
                <div
                  key={idx}
                  className="p-3 bg-white rounded border border-green-100"
                >
                  <div className="flex justify-between items-start mb-2">
                    <span className="font-medium text-sm text-gray-900">
                      {decision.agent_role}
                    </span>
                    <span className="text-xs font-semibold px-2 py-1 bg-blue-100 text-blue-800 rounded">
                      {Math.round(decision.confidence * 100)}% confidence
                    </span>
                  </div>
                  <p className="text-sm text-gray-700">{decision.decision}</p>
                  <p className="text-xs text-gray-600 mt-1 italic">
                    {decision.reasoning}
                  </p>
                </div>
              ))}
            </div>
          </div>

          {result.workflow_yaml && (
            <details className="mt-4">
              <summary className="text-sm font-medium text-gray-700 cursor-pointer hover:text-gray-900">
                View Generated YAML
              </summary>
              <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto text-gray-800">
                {result.workflow_yaml}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  );
}
