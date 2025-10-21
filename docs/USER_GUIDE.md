# Sentinel User Guide

## Getting Started

### Quick Start

1. **Start the Sentinel Server**
   ```bash
   cd sentinel/backend
   go run . -config config.yaml
   ```

2. **Start the Frontend**
   ```bash
   cd sentinel/frontend
   npm start
   ```

3. **Deploy Agents on Monitored Servers**
   ```bash
   cd agent
   sudo make run
   ```

4. **Access Dashboard**
   Open `http://localhost:3000` in your browser

## Dashboard Overview

### Main Dashboard

The main dashboard provides a fleet-wide overview:

- **Statistics Cards**: Total hosts, events, connections, and processes
- **Fleet Overview**: Grid of all monitored hosts with their status
- **Recent Activity**: Real-time feed of the latest events

**Host Status Indicators:**
- 🟢 **Green (Online)**: Host seen within last minute
- 🟡 **Yellow (Warning)**: Host seen 1-5 minutes ago
- 🔴 **Red (Offline)**: Host not seen for 5+ minutes

### Hosts View

View all monitored hosts with detailed information:

- Hostname
- Total events received
- Active connections count
- Active processes count
- Last seen timestamp
- Current status

**Actions:**
- Click on a host to see its detailed events

### Network View

Monitor active network connections across your fleet:

- Source and destination IP addresses
- Port numbers
- Associated process (PID and name)
- Connection state
- Timestamp

**Use Cases:**
- Identify unexpected outbound connections
- Monitor service-to-service communication
- Detect port scanning or suspicious activity
- Track bandwidth-heavy processes

### Processes View

View running processes across all servers:

- Process name and PID
- Parent process (PPID)
- User running the process
- Start time
- Current state

**Use Cases:**
- Monitor unauthorized process execution
- Track processes by specific users
- Identify long-running processes
- Detect unusual process hierarchies

### Event Stream

Real-time stream of all events from the fleet:

- Color-coded by event type
  - Blue: Network events
  - Green: Process events
- Includes full event details
- Auto-scrolls with new events
- Shows last 100 events

## Understanding Events

### Network Events

#### Connect Event
Triggered when a process initiates a TCP connection.

```json
{
  "event_type": "connect",
  "pid": 1234,
  "comm": "curl",
  "src_ip": "192.168.1.100",
  "dst_ip": "93.184.216.34",
  "dst_port": 443
}
```

**What it means:** Process `curl` (PID 1234) is connecting to 93.184.216.34:443

#### Accept Event
Triggered when a server accepts an incoming connection.

```json
{
  "event_type": "accept",
  "pid": 5678,
  "comm": "nginx",
  "src_ip": "0.0.0.0",
  "dst_port": 80
}
```

**What it means:** nginx (PID 5678) accepted a connection on port 80

#### Close Event
Triggered when a TCP connection is closed.

```json
{
  "event_type": "close",
  "pid": 1234,
  "comm": "curl"
}
```

**What it means:** Connection from process curl (PID 1234) was closed

### Process Events

#### Exec Event
Triggered when a new process is executed.

```json
{
  "event_type": "exec",
  "pid": 9012,
  "ppid": 8765,
  "comm": "bash",
  "filename": "/bin/bash",
  "username": "ubuntu",
  "uid": 1000
}
```

**What it means:** User `ubuntu` executed `/bin/bash` (new PID 9012, parent PID 8765)

#### Exit Event
Triggered when a process terminates.

```json
{
  "event_type": "exit",
  "pid": 9012,
  "comm": "bash",
  "exit_code": 0,
  "duration_ns": 15000000000
}
```

**What it means:** Process bash (PID 9012) exited with code 0 after running for 15 seconds

## Common Use Cases

### 1. Detecting Unauthorized Access

**Scenario:** Monitor for SSH sessions from unexpected users or IPs

**How to:**
1. Go to **Processes** view
2. Look for processes with `comm=sshd` or `comm=bash`
3. Check the `username` field
4. Cross-reference with **Network** view for connection source IPs

### 2. Troubleshooting Application Connectivity

**Scenario:** Application can't connect to database

**How to:**
1. Go to **Network** view
2. Filter by application process name
3. Check destination IPs and ports
4. Verify connection states

### 3. Monitoring Service Health

**Scenario:** Ensure critical services are running

**How to:**
1. Go to **Processes** view
2. Search for your service process names
3. Check `state` is "running"
4. Monitor for unexpected exits in **Event Stream**

### 4. Security Auditing

**Scenario:** Track all commands executed by specific user

**How to:**
1. Go to **Event Stream**
2. Filter events by `username`
3. Look for process exec events
4. Review `filename` field for executed commands

