import WorkflowGenerator from '@/components/WorkflowGenerator'
import ApprovalsPanel from '@/components/ApprovalsPanel'

export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
      <div className="max-w-6xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-xl">⚙️</span>
            </div>
            <h1 className="text-3xl font-bold text-white">
              Factory Stack Control Plane
            </h1>
          </div>
          <p className="text-purple-200">
            AI-Powered Multi-Agent Workflow Coordination & Governance
          </p>
        </div>

        {/* Architecture Overview */}
        <div className="grid grid-cols-5 gap-3 mb-8">
          {[
            { label: 'Governance', desc: 'Identity & Auth' },
            { label: 'Coordination', desc: '10 AI Agents' },
            { label: 'Execution', desc: 'DAG Engine' },
            { label: 'Integration', desc: 'Events' },
            { label: 'Observability', desc: 'Telemetry' },
          ].map((layer, i) => (
            <div
              key={i}
              className="bg-white bg-opacity-10 backdrop-blur-md border border-white border-opacity-20 rounded-lg p-3 text-center"
            >
              <div className="text-xs font-semibold text-purple-200 mb-1">
                Layer {5 - i}
              </div>
              <div className="text-sm font-bold text-white">{layer.label}</div>
              <div className="text-xs text-purple-300 mt-1">{layer.desc}</div>
            </div>
          ))}
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Workflow Generator */}
          <div className="lg:col-span-2">
            <WorkflowGenerator />
          </div>

          {/* Quick Stats */}
          <div className="bg-white bg-opacity-10 backdrop-blur-md border border-white border-opacity-20 rounded-lg p-6 h-fit">
            <h3 className="text-lg font-bold text-white mb-4">System Status</h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-purple-200">Active Agents</span>
                <span className="text-2xl font-bold text-green-400">10</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-purple-200">Workflows</span>
                <span className="text-2xl font-bold text-blue-400">42</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-purple-200">Test Coverage</span>
                <span className="text-2xl font-bold text-yellow-400">100+</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-purple-200">Avg Speedup</span>
                <span className="text-2xl font-bold text-pink-400">45%</span>
              </div>
            </div>
          </div>
        </div>

        {/* Approvals Panel */}
        <div className="mt-6">
          <ApprovalsPanel />
        </div>

        {/* Footer */}
        <div className="mt-12 text-center text-sm text-purple-300">
          <p>Factory Stack • Combining Governance + Multi-Agent + Efficient Execution</p>
        </div>
      </div>
    </main>
  )
}