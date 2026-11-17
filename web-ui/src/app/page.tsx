import TriagePanel from '@/components/TriagePanel'

export default function Home() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-50">
      <div className="max-w-2xl mx-auto px-4 py-12">
        {/* Simple Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            GitHub Issue Triage
          </h1>
          <p className="text-lg text-gray-600">
            Analyze and categorize GitHub issues automatically
          </p>
        </div>

        {/* Main Feature - Triage Panel */}
        <TriagePanel />

        {/* Simple Footer */}
        <div className="mt-12 text-center text-sm text-gray-600">
          <p>Powered by Agentic Software Factory • v1.0.0</p>
        </div>
      </div>
    </main>
  )
}