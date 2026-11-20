import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useStore } from '../store';
import {
  Home,
  Building2,
  Users,
  Network,
  Lock,
  FileText,
  AlertCircle,
  Menu,
  Trello,
  BookOpen,
  Zap,
} from 'lucide-react';
import clsx from 'clsx';

interface LayoutProps {
  children: React.ReactNode;
}

const NavLink: React.FC<{
  to: string;
  icon: React.ReactNode;
  label: string;
}> = ({ to, icon, label }) => {
  const location = useLocation();
  const isActive = location.pathname === to || location.pathname.startsWith(to + '/');

  return (
    <Link
      to={to}
      className={clsx(
        'flex items-center gap-3 px-4 py-2 rounded-lg transition-colors',
        isActive
          ? 'bg-blue-600 text-white'
          : 'text-gray-700 hover:bg-gray-100'
      )}
    >
      {icon}
      <span className="font-medium">{label}</span>
    </Link>
  );
};

export default function Layout({ children }: LayoutProps) {
  const stats = useStore((state) => state.stats);
  const error = useStore((state) => state.error);
  const [sidebarOpen, setSidebarOpen] = React.useState(true);

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <div
        className={clsx(
          'transition-all duration-300 bg-white border-r border-gray-200',
          sidebarOpen ? 'w-64' : 'w-20'
        )}
      >
        <div className="p-4 border-b">
          <div className="flex items-center justify-between">
            {sidebarOpen && (
              <h1 className="text-xl font-bold text-blue-600">Factory</h1>
            )}
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 hover:bg-gray-100 rounded-lg"
            >
              <Menu size={20} />
            </button>
          </div>
        </div>

        <nav className="p-4 space-y-2">
          <NavLink
            to="/"
            icon={<Home size={20} />}
            label="Dashboard"
          />
          <NavLink
            to="/tenants"
            icon={<Building2 size={20} />}
            label="Tenants"
          />
          <NavLink
            to="/agents"
            icon={<Users size={20} />}
            label="Agents"
          />
          <NavLink
            to="/services"
            icon={<Network size={20} />}
            label="Services"
          />
          <NavLink
            to="/credentials"
            icon={<Lock size={20} />}
            label="Credentials"
          />
          <NavLink
            to="/policies"
            icon={<FileText size={20} />}
            label="Policies"
          />
          <NavLink
            to="/logs"
            icon={<AlertCircle size={20} />}
            label="Logs"
          />
          <NavLink
            to="/projects"
            icon={<Trello size={20} />}
            label="Projects"
          />
          <NavLink
            to="/issues"
            icon={<BookOpen size={20} />}
            label="Issues"
          />
          <NavLink
            to="/converter"
            icon={<Zap size={20} />}
            label="Code Converter"
          />
        </nav>

        {sidebarOpen && (
          <div className="p-4 absolute bottom-0 w-64 border-t">
            <div className="text-sm text-gray-600">
              <div className="flex justify-between mb-2">
                <span>Tenants:</span>
                <span className="font-bold">{stats.tenants}</span>
              </div>
              <div className="flex justify-between mb-2">
                <span>Agents:</span>
                <span className="font-bold">{stats.agents}</span>
              </div>
              <div className="flex justify-between mb-2">
                <span>Services:</span>
                <span className="font-bold">{stats.services}</span>
              </div>
              <div className="flex justify-between">
                <span>Credentials:</span>
                <span className="font-bold">{stats.credentials}</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="bg-white border-b border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <h2 className="text-2xl font-bold text-gray-900">
              Agentic Software Factory
            </h2>
            <div className="flex items-center gap-4">
              <div className="text-sm text-gray-600">
                Status: <span className="text-green-600 font-bold">Online</span>
              </div>
            </div>
          </div>
        </header>

        {/* Error Banner */}
        {error && (
          <div className="bg-red-50 border-b border-red-200 px-6 py-3 flex items-center gap-3">
            <AlertCircle className="text-red-600" size={20} />
            <span className="text-red-800">{error}</span>
          </div>
        )}

        {/* Content */}
        <main className="flex-1 overflow-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
