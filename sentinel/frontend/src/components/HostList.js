import React from 'react';
import './HostList.css';

const HostList = ({ hosts, onSelectHost }) => {
  return (
    <div className="host-list">
      <h2>Monitored Hosts</h2>
      <table className="data-table">
        <thead>
          <tr>
            <th>Hostname</th>
            <th>Events</th>
            <th>Connections</th>
            <th>Processes</th>
            <th>Last Seen</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {hosts.map((host) => {
            const lastSeenTime = new Date(host.last_seen);
            const now = new Date();
            const diffMinutes = (now - lastSeenTime) / 1000 / 60;
            const status = diffMinutes < 1 ? 'Online' : diffMinutes < 5 ? 'Warning' : 'Offline';
            const statusClass = status.toLowerCase();

            return (
              <tr key={host.hostname} onClick={() => onSelectHost(host)}>
                <td>{host.hostname}</td>
                <td>{host.event_count}</td>
                <td>{host.connection_count}</td>
                <td>{host.process_count}</td>
                <td>{lastSeenTime.toLocaleString()}</td>
                <td>
                  <span className={`status-badge ${statusClass}`}>{status}</span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {hosts.length === 0 && (
        <div className="no-data">No hosts available</div>
      )}
    </div>
  );
};

export default HostList;
