# Sentinel - Server Fleet Monitoring System

A comprehensive server fleet monitoring system using eBPF agents and a centralized visualization platform.

## Architecture

### Components

1. **eBPF Agent** - Lightweight monitoring agent deployed on each server
   - Network activity monitoring (connections, traffic)
   - Process activity monitoring (exec, exit, with user info)
   - Real-time data collection using eBPF
   - Sends metrics to Sentinel server

2. **Sentinel Server** - Centralized monitoring platform
   - Backend API (Go) - Receives and processes agent data
   - Frontend Dashboard (React) - Visualizes fleet-wide metrics
   - Real-time monitoring and alerts
   - Historical data storage

## Directory Structure

```
.
├── agent/                 # eBPF monitoring agent
│   ├── ebpf/             # eBPF C programs
│   └── src/              # Userspace agent (Go)
├── sentinel/             # Centralized server
│   ├── backend/          # Backend API (Go)
│   └── frontend/         # Frontend dashboard (React)
├── docs/                 # Documentation
└── scripts/              # Build and deployment scripts
```

## Features

- **Network Monitoring**: Track TCP/UDP connections, data transfer rates
- **Process Monitoring**: Monitor process creation/termination with user context
- **Real-time Visualization**: Live dashboard showing fleet-wide activity
- **Historical Data**: Store and query historical metrics
- **Multi-server Support**: Monitor unlimited number of servers
- **Low Overhead**: eBPF-based monitoring with minimal performance impact

## Quick Start

### Building the Agent

```bash
cd agent
make
```

### Running the Sentinel Server

```bash
cd sentinel/backend
go build -o sentinel
./sentinel
```

### Starting the Frontend

```bash
cd sentinel/frontend
npm install
npm start
```

## Requirements

- Linux kernel 5.8+ (for eBPF)
- Go 1.20+
- Node.js 18+
- libbpf development files

## Security

Sentinel includes comprehensive security features:

- **Agent Authentication**: API key-based authentication for agents
- **TLS/SSL Support**: Encrypted communication (WSS/HTTPS)
- **Rate Limiting**: Protection against DoS attacks
- **Input Validation**: Comprehensive data validation and sanitization
- **Security Headers**: Standard HTTP security headers
- **Audit Logging**: Request logging for security monitoring

**For production deployments**:
1. Enable authentication in `sentinel/backend/config.yaml`
2. Generate API keys using the keygen tool
3. Enable TLS with valid certificates
4. Configure appropriate rate limits
5. Set up firewall rules

See [SECURITY.md](docs/SECURITY.md) for detailed security configuration and best practices.

### Quick Security Setup

Generate an API key:
```bash
cd sentinel/backend
go run tools/keygen/main.go --name "my-agent" --expires 90d
```

Enable authentication in `config.yaml`:
```yaml
security:
  require_auth: true
```

Configure agent with API key in `agent.yaml`:
```yaml
server:
  api_key: "your-generated-api-key"
```

## License

MIT
