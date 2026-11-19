import React, { useState } from 'react';
import { Plus, AlertCircle, MessageCircle, User, Calendar, Tag, CheckCircle2, XCircle } from 'lucide-react';

interface Issue {
  id: string;
  number: number;
  title: string;
  description: string;
  status: 'open' | 'in-progress' | 'resolved' | 'closed';
  priority: 'low' | 'medium' | 'high' | 'critical';
  assignedAgent?: string;
  createdBy: string;
  createdAt: Date;
  resolvedAt?: Date;
  labels: string[];
  comments: number;
}

const priorityColors = {
  low: 'bg-blue-100 text-blue-800 border-blue-300',
  medium: 'bg-yellow-100 text-yellow-800 border-yellow-300',
  high: 'bg-orange-100 text-orange-800 border-orange-300',
  critical: 'bg-red-100 text-red-800 border-red-300',
};

const statusIcons = {
  open: <AlertCircle size={16} className="text-red-600" />,
  'in-progress': <User size={16} className="text-blue-600" />,
  resolved: <CheckCircle2 size={16} className="text-green-600" />,
  closed: <XCircle size={16} className="text-gray-600" />,
};

const statusColors = {
  open: 'bg-red-50 border-red-200',
  'in-progress': 'bg-blue-50 border-blue-200',
  resolved: 'bg-green-50 border-green-200',
  closed: 'bg-gray-50 border-gray-200',
};

