import React, { useState, useEffect } from 'react';
import { fetchActiveProcesses } from '../services/api';
import './ProcessList.css';

const ProcessList = () => {
  const [processes, setProcesses] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadProcesses();
    const interval = setInterval(loadProcesses, 3000);
    return () => clearInterval(interval);
  }, []);

  const loadProcesses = async () => {
    try {
      const data = await fetchActiveProcesses();
      setProcesses(data || []);
      setLoading(false);
    } catch (error) {
      console.error('Failed to load processes:', error);
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
    <div className="process-list">
      <h2>Active Processes</h2>
      <table className="data-table">
        <thead>
          <tr>
            <th>Host</th>
            <th>PID</th>
            <th>PPID</th>
            <th>Process</th>
            <th>User</th>
            <th>State</th>
            <th>Start Time</th>
          </tr>
        </thead>
        <tbody>
          {processes.map((proc, index) => (
            <tr key={index}>
              <td>{proc.hostname}</td>
              <td>{proc.pid}</td>
              <td>{proc.ppid}</td>
              <td>{proc.comm}</td>
              <td>{proc.username}</td>
              <td>
                <span className={`status-badge ${proc.state}`}>{proc.state}</span>
              </td>
              <td>{formatTimestamp(proc.start_time)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {processes.length === 0 && (
        <div className="no-data">No active processes</div>
      )}
    </div>
  );
};

export default ProcessList;
