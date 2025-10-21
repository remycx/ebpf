import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

export const fetchHosts = async () => {
  const response = await api.get('/hosts');
  return response.data;
};

export const fetchHostEvents = async (hostname, limit = 100) => {
  const response = await api.get(`/hosts/${hostname}/events`, {
    params: { limit }
  });
  return response.data;
};

export const fetchStats = async () => {
  const response = await api.get('/stats');
  return response.data;
};

export const fetchActiveConnections = async () => {
  const response = await api.get('/network/connections');
  return response.data;
};

export const fetchActiveProcesses = async () => {
  const response = await api.get('/processes');
  return response.data;
};

export default api;
