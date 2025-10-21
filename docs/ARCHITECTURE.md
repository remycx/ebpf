# Sentinel Architecture

## Overview

Sentinel is a distributed server fleet monitoring system that uses eBPF for efficient, low-overhead monitoring of network and process activity across multiple servers.

## Components

### 1. Agent (eBPF Monitoring Agent)

The agent runs on each monitored server and consists of:

#### eBPF Programs (Kernel Space)
- **network_monitor.bpf.c**: Monitors network events
  - TCP connections (connect, accept)
  - Connection closures
  - Source/destination IPs and ports
  - Associated process information

- **process_monitor.bpf.c**: Monitors process events
  - Process execution (exec)
  - Process termination (exit)
  - User/group information
  - Parent-child relationships
  - Process duration

#### Userspace Program (Go)
- Loads eBPF programs into the kernel
- Reads events from eBPF ring buffers
- Enriches events with additional context
- Batches and sends events to Sentinel server
- Handles reconnection and error recovery

**Data Flow:**
```
Kernel Events → eBPF Programs → Ring Buffers → Userspace Agent → WebSocket → Sentinel Server
```

### 2. Sentinel Server (Backend)

The backend server provides:

#### API Server (Go)
- WebSocket endpoint for agent connections
- REST API for frontend queries
- Real-time event streaming to frontend
- CORS support for cross-origin requests

#### Storage Layer
- In-memory event storage (current implementation)
- Configurable retention period
- Efficient querying by hostname, time range
- Support for future database backends (PostgreSQL, TimescaleDB)

#### Event Processing
- Receives batched events from agents
- Updates host metadata (last seen, event counts)
- Processes network connections and process states
- Broadcasts events to connected frontend clients

**API Endpoints:**
- `GET /api/v1/events` - WebSocket for agent connections
- `GET /api/v1/events/stream` - WebSocket for frontend real-time updates
- `GET /api/v1/hosts` - List all monitored hosts
- `GET /api/v1/hosts/{hostname}/events` - Get events for specific host
- `GET /api/v1/stats` - Get overall statistics
- `GET /api/v1/network/connections` - Get active connections
- `GET /api/v1/processes` - Get active processes

### 3. Sentinel Dashboard (Frontend)

The frontend is a React-based single-page application that provides:

#### Views
- **Dashboard**: Fleet overview with stats and recent activity
- **Hosts**: List of all monitored servers with status
- **Network**: Active network connections across fleet
- **Processes**: Running processes with user information
- **Event Stream**: Real-time event feed

#### Features
- Real-time WebSocket updates
- Auto-refresh for non-real-time data
- Responsive design
- Color-coded status indicators
- Search and filtering capabilities

## Data Models

### Network Event
```json
{
  "type": "network",
  "hostname": "web-server-01",
  "pid": 1234,
  "uid": 1000,
  "gid": 1000,
  "comm": "nginx",
  "event_type": "connect",
  "src_ip": "192.168.1.100",
  "dst_ip": "10.0.1.50",
  "src_port": 45678,
  "dst_port": 443,
  "timestamp": 1234567890
}
```

### Process Event
```json
{
  "type": "process",
  "hostname": "web-server-01",
  "event_type": "exec",
  "pid": 5678,
  "ppid": 1234,
  "uid": 1000,
  "gid": 1000,
  "username": "www-data",
  "comm": "php-fpm",
  "filename": "/usr/sbin/php-fpm",
  "timestamp": 1234567890,
  "cpu_id": 2
}
```

## Communication Flow

### Agent → Server
1. Agent establishes WebSocket connection to server
2. Agent sends batched events every 5 seconds or when batch reaches 50 events
3. Server acknowledges receipt (implicit via WebSocket)
4. On connection loss, agent retries with exponential backoff

### Server → Frontend
1. Frontend establishes WebSocket connection for real-time updates
2. Server broadcasts new events to all connected frontends
3. Frontend makes REST API calls for historical data and queries
4. Frontend auto-refreshes data every 5 seconds

## Security Considerations

### Agent
- Requires root privileges for eBPF operations
- Should be deployed with TLS for production
- Supports certificate verification
- Rate limiting to prevent DoS

### Server
- CORS configuration for frontend access
- WebSocket authentication (to be implemented)
- Input validation on all endpoints
- Configurable data retention limits

### Frontend
- API requests over HTTPS in production
- No sensitive data stored in browser
- WebSocket connections secured with WSS

## Scalability

### Current Implementation
- In-memory storage: Suitable for small to medium deployments
- Single server instance
- Estimated capacity: 100-500 agents

### Future Enhancements
- Database backend (PostgreSQL, TimescaleDB)
- Horizontal scaling with load balancer
- Event queuing with Kafka/RabbitMQ
- Metrics aggregation and downsampling
- Estimated capacity: 1000+ agents

## Performance

### Agent Overhead
- eBPF: < 1% CPU overhead
- Memory: ~10-20 MB per agent
- Network: ~1-5 KB/s per agent (depends on activity)

### Server Performance
- Memory: ~100 MB base + ~1 MB per active agent
- CPU: ~5% on modern hardware with 100 agents
- Network: Aggregate of all agent traffic

## Monitoring and Observability

### Logs
- Agent logs: Event collection, connection status
- Server logs: Agent connections, API requests, errors
- Frontend logs: API errors, WebSocket status

### Metrics (Future)
- Events per second
- Agent connection count
- API latency
- Storage usage
- Queue depth

## Deployment Architectures

### Single Server (Development/Small Scale)
```
┌─────────────────┐
│   Monitored     │
│     Server      │
│   (+ Agent)     │
└────────┬────────┘
         │
    ┌────▼─────┐
    │ Sentinel │
    │  Server  │
    │    +     │
    │ Frontend │
    └──────────┘
```

### Multi-Server (Production)
```
┌──────────┐  ┌──────────┐  ┌──────────┐
│  Agent   │  │  Agent   │  │  Agent   │
│ Server 1 │  │ Server 2 │  │ Server N │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │
     └─────────────┼─────────────┘
                   │
            ┌──────▼──────┐
            │  Sentinel   │
            │   Server    │
            │  (Backend)  │
            └──────┬──────┘
                   │
            ┌──────▼──────┐
            │  Sentinel   │
            │  Frontend   │
            │  (nginx)    │
            └─────────────┘
```

### High Availability (Future)
```
         Load Balancer
              │
    ┌─────────┼─────────┐
    │         │         │
┌───▼───┐ ┌───▼───┐ ┌───▼───┐
│Server1│ │Server2│ │Server3│
└───┬───┘ └───┬───┘ └───┬───┘
    └─────────┼─────────┘
              │
         TimescaleDB
        (HA Cluster)
```

## Technology Stack

- **eBPF**: Kernel-level monitoring
- **libbpf**: eBPF program loading and management
- **Go**: Agent and backend implementation
- **WebSocket**: Real-time bidirectional communication
- **React**: Frontend framework
- **REST API**: HTTP-based queries
- **YAML**: Configuration format
