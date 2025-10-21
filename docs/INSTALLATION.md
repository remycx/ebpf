# Sentinel Installation Guide

## Prerequisites

### For Agent (on monitored servers)
- Linux kernel 5.8 or later
- clang and llvm
- libbpf development files
- Go 1.20 or later

### For Sentinel Server
- Go 1.20 or later
- Node.js 18 or later
- npm or yarn

## Installing Dependencies

### Ubuntu/Debian
```bash
# Agent dependencies
sudo apt-get update
sudo apt-get install -y clang llvm libbpf-dev linux-headers-$(uname -r) golang-go

# Server dependencies (if running server on this machine)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs
```

### RHEL/CentOS/Fedora
```bash
# Agent dependencies
sudo dnf install -y clang llvm libbpf-devel kernel-devel golang

# Server dependencies (if running server on this machine)
curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
sudo dnf install -y nodejs
```

## Building from Source

### Build Everything
```bash
./scripts/build-all.sh
```

### Build Individual Components

#### Agent
```bash
cd agent
make all
```

#### Backend
```bash
cd sentinel/backend
make build
```

#### Frontend
```bash
cd sentinel/frontend
npm install
npm run build
```

## Installation

### Install Agent
```bash
cd agent
sudo make install
```

This installs:
- `/usr/local/bin/sentinel-agent` - Agent binary
- `/etc/sentinel/agent.yaml` - Configuration file
- `/etc/sentinel/*.bpf.o` - eBPF programs

### Install Backend
```bash
cd sentinel/backend
sudo make install
```

This installs:
- `/usr/local/bin/sentinel` - Server binary
- `/etc/sentinel/sentinel.yaml` - Configuration file

### Install Frontend
The frontend is a static web application. You can serve it using:
- nginx
- Apache
- Any static file server

Example nginx configuration:
```nginx
server {
    listen 80;
    server_name sentinel.example.com;

    root /var/www/sentinel;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Configuration

### Agent Configuration
Edit `/etc/sentinel/agent.yaml`:

```yaml
agent:
  hostname: ""  # Auto-detected if empty
  tags:
    environment: production
    datacenter: us-east-1

server:
  url: "ws://sentinel-server:8080/api/v1/events"
  reconnect_delay: 5
  max_retries: -1
```

### Server Configuration
Edit `/etc/sentinel/sentinel.yaml`:

```yaml
server:
  listen: ":8080"

storage:
  type: memory
  retention_hours: 24
```

## Running Services

### Run Agent
```bash
sudo sentinel-agent -config /etc/sentinel/agent.yaml
```

### Run Backend
```bash
sentinel -config /etc/sentinel/sentinel.yaml
```

### Run Frontend Development Server
```bash
cd sentinel/frontend
npm start
```

## Systemd Service Files

### Agent Service
Create `/etc/systemd/system/sentinel-agent.service`:

```ini
[Unit]
Description=Sentinel Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sentinel-agent -config /etc/sentinel/agent.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Backend Service
Create `/etc/systemd/system/sentinel-server.service`:

```ini
[Unit]
Description=Sentinel Server
After=network.target

[Service]
Type=simple
User=sentinel
ExecStart=/usr/local/bin/sentinel -config /etc/sentinel/sentinel.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start services:
```bash
sudo systemctl daemon-reload
sudo systemctl enable sentinel-agent
sudo systemctl start sentinel-agent
sudo systemctl enable sentinel-server
sudo systemctl start sentinel-server
```

## Verification

### Check Agent Status
```bash
sudo systemctl status sentinel-agent
journalctl -u sentinel-agent -f
```

### Check Server Status
```bash
sudo systemctl status sentinel-server
journalctl -u sentinel-server -f
```

### Access Dashboard
Open your browser to: `http://localhost:3000` (development) or your configured domain.

## Troubleshooting

### Agent Issues

1. **Permission denied errors**: Make sure you're running as root
2. **eBPF program load failures**: Verify kernel version is 5.8+
3. **Cannot connect to server**: Check network connectivity and server URL

### Server Issues

1. **Port already in use**: Change the listen port in config.yaml
2. **WebSocket connection failures**: Check CORS settings and proxy configuration

### Frontend Issues

1. **Cannot connect to API**: Update API URL in `.env` or use proxy
2. **Build failures**: Clear node_modules and reinstall dependencies