### 5. Capacity Planning

**Scenario:** Understand network traffic patterns

**How to:**
1. Monitor **Network** view over time
2. Note peak connection counts
3. Identify most active services
4. Plan scaling based on patterns

## Configuration

### Agent Configuration

Edit `agent/agent.yaml`:

```yaml
agent:
  hostname: ""  # Leave empty for auto-detection
  tags:
    environment: production
    datacenter: us-east-1
    role: webserver

server:
  url: "ws://sentinel-server:8080/api/v1/events"
  reconnect_delay: 5
  max_retries: -1
```

**Parameters:**
- `hostname`: Custom hostname (default: system hostname)
- `tags`: Custom metadata tags for this host
- `url`: Sentinel server WebSocket URL
- `reconnect_delay`: Seconds to wait before reconnecting
- `max_retries`: Max connection retries (-1 = infinite)

### Server Configuration

Edit `sentinel/backend/config.yaml`:

```yaml
server:
  listen: ":8080"

storage:
  type: memory
  retention_hours: 24
```

**Parameters:**
- `listen`: Server listen address
- `type`: Storage backend (currently only "memory")
- `retention_hours`: How long to keep events

### Frontend Configuration

Create `sentinel/frontend/.env`:

```env
REACT_APP_API_URL=http://localhost:8080/api/v1
```

**Parameters:**
- `REACT_APP_API_URL`: Backend API base URL

## Best Practices

### Agent Deployment

1. **Start with a few servers**: Deploy to 2-3 servers initially
2. **Monitor resource usage**: Check CPU and memory impact
3. **Verify connectivity**: Ensure agents appear in dashboard
4. **Gradual rollout**: Deploy to rest of fleet incrementally

### Server Sizing

- **Small deployments (1-10 agents)**: 2 CPU cores, 2 GB RAM
- **Medium deployments (10-100 agents)**: 4 CPU cores, 8 GB RAM
- **Large deployments (100+ agents)**: 8+ CPU cores, 16+ GB RAM

### Security

1. **Use TLS**: Always use WSS (WebSocket Secure) in production
2. **Restrict access**: Firewall the server to only allow agent IPs
3. **Regular updates**: Keep agents and server up to date
4. **Audit logs**: Regularly review event logs for anomalies

### Performance Optimization

1. **Adjust batch size**: Modify agent batch size for your workload
2. **Tune retention**: Lower retention period if memory is constrained
3. **Use filters**: Frontend filtering reduces data transfer
4. **Database backend**: Use TimescaleDB for large deployments

## Troubleshooting

### Agent Not Appearing in Dashboard

1. Check agent logs: `journalctl -u sentinel-agent -f`
2. Verify network connectivity to server
3. Check WebSocket URL in agent config
4. Ensure server is running and accessible

### High Memory Usage on Server

1. Reduce `retention_hours` in server config
2. Consider database backend instead of memory
3. Increase server RAM
4. Implement event sampling

### Missing Events

1. Check agent is running as root
2. Verify kernel version is 5.8+
3. Check for eBPF program load errors in logs
4. Ensure required kernel headers are installed

### Frontend Not Loading Data

1. Check browser console for errors
2. Verify API URL is correct
3. Check CORS settings on server
4. Ensure server is running and accessible

## API Reference

For developers integrating with Sentinel:

### REST Endpoints

- `GET /api/v1/hosts` - List all hosts
- `GET /api/v1/hosts/{hostname}/events?limit=100` - Get host events
- `GET /api/v1/stats` - Get statistics
- `GET /api/v1/network/connections` - Get active connections
- `GET /api/v1/processes` - Get active processes

### WebSocket Endpoints

- `ws://server:8080/api/v1/events` - Agent connection (send events)
- `ws://server:8080/api/v1/events/stream` - Frontend connection (receive events)

## Advanced Topics

### Custom Event Processing

Modify `sentinel/backend/pkg/storage/memory.go` to add custom event processing:

```go
func (s *MemoryStore) processCustomEvent(data map[string]interface{}) {
    // Your custom logic here
}
```

### Integration with Alerting

Use the WebSocket stream to integrate with alerting systems:

```javascript
const ws = new WebSocket('ws://sentinel:8080/api/v1/events/stream');
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (shouldAlert(data)) {
        sendAlert(data);
    }
};
```

### Exporting Data

Query the API and export to CSV, JSON, or other formats:

```bash
curl http://sentinel:8080/api/v1/hosts/web-server-01/events?limit=1000 > events.json
```

## Getting Help

- **Issues**: Report bugs on GitHub
- **Questions**: Check documentation in `docs/`
- **Community**: Join our discussion forum
- **Support**: Contact support@example.com
