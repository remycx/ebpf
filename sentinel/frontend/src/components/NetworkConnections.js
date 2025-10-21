import React, { useState, useEffect } from 'react';
import { fetchActiveConnections } from '../services/api';
import './NetworkConnections.css';

const NetworkConnections = () => {
  const [connections, setConnections] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadConnections();
    const interval = setInterval(loadConnections, 3000);
    return () => clearInterval(interval);
  }, []);

  const loadConnections = async () => {
    try {
      const data = await fetchActiveConnections();
      setConnections(data || []);
      setLoading(false);
    } catch (error) {
      console.error('Failed to load connections:', error);
      setLoading(false);
    }
  };

  const formatTimestamp = (timestamp) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  return (
    <div className="network-connections">
      <h2>Active Network Connections</h2>
      <table className="data-table">
        <thead>
          <tr>
            <th>Host</th>
            <th>PID</th>
            <th>Process</th>
            <th>Source</th>
            <th>Destination</th>
            <th>State</th>
            <th>Time</th>
          </tr>
        </thead>
        <tbody>
          {connections.map((conn, index) => (
            <tr key={index}>
              <td>{conn.hostname}</td>
              <td>{conn.pid}</td>
              <td>{conn.comm}</td>
              <td>{conn.src_ip}:{conn.src_port}</td>
              <td>{conn.dst_ip}:{conn.dst_port}</td>
              <td>
                <span className={`status-badge ${conn.state}`}>{conn.state}</span>
              </td>
              <td>{formatTimestamp(conn.timestamp)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {connections.length === 0 && (
        <div className="no-data">No active connections</div>
      )}
    </div>
  );
};

export default NetworkConnections;
