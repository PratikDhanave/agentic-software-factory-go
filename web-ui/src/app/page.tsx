import Image from 'next/image'
import { Inter } from 'next/font/google'

const inter = Inter({ subsets: ['latin'], weight: ['400', '500', '600'] })

export default function Home() {
  return (
    <main className="min-h-screen bg-white text-gray-900 antialiased">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <header className="mb-8 border-b border-gray-200 dark:border-gray-700">
          <h1 className="text-3xl font-bold tracking-tighter text-gray-900 dark:text-gray-100">
            Agentic Software Factory
          </h1>
          <p className="mt-2 text-lg text-gray-600 dark:text-gray-400">
            Production-grade Agentic Software Engineering patterns for Go
          </p>
        </header>

        {/* Google-styled cards grid */}
        <section className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {/* Context Engineering Card */}
          <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-xl transition-shadow duration-300 dark:hover:shadow-gray-800/50">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-lg bg-indigo-100 dark:bg-indigo-900/20 flex items-center justify-center">
                  <svg className="w-5 h-5 text-indigo-600 dark:text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1l-3-3m8 5v1h4v-2a2 2 0 0 0-2-2h-4v2l-3-3m8 5v1h4v-2a2 2 0 0 0-2-2h-4v2l-3-3"/>
                  </svg>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Context Engineering</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Load and display project context with ADRs, specs, and source code</p>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                Load context from ADRs, specs, and source code files with priority-based token budgeting
              </p>
            </div>
          </div>

          {/* Specification Engineering Card */}
          <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-xl transition-shadow duration-300 dark:hover:shadow-gray-800/50">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-lg bg-green-100 dark:bg-green-900/20 flex items-center justify-center">
                  <svg className="w-5 h-5 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6.348-3.6a2.5 2.5 0 1 1 1.1 3.1 2.5 2.5 0 0 1-1.1 3.1 5 5 0 0 1-7.072 0 2.5 2.5 0 0 1-1.1-3.1Z"/>
                  </svg>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Specification Engineering</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Machine-readable specs with validation and prompt generation</p>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                YAML/JSON specs with constraints, inputs, outputs, and execution plan steps
              </p>
            </div>
          </div>

          {/* Verification Card */}
          <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-xl transition-shadow duration-300 dark:hover:shadow-gray-800/50">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-lg bg-red-100 dark:bg-red-900/20 flex items-center justify-center">
                  <svg className="w-5 h-5 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 9v2m-6-3h12a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v2m10 7V10m0 3v2m0 0v2m-6-3h12a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v2m10 7v-2m0 4v-2m-6 3h12a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v4"/>
                  </svg>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Verification</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Build, test, lint loops</p>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                Multi-phase verification (build, test, lint) with detailed error parsing and retry loops
              </p>
            </div>
          </div>
        </section>

        {/* Factory orchestration and MCP */}
        <section className="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {/* Factory */}
          <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-xl transition-shadow duration-300 dark:hover:shadow-gray-800/50">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-lg bg-orange-100 dark:bg-orange-900/20 flex items-center justify-center">
                  <svg className="w-5 h-5 text-orange-600 dark:text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16"/>
                  </svg>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Factory Orchestration</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Worker pools, task queues, and durable factory management</p>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                Factory orchestrator with auto-scaling worker pools, SQLite persistence, and task recovery
              </p>
            </div>
          </div>

          {/* MCP Server */}
          <div className="group bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-lg hover:shadow-gray-200/50 dark:hover:shadow-gray-800/50 transition-all duration-300">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-lg bg-purple-100 dark:bg-purple-900/20 flex items-center justify-center">
                  <svg className="w-5 h-5 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1l-3-3m8 5v1h4v-2a2 2 0 0 0-2-2h-4v2l-3-3M15 21l-3-5a2.5 2.5 0 0 1 1.5-3.5h5a2.5 2.5 0 0 1 1.5 3.5Z"/>
                  </svg>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">MCP Server</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">JSON-RPC 2.0 API</p>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                JSON-RPC 2.0 MCP server exposing factory tools and resources
              </p>
            </div>
          </div>
        </section>
      </div>
    </main>
  )
}