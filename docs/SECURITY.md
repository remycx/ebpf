# Sentinel Security Guide

## Overview

Sentinel implements multiple layers of security to protect your fleet monitoring infrastructure:

1. **Agent Authentication** - API key-based authentication for agents
2. **TLS/SSL Encryption** - Encrypted communication between components
3. **Rate Limiting** - Protection against DoS attacks
4. **Input Validation** - Comprehensive data validation
5. **Security Headers** - Standard HTTP security headers
6. **Audit Logging** - Request logging for security auditing

## Security Architecture

```
┌─────────────┐
│    Agent    │
│  (API Key)  │
└──────┬──────┘
       │ TLS/WSS
       ▼
┌─────────────────────────────────┐
│   Sentinel Server               │
│  ┌──────────────────────────┐  │
│  │  Rate Limiter            │  │
│  └────────────┬─────────────┘  │
│  ┌────────────▼─────────────┐  │
│  │  Auth Middleware         │  │
│  └────────────┬─────────────┘  │
│  ┌────────────▼─────────────┐  │
│  │  Input Validation        │  │
│  └────────────┬─────────────┘  │
│  ┌────────────▼─────────────┐  │
│  │  Event Processing        │  │
│  └──────────────────────────┘  │
└─────────────────────────────────┘
```

## Agent Authentication

### Enabling Authentication

1. **Generate API Key** using the management tool:
```bash
cd sentinel/backend
go run tools/keygen/main.go --name "production-agent-1"
```

2. **Update server configuration** (`sentinel/backend/config.yaml`):
```yaml
security:
  require_auth: true
  api_keys_file: "api-keys.yaml"
```

3. **Configure agent** with the generated key (`agent/agent.yaml`):
```yaml
server:
  api_key: "your-generated-api-key-here"
```

### API Key Management

#### Generating Keys

**Bootstrap Key**: On first start with auth enabled, Sentinel creates a bootstrap key:
```
Key: sentinel-bootstrap-key-change-me
```
**⚠️ IMPORTANT**: Change this immediately in production!

**Custom Keys**:
```bash
# Generate a new API key
go run tools/keygen/main.go --name "my-agent" --expires 90d

# Output:
# API Key ID: abc123...
# API Key: rAnDomKeY123...
#
# Add this to agent configuration:
# api_key: "rAnDomKeY123..."
```

#### Key Rotation

1. Generate a new API key
2. Update agent configuration with new key
3. Restart agent
4. Revoke old key from `api-keys.yaml`
5. Restart server

#### API Keys File Format

`api-keys.yaml`:
```yaml
keys:
  - id: "abc123"
    key_hash: "sha256_hash_of_key"
    name: "production-agent-1"
    created_at: "2024-01-01T00:00:00Z"
    expires_at: "2024-12-31T23:59:59Z"  # Optional
    active: true
    metadata:
      environment: "production"
      datacenter: "us-east-1"
```

**Never store plain keys in this file!** Only hashes are stored.

## TLS/SSL Configuration

### Generating Certificates

**For Production** (use real CA):
```bash
# Use Let's Encrypt, your organization's CA, or commercial CA
```

**For Development/Testing**:
```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt \
    -days 365 -nodes -subj "/CN=sentinel.example.com"

# Install certificates
sudo mkdir -p /etc/sentinel/certs
sudo mv server.{crt,key} /etc/sentinel/certs/
sudo chmod 600 /etc/sentinel/certs/server.key
```

### Enabling TLS

**Server configuration** (`config.yaml`):
```yaml
server:
  tls:
    enabled: true
    cert_file: "/etc/sentinel/certs/server.crt"
    key_file: "/etc/sentinel/certs/server.key"
```

**Agent configuration** (`agent.yaml`):
```yaml
server:
  url: "wss://sentinel.example.com:8080/api/v1/events"
  insecure_skip_tls: false  # Never true in production!
```

### Certificate Best Practices

1. **Use Strong Keys**: Minimum 2048-bit RSA or 256-bit ECDSA
2. **Regular Rotation**: Renew certificates before expiration
3. **Secure Storage**: Protect private keys (chmod 600)
4. **Valid Hostnames**: Match certificate CN/SAN to server hostname
5. **Trust Chain**: Ensure agents trust the CA

## Rate Limiting

### Configuration

```yaml
security:
  rate_limit: 10.0        # Requests per second per IP
  rate_limit_burst: 20    # Burst allowance
```

### Recommendations by Deployment Size

- **Small (1-10 agents)**: rate_limit: 5.0, burst: 10
- **Medium (10-100 agents)**: rate_limit: 10.0, burst: 20
- **Large (100+ agents)**: rate_limit: 50.0, burst: 100

### Bypassing Rate Limits

To bypass rate limits for specific IPs (e.g., internal network):
- Deploy server behind reverse proxy
- Configure proxy to not forward rate-limited requests
- Or modify middleware to check trusted proxy list

## Input Validation

Sentinel validates all incoming data:

### Event Data Limits

- **Max batch size**: 1 MB
- **Max events per batch**: 1,000
- **Max hostname length**: 255 characters
- **Max string field length**: 255 characters

### Validation Rules

1. **JSON Structure**: All data must be valid JSON
2. **Required Fields**: Events must have `type`, `hostname`, `timestamp`
3. **Type Validation**: Event types must be `network` or `process`
4. **Hostname Format**: Must conform to RFC 1123
5. **Numeric Fields**: PIDs, UIDs, ports must be numbers
6. **No Code Injection**: All strings are sanitized

