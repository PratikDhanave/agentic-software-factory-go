import React, { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { useStore } from './store';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import TenantsList from './pages/TenantsList';
import TenantDetail from './pages/TenantDetail';
import Agents from './pages/Agents';
import Services from './pages/Services';
import Credentials from './pages/Credentials';
import Policies from './pages/Policies';
import Logs from './pages/Logs';

export default function App() {
  const loadStats = useStore((state) => state.loadStats);

  useEffect(() => {
    loadStats();
    const interval = setInterval(loadStats, 10000);
    return () => clearInterval(interval);
  }, [loadStats]);

  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/tenants" element={<TenantsList />} />
          <Route path="/tenants/:id" element={<TenantDetail />} />
          <Route path="/tenants/:id/agents" element={<Agents />} />
          <Route path="/tenants/:id/services" element={<Services />} />
          <Route path="/agents/:id/credentials" element={<Credentials />} />
          <Route path="/tenants/:id/policies" element={<Policies />} />
          <Route path="/logs" element={<Logs />} />
        </Routes>
      </Layout>
    </Router>
  );
}
