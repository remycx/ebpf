import React, { useState, useEffect } from 'react';
import './App.css';
import Dashboard from './components/Dashboard';
import HostList from './components/HostList';
import NetworkConnections from './components/NetworkConnections';
import ProcessList from './components/ProcessList';
import EventStream from './components/EventStream';
import { useWebSocket } from './hooks/useWebSocket';
import { fetchHosts, fetchStats } from './services/api';

function App() {
  const [hosts, setHosts] = useState([]);
  const [stats, setStats] = useState({});
  const [selectedHost, setSelectedHost] = useState(null);
  const [activeTab, setActiveTab] = useState('dashboard');

  const { lastEvent, isConnected } = useWebSocket('ws://localhost:8080/api/v1/events/stream');

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      const [hostsData, statsData] = await Promise.all([
        fetchHosts(),
        fetchStats()
      ]);
      setHosts(hostsData);
      setStats(statsData);
    } catch (error) {
      console.error('Failed to load data:', error);
    }
  };

  return (
    <div className="App">
      <header className="header">
        <div className="header-content">
          <h1>Sentinel</h1>
          <div className="header-subtitle">Fleet Monitoring System</div>
        </div>
        <div className="connection-status">
          <span className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}></span>
          <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </header>

      <nav className="nav-tabs">
        <button
          className={activeTab === 'dashboard' ? 'active' : ''}
          onClick={() => setActiveTab('dashboard')}
        >
          Dashboard
        </button>
        <button
          className={activeTab === 'hosts' ? 'active' : ''}
          onClick={() => setActiveTab('hosts')}
        >
          Hosts
        </button>
        <button
          className={activeTab === 'connections' ? 'active' : ''}
          onClick={() => setActiveTab('connections')}
        >
          Network
        </button>
        <button
          className={activeTab === 'processes' ? 'active' : ''}
          onClick={() => setActiveTab('processes')}
        >
          Processes
        </button>
        <button
          className={activeTab === 'events' ? 'active' : ''}
          onClick={() => setActiveTab('events')}
        >
          Event Stream
        </button>
      </nav>

      <main className="main-content">
        {activeTab === 'dashboard' && (
          <Dashboard hosts={hosts} stats={stats} lastEvent={lastEvent} />
        )}
        {activeTab === 'hosts' && (
          <HostList hosts={hosts} onSelectHost={setSelectedHost} />
        )}
        {activeTab === 'connections' && (
          <NetworkConnections />
        )}
        {activeTab === 'processes' && (
          <ProcessList />
        )}
        {activeTab === 'events' && (
          <EventStream />
        )}
      </main>
    </div>
  );
}

export default App;