### Handling Validation Failures

Invalid data is rejected with HTTP 400 and logged:
```log
2024-01-01 12:00:00 Failed to process events from 192.168.1.10: invalid hostname format
```

## Security Headers

Sentinel automatically sets security headers on all responses:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

## Audit Logging

All requests are logged with:
- Timestamp
- HTTP method and path
- Response status code
- Duration
- Client IP address

**Example log**:
```
2024-01-01 12:00:00 GET /api/v1/events 200 45ms
```

### Log Locations

- **Development**: stdout
- **Production**: Configure with log aggregation (ELK, Splunk, etc.)

### Monitoring for Security Events

Monitor logs for:
- Failed authentication attempts
- Rate limit violations
- Validation errors
- Unusual traffic patterns

**Example alert queries**:
```bash
# Failed auth attempts
grep "Authentication failed" /var/log/sentinel/server.log

# Rate limit hits
grep "Rate limit exceeded" /var/log/sentinel/server.log

# Validation errors
grep "Failed to process events" /var/log/sentinel/server.log
```

## Network Security

### Firewall Rules

**Server**:
```bash
# Allow only agent IPs
sudo ufw allow from 192.168.1.0/24 to any port 8080

# Or allow specific IPs
sudo ufw allow from 192.168.1.10 to any port 8080
sudo ufw allow from 192.168.1.11 to any port 8080
```

**Agent**:
```bash
# Allow outbound to Sentinel server only
sudo ufw allow out to 192.168.1.5 port 8080
```

### Network Segmentation

1. **Agent Network**: Isolated segment for monitored servers
2. **Sentinel Network**: Separate segment for Sentinel server
3. **Management Network**: Admin access to Sentinel dashboard
4. **No Direct Internet**: Agents should not need internet access

### Reverse Proxy Setup

Use nginx/Caddy for additional security:

**nginx example**:
```nginx
server {
    listen 443 ssl http2;
    server_name sentinel.example.com;

    ssl_certificate /etc/ssl/certs/sentinel.crt;
    ssl_certificate_key /etc/ssl/private/sentinel.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location /api/v1/events {
        # Agent WebSocket endpoint - restrict to internal IPs
        allow 192.168.1.0/24;
        deny all;

        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        # Frontend API - allow from management network
        allow 10.0.0.0/8;
        deny all;

        proxy_pass http://localhost:8080;
    }
}
```

## Secrets Management

### API Keys

**Never commit API keys to version control!**

**Recommended storage**:
1. **HashiCorp Vault**: Centralized secrets management
2. **AWS Secrets Manager**: For AWS deployments
3. **Environment Variables**: For containerized deployments
4. **Encrypted Files**: With proper file permissions (600)

**Loading from environment**:
```yaml
server:
  api_key: "${SENTINEL_API_KEY}"
```

### TLS Certificates

Store in:
- `/etc/sentinel/certs/` with restrictive permissions
- Hardware Security Module (HSM) for high-security environments
- Cloud key management services (AWS KMS, GCP KMS)

## Security Checklist

### Pre-Production

- [ ] Enable authentication (`require_auth: true`)
- [ ] Generate unique API keys for each agent
- [ ] Enable TLS (`tls.enabled: true`)
- [ ] Use valid SSL certificates (not self-signed)
- [ ] Configure rate limiting appropriately
- [ ] Set up firewall rules
- [ ] Remove bootstrap API key
- [ ] Configure log aggregation
- [ ] Set up security monitoring/alerts
- [ ] Document incident response procedures

### Post-Deployment

- [ ] Monitor authentication failures
- [ ] Review access logs regularly
- [ ] Rotate API keys quarterly
- [ ] Renew SSL certificates before expiration
- [ ] Update Sentinel to latest security patches
- [ ] Conduct security audits
- [ ] Test disaster recovery procedures
- [ ] Review and update firewall rules

## Incident Response

### Compromised API Key

1. **Immediate**: Set `active: false` in `api-keys.yaml`
2. **Restart** Sentinel server
3. **Generate** new API key
4. **Update** affected agents
5. **Investigate** how key was compromised
6. **Review** logs for unauthorized access

### Suspected Breach

1. **Isolate**: Disconnect affected components
2. **Preserve**: Save logs and system state
3. **Investigate**: Analyze logs for IOCs
4. **Remediate**: Patch vulnerabilities, rotate credentials
5. **Monitor**: Enhanced monitoring post-incident
6. **Document**: Post-mortem and lessons learned

### DDoS Attack

1. **Identify**: Monitor rate limit logs
2. **Block**: Add firewall rules for attacking IPs
3. **Scale**: Increase rate limits or add capacity
4. **Upstream**: Use CDN/DDoS protection service
5. **Review**: Adjust rate limiting configuration

## Compliance

### Data Privacy

Sentinel collects:
- Process names and arguments
- Network connection metadata
- User/UID information
- Hostnames

**Ensure compliance with**:
- GDPR (if monitoring EU users)
- HIPAA (if in healthcare environment)
- SOC 2 (for SaaS deployments)
- Your organization's data policies

### Retention Policies

Configure appropriate retention in `config.yaml`:
```yaml
storage:
  retention_hours: 24  # Adjust based on compliance requirements
```

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Controls](https://www.cisecurity.org/controls/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

## Security Reporting

To report security vulnerabilities:
- **Email**: security@example.com
- **PGP Key**: [Include PGP key fingerprint]
- **Expected Response**: Within 48 hours

**Please do not open public issues for security vulnerabilities.**
