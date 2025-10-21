import React from 'react';
import './Dashboard.css';

const Dashboard = ({ hosts, stats, lastEvent }) => {
  const formatTimestamp = (timestamp) => {
    return new Date(timestamp * 1000).toLocaleTimeString();
  };

  const getHostStatus = (host) => {
    const lastSeenTime = new Date(host.last_seen);
    const now = new Date();
    const diffMinutes = (now - lastSeenTime) / 1000 / 60;

    if (diffMinutes < 1) return 'online';
    if (diffMinutes < 5) return 'warning';
    return 'offline';
  };

  return (
    <div className="dashboard">
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{hosts.length}</div>
          <div className="stat-label">Total Hosts</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.total_events || 0}</div>
          <div className="stat-label">Total Events</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.active_connections || 0}</div>
          <div className="stat-label">Active Connections</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.active_processes || 0}</div>
          <div className="stat-label">Active Processes</div>
        </div>
      </div>

      <div className="dashboard-grid">
        <div className="dashboard-section">
          <h2>Fleet Overview</h2>
          <div className="host-grid">
            {hosts.map((host) => (
              <div key={host.hostname} className={`host-card ${getHostStatus(host)}`}>
                <div className="host-name">{host.hostname}</div>
                <div className="host-stats">
                  <div className="host-stat">
                    <span className="label">Events:</span>
                    <span className="value">{host.event_count}</span>
                  </div>
                  <div className="host-stat">
                    <span className="label">Connections:</span>
                    <span className="value">{host.connection_count}</span>
                  </div>
                  <div className="host-stat">
                    <span className="label">Processes:</span>
                    <span className="value">{host.process_count}</span>
                  </div>
                </div>
                <div className="host-status">
                  Last seen: {new Date(host.last_seen).toLocaleString()}
                </div>
              </div>
            ))}
            {hosts.length === 0 && (
              <div className="no-data">No hosts connected</div>
            )}
          </div>
        </div>

        <div className="dashboard-section">
          <h2>Recent Activity</h2>
          <div className="activity-feed">
            {lastEvent ? (
              <div className="activity-item">
                <div className="activity-type">{lastEvent.type}</div>
                <div className="activity-host">{lastEvent.hostname}</div>
                <div className="activity-time">
                  {formatTimestamp(lastEvent.timestamp)}
                </div>
                <div className="activity-data">
                  {JSON.stringify(lastEvent.data, null, 2)}
                </div>
              </div>
            ) : (
              <div className="no-data">Waiting for events...</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