export default function Issues() {
  const [issues, setIssues] = useState<Issue[]>([
    {
      id: '1',
      number: 1,
      title: 'Dashboard not loading metrics',
      description: 'Stats are not updating in real-time. Need to add WebSocket integration.',
      status: 'in-progress',
      priority: 'high',
      assignedAgent: 'analyzer-1',
      createdBy: 'alice',
      createdAt: new Date(Date.now() - 2 * 60 * 60000),
      labels: ['bug', 'realtime', 'dashboard'],
      comments: 3,
    },
    {
      id: '2',
      number: 2,
      title: 'Service health check timing out',
      description: 'HTTP health checks are timing out for some services. Consider increasing timeout or adding retry logic.',
      status: 'open',
      priority: 'critical',
      createdBy: 'bob',
      createdAt: new Date(Date.now() - 30 * 60000),
      labels: ['bug', 'critical', 'health-checks'],
      comments: 1,
    },
    {
      id: '3',
      number: 3,
      title: 'Implement dark mode',
      description: 'Add dark theme toggle to improve user experience in low-light environments.',
      status: 'open',
      priority: 'low',
      createdBy: 'carol',
      createdAt: new Date(Date.now() - 24 * 60 * 60000),
      labels: ['enhancement', 'ui'],
      comments: 0,
    },
    {
      id: '4',
      number: 4,
      title: 'Credential rotation policy',
      description: 'Implement automatic credential rotation every 30 days.',
      status: 'resolved',
      priority: 'medium',
      assignedAgent: 'auditor-1',
      createdBy: 'dave',
      createdAt: new Date(Date.now() - 72 * 60 * 60000),
      resolvedAt: new Date(Date.now() - 12 * 60 * 60000),
      labels: ['security', 'feature'],
      comments: 5,
    },
  ]);

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newIssue, setNewIssue] = useState({
    title: '',
    description: '',
    priority: 'medium' as const,
    labels: '',
  });

  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [filterPriority, setFilterPriority] = useState<string | null>(null);

  const handleCreateIssue = (e: React.FormEvent) => {
    e.preventDefault();
    if (newIssue.title.trim()) {
      const issue: Issue = {
        id: String(issues.length + 1),
        number: Math.max(...issues.map((i) => i.number), 0) + 1,
        title: newIssue.title,
        description: newIssue.description,
        status: 'open',
        priority: newIssue.priority,
        createdBy: 'current-user',
        createdAt: new Date(),
        labels: newIssue.labels
          .split(',')
          .map((l) => l.trim())
          .filter((l) => l),
        comments: 0,
      };
      setIssues([issue, ...issues]);
      setNewIssue({ title: '', description: '', priority: 'medium', labels: '' });
      setShowCreateForm(false);
    }
  };

  const assignIssueToAgent = (issueId: string, agentId: string) => {
    setIssues(
      issues.map((issue) =>
        issue.id === issueId
          ? { ...issue, assignedAgent: agentId, status: 'in-progress' as const }
          : issue
      )
    );
  };

  const resolveIssue = (issueId: string) => {
    setIssues(
      issues.map((issue) =>
        issue.id === issueId
          ? { ...issue, status: 'resolved' as const, resolvedAt: new Date() }
          : issue
      )
    );
  };

  const filteredIssues = issues.filter((issue) => {
    if (filterStatus && issue.status !== filterStatus) return false;
    if (filterPriority && issue.priority !== filterPriority) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Issues</h1>
          <p className="text-gray-600 mt-2">Create issues for agents to pick up and resolve</p>
        </div>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          <Plus size={20} />
          New Issue
        </button>
      </div>

      {/* Create Form */}
      {showCreateForm && (
        <div className="bg-white rounded-lg shadow-sm p-6 border border-blue-200">
          <h2 className="text-lg font-bold text-gray-900 mb-4">Create New Issue</h2>
          <form onSubmit={handleCreateIssue} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
              <input
                type="text"
                placeholder="Issue title..."
                value={newIssue.title}
                onChange={(e) => setNewIssue({ ...newIssue, title: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <textarea
                placeholder="Describe the issue..."
                value={newIssue.description}
                onChange={(e) => setNewIssue({ ...newIssue, description: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 h-24"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <select
                  value={newIssue.priority}
                  onChange={(e) => setNewIssue({ ...newIssue, priority: e.target.value as any })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Labels (comma-separated)</label>
                <input
                  type="text"
                  placeholder="bug, feature, ui"
                  value={newIssue.labels}
                  onChange={(e) => setNewIssue({ ...newIssue, labels: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition-colors"
              >
                Create Issue
              </button>
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="bg-gray-200 text-gray-700 px-6 py-2 rounded-lg hover:bg-gray-300 transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <button
          onClick={() => setFilterStatus(filterStatus === 'open' ? null : 'open')}
          className={`px-4 py-2 rounded-lg border transition-colors ${
            filterStatus === 'open'
              ? 'bg-red-50 border-red-300 text-red-800'
              : 'bg-white border-gray-200 text-gray-700 hover:border-gray-300'
          }`}
        >
          Open ({issues.filter((i) => i.status === 'open').length})
        </button>
        <button
          onClick={() => setFilterStatus(filterStatus === 'in-progress' ? null : 'in-progress')}
          className={`px-4 py-2 rounded-lg border transition-colors ${
            filterStatus === 'in-progress'
              ? 'bg-blue-50 border-blue-300 text-blue-800'
              : 'bg-white border-gray-200 text-gray-700 hover:border-gray-300'
          }`}
        >
          In Progress ({issues.filter((i) => i.status === 'in-progress').length})
        </button>
        <button
          onClick={() => setFilterStatus(filterStatus === 'resolved' ? null : 'resolved')}
          className={`px-4 py-2 rounded-lg border transition-colors ${
            filterStatus === 'resolved'
              ? 'bg-green-50 border-green-300 text-green-800'
              : 'bg-white border-gray-200 text-gray-700 hover:border-gray-300'
          }`}
        >
          Resolved ({issues.filter((i) => i.status === 'resolved').length})
        </button>
      </div>

      {/* Issues List */}
      <div className="space-y-3">
        {filteredIssues.map((issue) => (
          <div
            key={issue.id}
            className={`${statusColors[issue.status]} border rounded-lg p-4 hover:shadow-md transition-shadow`}
          >
            <div className="flex items-start gap-4">
              <div className="pt-1">{statusIcons[issue.status]}</div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm text-gray-600">#{issue.number}</span>
                  <h3 className="text-lg font-bold text-gray-900">{issue.title}</h3>
                  <span className={`px-3 py-1 rounded-full text-xs font-medium border ${priorityColors[issue.priority]}`}>
                    {issue.priority}
                  </span>
                </div>

                <p className="text-gray-700 mb-3">{issue.description}</p>

                <div className="flex flex-wrap gap-2 mb-3">
                  {issue.labels.map((label) => (
                    <span key={label} className="bg-gray-200 text-gray-700 text-xs px-2 py-1 rounded">
                      {label}
                    </span>
                  ))}
                </div>

                <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
                  <div className="flex items-center gap-1">
                    <User size={14} />
                    <span>Created by {issue.createdBy}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Calendar size={14} />
                    <span>
                      {issue.createdAt.toLocaleDateString()} {issue.createdAt.toLocaleTimeString()}
                    </span>
                  </div>
                  <div className="flex items-center gap-1">
                    <MessageCircle size={14} />
                    <span>{issue.comments} comments</span>
                  </div>
                </div>
              </div>

              <div className="flex flex-col gap-2">
                {issue.assignedAgent ? (
                  <div className="bg-blue-100 text-blue-800 px-3 py-1 rounded text-sm font-medium text-center">
                    {issue.assignedAgent}
                  </div>
                ) : (
                  <select
                    defaultValue=""
                    onChange={(e) => e.target.value && assignIssueToAgent(issue.id, e.target.value)}
                    className="px-3 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Assign agent...</option>
                    <option value="analyzer-1">analyzer-1</option>
                    <option value="auditor-1">auditor-1</option>
                    <option value="optimizer-1">optimizer-1</option>
                  </select>
                )}

                {issue.status === 'in-progress' && (
                  <button
                    onClick={() => resolveIssue(issue.id)}
                    className="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700 transition-colors"
                  >
                    Resolve
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {filteredIssues.length === 0 && (
        <div className="text-center py-12 bg-white rounded-lg">
          <AlertCircle className="mx-auto text-gray-400 mb-3" size={40} />
          <p className="text-gray-600">No issues found</p>
        </div>
      )}
    </div>
  );
}
